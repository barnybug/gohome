package ener314

import "fmt"

type Device struct {
	hrf *HRF
}

func NewDevice() *Device {
	return &Device{}
}

func (d *Device) Start() error {
	var err error

	logs(LOG_INFO, "Resetting...")
	d.hrf, err = NewHRF()
	if err != nil {
		return err
	}

	err = d.hrf.Reset()
	if err != nil {
		return err
	}

	version := d.hrf.GetVersion()
	if version != 36 {
		return fmt.Errorf("Unexpected version: %d", version)
	}

	logs(LOG_INFO, "Configuring FSK")
	err = d.hrf.ConfigFSK()
	if err != nil {
		return err
	}

	logs(LOG_INFO, "Wait for ready...")
	d.hrf.WaitFor(ADDR_IRQFLAGS1, MASK_MODEREADY, true)

	logs(LOG_INFO, "Clearing FIFO...")
	d.hrf.ClearFifo()
	return nil
}

func (d *Device) Receive() *Message {
	msg := d.hrf.ReceiveFSKMessage()
	if msg == nil {
		return nil
	}
	if msg.ManuId != energenieManuId {
		logf(LOG_WARN, "Warning: ignored message from manufacturer %d", msg.ManuId)
		return nil
	}
	if msg.ProdId != eTRVProdId {
		logf(LOG_WARN, "Warning: ignored message from product %d", msg.ProdId)
		return nil
	}
	if len(msg.Records) == 0 {
		logf(LOG_WARN, "Warning: ignoring message with 0 records")
		return nil
	}
	return msg
}

func (d *Device) Respond(sensorId uint32, record Record) {
	message := &Message{
		ManuId:   energenieManuId,
		ProdId:   eTRVProdId,
		SensorId: sensorId,
		Records:  []Record{record},
	}
	err := d.hrf.SendFSKMessage(message)
	if err != nil {
		logs(LOG_ERROR, "Error sending", err)
	}
}

func (d *Device) Identify(sensorId uint32) {
	d.Respond(sensorId, Identify{})
}

func (d *Device) Join(sensorId uint32) {
	d.Respond(sensorId, JoinReport{})
}

func (d *Device) Voltage(sensorId uint32) {
	d.Respond(sensorId, Voltage{})
}

func (d *Device) ExerciseValve(sensorId uint32) {
	d.Respond(sensorId, ExerciseValve{})
}

func (d *Device) Diagnostics(sensorId uint32) {
	d.Respond(sensorId, Diagnostics{})
}

func (d *Device) TargetTemperature(sensorId uint32, temp float64) {
	if temp < 4 || temp > 30 {
		logf(LOG_WARN, "Temperature out of range: 4 < %.2f < 30, refusing", temp)
		return
	}
	d.Respond(sensorId, Temperature{temp})
}

func (d *Device) ReportInterval(sensorId uint32, interval uint16) {
	if interval < 1 || interval > 3600 {
		logf(LOG_WARN, "Interval out of range: 1 < %.2f < 3600, refusing", interval)
		return
	}
	logf(LOG_INFO, "Setting report interval for device %06x to %ds", sensorId, interval)
	d.Respond(sensorId, ReportInterval{interval})
}
