package graphite

import (
	"bytes"
	"io"
	"testing"
)

type DummyConn struct {
	*bytes.Buffer
}

func (self *DummyConn) Close() error {
	return nil
}

func TestGraphite(t *testing.T) {
	conn := &DummyConn{bytes.NewBuffer([]byte{})}
	dailer = func(network, address string) (io.ReadWriteCloser, error) {
		return conn, nil
	}
	gr := New("localhost")
	gr.Add("test.stat", 1, 42.0)
	err := gr.Flush()
	if err != nil {
		t.Error("Expected: no error got:", err)
	}
	expected := "test.stat 42 1\n"
	if conn.Buffer.String() != expected {
		t.Error("Expected:", expected, "got:", conn.Buffer.String())
	}
}
