# IOGear HDMI Control

Simple tool to control IOGear HDMI switcher over RS232

## Usage

Specify the port to switch to as the first argument:

    ~/src/iogearcontrol/cmd/iogear-control $ ./iogear-control 1
    Found port: /dev/cu.usbserial-AB0MSUVK
    map[F/W:V1.1.103 Input:port  1 Mode:NEXT Output:ON Pod:OFF]
    ~/src/iogearcontrol/cmd/iogear-control $

## Server

`serve` keeps the serial port open, reads the switcher status every 2
seconds, and exposes it over HTTP on `127.0.0.1:8341`:

    $ iogear-control serve
    $ curl http://127.0.0.1:8341/status
    {"F/W":"V1.1.103","Input":"port  3","Mode":"NEXT","Output":"ON","Pod":"OFF","connected":true,"error":"","updated":"..."}
    $ curl -X POST http://127.0.0.1:8341/switch/1
    $ curl -X POST http://127.0.0.1:8341/off
    $ curl -X POST http://127.0.0.1:8341/on

Every endpoint returns the latest status. Use `-listen` and `-poll` to
change the address and interval.

`ws://127.0.0.1:8341/ws` sends the status as JSON on connect and again
whenever the input, output, or connection state changes.

While the server is running it holds the serial port, so the
command-line forms above go through it automatically.

To run it at login, install the binary and load the LaunchAgent:

    $ go install ./cmd/iogear-control
    $ cp launchd/com.eric.iogear-control.plist ~/Library/LaunchAgents/
    $ launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.eric.iogear-control.plist

Logs go to `~/Library/Logs/iogear-control.log`.

## Tested With

### HDMI Switcher

<img src="./.README/61LjMaA9VaL._AC_SL1500_.jpg?raw=true" width="300px">

https://www.amazon.com/dp/B07FK9NMQV

### USB-to-RS232 cable

<img src="./.README/61ASVw1jt6L._AC_SL1500_.jpg?raw=true" width="300px">

https://www.amazon.com/dp/B07589ZF9X