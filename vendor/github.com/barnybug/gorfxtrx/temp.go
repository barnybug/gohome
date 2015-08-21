package gorfxtrx

import (
	"encoding/binary"
	"fmt"
)

// Struct for the Temp packets.
type Temp struct {
	typeId         byte
	SequenceNumber byte
	id             uint16
	Temp           float64
	Battery        byte
	Rssi           byte
}

var tempTypes = map[byte]string{
	0x01: "THR128/138, THC138",
	0x02: "THC238/268,THN132,THWR288,THRN122,THN122,AW129/131",
	0x03: "THWR800",
	0x04: "RTHN318",
	0x05: "La Crosse TX2, TX3, TX4, TX17",
	0x06: "TS15C",
	0x07: "Viking 02811",
	0x08: "La Crosse WS2300",
	0x09: "RUBiCSON",
	0x0a: "TFA 30.3133",
}

func (self *Temp) Receive(data []byte) {
	self.typeId = data[2]
	self.SequenceNumber = data[3]
	self.id = binary.BigEndian.Uint16(data[4:6])
	t := binary.BigEndian.Uint16(data[6:8])
	if data[6] >= 0x80 {
		self.Temp = -float64(t-0x8000) / 10
	} else {
		self.Temp = float64(t) / 10
	}
	self.Battery = (data[8] & 0x0f) * 10
	self.Rssi = data[8] >> 4
}

// Id of the device.
func (self *Temp) Id() string {
	return fmt.Sprintf("%02x:%02x", self.id>>8, self.id&0xff)
}

// Type of the device.
func (self *Temp) Type() string {
	return tempTypes[self.typeId]
}
