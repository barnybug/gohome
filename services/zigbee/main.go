// Service to translate zigbee2mqtt messages to/from gohome.
package zigbee

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/barnybug/gohome/pubsub/mqtt"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Service zigbee
type Service struct {
}

func (self *Service) ID() string {
	return "zigbee"
}

var topicMap = map[string]string{
	"state":                  "ack",
	"temperature":            "temp",
	"local_temperature_heat": "temp",
	"action":                 "button",
}
var fieldMap = map[string]string{
	"temperature":            "temp",
	"local_temperature_heat": "temp",
}
var ignoreMap = map[string]bool{
	"update_available": true,
}

func getDevice(topic string) string {
	ps := strings.Split(topic, "/")
	return ps[len(ps)-1]
}

var deviceUpdate = regexp.MustCompile(`^zigbee2mqtt/[^/]+$`)
var buttonAction = regexp.MustCompile(`^(button_\d+)_(.+)$`)

var dedup = map[string]string{}

func checkDup(message MQTT.Message) bool {
	payload := string(message.Payload())
	if last, ok := dedup[message.Topic()]; ok && payload == last {
		return true
	}
	dedup[message.Topic()] = payload
	return false
}

type LogMessage struct {
	Message string `json:"message"`
	Meta    struct {
		Description  string `json:"description"`
		FriendlyName string `json:"friendly_name"`
		Model        string `json:"model"`
		Supported    bool   `json:"supported"`
		Vendor       string `json:"vendor"`
	} `json:"meta"`
}

func checkLogMessage(message MQTT.Message) {
	// announce
	var msg LogMessage
	err := json.Unmarshal(message.Payload(), &msg)
	if err != nil {
		log.Printf("Failed to parse message %s: '%s'", message.Topic(), message.Payload())
	}
	if msg.Message != "interview_successful" {
		return
	}
	// announce new devices
	source := fmt.Sprintf("zigbee.%s", msg.Meta.FriendlyName)
	supported := "unsupported"
	if msg.Meta.Supported {
		supported = ""
	}
	name := ""
	for _, s := range []string{msg.Meta.Vendor, msg.Meta.Model, msg.Meta.Description, supported} {
		if s == "" {
			continue
		}
		if name != "" {
			name += " "
		}
		name += s
	}
	log.Printf("Announcing %s: %s", source, name)
	fields := pubsub.Fields{"source": source, "name": name}
	ev := pubsub.NewEvent("announce", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
}

func translate(message MQTT.Message) *pubsub.Event {
	if message.Topic() == "zigbee2mqtt/bridge/log" {
		checkLogMessage(message)
		return nil
	}
	if strings.HasPrefix(message.Topic(), "zigbee2mqtt/bridge/") {
		// ignore other bridge messages
		return nil
	}
	if strings.HasSuffix(message.Topic(), "/set") {
		// ignore reflected set
		return nil
	}
	if strings.HasSuffix(message.Topic(), "/action") {
		// ignore action
		return nil
	}
	if !deviceUpdate.MatchString(message.Topic()) {
		log.Printf("Ignoring topic: %s", message.Topic())
		return nil
	}
	if checkDup(message) {
		return nil
	}

	var data map[string]interface{}
	err := json.Unmarshal(message.Payload(), &data)
	if err != nil {
		log.Printf("Failed to parse message %s: '%s'", message.Topic(), message.Payload())
		return nil
	}
	if data["action"] == "" {
		// ignore empty action following a button press
		return nil
	}

	device := getDevice(message.Topic())
	source := fmt.Sprintf("zigbee.%s", device)
	topic := "zigbee"
	fields := pubsub.Fields{}
	for key, value := range data {
		if topicValue, ok := topicMap[key]; ok {
			// use presence of keys to determine topic
			topic = topicValue
		} else if key == "brightness" {

		}
		// map fields
		if key == "state" {
			fields["command"] = strings.ToLower(value.(string))
		} else if key == "brightness" {
			fields["level"] = DimToPercentage(int(value.(float64)))
		} else if key == "action" {
			// multiple button device
			ps := buttonAction.FindStringSubmatch(value.(string))
			if len(ps) == 0 {
				log.Printf("Failed to parse action %v", value)
				continue
			}
			source += ":" + ps[1]
			if ps[2] == "single" {
				fields["command"] = "on"
			} else {
				fields["command"] = ps[2]
			}
		} else if to, ok := fieldMap[key]; ok {
			fields[to] = value
		} else if ignoreMap[key] {
			continue
		} else {
			fields[key] = value // map unknowns as is
		}
	}
	fields["source"] = source
	ev := pubsub.NewEvent(topic, fields)
	services.Config.AddDeviceToEvent(ev)
	return ev
}

func (self *Service) handleCommand(ev *pubsub.Event) {
	id, ok := services.Config.LookupDeviceProtocol(ev.Device(), "zigbee")
	if !ok {
		return // command not for us
	}
	device := services.Config.Devices[ev.Device()]
	command := ev.Command()
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
	log.Printf("Setting device %s to %s\n", ev.Device(), command)

	// translate to zigbee2mqtt message
	topic := fmt.Sprintf("zigbee2mqtt/%s/set", id)
	body := map[string]interface{}{}
	body["state"] = strings.ToUpper(command)
	if ev.IsSet("level") {
		body["brightness"] = PercentageToDim(int(ev.IntField("level")))
	}
	temp := ev.IntField("temp")
	if temp > 0 {
		if device.Cap["colourtemp"] {
			mirek := int(1_000_000 / temp)
			body["color_temp"] = mirek
		} else {
			// emulate colour temperature with x/y/dim
			x, y, dim := KelvinToColorXYDim(int(temp))
			body["color"] = map[string]interface{}{"x": x, "y": y}
			if !ev.IsSet("level") {
				body["brightness"] = dim
			}
		}
	}
	if ev.IsSet("colour") {
		body["color"] = map[string]interface{}{"hex": ev.StringField("colour")}
	}
	payload, _ := json.Marshal(body)
	log.Println("Sending", topic, string(payload))
	token := mqtt.Client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		log.Println("Failed to publish message:", token.Error())
	}
}

func (self *Service) handleThermostat(ev *pubsub.Event) {
	target, ok := ev.Fields["target"].(float64)
	if !ok {
		log.Println("Error: thermostat event target field invalid:", ev)
		return
	}

	id, ok := services.Config.LookupDeviceProtocol(ev.Device(), "zigbee")
	if !ok {
		return // command not for us
	}
	// hive https://www.zigbee2mqtt.io/devices/SLR2b.html
	// translate to zigbee2mqtt message
	topic := fmt.Sprintf("zigbee2mqtt/%s/set", id)
	body := map[string]interface{}{
		"system_mode_heat":               "heat",
		"temperature_setpoint_hold_heat": "1",
		"occupied_heating_setpoint_heat": fmt.Sprint(int(target)),
	}
	if boost, ok := ev.Fields["boost"].(float64); ok {
		// boost (aka party mode) - so they show up on device
		body["system_mode_heat"] = "emergency_heating"
		body["temperature_setpoint_hold_duration_heat"] = int(boost / 60) // minutes
	}
	payload, _ := json.Marshal(body)
	log.Println("Sending", topic, string(payload))
	token := mqtt.Client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		log.Println("Failed to publish message:", token.Error())
	}
}

func (self *Service) Run() error {
	mqtt.Client.Subscribe("zigbee2mqtt/#", 1, func(client MQTT.Client, msg MQTT.Message) {
		ev := translate(msg)
		if ev != nil {
			services.Publisher.Emit(ev)
		}
	})

	commandChannel := services.Subscriber.Subscribe(pubsub.Prefix("command"))
	thermostatChannel := services.Subscriber.Subscribe(pubsub.Prefix("thermostat"))
	for {
		select {
		case command := <-commandChannel:
			self.handleCommand(command)
		case ev := <-thermostatChannel:
			self.handleThermostat(ev)
		}
	}
	return nil
}
