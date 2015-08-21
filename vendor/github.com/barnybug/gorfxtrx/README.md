## gorfxtrx

[![Build Status](https://travis-ci.org/barnybug/gorfxtrx.svg?branch=master)](https://travis-ci.org/barnybug/gorfxtrx)
[![GoDoc](https://godoc.org/github.com/barnybug/gorfxtrx?status.svg)](http://godoc.org/github.com/barnybug/gorfxtrx)

Go library for the RFXcom RFXtrx433 USB transceiver:

http://www.rfxcom.com/store/Transceivers/12103

The RFXtrx433 is great for home automation. It's affordable, both receives and transmits, and it supports a huge variety of devices.

### Supported transmitter / receivers
- Oregon weather devices (THGR810, WTGR800, THN132N, PCR800, etc.)
- X10 RF devices (Domia Lite, HE403, etc.)
- HomeEasy devices (HE300, HE301, HE303, HE305, etc.)

### RFXcom devices tested
- RFXcom RFXtrx433 USB Transceiver

### Installation
Run:

    go get github.com/barnybug/gorfxtrx

### Usage
Example:

```go
import (
    "fmt"
    "github.com/barnybug/gorfxtrx"
)

func main() {
    dev, err := gorfxtrx.Open("/dev/serial/by-id/usb-RFXCOM-...", true)
    if err != nil {
        panic("Error opening device")
    }

    for {
        packet, err := dev.Read()
        if err != nil {
            continue
        }

        fmt.Println("Received", packet)
    }
    dev.Close()
}
```

### Changelog
0.1.0

- First release
