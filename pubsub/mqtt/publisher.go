package mqtt

import (
	"log"

	"github.com/barnybug/gohome/pubsub"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Publisher for mqtt
type Publisher struct {
	broker  string
	client  MQTT.Client
	channel chan *pubsub.Event
}

// ID of Publisher
func (pub *Publisher) ID() string {
	return "mqtt: " + pub.broker
}

func (pub *Publisher) loop() {
	for ev := range pub.channel {
		// put all topics under gohome/<topic>/<device>
		topic := "gohome/" + ev.Topic
		if ev.Device() != "" {
			topic += "/" + ev.Device()
		}
		// log.Println("Publishing:", topic, string(ev.Bytes()))
		token := pub.client.Publish(topic, 1, ev.Retained, ev.Bytes())
		if token.Wait() && token.Error() != nil {
			log.Println("Failed to publish message:", token.Error())
		}
		// log.Println("Published", topic)
	}
}

// Emit an event
func (pub *Publisher) Emit(ev *pubsub.Event) {
	// Publishing inside a Subscribe callback can deadlock the MQTT client
	// so we hand off to a channel.
	select {
	case pub.channel <- ev:
		return
	default:
		log.Println("Publish channel FULL - dropping message!")
	}
}
