package backend

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	composeTypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

func TestStacksBackend(t *testing.T) {
	require := require.New(t)

	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore())

	// Attempt to create a stack with an invalid orchestrator type.
	_, err := b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorNone,
	})
	require.Error(err)
	require.Contains(err.Error(), "invalid orchestrator type")

	_, err = b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorKubernetes,
	})
	require.Error(err)
	require.Contains(err.Error(), "invalid orchestrator type")

	_, err = b.CreateStack(types.StackCreate{
		Orchestrator: "foobar",
	})
	require.Error(err)
	require.Contains(err.Error(), "invalid orchestrator type")

	// Ensure no stacks were created
	stacks, err := b.ListStacks()
	require.NoError(err)
	require.Empty(stacks)

	// Create a stack with a valid StackCreate
	stack1Spec := types.StackSpec{
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
			},
		},
	}

	resp, err := b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorSwarm,
		Spec:         stack1Spec,
	})
	require.NoError(err)
	require.Equal("1", resp.ID)

	// Create another stack
	stack2Spec := types.StackSpec{
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service2",
				Image: "image2",
			},
		},
	}

	resp, err = b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorSwarm,
		Spec:         stack2Spec,
	})
	require.NoError(err)
	require.Equal("2", resp.ID)

	// List both stacks
	stacks, err = b.ListStacks()
	require.NoError(err)
	require.Len(stacks, 2)

	found := map[string]string{
		"service1": "image1",
		"service2": "image2",
	}

	for _, stack := range stacks {
		require.Len(stack.Spec.Services, 1)
		serviceName := stack.Spec.Services[0].Name
		image, ok := found[serviceName]
		require.True(ok)
		require.Equal(image, stack.Spec.Services[0].Image)
		delete(found, serviceName)
	}
	require.Empty(found)

	// Get a stack by ID
	stack, err := b.GetStack("1")
	require.NoError(err)
	require.True(reflect.DeepEqual(stack.Spec, stack1Spec))
	require.Equal(stack.ID, "1")
	// TODO: require.Equal(stack.Orchestrator, types.OrchestratorSwarm)

	// Update a stack
	stack3Spec := types.StackSpec{
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service3",
				Image: "image3",
			},
		},
	}
	err = b.UpdateStack("2", stack3Spec)
	require.NoError(err)

	// Get the updated stack by ID
	stack, err = b.GetStack("2")
	require.NoError(err)
	require.True(reflect.DeepEqual(stack.Spec, stack3Spec))
	require.Equal(stack.ID, "2")

	// Remove a stack
	require.NoError(b.DeleteStack("2"))
	_, err = b.GetStack("2")
	require.Error(err)
	require.Contains(err.Error(), "stack not found")
}
