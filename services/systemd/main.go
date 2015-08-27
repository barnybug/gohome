// Service for launching and restarting service managed through systemd.
//
// See the gohome command line utility for controlling this.
package systemd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

// Service daemon
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "systemd"
}

func (self *Service) Run() error {
	// tail logs and retransmit under topic: log
	journalTailer()
	return nil
}

func journalTailer() {
	cmd := exec.Command("journalctl", "--user-unit=gohome@*", "-f", "-n0", "-q", "--output=json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var data map[string]interface{}
		err := json.Unmarshal([]byte(scanner.Text()), &data)
		if err != nil {
			log.Println("Error decoding json:", err)
			continue
		}

		if message, ok := data["MESSAGE"].(string); ok {
			var unit string
			if user_unit, ok := data["_SYSTEMD_USER_UNIT"].(string); ok {
				user_unit = strings.Replace(user_unit, "gohome@", "", 1)
				user_unit = strings.Replace(user_unit, ".service", "", 1)
				unit = user_unit
			} else {
				unit = "systemd"
			}
			message = fmt.Sprintf("[%s] %s", unit, message)
			fields := map[string]interface{}{
				"message": message,
			}
			ev := pubsub.NewEvent("log", fields)
			services.Publisher.Emit(ev)
		}
	}
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
