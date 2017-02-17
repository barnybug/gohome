package hwmon

import (
	"path/filepath"
	"testing"

	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
)

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	// Output:
}

func hasThermalZones() bool {
	matches, _ := filepath.Glob("/sys/devices/virtual/thermal/thermal_zone?/temp")
	return len(matches) > 0
}

func TestFindThermalDevices(t *testing.T) {
	if !hasThermalZones() {
		return
	}
	zones, err := findThermalDevices()
	assert.NoError(t, err, "No error returned")
	assert.NotEmpty(t, zones, "Zones should contain entries")
}

func TestReadTemp(t *testing.T) {
	if !hasThermalZones() {
		return
	}
	matches, err := filepath.Glob("/sys/devices/virtual/thermal/thermal_zone?/temp")
	assert.NoError(t, err, "No error returned")

	temp, err := readTemp(matches[0])
	assert.NoError(t, err, "No error returned")
	assert.NotZero(t, temp, "Temp is non-zero")
}
