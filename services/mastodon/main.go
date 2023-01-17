// Service to hook into mastodon and send and receive tweets. This allows the
// house to tweet alerts to you.
package mastodon

import (
	"context"
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	"github.com/mattn/go-mastodon"
)

func createClient() *mastodon.Client {
	m := services.Config.Mastodon
	config := mastodon.Config{
		Server:       m.Server,
		ClientID:     m.Client_id,
		ClientSecret: m.Client_secret,
		AccessToken:  m.Access_token,
	}
	return mastodon.NewClient(&config)
}

func sendToot(client *mastodon.Client, message string) error {
	toot := mastodon.Toot{
		Status:     message,
		Visibility: "private",
	}
	status, err := client.PostStatus(context.Background(), &toot)
	if err != nil {
		log.Printf("Could not send toot: %v", err)
		return err
	}
	log.Printf("Sent: %s", status.URL)
	return nil
}

func toot(client *mastodon.Client, ev *pubsub.Event) {
	msg, _ := ev.Fields["message"].(string)

	log.Printf("Sending toot: %s", msg)
	for retry := 0; retry < 3; retry++ {
		err := sendToot(client, msg)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
}

// Service mastodon
type Service struct{}

func (self *Service) ID() string {
	return "mastodon"
}

func (self *Service) Run() error {
	client := createClient()
	for ev := range services.Subscriber.Subscribe(pubsub.Prefix("alert")) {
		if ev.Target() == "mastodon" {
			toot(client, ev)
		}
	}
	return nil
}
