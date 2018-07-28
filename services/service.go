package services

import (
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"strings"
	"time"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/mqtt"
	"github.com/barnybug/gohome/util"
)

// Service interface
type Service interface {
	ID() string
	Run() error
}

type Flags interface {
	Flags()
}

type ConfigSubscriber interface {
	ConfigUpdated(path string)
}

var serviceMap map[string]Service = map[string]Service{}
var enabled []Service
var Config *config.Config
var RawConfig []byte

var Publisher pubsub.Publisher
var Subscriber pubsub.Subscriber

type ConfigEntry struct {
	value string
	event *util.Event
}

func (c *ConfigEntry) Wait() {
	c.event.Wait()
}

func (c *ConfigEntry) Get() string {
	c.event.Wait()
	return c.value
}

func (c *ConfigEntry) Set(s string) bool {
	c.value = s
	return c.event.Set()
}

var Configured = map[string]*ConfigEntry{
	"config":          &ConfigEntry{"", util.NewEvent()},
	"config/automata": &ConfigEntry{"", util.NewEvent()},
}

func SetupLogging() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	log.SetOutput(os.Stdout)
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func ConfigWatcher() {
	seen := map[string]uint32{}

	for ev := range Subscriber.FilteredChannel("config") {
		path := ev.Topic
		value := ev.StringField("config")
		hashValue := hash(value)
		previous := seen[path]
		if previous == hashValue {
			// ignore duplicate events - from services subscribing to gohome/#.
			continue
		}
		seen[path] = hashValue
		if path == "config" {
			// (re)load config
			conf, err := config.OpenRaw([]byte(value))
			if err != nil {
				log.Println("Error reloading config:", err)
				continue
			}
			Config = conf
			RawConfig = []byte(value)
		}
		c := Configured[path]
		if c.Set(value) {
			log.Printf("%s updated", path)
		}

		// notify any interested services
		for _, service := range enabled {
			if f, ok := service.(ConfigSubscriber); ok {
				f.ConfigUpdated(path)
			}
		}
	}
}

func SetupFlags() {
	for _, service := range enabled {
		// any service specific flags
		if f, ok := service.(Flags); ok {
			f.Flags()
		}
	}
	flag.Parse()
}

func SetupBroker() {
	// create Publisher
	url := os.Getenv("GOHOME_MQTT")
	if url == "" {
		log.Fatalln("Set GOHOME_MQTT to the mqtt server. eg: tcp://127.0.0.1:1883")
	}

	var broker *mqtt.Broker
	broker = mqtt.NewBroker(url)
	Publisher = broker.Publisher()
	if Publisher == nil {
		log.Fatalln("Failed to initialise pub endpoint")
	}
	Subscriber = broker.Subscriber()
	if Subscriber == nil {
		log.Fatalln("Failed to initialise sub endpoint")
	}

}

func Setup() {
	SetupBroker()
	// listen for config changes
	go ConfigWatcher()
	// wait for initial config
	Configured["config"].Wait()
}

func Launch(ss []string) {
	enabled = []Service{}
	for _, name := range ss {
		if service, ok := serviceMap[name]; ok {
			enabled = append(enabled, service)
		} else {
			log.Fatalf("Service %s does not exist", name)
		}
	}

	SetupFlags()

	// listen for commands
	go QuerySubscriber()

	for _, service := range enabled {
		log.Printf("Starting %s\n", service.ID())
		// run heartbeater
		go Heartbeat(service.ID())
		err := service.Run()
		if err != nil {
			log.Fatalf("Error running service %s: %s", service.ID(), err.Error())
		}
	}
}

func Heartbeat(id string) {
	started := time.Now()
	device := fmt.Sprintf("heartbeat.%s", id)
	fields := pubsub.Fields{
		"device":  device,
		"pid":     os.Getpid(),
		"started": started.Format(time.RFC3339),
	}

	// wait 5 seconds before heartbeating - if the process dies very soon
	time.Sleep(time.Second * 5)

	for {
		uptime := int(time.Now().Sub(started).Seconds())
		fields["uptime"] = uptime
		ev := pubsub.NewEvent("heartbeat", fields)
		ev.SetRetained(true)
		Publisher.Emit(ev)
		time.Sleep(time.Second * 60)
	}
}

func Register(service Service) {
	if _, exists := serviceMap[service.ID()]; exists {
		log.Fatalf("Duplicate service registered: %s", service.ID())
	}
	serviceMap[service.ID()] = service
}

func MatchDevices(n string) []string {
	if _, ok := Config.Devices[n]; ok {
		return []string{n}
	}

	matches := []string{}
	for name, dev := range Config.Devices {
		if strings.Contains(name, n) && dev.IsSwitchable() {
			matches = append(matches, name)
		}
	}
	return matches
}
