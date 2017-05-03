/*
Package gorfxtrx is a library for the RFXcom RFXtrx433 USB transceiver.

http://www.rfxcom.com/store/Transceivers/12103

Supported transmitter / receivers:

- Oregon weather devices (THGR810, WTGR800, THN132N, PCR800, etc.)

- X10 RF devices (Domia Lite, HE403, etc.)

- HomeEasy devices (HE300, HE301, HE303, HE305, etc.)

RFXcom devices tested:

- RFXcom RFXtrx433 USB Transceiver

Example usage:

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

*/
package gorfxtrx

import (
	"io"
	"log"
	"time"

	"github.com/tarm/serial"
)

// Device representing the serial connection to the USB device.
type Device struct {
	ser   io.ReadWriteCloser
	debug bool
}

// Open the device at the given path.
func Open(path string, debug bool) (*Device, error) {
	dev := Device{debug: debug}

	c := &serial.Config{Name: path, Baud: 38400}
	ser, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}

	if debug {
		dev.ser = LogReadWriteCloser{ser}
	} else {
		dev.ser = ser
	}

	log.Println("Sending reset")
	reset, _ := NewReset()
	err = dev.Send(reset)
	if err != nil {
		return nil, err
	}

	return &dev, nil
}

// Close the device.
func (self *Device) Close() {
	self.ser.Close()
}

// Read a packet from the device. Blocks until data is available.
func (self *Device) Read() (Packet, error) {
	buf := make([]byte, 257)
	for {
		// read length
		i, err := self.ser.Read(buf[0:1])
		if i == 0 && err == io.EOF {
			// empty read, sleep a bit recheck
			time.Sleep(time.Millisecond * 200)
			continue
		}
		if err != nil {
			return nil, err
		}
		if i == 0 {
			continue
		}

		// read rest of data
		l := int(buf[0])
		buf = buf[0 : l+1]
		for read := 0; read < l; read += i {
			i, err = self.ser.Read(buf[read+1:])
			if i == 0 && err == io.EOF {
				time.Sleep(time.Millisecond * 200)
				continue
			}
			if err != nil {
				return nil, err
			}
		}

		// parse packet
		packet, err := Parse(buf)
		if self.debug {
			log.Printf("Parse(%#v) = (%#v, %v)\n", buf, packet, err)
		}
		return packet, err
	}
}

// Send (transmit) a packet.
func (self *Device) Send(p OutPacket) error {
	buf := p.Send()
	_, err := self.ser.Write(buf)
	return err
}
