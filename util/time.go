package util

import (
	"fmt"
	"time"

	"errors"
)

type Scheduler struct {
	C <-chan time.Time
}

func NextSchedule(now time.Time, offset time.Duration, d time.Duration) time.Time {
	t := now.Truncate(d).Add(offset)
	if t.After(now) {
		return t
	} else {
		return t.Add(d)
	}
}

// A schedulable Ticker
// NewScheduler returns a new Scheduler containing a channel that will send
// the time with a period specified by the duration argument, at the specified
// offset into the day.
func NewScheduler(offset time.Duration, d time.Duration) *Scheduler {
	if d <= 0 {
		panic(errors.New("non-positive interval for NewScheduler"))
	}

	now := time.Now()
	next := NextSchedule(now, offset, d)
	dnext := next.Sub(now)

	// Give the channel a 1-element time buffer.
	// If the client falls behind while reading, we drop ticks
	// on the floor until the client catches up.
	c := make(chan time.Time, 1)
	t := &Scheduler{
		C: c,
	}

	time.AfterFunc(dnext, func() {
		for {
			c <- time.Now()
			next = next.Add(d)
			dnext = next.Sub(time.Now())
			time.Sleep(dnext)
		}
	})

	return t
}

func plural(n int, suffix string) string {
	switch n {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%d %s", n, suffix)
	default:
		return fmt.Sprintf("%d %ss", n, suffix)
	}
}

func number(n int, suffix string) string {
	switch n {
	case 0:
		return ""
	default:
		return fmt.Sprintf("%d%s", n, suffix)
	}
}

func joinpair(a, b string) string {
	if a != "" && b != "" {
		return a + " " + b
	}
	return a + b
}

func FriendlyDuration(d time.Duration) string {
	switch {
	case d.Hours() >= 24:
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) - days*24
		return joinpair(plural(days, "day"), plural(hours, "hour"))
	case d.Hours() >= 1:
		hours := int(d.Hours())
		mins := int(int(d.Minutes()) - 60*hours)
		return joinpair(plural(hours, "hour"), plural(mins, "minute"))
	case d.Minutes() >= 1:
		mins := int(d.Minutes())
		secs := int(int(d.Seconds()) - 60*mins)
		return joinpair(plural(mins, "minute"), plural(secs, "second"))
	case d.Seconds() >= 1:
		secs := int(d.Seconds())
		return plural(secs, "second")
	default:
		ms := int(d.Seconds() * 1000)
		return plural(ms, "millisecond")
	}
	return ""
}

func ShortDuration(d time.Duration) string {
	switch {
	case d.Hours() >= 24:
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) - days*24
		return joinpair(number(days, "d"), number(hours, "h"))
	case d.Hours() >= 1:
		hours := int(d.Hours())
		mins := int(int(d.Minutes()) - 60*hours)
		return joinpair(number(hours, "h"), number(mins, "m"))
	case d.Minutes() >= 1:
		mins := int(d.Minutes())
		secs := int(int(d.Seconds()) - 60*mins)
		return joinpair(number(mins, "m"), number(secs, "s"))
	case d.Seconds() >= 1:
		secs := int(d.Seconds())
		return number(secs, "s")
	default:
		ms := int(d.Seconds() * 1000)
		return number(ms, "ms")
	}
	return ""
}
