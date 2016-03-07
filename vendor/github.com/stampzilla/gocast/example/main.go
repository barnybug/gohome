// Package main provides an example of the stampzilla/gocast library
package main

import (
	"log"
	"time"

	"github.com/stampzilla/gocast/discovery"
	"github.com/stampzilla/gocast/events"
)

func main() {
	discovery := discovery.NewService()

	go discoveryListener(discovery)

	// Start a periodic discovery
	log.Println("Start discovery")
	discovery.Periodic(time.Second * 300)

	select {}
}

func discoveryListener(discovery *discovery.Service) {
	connected := map[string]bool{}
	for device := range discovery.Found() {
		device := device
		if _, ok := connected[device.Name()]; ok {
			log.Printf("Existing device discovered: %+v\n", device)
			continue
		}
		log.Printf("New device discovered: %+v\n", device)

		device.OnEvent(func(event events.Event) {
			switch data := event.(type) {
			case events.Connected:
				log.Printf("%s: connected", device.Name())
				connected[device.Name()] = true
			case events.Disconnected:
				log.Printf("%s: disconnected, reconnecting...", device.Name())
				delete(connected, device.Name())
				// Try to reconnect again
				device.Connect()
			case events.AppStarted:
				log.Printf("%s: App started: %s (%s)", device.Name(), data.DisplayName, data.AppID)
			case events.AppStopped:
				log.Printf("%s: App stopped: %s (%s)", device.Name(), data.DisplayName, data.AppID)
			//gocast.MediaEvent:
			//plexEvent:
			default:
				log.Printf("Unexpected event %T: %#v\n", data, data)
			}
		})
		device.Connect()
	}
}
