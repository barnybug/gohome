package dummy

import "github.com/barnybug/gohome/pubsub"

// Dummy Subscriber for testing
type Subscriber struct {
	Events []*pubsub.Event
}

func (self *Subscriber) Id() string {
	return "dummy"
}

func (self *Subscriber) replayEvents() <-chan *pubsub.Event {
	ch := make(chan *pubsub.Event)
	go func() {
		for _, ev := range self.Events {
			ch <- ev
		}
		close(ch)
	}()
	return ch
}

func (self *Subscriber) FilteredChannel(...string) <-chan *pubsub.Event {
	return self.replayEvents()
}

func (self *Subscriber) Channel() <-chan *pubsub.Event {
	return self.replayEvents()
}

func (self *Subscriber) Close(<-chan *pubsub.Event) {
}
