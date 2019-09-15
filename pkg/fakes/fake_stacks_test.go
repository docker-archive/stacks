package fakes

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/stretchr/testify/require"
)

func TestUpdateFakeStackStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeStackStore()
	store.SpecifyKeyPrefix("TestUpdateFakeStackStore")
	store.SpecifyErrorTrigger("TestUpdateFakeStackStore", FakeUnimplemented)

	stack1 := GetTestStack("stack1")
	stack2 := GetTestStack("stack2")

	id1, err := store.AddStack(stack1.Spec)
	require.NoError(err)

	astack, err := store.GetStack(id1)
	require.NoError(err)
	require.Equal(astack.ID, id1)
	require.True(reflect.DeepEqual(astack.Spec, stack1.Spec))

	updateErr :=
		store.UpdateStack(id1, stack2.Spec, astack.Version.Index)
	require.NoError(updateErr)

	// index out of whack
	updateErr =
		store.UpdateStack(id1, stack2.Spec, astack.Version.Index)
	require.Error(updateErr)

	// id missing
	updateErr =
		store.UpdateStack("123.456", stack2.Spec, astack.Version.Index)
	require.Error(updateErr)

	astack, err = store.GetStack(id1)
	require.NoError(err)
	require.Equal(astack.ID, id1)
	require.True(reflect.DeepEqual(astack.Spec, stack2.Spec))

	astack, err = store.GetStack("123.456")
	require.Error(err)

	// double creation
	_, err = store.AddStack(stack1.Spec)
	require.True(errdefs.IsAlreadyExists(err))
	require.Error(err)
}

func TestIsolationFakeStackStore(t *testing.T) {
	taintKey := "foo"
	taintValue := "bar"
	require := require.New(t)
	store := NewFakeStackStore()

	fixtures := GenerateStackFixtures(1, "TestIsolationFakeStackStore")
	spec := &fixtures[0].Spec

	id, err := store.AddStack(*spec)
	require.NoError(err)
	stack1, _ := store.GetStack(id)

	// 1. Isolation from creation argument

	require.True(reflect.DeepEqual(*spec, stack1.Spec))
	spec.Annotations.Labels[taintKey] = taintValue
	require.False(reflect.DeepEqual(*spec, stack1.Spec))

	// 2. Isolation between repeated calls to GetStack

	stackTaint, taintErr := store.GetStack(id)
	require.NoError(taintErr)
	stackTaint.Spec.Annotations.Labels[taintKey] = taintValue

	require.False(reflect.DeepEqual(stack1.Spec, stackTaint.Spec))

	// 3. Isolation from Update argument (using now changed spec)

	err = store.UpdateStack(id, *spec, 1)
	require.NoError(err)
	stackUpdated, _ := store.GetStack(id)

	require.True(reflect.DeepEqual(*spec, stackUpdated.Spec))
	delete(spec.Annotations.Labels, taintKey)
	require.False(reflect.DeepEqual(*spec, stackUpdated.Spec))

}

func TestSpecifiedErrorsFakeStackStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeStackStore()
	store.SpecifyKeyPrefix("SpecifiedError")
	store.SpecifyErrorTrigger("SpecifiedError", FakeUnimplemented)

	fixtures := GenerateStackFixtures(10, "TestSpecifiedErrorsFakeStackStore")

	var id string
	var err error

	// 0. Leaving untouched

	// 1. forced creation failure
	store.MarkStackSpecForError("SpecifiedError", &fixtures[1].Spec, "AddStack")

	_, err = store.AddStack(fixtures[1].Spec)
	require.True(errdefs.IsNotImplemented(err))
	require.Error(err)

	// 2. forced get failure after good create
	store.MarkStackSpecForError("SpecifiedError", &fixtures[2].Spec, "GetStack")

	id, err = store.AddStack(fixtures[2].Spec)
	require.NoError(err)
	_, err = store.GetStack(id)
	require.Error(err)

	// 3. forced update failure using untainted #0
	store.MarkStackSpecForError("SpecifiedError", &fixtures[3].Spec, "UpdateStack")

	id, err = store.AddStack(fixtures[3].Spec)
	require.NoError(err)
	_, err = store.GetStack(id)
	require.NoError(err)

	err = store.UpdateStack(id, fixtures[0].Spec, 1)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 4. acquired update failure using tainted #3
	id, err = store.AddStack(fixtures[4].Spec)
	require.NoError(err)

	// normal update using #0
	err = store.UpdateStack(id, fixtures[0].Spec, 1)
	require.NoError(err)

	// tainted update using tainted #3
	err = store.UpdateStack(id, fixtures[3].Spec, 2)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 5. forced remove failure
	store.MarkStackSpecForError("SpecifiedError", &fixtures[5].Spec, "DeleteStack")

	id, err = store.AddStack(fixtures[5].Spec)
	require.NoError(err)

	err = store.DeleteStack(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 6. acquired remove failure using tainted #5
	id, err = store.AddStack(fixtures[6].Spec)
	require.NoError(err)

	// update #6 using tainted #5
	err = store.UpdateStack(id, fixtures[5].Spec, 1)
	require.NoError(err)

	err = store.DeleteStack(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 7. forced query failure
	store.MarkStackSpecForError("SpecifiedError", &fixtures[7].Spec, "ListStacks")

	_, err = store.AddStack(fixtures[7].Spec)
	require.NoError(err)

	_, err = store.ListStacks()
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 8. force failures by manipulating raw datastructures
	id, err = store.AddStack(fixtures[8].Spec)
	require.NoError(err)

	rawStack := store.InternalGetStack(id)
	store.MarkStackSpecForError("SpecifiedError", &rawStack.CurrentSpec)

	err = store.DeleteStack(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	_, err = store.GetStack(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	err = store.UpdateStack(id, fixtures[0].Spec, 1)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// Perform a little raw API test coverage
	pointer := store.InternalDeleteStack(id)
	require.True(pointer == rawStack)

	pointer = store.InternalDeleteStack(id)
	require.Nil(pointer)

	pointer = store.InternalGetStack(id)
	require.Nil(pointer)

}

func TestCRDFakeStackStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeStackStore()

	stack, err := store.GetStack("doesntexist")
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(stack)

	// Add three items
	fixtures := GenerateStackFixtures(4, "TestCRDFakeStackStore")
	for i := 0; i < 3; i++ {
		id, err := store.AddStack(fixtures[i].Spec)
		require.NoError(err, fmt.Sprintf("failed to add fixture %d", i))
		require.NotNil(id)
	}

	// Assert we can list the three items and fetch them individually
	stacks, stacksErr := store.ListStacks()
	require.NoError(stacksErr)
	require.NotNil(stacks)
	require.Len(stacks, 3)

	found := make(map[string]struct{})
	for _, stack := range stacks {
		found[stack.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, id := range []string{"STK_1", "STK_2", "STK_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		stack, err = store.GetStack(id)
		require.NoError(err)
		require.Equal(stack.ID, id)

		// special test feature
		stack, err = store.GetStack(stack.Spec.Annotations.Name)
		require.NoError(err)
		require.Equal(stack.ID, id)
	}

	// Remove second stack
	require.NoError(store.DeleteStack(stacks[1].ID))

	// Remove second stack again
	require.Error(store.DeleteStack(stacks[1].ID))

	stacksPointers := store.InternalQueryStacks(nil)
	require.NotEmpty(stacksPointers)

	idFunction := func(i *interfaces.SnapshotStack) interface{} { return i }
	stacksPointers = store.InternalQueryStacks(idFunction)
	require.NotEmpty(stacksPointers)

	// Assert we can list the two items and fetch them individually
	stacks2, err2 := store.ListStacks()
	require.NoError(err2)
	require.NotNil(stacks2)
	require.Len(stacks2, 2)

	for _, id := range []string{"STK_1", "STK_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		stack, err = store.GetStack(id)
		require.NoError(err)
		require.Equal(stack.ID, id)
	}

	// Add a new stack
	id, err := store.AddStack(fixtures[3].Spec)
	require.NoError(err)
	require.NotNil(id)

	// Ensure that the deleted stack is not present
	stack, err = store.GetStack(stacks[1].ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))

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

	for _, name := range []string{"STK_1", "STK_3", "STK_4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}
}
