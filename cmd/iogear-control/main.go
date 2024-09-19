package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/eric/iogearcontrol"
	"go.bug.st/serial"
)

func main() {
	tty := os.Getenv("IOGEAR_HDMI_TTY")
	outputJson := false

	flag.StringVar(&tty, "tty", tty, "tty device to use")
	flag.BoolVar(&outputJson, "json", false, "output in json format")

	flag.Parse()

	if tty == "" {
		ports, err := serial.GetPortsList()
		if err != nil {
			log.Fatal(err)
		}
		if len(ports) == 0 {
			log.Fatal("No serial ports found!")
		}
		for _, port := range ports {
			if strings.Contains(port, "Bluetooth") || strings.Contains(port, "wifi") || strings.Contains(port, "debug-console") {
				continue
			}
			tty = port
			fmt.Fprintf(os.Stderr, "Found port: %v\n", port)
			break
		}
	}

	hs, err := iogearcontrol.NewHDMISwitcher(tty)
	if err != nil {
		log.Fatal(err)
	}
	defer hs.Close()

	port := 1
	command := "status"

	if flag.NArg() > 0 {
		command = flag.Arg(0)
		if command == "switch" && flag.NArg() > 1 {
			port, _ = strconv.Atoi(flag.Arg(1))
		}

		if flag.NArg() == 1 {
			port, err = strconv.Atoi(flag.Arg(0))
			if err == nil {
				command = "switch"
			}
		}
	}

	switch command {
	case "switch":
		err = hs.On()
		if err != nil {
			log.Fatal(err)
		}
		err = hs.Switch(port)
		if err != nil {
			log.Fatal(err)
		}
	case "off":
		err = hs.Off()
		if err != nil {
			log.Fatal(err)
		}
	case "on":
		err = hs.On()
		if err != nil {
			log.Fatal(err)
		}
	}

	response, err := hs.Status()
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
