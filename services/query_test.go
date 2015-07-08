package services

import (
	"fmt"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/dummy"
)

type MockService struct {
	queryHandlers map[string]QueryHandler
}

// ID of the service
func (service *MockService) ID() string {
	return "abc"
}

// Run the service
func (service *MockService) Run() error {
	return nil
}

func (service *MockService) QueryHandlers() QueryHandlers {
	return service.queryHandlers
}

func ExampleQuerySubscriber() {
	fields := pubsub.Fields{"query": "help"}
	query := pubsub.NewEvent("query", fields)
	li := dummy.Subscriber{
		Events: []*pubsub.Event{query},
	}
	Subscriber = &li
	em := dummy.Publisher{}
	Publisher = &em
	mock := MockService{
		queryHandlers: map[string]QueryHandler{"help": StaticHandler("squiggle")},
	}
	enabled = []Service{&mock}
	QuerySubscriber()
	fmt.Println(len(em.Events))
	fmt.Println(em.Events[0].StringField("message"))
	// Output:
	// 1
	// squiggle
}
