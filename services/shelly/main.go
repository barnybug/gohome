// Service to translate shelly messages to/from gohome.
package shelly

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/barnybug/gohome/pubsub/mqtt"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Service shelly
type Service struct {
}

func (self *Service) ID() string {
	return "shelly"
}

type StatusPayload struct {
	Ison       bool `json:"ison"`
	Brightness int  `json:"brightness"`
}

func translate(message MQTT.Message) *pubsub.Event {
	// shellies/shellydimmer2-C8C9A3708E3A/light/0 on
	// shellies/shellydimmer2-C8C9A3708E3A/light/0/status {"ison":false,"source":"mqtt","has_timer":false,"timer_started":0,"timer_duration":0,"timer_remaining":0,"mode":"white","brightness":100,"transition":0}
	// various others (ignored currently)
	// shellies/shellydimmer2-C8C9A3708E3A/input/0 0
	// shellies/shellydimmer2-C8C9A3708E3A/input/1 0
	// shellies/shellydimmer2-C8C9A3708E3A/light/0/power 0.0
	ps := strings.Split(message.Topic(), "/")
	if len(ps) == 5 && ps[2] == "light" && ps[len(ps)-1] == "status" {
		// shellies/shellydimmer2-C8C9A3708E3A/light/0
		source := "shelly." + strings.TrimPrefix(ps[1], "shelly")
		// shelly.dimmer2-C8C9A3708E3A
		var status StatusPayload
		if err := json.Unmarshal(message.Payload(), &status); err != nil {
			log.Printf("Error unmarshalling json: %s %s", err, string(message.Payload()))
			return nil
		}

		command := "off"
		if status.Ison {
			command = "on"
		}
		topic := "ack"
		fields := pubsub.Fields{
			"source":  source,
			"command": command,
			"level":   status.Brightness,
		}
		ev := pubsub.NewEvent(topic, fields)
		services.Config.AddDeviceToEvent(ev)
		return ev
	}
	return nil
}

func handleCommand(ev *pubsub.Event) {
	source, ok := services.Config.LookupDeviceProtocol(ev.Device(), "shelly")
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

	// shellies/shellydimmer2-C8C9A3708E3A/light/0/set
	topic := fmt.Sprintf("shellies/%s/light/0/set", "shelly"+source)
	body := map[string]interface{}{
		"turn": command,
	}
	if level, ok := ev.Fields["level"]; ok {
		body["brightness"] = level
	}
	payload, _ := json.Marshal(body)
	token := mqtt.Client.Publish(topic, 1, false, string(payload))
	if token.Wait() && token.Error() != nil {
		log.Println("Failed to publish message:", token.Error())
	}
}

func (self *Service) Run() error {
	commandChannel := services.Subscriber.Subscribe(pubsub.Prefix("command"))
	messageChannel := make(chan MQTT.Message)
	mqtt.Client.Subscribe("shellies/#", 1, func(client MQTT.Client, message MQTT.Message) {
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
