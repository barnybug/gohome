// Package cast service to integrate Google Chromecast.
//
// For example, this allows a chromecast device to switch on your hifi when it
// is turned on.
package cast

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"golang.org/x/net/context"

	"github.com/barnybug/go-cast"
	"github.com/barnybug/go-cast/controllers"
	"github.com/barnybug/go-cast/discovery"
	"github.com/barnybug/go-cast/events"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

// Service cast
type Service struct{}

// ID of the service
func (service *Service) ID() string {
	return "cast"
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	// no connection actually made
	conn, err := net.Dial("udp", "240.0.0.1:9")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

var connected = map[string]*cast.Client{}

func listener(ctx context.Context, client *cast.Client) {
LOOP:
	for {
		event := <-client.Events
		switch data := event.(type) {
		case events.Connected:
			log.Printf("%s: connected", client.Name())
			connected[client.Name()] = client

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

func sendAlert(client *cast.Client, message string) error {
	// open a fresh client - more reliable
	client = cast.NewClient(client.IP(), client.Port())
	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	media, err := client.Media(ctx)
	if err != nil {
		return err
	}
	params := url.Values{"text": []string{message}}
	host := fmt.Sprintf("%s:%d",
		GetOutboundIP(),
		services.Config.Espeak.Port)
	u := url.URL{
		Scheme:   "http",
		Host:     host,
		Path:     "speak",
		RawQuery: params.Encode(),
	}
	item := controllers.MediaItem{
		ContentId:   u.String(),
		ContentType: "audio/x-wav",
		StreamType:  "BUFFERED",
	}
	log.Printf("Playing url: %s", u.String())
	_, err = media.LoadMedia(ctx, item, 0, true, nil)
	return err
}

func (service *Service) alerts() {
	for ev := range services.Subscriber.FilteredChannel("alert") {
		pids := services.Config.LookupDeviceProtocol(ev.Target())
		message, ok := ev.Fields["message"].(string)
		if pids["cast"] == "" || !ok {
			continue
		}

		if client, ok := connected[pids["cast"]]; ok {
			log.Printf("Casting to %s message: %s", pids["cast"], message)
			err := sendAlert(client, message)
			if err != nil {
				log.Printf("Error casting media: %s", err)
			}
		} else {
			log.Printf("Couldn't find cast client: %s", pids["cast"])
		}
	}
}

// Run the service
func (service *Service) Run() error {
	// mdns is rather noisy. Disable logging for now.
	// log.SetFlags(0)
	// log.SetOutput(ioutil.Discard)
	ctx := context.Background()
	discover := discovery.NewService(ctx)

	go service.listener(discover)
	go service.alerts()

	discover.Run(ctx, time.Second*300)
	return nil
}
