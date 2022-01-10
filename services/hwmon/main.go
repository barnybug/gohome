// Service to track hardware stats (currently just temperatures)
package hwmon

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	linuxproc "github.com/c9s/goprocinfo/linux"
)

// Service hwmon
type Service struct {
	device string
	zones  map[string]string
}

// ID of the service
func (self *Service) ID() string {
	return "hwmon"
}

var reThermalZone = regexp.MustCompile(`thermal_zone(\d+)`)

func thermalZoneFieldName(path string) string {
	matches := reThermalZone.FindStringSubmatch(path)
	i, err := strconv.Atoi(matches[1])
	if err != nil {
		log.Fatal(err)
	}
	if i == 0 {
		return "temp"
	}
	return fmt.Sprintf("temp%d", i)
}

func deviceName() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("system.%s", hostname)
}

func readTemp(path string) (float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var temp float64
	_, err = fmt.Fscanf(f, "%f", &temp)
	if err != nil {
		return 0, err
	}
	return temp / 1000, nil
}

func readTemps(device string, zones map[string]string) {
	fields := pubsub.Fields{"device": device}
	for field, path := range zones {
		temp, err := readTemp(path)
		if err != nil {
			fmt.Printf("error reading %s: %s\n", path, err)
			continue
		}
		fields[field] = temp
	}
	ev := pubsub.NewEvent("temp", fields)
	services.Publisher.Emit(ev)
}

func findThermalDevices() (zones map[string]string, err error) {
	zones = map[string]string{}
	matches, err := filepath.Glob("/sys/devices/virtual/thermal/thermal_zone?/temp")
	if err != nil {
		return
	}
	for _, match := range matches {
		zones[thermalZoneFieldName(match)] = match
	}
	return
}

func readCPUStats(device string) {
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		log.Println("Reading cpu stats failed")
		return
	}
	cpu := stat.CPUStatAll
	busy := cpu.User + cpu.Nice + cpu.System + cpu.IRQ + cpu.SoftIRQ + cpu.Steal
	total := cpu.Idle + cpu.IOWait + busy
	fields := pubsub.Fields{
		"device":  device,
		"busy":    busy,
		"total":   total,
		"irq":     cpu.IRQ,
		"nice":    cpu.Nice,
		"softirq": cpu.SoftIRQ,
		"steal":   cpu.Steal,
		"system":  cpu.System,
		"user":    cpu.User,
	}
	ev := pubsub.NewEvent("load", fields)
	services.Publisher.Emit(ev)
}

func readLoadAvg(device string) {
	load, err := linuxproc.ReadLoadAvg("/proc/loadavg")
	if err != nil {
		log.Println("Reading loadavg failed")
		return
	}
	fields := pubsub.Fields{
		"device": device, "load_1m": load.Last1Min, "load_5m": load.Last5Min, "load_15": load.Last15Min,
		"running_processes": load.ProcessRunning, "total_processes": load.ProcessTotal}
	ev := pubsub.NewEvent("load", fields)
	services.Publisher.Emit(ev)
}

func readMem(device string) {
	mem, err := linuxproc.ReadMemInfo("/proc/meminfo")
	if err != nil {
		log.Println("Reading meminfo failed")
		return
	}
	fields := pubsub.Fields{
		"device":    device,
		"free":      mem.MemFree,
		"total":     mem.MemTotal,
		"available": mem.MemAvailable,
		"buffers":   mem.Buffers,
		"cached":    mem.Cached,
	}
	ev := pubsub.NewEvent("mem", fields)
	services.Publisher.Emit(ev)
}

type Mount struct {
	Filesystem              string
	Blocks, Used, Available int64
	Use                     string
	Mount                   string
}

func parseDfLine(line string) (Mount, error) {
	ret := Mount{}
	_, err := fmt.Sscanf(line, "%s %d %d %d %s %s", &ret.Filesystem, &ret.Blocks, &ret.Used, &ret.Available, &ret.Use, &ret.Mount)
	return ret, err
}

var ignoreFilesystems = map[string]bool{
	"dev":            true,
	"run":            true,
	"tmpfs":          true,
	"devtmpfs":       true,
	"overlay":        true,
	"overlayfs-root": true,
}

func readDf(device string) {
	cmd := exec.Command("df")
	stdout, _ := cmd.StdoutPipe()
	err := cmd.Start()
	if err != nil {
		log.Println("Reading df failed")
		return
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Scan() // skip header
	for scanner.Scan() {
		parsed, err := parseDfLine(scanner.Text())
		if err != nil {
			log.Printf("Failed to parse line: %s", scanner.Text())
			continue
		}
		if ignoreFilesystems[parsed.Filesystem] {
			continue
		}
		name := strings.ReplaceAll(strings.Trim(parsed.Mount, "/"), "/", "_")
		if name == "" {
			name = "root"
		}
		fields := pubsub.Fields{
			"device":    device + ":" + name,
			"used":      parsed.Used,
			"available": parsed.Available,
			"total":     parsed.Blocks,
		}
		ev := pubsub.NewEvent("mount", fields)
		services.Publisher.Emit(ev)
	}
	cmd.Wait()
}

func readUptime(device string) {
	uptime, err := linuxproc.ReadUptime("/proc/uptime")
	if err != nil {
		log.Println("Reading uptime failed")
		return
	}
	fields := pubsub.Fields{
		"device": device, "uptime": uptime.Total}
	ev := pubsub.NewEvent("uptime", fields)
	services.Publisher.Emit(ev)
}

func (self *Service) sample() {
	readTemps(self.device, self.zones)
	readCPUStats(self.device)
	readLoadAvg(self.device)
	readMem(self.device)
	readDf(self.device)
	readUptime(self.device)
}

func (self *Service) Init() error {
	// does not need config
	self.device = deviceName()
	var err error
	self.zones, err = findThermalDevices()
	if err != nil {
		return err
	}
	log.Printf("%d thermal zones", len(self.zones))
	return nil
}

// Run the service
func (self *Service) Run() error {
	self.sample()
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		self.sample()
	}
	return nil
}
