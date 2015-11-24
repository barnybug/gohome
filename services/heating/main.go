// Service to thermostatically control central heating by schedule and zone.
// Supports multiple temperature points on a daily schedule, temporary override
// ('party mode'), and hibernation when the house is empty.
package heating

import (
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

type Device struct {
	Schedule *Schedule
	Temp     float64
	At       time.Time
}

func (self *Device) Update(temp float64, at time.Time) {
	self.Temp = temp
	self.At = at
}

func (self *Device) Check(now time.Time, target float64) bool {
	valid := now.Sub(self.At) < maxTempAge
	if !valid {
		return false
	}
	return (self.Temp < target)
}

type Thermostat struct {
	HeatingDevice string
	Slop          float64
	Devices       map[string]*Device
	Schedules     map[string]*Schedule
	State         bool
	StateChanged  time.Time
	Occupied      bool
	PartyTemp     float64
	PartyUntil    time.Time
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

func NewThermostat(config config.HeatingConf, em pubsub.Publisher) *Thermostat {
	self := Thermostat{
		State:     false,
		Occupied:  isOccupied(),
		Publisher: em,
	}
	self.ConfigUpdated(config)
	return &self
}

func (self *Thermostat) ConfigUpdated(config config.HeatingConf) {
	schedules := map[string]*Schedule{}
	for k, conf := range config.Schedule {
		schedules[k] = NewSchedule(conf)
	}

	devices := map[string]*Device{}
	for _, name := range config.Sensors {
		sname := name[5:] // drop "temp."
		if _, ok := config.Schedule[sname]; !ok {
			log.Println("Missing schedule:", sname, "for device:", name)
			continue
		}
		devices[name] = &Device{Schedule: schedules[sname]}
	}
	self.HeatingDevice = config.Device
	self.Slop = config.Slop
	self.Devices = devices
	self.Schedules = schedules

}

func (self *Thermostat) Heartbeat(now time.Time) {
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

func (self *Thermostat) Event(ev *pubsub.Event) {
	now := ev.Timestamp.Local() // must use Local time, as schedule is in local
	switch ev.Topic {
	case "temp":
		// temperature device update
		device := services.Config.LookupDeviceName(ev)
		if dev, ok := self.Devices[device]; ok {
			temp, _ := ev.Fields["temp"].(float64)
			dev.Update(temp, now)
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
	case "command":
		if ev.Target() == "ch" {
			value, ok := ev.Fields["value"].(string)
			if ok {
				err := self.setParty(value, ev.Timestamp)
				if err == nil {
					self.Check(now)
					// TODO - response
				} else {
					// TODO - error response
				}
			}
		}
	}
}

func (self *Thermostat) setParty(value string, at time.Time) error {
	vs := strings.Split(value, " ")
	temp, err := strconv.ParseFloat(vs[0], 64)
	if err != nil {
		return err
	}
	duration := time.Duration(30) * time.Minute
	if len(vs) > 1 {
		duration, err = time.ParseDuration(vs[1])
		if err != nil {
			return err
		}
	}
	self.PartyTemp = temp
	self.PartyUntil = at.Add(duration)
	return nil
}

func (self *Thermostat) Target(device *Device, now time.Time) float64 {
	if now.Before(self.PartyUntil) {
		return self.PartyTemp
	} else if self.Occupied {
		return device.Schedule.Target(now)
	} else {
		return self.Schedules["unoccupied"].Target(now)
	}
}

func (self *Thermostat) Check(now time.Time) {
	state := false
	trigger := ""
	for id, device := range self.Devices {
		target := self.Target(device, now)
		if device.Check(now, target) {
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

func (self *Thermostat) Command() {
	command := "off"
	if self.State {
		command = "on"
	}
	ev := pubsub.NewCommand(self.HeatingDevice, command, 0)
	self.Publisher.Emit(ev)
}

func (self *Thermostat) ShortStatus(now time.Time) string {
	du := "unknown"
	if !self.StateChanged.IsZero() {
		du = util.ShortDuration(now.Sub(self.StateChanged))
	}
	return fmt.Sprintf("Heating: %v for %s", self.State, du)
}

func (self *Thermostat) Status(now time.Time) string {
	msg := self.ShortStatus(now)
	for name, device := range self.Devices {
		target := self.Target(device, now)
		n := strings.Split(name, ".")
		star := ""
		if device.Temp < target {
			star = "*"
		}
		if device.At.IsZero() {
			msg += fmt.Sprintf("\n%s: unknown [%v°C]", n[1], target)
		} else {
			msg += fmt.Sprintf("\n%s: %v°C at %s [%v°C]%s", n[1], device.Temp, device.At.Format(time.Stamp), target, star)
		}
	}
	return msg
}

func (self *Thermostat) Json(now time.Time) interface{} {
	data := map[string]interface{}{}
	data["heating"] = self.State
	if !self.StateChanged.IsZero() {
		data["changed"] = self.StateChanged
	}
	devices := map[string]interface{}{}
	for name, device := range self.Devices {
		target := self.Target(device, now)
		if device.At.IsZero() {
			devices[name] = map[string]interface{}{
				"temp":   nil,
				"target": target,
			}
		} else {
			devices[name] = map[string]interface{}{
				"temp":   device.Temp,
				"at":     device.At.Format(time.RFC3339),
				"target": target,
			}
		}
	}
	data["devices"] = devices
	return data
}

// Service heating
type Service struct {
	thermo *Thermostat
}

func (self *Service) ID() string {
	return "heating"
}

// Run the service
func (self *Service) Run() error {
	self.thermo = NewThermostat(services.Config.Heating, services.Publisher)
	ticker := util.NewScheduler(time.Duration(0), time.Minute)
	events := services.Subscriber.FilteredChannel("temp", "state", "command")
	for {
		select {
		case ev := <-events:
			self.thermo.Event(ev)
		case tick := <-ticker.C:
			self.thermo.Heartbeat(tick)
		}
	}
	return nil
}

func (self *Service) ConfigUpdated(path string) {
	if path == "gohome/config" {
		self.thermo.ConfigUpdated(services.Config.Heating)
	}
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
		Text: self.thermo.Status(now),
		Json: self.thermo.Json(now),
	}
}

func (self *Service) queryCh(q services.Question) string {
	err := self.thermo.setParty(q.Args, Clock())
	if err == nil {
		return fmt.Sprintf("Set to %v°C until %s", self.thermo.PartyTemp, self.thermo.PartyUntil.Format(time.Stamp))
	} else {
		return "usage: ch <temp> <duration>"
	}
}
