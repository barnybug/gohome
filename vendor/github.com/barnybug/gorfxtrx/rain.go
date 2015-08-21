package gorfxtrx

import (
	"encoding/binary"
	"fmt"
)

// Struct for the Rain packets.
type Rain struct {
	typeId         byte
	SequenceNumber byte
	id             uint16
	RainRate       float64
	RainTotal      float64
	Battery        byte
	Rssi           byte
}

var rainTypes = map[byte]string{
	0x01: "RGR126/682/918",
	0x02: "PCR800",
	0x03: "TFA",
	0x04: "UPM RG700",
	0x05: "WS2300",
	0x06: "La Crosse TX5",
}

func (self *Rain) Receive(data []byte) {
	self.typeId = data[2]
	self.SequenceNumber = data[3]
	self.id = binary.BigEndian.Uint16(data[4:6])
	self.RainRate = float64(binary.BigEndian.Uint16(data[6:8]))
	if self.typeId == 2 {
		self.RainRate /= 100
	}
	// total is 24-bit big endian
	self.RainTotal = (float64(data[8])*65536 + float64(binary.BigEndian.Uint16(data[9:11]))) / 10
	self.Battery = (data[11] & 0x0f) * 10
	self.Rssi = data[11] >> 4
}

// Id of the device.
func (self *Rain) Id() string {
	return fmt.Sprintf("%02x:%02x", self.id>>8, self.id&0xff)
}

// Type of the device.
func (self *Rain) Type() string {
	return rainTypes[self.typeId]
}
