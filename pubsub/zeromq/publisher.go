package zeromq

import (
	"fmt"
	"github.com/barnybug/gohome/pubsub"
	"time"

	zmq "github.com/pebbe/zmq4"
)

type Publisher struct {
	Channel  chan *pubsub.Event
	Endpoint string
	Connect  bool
}

func (self *Publisher) Id() string {
	return "zeromq: " + self.Endpoint
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
	pub, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		panic("Couldn't create zmq socket")
	}
	defer pub.Close()
	if self.Connect {
		pub.Connect(self.Endpoint)
	} else {
		pub.Bind(self.Endpoint)
	}

	// sending on a socket straight away silently fails, so wait 20ms. ugh.
	time.Sleep(time.Millisecond * 20)

	for ev := range self.Channel {
		// format: topic\0data
		//
		// The zmq recommendation is to use a multipart message for this now,
		// left unchanged as a legacy.
		data := fmt.Sprintf("%s\000%s", ev.Topic, ev.String())
		pub.Send(data, 0)
	}
}
