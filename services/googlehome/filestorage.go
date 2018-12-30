package googlehome

import (
	"encoding/gob"
	"log"
	"os"

	"github.com/RangelReale/osin"
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register(&osin.DefaultClient{})
	gob.Register(osin.AuthorizeData{})
	gob.Register(osin.AccessData{})
}

type FileStorage struct {
	clients   map[string]osin.Client
	Authorize map[string]*osin.AuthorizeData
	Access    map[string]*osin.AccessData
	Refresh   map[string]string
}

func NewFileStorage() *FileStorage {
	r := &FileStorage{
		clients:   make(map[string]osin.Client),
		Authorize: make(map[string]*osin.AuthorizeData),
		Access:    make(map[string]*osin.AccessData),
		Refresh:   make(map[string]string),
	}

	return r
}

func (s *FileStorage) Clone() osin.Storage {
	return s
}

func (s *FileStorage) Close() {
}

func (s *FileStorage) GetClient(id string) (osin.Client, error) {
	log.Printf("GetClient: %s\n", id)
	if c, ok := s.clients[id]; ok {
		return c, nil
	}
	return nil, osin.ErrNotFound
}

func (s *FileStorage) SetClient(id string, client osin.Client) error {
	log.Printf("SetClient: %s\n", id)
	s.clients[id] = client
	return nil
}

func (s *FileStorage) SaveAuthorize(data *osin.AuthorizeData) error {
	log.Printf("SaveAuthorize: %s\n", data.Code)
	s.Authorize[data.Code] = data
	s.Persist()
	return nil
}

func (s *FileStorage) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	log.Printf("LoadAuthorize: %s\n", code)
	if d, ok := s.Authorize[code]; ok {
		return d, nil
	}
	return nil, osin.ErrNotFound
}

func (s *FileStorage) RemoveAuthorize(code string) error {
	log.Printf("RemoveAuthorize: %s\n", code)
	delete(s.Authorize, code)
	s.Persist()
	return nil
}

func (s *FileStorage) SaveAccess(data *osin.AccessData) error {
	log.Printf("SaveAccess: %s\n", data.AccessToken)
	s.Access[data.AccessToken] = data
	if data.RefreshToken != "" {
		s.Refresh[data.RefreshToken] = data.AccessToken
	}
	s.Persist()
	return nil
}

func (s *FileStorage) LoadAccess(code string) (*osin.AccessData, error) {
	log.Printf("LoadAccess: %s\n", code)
	if d, ok := s.Access[code]; ok {
		return d, nil
	}
	return nil, osin.ErrNotFound
}

func (s *FileStorage) RemoveAccess(code string) error {
	log.Printf("RemoveAccess: %s\n", code)
	delete(s.Access, code)
	s.Persist()
	return nil
}

func (s *FileStorage) LoadRefresh(code string) (*osin.AccessData, error) {
	log.Printf("LoadRefresh: %s\n", code)
	if d, ok := s.Refresh[code]; ok {
		return s.LoadAccess(d)
	}
	return nil, osin.ErrNotFound
}

func (s *FileStorage) RemoveRefresh(code string) error {
	log.Printf("RemoveRefresh: %s\n", code)
	delete(s.Refresh, code)
	s.Persist()
	return nil
}

func (s *FileStorage) Persist() {
	w, err := os.Create("oauth.data")
	if err != nil {
		log.Printf("Error persisting: %s", err)
		return
	}
	defer w.Close()
	if err := gob.NewEncoder(w).Encode(s); err != nil {
		log.Printf("Error persisting: %s", err)
		return
	}
	return
}

func (s *FileStorage) Restore() error {
	r, err := os.Open("oauth.data")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer r.Close()
	if err := gob.NewDecoder(r).Decode(s); err != nil {
		return err
	}
	return nil
}
