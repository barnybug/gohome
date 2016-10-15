// Service to communicate with energenie sockets. This can both receive and
// transmit events.
package energenie

import (
	"fmt"
	"log"
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
	// if device, ok := devices[pids["energenie"]]; ok {
	// 	log.Printf("Setting device %s to %s\n", dev, command)
	// 	SetState(device, command == "on")
	// } else {
	// 	log.Println("Device not recognised:", device)
	// }
}

// func handleStateChange(msg *StateChangedMessage) {
// 	log.Printf("Device %s changed to %t\n", msg.Device.MACAddress, msg.State)

// 	source := msg.Device.MACAddress
// 	command := "off"
// 	if msg.State {
// 		command = "on"
// 	}
// 	fields := map[string]interface{}{
// 		"source":  source,
// 		"command": command,
// 	}
// 	ev := pubsub.NewEvent("energenie", fields)
// 	services.Publisher.Emit(ev)
// }

// Service energenie
type Service struct{}

func (self *Service) ID() string {
	return "energenie"
}

func emitTemp(msg *ener314.Message, record ener314.Temperature) {
	source := fmt.Sprintf("energenie.%06x", msg.SensorId)
	fields := map[string]interface{}{
		"source": source,
		"temp":   record.Value,
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
