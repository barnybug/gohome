package handlers

type Headers struct {
	Type      string `json:"type"`
	RequestId *int   `json:"requestId,omitempty"`
}
