// Service for a slack bot.
package slack

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/services"
	"github.com/nlopes/slack"
)

// Service daemon
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "slack"
}

func (self *Service) Run() error {
	if services.Config.Slack.Token == "" {
		log.Fatalln("Please set:\nslack:\n  token: \"...\"")
	}

	api := slack.New(services.Config.Slack.Token)
	// api.SetDebug(true)
	rtm := api.NewRTM()
	go slacker(rtm)
	logTransmitter(rtm)

	return nil
}

func lookupChannelByName(api *slack.RTM, name string) *slack.Channel {
	channels, err := api.GetChannels(true)
	if err != nil {
		log.Fatal(err)
	}
	for _, channel := range channels {
		if channel.Name == name {
			return &channel
		}
	}
	return nil
}

func logTransmitter(rtm *slack.RTM) {
	channel := lookupChannelByName(rtm, "logs")
	if channel == nil {
		log.Fatal("You must create #logs and invite me")
	}
	if !channel.IsMember {
		log.Fatal("You must invite me in to #logs")
	}

	for ev := range services.Subscriber.FilteredChannel("log") {
		rtm.SendMessage(rtm.NewOutgoingMessage(ev.StringField("message"), channel.ID))
	}
}

func slacker(rtm *slack.RTM) {
	go rtm.ManageConnection()

	greeted := false
	userId := ""
Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch event := msg.Data.(type) {
			case *slack.ConnectedEvent:
				// say hello in the first channel we're in
				if len(event.Info.Channels) > 0 {
					if !greeted {
						channel := event.Info.Channels[0]
						rtm.SendMessage(rtm.NewOutgoingMessage("gohome bot reporting for duty!", channel.ID))
					}
					greeted = true
				}
				// remember our id
				userId = event.Info.User.ID

			case *slack.MessageEvent:
				if event.User == userId || event.BotID != "" {
					// ignore messages from self or bots
					continue
				}
				// send the message as a query
				log.Println("Querying:", event.Text)
				ch := services.QueryChannel(event.Text, time.Duration(5)*time.Second)

				gotResponse := false
				for ev := range ch {
					// send back responses
					message := ev.StringField("message")
					if message == "" {
						message = ev.String()
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(message, event.Channel))
					gotResponse = true
				}

				if !gotResponse {
					rtm.SendMessage(rtm.NewOutgoingMessage("Sorry, nothing answered!", event.Channel))
				}

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", event.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				// Ignore other events..
				// case *slack.HelloEvent:
				// case *slack.PresenceChangeEvent:
				// case *slack.LatencyReport:
				// fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
