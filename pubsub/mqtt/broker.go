package mqtt

import (
	"fmt"
	"log"
	"math/rand"
	"os"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/barnybug/gohome/pubsub"
)

type Broker struct {
	broker string
	client *MQTT.MqttClient
	opts   *MQTT.ClientOptions
}

func createClient(broker string) (*MQTT.MqttClient, *MQTT.ClientOptions) {
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
	return client, opts
}

func NewBroker(broker string) *Broker {
	client, opts := createClient(broker)
	return &Broker{broker, client, opts}
}

func (self *Broker) Id() string {
	return "mqtt: " + self.broker
}

func (self *Broker) Subscriber() pubsub.Subscriber {
	return NewSubscriber(self)
}

func (self *Broker) Publisher() *Publisher {
	ch := make(chan *pubsub.Event)
	return &Publisher{broker: self.broker, channel: ch, client: self.client}
}
