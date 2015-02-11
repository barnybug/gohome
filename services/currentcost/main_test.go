package currentcost

import (
	"fmt"
	"time"
)

func ExampleUnparseables() {
	fmt.Println(parse(""))
	fmt.Println(parse("xyz"))
	fmt.Println(parse("<ch1><watts>abcde</watts></ch1><tmpr>19.1</tmpr>"))
	// Output:
	// <nil>
	// <nil>
	// <nil>
}

func ExampleParseable() {
	// TODO full message
	ev := parse("<ch1><watts>01234</watts></ch1><tmpr>19.1</tmpr>")
	loc, _ := time.LoadLocation("UTC")
	ev.Timestamp = time.Date(2014, 1, 2, 3, 4, 5, 987654321, loc)
	fmt.Println(ev)
	// Output:
	// {"origin":"cc","power":1234,"source":"cc01","temp":19.1,"timestamp":"2014-01-02 03:04:05.987654","topic":"power"}
}
