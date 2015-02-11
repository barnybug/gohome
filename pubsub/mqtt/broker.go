package mqtt

import (
	"fmt"
	"github.com/barnybug/gohome/pubsub"
	"log"
	"math/rand"
	"os"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

type Broker struct {
	broker string
	client *MQTT.MqttClient
}

func createClient(broker string) *MQTT.MqttClient {
	// generate a client id
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	r := rand.Int()
	clientId := fmt.Sprintf("gohome/%s-%d-%d", hostname, pid, r)
	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientId(clientId)
	opts.SetCleanSession(true)

	client := MQTT.NewClient(opts)
	_, err := client.Start()
	if err != nil {
		log.Fatalln("Couldn't Start mqtt:", err)
	}
	return client
}

func NewBroker(broker string) *Broker {
	client := createClient(broker)
	return &Broker{broker, client}
}

func (self *Broker) Subscriber() pubsub.Subscriber {
	// A bug in the MQTT libraries prevents us from using multiple subscriptions
	// to handle listening to multiple topics, so instead listen to all under
	// gohome, and filter client-side.
	ch := run(self.client, self.broker)
	return pubsub.NewFilteredSubscriber("mqtt: "+self.broker, ch)
}

func (self *Broker) Publisher() *Publisher {
	ch := make(chan *pubsub.Event)
	return &Publisher{broker: self.broker, channel: ch, client: self.client}
}
