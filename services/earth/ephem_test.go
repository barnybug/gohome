package earth

import (
	"fmt"
	"time"
)

func ExampleSunrise() {
	loc := Location{51.5072, -0.1275} // London
	t := time.Date(2014, 1, 2, 3, 4, 5, 6, time.UTC)
	fmt.Println(loc.Sunrise(t, ZenithOfficial))
	// Output:
	// 2014-01-02 08:06:15 +0000 UTC
}

func ExampleLight() {
	loc := Location{51.5072, -0.1275} // London
	t := time.Date(2014, 1, 2, 3, 4, 5, 6, time.UTC)
	fmt.Println(loc.Sunrise(t, ZenithCivil))
	// Output:
	// 2014-01-02 07:26:21 +0000 UTC
}

func ExampleSunset() {
	loc := Location{51.5072, -0.1275} // London
	t := time.Date(2014, 1, 2, 3, 4, 5, 6, time.UTC)
	fmt.Println(loc.Sunset(t, ZenithOfficial))
	// Output:
	// 2014-01-02 16:03:08 +0000 UTC
}

func ExampleSunset2() {
	loc := Location{51.5072, -0.1275} // London
	t := time.Date(2014, 6, 28, 3, 4, 5, 6, time.UTC)
	fmt.Println(loc.Sunset(t, ZenithOfficial))
	// Output:
	// 2014-06-28 20:21:40 +0000 UTC
}

func ExampleDark() {
	loc := Location{51.5072, -0.1275} // London
	t := time.Date(2014, 1, 2, 3, 4, 5, 6, time.UTC)
	fmt.Println(loc.Sunset(t, 85))
	// Output:
	// 2014-01-02 15:12:19 +0000 UTC
}

func ExampleDark2() {
	loc := Location{51.5072, -0.1275} // London
	t := time.Date(2014, 6, 28, 3, 4, 5, 6, time.UTC)
	fmt.Println(loc.Sunset(t, 85))
	// Output:
	// 2014-06-28 19:35:08 +0000 UTC
}
