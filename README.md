# Gohome

Home automation for the geek home. Built in Go.

[![Build Status](https://github.com/barnybug/gohome/workflows/Go/badge.svg)](https://github.com/barnybug/gohome/actions)
[![GoDoc](https://godoc.org/github.com/barnybug/gohome?status.svg)](http://godoc.org/github.com/barnybug/gohome)

## Features

- Lightweight, small memory footprint (runs on the Raspberry Pi)
- Modular, distributed and extendible
- Remotely controllable
- RFID keyless entry
- Smart burglar alarm
- Historic weather monitoring and alerting
- Zoned central heating control
- Automatic plant irrigation system
- Sunset / sunrise triggered home lighting

## Devices supported

- rfxcom RFXtrx433 USB device (http://www.rfxcom.com/)
- Homeeasy remote control sockets/lights (http://homeeasy.eu/)
- Arduino with relay module (http://arduino.cc/)
- ZTE 3g modem (ZTE MF110/MF627/MF636, SMS support)
- USB 125KHZ EM4100 RFID Proximity Reader (RFID tag reader)
- Oregon Scientific WMR100/200 weather station
- Audio out (espeak text to speech)
- Foscam wireless IP cameras
- Motion webcam application
- Currentcost electricity monitor
- LIRC infra-red transmitters

## Services supported

- MQTT
- Twitter
- Jabber
- REST API
- Email
- SMS
- Graphite (graphs)
- Wunderground (http://www.wunderground.com/)

## Installation

Installation is easy, just download the binary from the github releases page (builds are available for Linux 32-bit, 64-bit and ARM):
https://github.com/barnybug/gohome/releases/latest

For the raspberry pi, download the ARM build.

Rename and make the download executable:

    $ cp gohome-my-platform /usr/local/bin; chmod +x /usr/local/bin/gohome

You also will need mosquitto installed:

### Debian/Ubuntu/Raspbian:

    $ apt-get install mosquitto
    $ service mosquitto start

### ArchLinux

    $ packer -S mosquitto
    $ systemctl enable --now mosquitto

## Configuration

An example configuration is at:
https://github.com/barnybug/gohome/blob/master/config.yml

Edit this to match your setup and upload:

    $ curl -XPOST localhost:8723/config?path=config --data-binary config.yml

## Running

gohome runs as a set of distributed and independent processes/services. They
can run across different hosts connecting to the same network, with the pubsub
bus (MQTT) connecting all the components together.

To manually run a gohome service:

    $ gohome run <service>

The best way to manage the whole set of services is using your user systemd -
because you probably want to ensure they are restarted if they happen to
crash. Any recent Archlinux comes with this preconfigured, you just need to
install the gohome services by running the script `setup.sh` provided:

    $ cd systemd && ./setup.sh

This will enable and start all the services defined in by `SERVICES=...` in
`setup.sh`.

The 'services' don't necessarily have to be gohome itself - you can add
scripts to the system of your own crafting. The `script` service can wrap
external scripts and passes on any events printing to stdout by the external
script into gohome, allowing quick and easy integration.

## Building from source

To build yourself from source:

    $ go get github.com/barnybug/gohome
    $ cd $GOPATH/src/github.com/barnybug/gohome
    $ make install

This will produce a binary `gohome` in `~/go/bin` (ie. $GOPATH/bin), after this follow the steps as above.
