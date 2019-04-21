// Service to write event data to graphite.
package graphite

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/lib/graphite"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

var (
	gr *graphite.GraphiteWriter
)

var graphiteAggs = []string{"avg", "max", "min"}

var ignoredFields = map[string]bool{
	"topic":     true,
	"timestamp": true,
	"source":    true,
	"sensor":    true,
	"origin":    true,
	"device":    true,
	"repeat":    true,
}

var eventsTotal = map[string]int{}

func sendToGraphite(ev *pubsub.Event) {
	device := ev.Device()
	if _, ok := services.Config.Devices[device]; !ok {
		return
	}

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
		case string:
			if value == "off" {
				floatValue = 0
			} else if value == "on" {
				floatValue = 1
			} else {
				continue
			}
		default:
			// ignore non-numeric values
			continue
		}

		for _, x := range graphiteAggs {
			path := fmt.Sprintf("sensor.%s.%s.%s", device, metric, x)
			gr.Add(path, timestamp, floatValue)
		}

		// total events counter
		path := fmt.Sprintf("sensor.%s.%s.%s", device, metric, "total")
		eventsTotal[path]++
		total := eventsTotal[path]
		gr.Add(path, timestamp, float64(total))
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
	gr = graphite.NewWriter(services.Config.Graphite.Tcp)
	for ev := range services.Subscriber.Channel() {
		sendToGraphite(ev)
	}
	return nil
}
