package apcupsd

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
)

func Test_nisReadWriteCloserRead(t *testing.T) {
	in := []byte("HELLO : WORLD")
	out := make([]byte, 16)

	rwc := testRWC(in, nil)

	for {
		// First write returns "key : value" data, second should
		// return EOF
		n, err := rwc.Read(out)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read: %v", err)
		}

		if want, got := in, out[:n]; !bytes.Equal(want, got) {
			t.Fatalf("unexpected byte output:\n- want: %v\n-  got: %v",
				want, got)
		}
	}
}

func Test_nisReadWriteCloserWriteBufferTooLarge(t *testing.T) {
	rwc := testRWC(nil, nil)

	_, err := rwc.Write(make([]byte, math.MaxUint16+1))

	if want, got := errBufferTooLarge, err; want != got {
		t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func Test_nisReadWriteCloserWrite(t *testing.T) {
	in := []byte("HELLO : WORLD")
	out := make([]byte, 16)

	rwc := testRWC(nil, out)

	n, err := rwc.Write(in)
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Write prepends length; two bytes
	n += 2
	in = append([]byte{0, byte(len(in))}, in...)

	if want, got := in, out[:n]; !bytes.Equal(want, got) {
		t.Fatalf("unexpected byte output:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func testRWC(rb []byte, wb []byte) io.ReadWriteCloser {
	return newNISReadWriteCloser(&testReadWriterCloser{
		rb: rb,
		wb: wb,
	})
}

var _ io.ReadWriteCloser = &testReadWriterCloser{}

type testReadWriterCloser struct {
	ri int
	rb []byte

	wi int
	wb []byte
}

func (rwc *testReadWriterCloser) Read(b []byte) (int, error) {
	defer func() { rwc.ri++ }()

	switch rwc.ri {
	case 0:
		// Send length of read buffer
		binary.BigEndian.PutUint16(b[0:2], uint16(len(rwc.rb)))
		return 2, nil
	case 1:
		// Send read buffer
		n := copy(b, rwc.rb)
		return n, nil
	}

	// Signal EOF to nisReadWriteCloser
	binary.BigEndian.PutUint16(b[0:2], 0)
	return 2, nil
}

func (rwc *testReadWriterCloser) Write(b []byte) (int, error) {
	n := copy(rwc.wb, b)
	return n, nil
}

func (rwc *testReadWriterCloser) Close() error { return nil }
