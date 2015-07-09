package pubsub

import "sync"

type eventFilter func(*Event) bool

type eventChannel struct {
	filter eventFilter
	C      chan *Event
}

// filteredSubscriber is a subscriber filtered client-side.
type filteredSubscriber struct {
	id           string
	running      sync.Once
	channels     []eventChannel
	channelsLock sync.Mutex
}

// NewFilteredSubscriber creates a new filtered Subscriber
func NewFilteredSubscriber(id string, ch <-chan *Event) Subscriber {
	sub := &filteredSubscriber{id: id}
	go sub.run(ch)
	return sub
}

func (sub *filteredSubscriber) ID() string {
	return sub.id
}

func (sub *filteredSubscriber) run(ch <-chan *Event) {
	for event := range ch {
		sub.channelsLock.Lock()
		for _, ch := range sub.channels {
			if ch.filter(event) {
				ch.C <- event
			}
		}
		sub.channelsLock.Unlock()
	}
}

func (sub *filteredSubscriber) addChannel(filter eventFilter) eventChannel {
	ch := eventChannel{
		C:      make(chan *Event, 16),
		filter: filter,
	}
	sub.channelsLock.Lock()
	sub.channels = append(sub.channels, ch)
	sub.channelsLock.Unlock()
	return ch
}

func (sub *filteredSubscriber) Channel() <-chan *Event {
	ch := sub.addChannel(func(ev *Event) bool { return true })
	return ch.C
}

func stringSet(li []string) map[string]bool {
	ret := map[string]bool{}
	for _, i := range li {
		ret[i] = true
	}
	return ret
}

func (sub *filteredSubscriber) FilteredChannel(topics ...string) <-chan *Event {
	topicSet := stringSet(topics)
	ch := sub.addChannel(func(ev *Event) bool { return topicSet[ev.Topic] })
	return ch.C
}

func (sub *filteredSubscriber) Close(channel <-chan *Event) {
	var channels []eventChannel
	for _, ch := range sub.channels {
		if channel == chan *Event(ch.C) {
			close(ch.C)
		} else {
			channels = append(channels, ch)
		}
	}
	sub.channelsLock.Lock()
	sub.channels = channels
	sub.channelsLock.Unlock()
}
