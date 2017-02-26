package mqtt

import (
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/barnybug/gohome/pubsub"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Broker struct {
	broker     string
	subscriber *Subscriber
	client     MQTT.Client
}

func createClientOpts(broker string) *MQTT.ClientOptions {
	// generate a client id
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	r := rand.Int()
	clientId := fmt.Sprintf("gohome/%s-%d-%d", hostname, pid, r)
	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientId)
	opts.SetCleanSession(true)
	return opts
}

func NewBroker(broker string) *Broker {
	opts := createClientOpts(broker)
	ret := &Broker{broker: broker}
	ret.subscriber = NewSubscriber(ret)
	opts.SetDefaultPublishHandler(ret.subscriber.publishHandler)

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalln("Couldn't Start mqtt:", token.Error())
	}
	ret.client = client
	return ret
}

func (self *Broker) Id() string {
	return "mqtt: " + self.broker
}

func (self *Broker) Subscriber() pubsub.Subscriber {
	return self.subscriber
}

func (self *Broker) Publisher() *Publisher {
	ch := make(chan *pubsub.Event)
	return &Publisher{broker: self.broker, channel: ch, client: self.client}
}
