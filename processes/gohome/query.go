package main

import (
	"fmt"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func query(args []string) {
	if len(args) == 0 {
		Usage()
	}
	q := strings.Join(args[1:], " ")
	u := url.Values{"q": {q}}

	uri := fmt.Sprintf("%s/query/%s?%s",
		services.Config.Endpoints.Api, args[0], u.Encode())
	resp, err := http.Get(uri)
	if err != nil {
		log.Fatalf("error: %s\n", err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error: %s\n", err)
	}

	parts := strings.Split(string(data), "\n")
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		ev := pubsub.Parse(part)
		if ev == nil {
			continue
		}
		source := ev.Source()
		message := ev.StringField("message")

		if strings.Contains(message, "\n") {
			fmt.Printf("\x1b[32;1m%s\x1b[0m\n%s\n", source, message)
		} else {
			fmt.Printf("\x1b[32;1m%s\x1b[0m %s\n", source, message)
		}
	}
}
