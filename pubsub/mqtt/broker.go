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
	broker     string
	client     *MQTT.Client
	opts       *MQTT.ClientOptions
	subscriber pubsub.Subscriber
}

func createOpts(broker string) *MQTT.ClientOptions {
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

func createClient(opts *MQTT.ClientOptions) *MQTT.Client {
	client := MQTT.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalln("Couldn't Start mqtt:", token.Error())
	}
	return client
}

func NewBroker(broker string) *Broker {
	opts := createOpts(broker)
	ret := &Broker{broker: broker, opts: opts}
	// The default publish handler must be set on opts before creating the client now,
	// so create Subscriber first, then client.
	ret.subscriber = NewSubscriber(ret)
	ret.client = createClient(opts)
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
