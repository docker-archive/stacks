package e2e

import (
	"context"
	"testing"

	"github.com/docker/stacks/pkg/client"
	"github.com/docker/stacks/pkg/compose/loader"
	"github.com/docker/stacks/pkg/types"

	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestBasicCreate(t *testing.T) {
	ctx := context.Background()
	settings, err := client.SettingsFromEnv()
	assert.NilError(t, err)
	cli, err := client.NewClientWithSettings(*settings)
	assert.NilError(t, err)

	// Send a compose file and get the create type
	input, err := loader.LoadComposefile([]string{"./pkg/compose/tests/fixtures/default-env-file/docker-compose.yml"})
	assert.NilError(t, err)

	logrus.Info("Parsing compose file on the server.")
	stack, err := cli.ParseComposeInput(ctx, *input)
	assert.NilError(t, err)
	assert.Check(t, is.Len(stack.Spec.Services, 1))
	assert.Check(t, is.Len(stack.Spec.PropertyValues, 4))
	stack.Orchestrator = "swarm"

	logrus.Info("Creating stack.")
	options := types.StackCreateOptions{}

	_, err = cli.StackCreate(ctx, *stack, options)
	assert.NilError(t, err)

	// TODO - add validation of the returned stack

}
