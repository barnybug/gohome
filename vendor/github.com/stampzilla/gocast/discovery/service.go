// Package discovery provides a discovery service for chromecast devices
package discovery

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/stampzilla/gocast"
)

type Service struct {
	found     chan *gocast.Device
	entriesCh chan *mdns.ServiceEntry

	foundDevices map[string]*gocast.Device
	stopPeriodic chan struct{}
}

func NewService() *Service {
	s := &Service{
		found:        make(chan *gocast.Device),
		entriesCh:    make(chan *mdns.ServiceEntry, 10),
		foundDevices: make(map[string]*gocast.Device, 0),
	}

	go s.listener()

	return s
}

func (d *Service) Periodic(interval time.Duration) error {

	if d.stopPeriodic != nil {
		return fmt.Errorf("Periodic discovery is already running")
	}

	mdns.Query(&mdns.QueryParam{
		Service: "_googlecast._tcp",
		Domain:  "local",
		Timeout: time.Second * 3,
		Entries: d.entriesCh,
	})

	ticker := time.NewTicker(interval)
	d.stopPeriodic = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				mdns.Query(&mdns.QueryParam{
					Service: "_googlecast._tcp",
					Domain:  "local",
					Timeout: time.Second * 3,
					Entries: d.entriesCh,
				})
			case <-d.stopPeriodic:
				ticker.Stop()
				d.foundDevices = make(map[string]*gocast.Device, 0)
				return
			}
		}
	}()

	return nil
}

func (d *Service) Stop() {
	if d.stopPeriodic != nil {
		close(d.stopPeriodic)
		d.stopPeriodic = nil
	}
}

func (d *Service) Found() chan *gocast.Device {
	return d.found
}

func (d *Service) listener() {
	for entry := range d.entriesCh {
		// log.Printf("Got new entry: %#v\n", entry)

		name := strings.Split(entry.Name, "._googlecast")

		// Skip everything that doesn't have googlecast in the fdqn
		if len(name) < 2 {
			continue
		}

		key := entry.AddrV4.String() + ":" + strconv.Itoa(entry.Port)

		// Skip already found devices
		if _, ok := d.foundDevices[key]; ok {
			continue
		}

		device := gocast.NewDevice()
		device.SetName(decodeDnsEntry(name[0]))
		device.SetIp(entry.AddrV4)
		device.SetPort(entry.Port)

		info := decodeTxtRecord(entry.Info)
		device.SetUuid(info["id"])

		d.foundDevices[key] = device

		select {
		case d.found <- device:
		case <-time.After(time.Second):
		}
	}
}

func decodeDnsEntry(text string) string {
	text = strings.Replace(text, `\.`, ".", -1)
	text = strings.Replace(text, `\ `, " ", -1)

	re := regexp.MustCompile(`([\\][0-9][0-9][0-9])`)
	text = re.ReplaceAllStringFunc(text, func(source string) string {
		i, err := strconv.Atoi(source[1:])
		if err != nil {
			return ""
		}

		return string([]byte{byte(i)})
	})

	return text
}

func decodeTxtRecord(txt string) map[string]string {
	m := make(map[string]string)

	s := strings.Split(txt, "|")
	for _, v := range s {
		s := strings.Split(v, "=")
		if len(s) == 2 {
			m[s[0]] = s[1]
		}
	}

	return m
}
