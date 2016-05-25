package pushbullet

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mitsuse/pushbullet-go/requests"
	"github.com/mitsuse/pushbullet-go/responses"
)

// Push a note, which consists of "title" and "message" strings.
func (pb *Pushbullet) PostPushesNote(n *requests.Note) (*responses.Note, error) {
	res, err := pb.postPushes(n)
	if err != nil {
		return nil, err
	}

	var noteRes *responses.Note
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&noteRes); err != nil {
		return nil, err
	}

	return noteRes, err
}

// Push a link, which consists of "title", "message" and "url" strings.
func (pb *Pushbullet) PostPushesLink(l *requests.Link) (*responses.Link, error) {
	res, err := pb.postPushes(l)
	if err != nil {
		return nil, err
	}

	var linkRes *responses.Link
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&linkRes); err != nil {
		return nil, err
	}

	return linkRes, err
}

// Push an address, which consists of the place "name" and "address (searchquery)" for map.
func (pb *Pushbullet) PostPushesAddress(a *requests.Address) (*responses.Address, error) {
	res, err := pb.postPushes(a)
	if err != nil {
		return nil, err
	}

	var addressRes *responses.Address
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&addressRes); err != nil {
		return nil, err
	}

	return addressRes, err
}

// Push a checklist, which consists of "title" and the list of items.
func (pb *Pushbullet) PostPushesChecklist(c *requests.Checklist) (*responses.Checklist, error) {
	res, err := pb.postPushes(c)
	if err != nil {
		return nil, err
	}

	var checkRes *responses.Checklist
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&checkRes); err != nil {
		return nil, err
	}

	return checkRes, err
}

// Push a file, which consists of "title", "message" and the information of uploaded file.
func (pb *Pushbullet) PostPushesFile(f *requests.File) (*responses.File, error) {
	res, err := pb.postPushes(f)
	if err != nil {
		return nil, err
	}

	var fileRes *responses.File
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&fileRes); err != nil {
		return nil, err
	}

	return fileRes, err
}

func (pb *Pushbullet) postPushes(p interface{}) (*http.Response, error) {
	buffer := &bytes.Buffer{}

	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(p); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", ENDPOINT_PUSHES, buffer)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(pb.token, "")

	res, err := pb.client.Do(req)
	if err != nil {
		return nil, err
	}

	// TODO: Return an error value with human friendly message.
	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}

	return res, nil
}
