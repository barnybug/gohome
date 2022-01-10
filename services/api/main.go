// Package api is a service providing an HTTP REST API to access gohome and control devices.
//
// The endpoints supported are:
//
// http://localhost:8723/config?path=config - GET configuration or POST to update configuration
//
// http://localhost:8723/devices - list of devices and events
//
// http://localhost:8723/devices/control?id=device&control=0 - turn a device on or off
//
// http://localhost:8723/devices/<devicename> - single device with events
//
// http://localhost:8723/heating/status - get the status of heating
//
// http://localhost:8723/heating/set?temp=20&until=1h - set heating to 'temp' until 'until'
//
// http://localhost:8723/events/feed - continuous live stream of events (line delimited)
//
// http://localhost:8723/query/{query} - query a service, e.g. http://localhost:8723/query/heating/status
//
// http://localhost:8723/logs - stream logs, until disconnect
//
// http://localhost:8723/voice - perform a voice query command
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"github.com/barnybug/gohome/util"

	"github.com/gorilla/mux"
)

// Service api
type Service struct {
}

var Debug bool = false
var DeviceState = map[string]map[string]*pubsub.Event{}

// ID of the service
func (service *Service) ID() string {
	return "api"
}

func errorResponse(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func badRequest(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}

type VarsHandler func(http.ResponseWriter, *http.Request, map[string]string)

func (h VarsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	h(w, req, vars)
}

func apiIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	fmt.Fprintf(w, "<html>Gohome is listening</html>")

}

func jsonResponse(w http.ResponseWriter, obj interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	err := enc.Encode(obj)
	if err != nil {
		errorResponse(w, err)
	}
}

const DefaultQueryTimeout = 500

func query(endpoint string, q string, timeout int64, responses int64, w http.ResponseWriter) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	ch := services.QueryChannel(endpoint+" "+q, time.Duration(timeout)*time.Millisecond)

	var n int64 = 0
	for ev := range ch {
		fmt.Fprintf(w, "%s\r\n", ev.String())
		w.(http.Flusher).Flush()
		n++
		if n >= responses {
			break
		}
	}
}

func apiQuery(w http.ResponseWriter, r *http.Request) {
	endpoint := r.URL.Path[len("/query/"):]
	qvals := r.URL.Query()
	q := qvals.Get("q")
	timeout, err := strconv.ParseInt(qvals.Get("timeout"), 10, 32)
	if err != nil {
		timeout = DefaultQueryTimeout
	}
	responses, err := strconv.ParseInt(qvals.Get("responses"), 10, 32)
	if err != nil {
		responses = math.MaxInt64
	}
	query(endpoint, q, timeout, responses, w)
}

func apiVoice(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	log.Printf("Received voice request: '%s'", q)

	body := ""
	for key, value := range services.Config.Voice {
		re, err := regexp.Compile(key)
		if err != nil {
			continue
		}
		var match = re.FindStringSubmatchIndex(q)
		if match != nil {
			// Expand $1 matches in the command
			var dst []byte
			result := re.ExpandString(dst, value, q, match)
			body = string(result)
		}
	}
	if body == "" {
		log.Printf("Not understood: '%s'", q)
		fmt.Fprintf(w, "Not understood: '%s'", q)
		return
	}

	resp, err := services.RPC(body, time.Second*5)
	if err == nil {
		log.Printf("Voice response: '%s'", resp)
		fmt.Fprintf(w, resp)
	} else {
		w.WriteHeader(500)
		fmt.Fprintf(w, "error: %s", err)
	}
}

func getDevicesEvents() map[string]map[string]interface{} {
	ret := make(map[string]map[string]interface{})
	for device, conf := range services.Config.Devices {
		ret[device] = deviceEntry(conf, DeviceState[device])
	}
	// returns {device: {topic: {event}}}
	return ret
}

func deviceEntry(dev config.DeviceConf, events map[string]*pubsub.Event) map[string]interface{} {
	value := make(map[string]interface{})
	value["id"] = dev.Id
	value["name"] = dev.Name
	value["caps"] = dev.Caps
	value["group"] = dev.Group
	value["aliases"] = dev.Aliases
	if dev.Location != "" {
		value["location"] = dev.Location
	}
	ev := map[string]interface{}{}
	for topic, event := range events {
		ev[topic] = event.Map()
	}
	value["events"] = ev
	return value
}

func apiDevices(w http.ResponseWriter, r *http.Request) {
	ret := getDevicesEvents()
	jsonResponse(w, ret)
}

func apiDevicesSingle(w http.ResponseWriter, r *http.Request, params map[string]string) {
	device := params["device"]
	if dev, ok := services.Config.Devices[device]; ok {
		ret := deviceEntry(dev, DeviceState[device])
		jsonResponse(w, ret)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "not found: %s", device)
	}
}

func apiDevicesControl(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	device := q.Get("id")
	matches := services.MatchDevices(device)
	if len(matches) == 0 {
		badRequest(w, errors.New("device not found"))
		return
	}
	if len(matches) > 1 {
		badRequest(w, errors.New("device is ambiguous"))
		return
	}
	device = matches[0]

	// send command
	fields := pubsub.Fields{
		"topic":  "command",
		"device": device,
	}
	ev := pubsub.NewEvent("command", fields)
	for key, values := range q {
		ev.SetField(key, util.ParseArg(values[0]))
	}
	services.Publisher.Emit(ev)
	jsonResponse(w, true)
}

func apiHeatingStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	ch := services.QueryChannel("heating/status", 100*time.Millisecond)
	ev := <-ch
	ret := ev.Fields["json"]
	jsonResponse(w, ret)
}

func apiHeatingSet(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	for _, name := range []string{"id", "temp", "until"} {
		if _, ok := q[name]; !ok {
			err := fmt.Errorf("%s parameter required", name)
			badRequest(w, err)
			return
		}
	}
	arg := fmt.Sprintf("%s %s %s", q.Get("id"), q.Get("temp"), q.Get("until"))
	query("heating/ch", arg, DefaultQueryTimeout, 1, w)
}

func apiEventsFeed(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	topics := q.Get("topics")
	w.Header().Add("Content-Type", "application/json; boundary=NL")

	var ch <-chan *pubsub.Event
	if topics != "" {
		topics := strings.Split(topics, ",")
		var subs []pubsub.Topic
		for _, t := range topics {
			subs = append(subs, pubsub.Prefix(t))
		}
		ch = services.Subscriber.Subscribe(subs...)
	} else {
		ch = services.Subscriber.Subscribe(pubsub.All())
	}
	defer services.Subscriber.Close(ch)

	for ev := range ch {
		data := ev.Map()
		encoder := json.NewEncoder(w)
		err := encoder.Encode(data)
		if err == nil {
			_, err = w.Write([]byte("\r\n")) // separator
		}
		if err != nil {
			break
		}
		w.(http.Flusher).Flush()
	}
}

func convertJSON(v interface{}) interface{} {
	// convert json unfriendly types to json friendly.
	switch t := v.(type) {
	case map[interface{}]interface{}:
		// convert all keys to strings
		ret := map[string]interface{}{}
		for k, v := range t {
			ret[fmt.Sprint(k)] = convertJSON(v)
		}
		return ret
	case []interface{}:
		// convert all elements of array
		ret := []interface{}{}
		for _, v := range t {
			ret = append(ret, convertJSON(v))
		}
		return ret
	default:
		return v
	}
}

func apiConfig(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	path := q.Get("path")
	if path == "" {
		err := errors.New("path parameter required")
		badRequest(w, err)
		return
	}
	if !strings.HasPrefix(path, "config") {
		err := errors.New("path should begin with config")
		badRequest(w, err)
		return
	}

	// get existing value
	value := configurations[path]

	if r.Method == "GET" {
		w.Header().Add("Content-Type", "application/yaml; charset=utf-8")
		w.Write(value)
	} else if r.Method == "POST" {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errorResponse(w, err)
			return
		}

		sout := data
		if string(sout) != string(value) {
			// emit event
			fields := pubsub.Fields{
				"config": sout,
			}
			ev := pubsub.NewEvent(path, fields)
			ev.SetRetained(true) // config messages are retained
			services.Publisher.Emit(ev)
			log.Printf("%s changed, emitted config event", path)
		}
	}
}

func apiLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json; boundary=NL")

	ch := services.Subscriber.Subscribe(pubsub.Prefix("log"))
	defer services.Subscriber.Close(ch)
	for ev := range ch {
		_, err := fmt.Fprintf(w, "%s\r\n", ev)
		if err != nil {
			break
		}
		w.(http.Flusher).Flush()
	}
}

func router() *mux.Router {
	router := mux.NewRouter()
	router.Path("/").HandlerFunc(apiIndex)
	router.PathPrefix("/query/").HandlerFunc(apiQuery)
	router.Path("/voice").HandlerFunc(apiVoice)
	router.Path("/devices").HandlerFunc(apiDevices)
	router.Path("/devices/control").HandlerFunc(apiDevicesControl)
	router.Handle("/devices/{device}", VarsHandler(apiDevicesSingle))
	router.Path("/heating/status").HandlerFunc(apiHeatingStatus)
	router.Path("/heating/set").HandlerFunc(apiHeatingSet)
	router.Path("/events/feed").HandlerFunc(apiEventsFeed)
	router.Path("/config").HandlerFunc(apiConfig)
	router.Path("/logs").HandlerFunc(apiLogs)
	return router
}

type loggingHandler struct {
	Handler http.Handler
}

func (service loggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if Debug {
		log.Printf("%s %s\n", req.Method, req.RequestURI)
	}
	service.Handler.ServeHTTP(w, req)
}

func httpEndpoint() {
	// disabled logger as this prevents ResponseWriter.Flush being accessed
	// handler := handlers.LoggingHandler(os.Stdout, router())
	var handler http.Handler = router()
	handler = loggingHandler{Handler: handler}
	// Allow CORS+http auth (so the api can be placed behind http auth)
	corsHandler := CORSHandler{Handler: handler}
	corsHandler.SupportsCredentials = true
	corsHandler.AllowHeaders = func(headers []string) bool {
		for _, header := range headers {
			if header != "accept" && header != "authorization" {
				return false
			}
		}
		return true
	}
	http.Handle("/", corsHandler)
	addr := ":8723"
	log.Println("Listening on " + addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalln(err)
	}
}

var configurations map[string][]byte = map[string][]byte{}

func recordEvents() {
	config := pubsub.Prefix("config")
	for ev := range services.Subscriber.Subscribe(pubsub.All()) {
		// record to store
		if config.Match(ev.Topic) {
			configurations[ev.Topic] = ev.Raw
		} else if ev.Device() != "" {
			if _, ok := DeviceState[ev.Device()]; !ok {
				DeviceState[ev.Device()] = make(map[string]*pubsub.Event)
			}
			DeviceState[ev.Device()][ev.Topic] = ev
		}
	}
}

// Run the service
func (service *Service) Run() error {
	go recordEvents()
	httpEndpoint()
	return nil
}
