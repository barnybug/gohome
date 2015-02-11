package services

import (
	"github.com/barnybug/gohome/pubsub"
	"strings"
)

type Question struct {
	Verb string
	Args string
	From string
}

type Answer struct {
	Text string
	Json interface{}
}

type QueryHandler func(q Question) Answer

type QueryHandlers map[string]QueryHandler

type Queryable interface {
	Id() string
	QueryHandlers() QueryHandlers
}

// TextHandler adapts a string return value to an Answer
func TextHandler(fn func(q Question) string) func(q Question) Answer {
	return func(q Question) Answer {
		text := fn(q)
		return Answer{Text: text}
	}
}

func sendAnswer(request *pubsub.Event, source string, answer Answer) {
	fields := pubsub.Fields{
		"source": source,
		"target": request.StringField("source"),
	}
	if answer.Text != "" {
		fields["message"] = answer.Text
	}
	if answer.Json != nil {
		fields["json"] = answer.Json
	}

	remote := request.StringField("remote")
	if remote != "" {
		fields["remote"] = remote
	}

	topic := "alert"
	reply_to := request.StringField("reply_to")
	if reply_to != "" {
		topic = reply_to
	}

	response := pubsub.NewEvent(topic, fields)
	Publisher.Emit(response)
}

// StaticHandler just returns a hardcoded string - useful for "help"
func StaticHandler(msg string) QueryHandler {
	return func(_ Question) Answer {
		return Answer{Text: msg}
	}
}

func QuerySubscriber() {
	var queryables []Queryable
	for _, service := range enabled {
		if qs, ok := service.(Queryable); ok {
			queryables = append(queryables, qs)
		}
	}
	if len(queryables) == 0 {
		// no point running if no Queryable services
		return
	}

	// build map of handlers
	handlers := map[string][]QueryHandler{}
	for _, service := range queryables {
		for key, handler := range service.QueryHandlers() {
			handlers[key] = append(handlers[key], handler)
			handlers[service.Id()+"/"+key] = append(handlers[service.Id()+"/"+key], handler)
		}
	}

	for ev := range Subscriber.FilteredChannel("query") {
		parts := strings.SplitN(ev.StringField("query"), " ", 2)
		args := ""
		if len(parts) > 1 {
			args = parts[1]
		}
		first := strings.ToLower(parts[0])
		from := ev.StringField("source") + ":" + ev.StringField("remote")
		q := Question{first, args, from}

		for _, service := range queryables {
			if handlerList, ok := handlers[first]; ok {
				for _, handler := range handlerList {
					a := handler(q)
					sendAnswer(ev, service.Id(), a)
				}
			}
		}
	}
}
