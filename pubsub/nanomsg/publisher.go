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

type Publisher struct {
	Channel  chan *pubsub.Event
	Endpoint string
	Connect  bool
}

func (self *Publisher) Id() string {
	return "nanomsg: " + self.Endpoint
}

func NewPublisher(endpoint string, connect bool) *Publisher {
	ch := make(chan *pubsub.Event)
	self := Publisher{ch, endpoint, connect}
	go self.run()
	return &self
}

func (self *Publisher) Emit(ev *pubsub.Event) {
	self.Channel <- ev
}

func (self *Publisher) run() {
	sock, err := pub.NewSocket()
	if err != nil {
		log.Fatalln("pub.NewSocket error:", err)
	}
	sock.AddTransport(inproc.NewTransport())
	sock.AddTransport(tcp.NewTransport())
	defer sock.Close()
	if self.Connect {
		err = sock.Dial(self.Endpoint)
	} else {
		err = sock.Listen(self.Endpoint)
	}
	if err != nil {
		log.Fatalln("sock connect failed:", err)
	}

	// sending on a socket straight away silently fails, so wait 20ms. ugh.
	time.Sleep(time.Millisecond * 20)

	for ev := range self.Channel {
		// format: topic\0data
		data := fmt.Sprintf("%s\000%s", ev.Topic, ev.String())
		err := sock.Send([]byte(data))
		if err != nil {
			log.Fatalln("Failed to Send message:", err)
		}
	}
}
