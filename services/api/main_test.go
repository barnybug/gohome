package api

import (
	"fmt"
	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub/dummy"
	"github.com/barnybug/gohome/services"
	"net/http"
	"net/http/httptest"
	"net/url"
)

func ExampleInterfaces() {
	var _ services.Service = &ApiService{}
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
	// {"light.kitchen":{"id":"light.kitchen","name":"Kitchen","type":"light","group":"downstairs","state":null}}
}

func ExampleDevicesEvents() {
	services.Config = config.ExampleConfig
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiDevicesEvents(rec, &r)
	fmt.Println(rec.Body)
	// Output:
	// {}
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
