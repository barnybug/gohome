package main

import (
	"os"
	"testing"
)

// func ExampleQuery() {
// 	query([]string{""})
// 	// Output:
// }

func TestStatus(t *testing.T) {
	os.Setenv("GOHOME_STORE", "mock:")
	os.Args = []string{"gohome", "status"}
	main()
}
