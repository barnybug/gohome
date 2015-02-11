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

func NewCommand(device string, state bool, repeat int) *Event {
	fields := map[string]interface{}{
		"device": device,
		"state":  state,
	}
	if repeat > 0 {
		fields["repeat"] = repeat
	}
	return NewEvent("command", fields)
}

const TimeFormat = "2006-01-02 15:04:05.000000"

func (self *Event) Map() map[string]interface{} {
	data := make(map[string]interface{})
	data["topic"] = self.Topic
	data["timestamp"] = self.Timestamp.Format(TimeFormat)
	for k, v := range self.Fields {
		data[k] = v
	}
	return data
}

func (self *Event) Bytes() []byte {
	v, _ := json.Marshal(self.Map())
	return v
}

func (self *Event) String() string {
	return string(self.Bytes())
}

func (self *Event) StringField(name string) string {
	ret, _ := self.Fields[name].(string)
	return ret
}

func (self *Event) IntField(name string) int64 {
	ret, _ := self.Fields[name].(float64)
	return int64(ret)
}

func (self *Event) Target() string {
	return self.StringField("target")
}

func (self *Event) Device() string {
	return self.StringField("device")
}

func (self *Event) Source() string {
	return self.StringField("source")
}

func (self *Event) Command() string {
	return self.StringField("command")
}

func (self *Event) State() string {
	return self.StringField("state")
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
