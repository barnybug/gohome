package util

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ExampleNextSchedule() {
	now := time.Date(2012, 1, 1, 7, 0, 0, 0, time.UTC)
	d6h, _ := time.ParseDuration("6h")
	d8h, _ := time.ParseDuration("8h")
	d12h, _ := time.ParseDuration("12h")
	d22h, _ := time.ParseDuration("22h")
	d24h, _ := time.ParseDuration("24h")

	fmt.Println(NextSchedule(now, d6h, d24h))
	fmt.Println(NextSchedule(now, d8h, d24h))
	fmt.Println(NextSchedule(now, d22h, d24h))
	fmt.Println(NextSchedule(now, d8h, d12h))
	fmt.Println(NextSchedule(now, d6h, d12h))
	// Output:
	// 2012-01-02 06:00:00 +0000 UTC
	// 2012-01-01 08:00:00 +0000 UTC
	// 2012-01-01 22:00:00 +0000 UTC
	// 2012-01-01 08:00:00 +0000 UTC
	// 2012-01-01 18:00:00 +0000 UTC
}

func ExampleFriendlyDuration() {
	d1, _ := time.ParseDuration("48h")
	d2, _ := time.ParseDuration("26.5h")
	d3, _ := time.ParseDuration("5h59m")
	d4, _ := time.ParseDuration("37m1s")
	d5, _ := time.ParseDuration("1500ms")
	d6, _ := time.ParseDuration("500ms")
	d7, _ := time.ParseDuration("500ns")
	d8, _ := time.ParseDuration("0ms")

	fmt.Println(FriendlyDuration(d1))
	fmt.Println(FriendlyDuration(d2))
	fmt.Println(FriendlyDuration(d3))
	fmt.Println(FriendlyDuration(d4))
	fmt.Println(FriendlyDuration(d5))
	fmt.Println(FriendlyDuration(d6))
	fmt.Println(FriendlyDuration(d7))
	fmt.Println(FriendlyDuration(d8))
	// Output:
	// 2 days
	// 1 day 2 hours
	// 5 hours 59 minutes
	// 37 minutes 1 second
	// 1 second
	// 500 milliseconds
	// 500 nanoseconds
	// 0 seconds
}

func ExampleShortDuration() {
	d1, _ := time.ParseDuration("48h")
	d2, _ := time.ParseDuration("26.5h")
	d3, _ := time.ParseDuration("5h59m")
	d4, _ := time.ParseDuration("37m1s")
	d5, _ := time.ParseDuration("1500ms")
	d6, _ := time.ParseDuration("500ms")
	d7, _ := time.ParseDuration("500ns")

	fmt.Println(ShortDuration(d1))
	fmt.Println(ShortDuration(d2))
	fmt.Println(ShortDuration(d3))
	fmt.Println(ShortDuration(d4))
	fmt.Println(ShortDuration(d5))
	fmt.Println(ShortDuration(d6))
	fmt.Println(ShortDuration(d7))
	// Output:
	// 2d
	// 1d 2h
	// 5h 59m
	// 37m 1s
	// 1s
	// 500ms
	// 0s
}

var testParseDurationTable = []struct {
	s string
	t time.Duration
}{
	{"1s", time.Second},
	{"1s ", time.Second},
	{" 1s ", time.Second},
	{"1m", time.Minute},
	{"60m", time.Hour},
	{"1h", time.Hour},
	{"1h 5m", 65 * time.Minute},
	{"1h  5m", 65 * time.Minute},
	{"1d", 24 * time.Hour},
	{"1w", 7 * 24 * time.Hour},
	{"1y", 365 * 24 * time.Hour},
}

func TestParseDuration(t *testing.T) {
	for _, conf := range testParseDurationTable {
		result, err := ParseDuration(conf.s)
		assert.NoError(t, err)
		assert.Equal(t, conf.t, result)
	}
}

var testParseDurationErrorTable = []string{
	"xyz",
	"1x",
	"12",
	"h5m",
	"1h.5m",
	"1h 5x",
	"-1h",
	"1hh",
}

func TestParseDurationError(t *testing.T) {
	for _, s := range testParseDurationErrorTable {
		_, err := ParseDuration(s)
		assert.Error(t, err)
	}
}

var testParseRelativeTable = []struct {
	s string
	t string
}{
	{"1s", "2018-02-04T10:20:01Z"},
	{"1m", "2018-02-04T10:21:00Z"},
	{"1h", "2018-02-04T11:20:00Z"},
	{"1d", "2018-02-05T10:20:00Z"},
	{"1w", "2018-02-11T10:20:00Z"},
	{"1y", "2019-02-04T10:20:00Z"},

	{"Sat", "2018-02-10T00:00:00Z"},
	{"Sun", "2018-02-11T00:00:00Z"},
	{"Mon", "2018-02-05T00:00:00Z"},
	{"Monday", "2018-02-05T00:00:00Z"},
	{"mon", "2018-02-05T00:00:00Z"},
	{"monday", "2018-02-05T00:00:00Z"},

	{"Sat 7am", "2018-02-10T07:00:00Z"},
	{"Sun 1pm", "2018-02-11T13:00:00Z"},
}

func TestParseRelative(t *testing.T) {
	now := time.Date(2018, 2, 4, 10, 20, 0, 0, time.UTC) // Sunday
	for _, conf := range testParseRelativeTable {
		tm, err := ParseRelative(now, conf.s)
		assert.NoError(t, err)
		assert.Equal(t, conf.t, tm.Format(time.RFC3339))
	}
}
