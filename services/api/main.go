// Package api is a service providing an HTTP REST API to access gohome and control devices.
//
// The endpoints supported are:
//
// http://localhost:8723/query/{query} - query a service, e.g. http://localhost:8723/query/heating/status
//
// http://localhost:8723/voice - perform a voice query command
//
// http://localhost:8723/devices - list of devices
//
// http://localhost:8723/devices/events - list of device events
//
// http://localhost:8723/devices/control?id=device&control=0 - turn a device on or off
//
// http://localhost:8723/heating/status - get the status of heating
//
// http://localhost:8723/heating/set?temp=20&until=1h - set heating to 'temp' until 'until'
//
// http://localhost:8723/events/feed - continuous live stream of events (line delimited)
//
// http://localhost:8723/config?path=gohome/config - GET configuration or POST to update configuration
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
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

// ID of the service
func (service *Service) ID() string {
	return "api"
}

func errorResponse(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), 500)
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

func query(endpoint string, q string, w http.ResponseWriter) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	ch := services.QueryChannel(endpoint+" "+q, 100*time.Millisecond)

	for ev := range ch {
		fmt.Fprintf(w, ev.String()+"\r\n")
		w.(http.Flusher).Flush()
	}
}

func apiQuery(w http.ResponseWriter, r *http.Request) {
	endpoint := r.URL.Path[len("/query/"):]
	q := r.URL.Query().Get("q")
	query(endpoint, q, w)
}

func apiVoice(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

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
		fmt.Fprintf(w, "Not understood: '%s'", q)
		return
	}

	resp, err := services.RPC(body)
	if err == nil {
		fmt.Fprintf(w, resp)
	} else {
		w.WriteHeader(500)
		fmt.Fprintf(w, "error: %s", err)
	}
}

type deviceAndState struct {
	config.DeviceConf
	State interface{} `json:"state"`
}

func getDevicesState() map[string]interface{} {
	// Get state from store
	ret := make(map[string]interface{})
	nodes, _ := services.Stor.GetRecursive("gohome/state/devices")
	for _, node := range nodes {
		ev := pubsub.Parse(node.Value)
		name := node.Key[strings.LastIndex(node.Key, "/")+1:]
		ret[name] = ev.Map()
	}
	return ret
}

func apiDevices(w http.ResponseWriter, r *http.Request) {
	ret := make(map[string]deviceAndState)
	state := getDevicesState()

	for name, dev := range services.Config.Devices {
		ret[name] = deviceAndState{dev, state[name]}
	}

	jsonResponse(w, ret)
}

func apiDevicesEvents(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, getDevicesState())
}

func apiDevicesControl(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	device := q.Get("id")
	var command string
	if q.Get("control") == "1" {
		command = "on"
	} else {
		command = "off"
	}
	// send command
	ev := pubsub.NewCommand(device, command, 0)
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
	query("heating/ch", fmt.Sprintf("%s %s", q.Get("temp"), q.Get("until")), w)
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
		device := services.Config.LookupDeviceName(ev)
		if device != "" {
			data["device"] = device
		}
		encoder := json.NewEncoder(w)
		err := encoder.Encode(data)
		if err == nil {
			w.Write([]byte("\r\n")) // separator
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
		errorResponse(w, err)
		return
	}

	// retrieve key from store
	value, err := services.Stor.Get(q.Get("path"))
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
			services.Stor.Set(path, sout)
			// emit event
			fields := pubsub.Fields{
				"path": path,
			}
			ev := pubsub.NewEvent("config", fields)
			services.Publisher.Emit(ev)
			log.Printf("%s changed, emitted config event", path)
		}
	}
}

func apiLogs(w http.ResponseWriter, r *http.Request) {
	logs := []string{}
	infos, err := ioutil.ReadDir(config.LogPath(""))
	if err != nil {
		errorResponse(w, err)
		return
	}

	for _, info := range infos {
		logs = append(logs, info.Name())
	}
	jsonResponse(w, logs)
}

func apiLogsLog(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	filename := config.LogPath(params["file"])
	file, err := os.Open(filename)
	if err != nil {
		errorResponse(w, err)
		return
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, file)
}

func router() *mux.Router {
	router := mux.NewRouter()
	router.Path("/").HandlerFunc(apiIndex)
	router.PathPrefix("/query/").HandlerFunc(apiQuery)
	router.Path("/voice").HandlerFunc(apiVoice)
	router.Path("/devices").HandlerFunc(apiDevices)
	router.Path("/devices/events").HandlerFunc(apiDevicesEvents)
	router.Path("/devices/control").HandlerFunc(apiDevicesControl)
	router.Path("/heating/status").HandlerFunc(apiHeatingStatus)
	router.Path("/heating/set").HandlerFunc(apiHeatingSet)
	router.Path("/events/feed").HandlerFunc(apiEventsFeed)
	router.Path("/config").HandlerFunc(apiConfig)
	router.Path("/logs").HandlerFunc(apiLogs)
	router.Path("/logs/{file}").HandlerFunc(apiLogsLog)
	return router
}

type loggingHandler struct {
	Handler http.Handler
}

func (service loggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("%s %s\n", req.Method, req.RequestURI)
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
		device := services.Config.LookupDeviceName(ev)
		if device != "" {
			key := "gohome/state/devices/" + device
			services.Stor.Set(key, ev.String())
		}
	}
}

// Run the service
func (service *Service) Run() error {
	go recordEvents()
	httpEndpoint()
	return nil
}
