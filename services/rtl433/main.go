// Service to run rtl_433 and translate the output to sensor data.
package rtl433

import (
	"encoding/json"
	"fmt"
	"time"

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
	"CurrentCost-EnviR": "power",
	"Oregon-CM180":      "power",
	"Oregon-THGR122N":   "temp",
	"Oregon-THN132N":    "temp",
	"Nexus-TH":          "temp",
	"TFA-TwinPlus":      "temp",
	"Nexa-Security":     "sensor",
}
var fieldMap = map[string]string{
	"battery_ok":    "battery",
	"power0_W":      "power",
	"power1_W":      "power2",
	"power2_W":      "power3",
	"energy_kWh":    "total",
	"temperature_C": "temp",
	"humidity":      "humidity",
}
var skipFields = map[string]bool{
	"brand":  true,
	"model":  true,
	"id":     true,
	"time":   true,
	"mic":    true,
	"power1": true,
	"power2": true,
}

func translateEvent(data map[string]interface{}) *pubsub.Event {
	model, _ := data["model"].(string)
	id := int(data["id"].(float64))
	source := fmt.Sprintf("%s.%d", model, id)
	topic := "rtl433"
	if t, ok := modelMap[model]; ok {
		topic = t
	}
	fields := pubsub.Fields{
		"source": source,
	}
	for key, value := range data {
		if skipFields[key] {
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
	if timestamp, err := time.Parse("2006-01-02 15:04:05", data["time"].(string)); err == nil {
		ev.Timestamp = timestamp.UTC()
	}
	services.Config.AddDeviceToEvent(ev)
	return ev
}

var dedup [10]string
var dedup_position int

func isDuplicate(message MQTT.Message) bool {
	payload := string(message.Payload())

	for i := 0; i < len(dedup); i++ {
		if dedup[i] == payload {
			return true
		}
	}
	dedup[dedup_position] = payload
	dedup_position = (dedup_position + 1) % len(dedup)
	return false
}

func emit(data map[string]interface{}) {
	ev := translateEvent(data)
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
		if !isDuplicate(msg) {
			data := parse(msg.Payload())
			emit(data)
		}
	})

	select {}
}
