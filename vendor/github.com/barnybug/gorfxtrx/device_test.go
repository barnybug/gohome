package gorfxtrx

import (
	"fmt"
)

func ExampleRead() {
	replay := [][]byte{
		[]byte{0x0d},
		[]byte{0x01, 0x00, 0x01},
		[]byte{0x02, 0x53, 0x3e, 0x00},
		[]byte{0x0c, 0x2f, 0x01, 0x01},
		[]byte{0x00, 0x00},
		[]byte{0x00, 0x00},
	}
	ser := NewMockSerialPort(replay)
	dev := Device{ser: ser, debug: false}
	packet, err := dev.Read()
	fmt.Printf("%+v %v\n", packet, err)
	packet, err = dev.Read()
	fmt.Printf("%+v %v\n", packet, err)
	// Output:
	// &{data:[13 1 0 1 2 83 62 0 12 47 1 1 0 0] TransceiverType:83 FirmwareVersion:62} <nil>
	// <nil> <nil>
}
