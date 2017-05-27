package ener314

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var decodeFloat64Table = []struct {
	code     byte
	value    []byte
	expected float64
}{
	// 0000 Unsigned x.0 normal integer
	{0x01, []byte{0x80}, 128},
	{0x02, []byte{0x12, 0x80}, 4736},
	// 0001 Unsigned x.4 fixed point integer
	{0x12, []byte{0x12, 0x80}, 296},
	// 0010 Unsigned x.8 fixed point integer
	{0x22, []byte{0x12, 0x80}, 18.5},
	// 0011 Unsigned x.12 fixed point integer
	{0x32, []byte{0x12, 0x80}, 1.15625},
	// 0011 Unsigned x.16 fixed point integer
	{0x42, []byte{0x80, 0x00}, 0.5},
	// 0101 Unsigned x.20 fixed point integer
	{0x53, []byte{0x04, 0x00, 0x00}, 0.25},
	// 0110 Unsigned x.24 fixed point integer
	{0x63, []byte{0x80, 0x00, 0x00}, 0.5},
	// 0111 Characters
	{0x72, []byte{0x34, 0x32}, 42},
	// 1000 Signed x.0 normal integer
	{0x82, []byte{0x92, 0x80}, -4736},
	{0x82, []byte{0x12, 0x80}, 4736},
	// 1001 Signed x.8 fixed point integer
	{0x92, []byte{0x92, 0x80}, -18.5},
	{0x92, []byte{0x12, 0x80}, 18.5},
	// 1010 Signed x.16 fixed point integer
	{0xa2, []byte{0xc0, 0x00}, -0.25},
	{0xa2, []byte{0x40, 0x00}, 0.25},
	// 1011 Signed x.24 fixed point integer
	{0xb2, []byte{0xc0, 0x00, 0x00}, -0.25},
	{0xb2, []byte{0x40, 0x00, 0x00}, 0.25},
	// 1100 Enumeration
	{0xc1, []byte{0x34}, 52},
	// 1101 Reserved
	{0xd1, []byte{0x34}, 0},
	// 1110 Reserved
	{0xe1, []byte{0x34}, 0},
}

func TestDecodeFloat64(t *testing.T) {
	for _, tt := range decodeFloat64Table {
		assert.Equal(t, tt.expected, decodeFloat64(tt.code, tt.value))
	}
}

var encodeFloat64Table = []struct {
	enc      byte
	value    float64
	expected []byte
}{
	// 0000 Unsigned x.0 normal integer
	{ENC_UINT, 128, []byte{0x01, 0x80}},
	{ENC_UINT, 360, []byte{0x02, 0x01, 0x68}},
	{ENC_UINT, 75900, []byte{0x03, 0x1, 0x28, 0x7c}},

	// 1001 Signed x.8 fixed point integer
	{ENC_SFPp8, 9.5, []byte{0x92, 0x09, 0x80}},
	{ENC_SFPp8, 10.0, []byte{0x92, 0x0a, 0x00}},
	{ENC_SFPp8, 10.0625, []byte{0x92, 0x0a, 0x10}},
	{ENC_SFPp8, 10.125, []byte{0x92, 0x0a, 0x20}},
	{ENC_SFPp8, 10.25, []byte{0x92, 0x0a, 0x40}},
	{ENC_SFPp8, 10.5, []byte{0x92, 0x0a, 0x80}},
	{ENC_SFPp8, 18.5, []byte{0x92, 0x12, 0x80}},
	{ENC_SFPp8, 256.0, []byte{0x93, 0x01, 0x00, 0x00}},
}

func TestEncodeFloat64(t *testing.T) {
	for _, tt := range encodeFloat64Table {
		assert.Equal(t, tt.expected, encodeFloat64(tt.enc, tt.value))
	}
}

func ExampleDecodePacketJoin() {
	packet := []byte{0x04, 0x03, 0x04, 0x42, 0xd1, 0xf8, 0x17, 0x05, 0xd1, 0xd9, 0x0f, 0x30}
	cryptPacket(packet)
	message, _ := decodePacket(packet)
	fmt.Println(message)
	// Output:
	// {ManuId:4 ProdId:3 SensorId:00097f Records:[Join]}
}

func ExampleDecodePacketVoltage() {
	packet := []byte{0x04, 0x03, 0x13, 0x04, 0x20, 0x3b, 0x19, 0xd5, 0x8c, 0xf1, 0x5f, 0xf1, 0xd3, 0x7b}
	cryptPacket(packet)
	message, _ := decodePacket(packet)
	fmt.Println(message)
	// Output:
	// {ManuId:4 ProdId:3 SensorId:00097f Records:[Voltage{3.121094}]}
}

func ExampleDecodePacketTemp() {
	packet := []byte{0x04, 0x03, 0x0f, 0x42, 0x89, 0x00, 0x3a, 0x46, 0x9c, 0xa6, 0xe2, 0x35, 0x1f, 0xdc}
	cryptPacket(packet)
	message, _ := decodePacket(packet)
	fmt.Println(message)
	// Output:
	// {ManuId:4 ProdId:3 SensorId:00097f Records:[Temperature{17.699219}]}
}

func ExampleDecodePacketDiagnostics() {
	packet, _ := hex.DecodeString("0403704d00097f2602020000ed6a")
	message, _ := decodePacket(packet)
	fmt.Println(message)
	// Output:
	// {ManuId:4 ProdId:3 SensorId:00097f Records:[Diagnostics{512,[Valve exercise was successful]}]}
}

func ExampleCRCFailure() {
	packet := []byte{0x04, 0x03, 0x04, 0x42, 0xd1, 0xf8, 0x17, 0x05, 0xd1, 0xd9, 0x0f, 0x31}
	cryptPacket(packet)
	_, err := decodePacket(packet)
	fmt.Println(err)
	// Output:
	// CRC fail
}

var badPacketsTable = []string{
	// various truncations
	"",
	"04",
	"04 03",
	"04 03 49",
	"04 03 49 10",
	"04 03 49 10 00",
	"04 03 49 10 00 09",
	"04 03 49 10 00 09 7f",
	"04 03 49 10 00 09 7f 74",
	"04 03 49 10 00 09 7f 74 92",
	"04 03 49 10 00 09 7f 74 92 12",
	"04 03 49 10 00 09 7f 74 92 12 33",
	"04 03 49 10 00 09 7f 74 92 12 33 00",
	"04 03 49 10 00 09 7f 74 92 12 33 00 69",
	// zero length number
	"04 03 07 61 00 09 7f 74 90 92 12 00 00 39 27",
	// junk
	"04 9f 81 2c e2 de 45 5c 05 76 40 ab 22 6c af e4 9a 85 bb b4 78 3e b8 f9 83 d7 be a3 7e",
}

func TestBadPackets(t *testing.T) {
	for _, tt := range badPacketsTable {
		data, err := hex.DecodeString(strings.Replace(tt, " ", "", -1))
		if err != nil {
			assert.NoError(t, err, "decode hex")
		}
		ret, err := decodePacket(data)
		assert.Error(t, err, "error")
		assert.Nil(t, ret, "decodes to nil")
	}
}

func ExampleEncodeMessageJoin() {
	message := Message{
		ManuId: 0x04, ProdId: 0x03, SensorId: 0x00098b,
		Records: []Record{Join{}},
	}
	data := encodeMessage(&message)
	fmt.Println(hex.EncodeToString(data))
	// Output:
	// 0403000000098bea00000cab
}
