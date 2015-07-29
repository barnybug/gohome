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
	// Usage: gohome COMMAND [PROCESS/SERVICE]
	//
	// Commands:
	//    logs    Tail logs (all or select)
	//    restart Restart a process
	//    rotate  Rotate logs
	//    run     Run a process (foreground)
	//    service Execute a builtin service
	//    start   Start a process
	//    status  Get process status
	//    stop    Stop a process
	//    query   Query services
}
