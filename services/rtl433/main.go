// Service to run rtl_433 and translate the output to sensor data.
package rtl433

import (
	"encoding/json"
	"fmt"

	"github.com/barnybug/gohome/pubsub/mqtt"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Service rtl_433
type Service struct {
}

func (self *Service) ID() string {
	return "rtl433"
}

var modelMap = map[string]string{
	"CurrentCost-Sensable": "power",
	"Oregon-CM180":         "power",
	"Oregon-THGR122N":      "temp",
	"Oregon-THN132N":       "temp",
	"Nexus-TH":             "temp",
	"TFA-TwinPlus":         "temp",
}
var fieldMap = map[string]string{
	"battery_ok":    "battery",
	"power0":        "power",
	"power1":        "power2",
	"power2":        "power3",
	"power_W":       "power",
	"energy_kWh":    "total",
	"temperature_C": "temp",
	"humidity":      "humidity",
}

func translateEvent(data map[string]interface{}) *pubsub.Event {
	model, _ := data["model"].(string)
	source := fmt.Sprintf("%s.%v", model, data["id"])
	topic := "rtl433"
	if t, ok := modelMap[model]; ok {
		topic = t
	}
	fields := pubsub.Fields{
		"source": source,
	}
	for key, value := range data {
		if key == "brand" || key == "model" || key == "id" || key == "time" {
			continue
		}
		if value, ok := value.(float64); key == "humidity" && model == "TFA-TwinPlus" && ok {
			fields["rain"] = value + 28 // actually a rain gauge5555
		} else if to, ok := fieldMap[key]; ok {
			fields[to] = value
		} else {
			fields[key] = value // map unknowns as is
		}
	}
	ev := pubsub.NewEvent(topic, fields)
	// timezones
	// if timestamp, err := time.Parse("2006-01-02 15:04:05", data["time"].(string)); err == nil {
	// 	ev.Timestamp = timestamp
	// }
	services.Config.AddDeviceToEvent(ev)
	return ev
}

func emit(data map[string]interface{}) {
	ev := translateEvent(data)
	// TODO deduplicate
	services.Publisher.Emit(ev)
}

func parse(payload []byte) map[string]interface{} {
	var data map[string]interface{}
	err := json.Unmarshal(payload, &data)
	if err != nil {
		return nil
	}
	return data
}

func (self *Service) Run() error {
	mqtt.Client.Subscribe("rtl_433/#", 1, func(client MQTT.Client, msg MQTT.Message) {
		data := parse(msg.Payload())
		emit(data)
	})

	select {}
}
