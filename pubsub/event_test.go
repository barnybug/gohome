package pubsub

import (
	"fmt"
	"time"
)

func ExampleString() {
	ev := NewEvent("test", nil)
	loc, _ := time.LoadLocation("UTC")
	ev.Timestamp = time.Date(2014, 1, 2, 3, 4, 5, 987654321, loc)
	fmt.Println(ev.String())
	//Output: {"timestamp":"2014-01-02 03:04:05.987","topic":"test"}
}

func ExampleParseWithTimestamp() {
	ev := Parse(`{"timestamp":"2014-01-02 03:04:05.987","topic":"test","field":"value"}`, "")
	fmt.Println(ev.Topic)
	fmt.Println(ev.Timestamp)
	fmt.Println(ev.Fields)
	// Output:
	// test
	// 2014-01-02 03:04:05.987 +0000 UTC
	// map[field:value]
}

func ExampleParseWithoutTimestamp() {
	ev := Parse(`{"topic":"test","field":"value"}`, "")
	fmt.Println(ev.Topic)
	fmt.Println(ev.Fields)
	// Output:
	// test
	// map[field:value]
}

func ExampleParseBad() {
	ev := Parse(`{`, "")
	fmt.Println(ev)
	// Output:
	// <nil>
}
