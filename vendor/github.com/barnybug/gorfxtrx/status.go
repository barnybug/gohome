package gorfxtrx

import (
	"fmt"
	"sort"
	"strings"
)

// Struct for the Status packet type.
type Status struct {
	data            []byte
	TransceiverType byte
	FirmwareVersion byte
}

var statusTypes = map[byte]string{
	0x50: "310MHz",
	0x51: "315MHz",
	0x53: "433.92MHz",
	0x55: "868.00MHz",
	0x56: "868.00MHz FSK",
	0x57: "868.30MHz",
	0x58: "868.30MHz FSK",
	0x59: "868.35MHz",
	0x5A: "868.35MHz FSK",
	0x5B: "868.95MHz",
}

// Type of connected device.
func (self *Status) TypeString() string {
	if statusTypes[self.TransceiverType] != "" {
		return statusTypes[self.TransceiverType]
	}
	return "unknown"
}

// Protocols enabled for the device.
func (self *Status) Protocols() []string {
	devs := []string{}
	devs = append(devs, decodeFlags(self.data[7], []string{"ae blyss", "rubicson", "fineoffset/viking", "lighting4", "rsl", "byron sx", "rfu6", "edisplay"})...)
	devs = append(devs, decodeFlags(self.data[8], []string{"mertik", "lightwarerf", "hideki", "lacrosse", "fs20", "proguard", "blindst0", "blindst1"})...)
	devs = append(devs, decodeFlags(self.data[9], []string{"x10", "arc", "ac", "homeeasy", "ikeakoppla", "oregon", "ati", "visonic"})...)
	sort.Strings(devs)
	return devs
}

func (self *Status) Receive(data []byte) {
	self.data = data
	self.TransceiverType = data[5]
	self.FirmwareVersion = data[6]
}

func (self *Status) Send() []byte {
	return []byte{0x0d, 0x00, 0x00, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
}

func (self *Status) String() string {
	protocols := strings.Join(self.Protocols(), ", ")
	return fmt.Sprintf("Status: type: %s transceiver: %d firmware: %d protocols: %s", self.TypeString(), self.TransceiverType, self.FirmwareVersion, protocols)
}
