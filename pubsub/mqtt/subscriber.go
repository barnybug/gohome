package mqtt

import (
	"log"
	"sync"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/barnybug/gohome/pubsub"
)

type eventFilter func(*pubsub.Event) bool

type eventChannel struct {
	filter eventFilter
	C      chan *pubsub.Event
	topics []string
}

// Subscriber struct
type Subscriber struct {
	broker       *Broker
	channels     []eventChannel
	channelsLock sync.Mutex
	topicCount   map[string]int
}

func NewSubscriber(broker *Broker) pubsub.Subscriber {
	self := &Subscriber{broker: broker, topicCount: map[string]int{}}
	self.broker.opts.SetDefaultPublishHandler(self.publishHandler)
	return self
}

func (self *Subscriber) ID() string {
	return self.broker.Id()
}

func (self *Subscriber) publishHandler(client *MQTT.Client, msg MQTT.Message) {
	body := string(msg.Payload())
	event := pubsub.Parse(body)
	if event == nil {
		return
	}
	self.channelsLock.Lock()
	for _, ch := range self.channels {
		if ch.filter(event) {
			ch.C <- event
		}
	}
	self.channelsLock.Unlock()
}

func (self *Subscriber) addChannel(filter eventFilter, topics []string) eventChannel {
	// subscribe topics not yet subscribed to
	for _, topic := range topics {
		_, exists := self.topicCount[topic]
		if !exists {
			// fmt.Printf("StartSubscription: %+v\n", filter)
			// nil = all messages go to the default handler
			token := self.broker.client.Subscribe(topicName(topic), 1, nil)
			if token.Wait() && token.Error() != nil {
				log.Fatalln("Couldn't Subscribe to topic:", token.Error())
			}
		}
		self.topicCount[topic] += 1
	}

	ch := eventChannel{
		C:      make(chan *pubsub.Event, 16),
		filter: filter,
		topics: topics,
	}
	self.channelsLock.Lock()
	self.channels = append(self.channels, ch)
	self.channelsLock.Unlock()
	return ch
}

func (self *Subscriber) Channel() <-chan *pubsub.Event {
	ch := self.addChannel(func(ev *pubsub.Event) bool { return true }, []string{""})
	return ch.C
}

func stringSet(li []string) map[string]bool {
	ret := map[string]bool{}
	for _, i := range li {
		ret[i] = true
	}
	return ret
}

func (self *Subscriber) FilteredChannel(topics ...string) <-chan *pubsub.Event {
	topicSet := stringSet(topics)
	ch := self.addChannel(func(ev *pubsub.Event) bool { return topicSet[ev.Topic] }, topics)
	return ch.C
}

func topicName(topic string) string {
	if topic == "" {
		return "gohome/#"
	}
	return "gohome/" + topic + "/#"
}

func (self *Subscriber) Close(channel <-chan *pubsub.Event) {
	var channels []eventChannel
	for _, ch := range self.channels {
		if channel == chan *pubsub.Event(ch.C) {
			for _, topic := range ch.topics {
				self.topicCount[topic] -= 1
				if self.topicCount[topic] == 0 {
					// fmt.Printf("EndSubscription: %+v\n", topicName(topic))
					self.broker.client.Unsubscribe(topicName(topic))
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
