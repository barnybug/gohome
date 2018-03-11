// Service to retrieve sensors data for Mi Flora bluetooth sensors
package miflora

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/miflora"
)

const MaxRetries = 5

// Service miflora
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "miflora"
}

var adapter = "hci0"

var devices = map[string]*miflora.Miflora{}

func sendEvent(mac string, sensors miflora.Sensors) {
	source := fmt.Sprintf("miflora.%s", mac)
	fields := map[string]interface{}{
		"source":       source,
		"temp":         sensors.Temperature,
		"moisture":     sensors.Moisture,
		"light":        sensors.Light,
		"conductivity": sensors.Conductivity,
	}
	ev := pubsub.NewEvent("temp", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
}

func iterateSensors(f func(mac, name string) error) {
	for mac, name := range services.Config.Protocols["miflora"] {
		for i := 0; i < MaxRetries; i += 1 {
			err := f(mac, name)
			if err == nil {
				break
			}
			if i == MaxRetries-1 {
				// last retry
				log.Printf("Failed to read %s after %d retries: %s", name, MaxRetries, err)
			}
		}
	}

}

func checkEvent(sensors miflora.Sensors) bool {
	if sensors.Temperature < -20 || sensors.Temperature > 50 {
		return false
	}
	if sensors.Moisture > 100 {
		return false
	}
	if sensors.Conductivity > 1000 {
		return false
	}
	return true
}

func readSensors() {
	iterateSensors(func(mac, name string) error {
		dev, ok := devices[mac]
		if !ok {
			dev = miflora.NewMiflora(mac, adapter)
			firmware, err := dev.ReadFirmware()
			if err != nil {
				return err
			}
			log.Printf("%s: Firmware: %+v", name, firmware)
			devices[mac] = dev
		}

		sensors, err := dev.ReadSensors()
		if err == nil {
			// send data
			log.Printf("%s: %+v\n", name, sensors)
			if !checkEvent(sensors) {
				log.Printf("Ignoring sensor data outside sensible ranges: %+v", sensors)
			}
			sendEvent(mac, sensors)
		}
		return err
	})
}

// Run the service
func (self *Service) Run() error {
	readSensors()
	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		readSensors()
	}
	return nil
}
