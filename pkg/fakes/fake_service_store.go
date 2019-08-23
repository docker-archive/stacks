package fakes

import (
	"fmt"
	"sort"
	"sync"

	"github.com/containerd/typeurl"
	gogotypes "github.com/gogo/protobuf/types"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/types"
)

// FakeServiceStore contains the subset of Backend APIs for swarm.Service
type FakeServiceStore struct {
	mu          sync.Mutex
	curID       int
	labelErrors map[string]error
	keyPrefix   string

	services       map[string]*swarm.Service
	servicesByName map[string]string
}

func init() {
	typeurl.Register(&swarm.ServiceSpec{}, "github.com/docker/swarm/ServiceSpec")
}

// CopyServiceSpec duplicates the ServiceSpec
func CopyServiceSpec(spec swarm.ServiceSpec) (swarm.ServiceSpec, error) {
	var payload *gogotypes.Any
	var err error
	payload, err = typeurl.MarshalAny(&spec)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}
	iface, err := typeurl.UnmarshalAny(payload)
	if err != nil {
		return swarm.ServiceSpec{}, err
	}
	return *iface.(*swarm.ServiceSpec), nil
}

// NewFakeServiceStore creates a new FakeServiceStore
func NewFakeServiceStore() *FakeServiceStore {
	return &FakeServiceStore{
		// Don't start from ID 0, to catch any uninitialized types.
		curID:          1,
		services:       map[string]*swarm.Service{},
		servicesByName: map[string]string{},
		labelErrors:    map[string]error{},
	}
}

// resolveID takes a value that might be an ID or and figures out which it is,
// returning the ID
func (f *FakeServiceStore) resolveID(key string) string {
	id, ok := f.servicesByName[key]
	if !ok {
		return key
	}
	return id
}

func (f *FakeServiceStore) newID(objType string) string {
	index := f.curID
	f.curID++
	return fmt.Sprintf("id_%s_%v", objType, index)
}

// GetServices implements the GetServices method of the BackendClient,
// returning a list of services. It only supports 1 kind of filter, which is
// a filter for stack ID.
func (f *FakeServiceStore) GetServices(opts dockerTypes.ServiceListOptions) ([]swarm.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var (
		stackID   string
		hasFilter bool
	)
	// before doing anything, check if there is a filter and it's in the
	// correct form. This lets us error out early if it's not
	if opts.Filters.Len() != 0 {
		var ok bool
		stackID, ok = FakeGetStackIDFromLabelFilter(opts.Filters)
		if !ok {
			return nil, FakeInvalidArg
		}
		hasFilter = true
	}

	result := []swarm.Service{}

	for _, key := range f.SortedIDs() {
		// if we're filtering on stack ID, and this service doesn't match, then
		// we should skip this service
		service := f.services[key]
		if hasFilter && service.Spec.Annotations.Labels[types.StackLabel] != stackID {
			continue
		}
		// otherwise, we should append this service to the set
		if err := f.causeAnError(nil, "GetServices", service.Spec); err != nil {
			return nil, err
		}
		result = append(result, *service)
	}

	return result, nil
}

// GetService gets a swarm service
func (f *FakeServiceStore) GetService(idOrName string, _ bool) (swarm.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	service, ok := f.services[id]
	if !ok {
		return swarm.Service{}, errdefs.NotFound(fmt.Errorf("config %s not foun", id))
	}

	if err := f.causeAnError(nil, "GetService", service.Spec); err != nil {
		return swarm.Service{}, FakeUnavailable
	}
	return *service, nil
}

// CreateService creates a swarm service.
func (f *FakeServiceStore) CreateService(spec swarm.ServiceSpec, _ string, _ bool) (*dockerTypes.ServiceCreateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.causeAnError(nil, "CreateService", spec); err != nil {
		return nil, FakeInvalidArg
	}

	if _, ok := f.servicesByName[spec.Annotations.Name]; ok {
		return nil, FakeInvalidArg
	}
	copied, err := CopyServiceSpec(spec)
	if err != nil {
		return nil, err
	}

	// otherwise, create a service object
	service := &swarm.Service{
		ID: f.newID("service"),
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: uint64(1),
			},
		},
		Spec: copied,
	}

	f.servicesByName[spec.Annotations.Name] = service.ID
	f.services[service.ID] = service

	return &dockerTypes.ServiceCreateResponse{
		ID: service.ID,
	}, nil
}

// UpdateService updates the service to the provided spec.
func (f *FakeServiceStore) UpdateService(
	idOrName string,
	version uint64,
	spec swarm.ServiceSpec,
	_ dockerTypes.ServiceUpdateOptions,
	_ bool,
) (*dockerTypes.ServiceUpdateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)
	service, ok := f.services[id]
	if !ok {
		return nil, FakeNotFound
	}

	if version != service.Meta.Version.Index {
		return nil, FakeInvalidArg
	}

	copied, err := CopyServiceSpec(spec)
	service.Spec = copied
	service.Meta.Version.Index = service.Meta.Version.Index + 1
	return &dockerTypes.ServiceUpdateResponse{}, err
}

// RemoveService deletes the service
func (f *FakeServiceStore) RemoveService(idOrName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	service, ok := f.services[id]
	if !ok {
		return FakeNotFound
	}

	if err := f.causeAnError(nil, "RemoveService", service.Spec); err != nil {
		return err
	}

	f.InternalDeleteService(id)

	return nil
}

func (f *FakeServiceStore) causeAnError(err error, operation string, spec swarm.ServiceSpec) error {
	if err != nil {
		return err
	}

	key := f.constructErrorMark(operation)
	errorName, ok := spec.Annotations.Labels[key]
	if !ok {
		key := f.constructErrorMark("")
		errorName, ok = spec.Annotations.Labels[key]
		if !ok {
			return nil
		}
	}

	return f.labelErrors[errorName]
}

// SpecifyError associates an error to a key
func (f *FakeServiceStore) SpecifyError(errorKey string, err error) {
	f.labelErrors[errorKey] = err
}

// SpecifyKeyPrefix provides prefix to generated ID's
func (f *FakeServiceStore) SpecifyKeyPrefix(keyPrefix string) {
	f.keyPrefix = keyPrefix
}

func (f *FakeServiceStore) constructErrorMark(operation string) string {
	if len(operation) == 0 {
		return f.keyPrefix + ".serviceError"
	}
	return f.keyPrefix + "." + operation + ".serviceError"
}

// MarkInputForError mark ServiceSpec with potential errors
func (f *FakeServiceStore) MarkInputForError(errorKey string, input interface{}, ops ...string) {

	spec := input.(*swarm.ServiceSpec)
	if spec.Annotations.Labels == nil {
		spec.Annotations.Labels = make(map[string]string)
	}
	if len(ops) == 0 {
		key := f.constructErrorMark("")
		spec.Annotations.Labels[key] = errorKey
	} else {
		for _, operation := range ops {
			key := f.constructErrorMark(operation)
			spec.Annotations.Labels[key] = errorKey
		}
	}
}

// InternalAddService adds swarm.Service to storage without preconditions
func (f *FakeServiceStore) InternalAddService(id string, service *swarm.Service) {
	f.services[id] = service
	f.servicesByName[service.Spec.Annotations.Name] = id
}

// InternalGetService retrieves swarm.Service or nil from storage without preconditions
func (f *FakeServiceStore) InternalGetService(id string) *swarm.Service {
	service, ok := f.services[id]
	if !ok {
		return nil
	}
	return service
}

// InternalQueryServices retrieves all swarm.Service from storage while applying a transform
func (f *FakeServiceStore) InternalQueryServices(transform func(service *swarm.Service) interface{}) []interface{} {
	result := make([]interface{}, 0)

	for _, key := range f.SortedIDs() {
		item := f.InternalGetService(key)
		if transform == nil {
			result = append(result, item)
		} else {
			view := transform(item)
			if view != nil {
				result = append(result, view)
			}
		}
	}
	return result
}

// InternalDeleteService removes swarm.Service from storage without preconditions
func (f *FakeServiceStore) InternalDeleteService(id string) *swarm.Service {
	service, ok := f.services[id]
	if !ok {
		return nil
	}
	delete(f.services, id)
	delete(f.servicesByName, service.Spec.Annotations.Name)
	return service
}

// SortedIDs returns sorted Service IDs
func (f *FakeServiceStore) SortedIDs() []string {
	result := []string{}
	for key, value := range f.services {
		if value != nil {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}
