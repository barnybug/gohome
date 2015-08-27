package systemd

import (
	"fmt"
	"strings"
)

func ExampleInterfaces() {
	// Output:
}

func ExampleParseShowOutput() {
	reader := strings.NewReader(`MainPID=0
ExecMainStartTimestamp=Thu 2015-08-27 19:19:13 BST
Id=gohome@pubsub.service
ActiveState=failed

MainPID=21805
ExecMainStartTimestamp=Thu 2015-08-27 17:36:49 BST
Id=gohome@rfid.service
ActiveState=active
`)
	ret := parseShowOutput(reader)
	fmt.Printf("%+v\n", ret)
	// Output:
	// [{Process:pubsub Status:failed MainPid: Started:Thu 2015-08-27 19:19:13 BST} {Process:rfid Status:running MainPid:21805 Started:Thu 2015-08-27 17:36:49 BST}]
}
