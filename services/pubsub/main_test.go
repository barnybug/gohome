package pubsub

import (
	"fmt"
	"github.com/barnybug/gohome/services"
)

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	var _ services.Queryable = (*Service)(nil)
	// Output:
}

func ExampleQuery() {
	var query services.Queryable = &Service{}
	q := services.Question{"status", "", "jabber:123"}
	h := query.QueryHandlers()["status"]
	fmt.Println(h(q).Text)
	// Output:
	// processed: 0
}
