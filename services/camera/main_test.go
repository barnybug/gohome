package camera

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = &CameraService{}
	var _ services.ConfigSubscriber = &CameraService{}
	// Output:
}
