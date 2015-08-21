package gorfxtrx

import (
	"errors"
	"fmt"
	"strconv"
)

// Struct for X10 Lighting packets.
type LightingX10 struct {
	typeId         byte
	SequenceNumber byte
	HouseCode      byte
	UnitCode       byte
	command        byte
}

var lightingX10Types = map[byte]string{
	0x00: "X10 lighting",
	0x01: "ARC",
	0x02: "ELRO AB400D",
	0x03: "Waveman",
	0x04: "Chacon EMW200",
	0x05: "IMPULS",
	0x06: "RisingSun",
	0x07: "Philips SBC",
}

var lightingX10HouseCodes = map[byte]string{
	0x41: "a", 0x42: "b", 0x43: "c", 0x44: "d",
	0x45: "e", 0x46: "f", 0x47: "g", 0x48: "h",
	0x49: "i", 0x4A: "j", 0x4B: "k", 0x4C: "l",
	0x4D: "m", 0x4E: "n", 0x4F: "o", 0x50: "p",
}

var lightingX10HouseCodeBytes = reverseByteStringMap(lightingX10HouseCodes)

var lightingX10Commands = map[byte]string{
	0x00: "off",
	0x01: "on",
	0x02: "dim",
	0x03: "bright",
	0x05: "group off",
	0x06: "group on",
	0x07: "chime",
	0xFF: "illegal command",
}

var lightingX10CommandBytes = reverseByteStringMap(lightingX10Commands)

func NewLightingX10(typeId byte, id string, command string) (*LightingX10, error) {
	if len(id) != 3 {
		return nil, errors.New("id should be 3 characters (eg. a01)")
	}
	houseCode := lightingX10HouseCodeBytes[id[0:1]]
	unitCode, err := strconv.ParseUint(id[1:], 16, 8)
	if err != nil {
		return nil, err
	}
	return &LightingX10{
		typeId:    typeId,
		HouseCode: houseCode,
		UnitCode:  byte(unitCode),
		command:   lightingX10CommandBytes[command],
	}, nil
}

func (self *LightingX10) Receive(data []byte) {
	self.typeId = data[2]
	self.SequenceNumber = data[3]
	self.HouseCode = data[4]
	self.UnitCode = data[5]
	self.command = data[6]
}

// Type of the device.
func (self *LightingX10) Type() string {
	return lightingX10Types[self.typeId]
}

// Id of the device.
func (self *LightingX10) Id() string {
	return fmt.Sprintf("%s%02x", lightingX10HouseCodes[self.HouseCode], self.UnitCode)
}

// Command transmitted.
func (self *LightingX10) Command() string {
	return lightingX10Commands[self.command]
}

func (self *LightingX10) Send() []byte {
	return []byte{0x07, 0x10, self.typeId, self.SequenceNumber,
		self.HouseCode, self.UnitCode, self.command, 0}
}
