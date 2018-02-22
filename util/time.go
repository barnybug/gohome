package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	case d.Nanoseconds() >= 1000:
		ms := int(d.Seconds() * 1000)
		return plural(ms, "millisecond")
	case d.Nanoseconds() > 0:
		ns := d.Nanoseconds()
		return plural(int(ns), "nanosecond")
	}
	return "0 seconds"
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
	case d.Nanoseconds() >= 1000:
		ms := int(d.Seconds() * 1000)
		return number(ms, "ms")
	}
	return "0s"
}

var DOW = map[string]time.Weekday{
	"Monday":    time.Monday,
	"Tuesday":   time.Tuesday,
	"Wednesday": time.Wednesday,
	"Thursday":  time.Thursday,
	"Friday":    time.Friday,
	"Saturday":  time.Saturday,
	"Sunday":    time.Sunday,
	"Mon":       time.Monday,
	"Tue":       time.Tuesday,
	"Wed":       time.Wednesday,
	"Thu":       time.Thursday,
	"Fri":       time.Friday,
	"Sat":       time.Saturday,
	"Sun":       time.Sunday,
}

var reParts = regexp.MustCompile(`(\d+(?:\.\d+)?)([smhdwy])\s*`)

var durationUnits = map[string]time.Duration{
	"s": time.Second,
	"m": time.Minute,
	"h": time.Hour,
	"d": 24 * time.Hour,
	"w": 7 * 24 * time.Hour,
	"y": 365 * 24 * time.Hour,
}

var reDur1 = regexp.MustCompile(`^(\d+(?:\.\d+)?)([smhdwy])$`)
var reDur2 = regexp.MustCompile(`^(\d+(?:\.\d+)?)([smhdwy])\s*(\d+(?:\.\d+)?)([smhdwy])$`)

func duration(m []string) time.Duration {
	var i int
	i, _ = strconv.Atoi(m[0])
	return time.Duration(i) * durationUnits[m[1]]
}

// ParseDuration does the same as time.ParseDuration but understands more
// units (d for day, w for week, y for year).
func ParseDuration(s string) (total time.Duration, err error) {
	s = strings.TrimSpace(s)

	m1 := reDur1.FindStringSubmatch(s)
	if m1 != nil {
		return duration(m1[1:3]), nil
	}

	m2 := reDur2.FindStringSubmatch(s)
	if m2 != nil {
		return duration(m2[1:3]) + duration(m2[3:5]), nil
	}

	return 0, errors.New("invalid duration")
}

const weekday = "(?i)(Sun|Mon|Tue|Wed|Thu|Fri|Sat|Sunday|Monday|Tuesday|Wednesday|Thursday|Friday|Saturday)"

var reWeekday = regexp.MustCompile("^" + weekday + "$")
var reWeekdayTime = regexp.MustCompile("^" + weekday + ` (\d+(?:am|pm))$`)

func ParseDay(now time.Time, s string) time.Time {
	s = strings.Title(s)
	n := int(DOW[s] - now.Weekday())
	if n <= 0 {
		n += 7
	}
	return now.Truncate(time.Hour * 24).Add(24 * time.Hour * time.Duration(n))
}

func ParseTime(s string) time.Duration {
	t, _ := time.Parse("3pm", s)
	return time.Hour * time.Duration(t.Hour())
}

// ParseRelative understands durations and relative time points (eg Sunday 7pm)
func ParseRelative(now time.Time, s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	d, err := ParseDuration(s)
	if err == nil {
		return now.Add(d), nil
	}

	m1 := reWeekday.FindStringSubmatch(s)
	if m1 != nil {
		return ParseDay(now, m1[1]), nil
	}

	m2 := reWeekdayTime.FindStringSubmatch(s)
	if m2 != nil {
		d := ParseDay(now, m2[1]).Add(ParseTime(m2[2]))
		return d, nil
	}

	return time.Time{}, errors.New("invalid relative time")
}
