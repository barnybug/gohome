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
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

// Service automata
type Service struct {
	timers        map[string]*time.Timer
	automata      *gofsm.Automata
	configUpdated chan bool
	log           *os.File
}

// ID of the service
func (self *Service) ID() string {
	return "automata"
}

var reAction = regexp.MustCompile(`(\w+)\((.+)\)`)

type EventAction struct {
	service *Service
	event   *pubsub.Event
	change  gofsm.Change
}

type EventWrapper struct {
	event *pubsub.Event
}

type Expression interface {
	Match(event *pubsub.Event) bool
}

type Or struct {
	terms []Expression
}

func (self *Or) Match(event *pubsub.Event) bool {
	for _, term := range self.terms {
		if term.Match(event) {
			return true
		}
	}
	return false
}

type And struct {
	terms []Expression
}

func (self *And) Match(event *pubsub.Event) bool {
	for _, term := range self.terms {
		if !term.Match(event) {
			return false
		}
	}
	return true
}

type Equals struct {
	field string
	value string
}

func (self *Equals) Match(event *pubsub.Event) bool {
	matched, _ := filepath.Match(self.value, event.StringField(self.field))
	return matched
}

func ParseEqual(s string) Expression {
	if strings.Contains(s, "=") {
		ps := strings.SplitN(s, "=", 2)
		return &Equals{ps[0], ps[1]}
	} else {
		// default equals on device
		return &Equals{"device", s}
	}
}

func ParseAnd(s string) Expression {
	ps := strings.Split(s, " ")
	terms := []Expression{}
	for _, cond := range ps {
		terms = append(terms, ParseEqual(cond))
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return &And{terms}
}

func Parse(s string) Expression {
	ps := strings.Split(s, " or ")
	terms := []Expression{}
	for _, cond := range ps {
		terms = append(terms, ParseAnd(cond))
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return &Or{terms}
}

var parsingCache = map[string]Expression{}

func ParseCached(s string) Expression {
	if expr, ok := parsingCache[s]; ok {
		return expr
	}
	expr := Parse(s)
	parsingCache[s] = expr
	return expr
}

func (self EventWrapper) Match(when string) bool {
	expr := ParseCached(when)
	return expr.Match(self.event)
}

func (self EventWrapper) String() string {
	s := self.event.Device()
	for k, v := range self.event.Fields {
		if k == "device" {
			continue
		}
		s += fmt.Sprintf(" %s=%v", k, v)
	}
	return s
}

func (self *Service) ConfigUpdated(path string) {
	// trigger reload in main loop
	self.configUpdated <- true
}

func (self *Service) RestoreFile(automata *gofsm.Automata) {
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

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": services.TextHandler(self.queryStatus),
		"tag":    services.TextHandler(self.queryTag),
		"switch": services.TextHandler(self.querySwitch),
		"logs":   services.TextHandler(self.queryLogs),
		"script": services.TextHandler(self.queryScript),
		"help": services.StaticHandler("" +
			"status: get status\n" +
			"tag name: activate tag for name\n" +
			"switch device on|off: switch device\n" +
			"logs: get recent event logs\n" +
			"script: run a script\n"),
	}
}

func (self *Service) queryStatus(q services.Question) string {
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

func (self *Service) queryTag(q services.Question) string {
	tagName := strings.ToLower(q.Args)
	device := fmt.Sprintf("person.%s", tagName)
	if _, ok := services.Config.Devices[device]; !ok {
		return fmt.Sprintf("Tag %s not found", tagName)
	}
	fields := pubsub.Fields{
		"device":  device,
		"command": "on",
		"source":  tagName,
	}
	ev := pubsub.NewEvent("rfid", fields)
	services.Publisher.Emit(ev)
	return fmt.Sprintf("Emitted tag for %s", tagName)
}

func keywordArgs(args []string) map[string]string {
	ret := map[string]string{}
	for _, arg := range args {
		p := strings.SplitN(arg, "=", 2)
		if len(p) == 2 {
			ret[p[0]] = p[1]
		} else {
			ret[""] = p[0]
		}
	}
	return ret
}

var switchable = map[string]bool{
	"bell":   true,
	"dimmer": true,
	"lock":   true,
	"person": true,
	"switch": true,
}

func matchDevices(name string) []string {
	matches := []string{}
	for iname, idev := range services.Config.Devices {
		if strings.Contains(iname, name) && switchable[idev.Type] {
			matches = append(matches, iname)
		}
	}
	return matches
}

func parseInt(str string, def int) int {
	if num, err := strconv.Atoi(str); err == nil {
		return num
	}
	return def
}

func (self *Service) querySwitch(q services.Question) string {
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
	matches := matchDevices(name)
	if len(matches) == 0 {
		return fmt.Sprintf("device %s not found", name)
	}
	if len(matches) > 1 {
		return fmt.Sprintf("device %s is ambiguous", strings.Join(matches, ", "))
	}

	dev := services.Config.Devices[matches[0]]
	// rest of key=value arguments
	kwargs := keywordArgs(args[1:])
	command := "on"
	fields := pubsub.Fields{}
	for field, value := range kwargs {
		if field == "" {
			command = value
		} else if num, err := strconv.Atoi(value); err == nil {
			fields[field] = num
		} else {
			fields[field] = value
		}
	}
	switchCommand(matches[0], command, fields)
	return fmt.Sprintf("Switched %s %s", dev.Name, command)
}

func switchCommand(name string, command string, params pubsub.Fields) {
	ev := pubsub.NewEvent("command", params)
	ev.SetField("command", command)
	ev.SetField("device", name)
	services.Publisher.Emit(ev)
}

func (self *Service) queryScript(q services.Question) string {
	args := strings.Split(q.Args, " ")
	if len(args) == 0 {
		return "Expected a script name argument"
	}
	progname := path.Base(args[0])
	cmd := path.Join(util.ExpandUser("~/bin/gohome"), progname)
	log.Println("Running script:", cmd)

	output, err := exec.Command(cmd, args[1:]...).Output()
	if err != nil {
		return fmt.Sprintf("Command failed: %s", err)
	}
	return string(output)
}

func tail(filename string, lines int64) ([]byte, error) {
	n := fmt.Sprintf("-n%d", lines)
	return exec.Command("tail", n, filename).Output()
}

func (self *Service) queryLogs(q services.Question) string {
	data, err := tail(eventsLogPath, 25)
	if err != nil {
		return fmt.Sprintf("Couldn't retrieve logs:", err)
	}
	return string(data)
}

func (self *Service) PersistFile(automata *gofsm.Automata) {
	w, err := os.Create(config.ConfigPath("automata.state"))
	if err != nil {
		log.Fatalln("Persisting automata state failed:", err)
	}
	defer w.Close()
	enc := json.NewEncoder(w)
	enc.Encode(automata.Persist())
}

func (self *Service) PersistStore(automata *gofsm.Automata, automaton string) {
	state := automata.Persist()
	v := state[automaton]
	key := "gohome/state/automata/" + automaton
	value, _ := json.Marshal(v)
	err := services.Stor.Set(key, string(value))
	if err != nil {
		log.Println("Persisting automata state to store failed:", err)
	}
}

func (self *Service) RestoreStore(automata *gofsm.Automata) {
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

// Run the service
func (self *Service) Run() error {
	services.SetupStore()
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
	defer services.Subscriber.Close(ch)
	for {
		select {
		case ev := <-ch:
			if ev.Topic == "command" {
				// ignore direct commands - ack/homeeasy events indicate commands completing.
				continue
			}
			if ev.Retained {
				// ignore retained events from reconnecting
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
			// emit event
			fields := pubsub.Fields{
				"device":  change.Automaton,
				"state":   change.New,
				"trigger": trigger.String(),
			}
			ev := pubsub.NewEvent("state", fields)
			ev.SetRetained(true)
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

func (self *Service) appendLog(msg string) {
	now := time.Now()
	logMsg := fmt.Sprintf("%s: %s", now.Format(time.StampMilli), msg)
	fmt.Fprintln(self.log, logMsg)

	fields := pubsub.Fields{
		"message": msg,
		"source":  "event",
	}
	ev := pubsub.NewEvent("log", fields)
	services.Publisher.Emit(ev)
}

func (self EventAction) substitute(msg string) string {
	device := self.event.Device()
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
	msg = self.substitute(msg)
	self.service.appendLog(msg)
	log.Println("Log: ", msg)
}

func (self EventAction) Speak(msg string) {
	self.Alert(msg, "espeak")
}

func (self EventAction) Video(device string, preset int64, secs float64, ir bool) {
	log.Printf("Video: %s at %d for %.1fs (ir: %v)", device, preset, secs, ir)
	ev := pubsub.NewCommand(device, "video")
	ev.SetField("timeout", secs)
	ev.SetField("preset", preset)
	ev.SetField("ir", ir)
	services.Publisher.Emit(ev)
}

func (self EventAction) RingAlarm() {
	command("alarm.bell", "on")
}

func (self EventAction) QuietAlarm() {
	command("alarm.bell", "off")
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

func command(device string, cmd string) {
	ev := pubsub.NewCommand(device, cmd)
	services.Publisher.Emit(ev)
}

func (self EventAction) Query(query string) {
	log.Printf("Query %s", query)
	services.QueryChannel(query, time.Second*5)
	// results currently discarded
}

func (self EventAction) Command(device string, cmd string) {
	log.Printf("Sending %s %s", device, cmd)
	command(device, cmd)
}

var stateCommand = map[bool]string{
	false: "off",
	true:  "on",
}

func (self EventAction) Switch(device string, state bool) {
	command := stateCommand[state]
	switchCommand(device, command, pubsub.Fields{"repeat": 3, "level": 8})
}

func (self EventAction) Dim(device string, level int64) {
	switchCommand(device, "on", pubsub.Fields{"repeat": 3, "level": level})
}

func (self EventAction) Snapshot(device string, target string, message string) {
	message = self.substitute(message)
	ev := pubsub.NewCommand(device, "snapshot")
	ev.SetField("message", message)
	ev.SetField("notify", target)
	services.Publisher.Emit(ev)
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
		fields := pubsub.Fields{
			"device":  "timer." + name,
			"command": "on",
		}
		ev := pubsub.NewEvent("timer", fields)

		services.Publisher.Emit(ev)
	})
	self.service.timers[name] = timer
}
