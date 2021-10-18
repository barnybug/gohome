// Service for monitoring devices to ensure they're still alive and emitting
// events. Watches a given list of device ids, and alerts if an event has not
// been seen from a device in a configurable time period.
package watchdog

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"regexp"
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
	first := w.LastEvent.IsZero()
	recovered := w.Problem
	w.LastEvent = timestamp
	w.NextAlert = timestamp.Add(w.Timeout)
	if w.NextAlert.Before(now) {
		// event too old
		recovered = false
		// should have previously alerted - set to repeat interval
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
	if recovered {
		w.Problem = false
		if !w.Silent && !self.problems.Remove(w) && !first {
			// if it was briefly problematic but recovered, then don't alert
			self.recoveries.Add(w)
		}
		sendWatchdogEvent(device, "online")
	}
}

func sendWatchdogEvent(device, status string) {
	fields := pubsub.Fields{
		"device": device,
		"status": status,
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
		sendWatchdogEvent(w.Id, "offline")
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
	sniffer     *ArpSniffer
	problems    *Alerter
	recoveries  *Alerter
	timeout     *time.Timer
	nextProblem *Watch
	pings       chan string
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
			Problem:   true,
		}
	}
}

func (self *Service) setupHeartbeats() {
	nextAlert := time.Now().Add(time.Second * 241)
	// monitor gohome processes heartbeats
	for _, process := range services.Config.Watchdog.Processes {
		id := fmt.Sprintf("heartbeat.%s", process)
		name := fmt.Sprintf("%s service", process)
		// if a process misses 2 heartbeats, mark as problem
		watches[id] = &Watch{
			Id:        id,
			Name:      name,
			Timeout:   time.Second * 241,
			NextAlert: nextAlert,
			Problem:   true,
		}
	}
}

type ArpSniffer struct {
	cmd    *exec.Cmd
	OnRecv func(ip string)
}

func NewArpSniffer() *ArpSniffer {
	return &ArpSniffer{}
}

// 06:29:33.293180 ARP, Request who-has 192.168.10.254 (ff:ff:ff:ff:ff:ff) tell 192.168.10.241, length 46
var reTell = regexp.MustCompile(`tell ([0-9.]+)`)

func (a *ArpSniffer) Start() {
	// setcap CAP_NET_RAW=ep /usr/bin/tcpdump
	a.cmd = exec.Command("tcpdump", "-p", "-n", "-l", "arp", "and", "arp[6:2] == 1")
	stdout, err := a.cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to start tcpdump: %s", err)
		return
	}
	stderr, err := a.cmd.StderrPipe()
	if err := a.cmd.Start(); err != nil {
		log.Printf("Failed to start tcpdump: %s", err)
		return
	}
	// buffer stderr
	var stderrBuf bytes.Buffer
	go io.Copy(&stderrBuf, stderr)

	log.Printf("Sniffing for arp requests")
	// read stdout by line, parse for "tell <ip>"
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		match := reTell.FindStringSubmatch(scanner.Text())
		if match == nil {
			log.Printf("ArpSniffer: no match %s", scanner.Text())
			continue
		}
		ip := match[1]
		// log.Printf("ArpSniffer: %s", ip)
		a.OnRecv(ip)
	}
	if a.cmd.ProcessState.ExitCode() != 0 {
		log.Printf("ArpSniffer: tcpdump failed: %s", string(stderrBuf.Bytes()))
		return
	}
	if err := scanner.Err(); err != nil {
		log.Printf("ArpSniffer: tcpdump failed: %s", err)
		return
	}
}

func (self *Service) setupPings() {
	// This pings the list of hosts every 20s to verify if they're connected, and also
	// snoops on arp requests from hosts to detect more rapidly when they initially join the network.
	if self.pings != nil {
		close(self.pings)
	}

	lookup := map[string]string{}
	last := map[string]time.Time{}
	self.pings = make(chan string)
	go func() {
		for ip := range self.pings {
			device := lookup[ip]
			if device == "" {
				continue // filter unknown ips
			}
			if last[ip].After(time.Now().Add(-20 * time.Second)) {
				continue // suppress any more regularly than 20s
			}
			last[ip] = time.Now()
			fields := pubsub.Fields{
				"device":  device,
				"command": "on",
			}
			ev := pubsub.NewEvent("ping", fields)
			ev.SetRetained(true)
			services.Publisher.Emit(ev)
		}
	}()

	if self.pinger != nil {
		// reconfiguring - stop previous pinger
		self.pinger.Stop()
	}
	if self.sniffer == nil {
		self.sniffer = NewArpSniffer()
		go self.sniffer.Start()
	}
	self.sniffer.OnRecv = func(ip string) {
		self.pings <- ip
	}

	// create and run pinger
	p := fastping.NewPinger()
	self.pinger = p
	p.Network("udp") // use unprivileged
	p.MaxRTT = 20 * time.Second
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		self.pings <- addr.String()
	}

	// resolve host names -> devices
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

	// run pinger loop
	p.RunLoop()
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
	if self.suffix == "PROBLEM" && len(names) == 1 && !lastWatch.LastEvent.IsZero() {
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
