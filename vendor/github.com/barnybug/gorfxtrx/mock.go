package gorfxtrx

type MockSerialPort struct {
	replay [][]byte
}

func NewMockSerialPort(replay [][]byte) *MockSerialPort {
	self := &MockSerialPort{
		replay: replay,
	}
	return self
}

func (self *MockSerialPort) Read(b []byte) (int, error) {
	data := self.replay[0]
	self.replay = self.replay[1:]
	copy(b, data)
	return len(data), nil
}

func (self *MockSerialPort) Write(b []byte) (int, error) {
	// noop
	return len(b), nil
}

func (self *MockSerialPort) Close() error {
	return nil
}
