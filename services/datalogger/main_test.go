package datalogger

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	// Output:
}
