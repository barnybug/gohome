package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub/dummy"
	"github.com/barnybug/gohome/services"
)

func TestInterfaces(t *testing.T) {
	var _ services.Service = (*Service)(nil)
}

func TestIndex(t *testing.T) {
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiIndex(rec, &r)
	assert.Equal(t, rec.Body.String(), `<html>Gohome is listening</html>`)
}

func TestDevices(t *testing.T) {
	services.Config = config.ExampleConfig
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiDevices(rec, &r)
	assert.Contains(t, rec.Body.String(), `"light.kitchen":{"aliases":null,"caps":["switch"],"events":{},"group":"downstairs","id":"light.kitchen","name":"Kitchen","type":"switch"}`)
}

func TestDevicesSingle(t *testing.T) {
	services.Config = config.ExampleConfig
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiDevicesSingle(rec, &r, map[string]string{"device": "light.kitchen"})
	assert.Equal(t, rec.Body.String(), `{"aliases":null,"caps":["switch"],"events":{},"group":"downstairs","id":"light.kitchen","name":"Kitchen","type":"switch"}
`)
}

func TestDevicesSingleNotFound(t *testing.T) {
	services.Config = config.ExampleConfig
	rec := httptest.NewRecorder()
	r := http.Request{}
	apiDevicesSingle(rec, &r, map[string]string{"device": "abc"})
	assert.Equal(t, rec.Body.String(), "not found: abc")
}

func TestDevicesControl(t *testing.T) {
	services.Config = config.ExampleConfig
	me := dummy.Publisher{}
	services.Publisher = &me
	rec := httptest.NewRecorder()
	uri, _ := url.Parse("http://example.com/?id=light.kitchen")
	r := http.Request{URL: uri}
	apiDevicesControl(rec, &r)
	assert.Equal(t, rec.Body.String(), "true\n")
}

func TestDevicesControlMissing(t *testing.T) {
	services.Config = config.ExampleConfig
	me := dummy.Publisher{}
	services.Publisher = &me
	rec := httptest.NewRecorder()
	uri, _ := url.Parse("http://example.com/?id=x")
	r := http.Request{URL: uri}
	apiDevicesControl(rec, &r)
	assert.Equal(t, rec.Body.String(), "device not found\n")
}
