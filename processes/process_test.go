package processes

import (
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/services"

	"fmt"
)

func ExampleGetRunning() {
	services.Config = config.ExampleConfig
	fmt.Println(GetRunning())
	// Output:
	// map[]
}
