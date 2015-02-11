package main

import (
	"flag"
	"fmt"
	"github.com/barnybug/gohome/processes"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/services/api"
	"github.com/barnybug/gohome/services/arduino"
	"github.com/barnybug/gohome/services/automata"
	"github.com/barnybug/gohome/services/bills"
	"github.com/barnybug/gohome/services/camera"
	"github.com/barnybug/gohome/services/currentcost"
	"github.com/barnybug/gohome/services/daemon"
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
	"github.com/barnybug/gohome/services/sms"
	"github.com/barnybug/gohome/services/twitter"
	"github.com/barnybug/gohome/services/watchdog"
	"github.com/barnybug/gohome/services/weather"
	"github.com/barnybug/gohome/services/wunderground"
	"github.com/barnybug/gohome/services/xpl"
	"log"
	"os"
)

func registerServices() {
	// register available services
	services.Register(&api.ApiService{})
	services.Register(&arduino.ArduinoService{})
	services.Register(&automata.AutomataService{})
	services.Register(&bills.BillsService{})
	services.Register(&camera.CameraService{})
	services.Register(&currentcost.CurrentcostService{})
	services.Register(&datalogger.DataloggerService{})
	services.Register(&daemon.DaemonService{})
	services.Register(&earth.EarthService{})
	services.Register(&espeaker.EspeakerService{})
	services.Register(&graphite.GraphiteService{})
	services.Register(&heating.HeatingService{})
	services.Register(&irrigation.IrrigationService{})
	services.Register(&jabber.JabberService{})
	services.Register(&pubsub.PubsubService{})
	services.Register(&rfid.RfidService{})
	services.Register(&rfxtrx.RfxtrxService{})
	services.Register(&script.ScriptService{})
	services.Register(&sender.SenderService{})
	services.Register(&sms.SmsService{})
	services.Register(&twitter.TwitterService{})
	services.Register(&watchdog.WatchdogService{})
	services.Register(&weather.WeatherService{})
	services.Register(&wunderground.WundergroundService{})
	services.Register(&xpl.XplService{})
}

func Usage() {
	fmt.Println("Usage: gohome COMMAND [PROCESS/SERVICE]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("   logs    Tail logs (all or select)")
	fmt.Println("   restart Restart a process")
	fmt.Println("   rotate  Rotate logs")
	fmt.Println("   run     Run a process (foreground)")
	fmt.Println("   service Execute a builtin service")
	fmt.Println("   start   Start a process")
	fmt.Println("   status  Get process status")
	fmt.Println("   stop    Stop a process")
	fmt.Println("   query   Query services")
	fmt.Println()
	os.Exit(1)
}

func main() {
	log.SetOutput(os.Stdout)
	flag.Usage = Usage
	flag.Parse()
	if flag.NArg() < 1 {
		Usage()
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
		Usage()
	case "start":
		processes.Start(ps)
	case "stop":
		processes.Stop(ps)
	case "restart":
		processes.Restart(ps)
	case "status", "ps":
		processes.Status(ps)
	case "service":
		service(ps)
	case "run":
		processes.Run(ps)
	case "logs":
		processes.Logs(ps)
	case "rotate":
		processes.Rotate(ps)
	case "query":
		query(ps)
	}
}

// Start builtin services
func service(ss []string) {
	registerServices()
	services.Launch(ss)
}
