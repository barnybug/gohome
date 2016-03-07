// Package cast service to integrate Google Chromecast.
//
// For example, this allows a chromecast device to switch on your hifi when it
// is turned on.
package cast

import (
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/stampzilla/gocast/discovery"
	"github.com/stampzilla/gocast/events"
)

// Service xpl
type Service struct{}

// ID of the service
func (service *Service) ID() string {
	return "cast"
}

func (service *Service) listener(discover *discovery.Service) {
	connected := map[string]bool{}
	for device := range discover.Found() {
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
				fields := map[string]interface{}{
					"origin":  "cast",
					"command": "on",
					"source":  device.Name(),
					"app":     data.DisplayName,
				}
				event := pubsub.NewEvent("cast", fields)
				services.Publisher.Emit(event)
			case events.AppStopped:
				log.Printf("%s: App stopped: %s (%s)", device.Name(), data.DisplayName, data.AppID)
				fields := map[string]interface{}{
					"origin":  "cast",
					"command": "off",
					"source":  device.Name(),
				}
				event := pubsub.NewEvent("cast", fields)
				services.Publisher.Emit(event)
			default:
				log.Printf("Unexpected event %T: %#v\n", data, data)
			}
		})
		device.Connect()
	}

	log.Println("Listener finished")
}

// Run the service
func (service *Service) Run() error {
	discover := discovery.NewService()

	go service.listener(discover)

	discover.Periodic(time.Second * 300)
	select {}
}
