// Service to interface with espeak, providing text to speech.
//
// This will relay events on the 'alert' topic to espeak, taking the text from
// the field 'message'.
package espeaker

import (
	"github.com/barnybug/gohome/services"
	"io"
	"log"
	"os/exec"
	"strings"
)

var espeakStdin io.WriteCloser

func say(msg string) {
	log.Println("Saying:", msg)
	data := []byte(msg + "\n")
	_, err := espeakStdin.Write(data)
	if err != nil {
		log.Println("Error Writing to stdin:", err)
	}
}

type EspeakerService struct {
}

func (self *EspeakerService) Id() string {
	return "espeaker"
}

func (self *EspeakerService) Run() error {
	// start espeak process
	args := strings.Split(services.Config.Espeak.Args, " ")
	cmd := exec.Command("espeak", args...)
	var err error
	espeakStdin, err = cmd.StdinPipe()
	if err != nil {
		log.Fatalln("Couldn't create StdinPipe:", err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatalln("Couldn't start espeak:", err)
	}

	for ev := range services.Subscriber.FilteredChannel("alert") {
		msg, ok := ev.Fields["message"].(string)
		if ev.Target() == "espeak" && ok {
			say(msg)
		}
	}
	return nil
}
