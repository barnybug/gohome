// Service to thermostatically control central heating by schedule and zone.
// Supports multiple temperature points on a daily schedule, temporary override
// ('party mode'), and hibernation when the house is empty.
package heating

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

var Clock = func() time.Time {
	return time.Now()
}

var maxTempAge, _ = time.ParseDuration("6m")

type Schedule struct {
	Days map[time.Weekday][]MinuteTemp
}

type MinuteTemp struct {
	Minute int
	Temp   float64
}

var DOW = map[string]time.Weekday{
	time.Monday.String():    time.Monday,
	time.Tuesday.String():   time.Tuesday,
	time.Wednesday.String(): time.Wednesday,
	time.Thursday.String():  time.Thursday,
	time.Friday.String():    time.Friday,
	time.Saturday.String():  time.Saturday,
	time.Sunday.String():    time.Sunday,
}

func ParseHourMinute(at string) int {
	hm := strings.Split(at, ":")
	if len(hm) == 2 {
		hour, err := strconv.ParseInt(hm[0], 10, 32)
		if err == nil {
			min, err := strconv.ParseInt(hm[1], 10, 32)
			if err == nil {
				return int(hour*60 + min)
			}
		}
	}
	return 0
}

func NewSchedule(conf config.ScheduleConf) (*Schedule, error) {
	days := map[time.Weekday][]MinuteTemp{}
	for weekdays, mts := range conf {
		for _, weekday := range strings.Split(weekdays, ",") {
			if _, ok := DOW[weekday]; !ok {
				return nil, fmt.Errorf("Invalid weekday: %s", weekday)
			}
			sch := []MinuteTemp{}
			for _, arr := range mts {
				for at, temp := range arr {
					min := ParseHourMinute(at)
					sch = append(sch, MinuteTemp{min, temp})
				}
			}
			days[DOW[weekday]] = sch
		}
	}
	return &Schedule{Days: days}, nil
}

func (self *Schedule) Target(at time.Time) float64 {
	day_mins := at.Hour()*60 + at.Minute()
	sch, ok := self.Days[at.Weekday()]
	if !ok || day_mins < sch[0].Minute {
		// find last from a previous day
		// may have to look back multiple days (eg weekends missing)
		prev := at
		for i := 1; i < 7; i += 1 {
			prev = prev.Add(-24 * time.Hour)
			if sch, ok := self.Days[prev.Weekday()]; ok {
				return sch[len(sch)-1].Temp
			}
		}
	} else {
		// find value today
		var target float64
		for _, mt := range sch {
			if day_mins < mt.Minute {
				break
			}
			target = mt.Temp
		}
		return target
	}
	return 0
}

type Zone struct {
	Thermostat string
	Temp       float64
	At         time.Time
	Schedule   *Schedule
	PartyTemp  float64
	PartyUntil time.Time
	Sensor     string
}

func (self *Zone) Update(temp float64, at time.Time) {
	self.Temp = temp
	self.At = at
}

func (self *Zone) Check(now time.Time, target float64) bool {
	valid := now.Sub(self.At) < maxTempAge
	if !valid {
		return false
	}
	return (self.Temp < target)
}

func (self *Zone) setParty(temp float64, duration time.Duration, at time.Time) {
	self.PartyTemp = temp
	self.PartyUntil = at.Add(duration)
}

// Service heating
type Service struct {
	HeatingDevice    string
	Slop             float64
	Zones            map[string]*Zone
	Sensors          map[string]*Zone
	State            bool
	StateChanged     time.Time
	Occupied         bool
	UnoccupiedTarget float64
	Publisher        pubsub.Publisher
}

func isOccupied() bool {
	// get presence from store
	value, err := services.Stor.Get("gohome/state/events/state/house.presence")
	if err != nil {
		log.Println("Couldn't get current presence:", err)
		return true
	}
	event := pubsub.Parse(value)
	return (event.Fields["state"] != "Empty")
}

func (self *Service) Heartbeat(now time.Time) {
	self.Check(now, true)
	// emit event for datalogging
	fields := pubsub.Fields{
		"device":  self.HeatingDevice,
		"source":  "ch",
		"heating": self.State,
		"status":  self.Json(now),
	}
	ev := pubsub.NewEvent("heating", fields)
	self.Publisher.Emit(ev)
	// repeat current state
	self.Command()
}

func (self *Service) Event(ev *pubsub.Event) {
	now := ev.Timestamp.Local() // must use Local time, as schedule is in local
	switch ev.Topic {
	case "temp":
		// temperature device update
		device := services.Config.LookupDeviceName(ev)
		if zone, ok := self.Sensors[device]; ok {
			temp, _ := ev.Fields["temp"].(float64)
			zone.Update(temp, now)
			self.Check(now, false)
		}
	case "state":
		device := services.Config.LookupDeviceName(ev)
		if device == "house.presence" {
			// house state update
			state := ev.Fields["state"]
			self.Occupied = (state != "Empty")
			self.Check(now, false)
		}
	}
}

func (self *Service) setParty(zone string, temp float64, duration time.Duration, at time.Time) {
	if zone, ok := self.Zones[zone]; ok {
		zone.setParty(temp, duration, at)
	} else {
		log.Println("Not found:", zone)
	}
}

func (self *Service) Target(zone *Zone, now time.Time) float64 {
	if now.Before(zone.PartyUntil) {
		return zone.PartyTemp
	} else if self.Occupied {
		return zone.Schedule.Target(now)
	} else {
		return self.UnoccupiedTarget
	}
}

func (self *Service) Check(now time.Time, emitEvents bool) {
	state := false
	trigger := ""
	for id, zone := range self.Zones {
		target := self.Target(zone, now)
		if zone.Check(now, target) {
			state = true
			trigger = id
		}
		if emitEvents {
			// emit target event
			fields := pubsub.Fields{
				"device": zone.Thermostat,
				"source": "ch",
				"target": target,
			}
			ev := pubsub.NewEvent("thermostat", fields)
			self.Publisher.Emit(ev)
		}
	}

	if !self.State && state {
		log.Println("Turning on heating for:", trigger)
	} else if self.State && !state {
		log.Println("Turning off heating")
	}

	if self.State != state {
		self.State = state
		self.StateChanged = now
		self.Command()
	}

}

func (self *Service) Command() {
	command := "off"
	if self.State {
		command = "on"
	}
	ev := pubsub.NewCommand(self.HeatingDevice, command)
	self.Publisher.Emit(ev)
}

func (self *Service) ShortStatus(now time.Time) string {
	du := "unknown"
	if !self.StateChanged.IsZero() {
		du = util.ShortDuration(now.Sub(self.StateChanged))
	}
	return fmt.Sprintf("Heating: %v for %s", self.State, du)
}

func (self *Service) Status(now time.Time) string {
	msg := self.ShortStatus(now)
	for name, zone := range self.Zones {
		target := self.Target(zone, now)
		star := ""
		if zone.Temp < target {
			star = "*"
		}
		if zone.At.IsZero() {
			msg += fmt.Sprintf("\n%s: unknown [%v°C]", name, target)
		} else {
			msg += fmt.Sprintf("\n%s: %v°C at %s [%v°C]%s", name, zone.Temp, zone.At.Format(time.Stamp), target, star)
		}
	}
	return msg
}

func (self *Service) Json(now time.Time) interface{} {
	data := map[string]interface{}{}
	data["heating"] = self.State
	if !self.StateChanged.IsZero() {
		data["changed"] = self.StateChanged
	}
	devices := map[string]interface{}{}
	for name, zone := range self.Zones {
		target := self.Target(zone, now)
		if zone.At.IsZero() {
			devices[name] = map[string]interface{}{
				"temp":   nil,
				"target": target,
			}
		} else {
			devices[name] = map[string]interface{}{
				"temp":   zone.Temp,
				"at":     zone.At.Format(time.RFC3339),
				"target": target,
			}
		}
	}
	data["devices"] = devices
	return data
}

func (self *Service) ID() string {
	return "heating"
}

func (self *Service) Initialize(em pubsub.Publisher) {
	self.State = false
	self.Occupied = isOccupied()
	self.Publisher = em
	self.ConfigUpdated("gohome/config")
}

// Run the service
func (self *Service) Run() error {
	self.Initialize(services.Publisher)

	ticker := util.NewScheduler(time.Duration(0), time.Minute)
	events := services.Subscriber.FilteredChannel("temp", "state", "command")
	for {
		select {
		case ev := <-events:
			self.Event(ev)
		case tick := <-ticker.C:
			self.Heartbeat(tick)
		}
	}
	return nil
}

func (self *Service) ConfigUpdated(path string) {
	if path != "gohome/config" {
		return
	}
	conf := services.Config.Heating
	zones := map[string]*Zone{}
	sensors := map[string]*Zone{}
	for zone, zoneConf := range conf.Zones {
		thermostat := "thermostat." + zone
		schedule, err := NewSchedule(zoneConf.Schedule)
		if err != nil {
			log.Printf("Failed to load configuration: %s\n", err)
			return
		}
		z := &Zone{
			Schedule:   schedule,
			Sensor:     zoneConf.Sensor,
			Thermostat: thermostat,
		}
		zones[zone] = z
		sensors[zoneConf.Sensor] = z
	}
	self.HeatingDevice = conf.Device
	self.Slop = conf.Slop
	self.Zones = zones
	self.Sensors = sensors
	self.UnoccupiedTarget = conf.Unoccupied
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": self.queryStatus,
		"ch":     services.TextHandler(self.queryCh),
		"help": services.StaticHandler("" +
			"status: get status\n" +
			"ch temp [dur (1h)]: sets heating to temp for duration\n"),
	}
}

func (self *Service) queryStatus(q services.Question) services.Answer {
	now := Clock()
	return services.Answer{
		Text: self.Status(now),
		Json: self.Json(now),
	}
}

func parseSet(value string) (err error, zone string, temp float64, duration time.Duration) {
	vs := strings.Split(value, " ")
	if len(vs) < 2 {
		err = errors.New("required at least device and temperature")
		return
	}
	ps := strings.SplitN(vs[0], ".", 2)
	zone = ps[len(ps)-1] // drop "thermostat."
	temp, err = strconv.ParseFloat(vs[1], 64)
	if err != nil {
		return
	}
	duration = time.Duration(30) * time.Minute
	if len(vs) > 2 {
		duration, err = time.ParseDuration(vs[2])
		if err != nil {
			return
		}
	}
	return
}

func (self *Service) queryCh(q services.Question) string {
	err, zone, temp, duration := parseSet(q.Args)
	if err == nil {
		now := Clock()
		self.setParty(zone, temp, duration, now)
		self.Check(now, false)
		return fmt.Sprintf("Set to %v°C for %s", temp, util.FriendlyDuration(duration))
	} else {
		return "usage: ch <zone> <temp> <duration>"
	}
}
