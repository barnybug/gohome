package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub/dummy"
	"github.com/barnybug/gohome/services"
)

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	// Output:
}

func ExampleIndex() {
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiIndex(rec, &r)
	fmt.Println(rec.Body)
	// Output:
	// <html>Gohome is listening</html>
}

func ExampleDevices() {
	services.Stor = &services.MockStore{}
	services.Config = config.ExampleConfig
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiDevices(rec, &r)
	fmt.Println(rec.Body)
	// Output:
	// {"light.kitchen":{"aliases":null,"caps":["light"],"events":{},"group":"downstairs","id":"light.kitchen","name":"Kitchen","type":"light"}}
}

func ExampleDevicesSingle() {
	services.Config = config.ExampleConfig
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiDevicesSingle(rec, &r, map[string]string{"device": "light.kitchen"})
	fmt.Println(rec.Body)
	// Output:
	// {"aliases":null,"caps":["light"],"events":{},"group":"downstairs","id":"light.kitchen","name":"Kitchen","type":"light"}
}

func ExampleDevicesSingleNotFound() {
	services.Config = config.ExampleConfig
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiDevicesSingle(rec, &r, map[string]string{"device": "abc"})
	fmt.Println(rec.Body)
	// Output:
	// not found: abc
}

func ExampleDevicesControl() {
	services.Config = config.ExampleConfig
	me := dummy.Publisher{}
	services.Publisher = &me
	rec := httptest.NewRecorder()
	uri, _ := url.Parse("http://example.com/")
	r := http.Request{
		URL: uri,
	}
	apiDevicesControl(rec, &r)
	fmt.Println(rec.Body)
	// Output:
	// true
}
