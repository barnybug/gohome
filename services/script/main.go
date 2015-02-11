// Service to launch an external script/executable and transmit events on any
// valid data the process emits to stdout. This allows easy integration of
// third-party input devices developed in any other language.
package script

import (
	"bufio"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
	"io"
	"log"
	"os"
	"os/exec"
)

var espeakStdin io.WriteCloser

type ScriptService struct {
}

func (self *ScriptService) Id() string {
	return "script"
}

func (self *ScriptService) Run() error {
	// start script
	args := os.Args
	// skip to script name and arguments
	for i := range args {
		if args[i] == "--" {
			args = args[i+1:]
			break
		}
	}
	name := util.ExpandUser(args[0])
	args = args[1:]

	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalln("Couldn't create StdoutPipe:", err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatalln("Couldn't start:", name, args)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		ev := pubsub.Parse(line)
		if ev == nil {
			log.Printf("Ignored: '%s'\n", line)
			continue
		}

		services.Publisher.Emit(ev)
	}

	cmd.Wait()

	return nil
}
