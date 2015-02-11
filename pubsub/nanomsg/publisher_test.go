package nanomsg

import "github.com/barnybug/gohome/pubsub"

func ExampleInterfaces() {
	var _ pubsub.Publisher = &Publisher{}
	// Output:
}
