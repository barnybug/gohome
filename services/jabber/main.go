// Service to provide a jabber bot that can be queried, and provide alerts on
// activity.
//
// To authorise a user to talk to this jabber user, login manually as the user
// and add them as you would normally.
package jabber

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/barnybug/gohome/services"

	xmpp "github.com/mattn/go-xmpp"
)

// Service jabber
type Service struct {
	talk *xmpp.Client
}

func (self *Service) ID() string {
	return "jabber"
}

func IsSelf(from string) bool {
	return strings.HasPrefix(from, services.Config.Jabber.Jid)
}

type JabberClient struct {
	client *xmpp.Client
	recv   chan interface{}
	send   chan xmpp.Chat
}

func recvChannel(talk *xmpp.Client, ch chan interface{}) {
	for {
		chat, err := talk.Recv()
		if err != nil {
			return
		}
		ch <- chat
	}
}

func NewClient() (*JabberClient, error) {
	debug := false
	talk, err := xmpp.NewClient("talk.google.com:443",
		services.Config.Jabber.Jid,
		services.Config.Jabber.Pass,
		debug)
	if err != nil {
		return nil, err
	}
	ret := JabberClient{
		talk,
		make(chan interface{}, 1),
		make(chan xmpp.Chat, 1),
	}

	go recvChannel(talk, ret.recv)

	// keepalive interval to reconnect, otherwise the connection dies
	interval := time.Minute * 15
	keepalive := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-keepalive.C:
				talk.Close()
				talk, err = xmpp.NewClient("talk.google.com:443",
					services.Config.Jabber.Jid,
					services.Config.Jabber.Pass,
					debug)
				if err != nil {
					log.Fatal(err)
				}
				go recvChannel(talk, ret.recv)
			case chat := <-ret.send:
				talk.Send(chat)
			}

		}
	}()

	return &ret, nil
}

func (self *JabberClient) Recv() interface{} {
	return <-self.recv
}

func (self *JabberClient) Send(chat xmpp.Chat) {
	self.send <- chat
}

// Run the service
func (self *Service) Run() error {
	client, err := NewClient()
	if err != nil {
		log.Fatal(err)
	}

	presence := map[string]string{}

	go func() {
		for {
			chat := client.Recv()
			switch v := chat.(type) {
			case xmpp.Chat:
				if v.Text == "" || IsSelf(v.Remote) {
					continue
				}

				log.Println("Query:", v.Text)
				services.SendQuery(v.Text, "jabber", v.Remote, "alert")
			case xmpp.Presence:
				// ignore self
				if !IsSelf(v.From) {
					presence[v.From] = v.Show
					log.Println("Presence:", v.From, v.Show)
				}
			}
		}
	}()

	events := services.Subscriber.FilteredChannel("alert")
	for ev := range events {
		if ev.Target() != "jabber" {
			continue
		}

		remote := ev.StringField("remote")
		source := ev.Source()
		message := ev.StringField("message")
		if remote == "" {
			// pick first match
			keys := []string{}
			for remote := range presence {
				keys = append(keys, remote)
			}
			sort.Strings(keys)

			if len(keys) == 0 {
				log.Println("No jabber users online to send to")
				continue
			}
			remote = keys[0]
		}
		text := message
		if source != "" {
			if strings.Contains(message, "\n") {
				text = fmt.Sprintf("%s>\n%s", source, message)
			} else {
				text = fmt.Sprintf("%s> %s", source, message)
			}
		}
		client.Send(xmpp.Chat{Remote: remote, Type: "chat", Text: text})
		log.Printf("Sent '%s' to %s", text, remote)
	}
	return nil
}
