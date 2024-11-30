package heating

import (
	"fmt"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/dummy"
	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
)

var (
	evCold       = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 10.1, "timestamp": "2014-01-04 16:00:00.000"})
	evBorderline = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 14.2, "timestamp": "2014-01-04 16:10:00.000"})
	evHot        = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 19.0, "timestamp": "2014-01-04 16:31:00.000"})

	evAfterParty = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 19.0, "timestamp": "2014-01-04 17:10:00.000"})
)
var (
	timeLater  = evHot.Timestamp
	timeLater2 = evBorderline.Timestamp
	timeParty  = time.Date(2014, 1, 4, 16, 31, 0, 0, time.UTC)
)
var configYaml = `
device: heater.boiler
zones:
  hallway: temp.hallway
minimum: 10
slop: 0.3
`
var (
	testConfig config.HeatingConf
	em         *dummy.Publisher
	service    *Service
)

func SetupTests() {
	config := config.ExampleConfig
	yaml.Unmarshal([]byte(configYaml), &testConfig)
	config.Heating = testConfig
	em = &dummy.Publisher{}
	service = &Service{
		config:    &services.ConfigService{Value: config},
		Publisher: em,
	}
	service.Init()
	service.Zones["hallway"].Target = 18
}

func setClock(t time.Time) {
	Clock = func() time.Time { return t }
}

func fire(ev *pubsub.Event) {
	// set Clock to event time - as events trigger check of heating
	setClock(ev.Timestamp)
	service.handleEvent(ev)
}

func TestOnOff(t *testing.T) {
	SetupTests()
	assert.False(t, service.State)

	fire(evCold)
	// should switch on
	assert.True(t, service.State)
	assert.Equal(t, 1, len(em.Events))
	em.Events = em.Events[:0]

	// should stay on at 14.2, within slop
	fire(evBorderline)
	assert.True(t, service.State)

	service.Heartbeat()
	assert.True(t, service.State)
	assert.Equal(t, 3, len(em.Events))
	em.Events = em.Events[:0]

	// should switch off
	fire(evHot)
	assert.False(t, service.State)
	assert.Equal(t, 1, len(em.Events))
}

func TestStaleTemperature(t *testing.T) {
	SetupTests()
	fire(evCold)
	// should start On
	assert.True(t, service.State)

	// should switch off due to stale temperature data
	setClock(timeLater)
	service.Heartbeat()
	assert.False(t, service.State)

	// and stay off
	setClock(timeLater2)
	service.Heartbeat()
	assert.False(t, service.State)
}

func TestPartyMode(t *testing.T) {
	SetupTests()
	fire(evHot)
	assert.False(t, service.State)

	setClock(timeParty)
	q := services.Question{Verb: "party", Args: "thermostat.hallway 20 30m"}
	service.queryParty(q)
	assert.True(t, service.State)

	fire(evAfterParty)
	assert.False(t, service.State)
}

func TestPartyModeAll(t *testing.T) {
	SetupTests()
	fire(evHot)
	assert.False(t, service.State)

	setClock(timeParty)
	q := services.Question{Verb: "party", Args: "all 20 30m"}
	service.queryParty(q)
	assert.True(t, service.State)
}

func TestPartyModeTempOnly(t *testing.T) {
	SetupTests()
	fire(evHot)
	assert.False(t, service.State)

	setClock(timeParty)
	q := services.Question{Verb: "party", Args: "20"}
	service.queryParty(q)
	assert.True(t, service.State)
}

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	var _ services.Queryable = (*Service)(nil)
	// Output:
}

func ExampleStatus() {
	SetupTests()
	fmt.Println(service.Status(evCold.Timestamp))
	fire(evCold)
	fmt.Println(service.Status(evBorderline.Timestamp))
	// Output:
	// Heating: false for unknown
	// hallway unknown [18.0°C]
	// Heating: true for 10m
	// hallway 10.1°C +0.0°C/hr at Jan  4 16:00:00 [18.0°C]*
}

func ExampleQueryStatusText() {
	SetupTests()
	fire(evCold)
	setClock(evBorderline.Timestamp)
	q := services.Question{"status", "", ""}
	fmt.Println(service.queryStatus(q).Text)
	// Output:
	// Heating: true for 10m
	// hallway 10.1°C +0.0°C/hr at Jan  4 16:00:00 [18.0°C]*
}

var testQueries = []struct {
	query    string
	response string
}{
	{
		"",
		"Required at least temperature",
	},
	{
		"abc",
		"Invalid temperature",
	},
	{
		"hallway 18",
		"Set hallway to 18°C for 30 minutes",
	},
	{
		"thermostat.hallway 18",
		"Set hallway to 18°C for 30 minutes",
	},
	{
		"hallway 18 xyz",
		"invalid duration",
	},
	{
		"hallway 18 1m",
		"Set hallway to 18°C for 1 minute",
	},
	{
		"hallway 18 1h",
		"Set hallway to 18°C for 1 hour",
	},
	{
		"hallway 18 24h",
		"Set hallway to 18°C for 1 day",
	},
	{
		"hallway -1",
		"Below minimum temperature",
	},
	{
		"hallway 30",
		"Above maximum temperature",
	},
}

func TestQueries(t *testing.T) {
	SetupTests()
	setClock(evBorderline.Timestamp)
	fire(evCold)
	for _, tt := range testQueries {
		t.Run(tt.query, func(t *testing.T) {
			actual := service.queryParty(services.Question{"party", tt.query, ""})
			assert.Equal(t, tt.response, actual)
		})
	}
}
