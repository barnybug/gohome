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
	"os"
	"strings"
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
	var stopTimer *time.Timer

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
			if stopTimer != nil {
				stopTimer.Stop()
			}
			log.Printf("%s: App started: %s (%s)", client.Name(), data.DisplayName, data.AppID)
			EmitStopped(client.Name(), "on", data.DisplayName)
		case events.AppStopped:
			log.Printf("%s: App stopped: %s (%s)", client.Name(), data.DisplayName, data.AppID)
			// debounce
			if stopTimer != nil {
				stopTimer.Stop()
			}
			stopTimer = time.AfterFunc(3*time.Second, func() {
				// emit timer event
				EmitStopped(client.Name(), "off", data.DisplayName)
			})
		default:
			// ignored
		}
	}

}

func EmitStopped(name, command, app string) {
	source := fmt.Sprintf("cast.%s", name)
	fields := pubsub.Fields{
		"command": command,
		"source":  source,
		"app":     app,
	}
	ev := pubsub.NewEvent("cast", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
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

func sendAlert(client *cast.Client, message string, volume float64) error {
	// open a fresh client - more reliable
	client = cast.NewClient(client.IP(), client.Port())
	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		return err
	}
	media, err := client.Media(ctx)
	if err != nil {
		client.Close()
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

	// remember volume
	var previousVolume *controllers.Volume
	status, err := client.Receiver().GetStatus(ctx)
	if err == nil {
		previousVolume = status.Volume
	} else {
		defer client.Close()
	}

	if volume == 0 {
		volume = 0.1 // default
	}
	if previousVolume == nil || volume != *previousVolume.Level {
		setVolume(client, volume)
	} else {
		previousVolume = nil
	}

	log.Printf("Playing url: %s", u.String())
	// play media
	_, err = media.LoadMedia(ctx, item, 0, true, nil)

	// restore volume after media has had a chance to play
	if previousVolume != nil {
		time.AfterFunc(time.Second*10, func() {
			setVolume(client, *previousVolume.Level)
			client.Close()
		})
	}

	return err
}

func (service *Service) handleAlert(ev *pubsub.Event) {
	ident, ok := services.Config.LookupDeviceProtocol(ev.Target(), "cast")
	if !ok {
		return
	}
	message, ok := ev.Fields["message"].(string)
	if !ok {
		return
	}

	if client, ok := connected[ident]; ok {
		log.Printf("Casting to %s message: %s", ident, message)
		err := sendAlert(client, message, ev.FloatField("volume"))
		if err != nil {
			log.Printf("Error casting media: %s", err)
		}
	} else {
		log.Printf("Couldn't find cast client: %s", ident)
	}
}

func (service *Service) handleCommand(ev *pubsub.Event) {
	ident, ok := services.Config.LookupDeviceProtocol(ev.Device(), "cast")
	if !ok {
		return
	}

	client, ok := connected[ident]
	if !ok {
		log.Println("Couldn't find cast client:", ident)
		return
	}

	// open a new connection
	client = cast.NewClient(client.IP(), client.Port())
	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		log.Println(err)
		return
	}

	command := ev.Command()
	switch command {
	case "off":
		receiver := client.Receiver()
		receiver.QuitApp(ctx)
	case "on":
		level := ev.FloatField("volume")
		if level != 0 {
			setVolume(client, level)
		}
	default:
		log.Println("Command not recognised:", command)
		return
	}

}

func setVolume(client *cast.Client, level float64) {
	ctx := context.Background()
	receiver := client.Receiver()
	muted := false
	volume := controllers.Volume{Level: &level, Muted: &muted}
	_, err := receiver.SetVolume(ctx, &volume)
	if err == nil {
		log.Printf("Set %s volume to %.2f", client.Name(), level)
	} else {
		log.Println(err)
	}
}

// Writer to use for logging that filters out noise
type FilteredWriter struct{}

func (f FilteredWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "mdns:") {
		return 0, nil
	}
	return os.Stdout.Write(p)
}

// Run the service
func (service *Service) Run() error {
	// mdns is rather noisy. Disable logging for now.
	log.SetOutput(FilteredWriter{})
	ctx := context.Background()
	discover := discovery.NewService(ctx)

	go service.listener(discover)
	go discover.Run(ctx, time.Second*300)

	commands := services.Subscriber.Subscribe(pubsub.Prefix("command"))
	alerts := services.Subscriber.Subscribe(pubsub.Prefix("alert"))
	for {
		select {
		case ev := <-alerts:
			service.handleAlert(ev)
		case ev := <-commands:
			service.handleCommand(ev)
		}
	}
}
