package googlehome

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/RangelReale/osin"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

var DeviceEvents = map[string]map[string]*pubsub.Event{}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello there!")
}

type ActionRequestInput struct {
	Payload interface{}
}

type RawActionRequestInput struct {
	Intent  string
	Payload json.RawMessage `json:"payload"`
}

func (a *ActionRequestInput) UnmarshalJSON(b []byte) error {
	var raw RawActionRequestInput
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	switch raw.Intent {
	case "action.devices.SYNC":
		a.Payload = SyncRequestPayload{}
	case "action.devices.QUERY":
		var payload QueryRequestPayload
		err := json.Unmarshal(raw.Payload, &payload)
		if err != nil {
			return err
		}
		a.Payload = payload
	case "action.devices.EXECUTE":
		var payload ExecuteRequestPayload
		err := json.Unmarshal(raw.Payload, &payload)
		if err != nil {
			return err
		}
		a.Payload = payload
	default:
		return fmt.Errorf("intent not recognised: %s", raw.Intent)
	}

	return nil
}

type ActionRequest struct {
	Inputs    []ActionRequestInput
	RequestId string `json:"requestId"`
}

type SyncRequestPayload struct {
}

type QueryRequestPayload struct {
	Devices []struct {
		Id string
	}
}

type ColorState struct {
	Temperature int `json:"temperature,omitempty"`
	SpectrumRGB int `json:"spectrumRGB,omitempty"`
}

type DeviceState struct {
	Online                        bool        `json:"online"`
	On                            *bool       `json:"on,omitempty"`
	Brightness                    int         `json:"brightness,omitempty"`
	Color                         *ColorState `json:"color,omitempty"`
	Deactivate                    *bool       `json:"deactivate,omitempty"`
	ThermostatMode                *string     `json:"thermostatMode,omitempty"`
	ThermostatTemperatureSetpoint *float64    `json:"thermostatTemperatureSetpoint,omitempty"`
	ThermostatTemperatureAmbient  *float64    `json:"thermostatTemperatureAmbient,omitempty"`
}

type QueryResponsePayload struct {
	Devices map[string]DeviceState `json:"devices"`
}

type Execution struct {
	Command string
	Params  DeviceState
}

type ExecuteRequestPayload struct {
	Commands []struct {
		Devices []struct {
			Id string
		}
		Execution []Execution
	}
}

type ExecuteResult struct {
	Ids       []string    `json:"ids"`
	Status    string      `json:"status"`
	States    DeviceState `json:"states"`
	Errorcode string      `json:"errorCode,omitempty"`
}

type ExecuteResponsePayload struct {
	Commands []ExecuteResult `json:"commands"`
}

type ActionResponse struct {
	RequestId string      `json:"requestId"`
	Payload   interface{} `json:"payload"`
}

type DeviceName struct {
	Defaultnames []string `json:"defaultNames,omitempty"`
	Name         string   `json:"name"`
	Nicknames    []string `json:"nicknames,omitempty"`
}

type Device struct {
	Id              string                 `json:"id"`
	Type            string                 `json:"type"`
	Traits          []string               `json:"traits"`
	Name            DeviceName             `json:"name"`
	WillReportState bool                   `json:"willReportState"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	RoomHint        string                 `json:"roomHint,omitempty"`
}

type SyncResponsePayload struct {
	AgentUserId string   `json:"agentUserId"`
	Devices     []Device `json:"devices"`
}

func contains(array []string, s string) bool {
	for _, element := range array {
		if s == element {
			return true
		}
	}
	return false
}

var types = map[string]string{
	"light":  "action.devices.types.LIGHT",
	"heater": "action.devices.types.OUTLET",
	"pump":   "action.devices.types.OUTLET",
	"switch": "action.devices.types.OUTLET",
}

func syncRequest() (*SyncResponsePayload, error) {
	log.Println("Received sync request")
	out := []Device{}
	for _, device := range services.Config.Devices {
		if device.Group == "" {
			continue
		}
		if device.Location == "" {
			continue
		}

		typ := "action.devices.types.SWITCH"
		traits := []string{}
		attributes := map[string]interface{}{}

		if contains(device.Caps, "presence") {
			continue
		}
		if contains(device.Caps, "switch") {
			traits = append(traits, "action.devices.traits.OnOff")
		}
		if contains(device.Caps, "dimmer") {
			typ = "action.devices.types.LIGHT"
			traits = append(traits, "action.devices.traits.Brightness")
		}
		if contains(device.Caps, "colourtemp") || contains(device.Caps, "colour") {
			typ = "action.devices.types.LIGHT"
			traits = append(traits, "action.devices.traits.ColorSetting")
			if contains(device.Caps, "colour") {
				attributes["colorModel"] = "rgb"
			}
			if contains(device.Caps, "colourtemp") {
				attributes["colorTemperatureRange"] = map[string]interface{}{
					"temperatureMinK": 2200,
					"temperatureMaxK": 6700,
				}
			}
		}
		if contains(device.Caps, "scene") {
			typ = "action.devices.types.SCENE"
			traits = append(traits, "action.devices.traits.Scene")
			attributes["sceneReversible"] = contains(device.Caps, "reversible")
		}
		if contains(device.Caps, "thermostat") {
			typ = "action.devices.types.THERMOSTAT"
			traits = append(traits, "action.devices.traits.TemperatureSetting")
			attributes["availableThermostatModes"] = "heat"
			attributes["thermostatTemperatureUnit"] = "C"
		}
		if contains(device.Caps, "washer") {
			typ = "action.devices.types.WASHER"
		}
		if len(traits) == 0 {
			continue
		}

		ps := strings.SplitN(device.Id, ".", 2)
		if _, ok := types[ps[0]]; ok {
			typ = types[ps[0]]
		}
		nicknames := append([]string{device.Name}, device.Aliases...)
		o := Device{
			Id:              device.Id,
			Type:            typ,
			Traits:          traits,
			Attributes:      attributes,
			Name:            DeviceName{Name: device.Name, Nicknames: nicknames},
			RoomHint:        device.Location,
			WillReportState: false,
		}
		out = append(out, o)
	}
	payload := SyncResponsePayload{
		AgentUserId: "gohome",
		Devices:     out,
	}
	return &payload, nil
}

const ApplicationJson = "application/json"

func errorResponse(w http.ResponseWriter, code int, err error, message string) {
	r := map[string]interface{}{
		"success": false,
		"message": message,
	}
	if err != nil {
		r["error"] = fmt.Sprint(err)
	}
	w.Header().Add("Content-Type", ApplicationJson)
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(&r); err != nil {
		log.Fatal(err)
	}
}

func checkAuthorization(r *http.Request) error {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return errors.New("Authorization header invalid")
	}
	token := auth[7:]
	// TODO check token
	if token == "" {
		return errors.New("Authorization token invalid")
	}
	return nil
}

func pbool(b bool) *bool {
	return &b
}

func pstring(s string) *string {
	return &s
}

func queryRequest(request QueryRequestPayload) (*QueryResponsePayload, error) {
	log.Println("Received query request")
	result := QueryResponsePayload{
		Devices: map[string]DeviceState{},
	}
	for _, q := range request.Devices {
		id := q.Id
		if device, ok := services.Config.Devices[id]; ok {
			events := DeviceEvents[id]
			m := DeviceState{
				Online: true,
			}
			if contains(device.Caps, "switch") {
				if state, ok := events["state"]; ok {
					switch state.StringField("state") {
					case "On":
						m.On = pbool(true)
					case "Off":
						m.On = pbool(false)
					}
				}
			}
			if ack, ok := events["ack"]; ok {
				if contains(device.Caps, "dimmer") && ack.IsSet("level") {
					level := ack.FloatField("level")
					m.Brightness = int(level)
				}
				if contains(device.Caps, "temp") && ack.IsSet("temp") {
					temp := ack.FloatField("temp")
					m.Color = &ColorState{
						Temperature: int(temp),
					}
				}
			}
			if contains(device.Caps, "thermostat") {
				target := 0.0
				ambient := 0.0
				if thermostat, ok := events["thermostat"]; ok {
					target = thermostat.FloatField("target")
					// find equivalent trv device
					trvname := strings.Replace(device.Id, "thermostat.", "trv.", 1)
					if trv, ok := DeviceEvents[trvname]; ok {
						if event, ok := trv["temp"]; ok {
							ambient = event.FloatField("temp")
						}
					}
				}
				mode := "heat"
				m.ThermostatMode = &mode
				m.ThermostatTemperatureSetpoint = &target
				m.ThermostatTemperatureAmbient = &ambient
			}
			result.Devices[device.Id] = m
		}
	}

	return &result, nil
}

func executeCommands(device string, executions []Execution) ExecuteResult {
	result := ExecuteResult{
		Ids: []string{device},
	}
	command := pubsub.NewEvent("command", pubsub.Fields{"device": device})
	states := DeviceState{Online: true}
	for _, execution := range executions {
		switch execution.Command {
		case "action.devices.commands.OnOff":
			if execution.Params.On != nil {
				if *execution.Params.On == true {
					command.SetField("command", "on")
				} else {
					command.SetField("command", "off")
				}
				states.On = execution.Params.On
			}
		case "action.devices.commands.BrightnessAbsolute":
			if execution.Params.Brightness != 0 {
				brightness := execution.Params.Brightness
				command.SetField("command", "on")
				command.SetField("level", brightness)
				states.Brightness = brightness
			}
		case "action.devices.commands.ColorAbsolute":
			if execution.Params.Color != nil {
				color := execution.Params.Color
				command.SetField("command", "on")
				if color.Temperature != 0 {
					command.SetField("temp", color.Temperature)
				} else if color.SpectrumRGB != 0 {
					colour := fmt.Sprintf("#%06x", color.SpectrumRGB)
					command.SetField("colour", colour)
				}
				states.Color = execution.Params.Color
			}
		case "action.devices.commands.ActivateScene":
			if execution.Params.Deactivate != nil {
				if *execution.Params.Deactivate == true {
					command.SetField("command", "off")
				} else {
					command.SetField("command", "on")
				}
				// scenes are stateless
			}
		case "action.devices.commands.ThermostatTemperatureSetpoint":
			if execution.Params.ThermostatTemperatureSetpoint != nil {
				setpoint := execution.Params.ThermostatTemperatureSetpoint
				command.SetField("temp", *setpoint)
				states.ThermostatTemperatureSetpoint = setpoint
			}
		}
	}

	services.Publisher.Emit(command)
	result.Status = "SUCCESS"
	result.States = states
	return result
}

func executeRequest(request ExecuteRequestPayload) (*ExecuteResponsePayload, error) {
	log.Println("Received execute request")
	result := ExecuteResponsePayload{}
	for _, command := range request.Commands {
		for _, device := range command.Devices {
			executeResult := executeCommands(device.Id, command.Execution)
			result.Commands = append(result.Commands, executeResult)
		}
	}
	return &result, nil
}

func processInput(w http.ResponseWriter, requestId string, input ActionRequestInput) {
	var result interface{}
	var err error
	switch payload := input.Payload.(type) {
	case SyncRequestPayload:
		result, err = syncRequest()
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err, "Couldn't create payload")
			return
		}
	case QueryRequestPayload:
		result, err = queryRequest(payload)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err, "Couldn't create payload")
			return
		}

	case ExecuteRequestPayload:
		result, err = executeRequest(payload)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err, "Couldn't create payload")
			return
		}

	default:
		errorResponse(w, http.StatusBadRequest, nil, "Request not understood")
		return
	}
	response := ActionResponse{
		RequestId: requestId,
		Payload:   result,
	}

	w.Header().Add("Content-Type", ApplicationJson)
	b, err := json.Marshal(response)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("<- %s", b)
	w.Write(b)
	w.Write([]byte("\n"))
}

func actionsEndpoint(w http.ResponseWriter, r *http.Request) {
	err := checkAuthorization(r)
	if err != nil {
		errorResponse(w, http.StatusUnauthorized, err, "Authorization failed")
		return
	}
	var request ActionRequest
	if r.Body == nil {
		errorResponse(w, http.StatusBadRequest, err, "Failed to decode request")
		return
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorResponse(w, http.StatusUnauthorized, err, "Failed to read body")
		return
	}
	log.Printf("-> %s", body)
	if err := json.Unmarshal(body, &request); err != nil {
		errorResponse(w, http.StatusBadRequest, err, "Failed to decode request")
		return
	}

	if len(request.Inputs) == 0 {
		errorResponse(w, http.StatusBadRequest, err, "Failed to decode request")
		return
	}

	for _, input := range request.Inputs {
		processInput(w, request.RequestId, input)
	}
}

type loggingHandler struct {
	handler http.Handler
}

func (h loggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("%s %s %s (%s)", req.Method, req.URL, req.Proto, req.UserAgent())
	h.handler.ServeHTTP(w, req)
}

// Service googlehome
type Service struct{}

// ID of the service
func (self *Service) ID() string {
	return "googlehome"
}

func recordEvents() {
	for ev := range services.Subscriber.Channel() {
		if ev.Device() == "" {
			continue
		}
		if _, ok := DeviceEvents[ev.Device()]; !ok {
			DeviceEvents[ev.Device()] = make(map[string]*pubsub.Event)
		}
		DeviceEvents[ev.Device()][ev.Topic] = ev
	}
}

// Run the service
func (self *Service) Run() error {
	go recordEvents()
	// FileStorage implements the "osin.Storage" interface, persists to gob encoded file
	storage := NewFileStorage()
	c := services.Config.Googlehome
	storage.SetClient(c.Id, &osin.DefaultClient{
		Id:          c.Id,
		Secret:      c.Secret,
		RedirectUri: c.Redirect_uri,
	})
	err := storage.Restore()
	if err != nil {
		log.Fatalf("Restoring storage: %+v", err)
	}
	log.Printf("Restored tokens: %d authorize, %d access, %d refresh", len(storage.Authorize), len(storage.Access), len(storage.Refresh))
	storage.Persist()
	config := osin.NewServerConfig()
	config.AllowClientSecretInParams = true
	config.AllowedAccessTypes = osin.AllowedAccessType{
		osin.AUTHORIZATION_CODE, osin.REFRESH_TOKEN,
	}
	config.AccessExpiration = 7 * 86400 // 7 days
	server := osin.NewServer(config, storage)

	// Authorization code endpoint
	http.HandleFunc("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		resp := server.NewResponse()
		defer resp.Close()

		if ar := server.HandleAuthorizeRequest(resp, r); ar != nil {
			ar.Authorized = true
			server.FinishAuthorizeRequest(resp, r, ar)
		}
		if resp.IsError {
			log.Println("Authorize error:", resp.InternalError)
		}
		osin.OutputJSON(resp, w, r)
	})

	// Access token endpoint
	http.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		resp := server.NewResponse()
		defer resp.Close()

		if ar := server.HandleAccessRequest(resp, r); ar != nil {
			ar.Authorized = true
			server.FinishAccessRequest(resp, r, ar)
		}
		if resp.IsError {
			log.Println("Token error:", resp.InternalError)
		}
		osin.OutputJSON(resp, w, r)
	})
	http.HandleFunc("/actions", actionsEndpoint)
	http.ListenAndServe(":8085", loggingHandler{http.DefaultServeMux})
	return nil
}
