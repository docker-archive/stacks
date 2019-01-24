package stackstore

// TODO: temporary types
type StackSpec struct {
	Stuff string
}

type Stack struct {
	ID       string
	Metadata *StackMetadata
	Spec     *StackSpec
}

type StackMetadata struct {
	Name string
}

// END OF TEMPORARY TYPES

// StackStore defines an interface to an arbitrary store which is able to
// perform CRUD operations for all objects required by the Stacks Controller.
type StackStore interface {
	AddStack(*Stack) error
	GetStack(id string) (*Stack, error)
	ListStacks() ([]*Stack, error)
	UpdateStack(id string, spec *StackSpec) error
	DeleteStack(id string) error
}
