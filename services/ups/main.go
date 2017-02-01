// Service to collect stats from APC UPS devices.
package ups

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/mdlayher/apcupsd"
)

// Service ups
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "ups"
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": services.TextHandler(self.queryStatus),
		"help": services.StaticHandler("" +
			"status: get status\n"),
	}
}

func (self *Service) queryStatus(q services.Question) string {
	c, err := apcupsd.Dial("tcp", "127.0.0.1:3551")
	if err != nil {
		return fmt.Sprint("Failed to connect to apcupsd:", err)
	}
	status, err := c.Status()
	if err != nil {
		return fmt.Sprint("Failed to get status from apcupsd:", err)
	}
	return fmt.Sprintf(
		"Model:           %s\nStatus:          %s\nMains Voltage:   %.1fV\nLoad:            %.1f%%\nBattery Charge:  %.1f%%\nTime Left:       %s\nBattery Voltage: %.1fV\nTime on Battery: %s",
		status.Model,
		status.Status,
		status.LineVoltage,
		status.LoadPercent,
		status.BatteryChargePercent,
		status.TimeLeft,
		status.BatteryVoltage,
		status.TimeOnBattery)
}

// Run the service
func (self *Service) Run() error {
	for {
		c, err := apcupsd.Dial("tcp", "127.0.0.1:3551")
		if err != nil {
			log.Fatalln("Failed to connect to apcupsd:", err)
		}
		status, err := c.Status()
		if err != nil {
			log.Fatalln("Failed to get status from apcupsd:", err)
		}

		fields := pubsub.Fields{
			"source":   status.SerialNumber,
			"status":   status.Status,
			"linev":    status.LineVoltage,
			"loadpct":  status.LoadPercent,
			"bcharge":  status.BatteryChargePercent,
			"battv":    status.BatteryVoltage,
			"numxfers": status.NumberTransfers,
			"tonbatt":  status.TimeOnBattery.Seconds(),
			"selftest": status.Selftest,
		}
		ev := pubsub.NewEvent("ups", fields)
		services.Publisher.Emit(ev)
		c.Close()

		time.Sleep(time.Minute)
	}
	return nil
}
