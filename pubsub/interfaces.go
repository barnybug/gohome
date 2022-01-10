package pubsub

// Publisher interface
type Publisher interface {
	ID() string
	Emit(ev *Event)
	Close()
}

type Topic interface {
	Match(topic string) bool
}

// Subscriber interface
type Subscriber interface {
	ID() string
	Subscribe(...Topic) <-chan *Event
	Close(<-chan *Event)
}
