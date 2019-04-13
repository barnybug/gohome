// Service for providing daily estimates of electricity and gas bills.
package bills

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/lib/graphite"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

func tweet(message string, subtopic string, interval int64) {
	log.Println("Sending tweet", message)
	services.SendAlert(message, "twitter", subtopic, interval)
}

func daily(t time.Time) {
	// send daily stats
	g := graphite.NewQuerier(services.Config.Graphite.Url)
	var parts []string
	for _, device := range services.Config.DevicesByCap("power") {
		s := electricityBill(g, device)
		if s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) > 0 {
		msg := strings.Join(parts, ". ")
		tweet(msg, "bill", 0)
	}
}

func getHourlyTotals(g graphite.Querier, metric string) []graphite.Datapoint {
	// get hourly average usage
	target := fmt.Sprintf(`derivative(smartSummarize(%s,"1h","last"))`, metric)
	data, err := g.Query("midnight-25h", "midnight", target)
	if err != nil {
		log.Println("Failed to get graphite data")
		return nil
	}
	dps := data[0].Datapoints
	return dps[1:]
}

func electricityBill(g graphite.Querier, device config.DeviceConf) string {
	vat := services.Config.Bill.Vat/100 + 1
	series := fmt.Sprintf("sensor.%s.total.avg", device.Id)
	dps := getHourlyTotals(g, series)
	var max, total, day, night float64
	var peak time.Time
	for _, dp := range dps {
		total += dp.Value
		if dp.Value > max {
			peak = dp.At
			max = dp.Value
		}
		if dp.At.Hour() >= 6 && dp.At.Hour() < 18 {
			day += dp.Value
		} else {
			night += dp.Value
		}
	}
	// cost in currency units
	units := total / 1000 // kwh
	cost := ((units * services.Config.Bill.Electricity.Primary_Rate) + services.Config.Bill.Electricity.Standing_Charge) * vat

	msg := fmt.Sprintf("%s: yesterday I used %.2f kwh (%.2f day / %.2f night), costing %s%.2f. Peak was around %s.",
		device.Name, units, day/1000, night/1000, services.Config.Bill.Currency, cost/100,
		peak.Format(time.Kitchen))
	return msg
}

// Service bills
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "bills"
}

// Run the service
func (self *Service) Run() error {
	// schedule at 00:02
	offset, _ := time.ParseDuration("2m")
	repeat, _ := time.ParseDuration("24h")
	ticker := util.NewScheduler(offset, repeat)
	for t := range ticker.C {
		daily(t)
	}
	return nil
}

func (self *Service) Publishes() {
}
