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

// FakeConfigStore contains the subset of Backend APIs for swarm.Config
type FakeConfigStore struct {
	mu          sync.Mutex
	curID       int
	labelErrors map[string]error
	keyPrefix   string

	configs       map[string]*swarm.Config
	configsByName map[string]string
}

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

func (f *FakeConfigStore) newID(objType string) string {
	index := f.curID
	f.curID++
	return fmt.Sprintf("id_%s_%v", objType, index)
}

// GetConfigs implements the GetConfigs method of the BackendClient,
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
		// if we're filtering on stack ID, and this config doesn't match, then
		// we should skip this config
		config := f.configs[key]
		if hasFilter && config.Spec.Annotations.Labels[types.StackLabel] != stackID {
			continue
		}
		// otherwise, we should append this config to the set
		if err := f.causeAnError("GetConfigs", config.Spec); err != nil {
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

	config, ok := f.configs[id]
	if !ok {
		return swarm.Config{}, errdefs.NotFound(fmt.Errorf("config %s not found", id))
	}

	if err := f.causeAnError("GetConfig", config.Spec); err != nil {
		return swarm.Config{}, err
	}

	return *CopyConfig(*config), nil
}

// CreateConfig creates a swarm config.
func (f *FakeConfigStore) CreateConfig(spec swarm.ConfigSpec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.causeAnError("CreateConfig", spec); err != nil {
		return "", err
	}

	if _, ok := f.configsByName[spec.Annotations.Name]; ok {
		return "", FakeInvalidArg
	}
	copied := CopyConfigSpec(spec)

	// otherwise, create a config object
	config := &swarm.Config{
		ID: f.newID("config"),
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: uint64(1),
			},
		},
		Spec: *copied,
	}

	f.DirectAdd(config.ID, config)

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

	if err := f.causeAnError("UpdateConfig", config.Spec); err != nil {
		return err
	}

	if err := f.causeAnError("UpdateConfig", spec); err != nil {
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

	config, ok := f.configs[id]
	if !ok {
		return errdefs.NotFound(fmt.Errorf("config %s not found", id))
	}

	if err := f.causeAnError("RemoveConfig", config.Spec); err != nil {
		return err
	}

	f.DirectDelete(id)

	return nil
}

func (f *FakeConfigStore) causeAnError(operation string, spec swarm.ConfigSpec) error {
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
func (f *FakeConfigStore) SpecifyError(errorKey string, err error) {
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

// MarkInputForError mark ConfigSpec with potential errors
func (f *FakeConfigStore) MarkInputForError(errorKey string, input interface{}, ops ...string) {

	spec := input.(*swarm.ConfigSpec)
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

// DirectAdd adds swarm.Config to storage without preconditions
func (f *FakeConfigStore) DirectAdd(id string, config *swarm.Config) {
	f.configs[id] = config
	f.configsByName[config.Spec.Annotations.Name] = id
}

// DirectGet retrieves swarm.Config or nil from storage without preconditions
func (f *FakeConfigStore) DirectGet(id string) *swarm.Config {
	config, ok := f.configs[id]
	if !ok {
		return nil
	}
	return config
}

// DirectAll retrieves all swarm.Config from storage while applying a transform
func (f *FakeConfigStore) DirectAll(transform func(*swarm.Config) interface{}) []interface{} {
	result := make([]interface{}, 0, len(f.configs))

	for _, key := range f.SortedIDs() {
		item := f.DirectGet(key)
		if transform == nil {
			result = append(result, item)
		} else {
			result = append(result, transform(item))
		}
	}
	return result
}

// DirectDelete removes swarm.Config from storage without preconditions
func (f *FakeConfigStore) DirectDelete(id string) *swarm.Config {
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
