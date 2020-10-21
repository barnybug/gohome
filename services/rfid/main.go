// Service to publish events for tag inputs from a USB RFID (keyboard) reader.
//
// Warning: this 'grabs' the configured input device exclusively, so no other
// consoles will receive input from it anymore. Be sure you are grabbing the
// RFID reader, not the local keyboard.
package rfid

import (
	"fmt"
	"log"

	"github.com/barnybug/gohome/lib/evdev"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

func convertKeyCode(code uint16) uint16 {
	switch {
	case code >= 2 && code <= 10:
		return code - 2 + '1'
	case code == 11:
		return '0'
	case code == 96 || code == 28:
		return '\n'
	}
	return 0
}

func emit(code string) {
	log.Println("Publishing:", code)
	source := fmt.Sprintf("rfid.%s", code)
	fields := map[string]interface{}{
		"source":  source,
		"command": "on",
	}
	ev := pubsub.NewEvent("rfid", fields)
	services.Config.AddDeviceToEvent(ev)
	services.Publisher.Emit(ev)
}

func readEvents(dev *evdev.InputDevice) error {
	code := ""

	for {
		ev, err := dev.ReadOne()
		if err != nil {
			return err
		}

		if ev.Type == 1 && ev.Value == 1 {
			ch := convertKeyCode(ev.Code)
			switch ch {
			case '\n':
				emit(code)
				code = ""
			default:
				code += string(rune(ch))
			}
		}
	}
}

// Service rfid
type Service struct{}

func (self *Service) ID() string {
	return "rfid"
}

func (self *Service) Run() error {
	devname := services.Config.Rfid.Device
	dev, err := evdev.Open(devname)
	if err != nil {
		return err
	}
	defer dev.Close()

	err = dev.Grab()
	if err != nil {
		return err
	}
	log.Println("Connected")
	err = readEvents(dev)
	return err
}
