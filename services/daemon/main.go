// Service for launching and restarting other services. Like upstart/systemd, but simpler.
//
// See the gohome command line utility for controlling this.
package daemon

import (
	"fmt"
	"sort"
	"strings"

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
		"status":  services.TextHandler(self.queryStatus),
		"start":   services.TextHandler(self.queryStartStopRestart),
		"stop":    services.TextHandler(self.queryStartStopRestart),
		"restart": services.TextHandler(self.queryStartStopRestart),
		"help": services.StaticHandler("" +
			"status: get status\n" +
			"start process: start a process\n" +
			"stop process: stop a process\n" +
			"restart process: restart a process\n"),
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

type StringLogger struct {
	output string
}

func (logger *StringLogger) Println(args ...interface{}) {
	logger.output += fmt.Sprintln(args...)
}

func (logger *StringLogger) Printf(format string, args ...interface{}) {
	logger.output += fmt.Sprintf(format, args...)
}

func (self *Service) queryStartStopRestart(q services.Question) string {
	args := strings.Split(q.Args, " ")
	if len(args) == 0 {
		return "Expected a process argument"
	}

	out := &StringLogger{}
	switch q.Verb {
	case "start":
		processes.Start(args, out)
	case "stop":
		processes.Stop(args, out)
	case "restart":
		processes.Restart(args, out)
	}
	return fmt.Sprintf(out.output)
}
