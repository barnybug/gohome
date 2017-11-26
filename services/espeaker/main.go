// Service to interface with espeak, providing text to speech.
//
// This will relay events on the 'alert' topic to espeak, taking the text from
// the field 'message'.
package espeaker

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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

func speakEndpoint(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("text")
	if text == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing text parameter")
		return
	}

	log.Println("Streaming:", text)

	args := strings.Split(services.Config.Espeak.Args, " ")
	args = append(args, "--stdout")
	args = append(args, text)
	cmd := exec.Command("espeak", args...)
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		log.Println("Error opening espeak: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error")
		return
	}
	w.Header().Add("Content-Type", "audio/x-wav")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, stdout)
	stdout.Close()
	cmd.Wait()
}

func startWebserver() {
	http.HandleFunc("/speak", speakEndpoint)
	addr := fmt.Sprintf(":%d", services.Config.Espeak.Port)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("Webserver failed to start: ", err)
	}
}

// Run the service
func (self *Service) Run() error {
	go startWebserver()

	for ev := range services.Subscriber.FilteredChannel("alert") {
		msg, ok := ev.Fields["message"].(string)
		if ev.Target() == "espeak" && ok {
			say(msg)
		}
	}
	return nil
}
