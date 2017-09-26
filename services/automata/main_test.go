package automata

import (
	"testing"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
)

var (
	evOn      = pubsub.NewEvent("ack", pubsub.Fields{"device": "light.porch", "command": "on", "timestamp": "2017-09-26 19:24:22.069"})
	evState   = pubsub.NewEvent("state", pubsub.Fields{"device": "light.porch", "state": "On", "timestamp": "2017-09-26 19:24:22.069"})
	evTime    = pubsub.NewEvent("time", pubsub.Fields{"device": "time", "hhmm": "2230", "timestamp": "2017-09-26 22:30:00.000"})
	evMissing = pubsub.NewEvent("ack", pubsub.Fields{"timestamp": "2017-09-26 19:24:22.069"})
)

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	var _ services.Queryable = (*Service)(nil)
	// Output:
}

func TestEventSimple(t *testing.T) {
	event := EventWrapper{evOn}
	assert.True(t, event.Match("light.porch command=on"))
	assert.False(t, event.Match("light.porch command=off"))
}

func TestEventWildcard(t *testing.T) {
	event := EventWrapper{evOn}
	assert.True(t, event.Match("light.* command=on"))
	assert.True(t, event.Match("light.porch command=*"))
	assert.True(t, event.Match("light.* command=*"))
}

func TestEventOr(t *testing.T) {
	event := EventWrapper{evOn}
	assert.True(t, event.Match("door.* command=off or light.* command=on"))
	assert.True(t, event.Match("light.* command=on or door.* command=off"))
}

func BenchmarkEventTrue(b *testing.B) {
	event := EventWrapper{evOn}
	for i := 0; i < b.N; i++ {
		event.Match("door.* command=off or light.* command=on")
	}
}

func BenchmarkEventSimple(b *testing.B) {
	event := EventWrapper{evOn}
	for i := 0; i < b.N; i++ {
		event.Match("light.porch command=on")
	}
}

func BenchmarkEventFalse(b *testing.B) {
	event := EventWrapper{evMissing}
	for i := 0; i < b.N; i++ {
		event.Match("door.* command=off or light.* command=on")
	}
}

func TestEventMissing(t *testing.T) {
	event := EventWrapper{evMissing}
	assert.False(t, event.Match("light.porch command=on"))
}

func TestEventTime(t *testing.T) {
	event := EventWrapper{evTime}
	assert.False(t, event.Match("time hhmm=2229"))
	assert.True(t, event.Match("time hhmm=2230"))
}

func TestEventWrapperString(t *testing.T) {
	event := EventWrapper{evOn}
	assert.Equal(t, "light.porch command=on", event.String())
}
