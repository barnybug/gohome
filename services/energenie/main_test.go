package energenie

import (
	"testing"

	"github.com/barnybug/gohome/config"
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
