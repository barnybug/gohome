// Service for launching and restarting service managed through systemd.
//
// See the gohome command line utility for controlling this.
package systemd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
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
	cmd := exec.Command("journalctl", "--user-unit=gohome*.service", "--user-unit=gohome*.slice", "-f", "-n0", "-q", "--output=json")
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
			var source string
			if user_unit, ok := data["_SYSTEMD_USER_UNIT"].(string); ok {
				source = stripUnitName(user_unit)
			} else {
				source = "systemd"
			}
			fields := map[string]interface{}{
				"message": message,
				"source":  source,
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

func systemctl(first string, args ...string) string {
	cmdarg := append([]string{"--user", first}, args...)
	cmd := exec.Command("systemctl", cmdarg...)
	out, _ := cmd.CombinedOutput()
	return string(out)
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

type ByStatus []UnitStatus

func (a ByStatus) Len() int {
	return len(a)
}

func (a ByStatus) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByStatus) Less(i, j int) bool {
	x := strings.Compare(a[i].Status, a[j].Status)
	y := strings.Compare(a[i].Process, a[j].Process)
	return (x == 0 && y < 0) || (x != 0 && x > 0)
}

func (self *Service) queryStatus(q services.Question) string {
	host, _ := os.Hostname()
	table := [][]string{
		[]string{"Process", "Host", "Status", "PID", "Started"},
	}
	units := getStatus()
	sort.Sort(ByStatus(units))
	for _, unit := range units {
		table = append(table, []string{unit.Process, host, unit.Status, unit.MainPid, unit.Started})
	}
	return writeTable(table)
}

type UnitStatus struct {
	Process string
	Status  string
	MainPid string
	Started string
}

func parseShowOutput(reader io.Reader) []UnitStatus {
	scanner := bufio.NewScanner(reader)
	results := []UnitStatus{}
	current := UnitStatus{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			results = append(results, current)
			current = UnitStatus{}
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "Id":
			current.Process = stripUnitName(parts[1])
		case "MainPID":
			if parts[1] == "0" {
				parts[1] = ""
			}
			current.MainPid = parts[1]
		case "ActiveState":
			if parts[1] == "active" {
				parts[1] = "running"
			}
			current.Status = parts[1]
		case "ExecMainStartTimestamp":
			current.Started = parts[1]
		}
	}
	if current.Process != "" {
		results = append(results, current)
	}
	return results
}

func getStatus() []UnitStatus {
	cmd := exec.Command("systemctl", "--user", "show", "--property=Id,MainPID,ActiveState,ExecMainStartTimestamp", "gohome@*")
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		log.Println(err)
		return []UnitStatus{}
	}
	return parseShowOutput(stdout)
}

func stripUnitName(s string) string {
	return strings.Replace(
		strings.Replace(s, "gohome@", "", 1),
		".service", "", 1)
}

func (self *Service) queryStartStopRestart(q services.Question) string {
	args := strings.Split(q.Args, " ")
	if len(args) == 0 {
		return "Expected a process argument"
	}

	units := getStatus()
	names := map[string]bool{}
	for _, unit := range units {
		names[unit.Process] = true
	}

	pnames := []string{}
	fnames := []string{}
	for _, n := range args {
		// ignore any units not directed to us
		if _, ok := names[n]; ok {
			pnames = append(pnames, "gohome@"+n+".service")
			fnames = append(fnames, n)
		}
	}
	if len(pnames) == 0 {
		// no units
		return ""
	}
	systemctl(q.Verb, pnames...)
	return fmt.Sprintf("%sed %s", q.Verb, strings.Join(fnames, ", "))
}
