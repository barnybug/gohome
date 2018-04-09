// Service to communicate with yeelight light bulbs.
package yeelight

import (
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/edgard/yeelight"
)

// Service yeelight
type Service struct {
	lights map[string]*yeelight.Light
}

func (self *Service) ID() string {
	return "yeelight"
}

var reHexCode = regexp.MustCompile(`^#[0-9a-f]{6}$`)

func (self *Service) handleCommand(ev *pubsub.Event) {
	dev := ev.Device()
	ident, ok := services.Config.LookupDeviceProtocol(dev, "yeelight")
	if !ok {
		return // command not for us
	}
	command := ev.Command()
	if command != "off" && command != "on" {
		log.Println("Command not recognised:", command)
		return
	}
	if light, ok := self.lights[ident]; ok {
		log.Printf("Setting device %s to %s\n", dev, command)
		duration := 500
		if _, ok := ev.Fields["duration"]; ok {
			duration = int(ev.IntField("duration"))
		}

		switch ev.Command() {
		case "on":
			light.PowerOn(duration)
			level := int(ev.IntField("level"))
			if level != 0 {
				light.SetBrightness(level, duration)
			}
			colour := ev.StringField("colour")
			if reHexCode.MatchString(colour) {
				decoded, _ := hex.DecodeString(colour[1:])
				red := int(decoded[0])
				green := int(decoded[1])
				blue := int(decoded[2])
				light.SetRGB(red, green, blue, duration)
			}
			temp := int(ev.IntField("temp"))
			if temp != 0 {
				light.SetTemp(temp, duration)
			}
		case "off":
			light.PowerOff(duration)
		}
		light.Update()
		fields := pubsub.Fields{
			"device":  dev,
			"command": command,
			"level":   light.Bright,
			"temp":    light.ColorTemp,
		}
		ev := pubsub.NewEvent("ack", fields)
		services.Publisher.Emit(ev)
	} else {
		log.Println("Device not recognised:", dev)
	}
}

func announce(light yeelight.Light) {
	source := fmt.Sprintf("yeelight.%s", light.ID)
	fields := pubsub.Fields{
		"source":   source,
		"name":     light.Name,
		"location": light.Location,
	}
	ev := pubsub.NewEvent("announce", fields)
	services.Publisher.Emit(ev)
}

func (self *Service) discover() int {
	lights, err := yeelight.Discover(10 * time.Second)
	if err != nil {
		log.Fatal(err)
	}
	for i := range lights {
		light := lights[i]
		self.lights[light.ID] = &light
		source := fmt.Sprintf("yeelight.%s", light.ID)
		if _, ok := services.Config.LookupSource(source); !ok {
			log.Printf("Announcing new yeelight discovered: %s %s", light.ID, light.Location)
			announce(light)
		}
	}
	return len(lights)
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"discover": services.TextHandler(self.queryDiscover),
		"help":     services.StaticHandler("discover: run discovery\n"),
	}
}

func (self *Service) queryDiscover(q services.Question) string {
	devices := self.discover()
	return fmt.Sprintf("Discovered %d devices", devices)
}

func (self *Service) Run() error {
	commandChannel := services.Subscriber.FilteredChannel("command")
	self.lights = map[string]*yeelight.Light{}
	self.discover()
	log.Printf("Discovered %d lights", len(self.lights))
	// Rescan for new devices every hour
	autoDiscover := time.Tick(60 * time.Minute)

	for {
		select {
		case <-autoDiscover:
			self.discover()

		case command := <-commandChannel:
			self.handleCommand(command)
		}
	}
}
