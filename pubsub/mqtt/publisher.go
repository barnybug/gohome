package mqtt

import (
	"github.com/barnybug/gohome/pubsub"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

type Publisher struct {
	broker  string
	client  *MQTT.MqttClient
	channel chan *pubsub.Event
}

func (self *Publisher) Id() string {
	return "mqtt: " + self.broker
}

func (self *Publisher) Emit(ev *pubsub.Event) {
	// put all topics under gohome/
	topic := "gohome/" + ev.Topic
	r := self.client.Publish(MQTT.QOS_ONE, topic, ev.Bytes())
	<-r
}
