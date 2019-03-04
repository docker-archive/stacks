package store

import (
	"context"

	swarmapi "github.com/docker/swarmkit/api"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

// StackResourceKind defines the Kind of swarmkit Resources belonging to
// stacks.
const StackResourceKind = "github.com/docker/stacks/Stack"

// ResourcesClient is a subset of swarmkit's ControlClient interface for operating on
// Resources and Extensions
type ResourcesClient interface {
	CreateExtension(ctx context.Context, in *swarmapi.CreateExtensionRequest, opts ...grpc.CallOption) (*swarmapi.CreateExtensionResponse, error)
	GetResource(ctx context.Context, in *swarmapi.GetResourceRequest, opts ...grpc.CallOption) (*swarmapi.GetResourceResponse, error)
	UpdateResource(ctx context.Context, in *swarmapi.UpdateResourceRequest, opts ...grpc.CallOption) (*swarmapi.UpdateResourceResponse, error)
	ListResources(ctx context.Context, in *swarmapi.ListResourcesRequest, opts ...grpc.CallOption) (*swarmapi.ListResourcesResponse, error)
	CreateResource(ctx context.Context, in *swarmapi.CreateResourceRequest, opts ...grpc.CallOption) (*swarmapi.CreateResourceResponse, error)
	RemoveResource(ctx context.Context, in *swarmapi.RemoveResourceRequest, opts ...grpc.CallOption) (*swarmapi.RemoveResourceResponse, error)
}

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
func (s *StackStore) AddStack(st types.Stack, sst interfaces.SwarmStack) (string, error) {
	// first, marshal the stacks to a proto message
	any, err := MarshalStacks(&st, &sst)
	if err != nil {
		return "", err
	}

	// reuse the Annotations from the SwarmStack. However, since they're
	// actually different types, convert them
	annotations := &swarmapi.Annotations{
		Name:   sst.Spec.Annotations.Name,
		Labels: sst.Spec.Annotations.Labels,
	}

	// create a resource creation request
	req := &swarmapi.CreateResourceRequest{
		Annotations: annotations,
		Kind:        StackResourceKind,
		Payload:     any,
	}

	// now create the resource object
	resp, err := s.client.CreateResource(context.TODO(), req)
	if err != nil {
		return "", err
	}
	return resp.Resource.ID, nil
}

// UpdateStack updates an existing Stack object
func (s *StackStore) UpdateStack(id string, st types.StackSpec, sst interfaces.SwarmStackSpec, version uint64) error {
	// get the swarmkit resource
	resp, err := s.client.GetResource(context.TODO(), &swarmapi.GetResourceRequest{
		ResourceID: id,
	})
	if err != nil {
		return err
	}

	resource := resp.Resource
	// unmarshal the contents
	stack, swarmStack, err := UnmarshalStacks(resource)
	if err != nil {
		return err
	}

	// update the specs
	stack.Spec = st
	swarmStack.Spec = sst

	// marshal it all back
	any, err := MarshalStacks(stack, swarmStack)
	if err != nil {
		return err
	}

	// and then issue an update.
	_, err = s.client.UpdateResource(context.TODO(),
		&swarmapi.UpdateResourceRequest{
			ResourceID:      id,
			ResourceVersion: &swarmapi.Version{Index: version},
			// we don't need to set the value of Annotations. leaving it empty
			// indicates no change
			Payload: any,
		},
	)
	return err
}

// DeleteStack removes the stacks with the given ID.
func (s *StackStore) DeleteStack(id string) error {
	// this one is easy, no type conversion needed
	_, err := s.client.RemoveResource(
		context.TODO(), &swarmapi.RemoveResourceRequest{ResourceID: id},
	)
	return err
}

// GetStack retrieves and returns an existing types.Stack object by ID
func (s *StackStore) GetStack(id string) (types.Stack, error) {
	resp, err := s.client.GetResource(
		context.TODO(), &swarmapi.GetResourceRequest{ResourceID: id},
	)
	if err != nil {
		return types.Stack{}, err
	}
	resource := resp.Resource

	// now, we have to get the stack out of the resource object
	stack, _, err := UnmarshalStacks(resource)
	if err != nil {
		return types.Stack{}, err
	}
	if stack == nil {
		return types.Stack{}, errors.New("got back an empty stack")
	}

	// and then return the stack
	return *stack, nil
}

// GetSwarmStack retrieves and returns an exist types.SwarmStack object by ID.
func (s *StackStore) GetSwarmStack(id string) (interfaces.SwarmStack, error) {
	resp, err := s.client.GetResource(
		context.TODO(), &swarmapi.GetResourceRequest{ResourceID: id},
	)
	if err != nil {
		return interfaces.SwarmStack{}, err
	}
	resource := resp.Resource
	_, swarmStack, err := UnmarshalStacks(resource)
	if err != nil {
		return interfaces.SwarmStack{}, err
	}
	if swarmStack == nil {
		return interfaces.SwarmStack{}, errors.New("got back an empty stack")
	}
	return *swarmStack, nil
}

// ListStacks lists all available stack objects
func (s *StackStore) ListStacks() ([]types.Stack, error) {
	resp, err := s.client.ListResources(context.TODO(),
		&swarmapi.ListResourcesRequest{
			Filters: &swarmapi.ListResourcesRequest_Filters{
				// list only stacks
				Kind: StackResourceKind,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	// unmarshal and pack up all of the stack objects
	stacks := make([]types.Stack, 0, len(resp.Resources))
	for _, resource := range resp.Resources {
		stack, _, err := UnmarshalStacks(resource)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, *stack)
	}
	return stacks, nil
}

// ListSwarmStacks lists all available stack objects as SwarmStacks
func (s *StackStore) ListSwarmStacks() ([]interfaces.SwarmStack, error) {
	resp, err := s.client.ListResources(context.TODO(),
		&swarmapi.ListResourcesRequest{
			Filters: &swarmapi.ListResourcesRequest_Filters{
				// list only stacks
				Kind: StackResourceKind,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	stacks := make([]interfaces.SwarmStack, 0, len(resp.Resources))
	for _, resource := range resp.Resources {
		_, stack, err := UnmarshalStacks(resource)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, *stack)
	}
	return stacks, nil
}
