package irrigation

import (
	"fmt"
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/lib/graphite"
	"github.com/barnybug/gohome/services"
)

func ExampleColdDay() {
	services.Config = config.ExampleConfig
	response := `[{"target": "summarize(sensor.garden.temp.avg, \"1y\", \"avg\")", "datapoints": [[5.8, 1387584000]]}]`
	gr = &graphite.MockGraphite{Response: response}
	msg, _ := irrigationStats()
	fmt.Println(msg)
	// Output:
	// Irrigation: Watering garden for 10s (12h av was 5.8C)
}

func ExampleHotDay() {
	services.Config = config.ExampleConfig
	// services.Config = config.Open()
	// fmt.Println(services.Config.Irrigation)
	response := `[{"target": "summarize(sensor.garden.temp.avg, \"1y\", \"avg\")", "datapoints": [[25.0, 1387584000]]}]`
	gr = &graphite.MockGraphite{Response: response}
	msg, _ := irrigationStats()
	fmt.Println(msg)
	// Output:
	// Irrigation: Watering garden for 1m0s (12h av was 25.0C)
}
