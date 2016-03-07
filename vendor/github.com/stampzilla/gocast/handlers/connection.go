package handlers

import (
	"fmt"

	"github.com/stampzilla/gocast/events"
)

type Connection struct {
	Dispatch func(events.Event)
	Send     func(Headers) error
}

func (c *Connection) RegisterDispatch(dispatch func(events.Event)) {
	c.Dispatch = dispatch
}
func (c *Connection) RegisterSend(send func(Headers) error) {
	c.Send = send
}

func (c *Connection) Connect() {
	c.Send(Headers{Type: "CONNECT"})
}

func (c *Connection) Disconnect() {
	c.Send(Headers{Type: "CLOSE"})
}

func (c *Connection) Unmarshal(message string) {
	fmt.Println("Connection received: ", message)
}
