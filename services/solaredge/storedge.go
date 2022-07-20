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
	"errors"

	"github.com/goburrow/modbus"
	"github.com/u-root/u-root/pkg/uio"
)

const AddressStoredgeControl = 0xE004

// ControlInfo holds the SolarEdge SunSpec Implementation for Control parameters
// from the implementation technical note:
// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf
// With help from here:
// https://github.com/binsentsu/home-assistant-solaredge-modbus/
type ControlInfo struct {
	ControlMode          ControlMode
	ACChargePolicy       string
	ACChargeLimit        float32
	BackupReserved       float32
	DefaultMode          ChargeDischargeMode
	RemoteTimeout        uint32
	RemoteMode           ChargeDischargeMode
	RemoteChargeLimit    float32
	RemoteDischargeLimit float32
}

func ReadControlInfo(client modbus.Client) (ControlInfo, error) {
	data, err := client.ReadHoldingRegisters(AddressStoredgeControl, 14)
	if err != nil {
		return ControlInfo{}, err
	}
	return ParseControlInfo(data)
}

type ControlMode uint16

const (
	ControlModeDisabled ControlMode = iota
	ControlModeMaximizeSelfConsumption
	ControlModeTimeOfUse
	ControlModeBackupOnly
	ControlModeRemoteControl
)

func (c ControlMode) String() string {
	return []string{"Disabled", "Maximize Self Consumption", "Time of Use", "Backup Only", "Remote Control"}[c]
}

var ACChargePolicy = map[uint16]string{
	0: "Disabled",
	1: "Always Allowed",
	2: "Fixed Energy Limit",
	3: "Percent of Production",
}

type ChargeDischargeMode uint16

const (
	Off ChargeDischargeMode = iota
	ChargeFromExcessPVPowerOnly
	ChargeFromPVFirst
	ChargeFromPVAndAC
	MaximizeExport
	DischargeToMatchLoad
	Unused
	MaximizeSelfConsumption
)

func (c ChargeDischargeMode) String() string {
	return []string{"Off", "Charge from excess PV power only", "Charge from PV first", "Charge from PV and AC", "Maximize export", "Discharge to match load", "Unused", "Maximize self consumption"}[c]
}

// ParseControlInfo takes block of data read from the Modbus TCP connection and returns a new populated struct
func ParseControlInfo(data []byte) (ControlInfo, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) != 28 {
		return ControlInfo{}, errors.New("improper data size")
	}
	b := ControlInfo{
		ControlMode:          ControlMode(buf.Read16()),
		ACChargePolicy:       ACChargePolicy[buf.Read16()],
		ACChargeLimit:        decode_float32(buf),
		BackupReserved:       decode_float32(buf),
		DefaultMode:          ChargeDischargeMode(buf.Read16()),
		RemoteTimeout:        decode_bele32(buf),
		RemoteMode:           ChargeDischargeMode(buf.Read16()),
		RemoteChargeLimit:    decode_float32(buf),
		RemoteDischargeLimit: decode_float32(buf),
	}

	return b, nil
}
