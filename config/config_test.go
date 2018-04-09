package config

import "fmt"

var yml = `
general:
  email:
    admin:
      test@example.com
`

func ExampleOpenRaw() {
	config, _ := OpenRaw([]byte(yml))
	fmt.Println(config.General.Email.Admin)
	// Output:
	// test@example.com
}

func ExampleDeviceSourceId() {
	fmt.Println(ExampleConfig.Devices["light.kitchen"].SourceId())
	fmt.Println(ExampleConfig.Devices["light.glowworm"].SourceId())
	// Output:
	// b06
	// 00123453
}

func ExampleLookupDeviceProtocol() {
	fmt.Println(ExampleConfig.LookupDeviceProtocol("light.kitchen", "x10"))
	fmt.Println(ExampleConfig.LookupDeviceProtocol("light.kitchen", "homeeasy"))
	fmt.Println(ExampleConfig.LookupDeviceProtocol("light.glowworm", "homeeasy"))
	// Output:
	// b06 true
	//  false
	// 00123453 true
}

func ExampleDevicesByProtocol() {
	for _, device := range ExampleConfig.DevicesByProtocol("x10") {
		fmt.Println(device.Id)
	}
	// Output:
	// light.kitchen
}
