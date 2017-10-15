package camera

import (
	"io"
	"os/exec"
	"time"

	"github.com/barnybug/gohome/config"
)

type Rtsp struct {
	Conf config.CameraNodeConf
}

type ReaderCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (w ReaderCloser) Close() error {
	w.ReadCloser.Close()
	return w.cmd.Wait()
}

func (self *Rtsp) Snapshot() (io.ReadCloser, error) {
	// launch ffmpeg to grab single frame
	cmd := exec.Command("ffmpeg",
		"-i", self.Conf.Url,
		"-vframes", "1", "-f", "mjpeg", "-")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return ReaderCloser{stdout, cmd}, nil
}

func (self *Rtsp) Video(path string, duration time.Duration) error {
	return nil
}
