// Service for recording images, videos and controlling various IP cameras.
//
// Supported:
//
// - Foscam wireless IP cameras (http://www.foscam.co.uk)
//
// - Motion application (http://www.lavrsen.dk/foswiki/bin/view/Motion/WebHome)
package camera

import (
	"fmt"
	"log"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

var cameras map[string]Camera
var GotoDelay, _ = time.ParseDuration("4s")
var (
	snapshotDir string
)

type Camera interface {
	GotoPreset(preset int) error
	Snapshot(path string) error
	Video(path string, duration time.Duration) error
	Ir(bool) error
	Detect(bool) error
}

func notifySnapshot(url, filename, target, message string) {
	fields := pubsub.Fields{
		"url":      url,
		"filename": filename,
		"target":   target,
		"message":  message,
	}
	ev := pubsub.NewEvent("alert", fields)
	services.Publisher.Emit(ev)
}

func cameraPath(name, ext string) (filename string, url string) {
	ts := time.Now().Format("20060102T150405")
	dir := util.ExpandUser(services.Config.Camera.Path)
	filename = fmt.Sprintf("%s/%s-%s.%s", dir, name, ts, ext)
	url = fmt.Sprintf("%s/%s-%s.%s", services.Config.Camera.Url, name, ts, ext)
	return
}

func eventCommand(ev *pubsub.Event) {
	cam, ok := cameras[ev.Device()]
	if !ok {
		return
	}

	p, _ := ev.Fields["preset"].(float64)
	preset := int(p)

	switch ev.Fields["command"] {
	case "position":
		log.Printf("%s going to preset %d", ev.Device(), preset)
		cam.GotoPreset(preset)

	case "snapshot":
		log.Printf("%s taking snapshot", ev.Device())
		filename, url := cameraPath(ev.Device(), "jpg")
		go func() {
			if preset != 0 {
				log.Printf("Going to preset: %d", preset)
				cam.GotoPreset(preset)
				time.Sleep(GotoDelay)
			}
			err := cam.Snapshot(filename)
			if err != nil {
				log.Println("Error taking snapshot:", err)
			} else {
				log.Println("Snapshot:", filename)
				notify := ev.StringField("notify")
				message := ev.StringField("message")
				if notify != "" {
					notifySnapshot(url, filename, notify, message)
				}
			}
		}()

	case "video":
		filename, _ := cameraPath(ev.Device(), "mp4")
		timeout, ok := ev.Fields["timeout"].(float64)
		if !ok {
			timeout = 15
		}
		duration := time.Duration(timeout) * time.Second
		log.Printf("%s recording video for %s", ev.Device(), duration)
		go func() {
			if preset != 0 {
				log.Printf("Going to preset: %d", preset)
				cam.GotoPreset(preset)
				time.Sleep(GotoDelay)
			}
			if ir, ok := ev.Fields["ir"].(bool); ok {
				err := cam.Ir(ir)
				if err != nil {
					log.Println("Error setting ir:", err)
				}
			}
			err := cam.Video(filename, duration)
			if err != nil {
				log.Println("Error taking video:", err)
			} else {
				log.Println("Video:", filename)
			}
		}()

	case "ir":
		on, _ := ev.Fields["on"].(bool)
		log.Printf("%s infra-red turned %s", ev.Device(), on)
		cam.Ir(on)

	case "detection":
		on, _ := ev.Fields["on"].(bool)
		log.Printf("%s detection %s", ev.Device(), on)
		cam.Detect(on)
	}
}

func setupCameras() {
	cameras = map[string]Camera{}
	for name, conf := range services.Config.Camera.Cameras {
		switch conf.Protocol {
		case "foscam":
			cameras[name] = &Foscam{conf}
		case "motion":
			cameras[name] = &Motion{conf}
		case "webcam":
			cameras[name] = &Webcam{conf}
		}
	}
}

// Service camera
type Service struct {
}

// ID of the service
func (self *Service) ID() string {
	return "camera"
}

func (self *Service) ConfigUpdated(path string) {
	setupCameras()
}

// Run the service
func (self *Service) Run() error {
	setupCameras()

	for ev := range services.Subscriber.FilteredChannel("command") {
		fmt.Println("dev", ev.Device())
		if _, ok := cameras[ev.Device()]; ok {
			eventCommand(ev)
		}
	}
	return nil
}
