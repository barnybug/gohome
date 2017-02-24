package config

import "fmt"

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
