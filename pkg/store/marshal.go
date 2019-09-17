package store

import (
	// TODO(dperny): make better errors
	"github.com/pkg/errors"

	"github.com/containerd/typeurl"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/swarmkit/api"
	gogotypes "github.com/gogo/protobuf/types"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

func init() {
	typeurl.Register(&interfaces.SnapshotStack{}, "github.com/docker/interfaces/SnapshotStack")
}

// MarshalSnapshotStackSpec takes a interfaces.SnapshotStack and a
// types.StackSpec object and replaces the types.StackSpec object contained
// in interfaces.SnapshotStack and marshals it into a protocol buffer Any.
// Under the hood, this relies on marshaling the objects to JSON.
func MarshalSnapshotStackSpec(existingSnapshot *interfaces.SnapshotStack, stackSpec *types.StackSpec) (*gogotypes.Any, error) {
	existingSnapshot.CurrentSpec = *stackSpec
	return typeurl.MarshalAny(existingSnapshot)
}

// MarshalSnapshotStackSnapshot takes an existing interfaces.SnapshotStack and
// an interfaces.SnapshotStack object and replaces the existing snapshot data
// with that contained in the second interfaces.SnapshotStack
// and marshals it into a protocol buffer Any message.  The types.StackSpec is
// left unchanged
// Under the hood, this relies on marshaling the objects to JSON.
func MarshalSnapshotStackSnapshot(existingSnapshot *interfaces.SnapshotStack, snapshot *interfaces.SnapshotStack) (*gogotypes.Any, error) {
	// No accidental or sly changes to the StackSpec are permitted
	existingSnapshot.Services = snapshot.Services
	existingSnapshot.Configs = snapshot.Configs
	existingSnapshot.Secrets = snapshot.Secrets
	existingSnapshot.Networks = snapshot.Networks

	return typeurl.MarshalAny(existingSnapshot)
}

// MarshalStackSpec takes a types.StackSpec, wraps it in a SnapshotStack
// object and marshals it into a protocol buffer Any message.
// Under the hood, this relies on marshaling the objects to JSON.
func MarshalStackSpec(stackSpec *types.StackSpec) (*gogotypes.Any, error) {
	snapshotStack := interfaces.SnapshotStack{
		CurrentSpec: *stackSpec,
	}
	return typeurl.MarshalAny(&snapshotStack)

}

// UnmarshalStackSpec does the MarshalStackSpec operation in reverse --
// takes a proto message, and returns the StackSpec contained in it.
func UnmarshalStackSpec(payload *gogotypes.Any) (*types.StackSpec, error) {
	snapshotStack, err := UnmarshalSnapshotStack(payload)
	if err != nil {
		return nil, err
	}

	return &snapshotStack.CurrentSpec, nil
}

// UnmarshalSnapshotStack does the MarshalStackSpec operation in reverse --
// takes a proto message, and returns the SnapshotStack contained in it.
func UnmarshalSnapshotStack(payload *gogotypes.Any) (*interfaces.SnapshotStack, error) {
	iface, err := typeurl.UnmarshalAny(payload)
	if err != nil {
		return nil, err
	}
	// this is a naked cast, which means if for some reason this _isn't_ a
	// StackSpec object, the program will panic. This is fine, because if
	// such a thing were to occur, it would be panic-worthy.
	snapshotStack := iface.(*interfaces.SnapshotStack)

	return snapshotStack, nil
}

// ConstructStack takes a Swarmkit Resource object, calls UnmarshalStackSpec and
// constructs a fresh types.Stack and populates its fields (Meta, Version, and ID)
// contained in the Resource
func ConstructStack(resource *api.Resource) (*types.Stack, error) {

	// now, we have to get the stack out of the resource object
	stackSpec, err := UnmarshalStackSpec(resource.Payload)
	if err != nil {
		return &types.Stack{}, err
	}
	if stackSpec == nil {
		return &types.Stack{}, errors.New("got back an empty stack")
	}

	// extract the times from the swarmkit resource message.
	createdAt, err := gogotypes.TimestampFromProto(resource.Meta.CreatedAt)
	if err != nil {
		return &types.Stack{}, errors.Wrap(err, "error converting swarmkit timestamp")
	}
	updatedAt, err := gogotypes.TimestampFromProto(resource.Meta.UpdatedAt)
	if err != nil {
		return &types.Stack{}, errors.Wrap(err, "error converting swarmkit timestamp")
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
	return &stack, nil
}

// ConstructSnapshotStack takes a Swarmkit Resource object, calls UnmarshalSnapshotStack and
// constructs a fresh types.Stack and populates its fields (Meta, Version, and ID)
// contained in the Resource
func ConstructSnapshotStack(resource *api.Resource) (*interfaces.SnapshotStack, error) {

	// now, we have to get the stack out of the resource object
	snapshotStack, err := UnmarshalSnapshotStack(resource.Payload)
	if err != nil {
		return &interfaces.SnapshotStack{}, err
	}
	if snapshotStack == nil {
		return &interfaces.SnapshotStack{}, errors.New("got back an empty SnapshotStack")
	}

	// extract the times from the swarmkit resource message.
	createdAt, err := gogotypes.TimestampFromProto(resource.Meta.CreatedAt)
	if err != nil {
		return &interfaces.SnapshotStack{}, errors.Wrap(err, "error converting swarmkit timestamp")
	}
	updatedAt, err := gogotypes.TimestampFromProto(resource.Meta.UpdatedAt)
	if err != nil {
		return &interfaces.SnapshotStack{}, errors.Wrap(err, "error converting swarmkit timestamp")
	}

	snapshotStack.ID = resource.ID
	snapshotStack.Meta.Version.Index = resource.Meta.Version.Index
	snapshotStack.Meta.CreatedAt = createdAt
	snapshotStack.Meta.UpdatedAt = updatedAt

	return snapshotStack, nil
}
