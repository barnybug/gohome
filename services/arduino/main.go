// Service to communicate with serial devices (such as an arduino) for controlling
// relays. This could be used to hook in wired devices such as electric door locks,
// burglar alarm boxes, etc.
//
// The arduino sketch / wiring left as an exercise for the reader. Sketch simply
// switches a bank of relays based on received ASCII code: 65=relay #0 on,
// 66=relay #0 off, 67=relay #1 on, etc.
package arduino

import (
	"flag"
	"io"
	"log"
	"path/filepath"

	"github.com/barnybug/gohome/services"

	"github.com/tarm/goserial"
)

func defaultDevName() string {
	matches, _ := filepath.Glob("/dev/arduino_*")
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func command(dev io.Writer, code string, state bool) {
	// code = on, ascii(code) + 1 = off
	if !state {
		code = string(code[0] + 1)
	}
	log.Println("Sending:", code, state)
	dev.Write([]byte(code))
}

// Service arduino
type Service struct {
	devname string
}

// ID of the service
func (self *Service) ID() string {
	return "arduino"
}

// Run the service
func (self *Service) Run() error {
	c := &serial.Config{Name: self.devname, Baud: 9600}
	dev, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalln("Opening serial port:", err)
	}

	for ev := range services.Subscriber.FilteredChannel("command") {
		p := services.Config.LookupDeviceProtocol(ev.Device())
		code, ok := p["arduino"]
		if ok {
			command(dev, code, ev.Fields["state"].(bool))
		}
	}
	return nil
}

func (self *Service) Flags() {
	flag.StringVar(&self.devname, "d", defaultDevName(), "arduino device")
}
