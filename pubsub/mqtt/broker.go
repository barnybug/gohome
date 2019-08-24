package mqtt

import (
	"fmt"
	"log"
	"os"

	"github.com/barnybug/gohome/pubsub"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Broker struct {
	broker     string
	subscriber *Subscriber
	client     MQTT.Client
}

var Client MQTT.Client

func createClientOpts(broker, name string, persist bool) *MQTT.ClientOptions {
	// generate a client id
	hostname, _ := os.Hostname()
	clientID := fmt.Sprintf("gohome/%s-%s", hostname, name)
	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	if persist {
		opts.SetCleanSession(true)
	} else {
		// ensure subscriptions survive across disconnections
		opts.SetCleanSession(false)
	}
	return opts
}

func NewBroker(broker, name string) *Broker {
	persist := false
	opts := createClientOpts(broker, name, persist)
	ret := &Broker{broker: broker}
	ret.subscriber = NewSubscriber(ret, persist)
	opts.SetDefaultPublishHandler(ret.subscriber.publishHandler)
	opts.SetOnConnectHandler(ret.subscriber.connectHandler)

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalln("Couldn't Start mqtt:", token.Error())
	}
	Client = client
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
