package router

import "github.com/docker/stacks/pkg/interfaces"

// Backend abstracts the Stacks API.
type Backend interface {
	CreateStack(interfaces.StackSpec) (string, error)
	GetStack(id string) (interfaces.Stack, error)
	ListStacks() ([]interfaces.Stack, error)
	UpdateStack(id string, spec interfaces.StackSpec, version uint64) error
	DeleteStack(id string) error
}
