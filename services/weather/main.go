// Package weather is a service to alert a daily digest of the last day's weather conditions, and
// actively alert on unusual conditions (heavy rain, strong winds).
//
// This requires the graphite service to be recording event data.
package weather

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/lib/graphite"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

type td struct {
	temp float64
	noun string
}

var lowTemperatures = []td{
	td{-5, "a very cold"},
	td{-2, "a rather cold"},
	td{0, "a freezing"},
	td{2, "a frosty"},
	td{5, "a cold"},
	td{7, "a moderate"},
	td{10, "a pleasant"},
	td{15, "a hot"},
	td{25, "a scorching"},
}

var highTemperatures = []td{
	td{1, "a very cold"},
	td{4, "a rather cold"},
	td{6, "a piercing"},
	td{8, "a chilly"},
	td{11, "a cool"},
	td{15, "a moderate"},
	td{18, "a reasonably warm"},
	td{21, "a hot"},
	td{31, "a scorching"},
	td{36, "a sweltering"},
}

var lastRainTotal, lastOutsideTemp, lastOutsideHumd, avgWind float64

func tweet(message string, subtopic string, interval int64) {
	log.Println("Sending tweet", message)
	services.SendAlert(message, "twitter", subtopic, interval)
}

// Lookup descriptive text for given temperate range
func getTempDesc(t float64, temps []td) string {
	for _, temp := range temps {
		if t < temp.temp {
			return temp.noun
		}
	}
	return ""
}

// Generate weather message for yesterday
func weatherStats(g graphite.Querier) string {
	sensor := services.Config.Weather.Sensors.Temp + ".temp"
	highest := getLast24(g, sensor, "max")
	highestDesc := getTempDesc(highest, highTemperatures)
	lowest := getLast24(g, sensor, "min")
	lowestDesc := getTempDesc(lowest, lowTemperatures)
	if lowest == 0 && highest == 0 {
		return "Weather: I didn't get any outside temperature data yesterday!"
	}
	return fmt.Sprintf("Weather: Outside it got up to %s %.1f°C and went down to %s %.1f°C in the last 24 hours.",
		highestDesc, highest,
		lowestDesc, lowest)
}

// Get last 24 hour temperature min/max
func getLast24(g graphite.Querier, sensor string, cf string) float64 {
	target := fmt.Sprintf(`summarize(sensor.%s.%s,"100y","%s")`, sensor, cf, cf)
	data, err := g.Query("-24h", "now", target)
	if err != nil {
		log.Println("Failed to get graphite data")
		return 0.0
	}
	return data[0].Datapoints[0].Value
}

func tick() {
	// send weather stats
	g := graphite.NewQuerier(services.Config.Graphite.Url)
	msg := weatherStats(g)
	tweet(msg, "daily", 0)
}

// Service weather
type Service struct{}

// ID of the service
func (service *Service) ID() string {
	return "weather"
}

// Run the service
func (service *Service) Run() error {
	// schedule at 08:00
	offset, _ := time.ParseDuration("8h")
	repeat, _ := time.ParseDuration("24h")
	ticker := util.NewScheduler(offset, repeat)
	for {
		select {
		case <-ticker.C:
			tick()
		}
	}
}
