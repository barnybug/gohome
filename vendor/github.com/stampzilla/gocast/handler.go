package gocast

import (
	"github.com/stampzilla/gocast/events"
	"github.com/stampzilla/gocast/handlers"
)

type Handler interface {
	RegisterSend(func(handlers.Headers) error)
	RegisterDispatch(func(events.Event))

	Connect()
	Disconnect()
	Unmarshal(string)
}
