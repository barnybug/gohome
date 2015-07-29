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

func writeTable(table [][]string) string {
	var out string
	lengths := map[int]int{}
	for _, row := range table {
		for i, value := range row {
			if len(value) > lengths[i] {
				lengths[i] = len(value)
			}
		}
	}

	for _, row := range table {
		for i, value := range row {
			format := fmt.Sprintf("%%-%ds", lengths[i]+1)
			out += fmt.Sprintf(format, value)
		}
		out += "\n"
	}
	return out
}

func allProcesses() []string {
	var ret []string
	for key, _ := range services.Config.Processes {
		ret = append(ret, key)
	}
	return ret
}

func (self *Service) queryStatus(q services.Question) string {
	ps := allProcesses()
	sort.Strings(ps)

	running := processes.GetRunning()
	table := [][]string{
		[]string{"Process", "Status", "PID", "Started"},
	}
	for _, name := range ps {
		pinfo := running[name]
		if pinfo.Pid == 0 {
			table = append(table, []string{name, "stopped", "", ""})
		} else {
			table = append(table, []string{name, "running", fmt.Sprint(pinfo.Pid), pinfo.Started})
		}
	}
	return writeTable(table)
}
