package graphite

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*GraphiteService)(nil)
	// Output:
}
