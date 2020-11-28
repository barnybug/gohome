package pubsub

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type Fields map[string]interface{}

type Event struct {
	Topic     string
	Timestamp time.Time
	Fields    Fields
	Retained  bool
	Format    string
	Raw       []byte
	Published sync.WaitGroup
}

func NewEvent(topic string, fields Fields) *Event {
	timestamp := time.Now().UTC()
	if ts, ok := fields["timestamp"].(string); ok {
		delete(fields, "timestamp")
		timestamp, _ = time.Parse(TimeFormat, ts)
	}
	return &Event{Topic: topic, Timestamp: timestamp, Fields: fields, Format: "json"}
}

func NewRawEvent(topic string, raw []byte) *Event {
	timestamp := time.Now().UTC()
	return &Event{Topic: topic, Timestamp: timestamp, Fields: nil, Format: "raw", Raw: raw}
}

func NewCommand(device string, command string) *Event {
	fields := Fields{
		"topic":   "command",
		"device":  device,
		"command": command,
	}
	return NewEvent("command", fields)
}

const TimeFormat = "2006-01-02 15:04:05.000"

func (event *Event) Map() map[string]interface{} {
	data := make(map[string]interface{})
	data["topic"] = event.Topic
	data["timestamp"] = event.Timestamp.Format(TimeFormat)
	for k, v := range event.Fields {
		data[k] = v
	}
	return data
}

func (event *Event) Bytes() []byte {
	if event.Format == "json" {
		v, _ := json.Marshal(event.Map())
		return v
	} else if event.Format == "raw" {
		return event.Raw
	} else if event.Format == "yaml" {
		v, _ := yaml.Marshal(event.Map())
		return v
	}
	return nil
}

func (event *Event) String() string {
	return string(event.Bytes())
}

func (event *Event) IsSet(name string) bool {
	_, ok := event.Fields[name]
	return ok
}

func (event *Event) StringField(name string) string {
	ret, _ := event.Fields[name].(string)
	return ret
}

func (event *Event) IntField(name string) int64 {
	ret, _ := event.Fields[name].(float64)
	return int64(ret)
}

func (event *Event) FloatField(name string) float64 {
	ret, _ := event.Fields[name].(float64)
	return ret
}

func (event *Event) SetRepeat(repeat int) {
	event.Fields["repeat"] = repeat
}

func (event *Event) SetField(name string, value interface{}) {
	event.Fields[name] = value
}

func (event *Event) SetFields(fields map[string]interface{}) {
	for key, value := range fields {
		event.Fields[key] = value
	}
}

func (event *Event) SetRetained(retained bool) {
	event.Retained = retained
}

func (event *Event) Target() string {
	return event.StringField("target")
}

func (event *Event) Device() string {
	return event.StringField("device")
}

func (event *Event) Source() string {
	return event.StringField("source")
}

func (event *Event) Command() string {
	return event.StringField("command")
}

func (event *Event) State() string {
	return event.StringField("state")
}

func (event *Event) Ack() *Event {
	fields := Fields{
		"device":  event.Device(),
		"command": event.Command(),
	}
	return NewEvent("ack", fields)
}

func Parse(msg, topic string) *Event {
	var format string
	var fields map[string]interface{}
	if strings.HasPrefix(msg, "---") {
		// yaml
		err := yaml.Unmarshal([]byte(msg), &fields)
		if err != nil {
			return nil
		}
		format = "yaml"
	} else if strings.HasPrefix(msg, "{") {
		// json
		err := json.Unmarshal([]byte(msg), &fields)
		if err != nil {
			return nil
		}
		format = "json"
	} else {
		// raw
		return NewRawEvent(topic, []byte(msg))
	}
	topic, ok := fields["topic"].(string)
	if !ok {
		return nil
	}
	delete(fields, "topic")
	ev := NewEvent(topic, fields)
	ev.Format = format
	return ev
}
