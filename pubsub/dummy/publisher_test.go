package dummy

import "github.com/barnybug/gohome/pubsub"

func ExamplePublisher() {
	var _ pubsub.Publisher = (*Publisher)(nil)
	// Output:
}
