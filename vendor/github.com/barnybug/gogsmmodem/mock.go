package gogsmmodem

import (
	"fmt"
	"strings"
)

type MockSerialPort struct {
	replay  []string
	receive chan string
}

func NewMockSerialPort(replay []string) *MockSerialPort {
	self := &MockSerialPort{
		replay:  replay,
		receive: make(chan string, 16),
	}
	self.enqueueReads()
	return self
}

func (self *MockSerialPort) Read(b []byte) (int, error) {
	line := <-self.receive
	data := []byte(line)
	copy(b, data)
	return len(data), nil
}

func (self *MockSerialPort) enqueueReads() {
	// enqueue response(s) from replay
	for {
		if len(self.replay) == 0 || strings.Index(self.replay[0], "<-") != 0 {
			break
		}
		i := self.replay[0][2:]
		self.receive <- i
		self.replay = self.replay[1:]
	}
}

func (self *MockSerialPort) Write(b []byte) (int, error) {
	if len(self.replay) == 0 {
		fmt.Printf("Expected: no more interactions, got: %#v", string(b))
		panic("fail")
	}
	i := self.replay[0]
	if strings.Index(i, "->") != 0 {
		fmt.Printf("Replay isn't data to send:", i)
		panic("fail")
	}
	expected := i[2:]
	if expected != string(b) {
		fmt.Printf("Expected: %#v got: %#v", expected, string(b))
		panic("fail")
	}
	self.replay = self.replay[1:]
	self.enqueueReads()
	return len(b), nil
}

func (self *MockSerialPort) Close() error {
	return nil
}
