package evdev

import (
	"bytes"
	"encoding/binary"
	"os"
)

type InputDevice struct {
	devname string
	fd      *os.File
}

func Open(devname string) (*InputDevice, error) {
	fd, err := os.Open(devname)
	if err != nil {
		return nil, err
	}
	dev := InputDevice{devname: devname, fd: fd}
	return &dev, nil
}

func (self *InputDevice) Grab() error {
	err := ioctl(self.fd.Fd(), EVIOCGRAB, 1)
	return err
}

func (self *InputDevice) ReadOne() (*InputEvent, error) {
	event := InputEvent{}
	buffer := make([]byte, eventsize)

	_, err := self.fd.Read(buffer)
	if err != nil {
		return &event, err
	}

	b := bytes.NewBuffer(buffer)
	err = binary.Read(b, binary.LittleEndian, &event)
	if err != nil {
		return &event, err
	}

	return &event, err
}

func (self *InputDevice) Close() error {
	return self.fd.Close()
}
