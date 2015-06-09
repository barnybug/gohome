package pubsub

import "sync"

type eventFilter func(*Event) bool

type eventChannel struct {
	filter eventFilter
	C      chan *Event
}

// A subscriber filtered client-side.
type FilteredSubscriber struct {
	id           string
	running      sync.Once
	channels     []eventChannel
	channelsLock sync.Mutex
}

func NewFilteredSubscriber(id string, ch <-chan *Event) Subscriber {
	self := &FilteredSubscriber{id: id}
	go self.run(ch)
	return self
}

func (self *FilteredSubscriber) Id() string {
	return self.id
}

func (self *FilteredSubscriber) run(ch <-chan *Event) {
	for event := range ch {
		self.channelsLock.Lock()
		for _, ch := range self.channels {
			if ch.filter(event) {
				ch.C <- event
			}
		}
		self.channelsLock.Unlock()
	}
}

func (self *FilteredSubscriber) addChannel(filter eventFilter) eventChannel {
	ch := eventChannel{
		C:      make(chan *Event, 16),
		filter: filter,
	}
	self.channelsLock.Lock()
	self.channels = append(self.channels, ch)
	self.channelsLock.Unlock()
	return ch
}

func (self *FilteredSubscriber) Channel() <-chan *Event {
	ch := self.addChannel(func(ev *Event) bool { return true })
	return ch.C
}

func stringSet(li []string) map[string]bool {
	ret := map[string]bool{}
	for _, i := range li {
		ret[i] = true
	}
	return ret
}

func (self *FilteredSubscriber) FilteredChannel(topics ...string) <-chan *Event {
	topicSet := stringSet(topics)
	ch := self.addChannel(func(ev *Event) bool { return topicSet[ev.Topic] })
	return ch.C
}

func (self *FilteredSubscriber) Close(channel <-chan *Event) {
	var channels []eventChannel
	for _, ch := range self.channels {
		if channel == chan *Event(ch.C) {
			close(ch.C)
		} else {
			channels = append(channels, ch)
		}
	}
	self.channelsLock.Lock()
	self.channels = channels
	self.channelsLock.Unlock()
}
