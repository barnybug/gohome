package heating

import (
	"encoding/json"
	"fmt"
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/dummy"
	"github.com/barnybug/gohome/services"
	"testing"
	"time"

	"gopkg.in/v1/yaml"
)

var (
	evOff        = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 10.1, "timestamp": "2014-01-04 10:19:00.000000"})
	evCold       = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 10.1, "timestamp": "2014-01-04 16:00:00.000000"})
	evBorderline = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 14.2, "timestamp": "2014-01-04 16:10:00.000000"})
	evHot        = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 19.0, "timestamp": "2014-01-04 16:31:00.000000"})

	evEmpty = pubsub.NewEvent("house", pubsub.Fields{"source": "house", "state": "Empty", "timestamp": "2014-01-04 16:00:00.000000"})
	evFull  = pubsub.NewEvent("house", pubsub.Fields{"source": "house", "state": "Full", "timestamp": "2014-01-04 16:00:00.000000"})

	evParty      = pubsub.NewEvent("command", pubsub.Fields{"target": "ch", "value": "20 30m", "timestamp": "2014-01-04 16:31:00.000000"})
	evAfterParty = pubsub.NewEvent("temp", map[string]interface{}{"source": "wmr100.2", "temp": 19.0, "timestamp": "2014-01-04 17:10:00.000000"})
)
var (
	timeLater  = evHot.Timestamp
	timeLater2 = evBorderline.Timestamp
	timeJustOn = time.Date(2014, 1, 4, 10, 20, 0, 0, time.UTC)
)
var configYaml = `
sensors: [temp.hallway]
device: heater.boiler
slop: 0.3
schedule:
  unoccupied:
    Monday,Tuesday,Wednesday,Thursday,Friday,Saturday,Sunday:
      - '0:00': 9.0
  hallway:
    Saturday,Sunday:
      - '10:20': 18.0
      - '22:50': 10.0
    Monday,Tuesday,Wednesday,Thursday,Friday:
      - '7:30': 18.0
      - '8:10': 14.0
      - '17:30': 18.0
      - '22:20': 10.0
`
var (
	testConfig config.HeatingConf
	em         *dummy.Publisher
	th         *Thermostat
)

var mockHousePresence string = `{"device":"house.presence","source":"presence","state":"Full","timestamp":"2014-12-07 13:43:59.849429","topic":"house","trigger":"person.barnaby state=In"}`

func Setup() {
	// setup mock store
	services.Config = config.ExampleConfig
	services.Stor = services.NewMockStore()
	services.Stor.Set("gohome/state/devices/house.presence", mockHousePresence)

	yaml.Unmarshal([]byte(configYaml), &testConfig)
	em = &dummy.Publisher{}
	th = NewThermostat(testConfig, em)
}

func TestOnOff(t *testing.T) {
	Setup()
	if th.State != false {
		t.Error("Expected initial State: false")
	}

	th.Event(evCold)
	// should switch on
	if th.State != true {
		t.Error("Expected new State: true")
	}
	if len(em.Events) != 1 {
		t.Error("Expected 1 events, got", len(em.Events))
	}
	em.Events = em.Events[:0]

	// should stay on at 14.2, within slop
	th.Event(evBorderline)
	if th.State != true {
		t.Error("Expected State: true")
	}

	th.Heartbeat(evBorderline.Timestamp)
	if th.State != true {
		t.Error("Expected State: true")
	}
	if len(em.Events) != 2 {
		t.Error("Expected 2 events, got", len(em.Events))
	}
	em.Events = em.Events[:0]

	// should switch off
	th.Event(evHot)
	if th.State != false {
		t.Error("Expected State: false")
	}
	if len(em.Events) != 1 {
		t.Error("Expected 1 events, got", len(em.Events))
	}
}

func TestTimeChange(t *testing.T) {
	Setup()
	th.Event(evOff)
	// should start off
	if th.State != false {
		t.Error("Expected State: false")
	}

	th.Heartbeat(timeJustOn)
	// should switch on
	if th.State != true {
		t.Error("Expected new State: true")
	}
}

func TestStaleTemperature(t *testing.T) {
	Setup()
	th.Event(evCold)
	// should start On
	if th.State != true {
		t.Error("Expected State: true")
	}

	// should switch off due to stale temperature data
	th.Heartbeat(timeLater)
	if th.State != false {
		t.Error("Expected State: false")
	}

	// and stay off
	th.Heartbeat(timeLater2)
	if th.State != false {
		t.Error("Expected State: false")
	}
}

func TestOccupiedToEmptyToFull(t *testing.T) {
	Setup()
	th.Event(evCold)
	if th.State != true {
		t.Error("Expected State: true")
	}

	// should switch off due to house being empty
	th.Event(evEmpty)
	if th.State != false {
		t.Error("Expected State: false")
	}

	// should switch on due to house being full
	th.Event(evFull)
	if th.State != true {
		t.Error("Expected State: true")
	}
}

func TestPartyMode(t *testing.T) {
	Setup()
	th.Event(evHot)
	if th.State != false {
		t.Error("Expected State: false")
	}
	th.Event(evParty)
	if th.State != true {
		t.Error("Expected State: true")
	}
	th.Event(evAfterParty)
	if th.State != false {
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
	fmt.Println(th.Status(evCold.Timestamp))
	th.Event(evCold)
	fmt.Println(th.Status(evBorderline.Timestamp))
	// Output:
	// Heating: false for unknown
	// hallway: unknown [18°C]
	// Heating: true for 10m
	// hallway: 10.1°C at Jan  4 16:00:00 [18°C]*
}

func ExampleQueryStatusText() {
	Setup()
	Clock = func() time.Time { return evBorderline.Timestamp }
	service := Service{th}
	th.Event(evCold)
	q := services.Question{"status", "", ""}
	fmt.Println(service.queryStatus(q).Text)
	// Output:
	// Heating: true for 10m
	// hallway: 10.1°C at Jan  4 16:00:00 [18°C]*
}

func ExampleQueryStatusJson() {
	Setup()
	Clock = func() time.Time { return evBorderline.Timestamp }
	service := Service{th}
	th.Event(evCold)
	q := services.Question{"status", "", ""}
	data := service.queryStatus(q).Json
	s, _ := json.Marshal(data)
	fmt.Println(string(s))
	// Output:
	// {"changed":"2014-01-04T16:00:00Z","devices":{"temp.hallway":{"at":"2014-01-04T16:00:00Z","target":18,"temp":10.1}},"heating":true}
}

func ExampleQueryCh() {
	Setup()
	Clock = func() time.Time { return evBorderline.Timestamp }
	service := Service{th}
	th.Event(evCold)
	fmt.Println(service.queryCh(
		services.Question{"ch", "", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "abc", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "18", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "18 xyz", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "18 1m", ""}))
	fmt.Println(service.queryCh(
		services.Question{"ch", "18 1h", ""}))
	// Output:
	// usage: ch <temp> <duration>
	// usage: ch <temp> <duration>
	// Set to 18°C until Jan  4 16:40:00
	// usage: ch <temp> <duration>
	// Set to 18°C until Jan  4 16:11:00
	// Set to 18°C until Jan  4 17:10:00
}

func ExampleSchedule() {
	Setup()
	s := NewSchedule(testConfig.Schedule["hallway"])
	t1 := time.Date(2014, 1, 3, 8, 0, 0, 0, time.UTC) // Friday
	fmt.Println(s.Target(t1))
	t2 := time.Date(2014, 1, 4, 8, 0, 0, 0, time.UTC) // Saturday
	fmt.Println(s.Target(t2))
	t3 := time.Date(2014, 1, 4, 16, 0, 0, 0, time.UTC) // Saturday
	fmt.Println(s.Target(t3))
	// Output:
	// 18
	// 10
	// 18
}
