package backend

import (
	"fmt"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

// DefaultStacksBackend implements the interfaces.StacksBackend interface, which serves as the
// API handler for the Stacks APIs.
type DefaultStacksBackend struct {
	// stackStore is the underlying CRUD store of stacks.
	stackStore interfaces.StackStore

	// swarmBackend provides access to swarmkit operations on secrets
	// and configs, required for stack validation and conversion.
	swarmBackend interfaces.SwarmResourceBackend
}

// NewDefaultStacksBackend creates a new DefaultStacksBackend.
func NewDefaultStacksBackend(stackStore interfaces.StackStore, swarmBackend interfaces.SwarmResourceBackend) *DefaultStacksBackend {
	return &DefaultStacksBackend{
		stackStore:   stackStore,
		swarmBackend: swarmBackend,
	}
}

// CreateStack creates a new stack if the stack is valid.
func (b *DefaultStacksBackend) CreateStack(stackSpec types.StackSpec) (types.StackCreateResponse, error) {

	// FIXME: Structural Preconditions?
	if stackSpec.Annotations.Name == "" {
		return types.StackCreateResponse{}, fmt.Errorf("StackSpec contains no name")
	}

	id, err := b.stackStore.AddStack(stackSpec)
	if err != nil {
		return types.StackCreateResponse{}, fmt.Errorf("unable to store stack: %s", err)
	}

	return types.StackCreateResponse{
		ID: id,
	}, err
}

// GetStack retrieves a stack by its ID.
func (b *DefaultStacksBackend) GetStack(id string) (types.Stack, error) {
	stack, err := b.stackStore.GetStack(id)
	if err != nil {
		return types.Stack{}, fmt.Errorf("unable to retrieve stack %s: %s", id, err)
	}

	return stack, err
}

// ListStacks lists all stacks.
func (b *DefaultStacksBackend) ListStacks() ([]types.Stack, error) {
	return b.stackStore.ListStacks()
}

// UpdateStack updates a stack.
func (b *DefaultStacksBackend) UpdateStack(id string, spec types.StackSpec, version uint64) error {
	return b.stackStore.UpdateStack(id, spec, version)
}

// DeleteStack deletes a stack.
func (b *DefaultStacksBackend) DeleteStack(id string) error {
	return b.stackStore.DeleteStack(id)
}
