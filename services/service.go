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

// ServiceInit interface
type ServiceInit interface {
	Service
	Init() error
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

type ConfigStorage struct {
	config  map[string][]byte
	hashes  map[string]uint32
	channel <-chan *pubsub.Event
}

var Configurations *ConfigStorage

func SetupLogging() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	log.SetOutput(os.Stdout)
}

func hash(s []byte) uint32 {
	h := fnv.New32a()
	h.Write(s)
	return h.Sum32()
}

func CreateConfigStorage() *ConfigStorage {
	return &ConfigStorage{
		config:  map[string][]byte{},
		hashes:  map[string]uint32{},
		channel: Subscriber.FilteredChannel("config"),
	}
}

func (c *ConfigStorage) loopOnce() {
	ev := <-c.channel
	path := ev.Topic
	value := ev.Bytes()
	hashValue := hash(value)
	previous := c.hashes[path]
	if previous == hashValue {
		// ignore duplicate events - from services subscribing to gohome/#.
		return
	}
	c.hashes[path] = hashValue
	if path == "config" {
		// (re)load config
		conf, err := config.OpenRaw([]byte(value))
		if err != nil {
			log.Println("Error reloading config:", err)
			return
		}
		Config = conf
		RawConfig = []byte(value)
	}

	c.config[path] = value
	if previous != 0 {
		log.Printf("%s updated", path)
	}

	// notify any interested services
	for _, service := range enabled {
		if f, ok := service.(ConfigSubscriber); ok {
			f.ConfigUpdated(path)
		}
	}
}

func (c *ConfigStorage) WaitForConfig(path string) {
	for {
		if _, ok := c.config[path]; ok {
			return
		}
		c.loopOnce()
	}
}

func (c *ConfigStorage) Watch() {
	for {
		c.loopOnce()
	}
}

func (c *ConfigStorage) Get(path string) []byte {
	return c.config[path]
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

func SetupBroker(name string) {
	// create Publisher
	url := os.Getenv("GOHOME_MQTT")
	if url == "" {
		log.Fatalln("Set GOHOME_MQTT to the mqtt server. eg: tcp://127.0.0.1:1883")
	}

	var broker *mqtt.Broker
	broker = mqtt.NewBroker(url, name)
	Publisher = broker.Publisher()
	if Publisher == nil {
		log.Fatalln("Failed to initialise pub endpoint")
	}
	Subscriber = broker.Subscriber()
	if Subscriber == nil {
		log.Fatalln("Failed to initialise sub endpoint")
	}

}

func Setup(name string) {
	SetupBroker(name)
	Configurations = CreateConfigStorage()
	// wait for initial config
	Configurations.WaitForConfig("config")
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
		if service, ok := service.(ServiceInit); ok {
			err := service.Init()
			if err != nil {
				log.Fatalf("Error init service %s: %s", service.ID(), err.Error())
			}
			log.Printf("Initialized %s\n", service.ID())
		}
	}

	// listen for config changes
	go Configurations.Watch()

	for _, service := range enabled {
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

	// notify systemd ready
	util.SdNotify(false, util.SdNotifyReady)

	// wait 5 seconds before heartbeating - if the process dies very soon
	time.Sleep(time.Second * 5)

	for {
		uptime := int(time.Now().Sub(started).Seconds())
		fields["uptime"] = uptime
		ev := pubsub.NewEvent("heartbeat", fields)
		ev.SetRetained(true)
		Publisher.Emit(ev)
		ev.Published.Wait() // block on actually publishing
		time.Sleep(time.Second * 60)
		// notify systemd watchdog
		util.SdNotify(false, util.SdNotifyWatchdog)
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

func Shutdown() {
	if Publisher != nil {
		Publisher.Close()
	}
}
