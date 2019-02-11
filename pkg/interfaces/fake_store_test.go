package interfaces

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	composeTypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"

	"github.com/stretchr/testify/require"
)

func generateFixtures(n int) []types.StackSpec {
	fixtures := make([]types.StackSpec, n)
	return fixtures
}

func TestUpdateFakeStackStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeStackStore()

	spec1 := types.StackSpec{
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
			},
		},
	}

	spec2 := types.StackSpec{
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service2",
				Image: "image2",
			},
		},
	}

	id, err := store.AddStack(spec1)
	require.NoError(err)

	stack, err := store.GetStack(id)
	require.NoError(err)
	require.Equal(stack.ID, id)
	require.True(reflect.DeepEqual(stack.Spec, spec1))

	require.NoError(store.UpdateStack(id, spec2))

	stack, err = store.GetStack(id)
	require.NoError(err)
	require.Equal(stack.ID, id)
	require.True(reflect.DeepEqual(stack.Spec, spec2))
}

func TestCRDFakeStackStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeStackStore()

	// Assert the store is empty
	stacks, err := store.ListStacks()
	require.NoError(err)
	require.Empty(stacks)

	stack, err := store.GetStack("doesntexist")
	require.Error(err)
	require.True(IsErrNotFound(err))
	require.Empty(stack)

	// Add three items
	fixtures := generateFixtures(4)
	for i := 0; i < 3; i++ {
		id, err := store.AddStack(fixtures[i])
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

	stack, err = store.GetStack("1")
	require.NoError(err)
	require.Equal(stack.ID, "1")

	stack, err = store.GetStack("3")
	require.NoError(err)
	require.Equal(stack.ID, "3")

	// Remove a stack
	require.NoError(store.DeleteStack("2"))

	// Add a new stack
	id, err := store.AddStack(fixtures[3])
	require.NoError(err)
	require.NotNil(id)

	// Ensure that the deleted stack is not present
	stack, err = store.GetStack("2")
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
}

func TestIsErrNotFound(t *testing.T) {
	require.True(t, IsErrNotFound(errNotFound))
	require.False(t, IsErrNotFound(errors.New("other error")))
}
