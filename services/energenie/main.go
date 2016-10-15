// Service to communicate with energenie sockets. This can both receive and
// transmit events.
package energenie

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/barnybug/ener314"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

func handleCommand(ev *pubsub.Event) {
	dev := ev.Device()
	command := ev.Command()
	pids := services.Config.LookupDeviceProtocol(dev)
	if pids["energenie"] == "" {
		return // command not for us
	}
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
}

// Service energenie
type Service struct{}

func (self *Service) ID() string {
	return "energenie"
}

func round(f float64, dp int) float64 {
	shift := math.Pow(10, float64(dp))
	return math.Floor(f*shift+.5) / shift
}

func emitTemp(msg *ener314.Message, record ener314.Temperature) {
	source := fmt.Sprintf("energenie.%06x", msg.SensorId)
	value := record.Value
	fields := map[string]interface{}{
		"source": source,
		"temp":   round(value, 1),
	}
	ev := pubsub.NewEvent("temp", fields)
	services.Publisher.Emit(ev)
}

func (self *Service) Run() error {
	ener314.SetLevel(ener314.LOG_TRACE)
	dev := ener314.NewDevice()
	err := dev.Start()
	if err != nil {
		return err
	}

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
			emitTemp(msg, t)
		case ener314.Voltage:
			log.Printf("%06x Voltage: %.2fV\n", msg.SensorId, t.Value)
		case ener314.Diagnostics:
			log.Printf("%06x Diagnostic report: %s\n", msg.SensorId, t)
		}
	}
}
