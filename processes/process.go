package processes

import (
	"fmt"
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PidTime struct {
	Pid     int
	Started string
}

func GetRunning() map[string]PidTime {
	ret := map[string]PidTime{}
	// scan process list for matching processes
	out, _ := exec.Command("ps", "x", "-o", "pid,start_time,command").Output()
	lines := strings.Split(string(out), "\n")
	// ps pads and right aligns columns, so figure out position of next by
	// position of previous header
	header := lines[0]
	startcol := strings.Index(header, "PID") + 4
	commandcol := strings.Index(header, "START") + 6

	for _, line := range lines[1:] {
		if len(line) < commandcol {
			continue
		}
		pid := strings.TrimSpace(line[0 : startcol-1])
		start := strings.TrimSpace(line[startcol : commandcol-1])
		command := line[commandcol:]
		for name, process := range services.Config.Processes {
			i := strings.Index(command, process.Cmd)
			if i == 0 {
				pid, _ := strconv.ParseInt(pid, 10, 32)
				ret[name] = PidTime{Pid: int(pid), Started: start}
			}
		}
	}

	return ret
}

func processSpec(cf *config.ProcessConf) (fpath string, args []string, pattr *os.ProcAttr) {
	args = strings.Split(cf.Cmd, " ")
	fpath = args[0]
	if cf.Path != "" {
		fpath = path.Join(cf.Path, args[0])
	} else {
		var err error
		fpath, err = exec.LookPath(args[0])
		if err != nil {
			log.Fatalln(err)
		}
	}
	pattr = &os.ProcAttr{Dir: cf.Path}
	return
}

func logName(name string) (log string, err string) {
	log = path.Join(util.ExpandUser("~/go/log"), name+".log")
	err = path.Join(util.ExpandUser("~/go/log"), name+".err")
	return log, err
}

func Start(ps []string) []int {
	running := GetRunning()
	pids := []int{}
	IterateServices(ps, func(name string, cf config.ProcessConf) {
		if _, ok := running[name]; ok {
			log.Println(name, "already running")
		} else {
			fpath, args, pattr := processSpec(&cf)
			// open log file
			logname, errname := logName(name)
			logfile, err := os.OpenFile(logname, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
			if err != nil {
				log.Println("Couldn't open log file:", err)
				return
			}
			errfile, err := os.OpenFile(errname, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
			if err != nil {
				log.Println("Couldn't open log file:", err)
				return
			}
			// connect stdout/stderr to log file
			pattr.Files = []*os.File{nil, logfile, errfile}
			ps, err := os.StartProcess(fpath, args, pattr)
			if err == nil {
				log.Printf("Started %s (pid: %d)\n", name, ps.Pid)
				pids = append(pids, ps.Pid)
				go func() {
					// collect zombies
					ps.Wait()
				}()
			} else {
				log.Println("Error starting", name, err)
			}
		}
	})
	return pids
}

func Stop(ps []string) {
	if len(ps) == 0 {
		ps = allProcesses()
	}
	running := GetRunning()
	IterateServices(ps, func(name string, cf config.ProcessConf) {
		if pinfo, ok := running[name]; ok {
			p, err := os.FindProcess(pinfo.Pid)
			if err == nil {
				p.Signal(os.Interrupt)
				p.Wait()
				log.Println("Stopped", name)
			} else {
				log.Println("Error stopping", name, err)
			}
		} else {
			log.Println(name, "not running")
		}
	})
}

func Restart(ps []string) {
	Stop(ps)
	Start(ps)
}

func printTable(table [][]string) {
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
			fmt.Printf(format, value)
		}
		fmt.Println()
	}
}

func allProcesses() []string {
	var ret []string
	for key, _ := range services.Config.Processes {
		ret = append(ret, key)
	}
	return ret
}

func Status(ps []string) {
	if len(ps) == 0 {
		ps = allProcesses()
	}
	sort.Strings(ps)

	running := GetRunning()
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
	printTable(table)
}

func Run(ps []string) {
	running := GetRunning()
	IterateServices(ps, func(name string, cf config.ProcessConf) {
		if running[name].Pid == 0 {
			fmt.Print("Running ", name, "...\n")
			fpath, args, pattr := processSpec(&cf)
			pattr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}
			p, err := os.StartProcess(fpath, args, pattr)
			if err == nil {
				p.Wait()
			} else {
				fmt.Println("Error running", name, err)
			}
		} else {
			fmt.Println(name, "already running")
		}
	})
}

func IterateServices(ps []string, fn func(string, config.ProcessConf)) {
	for _, name := range ps {
		serviceconf := services.Config.Processes[name]
		fn(name, serviceconf)
	}
}

func Daemon() {
	running := GetRunning()
	for name, pid := range running {
		log.Printf("Found %s (pid: %d)\n", name, pid.Pid)
	}

	for {
		current := GetRunning()
		// check for dead processes
		for name, pid := range running {
			if current[name].Pid == 0 {
				log.Printf("Died %s (pid: %d)\n", name, pid.Pid)
			}
		}

		for name := range services.Config.Processes {
			// start any missing
			if running[name].Pid == 0 {
				pids := Start([]string{name})
				if len(pids) > 0 {
					current[name] = PidTime{Pid: pids[0], Started: ""}
				}
			}
		}
		running = current
		time.Sleep(5 * time.Second)
	}
}

func Logs(ps []string) {
	if len(ps) == 0 {
		ps = allProcesses()
	}

	args := []string{"tail", "-f"}
	IterateServices(ps, func(name string, cf config.ProcessConf) {
		l, e := logName(name)
		if _, err := os.Stat(l); err == nil {
			args = append(args, l)
		}
		if _, err := os.Stat(e); err == nil {
			args = append(args, e)
		}
	})
	pattr := os.ProcAttr{Files: []*os.File{nil, os.Stdout, os.Stderr}}
	tailpath, _ := exec.LookPath("tail")
	p, err := os.StartProcess(tailpath, args, &pattr)
	if err == nil {
		fmt.Println("Tailing logs...")
		p.Wait()
	} else {
		fmt.Println("Error tailing logs", err)
	}
}

const MAX_LOGS = 10

func rotateLog(logName string) {
	if _, err := os.Stat(logName); err != nil {
		return
	}
	for i := MAX_LOGS; i > 0; i-- {
		newName := fmt.Sprintf("%s.%d", logName, i)
		if i > 1 {
			oldName := fmt.Sprintf("%s.%d", logName, i-1)
			if _, err := os.Stat(oldName); err == nil {
				fmt.Printf("Renaming log %s->%s\n", oldName, newName)
				os.Rename(oldName, newName)
			}
		} else {
			// This newest log will be open and being written, so copy
			// contents and truncate the file in place. There is a small race
			// condition in doing it this way between the copy and truncate,
			// but we'll live with it.
			oldName := logName
			if _, err := os.Stat(oldName); err == nil {
				fmt.Printf("Copy/truncating %s->%s\n", oldName, newName)
				in, err := os.OpenFile(oldName, os.O_RDWR, 0644)
				if err != nil {
					fmt.Println("Error opening:", oldName, err)
					return
				}
				defer in.Close()

				out, err := os.OpenFile(newName, os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Println("Error opening:", newName, err)
					return
				}
				defer out.Close()

				_, err = io.Copy(out, in)
				if err != nil {
					fmt.Println("Error copying:", oldName, err)
					return
				}
				out.Sync()
				// optional: could gzip file at this point too

				// finally truncate existing file
				err = in.Truncate(0)
				if err != nil {
					fmt.Println("Error truncating:", oldName, err)
					return
				}
			}
		}
	}

}

// Rotate the logs .log -> .log.1, .log.1 -> .log.2, etc.
func Rotate(ps []string) {
	if len(ps) == 0 {
		ps = allProcesses()
	}
	IterateServices(ps, func(name string, cf config.ProcessConf) {
		l, e := logName(name)
		rotateLog(l)
		rotateLog(e)
	})
}
