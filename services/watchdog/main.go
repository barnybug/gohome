// Service for monitoring devices to ensure they're still alive and emitting
// events. Watches a given list of device ids, and alerts if an event has not
// been seen from a device in a configurable time period.
package watchdog

import (
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
	fastping "github.com/tatsushid/go-fastping"
)

type Watch struct {
	Name        string
	Id          string
	Timeout     time.Duration
	Problem     bool
	Recover     bool
	LastAlerted time.Time
	LastEvent   time.Time
	Silent      bool
}

type Watches []*Watch

func (self Watches) Less(i, j int) bool {
	a := self[i]
	b := self[j]
	if a.Problem != b.Problem {
		// sort i first if Problematic
		return a.Problem
	}
	// tied, sort by last event
	return a.LastEvent.Before(b.LastEvent)
}

func (self Watches) Len() int {
	return len(self)
}

func (self Watches) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

var watches = map[string]*Watch{}
var unmapped = map[string]bool{}
var repeatInterval, _ = time.ParseDuration("12h")
var alerts = time.NewTimer(time.Second)

func sendAlert(message string) {
	log.Printf("Sending watchdog alert: %s\n", message)
	services.SendAlert("💓 "+message, services.Config.Watchdog.Alert, "", 0)
}

func sendRecoveries() {
	names := []string{}
	for _, w := range watches {
		if w.Recover {
			if !w.Silent {
				names = append(names, w.Name)
			}
			w.Recover = false
		}
	}

	if len(names) > 0 {
		// send a single notification
		message := fmt.Sprintf("%s RECOVERED", listOfLots(names, 10))
		sendAlert(message)
		// delay multiple recovered messages
		alerts.Reset(120 * time.Second)
	} else {
		alerts.Reset(5 * time.Second)
	}
}

func ignoreTopics(topic string) bool {
	return topic == "log" || topic == "rpc" || topic == "query" || topic == "alert" || topic == "command" || topic == "watchdog" || topic == "state" || strings.HasPrefix(topic, "_")
}

func checkEvent(ev *pubsub.Event) {
	if ignoreTopics(ev.Topic) {
		return
	}
	device := ev.Device()
	if device != "" {
		mappedDevice(ev)
		touch(device, ev.Timestamp)
	} else if ev.Source() != "" {
		unmappedDevice(ev)
	}
}

func mappedDevice(ev *pubsub.Event) {
	if _, ok := unmapped[ev.Source()]; ok {
		delete(unmapped, ev.Source())
		announce(ev, true)
	}
}

func unmappedDevice(ev *pubsub.Event) {
	if _, ok := services.Config.LookupSource(ev.Source()); ok {
		// ignored
		return
	}
	if _, ok := unmapped[ev.Source()]; ok {
		// already alerted
		return
	}
	unmapped[ev.Source()] = true
	announce(ev, false)
}

func guessDeviceName(topic string) string {
	switch topic {
	case "temp":
		return "Thermometer"
	case "light":
		return "Light"
	case "chime":
		return "Chime"
	case "door":
		return "Door sensor"
	case "pir":
		return "PIR motion detector"
	case "power":
		return "Power meter"
	case "pressure":
		return "Barometer"
	case "rain":
		return "Rain meter"
	case "sensor":
		return "Sensor"
	case "soil":
		return "Soil moisture"
	case "ups":
		return "UPS"
	case "voltage":
		return "Battery voltage"
	case "wind":
		return "Wind meter"
	default:
		return "New device"
	}
}

func announce(ev *pubsub.Event, mapped bool) {
	source := ev.Source()
	if mapped {
		log.Printf("Announcing mapped device %s\n", source)
	} else {
		log.Printf("Announcing new device %s\n", source)
	}
	var message string
	if mapped {
		dev := services.Config.Devices[ev.Device()]
		message = fmt.Sprintf("✔️ Device configured: '%s'", dev.Name)
	} else {
		// some discovered devices have friendly names (eg tradfri)
		name := ev.StringField("name")
		if name == "" {
			name = guessDeviceName(ev.Topic)
		}
		message = fmt.Sprintf("🔎 Discovered: '%s' id: '%s' emitting '%s' events", name, source, ev.Topic)
	}
	services.SendAlert(message, services.Config.Watchdog.Alert, "", 0)
}

func touch(device string, timestamp time.Time) {
	// check if in devices monitored
	w := watches[device]
	if w == nil {
		return
	}

	// discard if timestamp is over a year, or event not latest by timestamp
	now := time.Now()
	age := now.Sub(timestamp)
	if age.Hours() > 24*365 || timestamp.Before(w.LastEvent) {
		return
	}
	w.LastEvent = timestamp

	// recovered?
	if w.Problem {
		w.Problem = false
		w.Recover = true // picked up by next sendRecoveries()

		// send watchdog event
		fields := pubsub.Fields{
			"device":  device,
			"command": "on",
		}
		ev := pubsub.NewEvent("watchdog", fields)
		services.Publisher.Emit(ev)
	}
}

func checkTimeouts() {
	timeouts := []string{}
	var lastEvent time.Time
	for _, w := range watches {
		if w.Problem {
			// check if should repeat
			if time.Since(w.LastAlerted) > repeatInterval {
				if !w.Silent {
					timeouts = append(timeouts, w.Name)
				}
				lastEvent = w.LastEvent
				w.LastAlerted = time.Now()
			}
		} else if time.Since(w.LastEvent) > w.Timeout {
			// first alert
			if !w.Silent {
				timeouts = append(timeouts, w.Name)
			}
			lastEvent = w.LastEvent
			w.Problem = true
			w.LastAlerted = time.Now()

			// send watchdog event
			fields := pubsub.Fields{
				"device":  w.Id,
				"command": "off",
			}
			ev := pubsub.NewEvent("watchdog", fields)
			services.Publisher.Emit(ev)
		}
	}

	if len(timeouts) > 0 {
		// send a single notification
		message := fmt.Sprintf("%s PROBLEM", listOfLots(timeouts, 10))
		if len(timeouts) == 1 {
			now := time.Now()
			message += " for " + util.FriendlyDuration(now.Sub(lastEvent))
		}
		sendAlert(message)
	}
}

func listOfLots(ss []string, limit int) string {
	if len(ss) > limit {
		return fmt.Sprintf("%s and %d others", ss[0], len(ss)-1)
	}
	return strings.Join(ss, ", ")
}

// Service watchdog
type Service struct {
	pinger *fastping.Pinger
}

func (self *Service) ID() string {
	return "watchdog"
}

func (self *Service) ConfigUpdated(path string) {
	if path != "config" {
		return
	}
	self.setup()
}

func (self *Service) setup() {
	previous := watches
	watches = map[string]*Watch{}
	self.setupDevices()
	self.setupHeartbeats()
	self.setupPings()
	// preserve last event when reloading config
	for k, v := range watches {
		if o, ok := previous[k]; ok {
			v.LastEvent = o.LastEvent
		}
	}
}

func (self *Service) setupDevices() {
	for _, d := range services.Config.Devices {
		if d.Watchdog.IsZero() {
			continue
		}
		// give devices grace period for first event
		name := d.Name
		if name == "" {
			// attempt a reasonable device name if missing from config
			name = strings.Title(strings.Replace(d.Id, ".", " ", -1))
		}
		watches[d.Id] = &Watch{
			Id:        d.Id,
			Name:      name,
			Timeout:   d.Watchdog.Duration,
			LastEvent: time.Time{},
			Silent:    d.Cap["silent"],
		}
	}
}

func (self *Service) setupHeartbeats() {
	// monitor gohome processes heartbeats
	for _, process := range services.Config.Watchdog.Processes {
		id := fmt.Sprintf("heartbeat.%s", process)
		// if a process misses 2 heartbeats, mark as problem
		watches[id] = &Watch{
			Id:        id,
			Name:      process,
			Timeout:   time.Second * 241,
			LastEvent: time.Time{},
		}
	}
}

func (self *Service) setupPings() {
	if self.pinger != nil {
		// reconfiguring - stop previous pinger
		self.pinger.Stop()
	}

	// create and run pinger
	p := fastping.NewPinger()
	p.Network("udp") // use unprivileged
	p.MaxRTT = 20 * time.Second
	lookup := map[string]string{}
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		device := lookup[addr.String()]
		fields := pubsub.Fields{
			"device":  device,
			"command": "on",
		}
		ev := pubsub.NewEvent("ping", fields)
		services.Publisher.Emit(ev)
	}
	for _, dev := range services.Config.DevicesByProtocol("ping") {
		host := dev.SourceId()
		addr, err := net.ResolveIPAddr("ip4:icmp", host)
		if err != nil {
			log.Printf("Failed to resolve host - delaying ping: %s", err)
		} else {
			log.Printf("Resolved %s to %s", host, addr)
			lookup[addr.String()] = dev.Id
			p.AddIPAddr(addr)
		}
	}
	p.RunLoop()
	self.pinger = p
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status":     services.TextHandler(self.queryStatus),
		"discovered": services.TextHandler(self.queryDiscovered),
		"help":       services.StaticHandler("status: get status\n"),
	}
}

func (self *Service) queryStatus(q services.Question) string {
	var out string

	// build list
	var list Watches
	for _, watch := range watches {
		list = append(list, watch)
	}

	// return oldest last
	sort.Sort(sort.Reverse(list))

	now := time.Now()
	for _, w := range list {
		symbol := "✔️"
		if w.Problem {
			symbol = "✖️️"
		}
		var ago string
		if w.LastEvent.IsZero() {
			ago = "never"
		} else {
			ago = util.ShortDuration(now.Sub(w.LastEvent))
		}
		out += fmt.Sprintf("%s %-8s %-20s %s\n", symbol, ago, w.Id, w.Name)
	}
	return out
}

func (self *Service) queryDiscovered(q services.Question) string {
	out := fmt.Sprintf("%d new devices discovered\n", len(unmapped))
	for source := range unmapped {
		out += fmt.Sprintf("%s\n", source)
	}
	return out
}

func (self *Service) Init() error {
	self.setup()
	return nil
}

func (self *Service) Run() error {
	ticker := time.NewTicker(time.Minute)
	events := services.Subscriber.Channel()
	for {
		select {
		case ev := <-events:
			checkEvent(ev)
		case <-ticker.C:
			checkTimeouts()
		case <-alerts.C:
			sendRecoveries()
		}
	}
}
