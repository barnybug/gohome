/*

MIT License

Copyright (c) 2022 Barnaby Gray

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
	"fmt"

	"github.com/goburrow/modbus"
	"github.com/u-root/u-root/pkg/uio"
)

const AddressBattery1 = 0xE100

// BatteryInfo holds the SolarEdge SunSpec Implementation for Battery parameters
// from the implementation technical note:
// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf
// With help from here:
// https://github.com/binsentsu/home-assistant-solaredge-modbus/
type BatteryInfo struct {
	Manufacturer                []byte
	Model                       []byte
	Firmware                    []byte
	Serial                      []byte
	DeviceID                    uint16
	RatedEnergy                 float32
	MaxPowerContinuousCharge    float32
	MaxPowerContinuousDischarge float32
	MaxPowerPeakCharge          float32
	MaxPowerPeakDischarge       float32
}

func (bm BatteryInfo) String() string {
	return fmt.Sprintf("Manufacturer: %s Model: %s Firmware: %s Serial: %s", bm.Manufacturer, bm.Model, bm.Firmware, bm.Serial)
}

func ReadBatteryInfo(client modbus.Client) (BatteryInfo, error) {
	data, err := client.ReadHoldingRegisters(AddressBattery1, 76)
	if err != nil {
		return BatteryInfo{}, err
	}
	return ParseBatteryInfo(data)
}

// ParseBatteryInfo takes block of data read from the Modbus TCP connection and returns a new populated struct
func ParseBatteryInfo(data []byte) (BatteryInfo, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) != 152 {
		return BatteryInfo{}, errors.New("improper data size")
	}
	b := BatteryInfo{}

	manu, _ := buf.ReadN(32)
	model, _ := buf.ReadN(32)
	firmware, _ := buf.ReadN(32)
	serial, _ := buf.ReadN(32)
	b.Manufacturer = bytes.Trim(manu, "\x00")
	b.Model = bytes.Trim(model, "\x00")
	b.Firmware = bytes.Trim(firmware, "\x00")
	b.Serial = bytes.Trim(serial, "\x00")

	b.DeviceID = buf.Read16()
	buf.Read16() // Reserved
	b.RatedEnergy = decode_float32(buf)
	b.MaxPowerContinuousCharge = decode_float32(buf)
	b.MaxPowerContinuousDischarge = decode_float32(buf)
	b.MaxPowerPeakCharge = decode_float32(buf)
	b.MaxPowerPeakDischarge = decode_float32(buf)
	return b, nil
}

type BatteryData struct {
	TempAvg          float32
	TempMax          float32 // zero
	Voltage          float32
	Current          float32
	Power            float32
	Discharged       uint64
	Charged          uint64
	BatteryMax       float32
	BatteryAvailable float32
	BatterySoH       float32
	BatterySoC       float32
	Status           uint16
}

func ReadBatteryData(client modbus.Client) (BatteryData, error) {
	data, err := client.ReadHoldingRegisters(AddressBattery1+0x6C, 28)
	if err != nil {
		return BatteryData{}, err
	}
	return ParseBatteryData(data)
}

var BatteryStatuses = map[uint16]string{
	3:  "Charging",
	4:  "Discharging",
	6:  "Idle",
	10: "Sleep",
}

func ParseBatteryData(data []byte) (BatteryData, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) != 56 {
		return BatteryData{}, errors.New("improper data size")
	}

	b := BatteryData{
		TempAvg:          decode_float32(buf),
		TempMax:          decode_float32(buf),
		Voltage:          decode_float32(buf),
		Current:          decode_float32(buf),
		Power:            decode_float32(buf),
		Discharged:       buf.Read64(),
		Charged:          buf.Read64(),
		BatteryMax:       decode_float32(buf),
		BatteryAvailable: decode_float32(buf),
		BatterySoH:       decode_float32(buf),
		BatterySoC:       decode_float32(buf),
		Status:           buf.Read16(),
	}

	return b, nil
}
