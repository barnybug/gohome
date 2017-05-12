package camera

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/barnybug/gohome/config"
)

type Foscam struct {
	Conf config.CameraNodeConf
}

var EmptyParams = map[string]string{}

func (self *Foscam) control(cgi string, params map[string]string) (resp *http.Response, err error) {
	params["user"] = self.Conf.User
	params["pwd"] = self.Conf.Password
	q := url.Values{}
	for k, v := range params {
		q[k] = []string{v}
	}
	url := fmt.Sprintf("%s%s?%s", self.Conf.Url, cgi, q.Encode())
	resp, err = http.Get(url)
	return
}

func (self *Foscam) GotoPreset(preset int) (err error) {
	n := fmt.Sprint(preset*2 + 31)
	resp, err := self.control("decoder_control.cgi", map[string]string{"command": n})
	if err != nil {
		return
	}
	resp.Body.Close()
	return
}

func (self *Foscam) Snapshot(filename string) (err error) {
	resp, err := self.control("snapshot.cgi", EmptyParams)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	fout, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer fout.Close()
	n, err := io.Copy(fout, resp.Body)
	fmt.Println(n)
	if err != nil {
		return
	}
	return
}

type TLReader struct {
	R   io.Reader
	End time.Time
}

func TimeLimitReader(r io.Reader, d time.Duration) io.Reader {
	return &TLReader{r, time.Now().Add(d)}
}

func (self *TLReader) Read(p []byte) (n int, err error) {
	if time.Now().After(self.End) {
		return 0, io.EOF
	}
	return self.R.Read(p)
}

func (self *Foscam) Video(filename string, duration time.Duration) (err error) {
	resp, err := self.control("videostream.asf", EmptyParams)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	fout, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer fout.Close()
	fin := TimeLimitReader(resp.Body, duration)
	_, err = io.Copy(fout, fin)
	if err != nil {
		return
	}
	transcoder := &FFMpegTranscoder{}
	filename, err = transcoder.Transcode(filename)
	return
}

var IrCodes = map[bool]string{true: "95", false: "94"}

func (self *Foscam) Ir(b bool) (err error) {
	resp, err := self.control("decoder_control.cgi", map[string]string{"command": IrCodes[b]})
	if err != nil {
		return
	}
	resp.Body.Close()
	return
}

func (self *Foscam) Detect(b bool) error {
	return nil
}
