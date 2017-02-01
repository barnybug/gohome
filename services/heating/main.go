// Service to thermostatically control central heating by schedule and zone.
// Supports multiple temperature points on a daily schedule, temporary override
// ('party mode'), and hibernation when the house is empty.
package heating

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
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
	Days map[time.Weekday][]ScheduleTemp
}

type ScheduleTemp struct {
	Start int
	End   int
	Temp  float64
}

const (
	MinimumTemperature = 1.0
	MaximumTemperature = 25.0
)

var DOW = map[string]time.Weekday{
	time.Monday.String():    time.Monday,
	time.Tuesday.String():   time.Tuesday,
	time.Wednesday.String(): time.Wednesday,
	time.Thursday.String():  time.Thursday,
	time.Friday.String():    time.Friday,
	time.Saturday.String():  time.Saturday,
	time.Sunday.String():    time.Sunday,
	"Mon":                   time.Monday,
	"Tue":                   time.Tuesday,
	"Wed":                   time.Wednesday,
	"Thu":                   time.Thursday,
	"Fri":                   time.Friday,
	"Sat":                   time.Saturday,
	"Sun":                   time.Sunday,
}

var reHourMinuteRange = regexp.MustCompile(`^(\d+):(\d+)-(\d+):(\d+)$`)

func ParseHourMinuteRange(s string) (int, int, error) {
	m := reHourMinuteRange.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, errors.New("Expected hh:mm-hh:mm")
	}
	sh, _ := strconv.Atoi(m[1])
	sm, _ := strconv.Atoi(m[2])
	eh, _ := strconv.Atoi(m[3])
	em, _ := strconv.Atoi(m[4])
	return sh*60 + sm, eh*60 + em, nil
}

func WeekdayRange(start, end time.Weekday) []time.Weekday {
	var days []time.Weekday
	i := start
	for {
		days = append(days, i)
		if i == end {
			break
		}
		if i == time.Saturday {
			i = time.Sunday
		} else {
			i += 1
		}
	}
	return days
}

func ParseWeekdays(s string) ([]time.Weekday, error) {
	var weekdays []time.Weekday
	for _, d := range strings.Split(s, ",") {
		if d == "Weekdays" {
			weekdays = append(weekdays, WeekdayRange(time.Monday, time.Friday)...)
		} else if d == "Weekends" {
			weekdays = append(weekdays, WeekdayRange(time.Saturday, time.Sunday)...)
		} else if d == "All" {
			weekdays = append(weekdays, WeekdayRange(time.Monday, time.Sunday)...)
		} else if strings.Contains(d, "-") {
			ps := strings.SplitN(d, "-", 2)
			if len(ps) != 2 {
				return nil, fmt.Errorf("Invalid range: %s", d)
			}
			var start, end time.Weekday
			var ok bool
			if start, ok = DOW[ps[0]]; !ok {
				return nil, fmt.Errorf("Invalid weekday: %s", ps[0])
			}
			if end, ok = DOW[ps[1]]; !ok {
				return nil, fmt.Errorf("Invalid weekday: %s", ps[1])
			}
			weekdays = append(weekdays, WeekdayRange(start, end)...)
		} else {
			if weekday, ok := DOW[d]; ok {
				weekdays = append(weekdays, weekday)
			} else {
				return nil, fmt.Errorf("Invalid weekday: %s", weekday)
			}
		}
	}
	return weekdays, nil
}

func NewSchedule(conf config.ScheduleConf) (*Schedule, error) {
	days := map[time.Weekday][]ScheduleTemp{}
	for weekdays, mts := range conf {
		wds, err := ParseWeekdays(weekdays)
		if err != nil {
			return nil, err
		}
		for _, weekday := range wds {
			sch := []ScheduleTemp{}
			for _, arr := range mts {
				for at, temp := range arr {
					start, end, err := ParseHourMinuteRange(at)
					if err != nil {
						return nil, err
					}
					if temp < MinimumTemperature || temp > MaximumTemperature {
						return nil, fmt.Errorf("Temperature %.1f outside range %.1f <= t <= %.1f", temp, MinimumTemperature, MaximumTemperature)
					}
					sch = append(sch, ScheduleTemp{start, end, temp})
				}
			}

			if existing, ok := days[weekday]; ok {
				// append to existing schedule for the day
				days[weekday] = append(existing, sch...)
			} else {
				days[weekday] = sch
			}
		}
	}
	return &Schedule{Days: days}, nil
}

func (self *Schedule) Target(at time.Time, def float64) float64 {
	day_mins := at.Hour()*60 + at.Minute()
	target := def
	if sts, ok := self.Days[at.Weekday()]; ok {
		var specific = 86401
		for _, st := range sts {
			if st.Start <= day_mins && day_mins < st.End && st.End-st.Start < specific {
				target = st.Temp
				specific = st.End - st.Start
			}
		}
	}
	return target
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
	HeatingDevice string
	Slop          float64
	Zones         map[string]*Zone
	Sensors       map[string]*Zone
	State         bool
	StateChanged  time.Time
	Occupied      bool
	Minimum       float64
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

func sensorTemp(sensor string) (temp float64, at time.Time) {
	// get temp from store
	value, err := services.Stor.Get("gohome/state/events/temp/" + sensor)
	if err != nil {
		return
	}
	event := pubsub.Parse(value)
	if event != nil {
		temp, _ = event.Fields["temp"].(float64)
		at = event.Timestamp
	}
	return
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

func (self *Service) setParty(zone string, temp float64, duration time.Duration, at time.Time) error {
	if zone, ok := self.Zones[zone]; ok {
		zone.setParty(temp, duration, at)
		return nil
	} else {
		return errors.New("Zone not found")
	}
}

func (self *Service) Target(zone *Zone, now time.Time) float64 {
	if now.Before(zone.PartyUntil) {
		return zone.PartyTemp
	} else if self.Occupied {
		return zone.Schedule.Target(now, self.Minimum)
	} else {
		return self.Minimum
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
			trv := target + self.Slop // adjusted target for trvs
			fields := pubsub.Fields{
				"device": zone.Thermostat,
				"source": "ch",
				"target": target,
				"trv":    trv,
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
	var keys []string
	length := 0
	for name := range self.Zones {
		keys = append(keys, name)
		if length < len(name) {
			length = len(name)
		}
	}
	sort.Strings(keys)

	for _, name := range keys {
		zone := self.Zones[name]
		target := self.Target(zone, now)
		star := ""
		if zone.Temp < target {
			star = "*"
		}
		f := "\n%-"
		f += fmt.Sprint(length)
		f += "s"
		if zone.At.IsZero() {
			msg += fmt.Sprintf(f+" unknown [%.1f°C]", name, target)
		} else {
			// pad names to same length
			msg += fmt.Sprintf(f+" %.1f°C at %s [%.1f°C]%s", name, zone.Temp, zone.At.Format(time.Stamp), target, star)
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
	services.SetupStore()
	self.State = false
	self.Occupied = isOccupied()
	self.Publisher = em
	self.ConfigUpdated("config")
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
	if path != "config" {
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

		// restore previous temperature
		temp, at := sensorTemp(zoneConf.Sensor)
		z.Update(temp, at)
	}
	self.HeatingDevice = conf.Device
	self.Slop = conf.Slop
	self.Zones = zones
	self.Sensors = sensors
	self.Minimum = conf.Minimum
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
	if temp < MinimumTemperature {
		err = errors.New("Below minimum temperature")
		return
	}
	if temp > MaximumTemperature {
		err = errors.New("Above maximum temperature")
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
		err = self.setParty(zone, temp, duration, now)
		if err == nil {
			self.Check(now, true)
			return fmt.Sprintf("Set %s to %v°C for %s", zone, temp, util.FriendlyDuration(duration))
		}
	}
	return fmt.Sprint(err)
}
