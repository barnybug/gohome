// Service to hook into lirc and send IR codes.
package lirc

import (
	"log"

	"github.com/barnybug/gohome/services"
	"github.com/chbmuc/lirc"
)

// Service lirc
type Service struct{}

func (self *Service) ID() string {
	return "lirc"
}

func (self *Service) Run() error {
	ir, err := lirc.Init("/var/run/lirc/lircd")
	if err != nil {
		return err
	}

	go ir.Run()

	for ev := range services.Subscriber.FilteredChannel("command") {
		device := ev.Device()
		code := ev.Command()
		if remote, ok := services.Config.LookupDeviceProtocol(device, "lirc"); ok {
			ir.Send(remote + " " + code)
			if err != nil {
				log.Printf("Error sending command: %s", err)
			}
		}
	}
	return nil
}
