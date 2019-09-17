package fakes

import (
	"fmt"
	"sort"
	"sync"

	"github.com/containerd/typeurl"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/types"
)

/*
 *   fake_service_store.go implementation is a customized-but-duplicate of
 *   fake_secret_store.go, fake_config_store.go, fake_network_store.go and
 *   fake_stack_store.go.
 *
 *   fake_service_store.go represents the interfaces.SwarmServiceBackend portions
 *   of the interfaces.BackendClient.
 *
 *   reconciler.fakeReconcilerClient exposes extra API to direct control
 *   of the internals of the implementation for testing.
 *
 *   SortedIDs() []string
 *   InternalDeleteService(id string) *swarm.Service
 *   InternalQueryServices(transform func(*swarm.Service) interface{}) []interface
 *   InternalGetService(id string) *swarm.Service
 *   InternalAddService(id string, service *swarm.Service)
 *   MarkServiceSpecForError(errorKey string, *swarm.ServiceSpec, ops ...string)
 *   SpecifyKeyPrefix(keyPrefix string)
 *   SpecifyErrorTrigger(errorKey string, err error)
 */

// FakeServiceStore contains the subset of Backend APIs SwarmServiceBackend
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
	typeurl.Register(&swarm.Service{}, "github.com/docker/swarm/Service")
}

// CopyServiceSpec duplicates the ServiceSpec
func CopyServiceSpec(spec swarm.ServiceSpec) *swarm.ServiceSpec {
	payload, _ := typeurl.MarshalAny(&spec)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*swarm.ServiceSpec)
}

// CopyService duplicates the ServiceSpec
func CopyService(spec swarm.Service) *swarm.Service {
	payload, _ := typeurl.MarshalAny(&spec)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*swarm.Service)
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

func (f *FakeServiceStore) newID() string {
	index := f.curID
	f.curID++
	if len(f.keyPrefix) == 0 {
		return fmt.Sprintf("SVC_%v", index)
	}
	return fmt.Sprintf("%s_SVC_%v", f.keyPrefix, index)
}

// GetServices implements the GetServices method of the SwarmServiceBackend,
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
		service := f.services[key]

		// if we're filtering on stack ID, and this service doesn't
		// match, then we should skip this service
		if hasFilter && service.Spec.Annotations.Labels[types.StackLabel] != stackID {
			continue
		}
		// otherwise, we should append this service to the set
		if err := f.maybeTriggerAnError("GetServices", service.Spec); err != nil {
			return nil, err
		}
		result = append(result, *CopyService(*service))
	}

	return result, nil
}

// GetService gets a swarm service
func (f *FakeServiceStore) GetService(idOrName string, _ bool) (swarm.Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	service := f.InternalGetService(id)
	if service == nil {
		return swarm.Service{}, errdefs.NotFound(fmt.Errorf("service %s not foun", id))
	}

	if err := f.maybeTriggerAnError("GetService", service.Spec); err != nil {
		return swarm.Service{}, err
	}
	return *CopyService(*service), nil
}

// CreateService creates a swarm service.
func (f *FakeServiceStore) CreateService(spec swarm.ServiceSpec, _ string, _ bool) (*dockerTypes.ServiceCreateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.maybeTriggerAnError("CreateService", spec); err != nil {
		return nil, err
	}

	if _, ok := f.servicesByName[spec.Annotations.Name]; ok {
		return nil, errdefs.AlreadyExists(fmt.Errorf("service %s already used", spec.Annotations.Name))
	}

	copied := CopyServiceSpec(spec)

	// otherwise, create a service object
	service := &swarm.Service{
		ID: f.newID(),
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: uint64(1),
			},
		},
		Spec: *copied,
	}

	f.InternalAddService(service.ID, service)

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
		return nil, errdefs.NotFound(fmt.Errorf("service %s not found", id))
	}

	if version != service.Meta.Version.Index {
		return nil, FakeInvalidArg
	}

	if err := f.maybeTriggerAnError("UpdateService", service.Spec); err != nil {
		return &dockerTypes.ServiceUpdateResponse{}, err
	}

	if err := f.maybeTriggerAnError("UpdateService", spec); err != nil {
		return &dockerTypes.ServiceUpdateResponse{}, err
	}

	copied := CopyServiceSpec(spec)
	service.Spec = *copied
	service.Meta.Version.Index = service.Meta.Version.Index + 1
	return &dockerTypes.ServiceUpdateResponse{}, nil
}

// RemoveService deletes the service
func (f *FakeServiceStore) RemoveService(idOrName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	service := f.InternalGetService(id)
	if service == nil {
		return errdefs.NotFound(fmt.Errorf("service %s not found", id))
	}

	if err := f.maybeTriggerAnError("RemoveService", service.Spec); err != nil {
		return err
	}

	f.InternalDeleteService(id)

	return nil
}

// utility function for interfaces.SwarmServiceBackend calls to trigger an error
func (f *FakeServiceStore) maybeTriggerAnError(operation string, spec swarm.ServiceSpec) error {
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

// SpecifyErrorTrigger associates an error to an errorKey so that when calls interfaces.SwarmServiceBackend find a marked swarm.ServiceSpec an error is returned
func (f *FakeServiceStore) SpecifyErrorTrigger(errorKey string, err error) {
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

// MarkServiceSpecForError marks a swarm.ServiceSpec to trigger an error when calls from interfaces.SwarmServiceBackend are configured for the errorKey.
// - All interfaces.SwarmServiceBackend calls may be triggered if len(ops)==0
// - Otherwise, ops may be any of the following: GetServices, GetService, CreateService, UpdateService, RemoveService
func (f *FakeServiceStore) MarkServiceSpecForError(errorKey string, spec *swarm.ServiceSpec, ops ...string) {

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
