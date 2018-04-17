package automata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func mockTime(s string) {
	t, _ := time.Parse(Kitchen24, s)
	now = func() time.Time { return t }
}

func TestTinterpolate(t *testing.T) {
	assert := assert.New(t)

	mockTime("19:00")
	assert.Equal(3000, tinterpolate("19:00", "22:00", 3000, 2200))
	mockTime("22:00")
	assert.Equal(2200, tinterpolate("19:00", "22:00", 3000, 2200))
	mockTime("20:30")
	assert.Equal(2600, tinterpolate("19:00", "22:00", 3000, 2200))
}
