package camera

import (
	"testing"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
)

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	var _ services.ConfigSubscriber = (*Service)(nil)
	// Output:
}

func TestBadConfig(t *testing.T) {
	yml := `
camera:
  cameras:
    cam.one:
      protocol: x
      watch: /a/b
      match: +++
`
	_, err := config.OpenRaw([]byte(yml))
	assert.Error(t, err)
}

func TestGoodConfig(t *testing.T) {
	assert := assert.New(t)
	yml := `
camera:
  cameras:
    cam.one:
      protocol: x
      watch: /a/b
      match: /some/
`
	c, err := config.OpenRaw([]byte(yml))
	assert.NoError(err)
	assert.True(c.Camera.Cameras["cam.one"].Match.MatchString("/a/b/some/file.mp4"))
}
