package watchdog

import (
	"testing"

	"github.com/barnybug/gohome/config"
	"github.com/barnybug/gohome/services"
	"github.com/stretchr/testify/assert"
)

func ExampleInterfaces() {
	var _ services.Service = (*Service)(nil)
	// Output:
}

func TestBadConfig(t *testing.T) {
	yml := `
devices:
  one:
    watchdog: xyz
`
	_, err := config.OpenRaw([]byte(yml))
	assert.Error(t, err)
}
