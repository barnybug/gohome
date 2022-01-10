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

var serviceMap map[string]Service = map[string]Service{}
var enabled []Service
var Config *config.Config

var Publisher pubsub.Publisher
var Subscriber pubsub.Subscriber

type Listener func()

type ConfigWaiter struct {
	Value   []byte
	hash    uint32
	events  <-chan *pubsub.Event
	update  func()
	Updated chan bool
}

func NewConfigWaiter(topic pubsub.Topic) *ConfigWaiter {
	return &ConfigWaiter{
		events:  Subscriber.Subscribe(topic),
		Updated: make(chan bool),
	}
}

func (c *ConfigWaiter) Wait() {
	if c.loopOne() {
		if c.update != nil {
			c.update()
		}
		c.notify()
	}
}

func (c *ConfigWaiter) notify() {
	// non-blocking send
	select {
	case c.Updated <- true:
	default:
	}
}

func (c *ConfigWaiter) loopOne() bool {
	ev := <-c.events
	value := ev.Bytes()
	hashValue := hash(value)
	previous := c.hash
	if previous == hashValue {
		// ignore duplicate events - from services subscribing to gohome/#.
		return false
	}
	c.hash = hashValue
	c.Value = value
	return true
}

type ConfigService struct {
	ConfigWaiter
	Value *config.Config
}

func NewConfigService() *ConfigService {
	cs := &ConfigService{
		ConfigWaiter{
			events:  Subscriber.Subscribe(pubsub.Exact("config")),
			Updated: make(chan bool),
		},
		nil,
	}
	cs.update = func() {
		// (re)load config
		conf, err := config.OpenRaw(cs.ConfigWaiter.Value)
		if err != nil {
			log.Println("Error reading config:", err)
			return
		}
		cs.Value = conf
		Config = conf // set global
	}
	return cs
}

func SetupLogging() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	log.SetOutput(os.Stdout)
}

func hash(s []byte) uint32 {
	h := fnv.New32a()
	h.Write(s)
	return h.Sum32()
}

var globalConfigService *ConfigService

func WaitForConfig() *ConfigService {
	if globalConfigService == nil {
		globalConfigService = NewConfigService()
		// await first config
		globalConfigService.Wait()
		// listen for updates
		go globalConfigService.Watch()
	}
	return globalConfigService
}

func (c *ConfigService) Watch() {
	for {
		c.Wait()
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

func SetupBroker(name string) {
	// create Publisher
	url := os.Getenv("GOHOME_MQTT")
	if url == "" {
		log.Fatalln("Set GOHOME_MQTT to the mqtt server. eg: tcp://127.0.0.1:1883")
	}

	broker := mqtt.NewBroker(url, name)
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
		} else {
			// services without Init
			WaitForConfig()
		}
	}

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
