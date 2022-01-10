package services

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/barnybug/gohome/pubsub"
)

// Query with `query`, waiting for `timeout` for results.
func Query(query string, timeout time.Duration) []*pubsub.Event {
	ch := QueryChannel(query, timeout)
	events := []*pubsub.Event{}
	for ev := range ch {
		events = append(events, ev)
	}
	return events
}

// Query with `query`, waiting for `timeout` for results.
func QueryChannel(query string, timeout time.Duration) <-chan *pubsub.Event {
	reply_to := fmt.Sprintf("_rpc.%d", rand.Int())
	ch := Subscriber.Subscribe(pubsub.Exact(reply_to))

	SendQuery(query, "rpc", "", reply_to)

	// close the listener after timeout
	go func() {
		time.Sleep(timeout)
		Subscriber.Close(ch)
	}()

	return ch
}

func RPC(query string, timeout time.Duration) (string, error) {
	ch := QueryChannel(query, timeout)
	for ev := range ch {
		return ev.StringField("message"), nil
	}
	return "", errors.New("timeout")
}
