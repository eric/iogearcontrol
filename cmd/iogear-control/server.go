package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eric/iogearcontrol"
)

// server owns the serial port so that commands and status polls never
// talk over each other on the line.
type server struct {
	tty string

	// serialMu guards hs and every exchange on the serial port.
	serialMu sync.Mutex
	hs       *iogearcontrol.HDMISwitcher

	stateMu   sync.RWMutex
	status    map[string]string
	connected bool
	lastErr   string
	updated   time.Time
}

func serve(tty, listen string, pollInterval time.Duration) error {
	s := &server{tty: tty}

	go func() {
		s.refresh()
		for range time.Tick(pollInterval) {
			s.refresh()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/switch/", s.handleSwitch)
	mux.HandleFunc("/on", s.handleCommand(func(hs *iogearcontrol.HDMISwitcher) error { return hs.On() }))
	mux.HandleFunc("/off", s.handleCommand(func(hs *iogearcontrol.HDMISwitcher) error { return hs.Off() }))

	log.Printf("Listening on http://%s (tty %s)", listen, tty)
	return http.ListenAndServe(listen, mux)
}

// withSwitcher runs fn against the switcher, opening the port if needed.
// Any error closes the port so the next call reopens it, which recovers
// from the USB adapter being unplugged and replugged.
func (s *server) withSwitcher(fn func(hs *iogearcontrol.HDMISwitcher) error) error {
	s.serialMu.Lock()
	defer s.serialMu.Unlock()

	if s.hs == nil {
		hs, err := iogearcontrol.NewHDMISwitcher(s.tty)
		if err != nil {
			return err
		}
		s.hs = hs
	}

	err := fn(s.hs)
	if err != nil {
		_ = s.hs.Close()
		s.hs = nil
	}
	return err
}

func (s *server) refresh() {
	var status map[string]string
	err := s.withSwitcher(func(hs *iogearcontrol.HDMISwitcher) error {
		var err error
		status, err = hs.Status()
		return err
	})

	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if err != nil {
		if s.connected || s.lastErr != err.Error() {
			log.Printf("Status failed: %v", err)
		}
		s.connected = false
		s.lastErr = err.Error()
		return
	}

	if !s.connected {
		log.Printf("Connected: %v", status)
	}
	s.status = status
	s.connected = true
	s.lastErr = ""
	s.updated = time.Now()
}

// snapshot returns the last known status. The switcher's own fields
// ("Input", "Output", ...) are kept at the top level so existing
// consumers can keep reading $.Input and $.Output.
func (s *server) snapshot() map[string]interface{} {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	result := map[string]interface{}{
		"connected": s.connected,
		"error":     s.lastErr,
	}
	if !s.updated.IsZero() {
		result["updated"] = s.updated.Format(time.RFC3339)
	}
	for key, value := range s.status {
		result[key] = value
	}
	return result
}

func (s *server) writeStatus(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(s.snapshot())
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.writeStatus(w, http.StatusOK)
}

func (s *server) handleSwitch(w http.ResponseWriter, r *http.Request) {
	input, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/switch/"))
	if err != nil || input < 1 || input > 4 {
		http.Error(w, "input must be 1-4", http.StatusBadRequest)
		return
	}

	s.handleCommand(func(hs *iogearcontrol.HDMISwitcher) error {
		if err := hs.On(); err != nil {
			return err
		}
		return hs.Switch(input)
	})(w, r)
}

// handleCommand runs a command, then re-reads the switcher so the
// response reflects the new state.
func (s *server) handleCommand(command func(hs *iogearcontrol.HDMISwitcher) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := s.withSwitcher(command)
		s.refresh()

		if err != nil {
			log.Printf("%s failed: %v", r.URL.Path, err)
			s.stateMu.Lock()
			s.lastErr = fmt.Sprintf("%s: %v", r.URL.Path, err)
			s.stateMu.Unlock()
			s.writeStatus(w, http.StatusBadGateway)
			return
		}

		log.Printf("%s ok", r.URL.Path)
		s.writeStatus(w, http.StatusOK)
	}
}
