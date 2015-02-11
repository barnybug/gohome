package espeaker

import "github.com/barnybug/gohome/services"

func ExampleInterfaces() {
	var _ services.Service = &EspeakerService{}
	// Output:
}
