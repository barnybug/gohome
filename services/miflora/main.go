// Service to retrieve sensors data for Mi Flora bluetooth sensors
package miflora

import (
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/miflora"
)

// Service miflora
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "miflora"
}

var adapter = "hci0"

func sendEvent(device string, sensors miflora.Sensors) {
	fields := map[string]interface{}{
		"device":       device,
		"temp":         sensors.Temperature,
		"moisture":     sensors.Moisture,
		"light":        sensors.Light,
		"conductivity": sensors.Conductivity,
	}
	ev := pubsub.NewEvent("temp", fields)
	services.Publisher.Emit(ev)
}

func readSensors() {
	for mac, device := range services.Config.Protocols["miflora"] {
		dev := miflora.NewMiflora(mac, adapter)
		for i := 0; i < 3; i += 1 {
			sensors, err := dev.ReadSensors()
			if err == nil {
				// send data
				log.Printf("%s: %+v (%d attempt)\n", device, sensors, i+1)
				sendEvent(device, sensors)
				break
			}
			if i == 2 {
				// last retry
				log.Printf("Failed to read %s after 3 retries: %s", device, err)
			}
		}
	}

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
