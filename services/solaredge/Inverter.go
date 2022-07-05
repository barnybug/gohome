/*

MIT License

Copyright (c) 2019 David Suarez

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
	"github.com/u-root/u-root/pkg/uio"
)

// InverterModel holds the SolarEdge SunSpec Implementation for Inverter parameters
// from the implementation technical note:
// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf
type InverterModel struct {
	SunSpec_DID     uint16
	SunSpec_Length  uint16
	AC_Current      uint16
	AC_CurrentA     uint16
	AC_CurrentB     uint16
	AC_CurrentC     uint16
	AC_Current_SF   int16
	AC_VoltageAB    uint16
	AC_VoltageBC    uint16
	AC_VoltageCA    uint16
	AC_VoltageAN    uint16
	AC_VoltageBN    uint16
	AC_VoltageCN    uint16
	AC_Voltage_SF   int16
	AC_Power        int16
	AC_Power_SF     int16
	AC_Frequency    uint16
	AC_Frequency_SF int16
	AC_VA           int16
	AC_VA_SF        int16
	AC_VAR          int16
	AC_VAR_SF       int16
	AC_PF           int16
	AC_PF_SF        int16
	AC_Energy_WH    int32
	AC_Energy_WH_SF uint16
	DC_Current      uint16
	DC_Current_SF   int16
	DC_Voltage      uint16
	DC_Voltage_SF   int16
	DC_Power        int16
	DC_Power_SF     int16
	Temp_Sink       int16
	Temp_SF         int16
	Status          uint16
	Status_Vendor   uint16
}

type MeterModel struct {
	SunSpec_DID       uint16
	SunSpec_Length    uint16
	M_AC_Current      uint16
	M_AC_CurrentA     uint16
	M_AC_CurrentB     uint16
	M_AC_CurrentC     uint16
	M_AC_Current_SF   int16
	M_AC_VoltageLN    uint16
	M_AC_VoltageAN    uint16
	M_AC_VoltageBN    uint16
	M_AC_VoltageCN    uint16
	M_AC_VoltageLL    uint16
	M_AC_VoltageAB    uint16
	M_AC_VoltageBC    uint16
	M_AC_VoltageCA    uint16
	M_AC_Voltage_SF   int16
	M_AC_Frequency    uint16
	M_AC_Frequency_SF int16
	M_AC_Power        int16
	M_AC_Power_A      int16
	M_AC_Power_B      int16
	M_AC_Power_C      int16
	M_AC_Power_SF     int16
	M_AC_VA           uint16
	M_AC_VA_A         uint16
	M_AC_VA_B         uint16
	M_AC_VA_C         uint16
	M_AC_VA_SF        int16
	M_AC_VAR          uint16
	M_AC_VAR_A        uint16
	M_AC_VAR_B        uint16
	M_AC_VAR_C        uint16
	M_AC_VAR_SF       int16
	M_AC_PF           uint16
	M_AC_PF_A         uint16
	M_AC_PF_B         uint16
	M_AC_PF_C         uint16
	M_AC_PF_SF        int16
	M_Exported        uint32
	M_Exported_A      uint32
	M_Exported_B      uint32
	M_Exported_C      uint32
	M_Imported        uint32
	M_Imported_A      uint32
	M_Imported_B      uint32
	M_Imported_C      uint32
	M_Energy_W_SF     int16
}

// NewCommonModel takes block of data read from the Modbus TCP connection and returns a new populated struct
func NewInverterModel(data []byte) (InverterModel, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) != 80 {
		return InverterModel{}, errors.New("improper data size")
	}

	im := InverterModel{
		SunSpec_DID:     buf.Read16(),
		SunSpec_Length:  buf.Read16(),
		AC_Current:      buf.Read16(),
		AC_CurrentA:     buf.Read16(),
		AC_CurrentB:     buf.Read16(),
		AC_CurrentC:     buf.Read16(),
		AC_Current_SF:   int16(buf.Read16()),
		AC_VoltageAB:    buf.Read16(),
		AC_VoltageBC:    buf.Read16(),
		AC_VoltageCA:    buf.Read16(),
		AC_VoltageAN:    buf.Read16(),
		AC_VoltageBN:    buf.Read16(),
		AC_VoltageCN:    buf.Read16(),
		AC_Voltage_SF:   int16(buf.Read16()),
		AC_Power:        int16(buf.Read16()),
		AC_Power_SF:     int16(buf.Read16()),
		AC_Frequency:    buf.Read16(),
		AC_Frequency_SF: int16(buf.Read16()),
		AC_VA:           int16(buf.Read16()),
		AC_VA_SF:        int16(buf.Read16()),
		AC_VAR:          int16(buf.Read16()),
		AC_VAR_SF:       int16(buf.Read16()),
		AC_PF:           int16(buf.Read16()),
		AC_PF_SF:        int16(buf.Read16()),
		AC_Energy_WH:    int32(buf.Read32()),
		AC_Energy_WH_SF: buf.Read16(),
		DC_Current:      buf.Read16(),
		DC_Current_SF:   int16(buf.Read16()),
		DC_Voltage:      buf.Read16(),
		DC_Voltage_SF:   int16(buf.Read16()),
		DC_Power:        int16(buf.Read16()),
		DC_Power_SF:     int16(buf.Read16()),
	}
	buf.Read16() // Skip address as per SunSpec Technical Note
	im.Temp_Sink = int16(buf.Read16())
	buf.Read16() // Skip address as per SunSpec Technical Note
	buf.Read16() // Skip address as per SunSpec Technical Note
	im.Temp_SF = int16(buf.Read16())
	im.Status = buf.Read16()
	im.Status_Vendor = buf.Read16()

	return im, nil
}

func NewMeterModel(data []byte) (MeterModel, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) <= 10 {
		return MeterModel{}, errors.New("improper data size")
	}

	im := MeterModel{
		SunSpec_DID:       buf.Read16(),
		SunSpec_Length:    buf.Read16(),
		M_AC_Current:      buf.Read16(),
		M_AC_CurrentA:     (buf.Read16()),
		M_AC_CurrentB:     (buf.Read16()),
		M_AC_CurrentC:     (buf.Read16()),
		M_AC_Current_SF:   int16(buf.Read16()),
		M_AC_VoltageLN:    buf.Read16(),
		M_AC_VoltageAN:    buf.Read16(),
		M_AC_VoltageBN:    buf.Read16(),
		M_AC_VoltageCN:    buf.Read16(),
		M_AC_VoltageLL:    buf.Read16(),
		M_AC_VoltageAB:    buf.Read16(),
		M_AC_VoltageBC:    buf.Read16(),
		M_AC_VoltageCA:    buf.Read16(),
		M_AC_Voltage_SF:   int16(buf.Read16()),
		M_AC_Frequency:    buf.Read16(),
		M_AC_Frequency_SF: int16(buf.Read16()),
		M_AC_Power:        int16(buf.Read16()),
		M_AC_Power_A:      int16(buf.Read16()),
		M_AC_Power_B:      int16(buf.Read16()),
		M_AC_Power_C:      int16(buf.Read16()),
		M_AC_Power_SF:     int16(buf.Read16()),
		M_AC_VA:           buf.Read16(),
		M_AC_VA_A:         buf.Read16(),
		M_AC_VA_B:         buf.Read16(),
		M_AC_VA_C:         buf.Read16(),
		M_AC_VA_SF:        int16(buf.Read16()),
		M_AC_VAR:          buf.Read16(),
		M_AC_VAR_A:        buf.Read16(),
		M_AC_VAR_B:        buf.Read16(),
		M_AC_VAR_C:        buf.Read16(),
		M_AC_VAR_SF:       int16(buf.Read16()),
		M_AC_PF:           buf.Read16(),
		M_AC_PF_A:         buf.Read16(),
		M_AC_PF_B:         buf.Read16(),
		M_AC_PF_C:         buf.Read16(),
		M_AC_PF_SF:        int16(buf.Read16()),
		M_Exported:        buf.Read32(),
		M_Exported_A:      buf.Read32(),
		M_Exported_B:      buf.Read32(),
		M_Exported_C:      buf.Read32(),
		M_Imported:        buf.Read32(),
		M_Imported_A:      buf.Read32(),
		M_Imported_B:      buf.Read32(),
		M_Imported_C:      buf.Read32(),
		M_Energy_W_SF:     int16(buf.Read16()),
	}

	return im, nil
}
