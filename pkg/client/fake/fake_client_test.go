package fake

import (
	"context"
	"testing"

	"github.com/docker/docker/errdefs"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	composeTypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"
)

var stackCreate = types.StackCreate{
	Orchestrator: types.OrchestratorSwarm,
	Spec: types.StackSpec{
		Metadata: types.Metadata{
			Name: "teststack",
			Labels: map[string]string{
				"key": "value",
			},
		},
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
			},
		},
	},
}

func TestFakeStackClientParseComposeInput(t *testing.T) {
	c := NewStackClient()
	stackCreate, err := c.ParseComposeInput(context.TODO(), types.ComposeInput{})
	require.NoError(t, err)
	require.Nil(t, stackCreate)
}

func TestFakeStackClientUpdateOutOfSequence(t *testing.T) {
	ctx := context.Background()
	require := require.New(t)
	c := NewStackClient()

	resp, err := c.StackCreate(ctx, stackCreate, types.StackCreateOptions{})
	require.NoError(err)
	stack, err := c.StackInspect(ctx, resp.ID)
	require.NoError(err)

	stackSpec := stack.Spec
	stackSpec.Services[0].Image = "newimage"
	require.NoError(c.StackUpdate(ctx, resp.ID, stack.Version, stackSpec, types.StackUpdateOptions{}))

	err = c.StackUpdate(ctx, resp.ID, stack.Version, stackSpec, types.StackUpdateOptions{})
	require.Error(err)
	require.Contains(err.Error(), "update out of sequence")
}

func TestFakeStackClientCRUD(t *testing.T) {
	ctx := context.Background()
	require := require.New(t)
	c := NewStackClient()

	// Create
	resp, err := c.StackCreate(ctx, stackCreate, types.StackCreateOptions{})
	require.NoError(err)
	require.Equal(resp.ID, "1")

	// List
	stacks, err := c.StackList(ctx, types.StackListOptions{})
	require.NoError(err)
	require.Len(stacks, 1)
	require.Equal(stacks[0].Spec, stackCreate.Spec)
	require.Equal(stacks[0].ID, resp.ID)

	// Inspect
	stack, err := c.StackInspect(ctx, resp.ID)
	require.NoError(err)
	require.Equal(stack.Spec, stackCreate.Spec)
	require.Equal(stack.ID, resp.ID)

	// Update
	stackSpec := stack.Spec
	stackSpec.Services[0].Image = "newimage"
	require.NoError(c.StackUpdate(ctx, resp.ID, stack.Version, stackSpec, types.StackUpdateOptions{}))
	stack, err = c.StackInspect(ctx, resp.ID)
	require.NoError(err)
	assert.DeepEqual(t, stackSpec, stack.Spec)

	// Delete
	err = c.StackDelete(ctx, resp.ID)
	require.NoError(err)
	stack, err = c.StackInspect(ctx, resp.ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(stack)
	stacks, err = c.StackList(ctx, types.StackListOptions{})
	require.NoError(err)
	require.Len(stacks, 0)
}
