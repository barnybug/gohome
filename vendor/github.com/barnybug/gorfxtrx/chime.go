package gorfxtrx

import (
	"encoding/binary"
	"fmt"
)

// Struct for the Chime packets.
type Chime struct {
	typeId         byte
	SequenceNumber byte
	id             uint16
	Chime          byte
	Battery        byte
	Rssi           byte
}

var chimeTypes = map[byte]string{
	0x00: "Byron SX",
}

func (self *Chime) Receive(data []byte) {
	// 07 16 00 03 00 7a 01 70
	self.typeId = data[2]
	self.SequenceNumber = data[3]
	self.id = binary.BigEndian.Uint16(data[4:6])
	self.Chime = data[6]
	self.Battery = (data[7] & 0x0f) * 10
	self.Rssi = data[7] >> 4
}

// Id of the device.
func (self *Chime) Id() string {
	return fmt.Sprintf("%02x:%02x", self.id>>8, self.id&0xff)
}

// Type of the device.
func (self *Chime) Type() string {
	return chimeTypes[self.typeId]
}
