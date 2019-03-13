package gorfxtrx

import (
	"encoding/binary"
	"fmt"
)

var powerFactor = 223.666

// Elec1 devices: OWL CM113, cent-a-meter, Electrisave
type Elec1 struct {
	Subtype  uint8
	SeqNr    uint8
	SensorId uint16
	Count    uint8
	Current1 float64
	Current2 float64
	Current3 float64
	Signal   uint8
	Battery  byte
}

func (self *Elec1) Receive(data []byte) {
	// 0D 59 01 0C 89 00 07 00 00 00 1A 00 00 79
	// ^^ length            ^^^^^ current1
	//    ^^ type                 ^^^^^ current2
	//                   ^^ count       ^^^^^ current3
	//             ^^^^^ sensor id
	self.Subtype = data[2]
	self.SeqNr = data[3]
	self.SensorId = binary.BigEndian.Uint16(data[4:6])
	self.Count = data[6]
	self.Current1 = float64(binary.BigEndian.Uint16(data[7:9])) / 10
	self.Current2 = float64(binary.BigEndian.Uint16(data[9:11])) / 10
	self.Current3 = float64(binary.BigEndian.Uint16(data[11:13])) / 10
	self.Signal = data[13] >> 4
	self.Battery = (data[13] & 0xF) * 10
}

func (self *Elec1) String() string {
	return fmt.Sprintf("Current id: %04x current1: %.1fA current2: %.1fA current3: %.1fA signal: %d battery: %d", self.SensorId, self.Current1, self.Current2, self.Current3, self.Signal, self.Battery)
}

// Elec3 devices: OWL CM180
type Elec3 struct {
	Subtype  uint8
	SeqNr    uint8
	SensorId uint16
	Power    uint32
	Total    float64
	Signal   uint8
	Battery  byte
}

func (self *Elec3) Receive(data []byte) {
	// 11 5a 02 1e 87 82 00 00 00 01 f2 00 00 00 00 1f 56 69
	// 11 5a 02 02 87 82 00 00 00 01 01 00 00 00 00 84 90 69: 257 W 151 Wh
	// ^^ length            ^^^^^^^^^^^ power1
	//    ^^ type
	//             ^^^^^ sensor id
	//                                  ^^^^^^^^^^^^^^^^^ total
	//                                                    ^^ signal/battery
	self.Subtype = data[2]
	self.SeqNr = data[3]
	self.SensorId = binary.BigEndian.Uint16(data[4:6])
	self.Power = binary.BigEndian.Uint32(data[7:11])
	p := append([]byte{0, 0}, data[11:17]...)
	self.Total = float64(binary.BigEndian.Uint64(p)) / powerFactor
	self.Signal = data[17] >> 4
	self.Battery = (data[17] & 0xF) * 10
}

func (self *Elec3) String() string {
	return fmt.Sprintf("Power id: %04x power: %dW total: %.2fWh signal: %d battery: %d", self.SensorId, self.Power, self.Total, self.Signal, self.Battery)
}
