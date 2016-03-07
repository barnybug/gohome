package events

type AppStarted struct {
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
