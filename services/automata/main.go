// Service for state machine based automation of behaviour. A whole variety of
// complex behaviour can be achieved by linking together triggering events and
// actions.
//
// This is the powerful glue behind the whole gohome system that links the dumb
// input/output services together in smart ways.
//
// Some examples:
//
// - debouncing the door bell
//
// - alert via twitter, sms, etc. when mail arrives
//
// - switch lights on when people get home
//
// - unlocking an electric door lock when an rfid tag is presented
//
// - when the sunsets turn on lights
//
// - a presence based smart burglar alarm system (when the house is empty, turn on the burglar alarm)
//
// The automata are configured via yaml configuration format configured under:
//
// http://localhost:8723/config?path=gohome/config/automata
//
// An example of the configuration is available in the gohome github repository.
//
// For more details on the configuration format, see:
//
// http://godoc.org/github.com/barnybug/gofsm
package automata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/barnybug/gohome/util"

	"github.com/barnybug/gofsm"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

var eventsLogPath = config.LogPath("events.log")

func openLogFile() *os.File {
	logfile, err := os.OpenFile(eventsLogPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Println("Couldn't open events.log:", err)
		logfile, _ = os.Open(os.DevNull)
		return logfile
	}
	return logfile
}

type AutomataService struct {
	timers        map[string]*time.Timer
	automata      *gofsm.Automata
	configUpdated chan bool
	log           *os.File
}

func (self *AutomataService) Id() string {
	return "automata"
}

var reAction = regexp.MustCompile(`(\w+)\((.+)\)`)

type EventAction struct {
	service *AutomataService
	event   *pubsub.Event
	change  gofsm.Change
}

type EventWrapper struct {
	event *pubsub.Event
}

func (self EventWrapper) String() string {
	device := services.Config.LookupDeviceName(self.event)
	s := device
	if self.event.Command() != "" {
		s += fmt.Sprintf(" command=%s", self.event.Command())
	} else if self.event.State() != "" {
		s += fmt.Sprintf(" state=%s", self.event.State())
	}
	return s
}

func (self *AutomataService) ConfigUpdated(path string) {
	if path == "gohome/config/automata" {
		// trigger reload in main loop
		self.configUpdated <- true
	}
}

func (self *AutomataService) RestoreFile(automata *gofsm.Automata) {
	r, err := os.Open(config.ConfigPath("automata.state"))
	if err != nil {
		log.Println("Restoring automata state failed:", err)
		return
	}
	dec := json.NewDecoder(r)
	var p gofsm.AutomataState
	err = dec.Decode(&p)
	if err != nil {
		log.Println("Restoring automata state failed:", err)
		return
	}
	automata.Restore(p)
}

func (self *AutomataService) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": services.TextHandler(self.queryStatus),
		"tag":    services.TextHandler(self.queryTag),
		"switch": services.TextHandler(self.querySwitch),
		"logs":   services.TextHandler(self.queryLogs),
		"help": services.StaticHandler("" +
			"status: get status\n" +
			"tag name: activate tag for name\n" +
			"switch device on|off: switch device\n" +
			"logs: get recent event logs\n"),
	}
}

func (self *AutomataService) queryStatus(q services.Question) string {
	var out string
	now := time.Now()
	var keys []string
	for k := range self.automata.Automaton {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	group := ""
	for _, k := range keys {
		if k == "events" {
			continue
		}
		if strings.Split(k, ".")[0] != group {
			group = strings.Split(k, ".")[0]
			out += group + "\n"
		}
		device := k
		if dev, ok := services.Config.Devices[k]; ok {
			device = dev.Name
		}
		aut := self.automata.Automaton[k]
		du := util.ShortDuration(now.Sub(aut.Since))
		out += fmt.Sprintf("- %s: %s for %s\n", device, aut.State.Name, du)
	}
	return out
}

func (self *AutomataService) queryTag(q services.Question) string {
	tagName := strings.ToLower(q.Args)
	if _, ok := services.Config.Devices["rfid."+tagName]; !ok {
		return fmt.Sprintf("Tag %s not found", tagName)
	}
	fields := map[string]interface{}{
		"source":  tagName,
		"command": "tag",
	}
	ev := pubsub.NewEvent("person", fields)
	services.Publisher.Emit(ev)
	return fmt.Sprintf("Emitted tag for %s", tagName)
}

func (self *AutomataService) querySwitch(q services.Question) string {
	if q.Args == "" {
		// return a list of the devices
		devices := []string{}
		for dev, _ := range services.Config.Devices {
			devices = append(devices, dev)
		}
		sort.Strings(devices)
		return strings.Join(devices, ", ")
	}
	args := strings.Split(q.Args, " ")
	name := args[0]
	state := true
	if len(args) > 1 {
		if args[1] == "off" {
			state = false
		}
	}
	matches := []string{}
	for dev, _ := range services.Config.Devices {
		if strings.Contains(dev, name) {
			matches = append(matches, dev)
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("device %s not found", name)
	}
	if len(matches) > 1 {
		return fmt.Sprintf("device %s is ambiguous", strings.Join(matches, ", "))
	}
	device := matches[0]
	ev := pubsub.NewCommand(device, state, 0)
	services.Publisher.Emit(ev)
	onoff := "off"
	if state {
		onoff = "on"
	}
	return fmt.Sprintf("Switched %s %s", matches[0], onoff)
}

func tail(filename string, lines int64) ([]byte, error) {
	n := fmt.Sprintf("-n%d", lines)
	return exec.Command("tail", n, filename).Output()
}

func (self *AutomataService) queryLogs(q services.Question) string {
	data, err := tail(eventsLogPath, 25)
	if err != nil {
		return fmt.Sprintf("Couldn't retrieve logs:", err)
	}
	return string(data)
}

func (self *AutomataService) PersistFile(automata *gofsm.Automata) {
	w, err := os.Create(config.ConfigPath("automata.state"))
	if err != nil {
		log.Fatalln("Persisting automata state failed:", err)
	}
	defer w.Close()
	enc := json.NewEncoder(w)
	enc.Encode(automata.Persist())
}

func (self *AutomataService) PersistStore(automata *gofsm.Automata, automaton string) {
	state := automata.Persist()
	v := state[automaton]
	key := "gohome/state/automata/" + automaton
	value, _ := json.Marshal(v)
	err := services.Stor.Set(key, string(value))
	if err != nil {
		log.Println("Persisting automata state to store failed:", err)
	}
}

func (self *AutomataService) RestoreStore(automata *gofsm.Automata) {
	p := gofsm.AutomataState{}
	for name := range automata.Automaton {
		key := "gohome/state/automata/" + name
		value, err := services.Stor.Get(key)
		var ps gofsm.AutomatonState
		if err == nil {
			err = json.Unmarshal([]byte(value), &ps)
		}
		if err != nil {
			log.Println("Restoring automata state from store failed:", err)
			continue
		}
		p[name] = ps
	}
	automata.Restore(p)
}

func loadAutomata() (*gofsm.Automata, error) {
	data, err := services.Stor.Get("gohome/config/automata")
	if err != nil {
		return nil, err
	}
	tmpl := template.New("Automata")
	tmpl, err = tmpl.Parse(data)
	if err != nil {
		return nil, err
	}
	context := map[string]interface{}{
		"devices": services.Config.Devices,
	}

	wr := new(bytes.Buffer)
	err = tmpl.Execute(wr, context)
	if err != nil {
		return nil, err
	}
	generated := wr.Bytes()
	automata, err := gofsm.Load(generated)
	if err != nil {
		return nil, err
	}
	return automata, nil
}

func timeit(name string, fn func()) {
	t1 := time.Now()
	fn()
	t2 := time.Now()
	log.Printf("%s took: %s", name, t2.Sub(t1))
}

func (self *AutomataService) Run() error {
	self.log = openLogFile()
	self.timers = map[string]*time.Timer{}
	self.configUpdated = make(chan bool, 2)
	// load templated automata
	var err error
	self.automata, err = loadAutomata()
	if err != nil {
		return err
	}

	// persistance can take a while and delay the workflow, so run in background
	chanPersist := make(chan string, 32)
	go func() {
		for automaton := range chanPersist {
			self.PersistStore(self.automata, automaton)
		}
	}()

	self.RestoreStore(self.automata)
	log.Printf("Initial states: %s", self.automata)

	ch := services.Subscriber.Channel()
	for {
		select {
		case ev := <-ch:
			if ev.Topic == "alert" || ev.Topic == "state" || ev.Topic == "heating" || strings.HasPrefix(ev.Topic, "_") {
				continue
			}
			if ev.Command() == "" && ev.State() == "" {
				continue
			}

			// send relevant events to the automata
			event := EventWrapper{ev}
			self.automata.Process(event)

		case change := <-self.automata.Changes:
			trigger := change.Trigger.(EventWrapper)
			s := fmt.Sprintf("%-17s %s->%s", "["+change.Automaton+"]", change.Old, change.New)
			log.Printf("%-40s (event: %s)", s, trigger)
			chanPersist <- change.Automaton
			if !strings.Contains(change.Automaton, ".") {
				continue
			}
			// emit event
			ps := strings.Split(change.Automaton, ".")
			topic := ps[0]
			source := ps[1]
			fields := pubsub.Fields{
				"source":  source,
				"state":   change.New,
				"trigger": trigger.String(),
			}
			ev := pubsub.NewEvent(topic, fields)
			services.Publisher.Emit(ev)

		case action := <-self.automata.Actions:
			wrapper := action.Trigger.(EventWrapper)
			ea := EventAction{self, wrapper.event, action.Change}
			err := DynamicCall(ea, action.Name)
			if err != nil {
				log.Println("Error:", err)
			}
		case <-self.configUpdated:
			// live reload the automata!
			log.Println("Automata config updated, reloading")
			updated, err := loadAutomata()
			if err != nil {
				log.Println("Failed to reload automata:", err)
				continue
			}
			self.RestoreStore(updated)
			self.automata = updated
			log.Println("Automata reloaded successfully")
		}
	}
	return nil
}

func (self *AutomataService) appendLog(msg string) {
	fmt.Fprintln(self.log, msg)
}

func (self EventAction) substitute(msg string) string {
	device := services.Config.LookupDeviceName(self.event)
	name := services.Config.Devices[device].Name
	now := time.Now()
	vals := map[string]string{
		"name":      name,
		"duration":  util.FriendlyDuration(self.change.Duration),
		"timestamp": now.Format(time.Kitchen),
		"datetime":  now.Format(time.StampMilli),
	}

	return Substitute(msg, vals)
}

func (self EventAction) Log(msg string) {
	msg = self.substitute("$datetime: " + msg)
	self.service.appendLog(msg)
}

func (self EventAction) Speak(msg string) {
	self.Alert(msg, "espeak")
}

func (self EventAction) Video(device string, preset int64, secs float64, ir bool) {
	log.Printf("Video: %s at %d for %.1fs (ir: %v)", device, preset, secs, ir)
	fields := pubsub.Fields{
		"device":  device,
		"state":   "video",
		"timeout": secs,
		"preset":  preset,
		"ir":      ir,
	}
	ev := pubsub.NewEvent("command", fields)
	services.Publisher.Emit(ev)
}

func (self EventAction) RingAlarm() {
	command("alarm.bell", true)
}

func (self EventAction) QuietAlarm() {
	command("alarm.bell", false)
}

func (self EventAction) Script(cmd string) {
	log.Println("Script:", cmd)
	// run exec non-blocking
	go func() {
		cmd = util.ExpandUser(cmd)
		_, err := exec.Command(cmd).Output()
		if err != nil {
			log.Printf("Exec %s: %s", cmd, err)
		}
	}()
}

func (self EventAction) Jabber(msg string) {
	self.Alert(msg, "jabber")
}

func (self EventAction) Sms(msg string) {
	self.Alert(msg, "sms")
}

func (self EventAction) Twitter(msg string) {
	self.Alert(msg, "twitter")
}

func (self EventAction) Alert(message string, target string) {
	message = self.substitute(message)
	log.Printf("%s: %s", strings.Title(target), message)
	services.SendAlert(message, target, "", 0)
}

func command(device string, state bool) {
	ev := pubsub.NewCommand(device, state, 0)
	services.Publisher.Emit(ev)
}

func (self EventAction) UnlockDoor() {
	log.Println("Unlocking door")
	command("door.front", true)
}

func (self EventAction) LockDoor() {
	log.Println("Locking door")
	command("door.front", false)
}

func (self EventAction) Switch(device string, state bool) {
	on := "off"
	if state {
		on = "on"
	}
	log.Printf("Switching %s %s", device, on)
	command(device, state)
}

func (self EventAction) StartTimer(name string, d int64) {
	// log.Printf("Starting timer: %s for %ds", name, d)
	duration := time.Duration(d) * time.Second
	if timer, ok := self.service.timers[name]; ok {
		// cancel any existing
		timer.Stop()
	}

	timer := time.AfterFunc(duration, func() {
		// emit timer event
		fields := map[string]interface{}{
			"source":  name,
			"command": "on",
		}
		ev := pubsub.NewEvent("timer", fields)
		services.Publisher.Emit(ev)
	})
	self.service.timers[name] = timer
}
