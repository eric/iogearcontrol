package main

import (
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

	if tty == "" {
		ports, err := serial.GetPortsList()
		if err != nil {
			log.Fatal(err)
		}
		if len(ports) == 0 {
			log.Fatal("No serial ports found!")
		}
		for _, port := range ports {
			if strings.Contains(port, "Bluetooth") || strings.Contains(port, "wifi") {
				continue
			}
			tty = port
			fmt.Printf("Found port: %v\n", port)
			break
		}
	}

	hs, err := iogearcontrol.NewHDMISwitcher(tty)
	if err != nil {
		log.Fatal(err)
	}
	defer hs.Close()

	port := 1

	if len(os.Args) > 1 {
		port, _ = strconv.Atoi(os.Args[1])
	}

	err = hs.Switch(port)
	if err != nil {
		log.Fatal(err)
	}

	response, err := hs.Status()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", response)
}
