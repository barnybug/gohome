package ener314

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testEncodingCases = []struct {
	record   Record
	encoding []byte
}{
	{Identify{}, []byte{'?' | 0x80, 0x00}},
	{Join{}, []byte{'j' | 0x80, 0x00}},
	{JoinReport{}, []byte{'j', 0x00}},
	{Voltage{}, []byte{'b' | 0x80, 0x00}},
	{Temperature{32}, []byte{'t' | 0x80, 0x92, 0x20, 0x00}},
	{Diagnostics{}, []byte{'&' | 0x80, 0x00}},
	{ExerciseValve{}, []byte{'#' | 0x80, 0x00}},
	{ReportInterval{300}, []byte{'R' | 0x80, 0x02, 0x01, 0x2c}},
	{SetValveState{VALVE_STATE_AUTO}, []byte{'%' | 0x80, 0x01, 0x02}},
	{SetPowerMode{POWER_MODE_LOW}, []byte{'$' | 0x80, 0x01, 0x01}},
}

func TestEncoding(t *testing.T) {
	for _, tt := range testEncodingCases {
		var buf bytes.Buffer
		tt.record.Encode(&buf)
		assert.Equal(t, tt.encoding, buf.Bytes())
	}
}
