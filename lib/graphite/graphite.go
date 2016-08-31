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
	url    string
	buffer string
}

var dailer = func(network, address string) (io.ReadWriteCloser, error) {
	return net.Dial(network, address)
}

func New(url string) *Graphite {
	return &Graphite{url: url}
}

func (graphite *Graphite) Add(path string, timestamp int64, value float64) error {
	line := fmt.Sprintf("%s %v %d\n", path, value, timestamp)
	graphite.buffer += line
	if len(graphite.buffer) > BatchSize {
		return graphite.Flush()
	}
	return nil
}

func (graphite *Graphite) Flush() error {
	conn, err := dailer("tcp", graphite.url+":2003")
	if err != nil {
		return err
	}
	conn.Write([]byte(graphite.buffer))
	conn.Close()
	graphite.buffer = ""
	return nil
}

type Datapoint struct {
	At    time.Time
	Value float64
}

func (graphite *Datapoint) UnmarshalJSON(data []byte) error {
	var v []float64
	json.Unmarshal(data, &v)
	if len(v) != 2 {
		return errors.New("Datapoint incorrect length")
	}
	graphite.Value = v[0]
	graphite.At = time.Unix(int64(v[1]), 0)
	return nil
}

type Dataseries struct {
	Target     string
	Datapoints []Datapoint
}

func (graphite *Graphite) Query(from, until, target string) ([]Dataseries, error) {
	vs := url.Values{
		"from":   []string{from},
		"until":  []string{until},
		"target": []string{target},
		"format": []string{"json"}}
	uri := fmt.Sprintf("%s/render?%s", graphite.url, vs.Encode())
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
