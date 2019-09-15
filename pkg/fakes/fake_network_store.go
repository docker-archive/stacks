package fakes

import (
	"fmt"
	"sort"
	"sync"

	"github.com/containerd/typeurl"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/types"
)

/*
 *   fake_network_store.go implementation is a customized-but-duplicate of
 *   fake_service_store.go, fake_config_store.go, fake_secret_store.go and
 *   fake_stack_store.go.
 *
 *   NOTE: NETWORKs do not have an UPDATE operation and therefore there are no
 *   relevant version numbers.
 *
 *   NOTE: NETWORKs are unlike Service, Secret, Stack, Config because there is
 *   no "NetworkSpec" object passed and stored in a created "Network".  The
 *   API is mildly different and the simple fix for test features is to create
 *   MarkNetworkCreateForError and MarkNetworkResourceForError for pre-create
 *   and post-create test changes.
 *
 *   fake_network_store.go represents the interfaces.SwarmNetworkBackend
 *   portions of the interfaces.BackendClient.
 *
 *   reconciler.fakeReconcilerClient exposes extra API to direct control
 *   of the internals of the implementation for testing.
 *
 *   SortedIDs() []string
 *   InternalDeleteNetwork(id string) *dockerTypes.NetworkResource
 *   InternalQueryNetworks(transform func(*dockerTypes.NetworkResource) interface{}) []interface
 *   InternalGetNetwork(id string) *dockerTypes.NetworkResource
 *   InternalAddNetwork(id string, secret *dockerTypes.NetworkResource)
 *   MarkNetworkCreateForError(errorKey string, *dockerTypes.NetworkCreate, ops ...string)
 *   MarkNetworkResourceForError(errorKey string, *dockerTypes.NetworkResource, ops ...string)
 *   SpecifyKeyPrefix(keyPrefix string)
 *   SpecifyErrorTrigger(errorKey string, err error)
 *   TransformNetworkCreateRequest(request dockerTypes.NetworkCreateRequest) *dockerTypes.NetworkResource
 */

// FakeNetworkStore contains the subset of Backend APIs for dockerTypes.NetworkResource
type FakeNetworkStore struct {
	mu          sync.Mutex
	curID       int
	labelErrors map[string]error
	keyPrefix   string

	networks       map[string]*dockerTypes.NetworkResource
	networksByName map[string]string
}

// These type registrations are for TESTING in order to create deep copies
func init() {
	typeurl.Register(&dockerTypes.NetworkCreate{}, "github.com/docker/api/NetworkCreate")
	typeurl.Register(&dockerTypes.NetworkResource{}, "github.com/docker/api/NetworkResource")
}

// CopyNetworkCreate duplicates the dockerTypes.NetworkCreate
func CopyNetworkCreate(spec dockerTypes.NetworkCreate) *dockerTypes.NetworkCreate {
	payload, _ := typeurl.MarshalAny(&spec)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*dockerTypes.NetworkCreate)
}

// CopyNetworkResource duplicates the dockerTypes.NetworkResource
func CopyNetworkResource(spec dockerTypes.NetworkResource) *dockerTypes.NetworkResource {
	payload, _ := typeurl.MarshalAny(&spec)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*dockerTypes.NetworkResource)
}

// NewFakeNetworkStore creates a new FakeNetworkStore
func NewFakeNetworkStore() *FakeNetworkStore {
	return &FakeNetworkStore{
		// Don't start from ID 0, to catch any uninitialized types.
		curID:          1,
		networks:       map[string]*dockerTypes.NetworkResource{},
		networksByName: map[string]string{},
		labelErrors:    map[string]error{},
	}
}

// resolveID takes a value that might be an ID or and figures out which it is,
// returning the ID
func (f *FakeNetworkStore) resolveID(key string) string {
	id, ok := f.networksByName[key]
	if !ok {
		return key
	}
	return id
}

func (f *FakeNetworkStore) newID() string {
	index := f.curID
	f.curID++
	if len(f.keyPrefix) == 0 {
		return fmt.Sprintf("NET_%v", index)
	}
	return fmt.Sprintf("%s_NET_%v", f.keyPrefix, index)
}

// GetNetworks implements the GetNetworks method of the SwarmNetworkBackend,
// returning a list of networks. It only supports 1 kind of filter, which is
// a filter for stack ID.
func (f *FakeNetworkStore) GetNetworks(plainFilters filters.Args) ([]dockerTypes.NetworkResource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var (
		stackID   string
		hasFilter bool
	)
	// before doing anything, check if there is a filter and it's in the
	// correct form. This lets us error out early if it's not
	if plainFilters.Len() != 0 {
		var ok bool
		stackID, ok = FakeGetStackIDFromLabelFilter(plainFilters)
		if !ok {
			return nil, FakeInvalidArg
		}
		hasFilter = true
	}

	networks := []dockerTypes.NetworkResource{}

	for _, key := range f.SortedIDs() {
		network := f.networks[key]

		// if we're filtering on stack ID, and this network doesn't
		// match, then we should skip this network
		if hasFilter && network.Labels[types.StackLabel] != stackID {
			continue
		}
		// otherwise, we should append this network to the set
		if err := f.maybeTriggerAnError("GetNetworks", *network); err != nil {
			return nil, err
		}
		networks = append(networks, *CopyNetworkResource(*network))
	}

	return networks, nil
}

// GetNetworksByName implements the GetNetworks method of the
// SwarmNetworkBackend, returning a list of networks by name
func (f *FakeNetworkStore) GetNetworksByName(name string) ([]dockerTypes.NetworkResource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	networks := []dockerTypes.NetworkResource{}

	for _, network := range f.networks {
		// if we're filtering on stack ID, and this network doesn't match, then
		// we should skip this network
		if network.Name != name {
			continue
		}
		// otherwise, we should append this network to the set
		if err := f.maybeTriggerAnError("GetNetworksByName", *network); err != nil {
			return nil, err
		}
		networks = append(networks, *CopyNetworkResource(*network))
	}

	return networks, nil
}

// GetNetwork gets a swarm network
func (f *FakeNetworkStore) GetNetwork(idOrName string) (dockerTypes.NetworkResource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	network, ok := f.networks[id]
	if !ok {
		return dockerTypes.NetworkResource{}, errdefs.NotFound(fmt.Errorf("network %s not found", id))
	}

	if err := f.maybeTriggerAnError("GetNetwork", *network); err != nil {
		return dockerTypes.NetworkResource{}, err
	}
	return *CopyNetworkResource(*network), nil
}

// TransformNetworkCreateRequest utility function populates NetworkResource from a NetworkCreateRequest
func (f *FakeNetworkStore) TransformNetworkCreateRequest(request dockerTypes.NetworkCreateRequest) *dockerTypes.NetworkResource {

	copied := CopyNetworkCreate(request.NetworkCreate)

	ipam := network.IPAM{}
	if copied.IPAM != nil {
		ipam = *copied.IPAM
	}
	configFrom := network.ConfigReference{}
	if copied.ConfigFrom != nil {
		configFrom = *copied.ConfigFrom
	}

	// otherwise, create a network object
	network := &dockerTypes.NetworkResource{
		Name:       request.Name,
		Driver:     copied.Driver,
		Scope:      copied.Scope,
		EnableIPv6: copied.EnableIPv6,
		IPAM:       ipam,
		Internal:   copied.Internal,
		Attachable: copied.Attachable,
		Ingress:    copied.Ingress,
		ConfigOnly: copied.ConfigOnly,
		ConfigFrom: configFrom,
		Options:    copied.Options,
		Labels:     copied.Labels,
	}

	return network
}

// CreateNetwork creates a swarm network.
func (f *FakeNetworkStore) CreateNetwork(request dockerTypes.NetworkCreateRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// maybeTriggerAnError happens below

	if _, ok := f.networksByName[request.Name]; ok {
		return "", errdefs.AlreadyExists(fmt.Errorf("network %s already used", request.Name))
	}

	network := f.TransformNetworkCreateRequest(request)
	network.ID = f.newID()

	if err := f.maybeTriggerAnError("CreateNetwork", *network); err != nil {
		return "", err
	}

	f.InternalAddNetwork(network.ID, network)

	return network.ID, nil
}

// RemoveNetwork deletes the network
func (f *FakeNetworkStore) RemoveNetwork(idOrName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	network := f.InternalGetNetwork(id)
	if network == nil {
		return errdefs.NotFound(fmt.Errorf("network %s not found", id))
	}

	if err := f.maybeTriggerAnError("RemoveNetwork", *network); err != nil {
		return err
	}

	f.InternalDeleteNetwork(id)

	return nil
}

// utility function for interfaces.SwarmNetworkBackend calls to trigger an error
func (f *FakeNetworkStore) maybeTriggerAnError(operation string, spec dockerTypes.NetworkResource) error {
	key := f.constructErrorMark(operation)
	errorName, ok := spec.Labels[key]
	if !ok {
		key := f.constructErrorMark("")
		errorName, ok = spec.Labels[key]
		if !ok {
			return nil
		}
	}

	return f.labelErrors[errorName]
}

// SpecifyErrorTrigger associates an error to an errorKey so that when calls interfaces.SwarmSecretBackend find a marked dockerTypes.NetworkResource an error is returned
func (f *FakeNetworkStore) SpecifyErrorTrigger(errorKey string, err error) {
	f.labelErrors[errorKey] = err
}

// SpecifyKeyPrefix provides prefix to generated ID's
func (f *FakeNetworkStore) SpecifyKeyPrefix(keyPrefix string) {
	f.keyPrefix = keyPrefix
}

func (f *FakeNetworkStore) constructErrorMark(operation string) string {
	if len(operation) == 0 {
		return f.keyPrefix + ".networkError"
	}
	return f.keyPrefix + "." + operation + ".networkError"
}

// MarkNetworkCreateForError marks a dockerTypes.NetworkCreate to trigger an error when calls from interfaces.SwarmNetworkBackend are configured for the errorKey.
// - All interfaces.SwarmNetworkBackend calls may be triggered if len(ops)==0
// - Otherwise, ops may be any of the following: GetNetworks, GetNetworksByName, GetNetwork, CreateNetwork, RemoveNetwork
func (f *FakeNetworkStore) MarkNetworkCreateForError(errorKey string, spec *dockerTypes.NetworkCreate, ops ...string) {

	if spec.Labels == nil {
		spec.Labels = make(map[string]string)
	}
	if len(ops) == 0 {
		key := f.constructErrorMark("")
		spec.Labels[key] = errorKey
	} else {
		for _, operation := range ops {
			key := f.constructErrorMark(operation)
			spec.Labels[key] = errorKey
		}
	}
}

// MarkNetworkResourceForError see MarkNetworkCreateForError, NetworkResource objects do not contain their creation object like a Spec.  It is simpler to have a second version of the Mark function
func (f *FakeNetworkStore) MarkNetworkResourceForError(errorKey string, network *dockerTypes.NetworkResource, ops ...string) {

	if network.Labels == nil {
		network.Labels = make(map[string]string)
	}
	if len(ops) == 0 {
		key := f.constructErrorMark("")
		network.Labels[key] = errorKey
	} else {
		for _, operation := range ops {
			key := f.constructErrorMark(operation)
			network.Labels[key] = errorKey
		}
	}
}

// InternalAddNetwork adds dockerTypes.NetworkResource to storage without preconditions
func (f *FakeNetworkStore) InternalAddNetwork(id string, network *dockerTypes.NetworkResource) {
	f.networks[id] = network
	f.networksByName[network.Name] = id
}

// InternalGetNetwork retrieves dockerTypes.NetworkResource or nil from storage without preconditions
func (f *FakeNetworkStore) InternalGetNetwork(id string) *dockerTypes.NetworkResource {
	network, ok := f.networks[id]
	if !ok {
		return nil
	}
	return network
}

// InternalQueryNetworks retrieves all dockerTypes.NetworkResource from storage while applying a transform
func (f *FakeNetworkStore) InternalQueryNetworks(transform func(*dockerTypes.NetworkResource) interface{}) []interface{} {
	result := make([]interface{}, 0)

	for _, key := range f.SortedIDs() {
		item := f.InternalGetNetwork(key)
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

// InternalDeleteNetwork removes dockerTypes.NetworkResource from storage without preconditions
func (f *FakeNetworkStore) InternalDeleteNetwork(id string) *dockerTypes.NetworkResource {
	network, ok := f.networks[id]
	if !ok {
		return nil
	}
	delete(f.networks, id)
	delete(f.networksByName, network.Name)
	return network
}

// SortedIDs returns sorted Stack IDs
func (f *FakeNetworkStore) SortedIDs() []string {
	result := []string{}
	for key, value := range f.networks {
		if value != nil {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}
