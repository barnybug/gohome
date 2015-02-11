package util

import (
	"os"
	"testing"
)

func TestExpandUser(t *testing.T) {
	path := ExpandUser("~/abc")
	expected := os.ExpandEnv("$HOME/abc")
	if path != expected {
		t.Error("Expected ", expected, ", got ", path)
	}
}
