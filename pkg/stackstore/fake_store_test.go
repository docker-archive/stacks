package stackstore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/docker/stacks/pkg/types"

	"github.com/stretchr/testify/require"
)

func generateFixtures(n int) []*types.Stack {
	fixtures := []*types.Stack{}
	for i := 1; i < n+1; i++ {
		fixtures = append(fixtures, &types.Stack{
			ID:   fmt.Sprintf("stack%d", i),
			Spec: types.StackSpec{},
		})
	}

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
		err = store.AddStack(fixtures[i])
		require.NoError(err, fmt.Sprintf("failed to add fixture %d", i))
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

	for _, name := range []string{"stack1", "stack2", "stack3"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}

	stack, err = store.GetStack("stack1")
	require.NoError(err)
	require.NotNil(stack)
	require.Equal(stack.ID, "stack1")

	stack, err = store.GetStack("stack3")
	require.NoError(err)
	require.NotNil(stack)
	require.Equal(stack.ID, "stack3")

	// Remove a stack
	require.NoError(store.DeleteStack("stack2"))

	// Add a new stack
	require.NoError(store.AddStack(fixtures[3]))

	// Ensure that the deleted stack is not present
	stack, err = store.GetStack("stack2")
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

	for _, name := range []string{"stack1", "stack3", "stack4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}

}

func TestIsErrNotFound(t *testing.T) {
	require.True(t, IsErrNotFound(errNotFound))
	require.False(t, IsErrNotFound(errors.New("other error")))
}
