package gorfxtrx

import "fmt"

// Struct for the TransmitAck packets.
type TransmitAck struct {
	Subtype uint8
	SeqNr   uint8
	State   uint8
}

var States = map[uint8]string{
	0:   "ACK",
	1:   "ACK_DELAYED",
	2:   "NACK",
	3:   "NACK_INVALID_AC_ADDRESS",
	255: "UNKNOWN",
}

func (self *TransmitAck) OK() bool {
	return self.State < 2
}

func (self *TransmitAck) Receive(data []byte) {
	// 04 02 01 00 00
	self.Subtype = data[2]
	self.SeqNr = data[3]
	self.State = data[4]
}

func (self *TransmitAck) String() string {
	return fmt.Sprintf("TransmitAck: %s", States[self.State])
}
