package main

import "os"

// func ExampleQuery() {
// 	query([]string{""})
// 	// Output:
// }

func ExampleHelp() {
	os.Args = []string{"gohome", "help"}
	main()
	// Output:
	// abc
}
