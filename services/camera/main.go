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
	"strings"
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
	Snapshot() (string, error)
	Video(duration time.Duration) (string, error)
	Ir(bool) error
	Detect(bool) error
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
		go func() {
			if preset != 0 {
				log.Printf("Going to preset: %d", preset)
				cam.GotoPreset(preset)
				time.Sleep(GotoDelay)
			}
			filename, err := cam.Snapshot()
			if err != nil {
				log.Println("Error taking snapshot:", err)
			} else {
				log.Println("Snapshot:", filename)
			}
		}()

	case "video":
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
			filename, err := cam.Video(duration)
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
	snapshotDir = util.ExpandUser(services.Config.Camera.Path)
	cameras = map[string]Camera{}
	for name, conf := range services.Config.Camera.Cameras {
		snapshot := fmt.Sprintf("%s/%s-%%s.%%s", snapshotDir, strings.TrimPrefix(name, "camera."))
		switch conf.Protocol {
		case "foscam":
			cameras[name] = &Foscam{snapshot, conf}
		case "motion":
			cameras[name] = &Motion{snapshot, conf}
		case "webvam":
			cameras[name] = &Webcam{snapshot, conf}
		}
	}
}

func timestampFilename(path, ext string) string {
	ts := time.Now().Format("20060102T150405")
	return fmt.Sprintf(path, ts, ext)
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
		if _, ok := cameras[ev.Device()]; ok {
			eventCommand(ev)
		}
	}
	return nil
}
