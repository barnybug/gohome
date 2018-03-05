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

type WatchdogDevice struct {
	Name        string
	Type        string
	Timeout     time.Duration
	Alerted     bool
	LastAlerted time.Time
	LastEvent   time.Time
}

type WatchdogDevices []*WatchdogDevice

func (self WatchdogDevices) Less(i, j int) bool {
	return self[i].LastEvent.Before(self[j].LastEvent)
}

func (self WatchdogDevices) Len() int {
	return len(self)
}

func (self WatchdogDevices) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

var devices = map[string]*WatchdogDevice{}
var unmapped = map[string]bool{}
var repeatInterval, _ = time.ParseDuration("12h")

func sendAlert(name string, state bool, since time.Time) {
	log.Printf("Sending %t watchdog alert for: %s\n", state, name)
	var message string
	if state {
		message = fmt.Sprintf("💓 %s RECOVERED", name)
	} else {
		now := time.Now()
		message = fmt.Sprintf("💓 %s PROBLEM for %s", name, util.FriendlyDuration(now.Sub(since)))
	}
	services.SendAlert(message, services.Config.Watchdog.Alert, "", 0)
}

func ignoreTopics(topic string) bool {
	return topic == "log" || topic == "rpc" || topic == "query" || strings.HasPrefix(topic, "_")
}

func checkEvent(ev *pubsub.Event) {
	device := ev.Device()
	if device != "" {
		mappedDevice(ev)
		touch(device, ev.Timestamp)
	} else if ev.Source() != "" && !ignoreTopics(ev.Topic) {
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
		message = fmt.Sprintf("✔️ Device configured %s", dev.Name)
	} else {
		// some discovered devices have friendly names (eg tradfri)
		name := ev.StringField("name")
		if name == "" {
			name = "New device"
		}
		message = fmt.Sprintf("🔎 %s found %s emitting %s events", name, source, ev.Topic)
	}
	services.SendAlert(message, services.Config.Watchdog.Alert, "", 0)
}

func touch(device string, timestamp time.Time) {
	// check if in devices monitored
	w := devices[device]
	if w == nil {
		return
	}

	// recovered?
	if w.Alerted {
		w.Alerted = false
		sendAlert(w.Name, true, w.LastEvent)
	}
	// if timestamp looks too old, use now instead
	now := time.Now()
	age := now.Sub(timestamp)
	if age.Hours() > 1 {
		timestamp = now
	}
	w.LastEvent = timestamp
}

func processPing(host string) {
	device := fmt.Sprintf("ping.%s", host)
	touch(device, time.Now())
}

func checkTimeouts() {
	timeouts := []string{}
	var lastEvent time.Time
	for _, w := range devices {
		if w.Alerted {
			// check if should repeat
			if time.Since(w.LastAlerted) > repeatInterval {
				timeouts = append(timeouts, w.Name)
				lastEvent = w.LastEvent
				w.LastAlerted = time.Now()
			}
		} else if time.Since(w.LastEvent) > w.Timeout {
			// first alert
			timeouts = append(timeouts, w.Name)
			lastEvent = w.LastEvent
			w.Alerted = true
			w.LastAlerted = time.Now()
		}
	}

	// send a single email for multiple devices
	if len(timeouts) > 0 {
		sendAlert(strings.Join(timeouts, ", "), false, lastEvent)
	}
}

// Service watchdog
type Service struct {
	pings  chan string
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
	now := time.Now()
	self.setupDevices(now)
	self.setupHeartbeats(now)
	self.setupPings(now)
}

func (self *Service) setupDevices(now time.Time) {
	for device, timeout := range services.Config.Watchdog.Devices {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			fmt.Println("Failed to parse:", timeout)
		}
		// give devices grace period for first event
		d := services.Config.Devices[device]
		devices[device] = &WatchdogDevice{
			Name:      fmt.Sprintf("%s (%s)", d.Name, d.Type),
			Type:      "device",
			Timeout:   duration,
			LastEvent: now,
		}
	}
}

func (self *Service) setupHeartbeats(now time.Time) {
	// monitor gohome processes heartbeats
	for _, process := range services.Config.Watchdog.Processes {
		device := fmt.Sprintf("heartbeat.%s", process)
		// if a process misses 2 heartbeats, mark as problem
		devices[device] = &WatchdogDevice{
			Name:      process,
			Type:      "process",
			Timeout:   time.Second * 241,
			LastEvent: now,
		}
	}
}

func (self *Service) setupPings(now time.Time) {
	if self.pinger != nil {
		// reconfiguring - stop previous pinger
		self.pinger.Stop()
	}

	for _, host := range services.Config.Watchdog.Pings {
		device := fmt.Sprintf("ping.%s", host)
		devices[device] = &WatchdogDevice{
			Name:      host,
			Type:      "ping",
			Timeout:   time.Second * 301,
			LastEvent: now,
		}
	}

	// create and run pinger
	p := fastping.NewPinger()
	p.Network("udp") // use unprivileged
	p.MaxRTT = 20 * time.Second
	lookup := map[string]string{}
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		host := lookup[addr.String()]
		self.pings <- host
	}
	for _, host := range services.Config.Watchdog.Pings {
		addr, err := net.ResolveIPAddr("ip4:icmp", host)
		if err != nil {
			log.Printf("Failed to resolve host - delaying ping: %s", err)
		} else {
			log.Printf("Resolved %s to %s", host, addr)
			lookup[addr.String()] = host
			p.AddIPAddr(addr)
		}
	}
	p.RunLoop()
	self.pinger = p
}

func (self *Service) Run() error {
	self.pings = make(chan string, 10)
	self.setup()
	ticker := time.NewTicker(time.Minute)
	events := services.Subscriber.Channel()
	for {
		select {
		case ev := <-events:
			checkEvent(ev)
		case <-ticker.C:
			checkTimeouts()
		case ping := <-self.pings:
			processPing(ping)
		}
	}
	return nil
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
	var list WatchdogDevices
	for _, device := range devices {
		list = append(list, device)
	}
	// return oldest last
	sort.Sort(sort.Reverse(list))

	now := time.Now()
	for _, w := range list {
		problem := ""
		if w.Alerted {
			problem = "PROBLEM"
		}
		ago := util.ShortDuration(now.Sub(w.LastEvent))
		out += fmt.Sprintf("- %-6s %7s %s %s\n", ago, w.Type, w.Name, problem)
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
