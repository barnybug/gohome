// Service to thermostatically control central heating by schedule and zone.
// Supports multiple temperature points on a daily schedule, temporary
// override ('party mode'), hibernation when the house is empty.
package heating

import (
	"encoding/json"
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
	Thermostat string    `json:"thermostat"`
	Temp       float64   `json:"temp"`
	Target     float64   `json:"target"`
	Rate       float64   `json:"rate"`
	At         time.Time `json:"at"`
	PartyTemp  float64   `json:"party_temp"`
	PartyUntil time.Time `json:"party_until"`
	Sensor     string    `json:"sensor"`
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

	// emit event to retained heating topic
	fields := pubsub.Fields{
		"device":  "heating",
		"heating": self.State,
		"status":  self.Json(Clock()),
	}
	ev := pubsub.NewEvent("heating", fields)
	ev.SetRetained(true)
	self.Publisher.Emit(ev)

	if self.HeatingDevice != "auto" {
		// repeat control command
		self.Command()
	}
}

func (self *Service) restoreState(ev *pubsub.Event) {
	// marshal into zones
	zones := map[string]*Zone{}
	if state, ok := ev.Fields["status"].(map[string]interface{}); ok {
		zoneJson, _ := json.Marshal(state["zones"])
		err := json.Unmarshal(zoneJson, &zones)
		if err != nil {
			log.Printf("Failed to unmarshal state: %s", err)
			return
		}

		for zone, z := range self.Zones {
			if old, ok := zones[zone]; ok {
				z.Temp = old.Temp
				z.Target = old.Target
				z.Rate = old.Rate
				z.At = old.At
				z.PartyTemp = old.PartyTemp
				z.PartyUntil = old.PartyUntil
				log.Printf("Restored zone '%s' temp: %v target: %v", zone, z.Temp, z.Target)
			}
		}
	}
}

func (self *Service) handleEvent(ev *pubsub.Event) {
	switch ev.Topic {
	case "heating":
		if ev.Retained {
			// restore heating state on restart
			self.restoreState(ev)
		}

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
			}
			if now.Before(zone.PartyUntil) {
				fields["boost"] = zone.PartyUntil.Sub(now).Seconds()
			}
			ev := pubsub.NewEvent("command", fields)
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
	zones := map[string]interface{}{}
	zoneJson, _ := json.Marshal(self.Zones)
	json.Unmarshal(zoneJson, &zones)
	data["zones"] = zones
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
	events := services.Subscriber.Subscribe(pubsub.Prefix("temp"), pubsub.Prefix("heating"))
	for {
		select {
		case ev := <-events:
			self.handleEvent(ev)
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
			self.Check(true)
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
