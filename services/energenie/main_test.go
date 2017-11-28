package energenie

import (
	"testing"

	"github.com/barnybug/ener314"
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/dummy"
	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
)

func TestInterfaces(t *testing.T) {
	var _ services.Service = (*Service)(nil)
	var _ services.Queryable = (*Service)(nil)
}

func TestQueryStatus(t *testing.T) {
	service := &Service{}
	q := services.Question{"status", "", ""}
	assert.Equal(t, "Queue:", service.queryStatus(q))
}

func TestQueryIdentify(t *testing.T) {
	services.Config = config.ExampleConfig
	service := &Service{}
	service.Initialize()
	q := services.Question{"identify", "", ""}
	assert.Equal(t, "Sensor name required", service.queryIdentify(q))

	q2 := services.Question{"identify", "x", ""}
	assert.Equal(t, "Sensor not found", service.queryIdentify(q2))

	q3 := services.Question{"identify", "trv.living", ""}
	assert.Equal(t, "Identify queued to sensor: 00097f", service.queryIdentify(q3))
}

func TestSending(t *testing.T) {
	services.Config = config.ExampleConfig
	services.Publisher = &dummy.Publisher{}
	service := &Service{}
	service.Initialize()
	var sent []SensorRequest
	service.sender = func(sensorId uint32, request SensorRequest) {
		sent = append(sent, request)
	}

	ev := pubsub.NewEvent("thermostat", pubsub.Fields{
		"device": "thermostat.living",
		"trv":    17.3,
		"target": 17.0,
	})
	service.handleThermostat(ev)

	msg := ener314.Message{
		SensorId: 0x00097f,
		Records:  []ener314.Record{ener314.Temperature{Value: 16}},
	}
	service.handleMessage(&msg)
	assert.Len(t, sent, 1)
	assert.Equal(t, TargetTemperature, sent[0].Action) // 1st
	assert.Equal(t, 17.3, sent[0].Temperature)

	ev2 := pubsub.NewEvent("command", pubsub.Fields{
		"device":  "thermostat.living",
		"command": "identify",
	})
	service.handleCommand(ev2)

	service.handleMessage(&msg)
	assert.Len(t, sent, 3)
	assert.Equal(t, Identify, sent[1].Action)          // 2nd
	assert.Equal(t, TargetTemperature, sent[2].Action) // 3rd
	assert.Equal(t, 17.3, sent[2].Temperature)

	service.handleMessage(&msg)
	assert.Len(t, sent, 4)
	assert.Equal(t, TargetTemperature, sent[3].Action) // 4th
	assert.Equal(t, 17.3, sent[3].Temperature)

	service.handleMessage(&msg)

	assert.Len(t, sent, 4)
}
