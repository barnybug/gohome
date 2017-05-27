package main

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/ener314"
)

func fatalIfErr(err error) {
	if err != nil {
		panic(fmt.Sprint("Error:", err))
	}
}

func main() {
	ener314.SetLevel(ener314.LOG_TRACE)
	dev := ener314.NewDevice()
	err := dev.Start()
	fatalIfErr(err)

	log.Printf("Device temperature (approx): %dC", dev.GetTemperature())

	for {
		// poll receive
		msg := dev.Receive()
		if msg == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		record := msg.Records[0] // only examine first record
		switch t := record.(type) {
		case ener314.Join:
			log.Printf("%06x Join\n", msg.SensorId)
			dev.Join(msg.SensorId)
		case ener314.Temperature:
			log.Printf("%06x Temperature: %.2f°C\n", msg.SensorId, t.Value)
			// dev.TargetTemperature(msg.SensorId, 10)
			// dev.Voltage(msg.SensorId)
			// dev.Diagnostics(msg.SensorId)
			// dev.SetValveState(msg.SensorId, ener314.VALVE_STATE_OPEN)
			// dev.ReportInterval(msg.SensorId, 300)
		case ener314.Voltage:
			log.Printf("%06x Voltage: %.2fV\n", msg.SensorId, t.Value)
		case ener314.Diagnostics:
			log.Printf("%06x Diagnostic report: %s\n", msg.SensorId, t)
		}

		log.Printf("Device temperature (approx): %dC", dev.GetTemperature())
		log.Printf("RSSI: %.1fdB", dev.GetRSSI())
	}
}
