package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

func config(path string, filenames []string) {
	if path != "config" && !strings.HasPrefix(path, "config/") {
		fmt.Println("Path must begin with 'config'")
		return
	}

	// concatenate files together
	data := &bytes.Buffer{}
	for _, filename := range filenames {
		f, err := os.Open(filename)
		if err != nil {
			fmt.Printf("Error opening %s: %s\n", filename, err)
			return
		}
		defer f.Close()
		_, err = io.Copy(data, f)
		if err != nil {
			fmt.Printf("Error reading %s: %s\n", filename, err)
			return
		}

		data.WriteString("\n")
		data.WriteByte('\n')
	}

	// emit event
	fields := pubsub.Fields{
		"config": string(data.Bytes()),
	}

	ev := pubsub.NewEvent(path, fields)
	ev.SetRetained(true) // config messages are retained
	services.SetupBroker()
	services.Publisher.Emit(ev)
	fmt.Printf("Updated %s (%d bytes)\n", path, data.Len())
}
