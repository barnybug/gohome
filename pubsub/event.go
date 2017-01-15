package pubsub

import (
	"encoding/json"
	"fmt"
	"time"
)

type Fields map[string]interface{}

type Event struct {
	Topic     string
	Timestamp time.Time
	Fields    Fields
}

func NewEvent(topic string, fields map[string]interface{}) *Event {
	timestamp := time.Now().UTC()
	if ts, ok := fields["timestamp"].(string); ok {
		delete(fields, "timestamp")
		timestamp, _ = time.Parse(TimeFormat, ts)
	}
	return &Event{Topic: topic, Timestamp: timestamp, Fields: fields}
}

func NewCommand(device string, command string) *Event {
	fields := map[string]interface{}{
		"topic":   "command",
		"device":  device,
		"command": command,
	}
	return NewEvent(fmt.Sprintf("command/%s", device), fields)
}

const TimeFormat = "2006-01-02 15:04:05.000000"

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
	v, _ := json.Marshal(event.Map())
	return v
}

func (event *Event) String() string {
	return string(event.Bytes())
}

func (event *Event) StringField(name string) string {
	ret, _ := event.Fields[name].(string)
	return ret
}

func (event *Event) IntField(name string) int64 {
	ret, _ := event.Fields[name].(float64)
	return int64(ret)
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

func Parse(msg string) *Event {
	// extract json
	var fields map[string]interface{}
	err := json.Unmarshal([]byte(msg), &fields)
	if err != nil {
		return nil
	}
	topic, ok := fields["topic"].(string)
	if !ok {
		return nil
	}
	delete(fields, "topic")
	return NewEvent(topic, fields)
}
