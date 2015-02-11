package nanomsg

import (
	"github.com/barnybug/gohome/pubsub"
	"log"
	"strings"

	"github.com/gdamore/mangos"
	"github.com/gdamore/mangos/protocol/sub"
	"github.com/gdamore/mangos/transport/inproc"
	"github.com/gdamore/mangos/transport/tcp"
)

func NewSubscriber(endpoint string, filter string, connect bool) pubsub.Subscriber {
	ch := make(chan *pubsub.Event, 16)
	go run(endpoint, filter, connect, ch)
	return pubsub.NewFilteredSubscriber("nanomsg: "+endpoint, ch)
}

func ParseMsg(msg string) *pubsub.Event {
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
	sock, err := sub.NewSocket()
	if err != nil {
		log.Fatalln("sub.NewSocket error:", err)
	}
	sock.AddTransport(inproc.NewTransport())
	sock.AddTransport(tcp.NewTransport())
	defer sock.Close()
	if connect {
		err = sock.Dial(endpoint)
	} else {
		err = sock.Listen(endpoint)
	}
	if err != nil {
		log.Fatalln("sock connect failed:", err)
	}
	err = sock.SetOption(mangos.OptionSubscribe, []byte(filter))
	if err != nil {
		log.Fatalln("sock SetOption failed:", err)
	}

	for {
		msg, e := sock.Recv()
		if e != nil {
			log.Fatal("error:", e)
		}

		event := ParseMsg(string(msg))
		if event != nil {
			ch <- event
		}
	}
}
