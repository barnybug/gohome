package mqtt

import (
	"log"
	"sync"

	"github.com/barnybug/gohome/pubsub"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type eventFilter func(*pubsub.Event) bool

type eventChannel struct {
	C      chan *pubsub.Event
	topics []pubsub.Topic
}

// Subscriber struct
type Subscriber struct {
	broker         *Broker
	channels       []eventChannel
	channelsLock   sync.Mutex
	topicCount     map[string]int
	topicCountLock sync.RWMutex
	persist        bool
}

func NewSubscriber(broker *Broker, persist bool) *Subscriber {
	return &Subscriber{broker: broker, topicCount: map[string]int{}, persist: persist}
}

func (self *Subscriber) ID() string {
	return self.broker.Id()
}

func (self *Subscriber) publishHandler(client MQTT.Client, msg MQTT.Message) {
	topic := msg.Topic()[7:] // skip "gohome/"
	body := string(msg.Payload())
	event := pubsub.Parse(body, topic)
	if event == nil {
		return
	}
	event.SetRetained(msg.Retained())
	self.channelsLock.Lock()
	// fmt.Printf("Event: %+v\n", event)
	for _, ch := range self.channels {
		for _, t := range ch.topics {
			if t.Match(topic) {
				// fmt.Printf("Sending to: %+v\n", ch.topics)
				ch.C <- event
				break
			}
		}
	}
	self.channelsLock.Unlock()
}

func (self *Subscriber) connectHandler(client MQTT.Client) {
	if self.persist {
		return // unnecessary on persistent connections
	}
	// (re)subscribe when (re)connected
	subs := map[string]byte{}
	self.topicCountLock.RLock()
	for topic, _ := range self.topicCount {
		subs[topic] = 1 // QOS
	}
	self.topicCountLock.RUnlock()

	if len(subs) > 0 {
		// nil = all messages go to the default handler
		log.Println("Connected, subscribing:", subs)
		if token := self.broker.client.SubscribeMultiple(subs, nil); token.Wait() && token.Error() != nil {
			log.Println("Error subscribing:", token.Error())
		}
	}
}

func topicToMqtt(topic pubsub.Topic) string {
	switch topic := topic.(type) {
	case *pubsub.AllTopic:
		return "gohome/#"
	case *pubsub.ExactTopic:
		return "gohome/" + topic.Exact
	case *pubsub.PrefixTopic:
		return "gohome/" + topic.Prefix + "/#"
	default:
		log.Panicln("Topic type unsupported")
	}
	return ""
}

func topicsToMqtt(topics []pubsub.Topic) []string {
	var ret []string
	for _, topic := range topics {
		ret = append(ret, topicToMqtt(topic))
	}
	return ret
}

func (self *Subscriber) addChannel(topics []pubsub.Topic) eventChannel {
	// subscribe topics not yet subscribed to
	subs := map[string]byte{}
	mqttTopics := topicsToMqtt(topics)
	self.topicCountLock.Lock()
	for _, topic := range mqttTopics {
		_, exists := self.topicCount[topic]
		if !exists {
			// log.Println("Subscribe", topic)
			subs[topic] = 1 // QOS
		}
		self.topicCount[topic] += 1
	}
	self.topicCountLock.Unlock()

	ch := eventChannel{
		C:      make(chan *pubsub.Event, 16),
		topics: topics,
	}
	self.channelsLock.Lock()
	self.channels = append(self.channels, ch)
	self.channelsLock.Unlock()

	if len(subs) > 0 {
		// nil = all messages go to the default handler
		if token := self.broker.client.SubscribeMultiple(subs, nil); token.Wait() && token.Error() != nil {
			log.Println("Error subscribing:", token.Error())
		}
	}

	return ch
}

func (self *Subscriber) Subscribe(topics ...pubsub.Topic) <-chan *pubsub.Event {
	ch := self.addChannel(topics)
	return ch.C
}

func (self *Subscriber) Close(channel <-chan *pubsub.Event) {
	var channels []eventChannel
	for _, ch := range self.channels {
		if channel == chan *pubsub.Event(ch.C) {
			for _, topic := range ch.topics {
				t := topicToMqtt(topic)
				self.topicCountLock.Lock()
				self.topicCount[t] -= 1
				current := self.topicCount[t]
				self.topicCountLock.Unlock()
				if current == 0 {
					// fmt.Printf("Unsubscribe: %+v\n", topicName(topic))
					if token := self.broker.client.Unsubscribe(t); token.Wait() && token.Error() != nil {
						log.Println("Error unsubscribing:", token.Error())
					}
				}
			}
			close(ch.C)
		} else {
			channels = append(channels, ch)
		}
	}
	self.channelsLock.Lock()
	self.channels = channels
	self.channelsLock.Unlock()
}
