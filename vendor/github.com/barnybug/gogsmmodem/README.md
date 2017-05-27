## gogsmmodem: Go library for GSM modems

[![Build Status](https://travis-ci.org/barnybug/gogsmmodem.svg?branch=master)](https://travis-ci.org/barnybug/gogsmmodem)

Go library for the sending and receiving SMS messages through a GSM modem.

### Tested devices
- ZTE MF110/MF627/MF636

### Installation
Run:

    go get github.com/barnybug/gogsmmodem

### Usage
Example:

```go

package main

import (
    "fmt"

    "github.com/barnybug/gogsmmodem"
    "github.com/tarm/serial"
)

func main() {
    conf := serial.Config{Name: "/dev/ttyUSB1", Baud: 115200}
    modem, err := gogsmmodem.Open(&conf, true)
    if err != nil {
        panic(err)
    }

    for packet := range modem.OOB {
        fmt.Printf("%#v\n", packet)
        switch p := packet.(type) {
        case gogsmmodem.MessageNotification:
            fmt.Println("Message notification:", p)
            msg, err := modem.GetMessage(p.Index)
            if err == nil {
                fmt.Printf("Message from %s: %s\n", msg.Telephone, msg.Body)
                modem.DeleteMessage(p.Index)
            }
        }
    }
}
```

### Changelog
0.1.0

- First release
