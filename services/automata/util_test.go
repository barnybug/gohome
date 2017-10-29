package automata

import "fmt"

type TestClass struct{}

func (self *TestClass) Print(msg string) {
	fmt.Println(msg)
}

func (self *TestClass) Add(a, b int64) {
	fmt.Printf("%d\n", (a + b))
}

func (self *TestClass) Not(p bool) {
	fmt.Printf("%v\n", !p)
}

func ExampleDynamicCallString() {
	obj := &TestClass{}
	err := DynamicCall(obj, `Print("message")`)
	fmt.Println(err)
	// Output:
	// message
	// <nil>
}

func ExampleDynamicCallInt() {
	obj := &TestClass{}
	DynamicCall(obj, `Add(1, 2)`)
	// Output:
	// 3
}

func ExampleDynamicCallBool() {
	obj := &TestClass{}
	DynamicCall(obj, `Not(true)`)
	// Output:
	// false
}

func ExampleBadDynamicCall() {
	obj := &TestClass{}
	err := DynamicCall(obj, `Print(123)`)
	fmt.Println(err)
	// Output:
	// Error calling: Print(123) reflect: Call using int64 as type string
}

func ExampleMissingDynamicCall() {
	obj := &TestClass{}
	err := DynamicCall(obj, `Missing(123)`)
	fmt.Println(err)
	// Output:
	// Error: Missing(123) not found
}
