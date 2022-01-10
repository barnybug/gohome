// Service for logging events to log files on disk.
//
// They are logged to a file named 'data.log' under a directory named by the event topic.
package datalogger

import (
	"log"
	"os"
	"path"
	"strings"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
)

var (
	logDir string
)

func ensureDirectory(path string) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// create
			os.Mkdir(path, 0755)
		} else {
			log.Fatal("Could not create directory")
		}
	}
}

func writeToLogFile(ev *pubsub.Event) {
	name := ev.Topic
	p := path.Join(logDir, name)
	ensureDirectory(p)
	p = path.Join(p, "data.log")
	// reopen the log file each time, so that log rotation can happen in the
	// background.
	// TODO: could this be done more smartly by checking inode and only
	// reopening when it changes?
	fio, err := os.OpenFile(p, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		log.Println("Couldn't write file:", err)
		return
	}
	defer fio.Close()

	fio.Write(ev.Bytes())
	fio.WriteString("\n")
}

func event(ev *pubsub.Event) {
	if strings.HasPrefix(ev.Topic, "_") || strings.HasPrefix(ev.Topic, "config") {
		return
	}
	writeToLogFile(ev)
}

// Service datalogger
type Service struct {
	config *services.ConfigService
}

// ID of the service
func (self *Service) ID() string {
	return "datalogger"
}

func (self *Service) setup() {
	if self.config.Value.Datalogger.Path == "" {
		log.Fatal("datalogger path not defined")
	}
	logDir = util.ExpandUser(services.Config.Datalogger.Path)
}

func (self *Service) Init() error {
	self.config = services.WaitForConfig()
	self.setup()
	return nil
}

func (self *Service) Run() error {
	events := services.Subscriber.Subscribe(pubsub.All())
	for {
		select {
		case ev := <-events:
			if ev.Retained {
				// ignore retained events from reconnecting
				continue
			}
			event(ev)
		case <-self.config.Updated:
			self.setup()
		}
	}
}
