// Service to communicate with serial devices (such as an arduino) for controlling
// relays. This could be used to hook in wired devices such as electric door locks,
// burglar alarm boxes, etc.
package raspi

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/stianeikeland/go-rpio"
)

// Service raspi
type Service struct {
}

// ID of the service
func (self *Service) ID() string {
	return "raspi"
}

var PinAliases = map[string]int{
	// pimoroni automation hat aliases
	"relay1":  13,
	"relay2":  19,
	"relay3":  16, // <- the relay on the pHat
	"output1": 5,
	"output2": 12,
	"output3": 6,
	"input1":  26,
	"input2":  20,
	"input3":  21,
}

var PinDirection = map[string]rpio.Direction{
	"relay1":  rpio.Output,
	"relay2":  rpio.Output,
	"relay3":  rpio.Output,
	"output1": rpio.Output,
	"output2": rpio.Output,
	"output3": rpio.Output,
	"input1":  rpio.Input,
	"input2":  rpio.Input,
	"input3":  rpio.Input,
}

type InterruptEvent struct {
	name  string
	state rpio.State
}

const PollInterval = time.Millisecond * 100

func interruptListener(pin rpio.Pin, name string, interrupts chan InterruptEvent) {
	state := pin.Read()
	for {
		current := pin.Read()
		if current != state {
			state = current
			interrupts <- InterruptEvent{name, state}
		}
		time.Sleep(PollInterval)
	}
}

func setupPins(interrupts chan InterruptEvent) {
	for name, _ := range services.Config.Protocols["raspi"] {
		if n, ok := PinAliases[name]; ok {
			dir := PinDirection[name]
			pin := rpio.Pin(n)
			if dir == rpio.Input {
				pin.Input()
				pin.PullOff()
				go interruptListener(pin, name, interrupts)
			} else {
				pin.Output()
			}
		} else {
			fmt.Println("Pin not recognised:", name)
		}
	}
}

func handleCommand(ev *pubsub.Event) {
	p := services.Config.LookupDeviceProtocol(ev.Device())
	if name, ok := p["raspi"]; ok {
		if n, ok := PinAliases[name]; ok {
			pin := rpio.Pin(n)
			state := rpio.Low
			if ev.Command() == "on" {
				state = rpio.High
			}
			log.Println("Switching", name, state)
			pin.Write(state)
		}
	}
}

func handleInterrupt(iv InterruptEvent) {
	log.Println("Input", iv.name, "changed to", iv.state)
	command := "off"
	if iv.state == rpio.High {
		command = "on"
	}
	device := services.Config.Protocols["raspi"][iv.name]
	fields := map[string]interface{}{
		"device":  device,
		"command": command,
	}
	ev := pubsub.NewEvent("raspi", fields)
	services.Publisher.Emit(ev)
}

// Run the service
func (self *Service) Run() error {
	err := rpio.Open()
	if err != nil {
		log.Fatalln("Couldn't open /dev/gpiomem")
	}
	defer rpio.Close()

	interrupts := make(chan InterruptEvent, 10)

	setupPins(interrupts)

	for {
		select {
		case ev := <-services.Subscriber.FilteredChannel("command"):
			handleCommand(ev)
		case iv := <-interrupts:
			handleInterrupt(iv)
		}
	}

	return nil
}
