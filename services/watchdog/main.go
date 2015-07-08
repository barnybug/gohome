// Service for monitoring devices to ensure they're still alive and emitting
// events. Watches a given list of device ids, and alerts if an event has not
// been seen from a device in a configurable time period.
package watchdog

import (
	"fmt"
	"log"
	"net/smtp"
	"sort"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

type WatchdogDevice struct {
	Name        string
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

var devices map[string]*WatchdogDevice
var repeatInterval, _ = time.ParseDuration("12h")

func sendEmail(name, state string, since time.Time) {
	log.Printf("Sending %s watchdog alert for: %s\n", state, name)
	subject := fmt.Sprintf("%s: %s", state, name)
	duration := time.Now().Sub(since)
	body := fmt.Sprintf("since %s (%s ago)", since.Local().Format(time.Stamp), util.ShortDuration(duration))

	email := services.Config.General.Email
	to := []string{email.Admin}
	msg := fmt.Sprintf("Subject: %s\n\n%s\n", subject, body)
	err := smtp.SendMail(email.Server, nil, email.From, to, []byte(msg))
	if err != nil {
		log.Println("Error sending email:", err)
	}
}

func checkEvent(ev *pubsub.Event) {
	// check if in devices monitored
	device := services.Config.LookupDeviceName(ev)
	w := devices[device]
	if w == nil {
		return
	}

	// recovered?
	if w.Alerted {
		w.Alerted = false
		sendEmail(w.Name, "RECOVERED", w.LastEvent)
	}
	w.LastEvent = ev.Timestamp
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
		sendEmail(strings.Join(timeouts, ", "), "PROBLEM", lastEvent)
	}
}

// Service watchdog
type Service struct{}

func (self *Service) ID() string {
	return "watchdog"
}

func (self *Service) Run() error {
	devices = map[string]*WatchdogDevice{}
	now := time.Now()
	for device, timeout := range services.Config.Watchdog.Devices {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			fmt.Println("Failed to parse:", timeout)
		}
		// give devices grace period for first event
		devices[device] = &WatchdogDevice{
			Name:      services.Config.Devices[device].Name,
			Timeout:   duration,
			LastEvent: now,
		}
	}

	// monitor gohome processes heartbeats
	for process, conf := range services.Config.Processes {
		if strings.HasPrefix(conf.Cmd, "gohome service") {
			device := fmt.Sprintf("heartbeat.%s", process)
			// if a process misses 2 heartbeats, mark as problem
			devices[device] = &WatchdogDevice{
				Name:      fmt.Sprintf("Process %s", process),
				Timeout:   time.Second * 121,
				LastEvent: now,
			}
		}
	}

	ticker := time.NewTicker(time.Minute)
	events := services.Subscriber.Channel()
	for {
		select {
		case ev := <-events:
			checkEvent(ev)
		case <-ticker.C:
			checkTimeouts()
		}
	}
	return nil
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": services.TextHandler(self.queryStatus),
		"help":   services.StaticHandler("status: get status\n"),
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
		out += fmt.Sprintf("- %-6s %s %s\n", ago, w.Name, problem)
	}
	return out
}
