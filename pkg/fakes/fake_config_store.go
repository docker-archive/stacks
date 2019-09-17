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
 *   fake_config_store.go implementation is a customized-but-duplicate of
 *   fake_service_store.go, fake_secret_store.go, fake_network_store.go and
 *   fake_stack_store.go.
 *
 *   fake_config_store.go represents the interfaces.SwarmConfigBackend portions
 *   of the interfaces.BackendClient.
 *
 *   reconciler.fakeReconcilerClient exposes extra API to direct control
 *   of the internals of the implementation for testing.
 *
 *   SortedIDs() []string
 *   InternalDeleteConfig(id string) *swarm.Config
 *   InternalQueryConfigs(transform func(*swarm.Config) interface{}) []interface
 *   InternalGetConfig(id string) *swarm.Config
 *   InternalAddConfig(id string, config *swarm.Config)
 *   MarkConfigSpecForError(errorKey string, *swarm.ConfigSpec, ops ...string)
 *   SpecifyKeyPrefix(keyPrefix string)
 *   SpecifyErrorTrigger(errorKey string, err error)
 */

// FakeConfigStore contains the subset of Backend APIs for swarm.Config
type FakeConfigStore struct {
	mu          sync.Mutex
	curID       int
	labelErrors map[string]error
	keyPrefix   string

	configs       map[string]*swarm.Config
	configsByName map[string]string
}

// These type registrations are for TESTING in order to create deep copies
func init() {
	typeurl.Register(&swarm.ConfigSpec{}, "github.com/docker/swarm/ConfigSpec")
	typeurl.Register(&swarm.Config{}, "github.com/docker/swarm/Config")
}

// CopyConfigSpec duplicates the ConfigSpec
func CopyConfigSpec(spec swarm.ConfigSpec) *swarm.ConfigSpec {
	// any errors ought to be impossible xor panicked in devel
	payload, _ := typeurl.MarshalAny(&spec)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*swarm.ConfigSpec)
}

// CopyConfig duplicates the Config
func CopyConfig(config swarm.Config) *swarm.Config {
	// any errors ought to be impossible xor panicked in devel
	payload, _ := typeurl.MarshalAny(&config)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*swarm.Config)
}

// NewFakeConfigStore creates a new FakeConfigStore
func NewFakeConfigStore() *FakeConfigStore {
	return &FakeConfigStore{
		// Don't start from ID 0, to catch any uninitialized types.
		curID:         1,
		configs:       map[string]*swarm.Config{},
		configsByName: map[string]string{},
		labelErrors:   map[string]error{},
	}
}

// resolveID takes a value that might be an ID or and figures out which it is,
// returning the ID
func (f *FakeConfigStore) resolveID(key string) string {
	id, ok := f.configsByName[key]
	if !ok {
		return key
	}
	return id
}

func (f *FakeConfigStore) newID() string {
	index := f.curID
	f.curID++
	if len(f.keyPrefix) == 0 {
		return fmt.Sprintf("CFG_%v", index)
	}
	return fmt.Sprintf("%s_CFG_%v", f.keyPrefix, index)
}

// GetConfigs implements the GetConfigs method of the SwarmConfigBackend,
// returning a list of configs. It only supports 1 kind of filter, which is
// a filter for stack ID.
func (f *FakeConfigStore) GetConfigs(opts dockerTypes.ConfigListOptions) ([]swarm.Config, error) {
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

	configs := []swarm.Config{}

	for _, key := range f.SortedIDs() {
		config := f.configs[key]

		// if we're filtering on stack ID, and this config doesn't
		// match, then we should skip this config
		if hasFilter && config.Spec.Annotations.Labels[types.StackLabel] != stackID {
			continue
		}
		// otherwise, we should append this config to the set
		if err := f.maybeTriggerAnError("GetConfigs", config.Spec); err != nil {
			return nil, err
		}
		configs = append(configs, *CopyConfig(*config))
	}

	return configs, nil
}

// GetConfig gets a swarm config
func (f *FakeConfigStore) GetConfig(idOrName string) (swarm.Config, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	config := f.InternalGetConfig(id)
	if config == nil {
		return swarm.Config{}, errdefs.NotFound(fmt.Errorf("config %s not found", id))
	}

	if err := f.maybeTriggerAnError("GetConfig", config.Spec); err != nil {
		return swarm.Config{}, err
	}

	return *CopyConfig(*config), nil
}

// CreateConfig creates a swarm config.
func (f *FakeConfigStore) CreateConfig(spec swarm.ConfigSpec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.maybeTriggerAnError("CreateConfig", spec); err != nil {
		return "", err
	}

	if _, ok := f.configsByName[spec.Annotations.Name]; ok {
		return "", errdefs.AlreadyExists(fmt.Errorf("config %s already used", spec.Annotations.Name))
	}
	copied := CopyConfigSpec(spec)

	// otherwise, create a config object
	config := &swarm.Config{
		ID: f.newID(),
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: uint64(1),
			},
		},
		Spec: *copied,
	}

	f.InternalAddConfig(config.ID, config)

	return config.ID, nil
}

// UpdateConfig updates the config to the provided spec.
func (f *FakeConfigStore) UpdateConfig(
	idOrName string,
	version uint64,
	spec swarm.ConfigSpec,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)
	config, ok := f.configs[id]
	if !ok {
		return errdefs.NotFound(fmt.Errorf("config %s not found", id))
	}

	if version != config.Meta.Version.Index {
		return FakeInvalidArg
	}

	if err := f.maybeTriggerAnError("UpdateConfig", config.Spec); err != nil {
		return err
	}

	if err := f.maybeTriggerAnError("UpdateConfig", spec); err != nil {
		return err
	}

	copied := CopyConfigSpec(spec)
	config.Spec = *copied
	config.Meta.Version.Index = config.Meta.Version.Index + 1
	return nil
}

// RemoveConfig deletes the config
func (f *FakeConfigStore) RemoveConfig(idOrName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	config := f.InternalGetConfig(id)
	if config == nil {
		return errdefs.NotFound(fmt.Errorf("config %s not found", id))
	}

	if err := f.maybeTriggerAnError("RemoveConfig", config.Spec); err != nil {
		return err
	}

	f.InternalDeleteConfig(id)

	return nil
}

// utility function for interfaces.SwarmConfigBackend calls to trigger an error
func (f *FakeConfigStore) maybeTriggerAnError(operation string, spec swarm.ConfigSpec) error {
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

// SpecifyErrorTrigger associates an error to an errorKey so that when calls interfaces.SwarmConfigBackend find a marked swarm.ConfigSpec an error is returned
func (f *FakeConfigStore) SpecifyErrorTrigger(errorKey string, err error) {
	f.labelErrors[errorKey] = err
}

// SpecifyKeyPrefix provides prefix to generated ID's
func (f *FakeConfigStore) SpecifyKeyPrefix(keyPrefix string) {
	f.keyPrefix = keyPrefix
}

func (f *FakeConfigStore) constructErrorMark(operation string) string {
	if len(operation) == 0 {
		return f.keyPrefix + ".configError"
	}
	return f.keyPrefix + "." + operation + ".configError"
}

// MarkConfigSpecForError marks a ConfigSpec to trigger an error when calls from interfaces.SwarmConfigBackend are configured for the errorKey.
// - All interfaces.SwarmConfigBackend calls may be triggered if len(ops)==0
// - Otherwise, ops may be any of the following: GetConfigs, GetConfig, CreateConfig, UpdateConfig, RemoveConfig
func (f *FakeConfigStore) MarkConfigSpecForError(errorKey string, spec *swarm.ConfigSpec, ops ...string) {

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

// InternalAddConfig adds swarm.Config to storage without preconditions
func (f *FakeConfigStore) InternalAddConfig(id string, config *swarm.Config) {
	f.configs[id] = config
	f.configsByName[config.Spec.Annotations.Name] = id
}

// InternalGetConfig retrieves swarm.Config or nil from storage without preconditions
func (f *FakeConfigStore) InternalGetConfig(id string) *swarm.Config {
	config, ok := f.configs[id]
	if !ok {
		return nil
	}
	return config
}

// InternalQueryConfigs retrieves all swarm.Config from storage while applying a transform
func (f *FakeConfigStore) InternalQueryConfigs(transform func(*swarm.Config) interface{}) []interface{} {
	result := make([]interface{}, 0)

	for _, key := range f.SortedIDs() {
		item := f.InternalGetConfig(key)
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

// InternalDeleteConfig removes swarm.Config from storage without preconditions
func (f *FakeConfigStore) InternalDeleteConfig(id string) *swarm.Config {
	config, ok := f.configs[id]
	if !ok {
		return nil
	}
	delete(f.configs, id)
	delete(f.configsByName, config.Spec.Annotations.Name)
	return config
}

// SortedIDs returns sorted Config IDs
func (f *FakeConfigStore) SortedIDs() []string {
	result := []string{}
	for key, value := range f.configs {
		if value != nil {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}
