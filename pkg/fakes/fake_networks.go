package fakes

import (
	"fmt"
	"sort"
	"sync"

	"github.com/containerd/typeurl"
	gogotypes "github.com/gogo/protobuf/types"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"

	"github.com/docker/stacks/pkg/types"
)

// FakeNetworkStore contains the subset of Backend APIs for dockerTypes.NetworkResource
type FakeNetworkStore struct {
	mu          sync.Mutex
	curID       int
	labelErrors map[string]error
	keyPrefix   string

	networks       map[string]*dockerTypes.NetworkResource
	networksByName map[string]string
}

func init() {
	typeurl.Register(&dockerTypes.NetworkCreate{}, "github.com/docker/api/NetworkCreate")
}

// CopyNetworkCreate duplicates the NetworkCreate
func CopyNetworkCreate(spec dockerTypes.NetworkCreate) (dockerTypes.NetworkCreate, error) {
	var payload *gogotypes.Any
	var err error
	payload, err = typeurl.MarshalAny(&spec)
	if err != nil {
		return dockerTypes.NetworkCreate{}, err
	}
	iface, err := typeurl.UnmarshalAny(payload)
	if err != nil {
		return dockerTypes.NetworkCreate{}, err
	}
	return *iface.(*dockerTypes.NetworkCreate), nil
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

func (f *FakeNetworkStore) newID(objType string) string {
	index := f.curID
	f.curID++
	return fmt.Sprintf("id_%s_%v", objType, index)
}

// GetNetworks implements the GetNetworks method of the BackendClient,
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
		// if we're filtering on stack ID, and this network doesn't match, then
		// we should skip this network
		if hasFilter && network.Labels[types.StackLabel] != stackID {
			continue
		}
		// otherwise, we should append this network to the set
		if err := f.causeAnError(nil, "GetNetworks", *network); err != nil {
			return nil, err
		}
		networks = append(networks, *network)
	}

	return networks, nil
}

// GetNetworksByName implements the GetNetworks method of the BackendClient,
// returning a list of networks by name
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
		if err := f.causeAnError(nil, "GetNetworks", *network); err != nil {
			return nil, err
		}
		networks = append(networks, *network)
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
		return dockerTypes.NetworkResource{}, FakeNotFound
	}

	if err := f.causeAnError(nil, "GetNetwork", *network); err != nil {
		return dockerTypes.NetworkResource{}, FakeUnavailable
	}
	return *network, nil
}

// CreateNetwork creates a swarm network.
func (f *FakeNetworkStore) CreateNetwork(request dockerTypes.NetworkCreateRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	copied, err := CopyNetworkCreate(request.NetworkCreate)
	if err != nil {
		return "", err
	}

	// otherwise, create a network object
	network := &dockerTypes.NetworkResource{
		ID:         f.newID("network"),
		Name:       request.Name,
		Driver:     copied.Driver,
		Scope:      copied.Scope,
		EnableIPv6: copied.EnableIPv6,
		IPAM:       *copied.IPAM,
		Internal:   copied.Internal,
		Attachable: copied.Attachable,
		Ingress:    copied.Ingress,
		ConfigOnly: copied.ConfigOnly,
		ConfigFrom: *copied.ConfigFrom,
		Options:    copied.Options,
		Labels:     copied.Labels,

		// FIXME
		//		Meta: swarm.Meta{
		//			Version: swarm.Version{
		//				Index: uint64(1),
		//			},
		//		},
	}

	if err := f.causeAnError(nil, "CreateNetwork", *network); err != nil {
		return "", err
	}

	if _, ok := f.networksByName[request.Name]; ok {
		return "", FakeInvalidArg
	}

	f.networksByName[request.Name] = network.ID
	f.networks[network.ID] = network

	return network.ID, nil
}

// RemoveNetwork deletes the config
func (f *FakeNetworkStore) RemoveNetwork(idOrName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	network, ok := f.networks[id]
	if !ok {
		return FakeNotFound
	}

	if err := f.causeAnError(nil, "RemoveNetwork", *network); err != nil {
		return err
	}

	delete(f.networks, network.ID)
	delete(f.networksByName, network.Name)

	return nil
}

func (f *FakeNetworkStore) causeAnError(err error, operation string, spec dockerTypes.NetworkResource) error {
	if err != nil {
		return err
	}

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

// SpecifyError associates an error to a key
func (f *FakeNetworkStore) SpecifyError(errorKey string, err error) {
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

// MarkInputForError mark NetworkCreate with potential errors
func (f *FakeNetworkStore) MarkInputForError(errorKey string, input interface{}, ops ...string) {

	spec := input.(*dockerTypes.NetworkCreate)
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

// DirectAdd adds dockerTypes.NetworkResource to storage without preconditions
func (f *FakeNetworkStore) DirectAdd(id string, iface interface{}) {
	var network *dockerTypes.NetworkResource = iface.(*dockerTypes.NetworkResource)
	f.networks[id] = network
	f.networksByName[network.Name] = id
}

// DirectGet retrieves dockerTypes.NetworkResource or nil from storage without preconditions
func (f *FakeNetworkStore) DirectGet(id string) interface{} {
	network, ok := f.networks[id]
	if !ok {
		return &dockerTypes.NetworkResource{}
	}
	return network
}

// DirectAll retrieves all dockerTypes.NetworkResource from storage while applying a transform
func (f *FakeNetworkStore) DirectAll(transform func(interface{}) interface{}) []interface{} {
	result := make([]interface{}, 0, len(f.networks))
	for _, item := range f.networks {
		if transform == nil {
			result = append(result, item)
		} else {
			result = append(result, transform(item))
		}
	}
	return result
}

// DirectDelete removes dockerTypes.NetworkResource from storage without preconditions
func (f *FakeNetworkStore) DirectDelete(id string) interface{} {
	network, ok := f.networks[id]
	if !ok {
		return &dockerTypes.NetworkResource{}
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
