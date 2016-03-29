// Package cast service to integrate Google Chromecast.
//
// For example, this allows a chromecast device to switch on your hifi when it
// is turned on.
package cast

import (
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
			fields := map[string]interface{}{
				"origin":  "cast",
				"command": "on",
				"source":  client.Name(),
				"app":     data.DisplayName,
			}
			event := pubsub.NewEvent("cast", fields)
			services.Publisher.Emit(event)
		case events.AppStopped:
			log.Printf("%s: App stopped: %s (%s)", client.Name(), data.DisplayName, data.AppID)
			fields := map[string]interface{}{
				"origin":  "cast",
				"command": "off",
				"source":  client.Name(),
			}
			event := pubsub.NewEvent("cast", fields)
			services.Publisher.Emit(event)
		default:
			// ignored
		}
	}

}

func (service *Service) listener(discover *discovery.Service) {
	ctx := context.Background()
	for client := range discover.Found() {
		if _, ok := connected[client.Name()]; ok {
			log.Printf("Existing client discovered: %s\n", client)
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
	ctx := context.Background()
	discover := discovery.NewService(ctx)

	go service.listener(discover)

	discover.Run(ctx, time.Second*300)
	return nil
}
