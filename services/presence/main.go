// Service to detect presence of people by pinging a device.
package presence

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
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
	Start(alive chan bool)
	Ping()
}

type Sniffer struct {
	mac string
}

func NewSniffer(mac string) Checker {
	return &Sniffer{mac: mac}
}
func (s *Sniffer) run(alive chan bool) {
	cmd := exec.Command("sudo", "tcpdump", "-p", "-n", "-l", "ether", "host", s.mac)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to start tcpdump: %s", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to start tcpdump: %s", err)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start tcpdump: %s", err)
		return
	}
	log.Printf("Sniffing mac %s (passive)", s.mac)

	// discard stderr
	go io.Copy(ioutil.Discard, stderr)

	// read stdout by line, send an event for each line
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		alive <- true
		// fmt.Println("sniff:", scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("tcpdump failed: %s", err)
		return
	}
}

func (s *Sniffer) Start(alive chan bool) {
	go s.run(alive)
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

func (a *Arpinger) run(alive chan bool) {
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

			alive <- true
		}
	}

}

func (a *Arpinger) Start(alive chan bool) {
	go a.run(alive)
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
	listeners map[string]chan bool
}

// singleton
var hcitool *Hcitool
var hcitoolStarted sync.Once

func (h *Hcitool) Register(mac string, alive chan bool) {
	mac = strings.ToUpper(mac)
	h.l.Lock()
	h.listeners[mac] = alive
	h.l.Unlock()
}

func (h *Hcitool) launch() {
	cmd := exec.Command("sudo", "stdbuf", "-oL", "hcitool", "lescan", "--passive", "--duplicates")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}

	// discard stderr
	go io.Copy(ioutil.Discard, stderr)
	go h.scan(stdout)
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
			log.Println("Bluetooth seen:", mac)
			ch <- true
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("hcitool failed: %s", err)
	}
}

func launchHcitool() {
	hcitool = &Hcitool{
		l:         &sync.Mutex{},
		listeners: map[string]chan bool{},
	}
	hcitool.launch()
}

func (s *Lescanner) run(alive chan bool) {
	hcitoolStarted.Do(launchHcitool)
	log.Printf("Scanning bluetooth %s (passive)", s.mac)
	hcitool.Register(s.mac, alive)

}

func (s *Lescanner) Start(alive chan bool) {
	go s.run(alive)
}

func (s *Lescanner) Ping() {
	// noop
}

func (w *Watchdog) watcher() {
	// start all
	alive := make(chan bool)
	for _, checker := range w.checkers {
		checker.Start(alive)
	}

	home := false
	responded := false
	active := false
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-alive:
			responded = true
			active = false
			if !home {
				log.Printf("%s home", w.device)
				home = true
				emit(w.device, home)
			}
		case <-ticker.C:
			if !responded {
				// send active pings
				for _, checker := range w.checkers {
					checker.Ping()
				}
				if !active {
					active = true
				} else {
					// passive and active checkers exhausted
					if home {
						log.Printf("%s away", w.device)
						home = false
						emit(w.device, home)
					}
				}
			}
			responded = false
		}
	}
}

func (self *Service) Run() error {
	people := map[string]bool{}
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
		watchdog := Watchdog{device, checkers}
		go watchdog.watcher()
	}

	ch := services.Subscriber.FilteredChannel("command")
	for ev := range ch {
		// manual command login/out command
		if _, ok := people[ev.Device()]; ok {
			emit(ev.Device(), ev.Command() == "on")
		}
	}
	return nil
}
