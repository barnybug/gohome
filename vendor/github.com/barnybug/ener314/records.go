package ener314

import (
	"fmt"
	"io"
)

type ByteAndBytesWriter interface {
	io.Writer
	io.ByteWriter
}

type Record interface {
	String() string
	Encode(buf ByteAndBytesWriter)
}

type Join struct{}

func (j Join) String() string {
	return "Join"
}

func (j Join) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_JOIN_CMD)
	buf.WriteByte(0)
}

type Temperature struct {
	Value float64
}

func (t Temperature) String() string {
	return fmt.Sprintf("Temperature{%f}", t.Value)
}

func (t Temperature) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_TEMP_SET)
	buf.Write(encodeFloat64(ENC_SFPp8, t.Value))
}

type Voltage struct {
	Value float64
}

func (v Voltage) String() string {
	return fmt.Sprintf("Voltage{%f}", v.Value)
}

func (v Voltage) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_REQUEST_VOLTAGE)
	buf.WriteByte(0)
}

var DiagnosticTable = []string{
	// From LSB bit, LSB byte:
	"Motor current below expectation",
	"Motor current always high",
	"Motor taking too long",
	"Discrepancy between air and pipe sensors",
	"Air sensor out of expected range",
	"Pipe sensor out of expected range",
	"Low power mode is enabled",
	"No target temperature has been set by host",
	// MSB byte:
	"Valve may be sticking",
	"Valve exercise was successful",
	"Valve exercise was unsuccessful",
	"Driver micro has suffered a watchdog reset and needs data refresh",
	"Driver micro has suffered a noise reset and needs data refresh",
	"Battery voltage has fallen below 2p2V and valve has been opened",
	"Request for heat messaging is enabled",
	"Request for heat",
}

type Diagnostics struct {
	Value uint16
}

func (v Diagnostics) String() string {
	var messages []string
	for i, text := range DiagnosticTable {
		if v.Value&(1<<uint(i)) != 0 {
			messages = append(messages, text)
		}
	}
	return fmt.Sprintf("Diagnostics{%d,%s}", v.Value, messages)
}

func (v Diagnostics) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_REQUEST_DIAGNOSTICS)
	buf.WriteByte(0)
}

type UnhandledRecord struct {
	ID    byte
	Type  byte
	Value []byte
}

func (t UnhandledRecord) String() string {
	return fmt.Sprintf("Unhandled{%02x,%02x,%v}", t.ID, t.Type, t.Value)
}

func (t UnhandledRecord) Encode(buf ByteAndBytesWriter) {
	// Unhandled
}

// Commands

type Identify struct{}

func (i Identify) String() string {
	return "Identify"
}

func (i Identify) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_IDENTIFY)
	buf.WriteByte(0)
}

type JoinReport struct{}

func (i JoinReport) String() string {
	return "JoinReport"
}

func (i JoinReport) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_JOIN_RESP)
	buf.WriteByte(0)
}

type ExerciseValve struct{}

func (v ExerciseValve) String() string {
	return "ExerciseValve"
}

func (v ExerciseValve) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_EXERCISE_VALVE)
	buf.WriteByte(0)
}

type ReportInterval struct {
	Value uint16
}

func (v ReportInterval) String() string {
	return "ReportInterval"
}

func (v ReportInterval) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_SET_REPORTING_INTERVAL)
	buf.Write(encodeInteger(ENC_UINT, uint32(v.Value)))
}

type SetValveState struct {
	State ValveState
}

func (v SetValveState) String() string {
	return "SetValveState"
}

func (v SetValveState) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_SET_VALVE_STATE)
	buf.Write(encodeInteger(ENC_UINT, uint32(v.State)))
}

type SetPowerMode struct {
	Mode PowerMode
}

func (v SetPowerMode) String() string {
	return "SetPowerMode"
}

func (v SetPowerMode) Encode(buf ByteAndBytesWriter) {
	buf.WriteByte(OT_SET_LOW_POWER_MODE)
	buf.Write(encodeInteger(ENC_UINT, uint32(v.Mode)))
}
