package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/stampzilla/gocast/events"
)

type Receiver struct {
	Dispatch func(events.Event)
	Send     func(Headers) error

	knownApplications map[string]ApplicationSession
}

func (r *Receiver) RegisterDispatch(dispatch func(events.Event)) {
	r.Dispatch = dispatch
}

func (r *Receiver) RegisterSend(send func(Headers) error) {
	r.Send = send
}

func (r *Receiver) Connect() {
	// Request a new status update
	r.Send(Headers{Type: "GET_STATUS"})
}

func (r *Receiver) Disconnect() {
	r.knownApplications = make(map[string]ApplicationSession, 0)
}

func (r *Receiver) Unmarshal(message string) {
	//fmt.Println("Receiver received: ", message)

	response := &StatusResponse{}
	err := json.Unmarshal([]byte(message), response)

	if err != nil {
		fmt.Printf("Failed to unmarshal status message:%s - %s\n", err, message)
		return
	}

	prev := make(map[string]ApplicationSession, 0)
	if r.knownApplications == nil {
		r.knownApplications = make(map[string]ApplicationSession, 0)
	}

	// Make a copy of known applications
	for k, v := range r.knownApplications {
		prev[k] = v
	}

	for _, app := range response.Status.Applications {
		// App allready running
		if _, ok := prev[app.AppID]; ok {
			// Remove it from the list of previous known apps
			delete(prev, app.AppID)
			continue
		}

		// New app, add it to the list
		r.knownApplications[app.AppID] = *app

		r.Dispatch(events.AppStarted{
			AppID:       app.AppID,
			DisplayName: app.DisplayName,
		})
	}

	// Loop thru all stopped apps
	for key, app := range prev {
		delete(r.knownApplications, key)

		r.Dispatch(events.AppStopped{
			AppID:       app.AppID,
			DisplayName: app.DisplayName,
		})
	}
}

type StatusResponse struct {
	Headers
	Status *ReceiverStatus `json:"status,omitempty"`
}

type ReceiverStatus struct {
	Headers
	Applications []*ApplicationSession `json:"applications"`
	Volume       *Volume               `json:"volume,omitempty"`
}

type ApplicationSession struct {
	AppID       string      `json:"appId,omitempty"`
	DisplayName string      `json:"displayName,omitempty"`
	Namespaces  []Namespace `json:"namespaces"`
	SessionID   string      `json:"sessionId,omitempty"`
	StatusText  string      `json:"statusText,omitempty"`
	TransportId string      `json:"transportId,omitempty"`
}

type Namespace struct {
	Name string `json:"name"`
}

type Volume struct {
	Level *float64 `json:"level,omitempty"`
	Muted *bool    `json:"muted,omitempty"`
}
