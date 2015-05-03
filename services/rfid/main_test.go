package rfid

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*RfidService)(nil)
	// Output:
}
