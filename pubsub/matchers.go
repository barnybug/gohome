package pubsub

import "strings"

type PrefixTopic struct {
	Prefix string
}

func Prefix(prefix string) *PrefixTopic {
	return &PrefixTopic{prefix}
}

func (t *PrefixTopic) Match(topic string) bool {
	return t.Prefix == topic || strings.HasPrefix(topic, t.Prefix+"/")
}

type AllTopic struct{}

func All() *AllTopic {
	return &AllTopic{}
}

func (t *AllTopic) Match(topic string) bool {
	return true
}

type ExactTopic struct {
	Exact string
}

func Exact(exact string) *ExactTopic {
	return &ExactTopic{exact}
}

func (t *ExactTopic) Match(topic string) bool {
	return t.Exact == topic
}
