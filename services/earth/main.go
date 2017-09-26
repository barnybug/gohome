// Service to track sunrise, sunset and emit events when they occur on a daily basis.
//
// The order of events is:
// sunrise -> light -> dark -> sunset
//
// sunrise/sunset correspond to official sunset times of the sun crossing the horizon.
//
// light/dark correspond to when the sun is 2° above the horizon, which
// corresponds to being fairly light. These events are a better trigger for
// internal house hold lights, because at sunrise/set it will likely still be
// rather dark inside!
package earth

import (
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

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
	} else if midnight := now.Round(time.Hour * 24); now.Before(midnight) {
		at = midnight
		name = "midnight"
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

func eventChannel(loc Location) chan TimeEvent {
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

// Service earth
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "earth"
}

// Run the service
func (self *Service) Run() error {
	loc := Location{
		Latitude:  services.Config.Earth.Latitude,
		Longitude: services.Config.Earth.Longitude,
	}
	ticker := util.NewScheduler(time.Duration(0), time.Minute)
	earth := eventChannel(loc)
	for {
		select {
		case tev := <-earth:
			ev := pubsub.NewEvent("earth",
				pubsub.Fields{"device": "earth", "command": tev.Event, "source": "home"})
			services.Publisher.Emit(ev)
		case tick := <-ticker.C:
			ev := pubsub.NewEvent("time",
				pubsub.Fields{"device": "time", "hhmm": tick.Format("1504")})
			services.Publisher.Emit(ev)
		}
	}
	return nil
}
