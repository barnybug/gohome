// Service to interface with espeak, providing text to speech.
//
// This will relay events on the 'alert' topic to espeak, taking the text from
// the field 'message'.
package espeaker

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/barnybug/gohome/services"
)

var espeakStdin io.WriteCloser

func say(msg string) error {
	log.Println("Saying:", msg)
	f, err := ioutil.TempFile("", "espeaker")
	if err != nil {
		return err
	}
	f.Close()
	defer os.Remove(f.Name())

	args := strings.Split(services.Config.Espeak.Args, " ")
	args = append(args, []string{"-w", f.Name()}...)
	args = append(args, msg)
	cmd := exec.Command("espeak", args...)
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("aplay", f.Name())
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// Service espeaker
type Service struct {
}

// ID of the service
func (self *Service) ID() string {
	return "espeaker"
}

// Run the service
func (self *Service) Run() error {
	for ev := range services.Subscriber.FilteredChannel("alert") {
		msg, ok := ev.Fields["message"].(string)
		if ev.Target() == "espeak" && ok {
			say(msg)
		}
	}
	return nil
}
