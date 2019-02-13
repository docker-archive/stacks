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
}

// NewDefaultStacksBackend creates a new DefaultStacksBackend.
func NewDefaultStacksBackend(stackStore interfaces.StackStore) *DefaultStacksBackend {
	return &DefaultStacksBackend{
		stackStore: stackStore,
	}
}

// CreateStack creates a new stack if the stack is valid.
func (b *DefaultStacksBackend) CreateStack(create types.StackCreate) (types.StackCreateResponse, error) {
	if create.Orchestrator != types.OrchestratorSwarm {
		return types.StackCreateResponse{}, fmt.Errorf("invalid orchestrator type %s. This backend only supports orchestrator type swarm", create.Orchestrator)
	}

	err := validateSpec(create.Spec)
	if err != nil {
		return types.StackCreateResponse{}, fmt.Errorf("invalid stack spec: %s", err)
	}

	// Create the Swarm Stack object
	stack := types.Stack{
		Spec:         create.Spec,
		Orchestrator: types.OrchestratorSwarm,
	}

	// TODO convert to SwarmStack
	swarmStack := interfaces.SwarmStack{
		Spec: interfaces.SwarmStackSpec{},
	}

	id, err := b.stackStore.AddStack(stack, swarmStack)
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

// GetSwarmStack retrieves a swarm stack by its ID.
// NOTE: this is an internal-only method used by the Swarm Stacks Reconciler.
func (b *DefaultStacksBackend) GetSwarmStack(id string) (interfaces.SwarmStack, error) {
	stack, err := b.stackStore.GetSwarmStack(id)
	if err != nil {
		return interfaces.SwarmStack{}, fmt.Errorf("unable to retrieve swarm stack %s: %s", id, err)
	}

	return stack, err
}

// ListStacks lists all stacks.
func (b *DefaultStacksBackend) ListStacks() ([]types.Stack, error) {
	return b.stackStore.ListStacks()
}

// ListSwarmStacks lists all swarm stacks.
// NOTE: this is an internal-only method used by the Swarm Stacks Reconciler.
func (b *DefaultStacksBackend) ListSwarmStacks() ([]interfaces.SwarmStack, error) {
	return b.stackStore.ListSwarmStacks()
}

// UpdateStack updates a stack.
func (b *DefaultStacksBackend) UpdateStack(id string, spec types.StackSpec) error {
	err := validateSpec(spec)
	if err != nil {
		return fmt.Errorf("invalid stack spec: %s", err)
	}

	// TODO: convert to SwarmStackSpec
	swarmSpec := interfaces.SwarmStackSpec{}

	return b.stackStore.UpdateStack(id, spec, swarmSpec)
}

// DeleteStack deletes a stack.
func (b *DefaultStacksBackend) DeleteStack(id string) error {
	return b.stackStore.DeleteStack(id)
}

// validateSpec returns an error if the provided StackSpec is not valid.
func validateSpec(_ types.StackSpec) error {
	// TODO(alexmavr): implement
	return nil
}
