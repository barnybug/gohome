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
	client *MQTT.Client
	opts   *MQTT.ClientOptions
}

func createClient(broker string) (*MQTT.Client, *MQTT.ClientOptions) {
	// generate a client id
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	r := rand.Int()
	clientId := fmt.Sprintf("gohome/%s-%d-%d", hostname, pid, r)
	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientId)
	opts.SetCleanSession(true)

	client := MQTT.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalln("Couldn't Start mqtt:", token.Error())
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
