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

	"github.com/gorilla/mux"
)

// Service api
type Service struct {
}

var Debug bool = false

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
	// Get state from store
	ret := make(map[string]map[string]interface{})
	// gohome/state/events/<topic>/<device>
	nodes, _ := services.Stor.GetRecursive("gohome/state/events")
	for _, node := range nodes {
		ks := strings.Split(node.Key, "/")
		if len(ks) != 5 {
			continue
		}
		ev := pubsub.Parse(node.Value)
		topic := ks[3]
		device := ks[4]
		if _, ok := ret[device]; !ok {
			ret[device] = make(map[string]interface{})
		}
		ret[device][topic] = ev.Map()
	}
	// returns {device: {topic: {event}}}
	return ret
}

func deviceEntry(dev config.DeviceConf, events map[string]interface{}) interface{} {
	value := make(map[string]interface{})
	value["id"] = dev.Id
	value["name"] = dev.Name
	value["caps"] = dev.Caps
	value["type"] = dev.Type
	value["group"] = dev.Group
	if dev.Location != "" {
		value["location"] = dev.Location
	}
	if events == nil {
		events = make(map[string]interface{})
	}
	value["events"] = events
	return value
}

func apiDevices(w http.ResponseWriter, r *http.Request) {
	ret := make(map[string]interface{})
	events := getDevicesEvents()

	for name, dev := range services.Config.Devices {
		ret[name] = deviceEntry(dev, events[name])
	}

	jsonResponse(w, ret)
}

func apiDevicesSingle(w http.ResponseWriter, r *http.Request, params map[string]string) {
	device := params["device"]
	if dev, ok := services.Config.Devices[device]; ok {
		events := getDevicesEvents()
		ret := deviceEntry(dev, events[device])
		jsonResponse(w, ret)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "not found: %s", device)
	}
}

func apiDevicesControl(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	device := q.Get("id")
	command := q.Get("command")
	control := q.Get("control")
	if control != "" { // compatibility
		if control == "1" {
			command = "on"
		} else {
			command = "off"
		}
	}
	// send command
	ev := pubsub.NewCommand(device, command)
	if q.Get("repeat") != "" {
		if repeat, err := strconv.Atoi(q.Get("repeat")); err == nil {
			ev.SetRepeat(repeat)
		}
	}
	if q.Get("level") != "" {
		if level, err := strconv.Atoi(q.Get("level")); err == nil {
			ev.SetField("level", level)
		}
	}
	if q.Get("colour") != "" {
		colour := q.Get("colour")
		ev.SetField("colour", colour)
	}
	if q.Get("temp") != "" {
		if temp, err := strconv.Atoi(q.Get("temp")); err == nil {
			ev.SetField("temp", temp)
		}
	}
	if q.Get("duration") != "" {
		if duration, err := strconv.Atoi(q.Get("duration")); err == nil {
			ev.SetField("duration", duration)
		}
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
		ch = services.Subscriber.FilteredChannel(topics...)
	} else {
		ch = services.Subscriber.Channel()
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

	// retrieve key from store
	storeKey := "gohome/" + path
	value, err := services.Stor.Get(storeKey)
	if err != nil {
		errorResponse(w, err)
		return
	}

	if r.Method == "GET" {
		w.Header().Add("Content-Type", "application/yaml; charset=utf-8")
		w.Write([]byte(value))
	} else if r.Method == "POST" {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errorResponse(w, err)
			return
		}

		sout := string(data)
		if sout != value {
			// set store
			services.Stor.Set(storeKey, sout)
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

	ch := services.Subscriber.FilteredChannel("log")
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

func recordEvents() {
	for ev := range services.Subscriber.Channel() {
		// record to store
		if ev.Device() != "" {
			key := fmt.Sprintf("gohome/state/events/%s/%s", ev.Topic, ev.Device())
			services.Stor.Set(key, ev.String())
		}
	}
}

// Run the service
func (service *Service) Run() error {
	services.SetupStore()
	go recordEvents()
	httpEndpoint()
	return nil
}
