package apcupsd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClientNoKnownKeyValuePairs(t *testing.T) {
	c, done := testClient(t, func() [][]byte {
		lenb, kvb := kvBytes("FOO : BAR")
		return [][]byte{
			lenb,
			kvb,
		}
	})
	defer done()

	s, err := c.Status()
	if err != nil {
		t.Fatalf("failed to retrieve status: %v", err)
	}

	if want, got := new(Status), s; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected Status:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestClientAllTypesKeyValuePairs(t *testing.T) {
	kvs := []string{
		"DATE     : 2016-09-06 22:13:28 -0400",
		"HOSTNAME : example",
		"LOADPCT  :  13.0 Percent Load Capacity",
		"BATTDATE : 2016-09-06",
		"TIMELEFT :  46.5 Minutes",
		"TONBATT  : 0 seconds",
		"NUMXFERS : 0",
		"SELFTEST : NO",
		"NOMPOWER : 865 Watts",
	}

	edt := time.FixedZone("EDT", -60*60*4)
	want := &Status{
		Date:            time.Date(2016, time.September, 6, 22, 13, 28, 0, edt),
		Hostname:        "example",
		LoadPercent:     13.0,
		BatteryDate:     time.Date(2016, time.September, 6, 0, 0, 0, 0, time.UTC),
		TimeLeft:        46*time.Minute + 30*time.Second,
		TimeOnBattery:   0 * time.Second,
		NumberTransfers: 0,
		Selftest:        false,
		NominalPower:    865,
	}

	c, done := testClient(t, func() [][]byte {
		var out [][]byte
		for _, kv := range kvs {
			lenb, kvb := kvBytes(kv)
			out = append(out, lenb)
			out = append(out, kvb)
		}

		return out
	})
	defer done()

	got, err := c.Status()
	if err != nil {
		t.Fatalf("failed to retrieve status: %v", err)
	}

	// Compare date UNIX timestamps separately for ease of testing
	if want, got := want.Date.Unix(), got.Date.Unix(); want != got {
		t.Fatalf("unexpected Date timestamp:\n- want: %v\n-  got: %v",
			want, got)
	}

	// Set to zero value for comparison
	want.Date = time.Time{}
	got.Date = time.Time{}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected Status:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func testClient(t *testing.T, fn func() [][]byte) (*Client, func()) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		c, err := l.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}

			panic(fmt.Sprintf("failed to accept connection: %v", err))
		}

		in := make([]byte, 128)
		n, err := c.Read(in)
		if err != nil {
			panic(fmt.Sprintf("failed to read from connection: %v", err))
		}

		status := []byte{0, 6, 's', 't', 'a', 't', 'u', 's'}
		if want, got := status, in[:n]; !bytes.Equal(want, got) {
			panic(fmt.Sprintf("unexpected request from Client:\n- want: %v\n - got: %v",
				want, got))
		}

		// Run against test function and append EOF to end of output bytes
		out := fn()
		out = append(out, []byte{0, 0})

		for _, o := range out {
			if _, err := c.Write(o); err != nil {
				panic(fmt.Sprintf("failed to write to connection: %v", err))
			}
		}

		wg.Done()
	}()

	c, err := Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial Client: %v", err)
	}

	done := func() {
		wg.Wait()
		_ = c.Close()
		_ = l.Close()
	}

	return c, done
}

// kvBytes is a helper to generate length and key/value byte buffers.
func kvBytes(kv string) ([]byte, []byte) {
	lenb := make([]byte, 2)
	binary.BigEndian.PutUint16(lenb, uint16(len(kv)))

	return lenb, []byte(kv)
}
