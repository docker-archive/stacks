package store

import (
	// TODO(dperny): make better errors
	"github.com/pkg/errors"

	"github.com/containerd/typeurl"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/swarmkit/api"
	gogotypes "github.com/gogo/protobuf/types"

	"github.com/docker/stacks/pkg/types"
)

func init() {
	typeurl.Register(&types.StackSpec{}, "github.com/docker/stacks/StackSpec")
}

// MarshalStackSpec takes a types.StackSpec object and marshals it into a
// protocol buffer Any message. Under the hood, this relies on marshaling
// the objects to JSON.
func MarshalStackSpec(stackSpec *types.StackSpec) (*gogotypes.Any, error) {
	return typeurl.MarshalAny(stackSpec)

}

// UnmarshalStackSpec does the MarshalStacks operation in reverse -- takes a proto
// message, and returns the stack contained in it.
func UnmarshalStackSpec(payload *gogotypes.Any) (*types.StackSpec, error) {
	iface, err := typeurl.UnmarshalAny(payload)
	if err != nil {
		return nil, err
	}
	// this is a naked cast, which means if for some reason this _isn't_ a
	// StackSpec object, the program will panic. This is fine, because if
	// such a thing were to occur, it would be panic-worthy.
	stackSpec := iface.(*types.StackSpec)

	return stackSpec, nil
}

// ConstructStack takes a Swarmkit Resource object, calls UnmarshalStackSpec and
// constructs a fresh types.Stack and populates its fields (Meta, Version, and ID)
// contained in the Resource
func ConstructStack(resource *api.Resource) (types.Stack, error) {

	// now, we have to get the stack out of the resource object
	stackSpec, err := UnmarshalStackSpec(resource.Payload)
	if err != nil {
		return types.Stack{}, err
	}
	if stackSpec == nil {
		return types.Stack{}, errors.New("got back an empty stack")
	}

	// extract the times from the swarmkit resource message.
	createdAt, err := gogotypes.TimestampFromProto(resource.Meta.CreatedAt)
	if err != nil {
		return types.Stack{}, errors.Wrap(err, "error converting swarmkit timestamp")
	}
	updatedAt, err := gogotypes.TimestampFromProto(resource.Meta.UpdatedAt)
	if err != nil {
		return types.Stack{}, errors.Wrap(err, "error converting swarmkit timestamp")
	}

	stack := types.Stack{
		ID: resource.ID,
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: resource.Meta.Version.Index,
			},
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Spec: *stackSpec,
	}
	return stack, nil
}
