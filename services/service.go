package services

import (
	"flag"
	"hash/fnv"
	"log"
	"net/url"
	"os"
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

var Publisher pubsub.Publisher
var Subscriber pubsub.Subscriber
var Stor Store

var Configured = util.NewEvent()

func SetupLogging() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
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
			if Configured.Set() {
				log.Println("Config updated")
			}
		}

		// notify any interested services
		for _, service := range enabled {
			if f, ok := service.(ConfigSubscriber); ok {
				f.ConfigUpdated(path)
			}
		}
	}
}

func SetupStore() {
	if Stor != nil {
		return
	}

	var err error

	address := "redis://:6379"
	if os.Getenv("GOHOME_STORE") != "" {
		address = os.Getenv("GOHOME_STORE")
	}

	url, err := url.Parse(address)
	if err != nil {
		log.Fatalln("could not parse store url: ", address, err)
	}

	switch url.Scheme {
	case "redis":
		Stor, err = NewRedisStore(url.Host)
	case "mock":
		// only for testing
		Stor = NewMockStore()
	default:
		log.Fatalln("scheme", url.Scheme, "not recognised")
	}

	if err != nil {
		log.Fatalln("error connecting to store:", err)
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

func Setup() {
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

	// listen for config changes
	go ConfigWatcher()
	// wait for initial config
	Configured.Wait()
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
	fields := pubsub.Fields{
		"device":  "heartbeat." + id,
		"pid":     os.Getpid(),
		"started": started.Format(time.RFC3339),
		"source":  id,
	}

	// wait 5 seconds before heartbeating - if the process dies very soon
	time.Sleep(time.Second * 5)

	for {
		uptime := int(time.Now().Sub(started).Seconds())
		fields["uptime"] = uptime
		ev := pubsub.NewEvent("heartbeat", fields)
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
