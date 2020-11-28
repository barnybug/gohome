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
	Name      string
	Id        string
	Timeout   time.Duration
	Problem   bool
	NextAlert time.Time
	LastEvent time.Time
	Silent    bool
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

func sendAlert(message string) {
	log.Printf("Sending watchdog alert: %s\n", message)
	services.SendAlert("💓 "+message, services.Config.Watchdog.Alert, "", 0)
}

func ignoreTopics(topic string) bool {
	return topic == "log" || topic == "rpc" || topic == "query" || topic == "alert" || topic == "command" || topic == "watchdog" || topic == "state" || strings.HasPrefix(topic, "_")
}

func (self *Service) checkEvent(ev *pubsub.Event) {
	if ignoreTopics(ev.Topic) {
		return
	}
	device := ev.Device()
	if device != "" {
		mappedDevice(ev)
		self.touch(device, ev.Timestamp)
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

func (self *Service) touch(device string, timestamp time.Time) {
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
	w.NextAlert = timestamp.Add(w.Timeout)
	if w.NextAlert.Before(now) {
		// must have previously alerted - set to repeat interval
		w.NextAlert = now.Add(repeatInterval)
	}
	// reschedule if:
	// - this was the next scheduled
	// - there's none scheduled (startup)
	// - the next scheduled is after this next alert
	if w == self.nextProblem || self.nextProblem == nil || self.nextProblem.NextAlert.After(w.NextAlert) {
		// update next problem
		self.scheduleNextTimeout()
	}

	// recovered?
	if w.Problem {
		w.Problem = false
		if !w.Silent && !self.problems.Remove(w) {
			// if it was briefly problematic but recovered, then don't alert
			self.recoveries.Add(w)
		}
		sendWatchdogEvent(device, "on")
	}
}

func sendWatchdogEvent(device, command string) {
	fields := pubsub.Fields{
		"device":  device,
		"command": command,
	}
	ev := pubsub.NewEvent("watchdog", fields)
	services.Publisher.Emit(ev)
}

func (self *Service) scheduleNextTimeout() {
	next := time.Time{}
	now := time.Now()
	self.nextProblem = nil
	for _, w := range watches {
		if w.NextAlert.IsZero() {
			continue
		}
		if w.NextAlert.Before(next) || next.IsZero() {
			next = w.NextAlert
			self.nextProblem = w
		}
	}

	if !next.IsZero() {
		duration := next.Sub(now)
		// log.Printf("Next scheduled: %s in %s at %s", self.nextProblem.Name, util.ShortDuration(duration), next)
		if duration < 0 {
			duration = 0
		}
		self.timeout.Reset(duration)
	} else {
		self.timeout.Stop()
	}
}

func (self *Service) checkTimeouts() {
	w := self.nextProblem
	if !w.Problem {
		sendWatchdogEvent(w.Id, "off")
		w.Problem = true
	}
	if !w.Silent {
		self.recoveries.Remove(w)
		self.problems.Add(w)
	}
	w.NextAlert = w.NextAlert.Add(repeatInterval)
	self.scheduleNextTimeout()
}

func listOfLots(ss []string, limit int) string {
	if len(ss) > limit {
		return fmt.Sprintf("%s and %d others", ss[0], len(ss)-1)
	}
	return strings.Join(ss, ", ")
}

// Service watchdog
type Service struct {
	pinger      *fastping.Pinger
	problems    *Alerter
	recoveries  *Alerter
	timeout     *time.Timer
	nextProblem *Watch
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
			v.Problem = o.Problem
			v.LastEvent = o.LastEvent
			v.NextAlert = o.NextAlert
		}
	}
	self.scheduleNextTimeout()
}

func (self *Service) setupDevices() {
	now := time.Now()
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
			NextAlert: now.Add(d.Watchdog.Duration),
			Silent:    d.Cap["silent"],
		}
	}
}

func (self *Service) setupHeartbeats() {
	nextAlert := time.Now().Add(time.Second * 241)
	// monitor gohome processes heartbeats
	for _, process := range services.Config.Watchdog.Processes {
		id := fmt.Sprintf("heartbeat.%s", process)
		// if a process misses 2 heartbeats, mark as problem
		watches[id] = &Watch{
			Id:        id,
			Name:      process,
			Timeout:   time.Second * 241,
			NextAlert: nextAlert,
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
	self.problems = NewAlerter("PROBLEM")
	self.recoveries = NewAlerter("RECOVERED")
	self.timeout = time.NewTimer(time.Hour)
	self.setup()
	return nil
}

type Alerter struct {
	Timer   *time.Timer
	watches map[*Watch]bool
	delayed bool
	suffix  string
}

func NewAlerter(suffix string) *Alerter {
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	return &Alerter{watches: map[*Watch]bool{}, suffix: suffix, Timer: timer}
}

func (self *Alerter) Add(watch *Watch) {
	self.watches[watch] = true
	if !self.delayed {
		// send this now, but delay any further
		self.sendAlert()
		self.delayed = true
		self.Timer.Reset(2 * time.Minute)
	}
}

func (self *Alerter) sendAlert() {
	// send (batched) notification
	var names []string
	var lastWatch *Watch
	for watch := range self.watches {
		names = append(names, watch.Name)
		lastWatch = watch
	}
	message := fmt.Sprintf("%s %s", listOfLots(names, 10), self.suffix)
	if self.suffix == "PROBLEM" && len(names) == 1 {
		duration := time.Now().Sub(lastWatch.LastEvent)
		message += " for " + util.FriendlyDuration(duration)
	}
	sendAlert(message)
	// clear out
	self.watches = map[*Watch]bool{}
}

func (self *Alerter) Remove(watch *Watch) bool {
	if !self.watches[watch] {
		return false
	}
	delete(self.watches, watch)
	return true
}

func (self *Alerter) TimerCallback() {
	if len(self.watches) > 0 {
		self.sendAlert()
	} else {
		// cooled down
		self.delayed = false
	}
}

func (self *Service) Run() error {
	events := services.Subscriber.Channel()
	for {
		select {
		case ev := <-events:
			self.checkEvent(ev)
		case <-self.timeout.C:
			self.checkTimeouts()
		case <-self.problems.Timer.C:
			self.problems.TimerCallback()
		case <-self.recoveries.Timer.C:
			self.recoveries.TimerCallback()
		}
	}
}
