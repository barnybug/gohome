// Service to communicate with sonoff tasmota firmware devices through MQTT.
// Supports:
// - on
// - off
// - power readings (Sonoff POW)
package tasmota

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/mqtt"
	"github.com/barnybug/gohome/services"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func handleCommand(ev *pubsub.Event) {
	dev := ev.Device()
	command := ev.Command()
	ident, ok := services.Config.LookupDeviceProtocol(dev, "tasmota")
	if !ok {
		return // command not for us
	}
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
	log.Printf("Setting device %s to %s\n", dev, command)
	topic := fmt.Sprintf("tasmota/cmnd/%s/power", ident)
	token := mqtt.Client.Publish(topic, 1, false, command)
	if token.Wait() && token.Error() != nil {
		log.Println("Failed to publish message:", token.Error())
	}
}

type Energy struct {
	Total     float64
	Yesterday float64
	Today     float64
	Period    float64
	Power     float64
	Factor    float64
	Voltage   float64
	Current   float64
}

func handleEnergy(message MQTT.Message) {
	var dst Energy
	err := json.Unmarshal(message.Payload(), &dst)
	if err != nil {
		log.Printf("Failed to decode Energy message: %s", err)
		return
	}
	ps := strings.Split(message.Topic(), "/")
	// tasmota/tele/<name>/ENERGY
	name := ps[len(ps)-2]
	fields := pubsub.Fields{
		"source":    fmt.Sprintf("tasmota.%s", name),
		"yesterday": dst.Yesterday,
		"today":     dst.Today,
		"period":    dst.Period,
		"power":     dst.Power,
		"factor":    dst.Factor,
		"voltage":   dst.Voltage,
		"current":   dst.Current,
	}
	ev := pubsub.NewEvent("power", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
}

func handlePower(message MQTT.Message) {
	ps := strings.Split(message.Topic(), "/")
	// tasmota/tele/<name>/POWER
	name := ps[len(ps)-2]
	command := strings.ToLower(string(message.Payload()))
	fields := pubsub.Fields{
		"source":  fmt.Sprintf("tasmota.%s", name),
		"command": command,
	}
	ev := pubsub.NewEvent("tasmota", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
}

// Service tasmota
type Service struct{}

func (self *Service) ID() string {
	return "tasmota"
}

func (self *Service) Run() error {
	commandChannel := services.Subscriber.FilteredChannel("command")
	messageChannel := make(chan MQTT.Message)
	mqtt.Client.Subscribe("tasmota/#", 1, func(client MQTT.Client, message MQTT.Message) {
		messageChannel <- message
	})

	for {
		select {
		case command := <-commandChannel:
			handleCommand(command)
		case message := <-messageChannel:
			if strings.HasSuffix(message.Topic(), "/ENERGY") {
				handleEnergy(message)
			}
			if strings.HasSuffix(message.Topic(), "/POWER") {
				handlePower(message)
			}
		}
	}
}
