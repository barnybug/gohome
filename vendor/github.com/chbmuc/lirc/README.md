LIRC
====

Go client for Linux Infrared Remote Control ([LIRC](http://www.lirc.org)) package

#### Usage

```go
package main

import (
  "github.com/chbmuc/lirc"
  "log"
  "time"
)

func keyPower(event lirc.Event) {
  log.Println("Power Key Pressed")
}

func keyTV(event lirc.Event) {
  log.Println("TV Key Pressed")
}

func keyAll(event lirc.Event) {
  log.Println(event)
}

func main() {
  // Initialize with path to lirc socket
  ir, err := lirc.Init("/var/run/lirc/lircd")
  if err != nil {
    panic(err)
  }

  // Receive Commands

  // attach key press handlers
  ir.Handle("", "KEY_POWER", keyPower)
  ir.Handle("", "KEY_TV", keyTV)
  ir.Handle("", "", keyAll)

  // run the receive service
  go ir.Run()

  // Send Commands
  reply := ir.Command(`LIST DenonTuner ""`)
  log.Println(reply.DataLength, reply.Data)

  err = ir.Send("DenonTuner PROG-SCAN")
  if err != nil {
    log.Println(err)
  }
  err = ir.SendLong("DenonTuner VOL-DOWN", time.Duration(time.Second * 3))
  if err != nil {
    log.Println(err)
  }
}
```
