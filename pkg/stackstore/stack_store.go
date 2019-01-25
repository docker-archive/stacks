package stackstore

import (
	"github.com/docker/stacks/pkg/types"
)

// StackStore defines an interface to an arbitrary store which is able to
// perform CRUD operations for all objects required by the Stacks Controller.
type StackStore interface {
	AddStack(*types.Stack) error
	GetStack(id string) (*types.Stack, error)
	ListStacks() ([]*types.Stack, error)
	UpdateStack(id string, spec *types.StackSpec) error
	DeleteStack(id string) error
}
