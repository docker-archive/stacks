package store

import (
	"context"

	"github.com/docker/stacks/pkg/types"
)

// StackStore is an implementation of the types.StackStore interface,
// which provides for the storage and retrieval of Stack objects from the
// swarmkit object store.
type StackStore struct {
	client ResourcesClient
}

// New creates a new StackStore using the provided client.
func New(client ResourcesClient) *StackStore {
	return &StackStore{
		client: client,
	}
}

// AddStack creates a new Stack object in the swarmkit data store. It returns
// the ID of the new object if successful, or an error otherwise.
func (s *StackStore) AddStack(stackSpec types.StackSpec) (string, error) {
	return AddStack(context.TODO(), s.client, stackSpec)
}

// UpdateStack updates an existing Stack object
func (s *StackStore) UpdateStack(id string, stackSpec types.StackSpec, version uint64) error {
	return UpdateStack(context.TODO(), s.client, id, stackSpec, version)
}

// DeleteStack removes the stacks with the given ID.
func (s *StackStore) DeleteStack(id string) error {
	return DeleteStack(context.TODO(), s.client, id)
}

// GetStack retrieves and returns an exist types.Stack object by ID.
func (s *StackStore) GetStack(id string) (types.Stack, error) {
	return GetStack(context.TODO(), s.client, id)
}

// ListStacks lists all available stack objects
func (s *StackStore) ListStacks() ([]types.Stack, error) {
	return ListStacks(context.TODO(), s.client)
}
