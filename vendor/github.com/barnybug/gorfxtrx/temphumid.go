package gorfxtrx

import (
	"encoding/binary"
	"fmt"
)

// Struct for the TempHumid packets
type TempHumid struct {
	TypeId         byte
	SequenceNumber byte
	id             uint16
	Temp           float64
	Humidity       byte
	HumidityStatus byte
	Battery        byte
	Rssi           byte
}

var tempHumidTypes = map[byte]string{
	0x01: "THGN122/123, THGN132, THGR122/228/238/268",
	0x02: "THGR810, THGN800",
	0x03: "RTGR328",
	0x04: "THGR328",
	0x05: "WTGR800",
	0x06: "THGR918, THGRN228, THGN500",
	0x07: "TFA TS34C, Cresta",
	0x08: "WT260,WT260H,WT440H,WT450,WT450H",
	0x09: "Viking 02035,02038",
}

func (self *TempHumid) Receive(data []byte) {
	self.TypeId = data[2]
	self.SequenceNumber = data[3]
	self.id = binary.BigEndian.Uint16(data[4:6])
	t := binary.BigEndian.Uint16(data[6:8])
	if data[6] >= 0x80 {
		self.Temp = -float64(t-0x8000) / 10
	} else {
		self.Temp = float64(t) / 10
	}
	self.Humidity = data[8]
	self.HumidityStatus = data[9]
	self.Battery = (data[10] & 0x0f) * 10
	self.Rssi = data[10] >> 4
}

// Id of the device.
func (self *TempHumid) Id() string {
	return fmt.Sprintf("%02x:%02x", self.id>>8, self.id&0xff)
}

// Type of the device.
func (self *TempHumid) Type() string {
	return tempHumidTypes[self.TypeId]
}
