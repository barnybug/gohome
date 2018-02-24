package miflora

import "testing"

func TestDecodeSensors1(t *testing.T) {
	data := []byte{0x3e, 0x00, 0x00, 0xf7, 0x1e, 0x00, 0x00, 0x02, 0x16, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	s := decodeSensors(data)
	if s.Temperature != 6.2 {
		t.Errorf("Temperature incorrect, got: %.1f, expected: %.1f", s.Temperature, 6.2)
	}
	if s.Moisture != 2 {
		t.Errorf("Moisture incorrect, got: %.1f, expected: %.1f", s.Moisture, 2)
	}
	if s.Light != 7927 {
		t.Errorf("Light incorrect, got: %.1f, expected: %.1f", s.Light, 7927)
	}
	if s.Conductivity != 22 {
		t.Errorf("Conductivity incorrect, got: %d, expected: %d", s.Conductivity, 22)
	}
}

func TestDecodeSensors2(t *testing.T) {
	data := []byte{0xfe, 0xff, 0x00, 0xf7, 0x1e, 0x00, 0x00, 0x02, 0x16, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	s := decodeSensors(data)
	if s.Temperature != -0.2 {
		t.Errorf("Temperature incorrect, got: %.1f, expected: %.1f", s.Temperature, -0.2)
	}
}
