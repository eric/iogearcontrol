package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eric/iogearcontrol"
	"go.bug.st/serial"
)

func main() {
	tty := os.Getenv("IOGEAR_HDMI_TTY")
	outputJson := false
	listen := "127.0.0.1:8341"
	pollInterval := 2 * time.Second

	flag.StringVar(&tty, "tty", tty, "tty device to use")
	flag.BoolVar(&outputJson, "json", false, "output in json format")
	flag.StringVar(&listen, "listen", listen, "address for the serve command to listen on")
	flag.DurationVar(&pollInterval, "poll", pollInterval, "how often the serve command reads the switcher status")

	flag.Parse()

	if tty == "" {
		ports, err := serial.GetPortsList()
		if err != nil {
			log.Fatal(err)
		}
		if len(ports) == 0 {
			log.Fatal("No serial ports found!")
		}
		// Prefer a USB-to-serial adapter; other USB devices (such as
		// usbmodem ports) also show up in the list.
		for _, port := range ports {
			if strings.Contains(port, "usbserial") {
				tty = port
				break
			}
		}
		if tty == "" {
			for _, port := range ports {
				if strings.Contains(port, "Bluetooth") || strings.Contains(port, "wifi") || strings.Contains(port, "debug-console") {
					continue
				}
				tty = port
				break
			}
		}
		if tty == "" {
			log.Fatal("No usable serial ports found!")
		}
		fmt.Fprintf(os.Stderr, "Found port: %v\n", tty)
	}

	if flag.Arg(0) == "serve" {
		log.Fatal(serve(tty, listen, pollInterval))
	}

	port := 1
	command := "status"

	if flag.NArg() > 0 {
		command = flag.Arg(0)
		if command == "switch" && flag.NArg() > 1 {
			port, _ = strconv.Atoi(flag.Arg(1))
		}

		if flag.NArg() == 1 {
			if n, err := strconv.Atoi(flag.Arg(0)); err == nil {
				port = n
				command = "switch"
			}
		}
	}

	// The server holds the serial port exclusively while it runs, so go
	// through it when it is up and only talk to the port directly when not.
	response, err := runViaServer(listen, command, port)
	if errors.Is(err, errNoServer) {
		response, err = runDirect(tty, command, port)
	}
	if err != nil {
		log.Fatal(err)
	}

	if outputJson {
		data, err := json.Marshal(response)
		if err == nil {
			fmt.Println(string(data))
		} else {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("%+v\n", response)
	}
}

func runDirect(tty, command string, port int) (map[string]string, error) {
	hs, err := iogearcontrol.NewHDMISwitcher(tty)
	if err != nil {
		return nil, err
	}
	defer hs.Close()

	switch command {
	case "switch":
		err = hs.On()
		if err != nil {
			return nil, err
		}
		err = hs.Switch(port)
		if err != nil {
			return nil, err
		}
	case "off":
		err = hs.Off()
		if err != nil {
			return nil, err
		}
	case "on":
		err = hs.On()
		if err != nil {
			return nil, err
		}
	}

	return hs.Status()
}
