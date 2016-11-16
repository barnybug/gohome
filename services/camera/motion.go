package camera

import (
	"fmt"
	"net/http"
	"time"

	"github.com/barnybug/gohome/config"
)

type Motion struct {
	Path string
	Conf config.CameraNodeConf
}

func (self *Motion) GotoPreset(preset int) error {
	return nil
}

func (self *Motion) Snapshot() (string, error) {
	return "", nil
}

func (self *Motion) Video(duration time.Duration) (string, error) {
	return "", nil
}

func (self *Motion) Ir(b bool) error {
	return nil
}

var DetectCommands = map[bool]string{
	true:  "detection/start",
	false: "detection/pause",
}

func (self *Motion) Detect(b bool) (err error) {
	url := fmt.Sprintf("%s/%s", self.Conf.Url, DetectCommands[b])
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return
}
