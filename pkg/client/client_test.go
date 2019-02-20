package client

import (
	"os"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestSettingsFromEnv(t *testing.T) {
	old := os.Getenv("DOCKER_HOST")
	os.Setenv("DOCKER_HOST", "http://localhost:2375")
	if old != "" {
		defer func() {
			os.Setenv("DOCKER_HOST", old)
		}()
	}
	settings, err := SettingsFromEnv()
	assert.NilError(t, err)
	assert.Check(t, is.Equal(settings.Scheme, "http"))
	assert.Check(t, is.Equal(settings.Host, "http://localhost:2375"))
}
