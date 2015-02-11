package services

import "github.com/barnybug/gohome/pubsub"

func SendAlert(message string, target string, subtopic string, interval int64) {
	fields := pubsub.Fields{
		"message": message,
		"target":  target,
	}
	if subtopic != "" {
		fields["subtopic"] = subtopic
		fields["interval"] = interval
	}
	ev := pubsub.NewEvent("alert", fields)
	Publisher.Emit(ev)
}

func SendQuery(query, source, remote, reply_to string) {
	fields := pubsub.Fields{
		"source":   source,
		"query":    query,
		"remote":   remote,
		"reply_to": reply_to,
	}
	ev := pubsub.NewEvent("query", fields)
	Publisher.Emit(ev)
}
