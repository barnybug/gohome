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

func qualifySource(source string) string {
	if !strings.Contains(source, ".") {
		return "esphome." + source
	}
	return source
}

func source(ps []string) string {
	source := qualifySource(ps[1])
	// esphome/sonoff1/switch/relay/state
	// esphome/yagala1/switch/relay1/state
	if len(ps) > 3 && ps[2] == "switch" && ps[3] != "relay" && ps[3] != "start_measuring" {
		source += ":" + ps[3]
	}
	return source
}

var rollup = map[string]bool{
	"pm10":    true,
	"pm25":    true,
	"pm100":   true,
	"voltage": true,
	"current": true,
}
var rollupState = map[string]map[string]interface{}{}

func translate(message MQTT.Message) *pubsub.Event {
	// esphome/flora.garden3/sensor/conductivity/state
	ps := strings.Split(message.Topic(), "/")
	payload := string(message.Payload())
	last := ps[len(ps)-1]
	if len(ps) < 3 {
		return nil
	}
	if ps[1] == "discover" {
		source := qualifySource(ps[2])
		announce(source)
		return nil
	}
	switch last {
	case "debug", "command":
		return nil // ignored
	case "status":
		log.Printf("%s status %s", source(ps), payload)
		return nil
	case "state":
		// continue
	default:
		log.Printf("Ignoring unknown topic: %s", message.Topic())
		return nil
	}

	if len(ps) < 5 {
		return nil
	}
	source := source(ps)
	// esphome/gosund1/sensor/power/state
	topic := ps[3]
	field := ps[3]
	if topic == "start_measuring" {
		if payload == "ON" {
			return nil
		}
		topic = "pm"
	}
	if ps[2] == "switch" && strings.HasPrefix(field, "relay") {
		// esphome/sonoff1/switch/relay/state
		topic = "ack"
		field = "command"
	}
	var value interface{} = payload
	if payload == "ON" || payload == "OFF" {
		value = strings.ToLower(payload)
	} else if fval, err := strconv.ParseFloat(payload, 64); err == nil {
		value = fval
	}
	if rollup[field] {
		if _, exists := rollupState[source]; !exists {
			rollupState[source] = map[string]interface{}{}
		}
		rollupState[source][field] = value
		return nil
	}

	fields := pubsub.Fields{
		"source": source,
		field:    value,
	}
	if field == "power" || field == "start_measuring" {
		for key, value := range rollupState[source] {
			fields[key] = value
		}
	}
	ev := pubsub.NewEvent(topic, fields)
	services.Config.AddDeviceToEvent(ev)
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
