// Service to publish weather data to wunderground.com.
package wunderground

import (
	"fmt"
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"log"
	"net/http"
	"net/url"
	"time"
)

var updateInterval, _ = time.ParseDuration("5m")

type Metrics map[string]interface{}

type Wunderground struct {
	Url        string
	Id         string
	Password   string
	batch      Metrics
	lastUpdate time.Time
}

func NewWunderground(conf config.WundergroundConf) *Wunderground {
	w := &Wunderground{
		Url:      conf.Url,
		Id:       conf.Id,
		Password: conf.Password,
		batch:    make(Metrics),
	}
	return w
}

const TimeFormat = "2006-01-02 15:04:05"

func (self *Wunderground) RequestUri(now time.Time) string {
	dateutc := now.Format(TimeFormat)

	// build api request
	vs := url.Values{
		"action":       []string{"updateraw"},
		"ID":           []string{self.Id},
		"PASSWORD":     []string{self.Password},
		"dateutc":      []string{dateutc},
		"softwaretype": []string{"gohome"},
	}
	for k, v := range self.batch {
		vs[k] = []string{fmt.Sprintf("%v", v)}
	}
	return self.Url + "?" + vs.Encode()
}

func (self *Wunderground) Add(fields map[string]interface{}) {
	for k, v := range fields {
		self.batch[k] = v
	}
}

func (self *Wunderground) Update(fields map[string]interface{}) {
	self.Add(fields)

	if time.Since(self.lastUpdate) > updateInterval {
		now := time.Now()
		uri := self.RequestUri(now)
		resp, err := http.Get(uri)
		if err != nil {
			log.Println("Failed to send weather stats:", err)
			return
		}
		defer resp.Body.Close()

		// reset
		self.lastUpdate = now
		self.batch = make(map[string]interface{})
	}
}

func processEvent(ev *pubsub.Event, w *Wunderground) {
	device := services.Config.LookupDeviceName(ev)
	switch device {
	case services.Config.Weather.Outside.Temp:
		temp := ev.Fields["temp"].(float64)
		humd := ev.Fields["humidity"].(float64)
		w.Update(Metrics{
			// Fahrenheit
			"tempf":    temp/5.0*9.0 + 32,
			"humidity": humd})
	case services.Config.Weather.Outside.Rain:
		hour_rain := ev.Fields["hour_total"].(float64)
		day_rain := ev.Fields["day_total"].(float64)
		w.Update(Metrics{
			// Inches
			"rainin":      hour_rain / 25.4,
			"dailyrainin": day_rain / 25.4,
		})
	case services.Config.Weather.Outside.Wind:
		speed := ev.Fields["speed"].(float64)
		dir := ev.Fields["dir"].(float64)
		w.Update(Metrics{
			// mph
			"windspeedmph": speed * 2.237,
			"winddir":      dir,
		})
	case "pressure.pressure":
		pressure := ev.Fields["pressure"].(float64)
		w.Update(Metrics{
			// Convert millibars -> Inches
			"baromin": pressure * 0.0295301,
		})
	}
}

type WundergroundService struct{}

func (self *WundergroundService) Id() string {
	return "wunderground"
}

func (self *WundergroundService) Run() error {
	w := NewWunderground(services.Config.Wunderground)
	for ev := range services.Subscriber.Channel() {
		processEvent(ev, w)
	}
	return nil
}
