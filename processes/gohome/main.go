package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/services/api"
	"github.com/barnybug/gohome/services/arduino"
	"github.com/barnybug/gohome/services/automata"
	"github.com/barnybug/gohome/services/bills"
	"github.com/barnybug/gohome/services/camera"
	"github.com/barnybug/gohome/services/currentcost"
	"github.com/barnybug/gohome/services/datalogger"
	"github.com/barnybug/gohome/services/earth"
	"github.com/barnybug/gohome/services/espeaker"
	"github.com/barnybug/gohome/services/graphite"
	"github.com/barnybug/gohome/services/heating"
	"github.com/barnybug/gohome/services/irrigation"
	"github.com/barnybug/gohome/services/jabber"
	"github.com/barnybug/gohome/services/pubsub"
	"github.com/barnybug/gohome/services/rfid"
	"github.com/barnybug/gohome/services/rfxtrx"
	"github.com/barnybug/gohome/services/script"
	"github.com/barnybug/gohome/services/sender"
	"github.com/barnybug/gohome/services/slack"
	"github.com/barnybug/gohome/services/sms"
	"github.com/barnybug/gohome/services/systemd"
	"github.com/barnybug/gohome/services/twitter"
	"github.com/barnybug/gohome/services/watchdog"
	"github.com/barnybug/gohome/services/weather"
	"github.com/barnybug/gohome/services/wunderground"
	"github.com/barnybug/gohome/services/xpl"
)

func registerServices() {
	// register available services
	services.Register(&api.Service{})
	services.Register(&arduino.Service{})
	services.Register(&automata.Service{})
	services.Register(&bills.Service{})
	services.Register(&camera.Service{})
	services.Register(&currentcost.Service{})
	services.Register(&datalogger.Service{})
	services.Register(&earth.Service{})
	services.Register(&espeaker.Service{})
	services.Register(&graphite.Service{})
	services.Register(&heating.Service{})
	services.Register(&irrigation.Service{})
	services.Register(&jabber.Service{})
	services.Register(&pubsub.Service{})
	services.Register(&rfid.Service{})
	services.Register(&rfxtrx.Service{})
	services.Register(&script.Service{})
	services.Register(&sender.Service{})
	services.Register(&slack.Service{})
	services.Register(&sms.Service{})
	services.Register(&systemd.Service{})
	services.Register(&twitter.Service{})
	services.Register(&watchdog.Service{})
	services.Register(&weather.Service{})
	services.Register(&wunderground.Service{})
	services.Register(&xpl.Service{})
}

func usage() {
	fmt.Println("Usage: gohome COMMAND [PROCESS/SERVICE]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("   logs    Tail logs (all or select)")
	fmt.Println("   restart Restart a process")
	fmt.Println("   rotate  Rotate logs")
	fmt.Println("   run     Run a service")
	fmt.Println("   start   Start a process")
	fmt.Println("   status  Get process status")
	fmt.Println("   stop    Stop a process")
	fmt.Println("   query   Query services")
	fmt.Println()
}

func main() {
	log.SetOutput(os.Stdout)
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	ps := []string{}
	if flag.NArg() > 1 {
		ps = flag.Args()[1:]
	}
	// ignore anything after '--'
	for i := range ps {
		if ps[i] == "--" {
			ps = ps[0:i]
			break
		}
	}

	services.Setup()

	command := flag.Args()[0]
	switch command {
	default:
		usage()
	case "start":
		start(ps)
	case "stop":
		stop(ps)
	case "restart":
		restart(ps)
	case "ps":
		queryArgs("ps")
	case "status":
		queryArgs("status")
	case "run":
		service(ps)
	case "query":
		query(ps)
	}
}

// Start builtin services
func service(ss []string) {
	registerServices()
	services.Launch(ss)
}

func start(ps []string) {
	queryArgs("start", ps...)
}

func stop(ps []string) {
	queryArgs("stop", ps...)
}

func restart(ps []string) {
	queryArgs("restart", ps...)
}
