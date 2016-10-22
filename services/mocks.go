package services

import (
	"fmt"
	"strings"
)

// A very basic, only partial implementation of Store.
// Enough to past tests.
type MockStore struct {
	data map[string]string
}

func NewMockStore() *MockStore {
	ret := MockStore{
		data: map[string]string{},
	}
	return &ret
}

func (self *MockStore) Get(key string) (string, error) {
	if value, ok := self.data[key]; ok {
		return value, nil
	}
	return "", fmt.Errorf("Key missing: ", key)
}

func (self *MockStore) Set(key string, value string) error {
	self.data[key] = value
	return nil
}

func (self *MockStore) SetWithTTL(key string, value string, ttl uint64) error {
	return self.Set(key, value)
}

func (self *MockStore) GetRecursive(prefix string) ([]Node, error) {
	var ret []Node
	for key, value := range self.data {
		if strings.HasPrefix(key, prefix) {
			ret = append(ret, Node{Key: key, Value: value})
		}
	}

	return ret, nil
}
