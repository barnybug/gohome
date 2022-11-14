// Service to translate zwave messages to/from gohome.
package zwave

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub/mqtt"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Service zwave
type Service struct {
	timers map[string]*time.Timer
}

func (self *Service) ID() string {
	return "zwave"
}

type Field struct {
	Topic string
	Field string
	Parse func(s string) interface{}
}

func parseAirTemperature(s string) interface{} {
	if i, err := strconv.Atoi(s); err == nil {
		celsius := (float64(i) - 32) * 5 / 9
		celsius = math.Round(celsius*10) / 10
		return celsius
	}
	return nil
}

func parseBool(s string) interface{} {
	switch s {
	case "true":
		return "on"
	case "false":
		return "off"
	default:
		return nil
	}
}

func parseAccessControl(s string) interface{} {
	switch s {
	case "22": // open
		return "on"
	case "23": // closed
		return "off"
	default:
		return nil
	}
}

func parseIlluminance(s string) interface{} {
	if f, err := strconv.ParseFloat(s, 64); err != nil {
		return f
	}
	return nil
}

func parseMotionSensorStatus(s string) interface{} {
	switch s {
	case "8": // Motion
		return "on"
	default:
		return nil
	}
}

var mapping = map[string]Field{
	"Air_temperature": {
		Topic: "temp",
		Field: "temp",
		Parse: parseAirTemperature,
	},
	"Door-Window": {
		Topic: "sensor",
		Field: "command",
		Parse: parseBool,
	},
	"Door_state": {
		Topic: "sensor",
		Field: "command",
		Parse: parseAccessControl,
	},
	"Illuminance": {
		Topic: "lux",
		Field: "lux",
		Parse: parseIlluminance,
	},
	"Motion_sensor_status": {
		Topic: "sensor",
		Field: "command",
		Parse: parseMotionSensorStatus,
	},
}

func (self *Service) translate(message MQTT.Message) *pubsub.Event {
	ps := strings.Split(message.Topic(), "/")
	if ps[1] == "_CLIENTS" || ps[1] == "driver" {
		return nil
	}
	last := ps[len(ps)-1]
	source := fmt.Sprintf("zwave.%s:%s", ps[1], last)
	field, ok := mapping[last]
	if !ok {
		return nil
	}
	payload := string(message.Payload())
	value := field.Parse(payload)
	if value == nil {
		return nil
	}
	fields := pubsub.Fields{}
	fields["source"] = source
	fields[field.Field] = value
	ev := pubsub.NewEvent(field.Topic, fields)
	services.Config.AddDeviceToEvent(ev)

	// special case for pir motion
	if last == "Motion_sensor_status" && value == "on" && ev.Device() != "" {
		// sensors do not send off, so trigger this on a timer delay
		if timer, present := self.timers[ev.Device()]; present {
			timer.Reset(time.Minute)
		} else {
			timer := time.AfterFunc(time.Minute, func() {
				ev := pubsub.NewEvent(field.Topic, fields)
				ev.Fields["command"] = "off"
				services.Config.AddDeviceToEvent(ev)
				services.Publisher.Emit(ev)
			})
			self.timers[ev.Device()] = timer
		}
	}

	return ev
}

func (self *Service) Run() error {
	self.timers = map[string]*time.Timer{}
	mqtt.Client.Subscribe("zwave/#", 1, func(client MQTT.Client, msg MQTT.Message) {
		ev := self.translate(msg)
		if ev != nil {
			services.Publisher.Emit(ev)
		}
	})

	select {}
}
