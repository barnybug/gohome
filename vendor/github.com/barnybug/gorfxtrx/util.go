package gorfxtrx

import (
	"io"
	"log"
)

// A logging ReadWriteCloser for debugging
type LogReadWriteCloser struct {
	f io.ReadWriteCloser
}

func (self LogReadWriteCloser) Read(b []byte) (int, error) {
	n, err := self.f.Read(b)
	log.Printf("Read(%#v) = (%d, %v)\n", b[:n], n, err)
	return n, err
}

func (self LogReadWriteCloser) Write(b []byte) (int, error) {
	n, err := self.f.Write(b)
	log.Printf("Write(%#v) = (%d, %v)\n", b, n, err)
	return n, err
}

func (self LogReadWriteCloser) Close() error {
	err := self.f.Close()
	log.Printf("Close() = %v\n", err)
	return err
}
