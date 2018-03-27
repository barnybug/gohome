package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/services/api"
	"github.com/barnybug/gohome/services/arduino"
	"github.com/barnybug/gohome/services/automata"
	"github.com/barnybug/gohome/services/bills"
	"github.com/barnybug/gohome/services/ble"
	"github.com/barnybug/gohome/services/camera"
	"github.com/barnybug/gohome/services/cast"
	"github.com/barnybug/gohome/services/cheerlights"
	"github.com/barnybug/gohome/services/currentcost"
	"github.com/barnybug/gohome/services/datalogger"
	"github.com/barnybug/gohome/services/earth"
	"github.com/barnybug/gohome/services/energenie"
	"github.com/barnybug/gohome/services/espeaker"
	"github.com/barnybug/gohome/services/graphite"
	"github.com/barnybug/gohome/services/heating"
	"github.com/barnybug/gohome/services/hwmon"
	"github.com/barnybug/gohome/services/irrigation"
	"github.com/barnybug/gohome/services/jabber"
	"github.com/barnybug/gohome/services/lirc"
	"github.com/barnybug/gohome/services/miflora"
	"github.com/barnybug/gohome/services/orvibo"
	"github.com/barnybug/gohome/services/presence"
	"github.com/barnybug/gohome/services/pushbullet"
	"github.com/barnybug/gohome/services/raspi"
	"github.com/barnybug/gohome/services/rfid"
	"github.com/barnybug/gohome/services/rfxtrx"
	"github.com/barnybug/gohome/services/script"
	"github.com/barnybug/gohome/services/sender"
	"github.com/barnybug/gohome/services/slack"
	"github.com/barnybug/gohome/services/sms"
	"github.com/barnybug/gohome/services/systemd"
	"github.com/barnybug/gohome/services/tasmota"
	"github.com/barnybug/gohome/services/telegram"
	"github.com/barnybug/gohome/services/twitter"
	"github.com/barnybug/gohome/services/ups"
	"github.com/barnybug/gohome/services/watchdog"
	"github.com/barnybug/gohome/services/weather"
	"github.com/barnybug/gohome/services/wunderground"
	"github.com/barnybug/gohome/services/xpl"
	"github.com/barnybug/gohome/services/yeelight"
)

func registerServices() {
	// register available services
	services.Register(&api.Service{})
	services.Register(&arduino.Service{})
	services.Register(&automata.Service{})
	services.Register(&bills.Service{})
	services.Register(&ble.Service{})
	services.Register(&camera.Service{})
	services.Register(&cast.Service{})
	services.Register(&cheerlights.Service{})
	services.Register(&currentcost.Service{})
	services.Register(&datalogger.Service{})
	services.Register(&earth.Service{})
	services.Register(&energenie.Service{})
	services.Register(&espeaker.Service{})
	services.Register(&graphite.Service{})
	services.Register(&heating.Service{})
	services.Register(&hwmon.Service{})
	services.Register(&irrigation.Service{})
	services.Register(&jabber.Service{})
	services.Register(&lirc.Service{})
	services.Register(&miflora.Service{})
	services.Register(&orvibo.Service{})
	services.Register(&presence.Service{})
	services.Register(&pushbullet.Service{})
	services.Register(&raspi.Service{})
	services.Register(&rfid.Service{})
	services.Register(&rfxtrx.Service{})
	services.Register(&script.Service{})
	services.Register(&sender.Service{})
	services.Register(&slack.Service{})
	services.Register(&sms.Service{})
	services.Register(&systemd.Service{})
	services.Register(&telegram.Service{})
	services.Register(&tasmota.Service{})
	services.Register(&twitter.Service{})
	services.Register(&ups.Service{})
	services.Register(&watchdog.Service{})
	services.Register(&weather.Service{})
	services.Register(&wunderground.Service{})
	services.Register(&xpl.Service{})
	services.Register(&yeelight.Service{})
}

func usage() {
	fmt.Println("Usage: gohome COMMAND [PROCESS/SERVICE]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("   config  path filename   Update config")
	fmt.Println("   logs                    Tail logs")
	fmt.Println("   restart [service]       Restart a service")
	fmt.Println("   run     [service]       Run a service")
	fmt.Println("   start   [service]       Start a service")
	fmt.Println("   status  [service]       Get service status")
	fmt.Println("   stop    [service]       Stop a process")
	fmt.Println("   query   ...             Query services")
	fmt.Println()
}

var emptyParams = url.Values{}

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

	services.SetupLogging()

	command := flag.Args()[0]
	switch command {
	default:
		usage()
	case "config":
		if len(ps) < 2 {
			usage()
			return
		}
		config(ps[0], ps[1])
	case "start":
		query("start", ps, emptyParams)
	case "stop":
		query("stop", ps, emptyParams)
	case "restart":
		query("restart", ps, emptyParams)
	case "ps":
		query("ps", []string{}, url.Values{"timeout": {"1000"}})
	case "status":
		if len(ps) == 0 {
			// all services
			query("status", []string{}, emptyParams)
		} else {
			// single service
			query(ps[0]+"/status", []string{}, url.Values{"responses": {"1"}})
		}
	case "run":
		service(ps)
	case "switch":
		commandSwitch(ps)
	case "query":
		if len(ps) == 0 {
			usage()
			return
		}
		query(ps[0], ps[1:], url.Values{"timeout": {"5000"}, "responses": {"1"}})
	case "logs":
		stream("logs", emptyParams)
	}
}

func commandSwitch(ps []string) {
	if len(ps) < 2 {
		usage()
		return
	}

	params := url.Values{
		"id":      []string{ps[0]},
		"command": []string{ps[1]},
	}
	for _, arg := range ps[2:] {
		ps := strings.SplitN(arg, "=", 2)
		if len(ps) > 1 {
			params[ps[0]] = ps[1:2]
		}
	}
	resp, err := request("devices/control", params)
	if err != nil {
		fmtFatalf("error: %s\n", err)
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

// Start builtin services
func service(ss []string) {
	services.Setup()
	registerServices()
	services.Launch(ss)
}
