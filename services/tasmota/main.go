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
	TotalStartTime string
	Total          float64
	Yesterday      float64
	Today          float64
	Period         float64
	Power          float64
	ApparentPower  float64
	ReactivePower  float64
	Factor         float64
	Voltage        float64
	Current        float64
}

type Sensor struct {
	Time   string
	ENERGY *Energy
}

func handleSensor(message MQTT.Message) {
	var sensor Sensor
	err := json.Unmarshal(message.Payload(), &sensor)
	if err != nil {
		log.Printf("Failed to decode SENSOR message: %s", err)
		return
	}
	if sensor.ENERGY == nil {
		return
	}
	en := sensor.ENERGY
	ps := strings.Split(message.Topic(), "/")
	// tasmota/tele/<name>/SENSOR
	name := ps[len(ps)-2]
	fields := pubsub.Fields{
		"source":    fmt.Sprintf("tasmota.%s", name),
		"yesterday": en.Yesterday,
		"today":     en.Today,
		"period":    en.Period,
		"power":     en.Power,
		"factor":    en.Factor,
		"voltage":   en.Voltage,
		"current":   en.Current,
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
	commandChannel := services.Subscriber.Subscribe(pubsub.Prefix("command"))
	messageChannel := make(chan MQTT.Message)
	mqtt.Client.Subscribe("tasmota/#", 1, func(client MQTT.Client, message MQTT.Message) {
		messageChannel <- message
	})

	for {
		select {
		case command := <-commandChannel:
			handleCommand(command)
		case message := <-messageChannel:
			ps := strings.Split(message.Topic(), "/")
			// tasmota/tele/<name>/SENSOR
			// tasmota/stat/<name>/POWER
			t1 := ps[1]
			t2 := ps[len(ps)-1]
			if t1 == "tele" && t2 == "SENSOR" {
				handleSensor(message)
			} else if t1 == "stat" && t2 == "POWER" {
				handlePower(message)
			}
		}
	}
}
