package store

import (
	"context"

	"github.com/docker/stacks/pkg/interfaces"
)

// StackStore is an implementation of the interfaces.StackStore interface,
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
func (s *StackStore) AddStack(st interfaces.Stack) (string, error) {
	return AddStack(context.TODO(), s.client, st)
}

// UpdateStack updates an existing Stack object
func (s *StackStore) UpdateStack(id string, st interfaces.StackSpec, version uint64) error {
	return UpdateStack(context.TODO(), s.client, id, st, version)
}

// DeleteStack removes the stacks with the given ID.
func (s *StackStore) DeleteStack(id string) error {
	return DeleteStack(context.TODO(), s.client, id)
}

// GetStack retrieves and returns an exist interfaces.Stack object by ID.
func (s *StackStore) GetStack(id string) (interfaces.Stack, error) {
	return GetStack(context.TODO(), s.client, id)
}

// ListStacks lists all available stack objects
func (s *StackStore) ListStacks() ([]interfaces.Stack, error) {
	return ListStacks(context.TODO(), s.client)
}
