package config

import "fmt"

var yml = `
general:
  email:
    admin:
      test@example.com

caps:
  light: [switch]
devices:
  alarm.house:
    name: House alarm
    watchdog: 12h
  light.kitchen:
    name: Kitchen light
    watchdog: 7d
  light.glowworm:
    name: Glowworm
    caps: [dimmer]
`

func ExampleOpenRaw() {
	config, _ := OpenRaw([]byte(yml))
	fmt.Println(config.General.Email.Admin)
	// Output:
	// test@example.com
}

func ExampleCaps() {
	config, _ := OpenRaw([]byte(yml))
	fmt.Println(config.Devices["alarm.house"].Id)
	fmt.Println(config.Devices["alarm.house"].Caps)
	fmt.Println(config.Devices["light.kitchen"].Id)
	fmt.Println(config.Devices["light.kitchen"].Caps)
	fmt.Println(config.Devices["light.glowworm"].Id)
	fmt.Println(config.Devices["light.glowworm"].Caps)
	// Output:
	// alarm.house
	// [alarm]
	// light.kitchen
	// [switch]
	// light.glowworm
	// [switch dimmer]
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
