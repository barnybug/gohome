// Service for providing daily estimates of electricity and gas bills.
package bills

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/lib/graphite"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

var (
	gr graphite.IGraphite
)

func tweet(message string, subtopic string, interval int64) {
	log.Println("Sending tweet", message)
	services.SendAlert(message, "twitter", subtopic, interval)
}

func tick(t time.Time) {
	// send weather stats
	msg := electricityBill()
	if msg != "" {
		tweet(msg, "bill", 0)
	}
}

func getAvgYesterday(metric string) (total float64, peak time.Time) {
	// get hourly average usage
	target := fmt.Sprintf(`summarize(%s,"1h","avg")`, metric)
	data, err := gr.Query("midnight yesterday", "midnight", target)
	if err != nil {
		log.Println("Failed to get graphite data")
		return
	}
	dps := data[0].Datapoints[:24]
	max := 0.0
	for _, dp := range dps {
		total += dp.Value
		if dp.Value > max {
			peak = dp.At
			max = dp.Value
		}
	}
	return
}

func electricityBill() string {
	vat := services.Config.Bill.Vat/100 + 1
	total, peak := getAvgYesterday("sensor.power.power.avg")
	units := total / 1000 // kwh
	// cost in currency units
	cost := (units * services.Config.Bill.Electricity.Primary_Rate) * vat / 100
	cost += services.Config.Bill.Electricity.Standing_Charge / 365

	msg := fmt.Sprintf("Electricity: yesterday I used %.2f kwh, costing %s%.2f. Peak was around %s.",
		units, services.Config.Bill.Currency, cost,
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
	// initialise
	gr = graphite.New(services.Config.Graphite.Host)

	// schedule at 00:02
	offset, _ := time.ParseDuration("2m")
	repeat, _ := time.ParseDuration("24h")
	ticker := util.NewScheduler(offset, repeat)
	for t := range ticker.C {
		tick(t)
	}
	return nil
}

func (self *Service) Publishes() {
}
