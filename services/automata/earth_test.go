package automata

import (
	"fmt"
	"time"
)

var london = Location{51.5072, -0.1275}
var time1 = time.Date(2014, 1, 2, 3, 4, 5, 6, time.UTC)
var time2 = time.Date(2014, 6, 28, 3, 4, 5, 6, time.UTC)
var time3 = time.Date(2014, 1, 2, 12, 0, 0, 0, time.UTC)

func ExampleSunrise() {
	fmt.Println(london.Sunrise(time1, ZenithOfficial))
	// Output:
	// 2014-01-02 08:06:15 +0000 UTC
}

func ExampleLight() {
	fmt.Println(london.Sunrise(time1, ZenithCivil))
	// Output:
	// 2014-01-02 07:26:21 +0000 UTC
}

func ExampleSunset() {
	fmt.Println(london.Sunset(time1, ZenithOfficial))
	// Output:
	// 2014-01-02 16:03:08 +0000 UTC
}

func ExampleSunset2() {
	fmt.Println(london.Sunset(time2, ZenithOfficial))
	// Output:
	// 2014-06-28 20:21:40 +0000 UTC
}

func ExampleDark() {
	fmt.Println(london.Sunset(time1, 85))
	// Output:
	// 2014-01-02 15:12:19 +0000 UTC
}

func ExampleDark2() {
	fmt.Println(london.Sunset(time2, 85))
	// Output:
	// 2014-06-28 19:35:08 +0000 UTC
}

func ExampleNextEvent() {
	fmt.Println(nextEvent(london, time1))
	fmt.Println(nextEvent(london, time2))
	fmt.Println(nextEvent(london, time3))
	// Output:
	// 2014-01-02 08:06:15 +0000 UTC sunrise
	// 2014-06-28 03:45:29 +0000 UTC sunrise
	// 2014-01-02 15:39:28 +0000 UTC dark
}

func ExamplePreviousEvent() {
	fmt.Println(previousEvent(london, time1))
	fmt.Println(previousEvent(london, time2))
	fmt.Println(previousEvent(london, time3))
	// Output:
	// 2014-01-01 16:02:04 +0000 UTC sunset
	// 2014-06-27 20:21:48 +0000 UTC sunset
	// 2014-01-02 08:29:57 +0000 UTC light
}
