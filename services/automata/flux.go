package automata

import (
	"errors"
	"regexp"
	"time"

	"github.com/barnybug/gohome/pubsub"
)

var now = func() time.Time { return time.Now() }

const Kitchen24 = "15:04"

type fluxParams struct {
	tStart, tEnd time.Time
	kStart, kEnd int
	lStart, lEnd int
}

var reTimeSpec = regexp.MustCompile(`^(\d\d?:\d\d)->(\d\d?:\d\d)$`)
var reParamSpec = regexp.MustCompile(`^(\w+)=(\d+)->(\d+)$`)

func fluxParse(args []interface{}) (p fluxParams, err error) {
	for _, arg := range args {
		s, ok := arg.(string)
		if !ok {
			err = errors.New("Unexpected argument (should be string)")
			return
		}
		m := reTimeSpec.FindStringSubmatch(s)
		if m != nil {
			p.tStart, _ = time.Parse(Kitchen24, m[1])
			p.tEnd, _ = time.Parse(Kitchen24, m[2])
			continue
		}

		m = reParamSpec.FindStringSubmatch(s)
		if m != nil {
			switch m[1] {
			case "temp":
				p.kStart = parseInt(m[2], 0)
				p.kEnd = parseInt(m[3], 0)
			case "level":
				p.lStart = parseInt(m[2], 0)
				p.lEnd = parseInt(m[3], 0)
			default:
				err = errors.New("Unexpected parameter name")
				return
			}
			continue
		}

		err = errors.New("Unexpected parameter")
		return
	}

	return
}

func fluxCommand(p fluxParams, device string) *pubsub.Event {
	fields := pubsub.Fields{
		"command": "on",
		"device":  device,
	}
	if p.kStart != 0 {
		k := tinterpolate(p.tStart, p.tEnd, p.kStart, p.kEnd)
		fields["temp"] = int(k)
	}
	if p.lStart != 0 {
		l := tinterpolate(p.tStart, p.tEnd, p.lStart, p.lEnd)
		fields["level"] = int(l)
	}

	ev := pubsub.NewEvent("command", fields)
	return ev
}

func tinterpolate(start time.Time, end time.Time, tempStart int, tempEnd int) int {
	n := now()
	ref := time.Date(0, 1, 1, n.Hour(), n.Minute(), n.Second(), 0, time.UTC)
	f := ref.Sub(start).Seconds() / end.Sub(start).Seconds()
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	return int(float64(tempEnd-tempStart)*f) + tempStart
}
