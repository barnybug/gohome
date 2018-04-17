package automata

import "time"

var now = func() time.Time { return time.Now() }

const Kitchen24 = "15:04"

// Interpolate an int between a start and end time
func tinterpolate(start string, end string, tempStart int, tempEnd int) int {
	tStart, _ := time.Parse(Kitchen24, start)
	tEnd, _ := time.Parse(Kitchen24, end)
	n := now()
	tRef := time.Date(0, 1, 1, n.Hour(), n.Minute(), n.Second(), 0, time.UTC)
	f := tRef.Sub(tStart).Seconds() / tEnd.Sub(tStart).Seconds()
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	return int(float64(tempEnd-tempStart)*f) + tempStart
}
