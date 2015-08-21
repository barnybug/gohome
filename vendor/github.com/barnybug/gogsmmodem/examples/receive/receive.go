// Example of receiving messages.
package main

import (
	"fmt"
	"log"

	"github.com/barnybug/gogsmmodem"
	"github.com/tarm/goserial"
)

func main() {
	conf := serial.Config{Name: "/dev/ttyUSB1", Baud: 115200}
	modem, err := gogsmmodem.Open(&conf, true)
	if err != nil {
		panic(err)
	}

	for packet := range modem.OOB {
		log.Printf("Received: %#v\n", packet)
		switch p := packet.(type) {
		case gogsmmodem.MessageNotification:
			log.Println("Message notification:", p)
			msg, err := modem.GetMessage(p.Index)
			if err == nil {
				fmt.Printf("Message from %s: %s\n", msg.Telephone, msg.Body)
				modem.DeleteMessage(p.Index)
			}
		}
	}
}
