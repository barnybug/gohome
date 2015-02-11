// Service to publish and subscribe to events throughout the system. This is the
// central hub that every other service uses as a message bus.
//
// The messages themselves are broadcast on a given topic, which can be
// subscribed to by any number of other interested services.
package pubsub

import (
	"fmt"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/mqtt"
	"github.com/barnybug/gohome/pubsub/nanomsg"
	"github.com/barnybug/gohome/pubsub/zeromq"
	"github.com/barnybug/gohome/services"
	"log"
	"sync"
	"sync/atomic"
)

type PubsubService struct {
	Processed uint64
}

func (self *PubsubService) Id() string {
	return "pubsub"
}

func (self *PubsubService) Run() error {
	ep := services.Config.Endpoints

	routing := map[pubsub.Subscriber][]pubsub.Publisher{}

	// setup subscribers
	var subscribers []pubsub.Subscriber
	var publishers []pubsub.Publisher
	if ep.Zeromq.Pub != "" {
		sub := zeromq.NewSubscriber(ep.Zeromq.Pub, "", false)
		subscribers = append(subscribers, sub)
	}
	if ep.Nanomsg.Pub != "" {
		sub := nanomsg.NewSubscriber(ep.Nanomsg.Pub, "", false)
		subscribers = append(subscribers, sub)
	}
	if ep.Mqtt.Broker != "" {
		broker := mqtt.NewBroker(ep.Mqtt.Broker)
		pub := broker.Publisher()
		publishers = append(publishers, pub)
		for _, sub := range subscribers {
			routing[sub] = append(routing[sub], pub)
		}
		// do not add the mqtt publisher to the mqtt subscriber, otherwise
		// this creates a loop. The mqtt broker itself relays messages.
		sub := broker.Subscriber()
		subscribers = append(subscribers, sub)
	}

	// setup publishers
	if ep.Zeromq.Sub != "" {
		pub := zeromq.NewPublisher(ep.Zeromq.Sub, false)
		publishers = append(publishers, pub)
		for _, sub := range subscribers {
			routing[sub] = append(routing[sub], pub)
		}
	}
	if ep.Nanomsg.Sub != "" {
		pub := nanomsg.NewPublisher(ep.Nanomsg.Sub, false)
		publishers = append(publishers, pub)
		for _, sub := range subscribers {
			routing[sub] = append(routing[sub], pub)
		}
	}

	log.Println("Subscriber endpoints:")
	for _, sub := range subscribers {
		log.Println("-", sub.Id())
	}
	log.Println("Publisher endpoints:")
	for _, pub := range publishers {
		log.Println("-", pub.Id())
	}

	// Copy from subscribers to publishers
	var wg sync.WaitGroup
	wg.Add(len(subscribers))
	for _, sub := range subscribers {
		// for each subscriber
		go func(sub pubsub.Subscriber) {
			pubs := routing[sub]
			// wait for an event
			for ev := range sub.Channel() {
				// send the event onto publishers
				for _, pub := range pubs {
					pub.Emit(ev)
				}
				atomic.AddUint64(&self.Processed, 1)
			}
			wg.Done()
		}(sub)
	}

	wg.Wait()

	return nil
}

func (self *PubsubService) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": services.TextHandler(self.queryStatus),
		"help":   services.StaticHandler("status: get status\n"),
	}
}

func (self *PubsubService) queryStatus(q services.Question) string {
	return fmt.Sprintf("processed: %d", self.Processed)
}
