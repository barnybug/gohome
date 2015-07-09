package dummy

import "github.com/barnybug/gohome/pubsub"

// Subscriber for testing
type Subscriber struct {
	Events []*pubsub.Event
}

// ID of Subscriber
func (sub *Subscriber) ID() string {
	return "dummy"
}

func (sub *Subscriber) replayEvents() <-chan *pubsub.Event {
	ch := make(chan *pubsub.Event)
	go func() {
		for _, ev := range sub.Events {
			ch <- ev
		}
		close(ch)
	}()
	return ch
}

// FilteredChannel by topic
func (sub *Subscriber) FilteredChannel(...string) <-chan *pubsub.Event {
	return sub.replayEvents()
}

// Channel with all events
func (sub *Subscriber) Channel() <-chan *pubsub.Event {
	return sub.replayEvents()
}

// Close the channel
func (sub *Subscriber) Close(<-chan *pubsub.Event) {
}
