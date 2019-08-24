package mqtt

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/barnybug/gohome/pubsub"
	MQTT "github.com/eclipse/paho.mqtt.golang"
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
	persist      bool
}

func NewSubscriber(broker *Broker, persist bool) *Subscriber {
	return &Subscriber{broker: broker, topicCount: map[string]int{}, persist: persist}
}

func (self *Subscriber) ID() string {
	return self.broker.Id()
}

func (self *Subscriber) publishHandler(client MQTT.Client, msg MQTT.Message) {
	body := string(msg.Payload())
	event := pubsub.Parse(body)
	if event == nil {
		return
	}
	event.SetRetained(msg.Retained())
	self.channelsLock.Lock()
	// fmt.Printf("Event: %+v\n", event)
	for _, ch := range self.channels {
		if ch.filter(event) {
			// fmt.Printf("Sending to: %+v\n", ch.topics)
			ch.C <- event
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
	for topic, _ := range self.topicCount {
		subs[topicName(topic)] = 1 // QOS
	}

	if len(subs) > 0 {
		// nil = all messages go to the default handler
		log.Println("Connected, subscribing:", subs)
		if token := self.broker.client.SubscribeMultiple(subs, nil); token.Wait() && token.Error() != nil {
			log.Println("Error subscribing:", token.Error())
		}
	}
}

func (self *Subscriber) addChannel(filter eventFilter, topics []string) eventChannel {
	// subscribe topics not yet subscribed to
	subs := map[string]byte{}
	for _, topic := range topics {
		_, exists := self.topicCount[topic]
		if !exists {
			// log.Println("Subscribe", topicName(topic))
			subs[topicName(topic)] = 1 // QOS
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

	if len(subs) > 0 {
		// nil = all messages go to the default handler
		if token := self.broker.client.SubscribeMultiple(subs, nil); token.Wait() && token.Error() != nil {
			log.Println("Error subscribing:", token.Error())
		}
	}

	return ch
}

func (self *Subscriber) Channel() <-chan *pubsub.Event {
	ch := self.addChannel(func(ev *pubsub.Event) bool { return true }, []string{""})
	return ch.C
}

func topicRegex(topics []string) *regexp.Regexp {
	expr := fmt.Sprintf("^(%s)($|/)", strings.Join(topics, "|"))
	return regexp.MustCompile(expr)
}

func eventMatcher(topics []string) eventFilter {
	re := topicRegex(topics)
	return func(ev *pubsub.Event) bool {
		return re.MatchString(ev.Topic)
	}
}

func (self *Subscriber) FilteredChannel(topics ...string) <-chan *pubsub.Event {
	ch := self.addChannel(eventMatcher(topics), topics)
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
					// fmt.Printf("Unsubscribe: %+v\n", topicName(topic))
					if token := self.broker.client.Unsubscribe(topicName(topic)); token.Wait() && token.Error() != nil {
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
