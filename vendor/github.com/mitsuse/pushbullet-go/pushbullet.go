/*
Package "pushbullet" provides interfaces for Pushbullet HTTP API.

Pushbullet is a web service,
which makes your devices work better together by allowing you to move things between them easily.

The official url: https://www.pushbullet.com/

Currently, this package supports only "pushes" except file.

See the API documentation for the details: https://docs.pushbullet.com/#http
*/
package pushbullet

import (
	"net/http"
)

type Pushbullet struct {
	client *http.Client
	token  string
}

/*
Create a client to call Pushbullet HTTP API.
This requires the access token.
The token is found in account settings.

Account settings: https://www.pushbullet.com/account
*/
func New(token string) *Pushbullet {
	return NewClient(token, http.DefaultClient)
}

/*
Create a client to call Pushbullet HTTP API.
This requires the access token and an aribitary *http.Client.
The token is found in account settings.

Account settings: https://www.pushbullet.com/account
*/
func NewClient(token string, c *http.Client) *Pushbullet {
	pb := &Pushbullet{
		token:  token,
		client: c,
	}

	return pb
}

func (pb *Pushbullet) Client() *http.Client {
	return pb.client
}
