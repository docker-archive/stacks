package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/compose/loader"
	"github.com/docker/stacks/pkg/types"
)

// StackClient is a fake implementation of the Stacks API.
type StackClient struct {
	stacks map[string]types.Stack
	idx    uint64
	mu     sync.RWMutex
}

// StackOptionFunc is the type used for functional arguments of the
// StackClient during its creation.
type StackOptionFunc func(*StackClient)

// WithStartingID is a StackOptionFunc which sets a starting ID. All
// allocated IDs will be greater than the provided startID.
func WithStartingID(startID uint64) func(*StackClient) {
	return func(c *StackClient) {
		c.idx = startID
	}
}

// NewStackClient creates a new StackClient.
func NewStackClient(optsFunc ...StackOptionFunc) *StackClient {
	c := &StackClient{
		stacks: make(map[string]types.Stack),
		idx:    1,
	}

	for _, f := range optsFunc {
		f(c)
	}

	return c
}

// ParseComposeInput is a passthrough to the actual loader implementation.
func (c *StackClient) ParseComposeInput(_ context.Context, input types.ComposeInput) (*types.StackCreate, error) {
	return loader.ParseComposeInput(input)
}

// StackCreate creates a new stack.
func (c *StackClient) StackCreate(_ context.Context, stack types.StackCreate, _ types.StackCreateOptions) (types.StackCreateResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	newStack := types.Stack{
		ID:   fmt.Sprintf("%d", c.idx),
		Spec: stack.Spec,
	}
	c.idx++
	c.stacks[newStack.ID] = newStack
	return types.StackCreateResponse{
		ID: newStack.ID,
	}, nil
}

// StackInspect inspects an existing stack.
func (c *StackClient) StackInspect(_ context.Context, id string) (types.Stack, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stack, ok := c.stacks[id]
	if !ok {
		return types.Stack{}, errdefs.NotFound(fmt.Errorf("stack not found"))
	}

	return stack, nil
}

// StackList lists all stacks.
func (c *StackClient) StackList(_ context.Context, _ types.StackListOptions) ([]types.Stack, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	allStacks := []types.Stack{}
	for _, stack := range c.stacks {
		allStacks = append(allStacks, stack)
	}

	return allStacks, nil
}

// StackUpdate updates a stack.
func (c *StackClient) StackUpdate(_ context.Context, id string, version types.Version, spec types.StackSpec, _ types.StackUpdateOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	stack, ok := c.stacks[id]
	if !ok {
		return errdefs.NotFound(fmt.Errorf("stack not found"))
	}

	if version.Index != stack.Version.Index {
		return fmt.Errorf("update out of sequence")
	}

	stack.Spec = spec
	stack.Version.Index++
	c.stacks[id] = stack
	return nil
}

// StackDelete deletes a stack.
func (c *StackClient) StackDelete(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.stacks, id)
	return nil
}
