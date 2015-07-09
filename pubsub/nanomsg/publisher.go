package nanomsg

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"

	"github.com/gdamore/mangos/transport/inproc"

	"github.com/gdamore/mangos/protocol/pub"
	"github.com/gdamore/mangos/transport/tcp"
)

// Publisher for nanomsg
type Publisher struct {
	Channel  chan *pubsub.Event
	Endpoint string
	Connect  bool
}

// ID of Publisher
func (publ *Publisher) ID() string {
	return "nanomsg: " + publ.Endpoint
}

// NewPublisher creates a new publisher
func NewPublisher(endpoint string, connect bool) *Publisher {
	ch := make(chan *pubsub.Event)
	publ := Publisher{ch, endpoint, connect}
	go publ.run()
	return &publ
}

// Emit an event
func (publ *Publisher) Emit(ev *pubsub.Event) {
	publ.Channel <- ev
}

func (publ *Publisher) run() {
	sock, err := pub.NewSocket()
	if err != nil {
		log.Fatalln("pub.NewSocket error:", err)
	}
	sock.AddTransport(inproc.NewTransport())
	sock.AddTransport(tcp.NewTransport())
	defer sock.Close()
	if publ.Connect {
		err = sock.Dial(publ.Endpoint)
	} else {
		err = sock.Listen(publ.Endpoint)
	}
	if err != nil {
		log.Fatalln("sock connect failed:", err)
	}

	// sending on a socket straight away silently fails, so wait 20ms. ugh.
	time.Sleep(time.Millisecond * 20)

	for ev := range publ.Channel {
		// format: topic\0data
		data := fmt.Sprintf("%s\000%s", ev.Topic, ev.String())
		err := sock.Send([]byte(data))
		if err != nil {
			log.Fatalln("Failed to Send message:", err)
		}
	}
}
