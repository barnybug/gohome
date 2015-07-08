// Service for launching and restarting other services. Like upstart/systemd, but simpler.
//
// See the gohome command line utility for controlling this.
package daemon

import (
	"fmt"
	"sort"

	"github.com/barnybug/gohome/processes"
	"github.com/barnybug/gohome/services"
)

// Service daemon
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "daemon"
}

func (self *Service) Run() error {
	processes.Daemon()
	return nil
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"status": services.TextHandler(self.queryStatus),
		"help":   services.StaticHandler("status: get status\n"),
	}
}

func (self *Service) queryStatus(q services.Question) string {
	running := processes.GetRunning()
	var out string
	var names []string
	for k := range running {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		pid := running[name]
		out += fmt.Sprintf("- %s [%d] since %s\n", name, pid.Pid, pid.Started)
	}
	return out
}
