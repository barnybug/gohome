package pubsub

import (
	"encoding/json"
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
		timestamp, _ = time.Parse(TimeFormat, ts)
	}
	return &Event{Topic: topic, Timestamp: timestamp, Fields: fields}
}

func NewCommand(device string, command string) *Event {
	fields := map[string]interface{}{
		"device":  device,
		"command": command,
	}
	return NewEvent("command", fields)
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
	timestamp, ok := fields["timestamp"].(string)
	if !ok {
		return nil
	}
	ts, _ := time.Parse(TimeFormat, timestamp)
	topic, ok := fields["topic"].(string)
	if !ok {
		return nil
	}
	event := Event{Topic: topic, Timestamp: ts}
	delete(fields, "topic")
	delete(fields, "timestamp")
	event.Fields = fields
	return &event
}
