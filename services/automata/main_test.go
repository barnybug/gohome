package automata

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = &AutomataService{}
	var _ services.Queryable = &AutomataService{}
	// Output:
}
