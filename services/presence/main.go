// Service to detect presence of people by pinging a device.
package presence

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

const interval = 30 * time.Second

// Service presence
type Service struct {
}

func (self *Service) ID() string {
	return "presence"
}

func emit(device string, state bool) {
	command := "off"
	if state {
		command = "on"
	}
	fields := pubsub.Fields{
		"device":  device,
		"command": command,
		"source":  "presence",
	}
	ev := pubsub.NewEvent("presence", fields)
	services.Publisher.Emit(ev)
}

type Watchdog struct {
	device   string
	checkers []Checker
}

type Checker interface {
	Start(alive chan string)
	Stop()
	Ping()
}

type Sniffer struct {
	mac    string
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func NewSniffer(mac string) Checker {
	return &Sniffer{mac: mac}
}
func (s *Sniffer) run(alive chan string) {
	s.cmd = exec.Command("sudo", "stdbuf", "-oL", "tcpdump", "-p", "-n", "-l", "ether", "host", s.mac)
	var err error
	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to start tcpdump: %s", err)
		return
	}
	s.stderr, err = s.cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to start tcpdump: %s", err)
		return
	}
	if err := s.cmd.Start(); err != nil {
		log.Printf("Failed to start tcpdump: %s", err)
		return
	}
	log.Printf("Sniffing mac %s (passive)", s.mac)

	// discard stderr
	go io.Copy(ioutil.Discard, s.stderr)

	// read stdout by line, send an event for each line
	scanner := bufio.NewScanner(s.stdout)
	for scanner.Scan() {
		l := len(scanner.Text())
		if l > 0 {
			alive <- "sniffed"
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("tcpdump failed: %s", err)
		return
	}
}

func (s *Sniffer) Start(alive chan string) {
	go s.run(alive)
}

func (s *Sniffer) Stop() {
	s.cmd.Wait()
	log.Println("Terminated tcpdump")
}

func (s *Sniffer) Ping() {
	// noop
}

type Arpinger struct {
	host    string
	control *sync.Cond
}

func NewArpinger(host string) Checker {
	return &Arpinger{host: host, control: sync.NewCond(&sync.Mutex{})}
}

func (a *Arpinger) run(alive chan string) {
	addr, err := net.ResolveIPAddr("ip4:icmp", a.host)
	if err != nil {
		log.Printf("Failed to resolve host, not pinging: %s", err)
		return
	}

	for {
		for {
			// wait for Ping
			a.control.L.Lock()
			a.control.Wait()
			a.control.L.Unlock()

			// log.Printf("Arpinging %s on %s", a.host, addr)
			cmd := exec.Command("sudo", "arping", "-f", addr.String())
			err = cmd.Run()
			if err != nil {
				log.Printf("arping %s failed: %s", addr.String(), err)
				return
			}

			alive <- "arping"
		}
	}

}

func (a *Arpinger) Start(alive chan string) {
	go a.run(alive)
}

func (a *Arpinger) Stop() {
	// noop
}

func (a *Arpinger) Ping() {
	a.control.Signal()
}

type Lescanner struct {
	mac string
}

func NewLescanner(mac string) Checker {
	return &Lescanner{mac: mac}
}

type Hcitool struct {
	l         sync.Locker
	listeners map[string]chan string
	stderr    bytes.Buffer
	cmd       *exec.Cmd
}

// singleton
var hcitool = &Hcitool{
	l:         &sync.Mutex{},
	listeners: map[string]chan string{},
}
var hcitoolStarted sync.Once

func (h *Hcitool) Register(mac string, alive chan string) {
	hcitoolStarted.Do(hcitool.launch)

	mac = strings.ToUpper(mac)
	h.l.Lock()
	h.listeners[mac] = alive
	h.l.Unlock()
}

func (h *Hcitool) Deregister(mac string) {
	mac = strings.ToUpper(mac)
	h.l.Lock()
	delete(h.listeners, mac)
	if len(h.listeners) == 0 {
		hcitool.terminate()
	}
	h.l.Unlock()
}

func (h *Hcitool) launch() {
	h.cmd = exec.Command("sudo", "stdbuf", "-oL", "hcitool", "lescan", "--passive", "--duplicates")
	stdout, err := h.cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}
	stderr, err := h.cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}
	if err := h.cmd.Start(); err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}

	go io.Copy(&h.stderr, stderr)
	go h.scan(stdout)
}

func (h *Hcitool) terminate() {
	h.cmd.Wait()
	log.Println("Terminated hcitool")
}

func (h *Hcitool) scan(stdout io.ReadCloser) {
	// read stdout by line, send an event for each line
	scanner := bufio.NewScanner(stdout)
	// drop first line
	scanner.Scan()
	for scanner.Scan() {
		line := scanner.Text()
		ps := strings.SplitN(line, " ", 2)
		mac := ps[0]
		h.l.Lock()
		ch, exists := h.listeners[mac]
		h.l.Unlock()
		if exists {
			// log.Println("Bluetooth seen:", mac)
			ch <- "bluetooth"
		}
	}

	stderr := h.stderr.Bytes()
	if len(stderr) > 0 {
		log.Printf("hcitool error: %s", string(stderr))
	}

	if err := scanner.Err(); err != nil {
		log.Printf("hcitool failed: %s", err)
	} else {
		log.Printf("hcitool exited: bluetooth monitoring disabled")
	}
}

func (s *Lescanner) run(alive chan string) {
	log.Printf("Scanning bluetooth %s (passive)", s.mac)
	hcitool.Register(s.mac, alive)

}

func (s *Lescanner) Start(alive chan string) {
	go s.run(alive)
}

func (s *Lescanner) Stop() {
	hcitool.Deregister(s.mac)
}

func (s *Lescanner) Ping() {
	// noop
}

func (w *Watchdog) watcher() {
	// start all
	alive := make(chan string)
	for _, checker := range w.checkers {
		checker.Start(alive)
	}

	home := false
	active := false
	timeout := time.NewTimer(interval)
	for {
		select {
		case name := <-alive:
			if !home {
				log.Printf("%s home (%s)", w.device, name)
				home = true
				emit(w.device, home)
			}
			// make next time period use passive checks
			active = false
			// reset timeout
			if !timeout.Stop() {
				<-timeout.C
			}
			timeout.Reset(interval)
		case <-timeout.C:
			// send active pings
			for _, checker := range w.checkers {
				checker.Ping()
			}
			if !active {
				// give active pings another timeout period to respond
				active = true
			} else {
				// passive and active checkers exhausted
				if home {
					log.Printf("%s away", w.device)
					home = false
					emit(w.device, home)
				}
			}
			// start timeout again
			timeout.Reset(interval)
		}
	}
}

func (w *Watchdog) Stop() {
	for _, checker := range w.checkers {
		checker.Stop()
	}
}

func (self *Service) shutdown(watchdogs []*Watchdog) {
	log.Println("Shutting down...")
	// Send INT to whole process group (pid=0)
	// Note: the only clean way of stopping hcitool is a SIGINT, any other signals
	// result in an unusable hci device requiring a down/up to reset.
	// Must sudo to kill the sudo'ed processes
	cmd := exec.Command("sudo", "kill", "-INT", "0")
	cmd.Run()
	for _, watchdog := range watchdogs {
		watchdog.Stop()
	}
	log.Println("Shut down complete")
}

func (self *Service) Run() error {
	people := map[string]bool{}
	watchdogs := []*Watchdog{}
	for device, checks := range services.Config.Presence {
		people[device] = true
		var checkers []Checker
		for _, conf := range checks {
			var checker Checker
			ps := strings.Split(conf, " ")
			if len(ps) != 2 {
				log.Printf("Error: misconfigured '%s'", conf)
				continue
			}
			switch ps[0] {
			case "sniff":
				checker = NewSniffer(ps[1])
			case "arping":
				checker = NewArpinger(ps[1])
			case "lescan":
				checker = NewLescanner(ps[1])
			}
			checkers = append(checkers, checker)
		}
		watchdog := &Watchdog{device, checkers}
		watchdogs = append(watchdogs, watchdog)
		go watchdog.watcher()
	}

	// Gracefully handle signals
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)

	commands := services.Subscriber.FilteredChannel("command")
L:
	for {
		select {
		case <-sigC:
			break L
		case ev := <-commands:
			// manual command login/out command
			if _, ok := people[ev.Device()]; ok {
				emit(ev.Device(), ev.Command() == "on")
			}
		}
	}

	self.shutdown(watchdogs)
	return nil
}
