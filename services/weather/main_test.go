package weather

import (
	"fmt"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/lib/graphite"
	"github.com/barnybug/gohome/services"
)

func ExampleWeatherStatsNoData() {
	services.Config = config.ExampleConfig
	response := `
[
    {
        "target": "sensor",
        "datapoints": [
            [null,0],
            [null,0]
        ]
    }
]`
	gr = &graphite.MockGraphite{Response: response}
	s := weatherStats()
	fmt.Println(s)
	// Output:
	// Weather: I didn't get any outside temperature data yesterday!
}

func ExampleWeatherStats() {
	services.Config = config.ExampleConfig
	response := `
[
    {
        "target": "sensor",
        "datapoints": [
            [13.8,0],
            [6.7,0]
        ]
    }
]`
	gr = &graphite.MockGraphite{Response: response}
	s := weatherStats()
	fmt.Println(s)
	// Output:
	// Weather: Outside it got up to a moderate 13.8°C and went down to a hot 13.8°C in the last 24 hours.
}
