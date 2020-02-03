// Service for recording images, videos and controlling various IP cameras.
//
// Supported:
//
// - Foscam wireless IP cameras (http://www.foscam.co.uk)
//
// - Motion application (http://www.lavrsen.dk/foswiki/bin/view/Motion/WebHome)
package camera

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
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
	Snapshot() (io.ReadCloser, error)
	Video(path string, duration time.Duration) error
}

type Moveable interface {
	GotoPreset(preset int) error
	Ir(bool) error
	Detect(bool) error
}

func alertSnapshot(url, filename, target, message string) {
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

func saveSnapshot(cam Camera, filename string) error {
	r, err := cam.Snapshot()
	if err != nil {
		return err
	}
	defer r.Close()
	fout, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fout.Close()
	_, err = io.Copy(fout, r)
	if err != nil {
		// delete file if incomplete
		defer os.Remove(filename)
	}
	return err
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
		if cam, ok := cam.(Moveable); ok {
			log.Printf("%s going to preset %d", ev.Device(), preset)
			cam.GotoPreset(preset)
		}

	case "snapshot":
		log.Printf("%s taking snapshot", ev.Device())
		filename, url := cameraPath(ev.Device(), "jpg")
		go func() {
			if preset != 0 {
				if cam, ok := cam.(Moveable); ok {
					log.Printf("Going to preset: %d", preset)
					cam.GotoPreset(preset)
					time.Sleep(GotoDelay)
				}
			}
			err := saveSnapshot(cam, filename)
			if err != nil {
				log.Println("Error taking snapshot:", err)
			} else {
				log.Println("Snapshot:", filename)
				notify := ev.StringField("notify")
				message := ev.StringField("message")
				notifyActivity("snapshot", ev.Device(), filename, url)
				if notify != "" {
					alertSnapshot(url, filename, notify, message)
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
				if cam, ok := cam.(Moveable); ok {
					log.Printf("Going to preset: %d", preset)
					cam.GotoPreset(preset)
					time.Sleep(GotoDelay)
				}
			}
			if ir, ok := ev.Fields["ir"].(bool); ok {
				if cam, ok := cam.(Moveable); ok {
					err := cam.Ir(ir)
					if err != nil {
						log.Println("Error setting ir:", err)
					}
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
		if cam, ok := cam.(Moveable); ok {
			log.Printf("%s infra-red turned %t", ev.Device(), on)
			cam.Ir(on)
		}
	case "detection":
		on, _ := ev.Fields["on"].(bool)
		if cam, ok := cam.(Moveable); ok {
			log.Printf("%s detection %t", ev.Device(), on)
			cam.Detect(on)
		}
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
		case "rtsp":
			cameras[name] = &Rtsp{conf}
		}
	}
}

func httpSnapshot(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id, ok := q["camera"]
	if !ok {
		fmt.Fprint(w, "?camera=... required")
		return
	}
	cam, ok := cameras[id[0]]
	if !ok {
		fmt.Fprint(w, "camera not found")
		return
	}
	data, err := cam.Snapshot()
	if err != nil {
		fmt.Fprint(w, "error reading camera:", err)
		return
	}
	defer func() {
		err := data.Close()
		if err != nil {
			log.Println("Error closing snapshot:", err)
		}
	}()
	w.Header().Add("Content-Type", "image/jpeg")
	w.WriteHeader(200)
	io.Copy(w, data)
}

func startWebserver() {
	http.HandleFunc("/snapshot", httpSnapshot)
	addr := fmt.Sprintf(":%d", services.Config.Camera.Port)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("Webserver failed to start: ", err)
	}
}

func notifyActivity(command, device, filename, url string) {
	fields := pubsub.Fields{
		"device":   device,
		"filename": filename,
		"command":  command,
		"url":      url,
	}
	ev := pubsub.NewEvent("camera", fields)
	ev.SetRetained(true)
	services.Publisher.Emit(ev)
}

type Watcher struct {
	process *os.Process
}

func (w *Watcher) Restart() {
	if w.process != nil {
		w.process.Kill()
	}
}

func (w *Watcher) Run() {
	for w.watch() {
	}
}

func (w *Watcher) watch() bool {
	args := []string{"-r", "-m", "-e", "create"}
	for _, conf := range services.Config.Camera.Cameras {
		if conf.Watch != "" {
			args = append(args, conf.Watch)
		}
	}

	// start inotifywait
	cmd := exec.Command("inotifywait", args...)
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		log.Fatal("Failed to establish watches:", err)
	}
	w.process = cmd.Process
	// tail
	log.Println("Watch running...")
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		ps := strings.Split(scanner.Text(), " ")
		if len(ps) < 3 {
			continue
		}
		filepath := ps[0]
		filename := ps[2]
		fullpath := path.Join(filepath, filename)
		// figure out which device
		for device, conf := range services.Config.Camera.Cameras {
			if conf.Watch != "" && strings.HasPrefix(filepath, conf.Watch+"/") {
				if conf.Match.Regexp == nil || conf.Match.MatchString(fullpath) {
					log.Println("Notify: ", fullpath)
					notifyActivity("on", device, fullpath, "")
				}
				break
			}
		}
	}

	// cleanup
	if err := cmd.Wait(); err != nil {
		if cmd.ProcessState.ExitCode() == -1 {
			log.Println("Process terminated by signal, restarting...", err)
			return true
		} else {
			log.Println("Error running watch", err)
			return false
		}
	}

	log.Println("Watch finished, restarting...")
	return true
}

// Service camera
type Service struct {
	watcher *Watcher
}

// ID of the service
func (self *Service) ID() string {
	return "camera"
}

func (self *Service) ConfigUpdated(path string) {
	setupCameras()
	self.watcher.Restart()
}

func (self *Service) Init() error {
	setupCameras()
	self.watcher = &Watcher{}
	return nil
}

// Run the service
func (self *Service) Run() error {
	go self.watcher.Run()
	go startWebserver()

	for ev := range services.Subscriber.FilteredChannel("command") {
		if _, ok := cameras[ev.Device()]; ok {
			eventCommand(ev)
		}
	}
	return nil
}
