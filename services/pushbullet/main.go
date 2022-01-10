// Service to send pushbullet messages.
package pushbullet

import (
	"log"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	"github.com/mitsuse/pushbullet-go"
	"github.com/mitsuse/pushbullet-go/requests"
)

var pb *pushbullet.Pushbullet

func sendMessage(ev *pubsub.Event) {

	if message, ok := ev.Fields["message"].(string); ok {
		log.Printf("Sending pushbullet note: %s", message)
		n := requests.NewNote()
		n.Title = "Gohome"
		n.Body = message

		if _, err := pb.PostPushesNote(n); err != nil {
			log.Printf("Pushbullet error: %s", err)
		}
	}
}

// Service pushbullet
type Service struct{}

func (self *Service) ID() string {
	return "pushbullet"
}

func (self *Service) Run() error {
	pb = pushbullet.New(services.Config.Pushbullet.Token)

	events := services.Subscriber.Subscribe(pubsub.Prefix("alert"))
	for ev := range events {
		if ev.Target() == "pushbullet" {
			sendMessage(ev)
		}
	}
	return nil
}
