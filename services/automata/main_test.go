package automata

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*AutomataService)(nil)
	var _ services.Queryable = (*AutomataService)(nil)
	// Output:
}
