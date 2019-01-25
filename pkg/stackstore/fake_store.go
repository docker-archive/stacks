package stackstore

import (
	"errors"
	"fmt"
	"github.com/docker/stacks/pkg/types"
	"sync"
)

// FakeStackStore stores stacks
type FakeStackStore struct {
	stacks map[string]*types.Stack
	sync.RWMutex
}

// NewFakeStackStore creates a new StackStore
func NewFakeStackStore() StackStore {
	return &FakeStackStore{
		stacks: make(map[string]*types.Stack),
	}
}

var errNotFound = errors.New("stack not found")

// IsErrNotFound return true if the error is a not-found error
func IsErrNotFound(err error) bool {
	return err == errNotFound
}

// AddStack adds a stack to the store
func (s *FakeStackStore) AddStack(stack *types.Stack) error {
	s.Lock()
	defer s.Unlock()
	if stack == nil {
		return fmt.Errorf("unable to create nil stack")
	}

	_, err := s.getStack(stack.ID)
	if err == nil {
		return fmt.Errorf("stack with id '%s' already exists", stack.ID)
	}

	s.stacks[stack.ID] = stack
	return nil
}

func (s *FakeStackStore) getStack(id string) (*types.Stack, error) {
	stack, ok := s.stacks[id]
	if !ok {
		return nil, errNotFound
	}

	return stack, nil
}

// UpdateStack updates the stack in the store
func (s *FakeStackStore) UpdateStack(id string, spec *types.StackSpec) error {
	s.Lock()
	defer s.Unlock()
	if spec == nil {
		return fmt.Errorf("unable to update stack '%s' with nil spec", id)
	}

	existingStack, err := s.getStack(id)
	if err != nil {
		return fmt.Errorf("unable to retrieve existing stack with ID '%s': %s", id, err)
	}

	existingStack.Spec = *spec
	return nil
}

// DeleteStack removes a stack from the store
func (s *FakeStackStore) DeleteStack(id string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.stacks, id)
	return nil
}

// GetStack retrieves a single stack from the store
func (s *FakeStackStore) GetStack(name string) (*types.Stack, error) {
	s.RLock()
	defer s.RUnlock()
	return s.getStack(name)
}

// ListStacks returns all known stacks from the store
func (s *FakeStackStore) ListStacks() ([]*types.Stack, error) {
	s.RLock()
	defer s.RUnlock()
	stacks := []*types.Stack{}
	for _, stack := range s.stacks {
		stacks = append(stacks, stack)
	}
	return stacks, nil
}
