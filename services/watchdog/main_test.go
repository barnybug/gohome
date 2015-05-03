package watchdog

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*WatchdogService)(nil)
	// Output:
}
