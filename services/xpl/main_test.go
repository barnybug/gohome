package xpl

import (
	"encoding/json"
	"os"

	"fmt"
)

const message = "xpl-stat\n{\nhop=1\nsource=slimdev-slimserv.FrontroomTouch\ntarget=*\n}\naudio.basic\n{\nstatus=stopped\nARTIST= \nALBUM= \nTRACK= \nPOWER=1\n}\n"

const otherMessage = "xpl-stat\n{\nhop=1\nsource=slimdev-slimserv.FrontroomTouch\ntarget=*\n}\n}\n"

func ExampleParse() {
	result := Parse(message)
	b, _ := json.Marshal(result)
	os.Stdout.Write(b)
	// Output:
	// {"audio.basic":{"ALBUM":" ","ARTIST":" ","POWER":"1","TRACK":" ","status":"stopped"},"xpl-stat":{"hop":"1","source":"slimdev-slimserv.FrontroomTouch","target":"*"}}
}

func ExampleProcess() {
	source, power := Process(message)
	fmt.Println(source, power)
	// Output: slimdev-slimserv.FrontroomTouch 1
}

func ExampleProcessOther() {
	source, power := Process(otherMessage)
	fmt.Println(source, power)
	// Output: slimdev-slimserv.FrontroomTouch
}
