// Example of retrieving stored messages.
package main

import (
	"fmt"

	"github.com/barnybug/gogsmmodem"
	"github.com/tarm/serial"
)

func main() {
	conf := serial.Config{Name: "/dev/ttyUSB1", Baud: 115200}
	modem, err := gogsmmodem.Open(&conf, true)
	if err != nil {
		panic(err)
	}

	li, _ := modem.ListMessages("ALL")
	fmt.Printf("%d stored messages\n", len(*li))
	for _, msg := range *li {
		fmt.Println(msg.Index, msg.Status, msg.Body)
	}
}
