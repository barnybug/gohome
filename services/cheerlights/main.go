// Service to listen to cheerlights and send commands.
package cheerlights

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

// Service cheerlights
type Service struct {
	state    State
	commands chan *Command
}

var Colours = map[string]string{
	// standard cheerlights
	"red":       "red",
	"green":     "green",
	"blue":      "blue",
	"cyan":      "cyan",
	"white":     "white",
	"oldlace":   "white",
	"warmwhite": "white",
	"purple":    "purple",
	"magenta":   "magenta",
	"yellow":    "yellow",
	"orange":    "orange",
	"pink":      "violet",

	// extras
	"flame":      "flame",
	"tangerine":  "tangerine",
	"aquamarine": "aquamarine",
	"turquoise":  "turquoise",
	"celeste":    "celeste",
	"violet":     "violet",
	"orchid":     "orchid",

	// some aliases
	"mauve": "orchid",
	"aqua":  "aquamarine",
	"gold":  "yellow",
}

type Command struct {
	Colours []string
}

func (self *Command) String() string {
	return strings.Join(self.Colours, " ")
}

type State struct {
	Colour string
	Mode   string
}

func (self State) Transition(previous State) []string {
	var codes []string
	if previous.Colour != self.Colour {
		codes = append(codes, self.Colour)
	}
	if previous.Mode != self.Mode || previous.Colour != self.Colour {
		codes = append(codes, self.Mode)
	}
	return codes
}

func newClient() *twittergo.Client {
	auth := services.Config.Twitter.Auth
	oclient := &oauth1a.ClientConfig{
		ConsumerKey:    auth.Consumer_key,
		ConsumerSecret: auth.Consumer_secret,
	}
	ouser := oauth1a.NewAuthorizedConfig(
		auth.Token,
		auth.Token_secret)
	return twittergo.NewClient(oclient, ouser)
}

func tracker(tweets chan *twittergo.Tweet) {
	client := newClient()

	query := url.Values{}
	query.Set("track", "cheerlights")

	for {
		// connect (or reconnect)
		log.Println("Connecting...")
		url := fmt.Sprintf("https://stream.twitter.com/1.1/statuses/filter.json?%v", query.Encode())
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Could not parse request: %v\n", err)
			break
		}
		resp, err := client.SendRequest(req)
		if err != nil {
			log.Println("Tracking failed, sleeping for 1 minute", err)
			time.Sleep(1 * time.Minute)
			continue
		}
		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Printf("Tracking failed, sleeping for 1 minute HTTP code: %d body: %s", resp.StatusCode, body)
			time.Sleep(1 * time.Minute)
			continue
		}
		log.Println("Connected")

		// read stream
		reader := bufio.NewScanner(resp.Body)
		for reader.Scan() {
			line := reader.Text()
			if line == "" {
				// keepalives
				continue
			}
			tweet := &twittergo.Tweet{}
			err := json.Unmarshal([]byte(line), tweet)
			if err != nil {
				log.Printf("Failed decoding tweet: %s [%s], reconnecting...", err, line)
				break
			}
			tweets <- tweet
		}
		log.Println("Disconnecting...")
		resp.Body.Close()
	}
}

func createRegex() *regexp.Regexp {
	tokens := []string{}
	for token := range Colours {
		tokens = append(tokens, token)
	}
	str := strings.Join(tokens, "|")
	str = fmt.Sprintf(`\b(%s)\b`, str)
	return regexp.MustCompile(str)
}

var reColour = createRegex()

func parseMessage(text string) *Command {
	text = strings.ToLower(text)
	matches := reColour.FindAllString(text, -1)
	cols := []string{}
	for _, match := range matches {
		colour := Colours[match]
		colour = strings.ToUpper(colour)
		cols = append(cols, colour)
	}
	if len(cols) == 0 {
		return nil
	}
	return &Command{Colours: cols}
}

func tweetListener(tweets chan *twittergo.Tweet, commands chan *Command) {
	for tweet := range tweets {
		log.Printf("%s: %s", tweet.User().ScreenName(), tweet.Text())

		command := parseMessage(tweet.Text())
		if command != nil {
			commands <- command
		}
	}
}

func (self *Service) setState(state State) {
	for _, code := range state.Transition(self.state) {
		ev := pubsub.NewCommand("light.cheer", code)
		services.Publisher.Emit(ev)
	}
	self.state = state
}

func (self *Service) commandSender(commands chan *Command) {
	// initially nil (ie not ticking)
	tick := time.NewTicker(time.Second)
	timer := time.NewTimer(time.Minute * 5)
	var seq []string
	offset := 0

	for {
		select {
		case command := <-commands:
			log.Printf("*** command: %s", command)
			seq = command.Colours
			offset = 0

			if len(seq) == 1 {
				// if just one colour, then use fade effect
				self.setState(State{seq[0], "FADE"})
			} else {
				self.setState(State{seq[0], ""})
			}
			// reset timers
			tick.Stop()
			tick = time.NewTicker(time.Second)
			timer.Stop()
			timer = time.NewTimer(time.Minute * 5)
		case <-tick.C:
			if len(seq) > 1 {
				offset += 1
				if offset >= len(seq) {
					offset = 0
				}
				self.setState(State{seq[offset], ""})
			}
		case <-timer.C:
			// when the timer expires, revert to flash flash (smooth)
			self.setState(State{"flash", "flash"})
			seq = []string{}
		}

	}
}

func (self *Service) ID() string {
	return "cheerlights"
}

func (self *Service) Run() error {
	tweets := make(chan *twittergo.Tweet)
	self.commands = make(chan *Command)
	go tracker(tweets)
	go tweetListener(tweets, self.commands)
	self.commandSender(self.commands)
	return nil
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"cheer": services.TextHandler(self.query),
		"help": services.StaticHandler("" +
			"cheer: colours\n"),
	}
}

func (self *Service) query(q services.Question) string {
	command := parseMessage(q.Args)
	if command != nil {
		self.commands <- command
		return "done"
	} else {
		return "not understood"
	}
}
