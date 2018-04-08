package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseArgs(t *testing.T) {
	command, params := ParseArgs([]string{"on", "a=b", "b=1"})
	assert.Equal(t, command, "on")
	assert.Equal(t, params, map[string]interface{}{"a": "b", "b": float64(1)})
}
