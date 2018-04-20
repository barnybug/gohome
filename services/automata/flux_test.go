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

func TestFluxParse(t *testing.T) {
	assert := assert.New(t)

	p, err := fluxParse([]interface{}{"19:00->23:00", "level=90->10", "temp=3000->2200"})
	assert.NoError(err)
	assert.Equal(19, p.tStart.Hour())
	assert.Equal(0, p.tStart.Minute())
	assert.Equal(23, p.tEnd.Hour())
	assert.Equal(0, p.tEnd.Minute())
	assert.Equal(3000, p.kStart)
	assert.Equal(2200, p.kEnd)
	assert.Equal(90, p.lStart)
	assert.Equal(10, p.lEnd)
}

func _t(s string) time.Time {
	t, _ := time.Parse(Kitchen24, s)
	return t
}

func TestInterpolate(t *testing.T) {
	assert := assert.New(t)

	mockTime("19:00")
	assert.Equal(3000, tinterpolate(_t("19:00"), _t("22:00"), 3000, 2200))
	mockTime("22:00")
	assert.Equal(2200, tinterpolate(_t("19:00"), _t("22:00"), 3000, 2200))
	mockTime("20:30")
	assert.Equal(2600, tinterpolate(_t("19:00"), _t("22:00"), 3000, 2200))
}

func TestFluxCommand(t *testing.T) {
	assert := assert.New(t)

	p := fluxParams{
		tStart: _t("19:00"),
		tEnd:   _t("22:00"),
		kStart: 3000,
		kEnd:   2200,
		lStart: 90,
		lEnd:   10,
	}
	mockTime("20:30")
	ev := fluxCommand(p, "device")
	assert.Equal(2600, ev.Fields["temp"])
	assert.Equal(50, ev.Fields["level"])
}
