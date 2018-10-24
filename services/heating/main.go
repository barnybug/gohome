// Service to thermostatically control central heating by schedule and zone.
// Supports multiple temperature points on a daily schedule, temporary
// override ('party mode'), hibernation when the house is empty and advanced
// preheating ('holiday mode').
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
		i = NextWeekday(i)
	}
	return days
}

func NextWeekday(w time.Weekday) time.Weekday {
	if w == time.Saturday {
		return time.Sunday
	} else {
		return w + 1
	}
}

const Midnight = 24 * 60

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
			if start, ok = util.DOW[ps[0]]; !ok {
				return nil, fmt.Errorf("Invalid weekday: %s", ps[0])
			}
			if end, ok = util.DOW[ps[1]]; !ok {
				return nil, fmt.Errorf("Invalid weekday: %s", ps[1])
			}
			weekdays = append(weekdays, WeekdayRange(start, end)...)
		} else {
			if weekday, ok := util.DOW[d]; ok {
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
	for _, day := range WeekdayRange(time.Sunday, time.Saturday) {
		days[day] = []ScheduleTemp{}
	}
	for weekdays, mts := range conf {
		wds, err := ParseWeekdays(weekdays)
		if err != nil {
			return nil, err
		}
		for _, weekday := range wds {
			for _, arr := range mts {
				for at, temp := range arr {
					start, end, err := ParseHourMinuteRange(at)
					if err != nil {
						return nil, err
					}
					if temp < MinimumTemperature || temp > MaximumTemperature {
						return nil, fmt.Errorf("Temperature %.1f outside range %.1f <= t <= %.1f", temp, MinimumTemperature, MaximumTemperature)
					}
					if start > end {
						// spanning midnight, split
						tomorrow := NextWeekday(weekday)
						s2 := ScheduleTemp{0, end, temp}
						days[tomorrow] = append(days[tomorrow], s2)
						end = Midnight
					}
					s := ScheduleTemp{start, end, temp}
					days[weekday] = append(days[weekday], s)
				}
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
	Unoccupied    float64
	Holiday       time.Time
	Publisher     pubsub.Publisher
}

func (self *Service) Heartbeat() {
	self.Check(true)
	// emit event for datalogging
	fields := pubsub.Fields{
		"device":  self.HeatingDevice,
		"source":  "ch",
		"heating": self.State,
		"status":  self.Json(Clock()),
	}
	ev := pubsub.NewEvent("heating", fields)
	self.Publisher.Emit(ev)
	// repeat current state
	self.Command()
}

func (self *Service) Event(ev *pubsub.Event) {
	switch ev.Topic {
	case "temp":
		// temperature device update
		device := ev.Device()
		if zone, ok := self.Sensors[device]; ok {
			temp, _ := ev.Fields["temp"].(float64)
			timestamp := ev.Timestamp.Local() // must use Local time, as schedule is in local
			zone.Update(temp, timestamp)
			self.Check(false)
		}
	case "state":
		device := ev.Device()
		if device == "house.presence" {
			// house state update
			state := ev.Fields["state"]
			self.Occupied = (state != "Empty")
			if self.Occupied && !self.Holiday.IsZero() {
				// back from holiday - zero
				self.Holiday = time.Time{}
			}
			self.Check(false)
		}
	case "command":
		// set target command
		device := ev.Device()
		z := strings.Replace(device, "thermostat.", "", 1)
		if zone, ok := self.Zones[z]; ok {
			temp, _ := ev.Fields["temp"].(float64)
			now := Clock()
			duration := time.Duration(30) * time.Minute
			zone.setParty(temp, duration, now)
			log.Printf("Set %s to %v°C for %s", z, temp, util.FriendlyDuration(duration))
			self.Check(true)
		}
	}
}

func (self *Service) setParty(name string, temp float64, duration time.Duration, at time.Time) error {
	if name == "all" {
		for _, zone := range self.Zones {
			zone.setParty(temp, duration, at)
		}
		return nil
	} else if zone, ok := self.Zones[name]; ok {
		zone.setParty(temp, duration, at)
		return nil
	} else {
		return errors.New("Zone not found")
	}
}

func (self *Service) Target(zone *Zone, now time.Time) float64 {
	if now.Before(zone.PartyUntil) {
		return zone.PartyTemp
	} else if self.Occupied || (!self.Holiday.IsZero() && now.After(self.Holiday)) {
		return zone.Schedule.Target(now, self.Minimum)
	} else {
		return self.Unoccupied
	}
}

func (self *Service) Check(emitEvents bool) {
	now := Clock()
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

	if !self.Holiday.IsZero() {
		msg += fmt.Sprintf("\nHoliday until: %s", self.Holiday.Format(time.ANSIC))
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
	self.Occupied = false // updated by retained state topic
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
		case <-ticker.C:
			self.Heartbeat()
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
		if old, ok := self.Zones[zone]; ok {
			// preserve temp/party when live reloading
			z.Temp = old.Temp
			z.At = old.At
			z.PartyTemp = old.PartyTemp
			z.PartyUntil = old.PartyUntil
		}
		zones[zone] = z
		sensors[zoneConf.Sensor] = z
	}
	self.HeatingDevice = conf.Device
	self.Slop = conf.Slop
	self.Zones = zones
	self.Sensors = sensors
	self.Minimum = conf.Minimum
	self.Unoccupied = conf.Unoccupied
	log.Printf("%d zones configured", len(self.Zones))
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status":  self.queryStatus,
		"ch":      services.TextHandler(self.queryParty),
		"party":   services.TextHandler(self.queryParty),
		"holiday": services.TextHandler(self.queryHoliday),
		"help": services.StaticHandler("" +
			"status: get status\n" +
			"party [zone] temp [duration (1h)]: sets heating to temp for duration\n" +
			"holiday duration: sets holiday mode for this duration\n"),
	}
}

func (self *Service) queryStatus(q services.Question) services.Answer {
	now := Clock()
	return services.Answer{
		Text: self.Status(now),
		Json: self.Json(now),
	}
}

func parseParty(value string) (err error, zone string, temp float64, duration time.Duration) {
	vs := strings.Split(value, " ")
	if value == "" {
		err = errors.New("Required at least temperature")
		return
	}
	if len(vs) == 1 {
		// "all"
		vs = []string{"all", vs[0]}
	}
	ps := strings.SplitN(vs[0], ".", 2)
	zone = ps[len(ps)-1] // drop "thermostat."
	temp, err = strconv.ParseFloat(vs[1], 64)
	if err != nil {
		err = errors.New("Invalid temperature")
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
		duration, err = util.ParseDuration(vs[2])
		if err != nil {
			return
		}
	}
	return
}

func (self *Service) queryParty(q services.Question) string {
	err, zone, temp, duration := parseParty(q.Args)
	if err == nil {
		now := Clock()
		err = self.setParty(zone, temp, duration, now)
		if err == nil {
			self.Check(true)
			return fmt.Sprintf("Set %s to %v°C for %s", zone, temp, util.FriendlyDuration(duration))
		}
	}
	return fmt.Sprint(err)
}

func (self *Service) queryHoliday(q services.Question) string {
	if len(q.Args) == 0 {
		return "Duration required"
	}
	until, err := util.ParseRelative(Clock(), q.Args)
	if err != nil {
		return fmt.Sprint(err)
	}

	self.Holiday = until
	return fmt.Sprintf("Holiday until %s", until.Format(time.ANSIC))
}
