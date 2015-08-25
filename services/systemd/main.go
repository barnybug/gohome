// Service for launching and restarting service managed through systemd.
//
// See the gohome command line utility for controlling this.
package systemd

import (
	"os/exec"
	"strings"

	"github.com/barnybug/gohome/services"
)

// Service daemon
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "systemd"
}

func (self *Service) Run() error {
	select {} // sleep forever
	return nil
}

func (self *Service) QueryHandlers() services.QueryHandlers {
	return services.QueryHandlers{
		"ps":      services.TextHandler(self.queryStatus),
		"status":  services.TextHandler(self.queryStatus),
		"start":   services.TextHandler(self.queryStartStopRestart),
		"stop":    services.TextHandler(self.queryStartStopRestart),
		"restart": services.TextHandler(self.queryStartStopRestart),
		"help": services.StaticHandler("" +
			"status: get status\n" +
			"ps: alias for 'status'\n" +
			"start process: start a process\n" +
			"stop process: stop a process\n" +
			"restart process: restart a process\n"),
	}
}

func allProcesses() []string {
	var ret []string
	for key, _ := range services.Config.Processes {
		ret = append(ret, key)
	}
	return ret
}

func systemctl(first string, args ...string) string {
	cmdarg := append([]string{"--user", first}, args...)
	cmd := exec.Command("systemctl", cmdarg...)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func (self *Service) queryStatus(q services.Question) string {
	// ps := allProcesses()
	return systemctl("status")
}

func (self *Service) queryStartStopRestart(q services.Question) string {
	args := strings.Split(q.Args, " ")
	if len(args) == 0 {
		return "Expected a process argument"
	}

	pnames := []string{}
	for _, n := range args {
		pnames = append(pnames, "gohome@"+n+".service")
	}
	return systemctl(q.Verb, pnames...)
}
