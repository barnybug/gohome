package datalogger

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = &DataloggerService{}
	var _ services.ConfigSubscriber = &DataloggerService{}
	// Output:
}
