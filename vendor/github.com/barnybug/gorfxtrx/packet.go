package gorfxtrx

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// Interface representing a received packet.
type Packet interface {
	// Deserialize packet from wire format
	Receive(data []byte)
}

// Interface representing a transmittable packet.
type OutPacket interface {
	// Serialize packet to wire format
	Send() []byte
}

type PacketType struct {
	length int
	name   string
	packet func() Packet
}

var PacketTypes = map[byte]PacketType{
	0x1:  PacketType{20, "Status", func() Packet { return &Status{} }},
	0x2:  PacketType{4, "TransmitAck", func() Packet { return &TransmitAck{} }},
	0x10: PacketType{7, "LightingX10", func() Packet { return &LightingX10{} }},
	0x11: PacketType{11, "LightingHE", func() Packet { return &LightingHE{} }},
	0x16: PacketType{7, "Chime", func() Packet { return &Chime{} }},
	0x50: PacketType{8, "Temp", func() Packet { return &Temp{} }},
	0x52: PacketType{10, "TempHumid", func() Packet { return &TempHumid{} }},
	0x55: PacketType{11, "Rain", func() Packet { return &Rain{} }},
	0x56: PacketType{16, "Wind", func() Packet { return &Wind{} }},
	0x5a: PacketType{17, "Power", func() Packet { return &Power{} }},
}

// Parse a packet from a byte array.
func Parse(data []byte) (Packet, error) {
	if data[0] == 0 {
		// ignore the empty packet - not an error
		return nil, nil
	}
	dlen := len(data) - 1
	if int(data[0]) != dlen {
		return nil, errors.New(fmt.Sprintf("Packet unexpected length: %d != %d", dlen, int(data[0])))
	}

	var pkt Packet
	if packetType, ok := PacketTypes[data[1]]; ok {
		if dlen != packetType.length {
			return nil, fmt.Errorf("%s packet incorrect length, expected: %d actual: %d",
				packetType.name, packetType.length, dlen)
		}
		pkt = packetType.packet()
	} else {
		pkt = &Unknown{}
	}
	pkt.Receive(data)
	return pkt, nil
}

// Struct for an Unknown packet type.
type Unknown struct {
	data []byte
}

func (self *Unknown) Receive(data []byte) {
	self.data = data
}

func (self *Unknown) Send() []byte {
	return self.data
}

func (self *Unknown) String() string {
	return "Unknown: " + hex.EncodeToString(self.data)
}
