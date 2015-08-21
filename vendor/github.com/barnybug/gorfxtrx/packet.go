package gorfxtrx

import (
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
	switch data[1] {
	case 0x01:
		if dlen != 13 {
			return nil, errors.New("Status packet too short")
		}
		pkt = &Status{}
	case 0x10:
		if dlen != 7 {
			return nil, errors.New("LightingX10 packet too short")
		}
		pkt = &LightingX10{}
	case 0x11:
		if dlen != 11 {
			return nil, errors.New("LightingHE packet too short")
		}
		pkt = &LightingHE{}
	// 0x12-0x15: lighting - to add support
	case 0x16:
		if dlen != 7 {
			return nil, errors.New("Chime packet too short")
		}
		pkt = &Chime{}
	case 0x50:
		if dlen != 8 {
			return nil, errors.New("Temp packet too short")
		}
		pkt = &Temp{}
	case 0x52:
		if dlen != 10 {
			return nil, errors.New("TempHumid packet too short")
		}
		pkt = &TempHumid{}
	case 0x55:
		if dlen != 11 {
			return nil, errors.New("Rain packet too short")
		}
		pkt = &Rain{}
	case 0x56:
		if dlen != 16 {
			return nil, errors.New("Wind packet too short")
		}
		pkt = &Wind{}
	}

	if pkt != nil {
		pkt.Receive(data)
	}
	return pkt, nil
}
