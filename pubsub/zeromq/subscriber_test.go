package zeromq

import "testing"

func TestChannel(t *testing.T) {
	listener := NewSubscriber("inproc://1", "", false)
	ch := listener.Channel()
	listener.Close(ch)
}

func TestFilteredChannel(t *testing.T) {
	listener := NewSubscriber("inproc://2", "", false)
	ch := listener.Channel()
	listener.Close(ch)
}

func TestClose(t *testing.T) {
	listener := NewSubscriber("inproc://3", "", false)
	ch := listener.Channel()
	listener.Close(ch)
}

func TestCloseTwice(t *testing.T) {
	listener := NewSubscriber("inproc://4", "", false)
	ch := listener.Channel()
	listener.Close(ch)
	listener.Close(ch)
}
