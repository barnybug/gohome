package graphite

import (
	"bytes"
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

type Querier interface {
	Query(from, until, target string) ([]Dataseries, error)
}

type GraphiteQuerier struct {
	url    string
	host   string
	buffer string
}

func NewQuerier(url string) Querier {
	return &GraphiteQuerier{url: url}
}

type Datapoint struct {
	At    time.Time
	Value float64
}

func (dp *Datapoint) UnmarshalJSON(data []byte) error {
	var v []float64
	json.Unmarshal(data, &v)
	if len(v) != 2 {
		return errors.New("Datapoint incorrect length")
	}
	dp.Value = v[0]
	dp.At = time.Unix(int64(v[1]), 0)
	return nil
}

type Dataseries struct {
	Target     string
	Datapoints []Datapoint
}

func (graphite *GraphiteQuerier) Query(from, until, target string) ([]Dataseries, error) {
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

type Writer interface {
	Add(path string, timestamp int64, value float64) error
	Flush() error
}

type GraphiteWriter struct {
	host   string
	buffer bytes.Buffer
}

var dialer = func(network, address string) (io.ReadWriteCloser, error) {
	return net.Dial(network, address)
}

func NewWriter(host string) *GraphiteWriter {
	return &GraphiteWriter{host: host}
}

func (graphite *GraphiteWriter) Add(path string, timestamp int64, value float64) error {
	line := fmt.Sprintf("%s %v %d\n", path, value, timestamp)
	graphite.buffer.WriteString(line)
	if graphite.buffer.Len() > BatchSize {
		return graphite.Flush()
	}
	return nil
}

func (graphite *GraphiteWriter) Flush() error {
	conn, err := dialer("tcp", graphite.host+":2003")
	if err != nil {
		return err
	}
	io.Copy(conn, &graphite.buffer)
	conn.Close()
	graphite.buffer.Reset()
	return nil
}
