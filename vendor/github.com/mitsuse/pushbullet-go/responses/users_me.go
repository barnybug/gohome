package responses

type User struct {
	Iden            string       `json:"iden"`
	Email           string       `json:"email"`
	EmailNormalized string       `json:"email_normalized"`
	Name            string       `json:"name"`
	ImageUrl        string       `json:"image_url"`
	Created         float64      `json:"created"`
	Modified        float64      `json:"modified"`
	Preferences     *Preferences `json:"preferences"`
}

type Preferences struct {
	OnBoarding *OnBoarding `json:"onboarding"`
	Social     bool        `json:"social"`
}

type OnBoarding struct {
	App       bool `json:"app"`
	Friends   bool `json:"friends"`
	Extension bool `json:"extension"`
}
