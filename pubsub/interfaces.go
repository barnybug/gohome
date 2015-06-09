package pubsub

type Publisher interface {
	Id() string
	Emit(ev *Event)
}

type Subscriber interface {
	Id() string
	FilteredChannel(...string) <-chan *Event
	Channel() <-chan *Event
	Close(<-chan *Event)
}
