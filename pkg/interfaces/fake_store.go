package interfaces

import (
	"errors"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/types"
)

// FakeStackStore stores stacks
type FakeStackStore struct {
	stacks map[string]types.Stack
	sync.RWMutex
	curID int
}

// NewFakeStackStore creates a new StackStore
func NewFakeStackStore() StackStore {
	return &FakeStackStore{
		stacks: make(map[string]types.Stack),
		// Don't start from ID 0, to catch any uninitialized types.
		curID: 1,
	}
}

var errNotFound = errdefs.NotFound(errors.New("stack not found"))

// IsErrNotFound return true if the error is a not-found error.
func IsErrNotFound(err error) bool {
	return err == errNotFound
}

// AddStack adds a stack to the store.
func (s *FakeStackStore) AddStack(stackSpec types.StackSpec) (string, error) {
	s.Lock()
	defer s.Unlock()

	stack := types.Stack{
		ID: fmt.Sprintf("%d", s.curID),
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: 1,
			},
		},
		Spec: stackSpec,
	}

	s.stacks[stack.ID] = stack

	s.curID++
	return stack.ID, nil
}

func (s *FakeStackStore) getStack(id string) (types.Stack, error) {
	stack, ok := s.stacks[id]
	if !ok {
		return types.Stack{}, errNotFound
	}

	return stack, nil
}

// UpdateStack updates the stack in the store.
func (s *FakeStackStore) UpdateStack(id string, stackSpec types.StackSpec, version uint64) error {
	s.Lock()
	defer s.Unlock()

	existingStack, err := s.getStack(id)
	if err != nil {
		return errNotFound
	}

	if existingStack.Version.Index != version {
		return fmt.Errorf("update out of sequence")
	}
	existingStack.Version.Index++

	existingStack.Spec = stackSpec
	s.stacks[id] = existingStack
	return nil
}

// DeleteStack removes a stack from the store.
func (s *FakeStackStore) DeleteStack(id string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.stacks, id)
	return nil
}

// GetStack retrieves a single stack from the store.
func (s *FakeStackStore) GetStack(id string) (types.Stack, error) {
	s.RLock()
	defer s.RUnlock()
	stack, err := s.getStack(id)
	return stack, err
}

// ListStacks returns all known stacks from the store.
func (s *FakeStackStore) ListStacks() ([]types.Stack, error) {
	s.RLock()
	defer s.RUnlock()
	stacks := []types.Stack{}
	for _, stack := range s.stacks {
		stacks = append(stacks, stack)
	}
	return stacks, nil
}
