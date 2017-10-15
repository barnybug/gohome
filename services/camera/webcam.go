package camera

import (
	"io"
	"net/http"
	"time"

	"github.com/barnybug/gohome/config"
)

type Webcam struct {
	Conf config.CameraNodeConf
}

func (self *Webcam) Snapshot() (r io.ReadCloser, err error) {
	var resp *http.Response
	resp, err = http.Get(self.Conf.Url)
	if err != nil {
		return
	}
	return resp.Body, nil
}

func (self *Webcam) Video(path string, duration time.Duration) error {
	return nil
}
