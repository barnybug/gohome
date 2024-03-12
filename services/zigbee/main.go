// Service to translate zigbee2mqtt messages to/from gohome.
package zigbee

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

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

type Field struct {
	Topic string
	Field string
	Parse func(value interface{}) interface{}
}

func parseHeating(value interface{}) interface{} {
	switch value {
	case "idle":
		return false
	case "heat":
		return true
	default:
		return nil
	}
}

func parseBrightness(value interface{}) interface{} {
	if f, ok := value.(float64); ok {
		return DimToPercentage(int(f))
	}
	return nil
}

func parseContact(value interface{}) interface{} {
	// sense is inverted: contact true means switch closed, ie sensor off
	if value == true {
		return "off"
	} else if value == false {
		return "on"
	}
	return nil
}

var mapping = map[string]Field{
	"state": {
		Topic: "ack",
		Field: "state",
	},
	"temperature": {
		Topic: "temp",
		Field: "temp",
	},
	"local_temperature_heat": {
		Topic: "temp",
		Field: "temp",
	},
	"action": {
		Topic: "button",
		Field: "action",
	},
	"running_state_heat": {
		Topic: "temp",
		Field: "heating",
		Parse: parseHeating,
	},
	"brightness": {
		Topic: "ack",
		Field: "level",
		Parse: parseBrightness,
	},
	"contact": {
		Topic: "sensor",
		Field: "command",
		Parse: parseContact,
	},
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
var dedupOld = map[string]string{}

func checkDup(message MQTT.Message) bool {
	payload := string(message.Payload())
	if last, ok := dedup[message.Topic()]; ok {
		if payload == last {
			return true
		}
	} else if last, ok := dedupOld[message.Topic()]; ok {
		if payload == last {
			return true
		}
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
	if strings.HasPrefix(message.Topic(), "zigbee2mqtt/bridge/") || strings.HasPrefix(message.Topic(), "zigbee2mqtt/901/") {
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
	if action, ok := data["action"]; ok && (action == "" || action == nil) {
		// ignore empty action following a button press
		return nil
	}

	device := getDevice(message.Topic())
	source := fmt.Sprintf("zigbee.%s", device)
	topic := "zigbee"
	fields := pubsub.Fields{}
	for key, value := range data {
		// map fields
		if ignoreMap[key] {
			continue
		} else if key == "state" {
			fields["command"] = strings.ToLower(value.(string))
		} else if key == "action" {
			if value == "off" || value == "on" {
				fields["command"] = value
			} else {
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
			}
		} else if field, ok := mapping[key]; ok {
			topic = field.Topic
			if field.Parse != nil {
				value = field.Parse(value)
			}
			if value != nil {
				fields[field.Field] = value
			}
		} else {
			if value != nil {
				fields[key] = value // map unknowns as is
			}
		}
	}
	fields["source"] = source
	ev := pubsub.NewEvent(topic, fields)
	services.Config.AddDeviceToEvent(ev)
	return ev
}

func splitEndpoint(s string) (string, string) {
	ns := strings.SplitN(s, ":", 2)
	if len(ns) == 2 {
		return ns[0], ns[1]
	}
	return ns[0], ""
}

func (self *Service) handleCommand(ev *pubsub.Event) {
	id, ok := services.Config.LookupDeviceProtocol(ev.Device(), "zigbee")
	if !ok {
		return // command not for us
	}
	device := services.Config.Devices[ev.Device()]

	// translate to zigbee2mqtt message
	zid, ep := splitEndpoint(id)
	topic := fmt.Sprintf("zigbee2mqtt/%s/set", zid)
	var body map[string]interface{}
	if device.Cap["switch"] {
		command := ev.Command()
		if command != "off" && command != "on" {
			log.Println("switch: command not recognised:", command)
			return
		}
		body = map[string]interface{}{
			"state": strings.ToUpper(command),
		}
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
	} else if device.Cap["thermostat"] {
		// hive https://www.zigbee2mqtt.io/devices/SLR2b.html
		if ep == "" {
			ep = "heat"
		}
		// WATER endpoint doesn't seem to accept 10 anymore - contrlled purely by system_mode_water off/on ??
		if ep == "water" {
			log.Println("ignoring water temporarily")
			return
		}
		target, ok := ev.Fields["target"].(float64)
		if !ok {
			log.Println("Error: thermostat event target field invalid:", ev)
			return
		}
		mode := "heat"
		if target <= 10 {
			mode = "off"
		}
		body = map[string]interface{}{
			"system_mode_" + ep:               mode,
			"temperature_setpoint_hold_" + ep: "1",
			"occupied_heating_setpoint_" + ep: target,
		}
		if boost, ok := ev.Fields["boost"].(float64); ok {
			// boost (aka party mode) - so they show up on device
			body["system_mode_"+ep] = "emergency_heating"
			body["temperature_setpoint_hold_duration_"+ep] = int(boost / 60) // minutes
		}
	} else {
		log.Println("command to unrecognised device:", device)
		return
	}
	payload, _ := json.Marshal(body)
	// log.Println("Sending", topic, string(payload))
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
	rolloverDedup := time.NewTicker(30 * time.Second)
	for {
		select {
		case command := <-commandChannel:
			self.handleCommand(command)
		case <-rolloverDedup.C:
			dedupOld = dedup
			dedup = map[string]string{}
		}
	}
	return nil
}
