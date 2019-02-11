package backend

import (
	"fmt"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

// StacksBackend implements the router.Backend interface, which serves as the
// API handler for the Stacks APIs.
type StacksBackend struct {
	// stackStore is the underlying CRUD store of stacks.
	stackStore interfaces.StackStore
}

// NewStacksBackend creates a new StacksBackend
func NewStacksBackend(stackStore interfaces.StackStore) *StacksBackend {
	return &StacksBackend{
		stackStore: stackStore,
	}
}

// CreateStack creates a new stack if the stack is valid.
func (b *StacksBackend) CreateStack(create types.StackCreate) (types.StackCreateResponse, error) {
	if create.Orchestrator != types.OrchestratorSwarm {
		return types.StackCreateResponse{}, fmt.Errorf("invalid orchestrator type %s. This backend only supports orchestrator type swarm", create.Orchestrator)
	}

	err := validateSpec(create.Spec)
	if err != nil {
		return types.StackCreateResponse{}, fmt.Errorf("invalid stack spec: %s", err)
	}

	id, err := b.stackStore.AddStack(create.Spec)
	if err != nil {
		return types.StackCreateResponse{}, fmt.Errorf("unable to store stack: %s", err)
	}

	return types.StackCreateResponse{
		ID: id,
	}, err
}

// GetStack retrieves a stack by its ID.
func (b *StacksBackend) GetStack(id string) (types.Stack, error) {
	stack, err := b.stackStore.GetStack(id)
	if err != nil {
		return types.Stack{}, fmt.Errorf("unable to retrieve stack %s: %s", id, err)
	}

	return stack, err
}

// ListStacks lists all stacks.
func (b *StacksBackend) ListStacks() ([]types.Stack, error) {
	// TODO: consider adding filters

	stacks, err := b.stackStore.ListStacks()
	return stacks, err
}

// UpdateStack updates a stack.
func (b *StacksBackend) UpdateStack(id string, spec types.StackSpec) error {
	err := validateSpec(spec)
	if err != nil {
		return fmt.Errorf("invalid stack spec: %s", err)
	}

	return b.stackStore.UpdateStack(id, spec)
}

// DeleteStack deletes a stack.
func (b *StacksBackend) DeleteStack(id string) error {
	return b.stackStore.DeleteStack(id)
}

// validateSpec returns an error if the provided StackSpec is not valid.
func validateSpec(_ types.StackSpec) error {
	// TODO(alexmavr): implement
	return nil
}
