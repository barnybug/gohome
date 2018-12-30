package googlehome

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
)

func TestMissingAuthorization(t *testing.T) {
	req, _ := http.NewRequest("POST", "/actions", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actionsEndpoint)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestBadAuthorization(t *testing.T) {
	req, _ := http.NewRequest("POST", "/actions", nil)
	req.Header.Add("Authorization", "xyz")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actionsEndpoint)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSync(t *testing.T) {
	services.Config = config.ExampleConfig

	body := `{"inputs":[{"intent":"action.devices.SYNC"}],"requestId":"1"}`
	req, _ := http.NewRequest("POST", "/actions", strings.NewReader(body))
	req.Header.Add("Authorization", "Bearer xyz")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actionsEndpoint)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, ApplicationJson, rr.Header().Get("Content-Type"))
	assert.Equal(t, `{"requestId":"1","payload":{"agentUserId":"gohome","devices":[{"id":"light.kitchen","type":"action.devices.types.LIGHT","traits":["action.devices.traits.OnOff"],"name":{"name":"Kitchen","nicknames":["Kitchen"]},"willReportState":false},{"id":"light.glowworm","type":"action.devices.types.LIGHT","traits":["action.devices.traits.Brightness"],"name":{"name":"Glowworm","nicknames":["Glowworm","glow worm"]},"willReportState":false},{"id":"thermostat.living","type":"action.devices.types.THERMOSTAT","traits":["action.devices.traits.TemperatureSetting"],"name":{"name":"Living room thermostat","nicknames":["Living room thermostat"]},"willReportState":false,"attributes":{"availableThermostatModes":"heat","thermostatTemperatureUnit":"C"},"roomHint":"Living Room"}]}}
`, rr.Body.String())
}

func TestQueryLight(t *testing.T) {
	services.Config = config.ExampleConfig

	body := `{"inputs":[{"intent":"action.devices.QUERY","payload":{"devices":[{"id":"light.glowworm"}]}}],"requestId":"123"}`
	req, _ := http.NewRequest("POST", "/actions", strings.NewReader(body))
	req.Header.Add("Authorization", "Bearer xyz")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actionsEndpoint)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, ApplicationJson, rr.Header().Get("Content-Type"))
	assert.Equal(t, `{"requestId":"123","payload":{"devices":{"light.glowworm":{"online":true}}}}
`, rr.Body.String())
}

func TestQueryThermostat(t *testing.T) {
	services.Config = config.ExampleConfig
	DeviceEvents["thermostat.living"] = map[string]*pubsub.Event{
		"thermostat": pubsub.NewEvent("thermostat", pubsub.Fields{"target": float64(17)}),
	}
	DeviceEvents["trv.living"] = map[string]*pubsub.Event{
		"temp": pubsub.NewEvent("temp", pubsub.Fields{"temp": float64(19.5)}),
	}

	body := `{"inputs":[{"intent":"action.devices.QUERY","payload":{"devices":[{"id":"thermostat.living"}]}}],"requestId":"123"}`
	req, _ := http.NewRequest("POST", "/actions", strings.NewReader(body))
	req.Header.Add("Authorization", "Bearer xyz")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(actionsEndpoint)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, ApplicationJson, rr.Header().Get("Content-Type"))
	assert.Equal(t, `{"requestId":"123","payload":{"devices":{"thermostat.living":{"online":true,"thermostatMode":"heat","thermostatTemperatureSetpoint":17,"thermostatTemperatureAmbient":19.5}}}}
`, rr.Body.String())
}
