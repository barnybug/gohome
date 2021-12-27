// Service to water the garden based on how warm it has been.
//
// This will schedule two watering cycles a day at am/pm, based on the
// temperature of an outdoor sensor for the last 12h. Tweets each time it waters
// so you can keep an eye on it!
package irrigation

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/barnybug/gohome/lib/graphite"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

func calculateDuration(g graphite.Querier) (duration time.Duration, avgTemp float64) {
	stat := fmt.Sprintf("sensor.%s.temp.avg", services.Config.Irrigation.Sensor)
	avgTemp = getLastN(g, "-12h", stat)

	// linear scale between min_temp - max_temp
	i := services.Config.Irrigation
	t := (avgTemp - i.Min_Temp) / (i.Max_Temp - i.Min_Temp)
	// limit to 0.0-1.0
	t = math.Max(math.Min(t, 1.0), 0.0)

	seconds := math.Pow(t, i.Factor)*(i.Max_Time-i.Min_Time) + i.Min_Time
	duration = time.Duration(seconds) * time.Second
	return
}

func getLastN(g graphite.Querier, from string, metric string) float64 {
	target := fmt.Sprintf(`summarize(%s,"1y","avg")`, metric)
	data, err := g.Query(from, "now", target)
	if err != nil {
		log.Println("Failed to get graphite data:", err)
		return 0.0
	}
	return data[0].Datapoints[0].Value
}

func tweet(message string, subtopic string, interval int64) {
	log.Println("Sending tweet", message)
	services.SendAlert(message, "twitter", subtopic, interval)
}

func irrigationStats(g graphite.Querier) (msg string, duration time.Duration) {
	duration, avgTemp := calculateDuration(g)
	if duration == 0 {
		msg = fmt.Sprintf("Irrigation: Not watering garden (12h av was %.1fC)", avgTemp)
	} else {
		msg = fmt.Sprintf("Irrigation: Watering garden for %s (12h av was %.1fC)", duration, avgTemp)
	}
	return
}

func run(t time.Time) {
	g := graphite.NewQuerier(services.Config.Graphite.Url)
	msg, duration := irrigationStats(g)
	log.Println(msg)
	tweet(msg, "irrigation", 0)

	// switch on
	command(services.Config.Irrigation.Device, true, 3)
	time.AfterFunc(duration, func() {
		// switch off
		command(services.Config.Irrigation.Device, false, 3)
	})
}

func command(device string, state bool, repeat int) {
	command := "off"
	if state {
		command = "on"
	}
	ev := pubsub.NewCommand(device, command)
	ev.SetRepeat(repeat)
	services.Publisher.Emit(ev)
}

// Service irrigation
type Service struct{}

func (self *Service) ID() string {
	return "irrigation"
}

// Run the service
func (self *Service) Run() error {
	// schedule at given time and interval
	now := time.Now()
	run(now)
	return nil
}
