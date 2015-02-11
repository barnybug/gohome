package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	zmq "github.com/pebbe/zmq4"
)

var (
	endpoint = flag.String("s", "tcp://localhost:8791", "subscribe zmq endpoint")
	filter   = flag.String("f", "", "prefix to filter events with")
)

func main() {
	flag.Parse()

	subscriber, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatalln(err)
	}
	subscriber.Connect(*endpoint)
	subscriber.SetSubscribe(*filter)

	for {
		msg, _ := subscriber.Recv(0)
		parts := strings.Split(msg, "\000")
		fmt.Println(strings.Join(parts, ": "))
	}
}
