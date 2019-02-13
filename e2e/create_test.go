package e2e

import (
	"testing"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestBasicCreate(t *testing.T) {
	logrus.Info("XXX Connecting...")
	_, err := client.NewClientWithOpts(client.FromEnv)
	assert.NilError(t, err)
	logrus.Info("XXX Connected...")

	// TODO - build a real test here..
}
