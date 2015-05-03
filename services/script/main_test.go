package script

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*ScriptService)(nil)
	// Output:
}
