package interfaces

import (
	"errors"
	"fmt"
	"testing"

	"github.com/docker/stacks/pkg/types"

	"github.com/stretchr/testify/require"
)

func generateFixtures(n int) []types.StackSpec {
	fixtures := make([]types.StackSpec, n)
	return fixtures
}

func TestCRUDFakeStackStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeStackStore()

	// Assert the store is empty
	stacks, err := store.ListStacks()
	require.NoError(err)
	require.Empty(stacks)

	stack, err := store.GetStack("doesntexist")
	require.Error(err)
	require.True(IsErrNotFound(err))
	require.Nil(stack)

	// Add three items
	fixtures := generateFixtures(4)
	for i := 0; i < 3; i++ {
		id, err := store.AddStack(fixtures[i])
		require.NoError(err, fmt.Sprintf("failed to add fixture %d", i))
		require.NotEmpty(id)
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

	for _, id := range []string{"0", "1", "2"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
	}

	stack, err = store.GetStack("0")
	require.NoError(err)
	require.NotNil(stack)
	require.Equal(stack.ID, "0")

	stack, err = store.GetStack("2")
	require.NoError(err)
	require.NotNil(stack)
	require.Equal(stack.ID, "2")

	// Remove a stack
	require.NoError(store.DeleteStack("1"))

	// Add a new stack
	id, err := store.AddStack(fixtures[3])
	require.NoError(err)
	require.NotEmpty(id)

	// Ensure that the deleted stack is not present
	stack, err = store.GetStack("1")
	require.Error(err)
	require.True(IsErrNotFound(err))
	require.Nil(stack)

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

	for _, name := range []string{"0", "2", "3"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}
}

func TestIsErrNotFound(t *testing.T) {
	require.True(t, IsErrNotFound(errNotFound))
	require.False(t, IsErrNotFound(errors.New("other error")))
}
