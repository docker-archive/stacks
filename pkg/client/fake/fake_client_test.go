package fake

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/docker/stacks/pkg/types"
)

var stackCreate = types.StackSpec{
	Annotations: swarm.Annotations{
		Name: "teststack",
		Labels: map[string]string{
			"key": "value",
		},
	},
	Services: []swarm.ServiceSpec{
		{
			Annotations: swarm.Annotations{
				Name: "service1",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image: "image1",
				},
			},
		},
	},
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
	stackSpec.Services[0].TaskTemplate.ContainerSpec.Image = "newimage"
	require.NoError(c.StackUpdate(ctx, resp.ID, types.Version{Index: stack.Meta.Version.Index}, stackSpec, types.StackUpdateOptions{}))

	err = c.StackUpdate(ctx, resp.ID, types.Version{Index: stack.Meta.Version.Index}, stackSpec, types.StackUpdateOptions{})
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
	require.Equal(stacks[0].Spec, stackCreate)
	require.Equal(stacks[0].ID, resp.ID)

	// Inspect
	stack, err := c.StackInspect(ctx, resp.ID)
	require.NoError(err)
	require.Equal(stack.Spec, stackCreate)
	require.Equal(stack.ID, resp.ID)

	// Update
	stackSpec := stack.Spec
	stackSpec.Services[0].TaskTemplate.ContainerSpec.Image = "newimage"
	require.NoError(c.StackUpdate(ctx, resp.ID, types.Version{Index: stack.Meta.Version.Index}, stackSpec, types.StackUpdateOptions{}))
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
