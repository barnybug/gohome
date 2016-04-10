package camera

import (
	"os/exec"
	"path/filepath"
	"strings"
)

type Transcoder interface {
	Transcode(in string) (out string, err error)
}

func stripExt(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// Transcoder using VLC to transcode to .webm format
type VLCTranscoder struct{}

func (self *VLCTranscoder) Transcode(in string) (out string, err error) {
	out = stripExt(in) + ".webm"
	cmd := exec.Command("/usr/bin/vlc",
		"-Idummy", in,
		"--sout=#transcode{vcodec=VP80,vb=800,scale=1,acodec=vorb,ab=128,channels=1,samplerate=44100}:file{dst='"+out+"'}",
		"vlc://quit")
	err = cmd.Run()
	return
}

// Transcoder using ffmpeg to transcode to .avi format
type FFMpegTranscoder struct{}

func (self *FFMpegTranscoder) Transcode(in string) (out string, err error) {
	out = stripExt(in) + ".avi"
	cmd := exec.Command("/usr/bin/ffmpeg",
        "-i", in,
        "-c:a", "copy", "-c:v", "copy",
        out)
	err = cmd.Run()
	return
}

// Test transcoder - just copies
type DummyTranscoder struct{}

func (self *DummyTranscoder) Transcode(in string) (out string, err error) {
	out = stripExt(in) + ".copy"
	cmd := exec.Command("/usr/bin/cp",
		in, out)
	err = cmd.Run()
	return
}
