package arduino

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*ArduinoService)(nil)
	// Output:
}
