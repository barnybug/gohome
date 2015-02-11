package evdev

import (
	"fmt"
	"io"
	"testing"
)

func TestOpen(t *testing.T) {
	// hard to test this without grabbing a real input device,
	// so just basic test of functions and error handling.
	dev, err := Open("/dev/null")
	if err != nil {
		t.Error("Open: expected no error got", err)
	}
	err = dev.Grab()
	if fmt.Sprint(err) != "inappropriate ioctl for device" {
		t.Error("inappropriate ioctl for device")
	}
	_, err = dev.ReadOne()
	if err != io.EOF {
		t.Error("ReadOne: expected EOF got", err)
	}
	err = dev.Close()
	if err != nil {
		t.Error("Close: expected no error got", err)
	}
}
