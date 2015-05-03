package camera

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = (*CameraService)(nil)
	var _ services.ConfigSubscriber = (*CameraService)(nil)
	// Output:
}
