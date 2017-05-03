// Service to send and receive SMS messages via a GSM modem/USB dongle. This
// allows you to text alerts to your phone on events in the house, and also to
// command the house by text.
//
// Tested with the ZTE MF110/MF627/MF636 (available some time under Three in the
// UK as a 3G dongle).
package sms

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	"github.com/barnybug/gogsmmodem"
	"github.com/tarm/serial"
)

var modem *gogsmmodem.Modem

func expandDevName() string {
	matches, _ := filepath.Glob(services.Config.SMS.Device)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func sendMessage(ev *pubsub.Event) {
	telephone := services.Config.SMS.Telephone
	if t, ok := ev.Fields["telephone"].(string); ok {
		telephone = t
	}
	if msg, ok := ev.Fields["message"].(string); ok {
		log.Printf("Sending to %s: %s", telephone, msg)
		modem.SendMessage(telephone, msg)
	}
}

// Service sms
type Service struct{}

func (self *Service) ID() string {
	return "sms"
}

func (self *Service) Run() error {
	// connect to modem
	devname := expandDevName()
	if devname == "" {
		log.Fatalln("Device not found")
	}
	conf := serial.Config{Name: devname, Baud: 115200}
	var err error
	modem, err = gogsmmodem.Open(&conf, false)
	if err != nil {
		return err
	}
	log.Println("Connected")

	log.Println("Checking message storage for unread")
	msgs, err := modem.ListMessages("ALL")
	if err == nil {
		for _, msg := range *msgs {
			if msg.Status == "REC UNREAD" {
				fmt.Printf("Message from %s: %s\n", msg.Telephone, msg.Body)
				services.SendQuery(msg.Body, "sms", msg.Telephone, "alert")
			}
			// delete - any unread have been read
			modem.DeleteMessage(msg.Index)
		}
	}

	log.Println("Ready")

	events := services.Subscriber.FilteredChannel("alert")
	for {
		select {
		case ev := <-events:
			if ev.Target() == "sms" {
				sendMessage(ev)
			}

		case p := <-modem.OOB:
			log.Printf("Received: %#v\n", p)
			switch p := p.(type) {
			case gogsmmodem.MessageNotification:
				msg, err := modem.GetMessage(p.Index)
				if err == nil {
					fmt.Printf("Message from %s: %s\n", msg.Telephone, msg.Body)
					services.SendQuery(msg.Body, "sms", msg.Telephone, "alert")
					modem.DeleteMessage(p.Index)
				}
			}

		}
	}
	return nil
}
