package stackstore

import (
	"errors"
	"fmt"
	"sync"
)

type FakeStackStore struct {
	stacks map[string]*Stack
	sync.RWMutex
}

func NewFakeStackStore() StackStore {
	return &FakeStackStore{
		stacks: make(map[string]*Stack),
	}
}

var errNotFound = errors.New("stack not found")

func IsErrNotFound(err error) bool {
	return err == errNotFound
}

func (s *FakeStackStore) AddStack(stack *Stack) error {
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

func (s *FakeStackStore) getStack(id string) (*Stack, error) {
	stack, ok := s.stacks[id]
	if !ok {
		return nil, errNotFound
	}

	return stack, nil
}

func (s *FakeStackStore) UpdateStack(id string, spec *StackSpec) error {
	s.Lock()
	defer s.Unlock()
	if spec == nil {
		return fmt.Errorf("unable to update stack '%s' with nil spec", id)
	}

	existingStack, err := s.getStack(id)
	if err != nil {
		return fmt.Errorf("unable to retrieve existing stack with ID '%s': %s", id, err)
	}

	existingStack.Spec = spec
	return nil
}

func (s *FakeStackStore) DeleteStack(id string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.stacks, id)
	return nil
}

func (s *FakeStackStore) GetStack(name string) (*Stack, error) {
	s.RLock()
	defer s.RUnlock()
	return s.getStack(name)
}

func (s *FakeStackStore) ListStacks() ([]*Stack, error) {
	s.RLock()
	defer s.RUnlock()
	stacks := []*Stack{}
	for _, stack := range s.stacks {
		stacks = append(stacks, stack)
	}
	return stacks, nil
}
