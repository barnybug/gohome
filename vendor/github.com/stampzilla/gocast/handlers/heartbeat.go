package handlers

import (
	"time"

	"github.com/stampzilla/gocast/events"
)

type Heartbeat struct {
	Dispatch func(events.Event)
	Send     func(Headers) error

	ticker   *time.Ticker
	shutdown chan struct{}
}

func (h *Heartbeat) RegisterDispatch(dispatch func(events.Event)) {
	h.Dispatch = dispatch
}
func (h *Heartbeat) RegisterSend(send func(Headers) error) {
	h.Send = send
}

func (h *Heartbeat) Connect() {
	if h.ticker != nil {
		h.ticker.Stop()
		if h.shutdown != nil {
			close(h.shutdown)
			h.shutdown = nil
		}
	}

	h.ticker = time.NewTicker(time.Second * 5)
	h.shutdown = make(chan struct{})
	go func() {
		for {
			select {
			case <-h.ticker.C:
				h.Ping()
			case <-h.shutdown:
				return
			}
		}
	}()

}

func (h *Heartbeat) Disconnect() {
	if h.ticker != nil {
		h.ticker.Stop()
		if h.shutdown != nil {
			close(h.shutdown)
			h.shutdown = nil
		}
	}
}

func (h *Heartbeat) Unmarshal(message string) {
	// log.Println("Heartbeat received:", message)
}

func (h *Heartbeat) Ping() {
	h.Send(Headers{Type: "PING"})
}

func (h *Heartbeat) Pong() {
	h.Send(Headers{Type: "PONG"})
}
