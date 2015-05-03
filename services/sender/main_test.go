package sender

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*SenderService)(nil)
	// Output:
}
