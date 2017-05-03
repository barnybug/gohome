// Service for capturing currentcost meter electricity data.
package currentcost

import (
	"bufio"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	"github.com/tarm/serial"
)

var rePower = regexp.MustCompile(`<ch1><watts>(\d{5})</watts>`)
var reTemp = regexp.MustCompile(`<tmpr>([0-9.]+)</tmpr>`)

func parse(msg string) *pubsub.Event {
	m := rePower.FindStringSubmatch(msg)
	if m != nil {
		power, _ := strconv.ParseInt(m[1], 10, 32)
		temp := 0.0
		m := reTemp.FindStringSubmatch(msg)
		if m != nil {
			temp, _ = strconv.ParseFloat(m[1], 64)
		}

		fields := map[string]interface{}{
			"source": "cc01",
			"power":  power,
			"temp":   temp,
		}
		return pubsub.NewEvent("power", fields)
	}
	return nil
}

// Service currentcost
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "currentcost"
}

func (self *Service) Run() error {
	device := services.Config.Currentcost.Device
	c := &serial.Config{Name: device, Baud: 2400}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalln("Opening serial port:", err)
	}
	log.Println("Connected")

	reader := bufio.NewReader(s)
	var buffer string
	for {
		line, err := reader.ReadString('\n')
		buffer += line
		if err != nil && err != io.EOF {
			log.Fatalln("Error reading line:", err, line, err == io.EOF)
		}
		if line == "" {
			// empty read, wait a bit
			time.Sleep(time.Millisecond * 500)
			continue
		}
		if !strings.HasSuffix(line, "\n") {
			// partial line
			continue
		}
		if line == "\n" {
			continue
		}
		ev := parse(string(buffer))
		if ev == nil {
			log.Println("Couldn't parse:", buffer)
		} else {
			services.Publisher.Emit(ev)
		}
		buffer = ""
	}
	return nil
}
