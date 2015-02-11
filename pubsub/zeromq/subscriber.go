package zeromq

import (
	"github.com/barnybug/gohome/pubsub"
	"log"
	"strings"

	zmq "github.com/pebbe/zmq4"
)

func NewSubscriber(endpoint string, filter string, connect bool) pubsub.Subscriber {
	ch := make(chan *pubsub.Event, 16)
	go run(endpoint, filter, connect, ch)
	return pubsub.NewFilteredSubscriber("zeromq: "+endpoint, ch)
}

func ParseZmq(msg string) *pubsub.Event {
	// format: topic\0data
	//
	// topic is only used for subscription matching, present
	// in data so ignored otherwise)
	p := strings.Split(msg, "\000")
	if len(p) != 2 {
		return nil
	}
	return pubsub.Parse(p[1])
}

func run(endpoint string, filter string, connect bool, ch chan *pubsub.Event) {
	sub, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatalln("zmq NewSocket failed:", err)
	}
	defer sub.Close()
	if connect {
		err = sub.Connect(endpoint)
	} else {
		err = sub.Bind(endpoint)
	}
	if err != nil {
		log.Fatalln("zmq Bind failed:", err)
	}
	sub.SetSubscribe(filter)

	for {
		msg, e := sub.Recv(0)
		if e != nil {
			log.Fatal("error:", e)
		}

		event := ParseZmq(msg)
		if event != nil {
			ch <- event
		}
	}
}
