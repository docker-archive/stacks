package interfaces

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	composeTypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"

	"github.com/stretchr/testify/require"
)

func generateFixtures(n int) []stackPair {
	fixtures := make([]stackPair, n)
	return fixtures
}

func getTestSpecs(name, image string) (types.StackSpec, SwarmStackSpec) {
	spec := types.StackSpec{
		Services: []composeTypes.ServiceConfig{
			{
				Name:  name,
				Image: image,
			},
		},
	}

	swarmSpec := SwarmStackSpec{
		Services: []swarm.ServiceSpec{
			{
				Annotations: swarm.Annotations{
					Name: name,
				},
				TaskTemplate: swarm.TaskSpec{
					ContainerSpec: &swarm.ContainerSpec{
						Image: image,
					},
				},
			},
		},
	}

	return spec, swarmSpec
}

func getTestStacks(name, image string) (types.Stack, SwarmStack) {
	spec, swarmSpec := getTestSpecs(name, image)
	return types.Stack{
			Orchestrator: types.OrchestratorSwarm,
			Spec:         spec,
		}, SwarmStack{
			Spec: swarmSpec,
		}
}

func TestUpdateFakeStackStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeStackStore()

	stack1, swarmStack1 := getTestStacks("service1", "image1")
	stack2, swarmStack2 := getTestStacks("service2", "image2")

	id, err := store.AddStack(stack1, swarmStack1)
	require.NoError(err)

	stack, err := store.GetStack(id)
	require.NoError(err)
	require.Equal(stack.ID, id)
	require.True(reflect.DeepEqual(stack.Spec, stack1.Spec))

	swarmStack, err := store.GetSwarmStack(id)
	require.NoError(err)
	require.Equal(swarmStack.ID, id)
	require.True(reflect.DeepEqual(swarmStack.Spec, swarmStack1.Spec))

	require.NoError(store.UpdateStack(id, stack2.Spec, swarmStack2.Spec))

	stack, err = store.GetStack(id)
	require.NoError(err)
	require.Equal(stack.ID, id)
	require.True(reflect.DeepEqual(stack.Spec, stack2.Spec))

	swarmStack, err = store.GetSwarmStack(id)
	require.NoError(err)
	require.Equal(swarmStack.ID, id)
	require.True(reflect.DeepEqual(swarmStack.Spec, swarmStack2.Spec))
}

func TestCRDFakeStackStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeStackStore()

	// Assert the store is empty
	stacks, err := store.ListStacks()
	require.NoError(err)
	require.Empty(stacks)

	swarmStacks, err := store.ListSwarmStacks()
	require.NoError(err)
	require.Empty(swarmStacks)

	stack, err := store.GetStack("doesntexist")
	require.Error(err)
	require.True(IsErrNotFound(err))
	require.Empty(stack)

	swarmStack, err := store.GetSwarmStack("doesntexist")
	require.Error(err)
	require.True(IsErrNotFound(err))
	require.Empty(swarmStack)

	// Add three items
	fixtures := generateFixtures(4)
	for i := 0; i < 3; i++ {
		id, err := store.AddStack(fixtures[i].Stack, fixtures[i].SwarmStack)
		require.NoError(err, fmt.Sprintf("failed to add fixture %d", i))
		require.NotNil(id)
	}

	// Assert we can list the three items and fetch them individually
	stacks, err = store.ListStacks()
	require.NoError(err)
	require.NotNil(stacks)
	require.Len(stacks, 3)

	found := make(map[string]struct{})
	for _, stack := range stacks {
		found[stack.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, id := range []string{"1", "2", "3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
	}

	swarmStacks, err = store.ListSwarmStacks()
	require.NoError(err)
	require.Len(swarmStacks, 3)

	found = make(map[string]struct{})
	for _, stack := range swarmStacks {
		found[stack.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, id := range []string{"1", "2", "3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
	}

	stack, err = store.GetStack("1")
	require.NoError(err)
	require.Equal(stack.ID, "1")

	swarmStack, err = store.GetSwarmStack("1")
	require.NoError(err)
	require.Equal(swarmStack.ID, "1")

	stack, err = store.GetStack("3")
	require.NoError(err)
	require.Equal(stack.ID, "3")

	swarmStack, err = store.GetSwarmStack("3")
	require.NoError(err)
	require.Equal(swarmStack.ID, "3")

	// Remove a stack
	require.NoError(store.DeleteStack("2"))

	// Add a new stack
	id, err := store.AddStack(fixtures[3].Stack, fixtures[3].SwarmStack)
	require.NoError(err)
	require.NotNil(id)

	// Ensure that the deleted stack is not present
	stack, err = store.GetStack("2")
	require.Error(err)
	require.True(IsErrNotFound(err))

	swarmStack, err = store.GetSwarmStack("2")
	require.Error(err)
	require.True(IsErrNotFound(err))

	// Ensure the expected list of stacks is present
	stacks, err = store.ListStacks()
	require.NoError(err)
	require.NotNil(stacks)
	require.Len(stacks, 3)

	found = make(map[string]struct{})
	for _, stack := range stacks {
		found[stack.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, name := range []string{"1", "3", "4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}

	swarmStacks, err = store.ListSwarmStacks()
	require.NoError(err)
	require.NotNil(swarmStacks)
	require.Len(swarmStacks, 3)

	found = make(map[string]struct{})
	for _, stack := range swarmStacks {
		found[stack.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, name := range []string{"1", "3", "4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}
}

func TestIsErrNotFound(t *testing.T) {
	require.True(t, IsErrNotFound(errNotFound))
	require.False(t, IsErrNotFound(errors.New("other error")))
}
