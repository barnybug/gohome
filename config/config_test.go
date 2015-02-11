package config

import (
	"encoding/json"
	"fmt"
	"github.com/barnybug/gohome/pubsub"
)

var yml = `
general:
  email:
    admin:
      test@example.com
protocols:
  x10:
    a01: device.one
`

func ExampleOpenRaw() {
	config, _ := OpenRaw([]byte(yml))
	fmt.Println(config.General.Email.Admin)
	// Output:
	// test@example.com
}

func ExampleLookupDeviceName() {
	config, _ := OpenRaw([]byte(yml))
	fields := pubsub.Fields{"source": "a01"}
	ev := pubsub.NewEvent("x10", fields)
	fmt.Println(config.LookupDeviceName(ev))
	// Output:
	// device.one
}

func ExampleLookupDeviceNameMissing() {
	config, _ := OpenRaw([]byte(yml))
	fields := pubsub.Fields{"source": "a02"}
	ev := pubsub.NewEvent("x10", fields)
	fmt.Println(config.LookupDeviceName(ev))
	// Output:
	// x10.a02
}

func ExampleLookupDeviceProtocol() {
	config := ExampleConfig
	m := config.LookupDeviceProtocol("light.glowworm")
	s, _ := json.Marshal(m)
	fmt.Println(string(s))
	// Output:
	// {"homeeasy":"00123453","x10":"b03"}
}
