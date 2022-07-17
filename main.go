package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
)

type HDMISwitcher struct {
	port serial.Port
}

func NewHDMISwitcher(device string) (*HDMISwitcher, error) {
	port, err := serial.Open(device, &serial.Mode{
		BaudRate: 19200,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	})
	if err != nil {
		return nil, err
	}

	err = port.SetReadTimeout(50 * time.Millisecond)
	if err != nil {
		return nil, err
	}

	return &HDMISwitcher{port: port}, nil
}

func (hs *HDMISwitcher) Send(command string) (string, error) {
	// Reset the buffers
	_ = hs.port.ResetInputBuffer()
	_ = hs.port.ResetOutputBuffer()

	if !strings.HasSuffix(command, "\n") {
		command += "\n"
	}

	n, err := hs.port.Write([]byte(command))
	if err != nil {
		return "", err
	}
	if n != len(command) {
		return "", fmt.Errorf("failed to send command: %s", command)
	}

	builder := new(strings.Builder)

	buf := make([]byte, 1000)
	for {
		n, err := hs.port.Read(buf)
		if err != nil {
			return builder.String(), err
		}
		if n == 0 {
			break
		}

		builder.Write(buf[:n])
	}

	status, response, _ := strings.Cut(builder.String(), "\r\n")
	result := status[len(command):]
	if result != "Command OK" {
		return response, fmt.Errorf("failed to send command: %s", status)
	}

	return response, nil
}

func (hs *HDMISwitcher) Switch(input int) error {
	_, err := hs.Send(fmt.Sprintf("sw i%02d", input))
	if err != nil {
		return err
	}

	return nil
}

func (hs *HDMISwitcher) Status() (map[string]string, error) {
	response, err := hs.Send("read")
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)

	entries := strings.Split(response, "\r\n")
	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, ": ")
		if ok {
			result[key] = strings.TrimSpace(value)
		}
	}

	return result, nil
}

func (hs *HDMISwitcher) Close() error {
	return hs.port.Close()
}

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

	hs, err := NewHDMISwitcher(tty)
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
