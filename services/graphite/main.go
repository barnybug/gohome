// Service to write event data to graphite.
package graphite

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/barnybug/gohome/lib/graphite"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

var (
	gr *graphite.Graphite
)

var GraphiteAggs = []string{"avg", "max", "min"}

var ignoredFields = map[string]bool{
	"topic":     true,
	"timestamp": true,
	"source":    true,
	"sensor":    true,
	"origin":    true,
	"device":    true,
}

func sendToGraphite(ev *pubsub.Event) {
	device := services.Config.LookupDeviceName(ev)
	if device == "" {
		return
	}
	if _, ok := services.Config.Devices[device]; !ok {
		return
	}

	// drop type from device name
	p := strings.Split(device, ".")
	device = p[len(p)-1]
	timestamp := time.Now().UTC().Unix()
	for metric, value := range ev.Fields {
		if ignoredFields[metric] {
			continue
		}

		var floatValue float64
		switch value.(type) {
		case bool:
			if value == true {
				floatValue = 1
			} else {
				floatValue = 0
			}
		case uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:
			floatValue = value.(float64)
		default:
			//log.Printf("Ignoring non-numeric value: %s:%s %v\n", device, metric, value)
			continue
		}

		for _, x := range GraphiteAggs {
			path := fmt.Sprintf("sensor.%s.%s.%s", device, metric, x)
			gr.Add(path, timestamp, floatValue)
		}

	}

	if err := gr.Flush(); err != nil {
		log.Println("Flush failed:", err)
	}
}

// Service graphite
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "graphite"
}

// Run the service
func (self *Service) Run() error {
	gr = graphite.New(services.Config.Graphite.Host)
	for ev := range services.Subscriber.Channel() {
		sendToGraphite(ev)
	}
	return nil
}
