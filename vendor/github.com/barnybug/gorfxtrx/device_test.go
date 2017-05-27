package gorfxtrx

import (
	"fmt"
)

func ExampleRead() {
	data := []byte{0x14, 0x01, 0x00, 0x01, 0x03, 0x53, 0x09, 0x20, 0x00, 0x2f, 0x00, 0x01, 0x01, 0x1c, 0x01, 0x00, 0x49, 0x00, 0x00, 0x00, 0x00}
	replay := [][]byte{
		data[0:1],
		data[1:4],
		data[4:8],
		data[8:12],
		data[12:14],
		data[14:16],
		data[16:20],
		[]byte{0x00, 0x00},
		[]byte{0x00, 0x00},
	}
	ser := NewMockSerialPort(replay)
	dev := Device{ser: ser, debug: false}
	packet, err := dev.Read()
	fmt.Printf("%v %v\n", packet, err)
	packet, err = dev.Read()
	fmt.Printf("%v %v\n", packet, err)
	// Output:
	// Status: type: 433.92MHz transceiver: 83 firmware: 9 protocols: ac, arc, byron sx, homeeasy, oregon, x10 <nil>
	// <nil> <nil>
}
