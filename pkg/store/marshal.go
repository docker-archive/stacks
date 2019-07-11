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
	typeurl.Register(&types.Stack{}, "github.com/docker/stacks/Stack")
}

// MarshalStacks takes a Stack object and marshals it into a protocol buffer
// Any message. Under the hood, this relies on marshaling the objects to JSON.
func MarshalStacks(stack *types.Stack) (*gogotypes.Any, error) {
	return typeurl.MarshalAny(stack)

}

// UnmarshalStacks does the MarshalStacks operation in reverse -- takes a proto
// message, and returns the stack contained in it. Note that
// UnmarshalStacks takes a Swarmkit Resource object, instead of an Any proto.
// This is because UnmarshalStacks does the work of updating the fields in the
// Stack (Meta, Version, and ID) that are derrived from the values assigned by
// swarmkit and contained in the Resource
func UnmarshalStacks(resource *api.Resource) (*types.Stack, error) {
	iface, err := typeurl.UnmarshalAny(resource.Payload)
	if err != nil {
		return nil, err
	}
	// this is a naked cast, which means if for some reason this _isn't_ a
	// Stack object, the program will panic. This is fine, because if
	// such a thing were to occur, it would be panic-worthy.
	stack := iface.(*types.Stack)

	// extract the times from the swarmkit resource message.
	createdAt, err := gogotypes.TimestampFromProto(resource.Meta.CreatedAt)
	if err != nil {
		return nil, errors.Wrap(err, "error converting swarmkit timestamp")
	}
	updatedAt, err := gogotypes.TimestampFromProto(resource.Meta.UpdatedAt)
	if err != nil {
		return nil, errors.Wrap(err, "error converting swarmkit timestamp")
	}

	stack.ID = resource.ID
	stack.Meta = swarm.Meta{
		Version: swarm.Version{
			Index: resource.Meta.Version.Index,
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	return stack, nil
}
