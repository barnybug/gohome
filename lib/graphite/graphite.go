package graphite

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const BatchSize = 4096

type IGraphite interface {
	Add(path string, timestamp int64, value float64) error
	Flush() error
	Query(from, until, target string) ([]Dataseries, error)
}

type Graphite struct {
	host   string
	buffer string
}

var dailer = func(network, address string) (io.ReadWriteCloser, error) {
	return net.Dial(network, address)
}

func New(host string) *Graphite {
	return &Graphite{host: host}
}

func (self *Graphite) Add(path string, timestamp int64, value float64) error {
	line := fmt.Sprintf("%s %v %d\n", path, value, timestamp)
	self.buffer += line
	if len(self.buffer) > BatchSize {
		return self.Flush()
	}
	return nil
}

func (self *Graphite) Flush() error {
	conn, err := dailer("tcp", self.host+":2003")
	if err != nil {
		return err
	}
	conn.Write([]byte(self.buffer))
	conn.Close()
	self.buffer = ""
	return nil
}

type Datapoint struct {
	At    time.Time
	Value float64
}

func (self *Datapoint) UnmarshalJSON(data []byte) error {
	var v []float64
	json.Unmarshal(data, &v)
	if len(v) != 2 {
		return errors.New("Datapoint incorrect length")
	}
	self.Value = v[0]
	self.At = time.Unix(int64(v[1]), 0)
	return nil
}

type Dataseries struct {
	Target     string
	Datapoints []Datapoint
}

func (self *Graphite) Query(from, until, target string) ([]Dataseries, error) {
	vs := url.Values{
		"from":   []string{from},
		"until":  []string{until},
		"target": []string{target},
		"format": []string{"json"}}
	uri := fmt.Sprintf("http://%s/graphite/render?%s", self.host, vs.Encode())
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	var v []Dataseries
	if err := d.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
