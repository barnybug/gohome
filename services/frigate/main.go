// Service to translate frigate events to/from gohome.
package frigate

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/pubsub/mqtt"
	"github.com/barnybug/gohome/services"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// Service frigate
type Service struct {
}

func (self *Service) ID() string {
	return "frigate"
}

const Timeout = 3 * time.Minute
const TimeFormat = "2006-01-02 15:04:05.000"

// {"before": {"id": "1688400328.42808-weawsp", "camera": "front", "frame_time": 1688453551.154563, "snapshot_time": 1688400330.671039, "label": "car", "sub_label": null, "top_score": 0.998046875, "false_positive": false, "start_time": 1688400328.42808, "end_time": null, "score": 0.97412109375, "box": [184, 96, 381, 246], "area": 29550, "ratio": 1.3133333333333332, "region": [81, 0, 433, 352], "stationary": true, "motionless_count": 198274, "position_changes": 2, "current_zones": [], "entered_zones": [], "has_clip": true, "has_snapshot": false}, "after": {"id": "1688400328.42808-weawsp", "camera": "front", "frame_time": 1688453611.450034, "snapshot_time": 1688400330.671039, "label": "car", "sub_label": null, "top_score": 0.998046875, "false_positive": false, "start_time": 1688400328.42808, "end_time": null, "score": 0.97412109375, "box": [184, 96, 381, 246], "area": 29550, "ratio": 1.3133333333333332, "region": [81, 0, 433, 352], "stationary": true, "motionless_count": 198515, "position_changes": 2, "current_zones": [], "entered_zones": [], "has_clip": true, "has_snapshot": false}, "type": "update"}
type EventDetails struct {
	Id              string   `json:"id"`
	Camera          string   `json:"camera"`
	FrameTime       float64  `json:"frame_time"`
	SnapshotTime    float64  `json:"snapshot_time"`
	Label           string   `json:"label"`
	SubLabel        string   `json:"sub_label"`
	TopScore        float64  `json:"top_score"`
	FalsePositive   bool     `json:"false_positive"`
	StartTime       float64  `json:"start_time"`
	EndTime         *float64 `json:"end_time"`
	Score           float64  `json:"score"`
	Stationary      bool     `json:"stationary"`
	MotionlessCount int64    `json:"motionless_count"`
	PositionChanges int64    `json:"position_changes"`
	HasClip         bool     `json:"has_clip"`
	HasSnapshot     bool     `json:"has_snapshot"`
}

type Event struct {
	Type   string       `json:"type"`
	Before EventDetails `json:"before"`
	After  EventDetails `json:"after"`
}

func (self *Service) Run() error {
	messages := make(chan MQTT.Message)
	mqtt.Client.Subscribe("frigate/#", 1, func(client MQTT.Client, message MQTT.Message) {
		messages <- message
	})

	cameraActiveEvent := map[string]string{}
	cameraLastNotify := map[string]time.Time{}

	for msg := range messages {
		if msg.Retained() {
			continue
		}

		ps := strings.Split(msg.Topic(), "/")
		if len(ps) == 2 && ps[1] == "events" {
			var event Event
			err := json.Unmarshal(msg.Payload(), &event)
			if err != nil {
				log.Printf("Error parsing event: %s", err)
				continue
			}
			log.Printf("%s: %s event: '%s'", event.Before.Camera, event.Type, event.Before.Id)
			switch event.Type {
			case "new":
				cameraActiveEvent[event.Before.Camera] = event.Before.Id
			}
		} else if len(ps) == 4 && ps[3] == "snapshot" {
			camera := ps[1]
			object := ps[2]
			eventId := cameraActiveEvent[camera]
			log.Printf("%s: %s snapshot (event: '%s')", camera, object, eventId)
			// dedup snapshots
			now := time.Now()
			if now.Before(cameraLastNotify[camera].Add(Timeout)) {
				continue
			}
			cameraLastNotify[camera] = now
			// https://frigate/api/events/1688450573.161219-rbzd4p/clip.mp4
			clipUrl := fmt.Sprintf("%sapi/events/%s/clip.mp4", services.Config.Frigate.Url, eventId)
			// https://frigate/clips/doorbell-1688450573.161219-rbzd4p.jpg
			snapshotUrl := fmt.Sprintf("%sclips/%s-%s.jpg", services.Config.Frigate.Url, camera, eventId)
			message := fmt.Sprintf("%s detected a %s [clip 🎥](%s)", camera, object, clipUrl)
			fields := pubsub.Fields{
				"url":      snapshotUrl,
				"target":   "telegram",
				"message":  message,
				"markdown": true,
			}
			ev := pubsub.NewEvent("alert", fields)
			services.Publisher.Emit(ev)
		}
	}
	return nil
}
