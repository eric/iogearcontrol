package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// errNoServer means nothing is listening, so the caller should talk to
// the serial port directly.
var errNoServer = errors.New("server not running")

// runViaServer sends command to a running `serve` instance and returns
// the switcher status in the same shape as HDMISwitcher.Status.
func runViaServer(listen, command string, port int) (map[string]string, error) {
	method, path := http.MethodGet, "/status"
	switch command {
	case "switch":
		method, path = http.MethodPost, fmt.Sprintf("/switch/%d", port)
	case "on", "off":
		method, path = http.MethodPost, "/"+command
	}

	req, err := http.NewRequest(method, "http://"+listen+path, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errNoServer
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("server rejected %s %s", method, path)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("reading server response: %w", err)
	}

	if msg, _ := body["error"].(string); msg != "" && resp.StatusCode != http.StatusOK {
		return nil, errors.New(msg)
	}
	if connected, _ := body["connected"].(bool); !connected {
		return nil, fmt.Errorf("server is not connected to the switcher: %v", body["error"])
	}

	result := make(map[string]string)
	for key, value := range body {
		switch key {
		case "connected", "error", "updated":
			continue
		}
		if s, ok := value.(string); ok {
			result[key] = s
		}
	}
	return result, nil
}
