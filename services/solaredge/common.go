/*
MIT License

# Copyright (c) 2019 David Suarez

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package solaredge

import (
	"bytes"
	"errors"
	"math"

	"github.com/u-root/u-root/pkg/uio"
)

// CommonModel holds the SolarEdge SunSpec Implementation for Common parameters
// from the implementation technical note:
// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf
type CommonModel struct {
	C_SunSpec_ID     uint32
	C_SunSpec_DID    uint16
	C_SunSpec_Length uint16
	C_Manufacturer   []byte
	C_Model          []byte
	C_Version        []byte // Version defined in SunSpec implementation note as String(16) however is incorrect
	C_SerialNumber   []byte
	C_DeviceAddress  uint16
}

type CommonMeter struct {
	C_SunSpec_DID    uint16
	C_SunSpec_Length uint16
	C_Manufacturer   []byte
	C_Model          []byte
	C_Option         []byte
	C_Version        []byte // Version defined in SunSpec implementation note as String(16) however is incorrect
	C_SerialNumber   []byte
	C_DeviceAddress  uint16
}

// NewCommonModel takes block of data read from the Modbus TCP connection and returns a new
// populated struct
func NewCommonModel(data []byte) (CommonModel, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) != 140 {
		return CommonModel{}, errors.New("Improper Data Size")
	}

	var cm CommonModel
	cm.C_Manufacturer = make([]byte, 32)
	cm.C_Model = make([]byte, 32)
	cm.C_Version = make([]byte, 32)
	cm.C_SerialNumber = make([]byte, 32)

	cm.C_SunSpec_ID = buf.Read32()
	cm.C_SunSpec_DID = buf.Read16()
	cm.C_SunSpec_Length = buf.Read16()
	buf.ReadBytes(cm.C_Manufacturer[:])
	buf.ReadBytes(cm.C_Model[:])
	buf.ReadBytes(cm.C_Version[:])
	buf.ReadBytes(cm.C_SerialNumber[:])

	cm.C_Manufacturer = bytes.Trim(cm.C_Manufacturer, "\x00")
	cm.C_Model = bytes.Trim(cm.C_Model, "\x00")
	cm.C_Version = bytes.Trim(cm.C_Version, "\x00")
	cm.C_SerialNumber = bytes.Trim(cm.C_SerialNumber, "\x00")

	return cm, nil
}

func NewCommonMeter(data []byte) (CommonMeter, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) < 100 {
		return CommonMeter{}, errors.New("Improper Data Size")
	}

	var cm CommonMeter
	cm.C_Manufacturer = make([]byte, 32)
	cm.C_Model = make([]byte, 32)
	cm.C_Version = make([]byte, 16)
	cm.C_Option = make([]byte, 16)
	cm.C_SerialNumber = make([]byte, 16)

	cm.C_SunSpec_DID = buf.Read16()
	cm.C_SunSpec_Length = buf.Read16()
	buf.ReadBytes(cm.C_Manufacturer[:])
	buf.ReadBytes(cm.C_Model[:])
	buf.ReadBytes(cm.C_Option[:])
	buf.ReadBytes(cm.C_Version[:])
	buf.ReadBytes(cm.C_SerialNumber[:])

	cm.C_Manufacturer = bytes.Trim(cm.C_Manufacturer, "\x00")
	cm.C_Model = bytes.Trim(cm.C_Model, "\x00")
	cm.C_Option = bytes.Trim(cm.C_Option, "\x00")
	cm.C_Version = bytes.Trim(cm.C_Version, "\x00")
	cm.C_SerialNumber = bytes.Trim(cm.C_SerialNumber, "\x00")

	return cm, nil
}

func decode_float32(buf *uio.Lexer) float32 {
	bits := decode_bele32(buf)
	return math.Float32frombits(bits)
}

func decode_bele32(buf *uio.Lexer) uint32 {
	// madness: words are BE, but longs the words LE of the words
	a := buf.Read16()
	b := buf.Read16()
	return uint32(b)<<16 + uint32(a)
}

func decode_bele64(buf *uio.Lexer) uint64 {
	// madness: words are BE, but longs the words LE of the words
	a := buf.Read16()
	b := buf.Read16()
	c := buf.Read16()
	d := buf.Read16()
	return uint64(d)<<48 + uint64(c)<<32 + uint64(b)<<16 + uint64(a)
}

func encode_bele32(buf *uio.Lexer, v uint32) {
	buf.Write16(uint16(v & 0xFFFF))
	buf.Write16(uint16(v >> 16))
}

func encode_float32(buf *uio.Lexer, v float32) {
	vb := math.Float32bits(v)
	encode_bele32(buf, vb)
}
