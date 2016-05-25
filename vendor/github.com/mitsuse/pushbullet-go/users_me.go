package pushbullet

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mitsuse/pushbullet-go/responses"
)

// Get the current user.
func (pb *Pushbullet) GetUsersMe() (*responses.User, error) {
	req, err := http.NewRequest("GET", ENDPOINT_USERS_ME, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(pb.token, "")

	res, err := pb.client.Do(req)
	if err != nil {
		return nil, err
	}

	// TODO: Return an error value with human friendly message.
	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}

	var me *responses.User

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&me); err != nil {
		return nil, err
	}

	return me, nil
}
