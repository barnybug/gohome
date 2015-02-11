package graphite

import "encoding/json"

type MockGraphite struct {
	Response string
}

func (self *MockGraphite) Add(path string, timestamp int64, value float64) error {
	return nil
}

func (self *MockGraphite) Flush() error {
	return nil
}

func (self *MockGraphite) Query(from, until, target string) ([]Dataseries, error) {
	var v []Dataseries
	err := json.Unmarshal([]byte(self.Response), &v)
	if err != nil {
		panic(err)
	}
	//Dataseries: graphite.Dataseries{Datapoints: dp}
	return v, nil
}
