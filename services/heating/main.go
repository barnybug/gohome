// Service to thermostatically control central heating by schedule and zone.
// Supports multiple temperature points on a daily schedule, temporary
// override ('party mode'), hibernation when the house is empty.
package heating

import (
	"errors"
	"fmt"
	"log"
	"sort"
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

var maxTempAge, _ = time.ParseDuration("6m")

const (
	MinimumTemperature = 1.0
	MaximumTemperature = 25.0
)

const Midnight = 24 * 60

type Zone struct {
	Thermostat string
	Temp       float64
	Target     float64
	Rate       float64
	At         time.Time
	PartyTemp  float64
	PartyUntil time.Time
	Sensor     string
}

func (self *Zone) Update(temp float64, at time.Time) {
	if !self.At.IsZero() && !self.At.Equal(at) {
		gap := at.Sub(self.At).Seconds()
		rate := (temp - self.Temp) / gap
		// weighted rolling average
		self.Rate = rate*0.8 + self.Rate*0.2
	}
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
	config        *services.ConfigService
	HeatingDevice string
	Slop          float64
	Zones         map[string]*Zone
	Sensors       map[string]*Zone
	State         bool
	StateChanged  time.Time
	Minimum       float64
	Publisher     pubsub.Publisher
}

func (self *Service) Heartbeat() {
	self.Check(true)
	if self.HeatingDevice == "auto" {
		// no specific device to switch on or off, the hive thermostat directly controls
		// the call for heat from the boiler controller.
		return
	}
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
	} else {
		return zone.Target
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
			fields := pubsub.Fields{
				"device": zone.Thermostat,
				"source": "ch",
				"target": target,
				"temp":   zone.Temp,
			}
			if now.Before(zone.PartyUntil) {
				fields["boost"] = zone.PartyUntil.Sub(now).Seconds()
			}
			ev := pubsub.NewEvent("thermostat", fields)
			self.Publisher.Emit(ev)
		}
	}
	if self.HeatingDevice == "auto" {
		return
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
			msg += fmt.Sprintf(f+" %.1f°C %+.1f°C/hr at %s [%.1f°C]%s", name, zone.Temp, zone.Rate*3600, zone.At.Format(time.Stamp), target, star)
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
				"rate":   zone.Rate,
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

func (self *Service) Init() error {
	self.State = false
	if self.Publisher == nil {
		self.Publisher = services.Publisher
	}
	if self.config == nil {
		self.config = services.WaitForConfig()
	}
	self.configUpdated()
	return nil
}

// Run the service
func (self *Service) Run() error {
	// Run at 2s past the minute to give automata time to send targets
	ticker := util.NewScheduler(time.Duration(2), time.Minute)
	events := services.Subscriber.Subscribe(pubsub.Prefix("temp"), pubsub.Prefix("command"))
	for {
		select {
		case ev := <-events:
			self.Event(ev)
		case <-ticker.C:
			self.Heartbeat()
		case <-self.config.Updated:
			self.configUpdated()
		}
	}
}

func (self *Service) configUpdated() {
	conf := self.config.Value.Heating
	zones := map[string]*Zone{}
	sensors := map[string]*Zone{}
	for zone, sensor := range conf.Zones {
		thermostat := "thermostat." + zone
		z := &Zone{
			Sensor:     sensor,
			Thermostat: thermostat,
			Target:     conf.Minimum,
		}
		if old, ok := self.Zones[zone]; ok {
			// preserve temp/party when live reloading
			z.Temp = old.Temp
			z.Target = old.Target
			z.Rate = old.Rate
			z.At = old.At
			z.PartyTemp = old.PartyTemp
			z.PartyUntil = old.PartyUntil
		}
		zones[zone] = z
		sensors[sensor] = z
	}
	self.HeatingDevice = conf.Device
	self.Slop = conf.Slop
	self.Zones = zones
	self.Sensors = sensors
	self.Minimum = conf.Minimum
	log.Printf("%d zones configured", len(self.Zones))
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": self.queryStatus,
		"target": self.queryTarget,
		"ch":     services.TextHandler(self.queryParty),
		"party":  services.TextHandler(self.queryParty),
		"help": services.StaticHandler("" +
			"status: get status\n" +
			"party [zone] temp [duration (1h)]: sets heating to temp for duration\n"),
	}
}

func (self *Service) queryStatus(q services.Question) services.Answer {
	now := Clock()
	return services.Answer{
		Text: self.Status(now),
		Json: self.Json(now),
	}
}

func parseTarget(value string) (err error, zone string, target float64) {
	vs := strings.Split(value, " ")
	if len(vs) != 2 {
		err = errors.New("Required zone temperature")
		return
	}
	zone = vs[0]
	target, err = strconv.ParseFloat(vs[1], 64)
	if err != nil {
		err = errors.New("Invalid temperature")
		return
	}
	if target < MinimumTemperature {
		err = errors.New("Below minimum temperature")
		return
	}
	if target > MaximumTemperature {
		err = errors.New("Above maximum temperature")
		return
	}
	return
}

func (self *Service) queryTarget(q services.Question) services.Answer {
	err, name, target := parseTarget(q.Args)
	if err != nil {
		return services.Answer{Text: fmt.Sprint(err)}
	}
	if zone, ok := self.Zones[name]; ok {
		if zone.Target != target {
			log.Printf("Set %s to target: %v°C", name, target)
		}
		zone.Target = target
	} else {
		return services.Answer{Text: "Invalid zone"}
	}
	return services.Answer{} // no response
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
