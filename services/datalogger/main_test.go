package datalogger

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*DataloggerService)(nil)
	var _ services.ConfigSubscriber = (*DataloggerService)(nil)
	// Output:
}
