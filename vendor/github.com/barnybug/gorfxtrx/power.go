package gorfxtrx

import (
	"encoding/binary"
	"fmt"
)

// Struct for the Power packets.
type Power struct {
	Subtype  uint8
	SeqNr    uint8
	SensorId uint16
	Power    uint32
	Total    float64
	Signal   uint8
	Battery  byte
}

var powerFactor = 223.666

func (self *Power) Receive(data []byte) {
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

func (self *Power) String() string {
	return fmt.Sprintf("Power id: %04x power: %dW total: %.2fWh signal: %d battery: %d", self.SensorId, self.Power, self.Total, self.Signal, self.Battery)
}
