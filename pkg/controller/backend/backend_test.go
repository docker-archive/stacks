package backend

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/mocks"
	"github.com/docker/stacks/pkg/types"
)

func TestStacksBackendUpdateOutOfSequence(t *testing.T) {
	// This test ensures that we cannot globber changes by performing updates
	// with invalid versions.
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	// Create a stack with a valid StackCreate
	response, err := b.CreateStack(types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "teststack",
		},
	})
	require.NoError(err)

	// Inspect the stack
	stack, err := b.GetStack(response.ID)
	require.NoError(err)

	stack.Spec.Annotations.Name = "test1"

	err = b.UpdateStack(stack.ID, stack.Spec, stack.Version.Index)
	require.NoError(err)

	stack.Spec.Annotations.Name = "test2"
	err = b.UpdateStack(stack.ID, stack.Spec, stack.Version.Index)
	require.Error(err)
	require.Contains(err.Error(), "out of sequence")

	stack, err = b.GetStack(stack.ID)
	require.NoError(err)
	require.Equal(stack.Spec.Annotations.Name, "test1")
}

func TestStacksBackendInvalidCreate(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	_, err := b.CreateStack(types.StackSpec{})
	require.Error(err)
	require.Contains(err.Error(), "contains no name")

	// Ensure no stacks were created
	stacks, err := b.ListStacks()
	require.NoError(err)
	require.Empty(stacks)
}

func TestStacksBackendCRUD(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	// Create a stack with a valid StackCreate
	service1Spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "service1",
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "image1",
			},
		},
	}

	service2Spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "service2",
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "image2",
			},
		},
	}

	service3Spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "service3",
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "image3",
			},
		},
	}

	stack1Spec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "stack1",
		},
		Services: []swarm.ServiceSpec{
			service1Spec,
		},
	}

	response, err := b.CreateStack(stack1Spec)
	require.NoError(err)
	require.Equal("|0001", response.ID)

	// Create another stack
	stack2Spec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "stack2",
		},
		Services: []swarm.ServiceSpec{
			service2Spec,
		},
	}

	response, err = b.CreateStack(stack2Spec)
	require.NoError(err)
	require.Equal("|0002", response.ID)

	// List both stacks
	stacks, err := b.ListStacks()
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
		require.Equal(image, stack.Spec.Services[0].TaskTemplate.ContainerSpec.Image)
		delete(found, serviceName)
	}
	require.Empty(found)

	// Get a stack by ID
	stack, err := b.GetStack("|0001")
	require.NoError(err)
	require.True(reflect.DeepEqual(stack.Spec, stack1Spec))
	require.Equal(stack.ID, "|0001")
	// TODO: require.Equal(stack.Orchestrator, types.OrchestratorSwarm)

	// Update a stack
	stack3Spec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "stack3",
		},
		Services: []swarm.ServiceSpec{
			service3Spec,
		},
	}

	stack2, err := b.GetStack("|0002")
	require.NoError(err)
	err = b.UpdateStack("|0002", stack3Spec, stack2.Version.Index)
	require.NoError(err)

	// Get the updated stack by ID
	stack, err = b.GetStack("|0002")
	require.NoError(err)
	require.True(reflect.DeepEqual(stack.Spec, stack3Spec))
	require.Equal(stack.ID, "|0002")

	// Remove a stack
	require.NoError(b.DeleteStack("|0002"))
	_, err = b.GetStack("|0002")
	require.Error(err)
	require.Contains(err.Error(), "stack not found")
}
