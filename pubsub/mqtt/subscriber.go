package mqtt

import (
	"github.com/barnybug/gohome/pubsub"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

func run(client *MQTT.MqttClient, broker string) chan *pubsub.Event {
	ch := make(chan *pubsub.Event, 16)

	// listen to all topics under 'gohome'
	filter, _ := MQTT.NewTopicFilter("gohome/#", 1)
	client.StartSubscription(func(client *MQTT.MqttClient, msg MQTT.Message) {
		body := string(msg.Payload())
		event := pubsub.Parse(body)
		if event != nil {
			ch <- event
		}
	}, filter)

	return ch
}
