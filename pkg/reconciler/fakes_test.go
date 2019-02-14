package reconciler

// this file contains fakes used to test the reconciler

import (
	"errors"
	"fmt"
	"sync"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
)

// fakeReconcilerClient is a fake implementing the ReconcilerClient interface,
// which is used to test the reconciler. fakeReconcilerClient only implements
// the strict subset of features needed to make the reconciler go. Most
// notably, it has a half-ass implementation of Filters that only works for
// stack ID labels.
type fakeReconcilerClient struct {
	mu sync.Mutex

	// variable for making IDs. increment this every time we make a new ID.
	// easier to do this than to import github.com/docker/swarmkit/identity
	totallyRandomIDBase int

	// maps id -> stack
	stacks map[string]*interfaces.SwarmStack
	// maps name -> id
	stacksByName map[string]string

	services       map[string]*swarm.Service
	servicesByName map[string]string
}

// error definitions to reuse
var (
	notFound    = errdefs.NotFound(errors.New("not found"))
	invalidArg  = errdefs.InvalidParameter(errors.New("not valid"))
	unavailable = errdefs.Unavailable(errors.New("not available"))
)

func newFakeReconcilerClient() *fakeReconcilerClient {
	return &fakeReconcilerClient{
		stacks:         map[string]*interfaces.SwarmStack{},
		stacksByName:   map[string]string{},
		services:       map[string]*swarm.Service{},
		servicesByName: map[string]string{},
	}
}

// GetSwarmStack gets a SwarmStack
func (f *fakeReconcilerClient) GetSwarmStack(idOrName string) (interfaces.SwarmStack, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := resolveID(f.stacksByName, idOrName)

	stack, ok := f.stacks[id]
	if !ok {
		return interfaces.SwarmStack{}, notFound
	}

	// if you add the "makemefail" label to a stack, attempting to get it will
	// fail
	if _, ok := stack.Spec.Annotations.Labels["makemefail"]; ok {
		return interfaces.SwarmStack{}, unavailable
	}
	return *stack, nil
}

// GetService gets a swarm service
func (f *fakeReconcilerClient) GetService(idOrName string, _ bool) (swarm.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := resolveID(f.servicesByName, idOrName)

	service, ok := f.services[id]
	if !ok {
		return swarm.Service{}, notFound
	}

	if _, ok := service.Spec.Annotations.Labels["makemefail"]; ok {
		return swarm.Service{}, unavailable
	}
	return *service, nil
}

// CreateService creates a swarm service. Including the label "makemefail" in
// the spec will cause creation to fail.
func (f *fakeReconcilerClient) CreateService(spec swarm.ServiceSpec, _ string, _ bool) (*dockerTypes.ServiceCreateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := spec.Annotations.Labels["makemefail"]; ok {
		return nil, invalidArg
	}

	if _, ok := f.servicesByName[spec.Annotations.Name]; ok {
		return nil, invalidArg
	}

	// otherwise, create a service object
	service := &swarm.Service{
		ID: f.newID("service"),
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: uint64(1),
			},
		},
		Spec: spec,
	}

	f.servicesByName[spec.Annotations.Name] = service.ID
	f.services[service.ID] = service

	return &dockerTypes.ServiceCreateResponse{
		ID: service.ID,
	}, nil
}

// resolveID takes a value that might be an ID or and figures out which it is,
// returning the ID
func resolveID(namesToIds map[string]string, key string) string {
	id, ok := namesToIds[key]
	if !ok {
		return key
	}
	return id
}

func (f *fakeReconcilerClient) newID(objType string) string {
	index := f.totallyRandomIDBase
	f.totallyRandomIDBase++
	return fmt.Sprintf("id_%s_%v", objType, index)
}
