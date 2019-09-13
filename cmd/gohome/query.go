package main

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/barnybug/gohome/pubsub"
)

func fmtFatalf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	os.Exit(1)
}

func stream(path string, params url.Values) {
	resp, err := request(path, params)
	if err != nil {
		if strings.HasSuffix(err.Error(), " EOF") { // yuck
			fmtFatalf("Server disconnected\n")
		} else {
			fmtFatalf("error: %s\n", err)
		}
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)

	n := 0
	for scanner.Scan() {
		ev := pubsub.Parse(scanner.Text(), "")
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
		n += 1
	}
	if n == 0 {
		fmt.Println("No response")
	}
}

func request(path string, params url.Values) (*http.Response, error) {
	if os.Getenv("GOHOME_API") == "" {
		fmtFatalf("Set GOHOME_API to the gohome api url.")
	}
	// add http auth
	api := os.Getenv("GOHOME_API")
	uri := fmt.Sprintf("%s/%s", api, path)
	if len(params) > 0 {
		uri += "?" + params.Encode()
	}
	resp, err := http.Get(uri)
	return resp, err
}

func query(first string, rest []string, params url.Values) {
	q := strings.Join(rest, " ")
	u := url.Values{"q": {q}}
	for key, value := range params {
		u[key] = value
	}

	path := fmt.Sprintf("query/%s", first)
	stream(path, u)
}
