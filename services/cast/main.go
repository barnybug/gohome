// Package cast service to integrate Google Chromecast.
//
// For example, this allows a chromecast device to switch on your hifi when it
// is turned on.
package cast

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/barnybug/go-cast"
	"github.com/barnybug/go-cast/discovery"
	"github.com/barnybug/go-cast/events"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

// Service xpl
type Service struct{}

// ID of the service
func (service *Service) ID() string {
	return "cast"
}

var connected = map[string]bool{}

func listener(ctx context.Context, client *cast.Client) {
LOOP:
	for {
		event := <-client.Events
		switch data := event.(type) {
		case events.Connected:
			log.Printf("%s: connected", client.Name())
			connected[client.Name()] = true
		case events.Disconnected:
			log.Printf("%s: disconnected, reconnecting...", client.Name())
			client.Close()
			delete(connected, client.Name())
			// Try to reconnect again
			err := client.Connect(ctx)
			if err != nil {
				log.Printf("Failed to reconnect to %s, removing: %s", client.Name(), err)
				break LOOP
			}
		case events.AppStarted:
			log.Printf("%s: App started: %s (%s)", client.Name(), data.DisplayName, data.AppID)
			source := fmt.Sprintf("cast.%s", client.Name())
			fields := map[string]interface{}{
				"command": "on",
				"source":  source,
				"app":     data.DisplayName,
			}
			ev := pubsub.NewEvent("cast", fields)
			services.Config.AddDeviceToEvent(ev)
			services.Publisher.Emit(ev)
		case events.AppStopped:
			log.Printf("%s: App stopped: %s (%s)", client.Name(), data.DisplayName, data.AppID)
			source := fmt.Sprintf("cast.%s", client.Name())
			fields := map[string]interface{}{
				"command": "off",
				"source":  source,
			}
			ev := pubsub.NewEvent("cast", fields)
			services.Config.AddDeviceToEvent(ev)
			services.Publisher.Emit(ev)
		default:
			// ignored
		}
	}

}

func (service *Service) listener(discover *discovery.Service) {
	ctx := context.Background()
	for client := range discover.Found() {
		if _, ok := connected[client.Name()]; ok {
			continue
		}
		log.Printf("New client discovered: %s\n", client)

		err := client.Connect(ctx)
		if err == nil {
			go listener(ctx, client)
		} else {
			log.Printf("Failed to connect to %s: %s", client.Name(), err)
		}
	}

	log.Println("Listener finished")
}

// Run the service
func (service *Service) Run() error {
	// mdns is rather noisy. Disable logging for now.
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
	ctx := context.Background()
	discover := discovery.NewService(ctx)

	go service.listener(discover)

	discover.Run(ctx, time.Second*300)
	return nil
}
