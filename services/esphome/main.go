// Service to translate esphome messages to/from gohome.
package esphome

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/barnybug/gohome/pubsub/mqtt"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Service esphome
type Service struct {
}

func (self *Service) ID() string {
	return "esphome"
}

func translate(message MQTT.Message) *pubsub.Event {
	// esphome/flora.garden3/conductivity/state
	// esphome/sonoff1.relay/state
	ps := strings.Split(message.Topic(), "/")
	last := ps[len(ps)-1]
	if last == "debug" || last == "status" || last == "command" {
		return nil
	}
	if last != "state" {
		log.Printf("Ignoring unknown topic: %s", message.Topic())
		return nil
	}

	i := len(ps) - 2
	if len(ps) == 3 {
		i += 1
	}
	sensor := ps[i]   // "conductivity"
	source := ps[i-1] // "flora.garden3"
	fields := pubsub.Fields{
		"source": source,
	}
	topic := sensor
	payload := string(message.Payload())
	if payload == "ON" || payload == "OFF" {
		fields[sensor] = strings.ToLower(payload)
	} else if value, err := strconv.ParseFloat(payload, 64); err == nil {
		fields[sensor] = value
	} else {
		fields[sensor] = payload
	}
	ev := pubsub.NewEvent(topic, fields)
	services.Config.AddDeviceToEvent(ev)
	return ev
}

func handleCommand(ev *pubsub.Event) {
	device := services.Config.Devices[ev.Device()]
	command := ev.Command()
	// relay all commands?
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
	log.Printf("Setting device %s to %s\n", device.Id, command)
	topic := fmt.Sprintf("esphome/%s/command", device.Source)
	token := mqtt.Client.Publish(topic, 1, false, strings.ToUpper(command))
	if token.Wait() && token.Error() != nil {
		log.Println("Failed to publish message:", token.Error())
	}
}

func (self *Service) Run() error {
	commandChannel := services.Subscriber.FilteredChannel("command")
	messageChannel := make(chan MQTT.Message)
	mqtt.Client.Subscribe("esphome/#", 1, func(client MQTT.Client, message MQTT.Message) {
		messageChannel <- message
	})
	for {
		select {
		case command := <-commandChannel:
			handleCommand(command)
		case message := <-messageChannel:
			ev := translate(message)
			if ev != nil {
				services.Publisher.Emit(ev)
			}
		}
	}
}
