package services

import (
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/mqtt"
	"github.com/barnybug/gohome/pubsub/nanomsg"
	"github.com/barnybug/gohome/pubsub/zeromq"
)

type Service interface {
	Id() string
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

func SetupLogging() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(os.Stdout)
}

func SetupConfigFile() {
	var err error
	Config, err = config.Open()
	if err != nil {
		log.Fatalln("Error opening config:", err)
	}
}

func ConfigWatcher() {
	for ev := range Subscriber.FilteredChannel("config") {
		path := ev.StringField("path")
		if path == "gohome/config" {
			// reload config
			err := loadConfigFromStore()
			if err != nil {
				log.Println("Error reloading config:", err)
				continue
			}
			log.Println("Config updated")
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

func loadConfigFromStore() error {
	value, err := Stor.Get("gohome/config")
	if err != nil {
		return err
	}
	conf, err := config.OpenRaw([]byte(value))
	if err != nil {
		return err
	}
	Config = conf
	return nil
}

func SetupConfig() {
	err := loadConfigFromStore()
	if err != nil {
		log.Fatalln("Error opening config:", err)
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

func SetupEndpoints() {
	// create Publisher
	ep := Config.Endpoints
	var broker *mqtt.Broker
	if ep.Mqtt.Broker != "" {
		broker = mqtt.NewBroker(ep.Mqtt.Broker)
		Publisher = broker.Publisher()
	} else if ep.Zeromq.Pub != "" {
		Publisher = zeromq.NewPublisher(ep.Zeromq.Pub, true)
	} else if ep.Nanomsg.Pub != "" {
		Publisher = nanomsg.NewPublisher(ep.Nanomsg.Pub, true)
	}
	if Publisher == nil {
		log.Fatalln("Failed to initialise pub endpoint")
	}

	// create Subscriber
	if ep.Mqtt.Broker != "" {
		Subscriber = broker.Subscriber()
	} else if ep.Zeromq.Sub != "" {
		Subscriber = zeromq.NewSubscriber(ep.Zeromq.Sub, "", true)
	} else if ep.Nanomsg.Sub != "" {
		Subscriber = nanomsg.NewSubscriber(ep.Nanomsg.Sub, "", true)
	}
	if Subscriber == nil {
		log.Fatalln("Failed to initialise sub endpoint")
	}

	// listen for config changes
	go ConfigWatcher()
	// listen for commands
	go QuerySubscriber()
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

	SetupEndpoints()
	SetupFlags()

	for _, service := range enabled {
		log.Printf("Starting %s\n", service.Id())
		// run heartbeater
		go Heartbeat(service.Id())
		err := service.Run()
		if err != nil {
			log.Fatalf("Error running service %s: %s", service.Id(), err.Error())
		}
	}
}

func Heartbeat(id string) {
	prefix := "gohome/heartbeat/" + id
	started := time.Now().Format(time.RFC3339)
	data, _ := json.Marshal(map[string]interface{}{
		"pid":     os.Getpid(),
		"started": started,
	})
	value := string(data)
	for {
		Stor.SetWithTTL(prefix, value, 61)
		time.Sleep(time.Second * 60)
	}
}

func Setup() {
	SetupLogging()
	SetupStore()
	SetupConfig()
}

func Register(service Service) {
	if _, exists := serviceMap[service.Id()]; exists {
		log.Fatalf("Duplicate service registered: %s", service.Id())
	}
	serviceMap[service.Id()] = service
}
