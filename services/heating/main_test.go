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
	evOff        = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 10.1, "timestamp": "2014-01-04 10:19:00.000000"})
	evCold       = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 10.1, "timestamp": "2014-01-04 16:00:00.000000"})
	evBorderline = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 14.2, "timestamp": "2014-01-04 16:10:00.000000"})
	evHot        = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 19.0, "timestamp": "2014-01-04 16:31:00.000000"})

	evEmpty = pubsub.NewEvent("state", pubsub.Fields{"device": "house.presence", "state": "Empty", "timestamp": "2014-01-04 16:00:00.000000"})
	evFull  = pubsub.NewEvent("state", pubsub.Fields{"device": "house.presence", "state": "Full", "timestamp": "2014-01-04 16:00:00.000000"})

	evAfterParty = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 19.0, "timestamp": "2014-01-04 17:10:00.000000"})
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
        - '10:20': 18.0
        - '22:50': 10.0
      Monday,Tuesday,Wednesday,Thursday,Friday:
        - '7:30': 18.0
        - '8:10': 14.0
        - '17:30': 18.0
        - '22:20': 10.0
unoccupied: 9.0
`
var (
	testConfig config.HeatingConf
	em         *dummy.Publisher
	service    *Service
)

var mockHousePresence string = `{"device":"house.presence","source":"presence","state":"Full","timestamp":"2014-12-07 13:43:59.849429","topic":"house","trigger":"person.barnaby state=In"}`

func SetupStor() {
	// setup mock store
	services.Stor = services.NewMockStore()
	services.Stor.Set("gohome/state/events/state/house.presence", mockHousePresence)
}

func SetupService() {
	services.Config = config.ExampleConfig
	yaml.Unmarshal([]byte(configYaml), &testConfig)
	services.Config.Heating = testConfig
	em = &dummy.Publisher{}
	service = &Service{}
	service.Initialize(em)
}

func Setup() {
	SetupStor()
	SetupService()
}

func TestOnOff(t *testing.T) {
	Setup()
	if service.State != false {
		t.Error("Expected initial State: false")
	}

	service.Event(evCold)
	// should switch on
	if service.State != true {
		t.Error("Expected new State: true")
	}
	if len(em.Events) != 1 {
		t.Error("Expected 1 events, got", len(em.Events))
	}
	em.Events = em.Events[:0]

	// should stay on at 14.2, within slop
	service.Event(evBorderline)
	if service.State != true {
		t.Error("Expected State: true")
	}

	service.Heartbeat(evBorderline.Timestamp)
	if service.State != true {
		t.Error("Expected State: true")
	}
	if len(em.Events) != 3 {
		t.Error("Expected 3 events, got", len(em.Events))
	}
	em.Events = em.Events[:0]

	// should switch off
	service.Event(evHot)
	if service.State != false {
		t.Error("Expected State: false")
	}
	if len(em.Events) != 1 {
		t.Error("Expected 1 events, got", len(em.Events))
	}
}

func TestTimeChange(t *testing.T) {
	Setup()
	service.Event(evOff)
	// should start off
	if service.State != false {
		t.Error("Expected State: false")
	}

	service.Heartbeat(timeJustOn)
	// should switch on
	if service.State != true {
		t.Error("Expected new State: true")
	}
}

func TestStaleTemperature(t *testing.T) {
	Setup()
	service.Event(evCold)
	// should start On
	if service.State != true {
		t.Error("Expected State: true")
	}

	// should switch off due to stale temperature data
	service.Heartbeat(timeLater)
	if service.State != false {
		t.Error("Expected State: false")
	}

	// and stay off
	service.Heartbeat(timeLater2)
	if service.State != false {
		t.Error("Expected State: false")
	}
}

func TestTemperatureFromStateOn(t *testing.T) {
	SetupStor()
	mockTemp := `{"topic": "temp", "source": "wmr100.2", "temp": 10.1, "timestamp": "2014-01-04 16:00:00.000000"}`
	services.Stor.Set("gohome/state/events/temp/temp.hallway", mockTemp)
	SetupService()

	service.Heartbeat(evCold.Timestamp)
	// should start On
	assert.True(t, service.State, "Expected State: true")
}

func TestTemperatureFromStateOff(t *testing.T) {
	SetupStor()
	mockTemp := `{"topic": "temp", "source": "wmr100.2", "temp": 18.0, "timestamp": "2014-01-04 16:00:00.000000"}`
	services.Stor.Set("gohome/state/events/temp/temp.hallway", mockTemp)
	SetupService()

	service.Heartbeat(evCold.Timestamp)
	// should start Off
	assert.False(t, service.State, "Expected State: false")
}

func TestOccupiedToEmptyToFull(t *testing.T) {
	Setup()
	service.Event(evCold)
	if service.State != true {
		t.Error("Expected State: true")
	}

	// should switch off due to house being empty
	service.Event(evEmpty)
	if service.State != false {
		t.Error("Expected State: false")
	}

	// should switch on due to house being full
	service.Event(evFull)
	if service.State != true {
		t.Error("Expected State: true")
	}
}

func TestPartyMode(t *testing.T) {
	Setup()
	service.Event(evHot)
	if service.State != false {
		t.Error("Expected State: false")
	}
	Clock = func() time.Time { return timeParty }
	q := services.Question{Verb: "ch", Args: "thermostat.hallway 20 30m"}
	service.queryCh(q)
	if service.State != true {
		t.Error("Expected State: true")
	}
	service.Event(evAfterParty)
	if service.State != false {
		t.Error("Expected State: false")
	}
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
	// hallway: unknown [18°C]
	// Heating: true for 10m
	// hallway: 10.1°C at Jan  4 16:00:00 [18°C]*
}

func ExampleQueryStatusText() {
	Setup()
	Clock = func() time.Time { return evBorderline.Timestamp }
	service.Event(evCold)
	q := services.Question{"status", "", ""}
	fmt.Println(service.queryStatus(q).Text)
	// Output:
	// Heating: true for 10m
	// hallway: 10.1°C at Jan  4 16:00:00 [18°C]*
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

func ExampleQueryCh() {
	Setup()
	Clock = func() time.Time { return evBorderline.Timestamp }
	service.Event(evCold)
	fmt.Println(service.queryCh(
		services.Question{"ch", "", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "abc", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "thermostat.hallway 18", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "thermostat.hallway 18 xyz", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "thermostat.hallway 18 1m", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "thermostat.hallway 18 1h", ""}))
	// Output:
	// usage: ch <zone> <temp> <duration>
	// usage: ch <zone> <temp> <duration>
	// Set to 18°C for 30 minutes
	// usage: ch <zone> <temp> <duration>
	// Set to 18°C for 1 minute
	// Set to 18°C for 1 hour
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
		time.Date(2014, 1, 4, 8, 0, 0, 0, time.UTC), // Saturday 8am
		10.0,
	},
	{
		time.Date(2014, 1, 4, 16, 0, 0, 0, time.UTC), // Saturday 4pm
		18.0,
	},
}

var scheduleConf = `
Weekends:
- '10:20': 18.0
- '22:50': 10.0
Monday,Tuesday,Wednesday,Thursday,Friday:
- '7:30': 18.0
- '8:10': 14.0
- '17:30': 18.0
- '22:20': 10.0`

func TestSchedule(t *testing.T) {
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(scheduleConf), &schedule)
	s, err := NewSchedule(schedule)
	require.Nil(t, err)
	for _, tt := range testScheduleTable {
		assert.Equal(t, tt.temp, s.Target(tt.t))
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
		assert.Equal(t, tt.temp, s.Target(bt))
	}
}

var testScheduleWithoutWeekendsTable = []struct {
	t    time.Time
	temp float64
}{
	{
		time.Date(2014, 1, 3, 7, 59, 0, 0, time.UTC), // Friday 7:59am
		10.0,
	},
	{
		time.Date(2014, 1, 3, 7, 59, 0, 0, time.UTC), // Friday 7:59am
		10.0,
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
		10.0,
	},
	{
		time.Date(2014, 1, 4, 8, 0, 0, 0, time.UTC), // Saturday 8am
		10.0,
	},
	{
		time.Date(2014, 1, 5, 8, 0, 0, 0, time.UTC), // Sunday 8am
		10.0,
	},
	{
		time.Date(2014, 1, 6, 8, 0, 0, 0, time.UTC), // Monday 8am
		17.0,
	},
}

func TestScheduleWithoutWeekends(t *testing.T) {
	conf := `
Weekdays:
- '8:00': 17
- '18:00': 10`
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(conf), &schedule)
	s, err := NewSchedule(schedule)
	require.Nil(t, err)
	for _, tt := range testScheduleWithoutWeekendsTable {
		assert.Equal(t, tt.temp, s.Target(tt.t))
	}
}

func TestScheduleParseError(t *testing.T) {
	conf := `
Monkeys:
- '8:00': 17`
	var schedule config.ScheduleConf
	yaml.Unmarshal([]byte(conf), &schedule)
	s, err := NewSchedule(schedule)
	assert.Error(t, err)
	assert.Nil(t, s)
}
