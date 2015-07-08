package automata

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	var _ services.Queryable = (*Service)(nil)
	// Output:
}
