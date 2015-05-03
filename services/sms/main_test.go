package sms

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*SmsService)(nil)
	// Output:
}
