package sender

import "github.com/barnybug/gohome/services"

func ExampleService() {
	var _ services.Service = (*Service)(nil)
	// Output:
}
