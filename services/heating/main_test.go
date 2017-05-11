package heating

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/dummy"
	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.in/v1/yaml"
)

var (
	evOff        = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 10.1, "timestamp": "2014-01-04 10:19:00.000"})
	evCold       = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 10.1, "timestamp": "2014-01-04 16:00:00.000"})
	evColder     = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 9.0, "timestamp": "2014-01-04 16:00:00.000"})
	evBorderline = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 14.2, "timestamp": "2014-01-04 16:10:00.000"})
	evHot        = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 19.0, "timestamp": "2014-01-04 16:31:00.000"})

	evEmpty = pubsub.NewEvent("state", pubsub.Fields{"device": "house.presence", "state": "Empty", "timestamp": "2014-01-04 16:00:00.000"})
	evFull  = pubsub.NewEvent("state", pubsub.Fields{"device": "house.presence", "state": "Full", "timestamp": "2014-01-04 16:00:00.000"})

	evAfterParty = pubsub.NewEvent("temp", pubsub.Fields{"device": "temp.hallway", "temp": 19.0, "timestamp": "2014-01-04 17:10:00.000"})
)
var (
	timeLater  = evHot.Timestamp
	timeLater2 = evBorderline.Timestamp
	timeJustOn = time.Date(2014, 1, 4, 10, 20, 0, 0, time.UTC)
	timeParty  = time.Date(2014, 1, 4, 16, 31, 0, 0, time.UTC)
)
var configYaml = `
device: heater.boiler
slop: 0.3
zones:
  hallway:
    sensor: temp.hallway
    schedule:
      Saturday,Sunday:
        - 10:20-22:50: 18.0
      Monday-Friday:
        - 07:30-08:10: 18.0
        - 17:30-22:20: 18.0
minimum: 10.0
`
var (
	testConfig config.HeatingConf
	em         *dummy.Publisher
	service    *Service
)

func SetupStor() {
	// setup mock store
	services.Stor = services.NewMockStore()
}

func SetupService() {
	services.Config = config.ExampleConfig
	yaml.Unmarshal([]byte(configYaml), &testConfig)
	services.Config.Heating = testConfig
	em = &dummy.Publisher{}
	service = &Service{}
	service.Initialize(em)
	service.Event(evFull) // retained state
}

func Setup() {
	SetupStor()
	SetupService()
}

func TestOnOff(t *testing.T) {
	Setup()
	assert.False(t, service.State)

	service.Event(evCold)
	// should switch on
	assert.True(t, service.State)
	assert.Equal(t, 1, len(em.Events))
	em.Events = em.Events[:0]

	// should stay on at 14.2, within slop
	service.Event(evBorderline)
	assert.True(t, service.State)

	service.Heartbeat(evBorderline.Timestamp)
	assert.True(t, service.State)
	assert.Equal(t, 3, len(em.Events))
	em.Events = em.Events[:0]

	// should switch off
	service.Event(evHot)
	assert.False(t, service.State)
	assert.Equal(t, 1, len(em.Events))
}

func TestTimeChange(t *testing.T) {
	Setup()
	service.Event(evOff)
	assert.False(t, service.State)

	service.Heartbeat(timeJustOn)
	// should switch on
	assert.True(t, service.State)
}

func TestStaleTemperature(t *testing.T) {
	Setup()
	service.Event(evCold)
	// should start On
	assert.True(t, service.State)

	// should switch off due to stale temperature data
	service.Heartbeat(timeLater)
	assert.False(t, service.State)

	// and stay off
	service.Heartbeat(timeLater2)
	assert.False(t, service.State)
}

func TestTemperatureFromStateOn(t *testing.T) {
	SetupStor()
	mockTemp := `{"topic": "temp", "source": "wmr100.2", "temp": 10.1, "timestamp": "2014-01-04 16:00:00.000"}`
	services.Stor.Set("gohome/state/events/temp/temp.hallway", mockTemp)
	SetupService()

	service.Heartbeat(evCold.Timestamp)
	// should start On
	assert.True(t, service.State, "Expected State: true")
}

func TestTemperatureFromStateOff(t *testing.T) {
	SetupStor()
	mockTemp := `{"topic": "temp", "source": "wmr100.2", "temp": 18.0, "timestamp": "2014-01-04 16:00:00.000"}`
	services.Stor.Set("gohome/state/events/temp/temp.hallway", mockTemp)
	SetupService()

	service.Heartbeat(evCold.Timestamp)
	// should start Off
	assert.False(t, service.State, "Expected State: false")
}

func TestOccupiedToEmptyToFull(t *testing.T) {
	Setup()
	service.Event(evCold)
	assert.True(t, service.State)

	// should switch off due to house being empty
	service.Event(evEmpty)
	assert.False(t, service.State)

	// should switch on due to house being too cold
	service.Event(evColder)
	assert.True(t, service.State)

	// should switch on due to house being full
	service.Event(evFull)
	assert.True(t, service.State)
}

func TestPartyMode(t *testing.T) {
	Setup()
	service.Event(evHot)
	assert.False(t, service.State)

	Clock = func() time.Time { return timeParty }
	q := services.Question{Verb: "ch", Args: "thermostat.hallway 20 30m"}
	service.queryCh(q)
	assert.True(t, service.State)

	service.Event(evAfterParty)
	assert.False(t, service.State)
}

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	var _ services.Queryable = (*Service)(nil)
	// Output:
}

func ExampleStatus() {
	Setup()
	fmt.Println(service.Status(evCold.Timestamp))
	service.Event(evCold)
	fmt.Println(service.Status(evBorderline.Timestamp))
	// Output:
	// Heating: false for unknown
	// hallway unknown [18.0°C]
	// Heating: true for 10m
	// hallway 10.1°C at Jan  4 16:00:00 [18.0°C]*
}

func ExampleQueryStatusText() {
	Setup()
	Clock = func() time.Time { return evBorderline.Timestamp }
	service.Event(evCold)
	q := services.Question{"status", "", ""}
	fmt.Println(service.queryStatus(q).Text)
	// Output:
	// Heating: true for 10m
	// hallway 10.1°C at Jan  4 16:00:00 [18.0°C]*
}

func ExampleQueryStatusJson() {
	Setup()
	Clock = func() time.Time { return evBorderline.Timestamp }
	service.Event(evCold)
	q := services.Question{"status", "", ""}
	data := service.queryStatus(q).Json
	s, _ := json.Marshal(data)
	fmt.Println(string(s))
	// Output:
	// {"changed":"2014-01-04T16:00:00Z","devices":{"hallway":{"at":"2014-01-04T16:00:00Z","target":18,"temp":10.1}},"heating":true}
}

var testQueries = []struct {
	query    string
	response string
}{
	{
		"",
		"required at least device and temperature",
	},
	{
		"abc",
		"required at least device and temperature",
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
		"time: invalid duration xyz",
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
	Setup()
	Clock = func() time.Time { return evBorderline.Timestamp }
	service.Event(evCold)
	for _, tt := range testQueries {
		actual := service.queryCh(services.Question{"ch", tt.query, ""})
		assert.Equal(t, tt.response, actual)
	}
}

var testScheduleTable = []struct {
	t    time.Time
	temp float64
}{
	{
		time.Date(2014, 1, 6, 8, 0, 0, 0, time.UTC), // Monday 8am
		18.0,
	},
	{
		time.Date(2014, 1, 6, 8, 10, 0, 0, time.UTC), // Monday 8:10am
		14.0,
	},
	{
		time.Date(2014, 1, 6, 17, 20, 0, 0, time.UTC), // Monday 5:29pm
		14.0,
	},
	{
		time.Date(2014, 1, 6, 17, 30, 0, 0, time.UTC), // Monday 5:30pm
		18.0,
	},
	{
		time.Date(2014, 1, 6, 22, 19, 0, 0, time.UTC), // Monday 10:19pm
		18.0,
	},
	{
		time.Date(2014, 1, 10, 8, 0, 0, 0, time.UTC), // Friday 8am
		18.0,
	},
	{
		time.Date(2014, 1, 10, 8, 10, 0, 0, time.UTC), // Friday 8:10am
		14.0,
	},
	{
		time.Date(2014, 1, 4, 8, 0, 0, 0, time.UTC), // Saturday 8am
		0.0,
	},
	{
		time.Date(2014, 1, 4, 16, 0, 0, 0, time.UTC), // Saturday 4pm
		18.0,
	},
}

var scheduleConf = `
Weekends:
- 10:20-22:50: 18.0
Monday,Tue-Thu,Fri:
- 07:30-08:10: 18.0
- 08:10-17:30: 14.0
Weekdays:
- 17:30-22:20: 18.0
`

func TestSchedule(t *testing.T) {
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(scheduleConf), &schedule)
	s, err := NewSchedule(schedule)
	require.Nil(t, err)
	for _, tt := range testScheduleTable {
		assert.Equal(t, tt.temp, s.Target(tt.t, 0))
	}
}

func TestScheduleBST(t *testing.T) {
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(scheduleConf), &schedule)
	s, err := NewSchedule(schedule)
	require.Nil(t, err)
	bst := time.FixedZone("BST", 3600)
	for _, tt := range testScheduleTable {
		bt := time.Date(tt.t.Year(), tt.t.Month(), tt.t.Day(), tt.t.Hour(), tt.t.Minute(), tt.t.Second(), tt.t.Nanosecond(), bst)
		assert.Equal(t, tt.temp, s.Target(bt, 0))
	}
}

var testScheduleWithoutWeekendsTable = []struct {
	t    time.Time
	temp float64
}{
	{
		time.Date(2014, 1, 3, 7, 59, 0, 0, time.UTC), // Friday 7:59am
		0.0,
	},
	{
		time.Date(2014, 1, 3, 7, 59, 0, 0, time.UTC), // Friday 7:59am
		0.0,
	},
	{
		time.Date(2014, 1, 3, 8, 0, 0, 0, time.UTC), // Friday 8am
		17.0,
	},
	{
		time.Date(2014, 1, 3, 17, 59, 0, 0, time.UTC), // Friday 5:59pm
		17.0,
	},
	{
		time.Date(2014, 1, 3, 18, 0, 0, 0, time.UTC), // Friday 6pm
		0.0,
	},
	{
		time.Date(2014, 1, 4, 8, 0, 0, 0, time.UTC), // Saturday 8am
		0.0,
	},
	{
		time.Date(2014, 1, 5, 8, 0, 0, 0, time.UTC), // Sunday 8am
		0.0,
	},
	{
		time.Date(2014, 1, 6, 8, 0, 0, 0, time.UTC), // Monday 8am
		17.0,
	},
}

func TestScheduleWithoutWeekends(t *testing.T) {
	conf := `
Weekdays:
- 8:00-18:00: 17
`
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(conf), &schedule)
	s, err := NewSchedule(schedule)
	require.Nil(t, err)
	for _, tt := range testScheduleWithoutWeekendsTable {
		assert.Equal(t, tt.temp, s.Target(tt.t, 0))
	}
}

func TestScheduleEmpty(t *testing.T) {
	conf := `{}`
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(conf), &schedule)
	s, err := NewSchedule(schedule)
	require.Nil(t, err)
	assert.Equal(t, 1.0, s.Target(time.Date(2014, 1, 3, 7, 59, 0, 0, time.UTC), 1))
}

func TestScheduleAll(t *testing.T) {
	conf := `
All:
- 0:00-24:00: 10`
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(conf), &schedule)
	s, err := NewSchedule(schedule)
	require.Nil(t, err)
	assert.Equal(t, 10.0, s.Target(time.Date(2014, 1, 3, 7, 59, 0, 0, time.UTC), 0))
}

func TestScheduleOverlap(t *testing.T) {
	conf := `
All:
- 08:00-09:00: 15
- 00:00-24:00: 10
Fri:
- 08:30-08:35: 20
`
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(conf), &schedule)
	s, err := NewSchedule(schedule)
	require.Nil(t, err)
	assert.Equal(t, 10.0, s.Target(time.Date(2014, 1, 3, 7, 59, 0, 0, time.UTC), 0))
	assert.Equal(t, 15.0, s.Target(time.Date(2014, 1, 3, 8, 0, 0, 0, time.UTC), 0))
	assert.Equal(t, 15.0, s.Target(time.Date(2014, 1, 2, 8, 30, 0, 0, time.UTC), 0)) // Thursdau
	assert.Equal(t, 20.0, s.Target(time.Date(2014, 1, 3, 8, 30, 0, 0, time.UTC), 0)) // Friday
	assert.Equal(t, 15.0, s.Target(time.Date(2014, 1, 3, 8, 35, 0, 0, time.UTC), 0))
	assert.Equal(t, 10.0, s.Target(time.Date(2014, 1, 3, 9, 0, 0, 0, time.UTC), 0))
}

var testScheduleParseErrorTable = []string{
	"Monkeys: ['8:00-9:00': 17]",
	"Monkeys: ['8:00-9': 17]",
	"Monkeys: ['8:00-': 17]",
	"Monkeys: ['8:00': 17]",
	"Monday: ['8:': 17]",
	"Monday: [':00': 17]",
	"Monday: [':': 17]",
	"Monday: ['0:00-1:00': -1]",
	"Monday: ['0:00-1:00': 60]",
}

func TestScheduleParseError(t *testing.T) {
	for _, conf := range testScheduleParseErrorTable {
		var schedule config.ScheduleConf
		yaml.Unmarshal([]byte(conf), &schedule)
		s, err := NewSchedule(schedule)
		assert.Error(t, err)
		assert.Nil(t, s)
	}
}
