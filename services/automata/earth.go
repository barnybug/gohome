package automata

import (
	"log"
	"math"
	"time"

	"github.com/barnybug/gohome/services"
)

type Location struct {
	Latitude  float64
	Longitude float64
}

const (
	DegToRad           = math.Pi / 180
	RadToDeg           = 180 / math.Pi
	ZenithLight        = 88
	ZenithOfficial     = 90 + 50.0/60
	ZenithCivil        = 96
	ZenithNautical     = 102
	ZenithAstronomical = 108
)

func (pos Location) Sunrise(day time.Time, zenith float64) time.Time {
	return pos.calculate(day, zenith, true)
}

func (pos Location) Sunset(day time.Time, zenith float64) time.Time {
	return pos.calculate(day, zenith, false)
}

func (pos Location) calculate(now time.Time, zenith float64, sunrise bool) time.Time {
	// Sunrise/Sunset Algorithm taken from:
	// http://williams.best.vwh.net/sunrise_sunset_algorithm.htm
	day := now.YearDay()

	// convert the longitude to hour value and calculate an approximate time
	var lnHour = pos.Longitude / 15
	t := 0.0
	if sunrise {
		t = float64(day) + ((6 - lnHour) / 24)
	} else {
		t = float64(day) + ((18 - lnHour) / 24)
	}

	// calculate the Sun's mean anomaly
	M := (0.9856 * t) - 3.289

	// calculate the Sun's true longitude
	L := M + (1.916 * math.Sin(M*DegToRad)) + (0.020 * math.Sin(2*M*DegToRad)) + 282.634
	if L > 360 {
		L = L - 360
	} else if L < 0 {
		L = L + 360
	}

	// calculate the Sun's right ascension
	RA := RadToDeg * math.Atan(0.91764*math.Tan(L*DegToRad))
	if RA > 360 {
		RA = RA - 360
	} else if RA < 0 {
		RA = RA + 360
	}

	// right ascension value needs to be in the same quadrant
	Lquadrant := (math.Floor(L / 90)) * 90
	RAquadrant := (math.Floor(RA / 90)) * 90
	RA = RA + (Lquadrant - RAquadrant)

	// right ascension value needs to be converted into hours
	RA /= 15

	// calculate the Sun's declination
	sinDec := 0.39782 * math.Sin(L*DegToRad)
	cosDec := math.Cos(math.Asin(sinDec))

	// calculate the Sun's local hour angle
	cosH := (math.Cos(zenith*DegToRad) - (sinDec * math.Sin(pos.Latitude*DegToRad))) / (cosDec * math.Cos(pos.Latitude*DegToRad))
	H := 0.0
	if sunrise {
		H = 360 - RadToDeg*math.Acos(cosH)
	} else {
		H = RadToDeg * math.Acos(cosH)
	}
	H /= 15

	// calculate local mean time of rising/setting
	T := H + RA - (0.06571 * t) - 6.622

	// adjust back to UTC
	UT := T - lnHour
	if UT > 24 {
		UT -= 24
	} else if UT < 0 {
		UT += 24
	}

	hour := int(UT) % 24
	minute := int(UT*60) % 60
	second := int(UT*3600) % 60
	return time.Date(now.Year(), now.Month(), now.Day(),
		hour, minute, second, 0, time.UTC)
}

func nextEvent(loc Location) (at time.Time, name string) {
	now := time.Now()
	if sunrise := loc.Sunrise(now, ZenithOfficial); now.Before(sunrise) {
		at = sunrise
		name = "sunrise"
	} else if light := loc.Sunrise(now, ZenithLight); now.Before(light) {
		at = light
		name = "light"
	} else if dark := loc.Sunset(now, ZenithLight); now.Before(dark) {
		at = dark
		name = "dark"
	} else if sunset := loc.Sunset(now, ZenithOfficial); now.Before(sunset) {
		at = sunset
		name = "sunset"
	} else if sunrise := loc.Sunrise(now.Add(time.Hour*24), ZenithOfficial); now.Before(sunrise) {
		at = sunrise
		name = "sunrise"
	} else {
		log.Println("This shouldn't happen")
	}
	return
}

type TimeEvent struct {
	At    time.Time
	Event string
}

func earthChannel() chan TimeEvent {
	loc := Location{
		Latitude:  services.Config.Earth.Latitude,
		Longitude: services.Config.Earth.Longitude,
	}
	ch := make(chan TimeEvent)
	go func() {
		for {
			at, event := nextEvent(loc)
			delay := at.Sub(time.Now())
			log.Printf("Next: %s at %v (in %s)\n", event, at.Local(), delay)
			time.Sleep(delay)
			ch <- TimeEvent{at, event}
		}
	}()
	return ch
}
