package responses

type Device struct {
	Iden         string  `json:"iden"`
	PushToken    string  `json:"push_token"`
	FinderPrint  string  `jsonL"fingerprint"`
	Nickname     string  `json:"nickname"`
	Manufacturer string  `json:"manufacturer"`
	Type         string  `json:"type"`
	Model        string  `json:"model"`
	AppVersion   int     `json:"app_version"`
	Created      float64 `json:"created"`
	Modified     float64 `json:"modified"`
	Active       bool    `json:"active"`
	Pushable     bool    `json:"pushable"`
}
