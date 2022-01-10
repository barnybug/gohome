package dummy

import "github.com/barnybug/gohome/pubsub"

// Subscriber for testing
type Subscriber struct {
	subscriptions []pubsub.Topic
	Events        []*pubsub.Event
}

// ID of Subscriber
func (sub *Subscriber) ID() string {
	return "dummy"
}

func (sub *Subscriber) replayEvents() <-chan *pubsub.Event {
	ch := make(chan *pubsub.Event)
	go func() {
		for _, ev := range sub.Events {
			for _, s := range sub.subscriptions {
				if s.Match(ev.Topic) {
					ch <- ev
					break
				}
			}
		}
		close(ch)
	}()
	return ch
}

func (sub *Subscriber) Subscribe(topics ...pubsub.Topic) <-chan *pubsub.Event {
	sub.subscriptions = append(sub.subscriptions, topics...)
	return sub.replayEvents()
}

// Close the channel
func (sub *Subscriber) Close(<-chan *pubsub.Event) {
}
