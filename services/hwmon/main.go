// Service to track hardware stats (currently just temperatures)
package hwmon

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

// Service hwmon
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "hwmon"
}

var reNumber = regexp.MustCompile(`(\d+)`)

func deviceName(path string) string {
	nums := reNumber.FindAllString(path, -1)
	i, err := strconv.Atoi(nums[len(nums)-1])
	if err != nil {
		log.Fatal(err)
	}

	hostname, _ := os.Hostname()
	return fmt.Sprintf("thermal.%s%d", hostname, i)
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

func readTemps(zones map[string]string) {
	for name, path := range zones {
		temp, err := readTemp(path)
		if err != nil {
			fmt.Printf("error reading %s: %s\n", path, err)
			continue
		}

		ev := pubsub.NewEvent("temp",
			pubsub.Fields{"temp": temp, "device": name})
		services.Publisher.Emit(ev)
	}
}

func findThermalDevices() (zones map[string]string, err error) {
	zones = map[string]string{}
	matches, err := filepath.Glob("/sys/devices/virtual/thermal/thermal_zone?/temp")
	if err != nil {
		return
	}
	for _, match := range matches {
		zones[deviceName(match)] = match
	}
	return
}

// Run the service
func (self *Service) Run() error {
	zones, err := findThermalDevices()
	if err != nil {
		return err
	}
	log.Printf("%d thermal zones", len(zones))

	readTemps(zones) // initial read
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		readTemps(zones)
	}
	return nil
}
