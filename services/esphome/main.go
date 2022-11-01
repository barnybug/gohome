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

func announce(source string) {
	fields := pubsub.Fields{"source": source}
	ev := pubsub.NewEvent("announce", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
}

func source(ps []string) string {
	source := ps[1] // "flora.garden3"
	if !strings.Contains(source, ".") {
		source = "esphome." + source
	}
	if ps[2] == "switch" {
		source += ":" + ps[3]
	}
	return source
}

func translate(message MQTT.Message) *pubsub.Event {
	// esphome/flora.garden3/sensor/conductivity/state
	// esphome/sonoff1/switch/relay/state
	ps := strings.Split(message.Topic(), "/")
	last := ps[len(ps)-1]
	if last == "debug" || last == "command" {
		return nil
	}
	source := source(ps)
	payload := string(message.Payload())
	if last == "status" {
		log.Printf("%s status %s", source, payload)
		return nil
	}
	if last != "state" && last != "status" {
		log.Printf("Ignoring unknown topic: %s", message.Topic())
		return nil
	}

	var topic, field string
	if ps[2] == "switch" {
		topic = "ack"
		field = "command"
	} else {
		field = ps[3] // "conductivity"
		topic = field
	}
	fields := pubsub.Fields{
		"source": source,
	}
	if payload == "ON" || payload == "OFF" {
		fields[field] = strings.ToLower(payload)
	} else if value, err := strconv.ParseFloat(payload, 64); err == nil {
		fields[field] = value
	} else {
		fields[field] = payload
	}
	ev := pubsub.NewEvent(topic, fields)
	services.Config.AddDeviceToEvent(ev)
	if topic == "ack" && ev.Device() != "" {
		log.Printf("Set device %s to %s", ev.Device(), ev.Command())
	}
	return ev
}

func handleCommand(ev *pubsub.Event) {
	source, ok := services.Config.LookupDeviceProtocol(ev.Device(), "esphome")
	if !ok {
		return // command not for us
	}
	command := ev.Command()

	// relay all commands?
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
	log.Printf("Setting device %s to %s\n", ev.Device(), command)

	sub := "relay" // default
	ps := strings.SplitN(source, ":", 2)
	if len(ps) == 2 { // multiple switches
		source = ps[0]
		sub = ps[1]
	}
	topic := fmt.Sprintf("esphome/%s/switch/%s/command", source, sub)
	token := mqtt.Client.Publish(topic, 1, false, strings.ToUpper(command))
	if token.Wait() && token.Error() != nil {
		log.Println("Failed to publish message:", token.Error())
	}
}

func (self *Service) Run() error {
	commandChannel := services.Subscriber.Subscribe(pubsub.Prefix("command"))
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
