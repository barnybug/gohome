# Gohome

Home automation for the geek home. Built in Go.

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

## Services supported

- MQTT
- ZeroMQ
- Nanomsg
- Twitter
- Jabber
- REST API
- Email
- SMS
- Graphite (graphs)
- Wunderground (http://www.wunderground.com/)

## Installation

Installation is easy, just download the binary from:
[gobuild.io](http://gobuild.io/download/github.com/barnybug/gohome).

For the raspberry pi, download the ARM build.

You also will need redis and mosquitto installed:

### Debian/Ubuntu/Raspbian:

    $ apt-get install redis
    TODO: mosquitto installation steps

### ArchLinux

    $ pacman -S redis
    $ systemctl enable redis
    $ systemctl start redis

    $ packer -S mosquitto
    $ systemctl enable mosquitto
    $ systemctl start mosquitto

## Configuration

Redis is used to store config.

An example configuration is at:
http://github.com/barnybug/gohome/config.yml

Edit this to match your setup and upload to redis:

    $ redis-cli -x set gohome/config < config.yml

## Running

Now start the gohome daemon:

    $ gohome start daemon

This will start all the processes defined in the config and ensure they stay
running.

gohome runs as a set of distributed and independent processes/services. These
don't necessarily have to be gohome itself - you can add scripts to the system
of your own crafting. Because they run as independent processes they can run
across different hardware, with the pubsub bus (MQTT/Zeromq/Nanomsg) connecting
all the components together.

## Building from source

To build yourself from source:

### Debian/Ubuntu/Raspbian:

    $ apt-get install -y golang mercurial libzmq4
    $ go get github.com/nitrous-io/goop
    $ goop go install github.com/barnybug/gohome

### ArchLinux

    $ pacman -S go libzmq4
    $ go get github.com/nitrous-io/goop
    $ goop go install github.com/barnybug/gohome

This will produce a binary `gohome', after this follow the steps as above.
