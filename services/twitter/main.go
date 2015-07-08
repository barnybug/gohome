// Service to hook into twitter and send and receive tweets. This allows the
// house to tweet alerts to you.
//
// TODO: DM receiving not implemented yet.
package twitter

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
)

var lastSubtopic = map[string]time.Time{}

func createClient() *twittergo.Client {
	a := services.Config.Twitter.Auth
	config := &oauth1a.ClientConfig{
		ConsumerKey:    a.Consumer_key,
		ConsumerSecret: a.Consumer_secret,
	}
	user := oauth1a.NewAuthorizedConfig(a.Token, a.Token_secret)
	return twittergo.NewClient(config, user)
}

func sendTweet(client *twittergo.Client, message string) {
	data := url.Values{}
	data.Set("status", message)
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest("POST", "/1.1/statuses/update.json", body)
	if err != nil {
		log.Printf("Could not parse request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.SendRequest(req)
	if err != nil {
		log.Printf("Could not send request: %v", err)
		return
	}
	tweet := &twittergo.Tweet{}
	err = resp.Parse(tweet)
	if err != nil {
		if rle, ok := err.(twittergo.RateLimitError); ok {
			log.Printf("Rate limited, reset at %v\n", rle.Reset)
		} else if errs, ok := err.(twittergo.Errors); ok {
			for i, val := range errs.Errors() {
				log.Printf("Error #%v - ", i+1)
				log.Printf("Code: %v ", val.Code())
				log.Printf("Msg: %v\n", val.Message())
			}
		} else {
			log.Printf("Problem parsing response: %v\n", err)
		}
	}
}

func tweet(client *twittergo.Client, ev *pubsub.Event) {
	msg, _ := ev.Fields["message"].(string)
	subtopic, _ := ev.Fields["subtopic"].(string)
	interval, _ := ev.Fields["interval"].(float64)
	if subtopic != "" && interval != 0 {
		now := time.Now()
		if last, ok := lastSubtopic[subtopic]; ok {
			if last.Add(time.Duration(interval) * time.Second).After(now) {
				log.Printf("Tweet surpressed: %v", msg)
				return
			}
		}
		lastSubtopic[subtopic] = now
	}

	log.Printf("Sending tweet: %s", msg)
	sendTweet(client, msg)
}

// Service twitter
type Service struct{}

func (self *Service) ID() string {
	return "twitter"
}

func (self *Service) Run() error {
	client := createClient()
	for ev := range services.Subscriber.FilteredChannel("alert") {
		if ev.Target() == "twitter" {
			tweet(client, ev)
		}
	}
	return nil
}
