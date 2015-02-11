// Service for launching and restarting other services. Like upstart/systemd, but simpler.
//
// See the gohome command line utility for controlling this.
package daemon

import "github.com/barnybug/gohome/processes"

type DaemonService struct{}

func (self *DaemonService) Id() string {
	return "daemon"
}

func (self *DaemonService) Run() error {
	processes.Daemon()
	return nil
}
