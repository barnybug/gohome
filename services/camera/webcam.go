package camera

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/barnybug/gohome/config"
)

type Webcam struct {
	Path string
	Conf config.CameraNodeConf
}

func (self *Webcam) GotoPreset(preset int) error {
	return nil
}

func (self *Webcam) Snapshot() (filename string, err error) {
	var resp *http.Response
	resp, err = http.Get(self.Conf.Url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	filename = timestampFilename(self.Path, "jpg")
	fout, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer fout.Close()
	_, err = io.Copy(fout, resp.Body)
	if err != nil {
		// delete file if incomplete
		defer os.Remove(filename)
	}
	return
}

func (self *Webcam) Video(duration time.Duration) (string, error) {
	return "", nil
}

func (self *Webcam) Ir(b bool) error {
	return nil
}

func (self *Webcam) Detect(b bool) (err error) {
	return
}
