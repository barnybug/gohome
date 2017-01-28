package util

import "sync"

type Event struct {
	cond *sync.Cond
	set  bool
}

func NewEvent() *Event {
	return &Event{
		cond: &sync.Cond{L: &sync.Mutex{}},
	}
}

func (e *Event) Set() bool {
	e.cond.L.Lock()
	previous := e.set
	e.set = true
	e.cond.L.Unlock()
	e.cond.Signal()
	return previous
}

func (e *Event) Wait() {
	e.cond.L.Lock()
	for !e.set {
		e.cond.Wait()
	}
	e.cond.L.Unlock()
}
