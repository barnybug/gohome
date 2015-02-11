// Service to publish events for tag inputs from a USB RFID (keyboard) reader.
//
// Warning: this 'grabs' the configured input device exclusively, so no other
// consoles will receive input from it anymore. Be sure you are grabbing the
// RFID reader, not the local keyboard.
package rfid

import (
	"github.com/barnybug/gohome/lib/evdev"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"log"
	"time"
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
	fields := map[string]interface{}{
		"origin":  "rfid",
		"command": "tag",
		"source":  code,
	}
	event := pubsub.NewEvent("rfid", fields)
	services.Publisher.Emit(event)
}

func readEvents(dev *evdev.InputDevice) {
	code := ""

	for {
		ev, err := dev.ReadOne()
		if err != nil {
			log.Fatal("Error reading: ", err)
			break
		}

		if ev.Type == 1 && ev.Value == 1 {
			ch := convertKeyCode(ev.Code)
			switch ch {
			case '\n':
				emit(code)
				code = ""
			default:
				code += string(ch)
			}
		}
	}
}

type RfidService struct{}

func (self *RfidService) Id() string {
	return "rfid"
}

func (self *RfidService) Run() error {
	for {
		devname := services.Config.Rfid.Device
		dev, err := evdev.Open(devname)
		if err != nil {
			log.Fatalf("Error opening device %s: %s", devname, err)
		}
		dev.Grab()
		log.Println("Connected")

		readEvents(dev)

		log.Println("Disconnected")
		dev.Close()
		time.Sleep(time.Second)
		log.Println("Reconnecting...")
	}
	return nil
}
