package automata

import (
	"testing"
	"time"

	"github.com/barnybug/gofsm"
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/dummy"
	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
)

var service = &Service{}
var (
	evOn      = NewEventContext(service, pubsub.NewEvent("ack", pubsub.Fields{"device": "light.porch", "command": "on", "timestamp": "2017-09-26 19:24:22.069"}))
	evState   = NewEventContext(service, pubsub.NewEvent("state", pubsub.Fields{"device": "light.porch", "state": "On", "timestamp": "2017-09-26 19:24:22.069"}))
	evTime    = NewEventContext(service, pubsub.NewEvent("time", pubsub.Fields{"device": "time", "hhmm": "2230", "timestamp": "2017-09-26 22:30:00.000"}))
	evMissing = NewEventContext(service, pubsub.NewEvent("ack", pubsub.Fields{"timestamp": "2017-09-26 19:24:22.069"}))
)

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	var _ services.Queryable = (*Service)(nil)
	// Output:
}

func TestEventSimple(t *testing.T) {
	assert.True(t, evOn.Match("device=='light.porch' && command=='on'"))
	assert.False(t, evOn.Match("device=='light.porch' && command=='off'"))
}

func TestEventType(t *testing.T) {
	assert.True(t, evOn.Match("type=='light' && command=='on'"))
	assert.True(t, evOn.Match("type=='light'"))
}

func TestEventOr(t *testing.T) {
	assert.True(t, evOn.Match("device=='door.front' && command=='off' || device=='light.porch' && command=='on'"))
	assert.True(t, evOn.Match("device=='light.porch' && command=='on' || device=='door.front' && command=='off'"))
}

func TestEventNotABoolean(t *testing.T) {
	assert.False(t, evOn.Match("'abc'"))
}

func TestBadExpression(t *testing.T) {
	assert.False(t, evOn.Match("blah()"))
}

func BenchmarkEventTrue(b *testing.B) {
	for i := 0; i < b.N; i++ {
		evOn.Match("device=='door.front' && command=='off' || device=='light.porch' && command=='on'")
	}
}

func BenchmarkEventSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		evOn.Match("device=='light.porch' && command=='on'")
	}
}

func BenchmarkEventFalse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		evMissing.Match("device=='door.front' && command=='off' || device=='light.porch' && command=='on'")
	}
}

func TestEventMissing(t *testing.T) {
	assert.False(t, evMissing.Match("device=='light.porch' && command=='on'"))
}

func TestEventTime(t *testing.T) {
	assert.False(t, evTime.Match("device=='time' && hhmm=='2229'"))
	assert.True(t, evTime.Match("device=='time' && hhmm=='2230'"))
}

func TestEventContextString(t *testing.T) {
	assert.Equal(t, "light.porch command=on", evOn.String())
}

func testChangeContext() ChangeContext {
	ev := pubsub.NewEvent("state", pubsub.Fields{"device": "light.kitchen", "state": "On", "timestamp": "2017-09-26 19:24:22.069", "number": 2.5})
	now := time.Now()
	change := gofsm.Change{"", "", "", now, time.Minute, nil}
	return ChangeContext{ev, change}
}

func TestFormat(t *testing.T) {
	services.Config = config.ExampleConfig
	context := testChangeContext()

	assert.Equal(t, "test", context.Format("test"))
	assert.Equal(t, "$missing", context.Format("$missing"))
	assert.Equal(t, "light.kitchen", context.Format("$id"))
	assert.Equal(t, "light", context.Format("$type"))
	assert.Equal(t, "Kitchen", context.Format("$name"))
	assert.Equal(t, "1 minute", context.Format("$duration"))
	assert.Equal(t, "On", context.Format("$state"))
	assert.Equal(t, "2.5", context.Format("$number"))
}

func TestLog(t *testing.T) {
	services.Publisher = &dummy.Publisher{}
	context := testChangeContext()

	_, err := service.Log("argument")
	assert.Error(t, err)
	_, err = service.Log(1)
	assert.Error(t, err)

	_, err = service.Log(context, "test")
	assert.NoError(t, err)
}

func TestCommand(t *testing.T) {
	services.Publisher = &dummy.Publisher{}
	context := testChangeContext()

	_, err := service.Command("argument")
	assert.Error(t, err)
	_, err = service.Command(1)
	assert.Error(t, err)

	_, err = service.Command(context, "command")
	assert.NoError(t, err)
}

func TestStartTimer(t *testing.T) {
	services.Publisher = &dummy.Publisher{}
	service.timers = map[string]*time.Timer{}
	context := testChangeContext()

	_, err := service.StartTimer(context, "command", "a")
	assert.Error(t, err)

	_, err = service.StartTimer(context, "command", 1.0)
	assert.NoError(t, err)
	_, err = service.StartTimer(context, "command", 1)
	assert.NoError(t, err)
	_, err = service.StartTimer(context, "command", int64(1))
	assert.NoError(t, err)
}

func TestCheckArguments(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(checkArguments([]interface{}{}))
	assert.NoError(checkArguments([]interface{}{"expected"}, "string"))
	assert.NoError(checkArguments([]interface{}{"expected", 1.0}, "string", "float64"))
	assert.NoError(checkArguments([]interface{}{"expected", 1.0}, "string", "int"))
	assert.NoError(checkArguments([]interface{}{"a"}, "string", "..."))
	assert.NoError(checkArguments([]interface{}{"a", "b"}, "string", "..."))
	assert.NoError(checkArguments([]interface{}{"a", "b", 1}, "string", "..."))

	assert.Error(checkArguments([]interface{}{}, "string"))
	assert.Error(checkArguments([]interface{}{"unexpected"}))
	assert.Error(checkArguments([]interface{}{1}, "string"))
	assert.Error(checkArguments([]interface{}{"a"}, "float64"))
	assert.Error(checkArguments([]interface{}{"a", "a"}, "string", "float64"))
	assert.Error(checkArguments([]interface{}{"x"}, "int"))
	assert.Error(checkArguments([]interface{}{}, "string", "..."))
}
