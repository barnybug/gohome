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

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

var Clock = func() time.Time {
	return time.Now()
}

var maxTempAge, _ = time.ParseDuration("3m")

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

func NewSchedule(conf map[string][]map[string]float64) *Schedule {
	days := map[time.Weekday][]MinuteTemp{}
	for weekdays, mts := range conf {
		for _, weekday := range strings.Split(weekdays, ",") {
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
	return &Schedule{Days: days}
}

func (self *Schedule) Target(at time.Time) (temp float64) {
	day_mins := at.Hour()*60 + at.Minute()
	sch, ok := self.Days[at.Weekday()]
	if !ok {
		return 0.0
	}
	if day_mins < sch[0].Minute {
		// carry over last setting from yesterday
		yst := at.Add(-24 * time.Hour)
		sch = self.Days[yst.Weekday()]
		temp = sch[len(sch)-1].Temp
	} else {
		// find value today
		for _, mt := range sch {
			if day_mins >= mt.Minute {
				temp = mt.Temp
			}
		}
	}
	return
}

type Thermostat struct {
	Temp       float64
	At         time.Time
	Schedule   *Schedule
	PartyTemp  float64
	PartyUntil time.Time
}

func (self *Thermostat) Update(temp float64, at time.Time) {
	self.Temp = temp
	self.At = at
}

func (self *Thermostat) Check(now time.Time, target float64) bool {
	valid := now.Sub(self.At) < maxTempAge
	if !valid {
		return false
	}
	return (self.Temp < target)
}

func (self *Thermostat) setParty(temp float64, duration time.Duration, at time.Time) {
	self.PartyTemp = temp
	self.PartyUntil = at.Add(duration)
}

// Service heating
type Service struct {
	HeatingDevice string
	Slop          float64
	Thermostats   map[string]*Thermostat
	Schedules     map[string]*Schedule
	State         bool
	StateChanged  time.Time
	Occupied      bool
	Publisher     pubsub.Publisher
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
	self.Check(now)
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
		zone := device[5:] // drop "temp."
		if thermostat, ok := self.Thermostats[zone]; ok {
			temp, _ := ev.Fields["temp"].(float64)
			thermostat.Update(temp, now)
			self.Check(now)
		}
	case "state":
		device := services.Config.LookupDeviceName(ev)
		if device == "house.presence" {
			// house state update
			state := ev.Fields["state"]
			self.Occupied = (state != "Empty")
			self.Check(now)
		}
	}
}

func (self *Service) setParty(zone string, temp float64, duration time.Duration, at time.Time) {
	if thermostat, ok := self.Thermostats[zone]; ok {
		thermostat.setParty(temp, duration, at)
	} else {
		log.Println("Not found:", zone)
	}
}

func (self *Service) Target(thermostat *Thermostat, now time.Time) float64 {
	if now.Before(thermostat.PartyUntil) {
		return thermostat.PartyTemp
	} else if self.Occupied {
		return thermostat.Schedule.Target(now)
	} else {
		return self.Schedules["unoccupied"].Target(now)
	}
}

func (self *Service) Check(now time.Time) {
	state := false
	trigger := ""
	for id, thermostat := range self.Thermostats {
		target := self.Target(thermostat, now)
		if thermostat.Check(now, target) {
			state = true
			trigger = id
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
	for name, thermostat := range self.Thermostats {
		target := self.Target(thermostat, now)
		star := ""
		if thermostat.Temp < target {
			star = "*"
		}
		if thermostat.At.IsZero() {
			msg += fmt.Sprintf("\n%s: unknown [%v°C]", name, target)
		} else {
			msg += fmt.Sprintf("\n%s: %v°C at %s [%v°C]%s", name, thermostat.Temp, thermostat.At.Format(time.Stamp), target, star)
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
	for name, thermostat := range self.Thermostats {
		target := self.Target(thermostat, now)
		if thermostat.At.IsZero() {
			devices[name] = map[string]interface{}{
				"temp":   nil,
				"target": target,
			}
		} else {
			devices[name] = map[string]interface{}{
				"temp":   thermostat.Temp,
				"at":     thermostat.At.Format(time.RFC3339),
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
	schedules := map[string]*Schedule{}
	for k, sconf := range conf.Schedule {
		schedules[k] = NewSchedule(sconf)
	}

	thermostats := map[string]*Thermostat{}
	for _, name := range conf.Sensors {
		zone := name[5:] // drop "temp."
		if _, ok := conf.Schedule[zone]; !ok {
			log.Println("Missing schedule:", zone, "for device:", name)
			continue
		}
		thermostats[zone] = &Thermostat{Schedule: schedules[zone]}
	}
	self.HeatingDevice = conf.Device
	self.Slop = conf.Slop
	self.Thermostats = thermostats
	self.Schedules = schedules
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
		self.Check(now)
		return fmt.Sprintf("Set to %v°C for %s", temp, util.FriendlyDuration(duration))
	} else {
		return "usage: ch <zone> <temp> <duration>"
	}
}
