// Service to communicate with serial devices (such as an arduino) for controlling
// relays. This could be used to hook in wired devices such as electric door locks,
// burglar alarm boxes, etc.
package raspi

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/ener314/rpio"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
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
	for _, dev := range services.Config.DevicesByProtocol("raspi") {
		if n, ok := PinAliases[dev.Source]; ok {
			dir := PinDirection[dev.Source]
			pin := rpio.Pin(n)
			if dir == rpio.Input {
				pin.Input()
				pin.PullOff()
				go interruptListener(pin, dev.Source, interrupts)
			} else {
				pin.Output()
			}
		} else {
			fmt.Println("Pin not recognised:", dev.Source)
		}
	}
}

func handleCommand(ev *pubsub.Event) {
	if ident, ok := services.Config.LookupDeviceProtocol(ev.Device(), "raspi"); ok {
		if n, ok := PinAliases[ident]; ok {
			pin := rpio.Pin(n)
			state := rpio.Low
			if ev.Command() == "on" {
				state = rpio.High
			}
			log.Println("Switching", ident, state)
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
	source := fmt.Sprintf("raspi.%s", iv.name)
	fields := pubsub.Fields{
		"source":  source,
		"command": command,
	}
	ev := pubsub.NewEvent("raspi", fields)
	services.Config.AddDeviceToEvent(ev)
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
	commands := services.Subscriber.Subscribe(pubsub.Prefix("command"))

	for {
		select {
		case ev := <-commands:
			handleCommand(ev)
		case iv := <-interrupts:
			handleInterrupt(iv)
		}
	}

	return nil
}
