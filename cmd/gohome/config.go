package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

func config(path, filename string) {
	if path != "config" && !strings.HasPrefix(path, "config/") {
		fmt.Println("Path must begin with 'config'")
		return
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error opening %s: %s\n", filename, err)
		return
	}

	// emit event
	fields := pubsub.Fields{
		"config": string(data),
	}
	ev := pubsub.NewEvent(path, fields)
	ev.SetRetained(true) // config messages are retained
	services.SetupBroker()
	services.Publisher.Emit(ev)
	fmt.Printf("Updated %s (%d bytes)", path, len(data))
}
