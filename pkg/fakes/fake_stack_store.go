package fakes

import (
	"fmt"
	"sort"
	"sync"

	"github.com/containerd/typeurl"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

/*
 *   fake_stack_store.go implementation is a customized-but-duplicate of
 *   fake_service_store.go, fake_secret_store.go, fake_network_store.go and
 *   fake_config_store.go.
 *
 *   fake_stack_store.go represents the interfaces.StacksBackend and
 *   interfaces.StackStore portions of the interfaces.BackendClient.
 *
 *   reconciler.fakeReconcilerClient exposes extra API to direct control
 *   of the internals of the implementation for testing.
 *
 *   SortedIDs() []string
 *   InternalDeleteStack(id string) *types.SnapshotStack
 *   InternalQueryStacks(transform func(*types.StackSpec) interface{}) []interface
 *   InternalGetStack(id string) *types.SnapshotStack
 *   InternalAddStack(id string, config *types.SnapshotStack)
 *   MarkStackSpecForError(errorKey string, *types.StackSpec, ops ...string)
 *   SpecifyKeyPrefix(keyPrefix string)
 *   SpecifyErrorTrigger(errorKey string, err error)
 */

// FakeStackStore contains the subset of Backend APIs for types.Stack
type FakeStackStore struct {
	sync.RWMutex
	curID       int
	labelErrors map[string]error
	keyPrefix   string

	stacks       map[string]*interfaces.SnapshotStack
	stacksByName map[string]string
}

// These type registrations are for TESTING in order to create deep copies
func init() {
	typeurl.Register(&types.StackSpec{}, "github.com/docker/stacks/StackSpec")
	typeurl.Register(&interfaces.SnapshotStack{}, "github.com/docker/interfaces/SnapshotStack")
}

// CopyStackSpec duplicates the types.StackSpec
func CopyStackSpec(spec types.StackSpec) *types.StackSpec {
	payload, _ := typeurl.MarshalAny(&spec)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*types.StackSpec)
}

func fakeConstructStack(snapshotStack *interfaces.SnapshotStack) types.Stack {

	stackSpec := CopyStackSpec(snapshotStack.CurrentSpec)

	stack := types.Stack{
		ID:   snapshotStack.ID,
		Meta: snapshotStack.Meta,
		Spec: *stackSpec,
	}
	return stack
}

// NewFakeStackStore creates a new FakeStackStore
func NewFakeStackStore() *FakeStackStore {
	return &FakeStackStore{
		// Don't start from ID 0, to catch any uninitialized types.
		curID:        1,
		stacks:       make(map[string]*interfaces.SnapshotStack),
		stacksByName: map[string]string{},
		labelErrors:  map[string]error{},
	}
}

// resolveID takes a value that might be an ID or and figures out which it is,
// returning the ID
func (s *FakeStackStore) resolveID(key string) string {
	id, ok := s.stacksByName[key]
	if !ok {
		return key
	}
	return id
}

func (s *FakeStackStore) newID() string {
	index := s.curID
	s.curID++
	if len(s.keyPrefix) == 0 {
		return fmt.Sprintf("STK_%v", index)
	}
	return fmt.Sprintf("%s_STK_%v", s.keyPrefix, index)
}

// AddStack adds a stack to the store.
func (s *FakeStackStore) AddStack(spec types.StackSpec) (string, error) {
	s.Lock()
	defer s.Unlock()

	if err := s.maybeTriggerAnError("AddStack", spec); err != nil {
		return "", err
	}

	if _, ok := s.stacksByName[spec.Annotations.Name]; ok {
		return "", FakeInvalidArg
	}

	copied := CopyStackSpec(spec)

	stackID := s.newID()

	for _, service := range copied.Services {
		if service.Annotations.Labels == nil {
			service.Annotations.Labels = map[string]string{}
		}
		service.Annotations.Labels[types.StackLabel] = stackID
	}

	for _, config := range copied.Configs {
		if config.Annotations.Labels == nil {
			config.Annotations.Labels = map[string]string{}
		}
		config.Annotations.Labels[types.StackLabel] = stackID
	}

	for _, secret := range copied.Secrets {
		if secret.Annotations.Labels == nil {
			secret.Annotations.Labels = map[string]string{}
		}
		secret.Annotations.Labels[types.StackLabel] = stackID
	}

	for _, network := range copied.Networks {
		if network.Labels == nil {
			network.Labels = map[string]string{}
		}
		network.Labels[types.StackLabel] = stackID
	}

	snapshot := &interfaces.SnapshotStack{
		SnapshotResource: interfaces.SnapshotResource{
			ID: stackID,
			Meta: swarm.Meta{
				Version: swarm.Version{
					Index: uint64(1),
				},
			},
			Name: copied.Annotations.Name,
		},
		CurrentSpec: *copied,
		Services:    []interfaces.SnapshotResource{},
		Networks:    []interfaces.SnapshotResource{},
		Secrets:     []interfaces.SnapshotResource{},
		Configs:     []interfaces.SnapshotResource{},
	}

	s.InternalAddStack(snapshot.ID, snapshot)

	return snapshot.ID, nil
}

// UpdateStack updates the stack in the store.
func (s *FakeStackStore) UpdateStack(idOrName string, stackSpec types.StackSpec, version uint64) error {
	s.Lock()
	defer s.Unlock()

	copied := CopyStackSpec(stackSpec)

	id := s.resolveID(idOrName)

	stack := s.InternalGetStack(id)
	if stack == nil {
		return errdefs.NotFound(fmt.Errorf("stack %s not found", id))
	}

	if stack.Version.Index != version {
		return fmt.Errorf("update out of sequence")
	}

	if err := s.maybeTriggerAnError("UpdateStack", stack.CurrentSpec); err != nil {
		return err
	}

	if err := s.maybeTriggerAnError("UpdateStack", stackSpec); err != nil {
		return err
	}

	stack.Version.Index++
	stackID := stack.ID

	for _, service := range copied.Services {
		if service.Annotations.Labels == nil {
			service.Annotations.Labels = map[string]string{}
		}
		service.Annotations.Labels[types.StackLabel] = stackID
	}
	for _, config := range copied.Configs {
		if config.Annotations.Labels == nil {
			config.Annotations.Labels = map[string]string{}
		}
		config.Annotations.Labels[types.StackLabel] = stackID
	}
	for _, secret := range copied.Secrets {
		if secret.Annotations.Labels == nil {
			secret.Annotations.Labels = map[string]string{}
		}
		secret.Annotations.Labels[types.StackLabel] = stackID
	}
	for _, network := range copied.Networks {
		if network.Labels == nil {
			network.Labels = map[string]string{}
		}
		network.Labels[types.StackLabel] = stackID
	}
	stack.CurrentSpec = *copied
	s.stacks[id] = stack
	return nil
}

// UpdateSnapshotStack updates the snapshot in the store.
func (s *FakeStackStore) UpdateSnapshotStack(idOrName string, snapshot interfaces.SnapshotStack, version uint64) error {
	s.Lock()
	defer s.Unlock()

	id := s.resolveID(idOrName)

	stack := s.InternalGetStack(id)
	if stack == nil {
		return errdefs.NotFound(fmt.Errorf("stack %s not found", id))
	}

	if stack.Version.Index != version {
		return fmt.Errorf("update out of sequence")
	}

	if err := s.maybeTriggerAnError("UpdateSnapshotStack", stack.CurrentSpec); err != nil {
		return err
	}

	if err := s.maybeTriggerAnError("UpdateSnapshotStack", snapshot.CurrentSpec); err != nil {
		return err
	}

	stack.Version.Index++

	// No accidental or sly changes to the StackSpec are permitted
	stack.Services = snapshot.Services
	stack.Configs = snapshot.Configs
	stack.Secrets = snapshot.Secrets
	stack.Networks = snapshot.Networks

	s.stacks[id] = stack
	return nil
}

// DeleteStack removes a stack from the store.
func (s *FakeStackStore) DeleteStack(idOrName string) error {
	s.Lock()
	defer s.Unlock()

	id := s.resolveID(idOrName)

	stack := s.InternalGetStack(id)
	if stack == nil {
		return errdefs.NotFound(fmt.Errorf("stack %s not found", id))
	}
	if err := s.maybeTriggerAnError("DeleteStack", stack.CurrentSpec); err != nil {
		return err
	}
	s.InternalDeleteStack(id)
	return nil
}

// GetStack retrieves a single stack from the store.
func (s *FakeStackStore) GetStack(idOrName string) (types.Stack, error) {
	s.RLock()
	defer s.RUnlock()
	id := s.resolveID(idOrName)
	stack := s.InternalGetStack(id)
	if stack == nil {
		return types.Stack{}, errdefs.NotFound(fmt.Errorf("stack %s not found", id))
	}
	return fakeConstructStack(stack), s.maybeTriggerAnError("GetStack", stack.CurrentSpec)
}

// GetSnapshotStack retrieves a single stack from the store.
func (s *FakeStackStore) GetSnapshotStack(idOrName string) (interfaces.SnapshotStack, error) {
	s.RLock()
	defer s.RUnlock()
	id := s.resolveID(idOrName)
	stack := s.InternalGetStack(id)
	if stack == nil {
		return interfaces.SnapshotStack{}, errdefs.NotFound(fmt.Errorf("stack %s not found", id))
	}
	return *stack, s.maybeTriggerAnError("GetSnapshotStack", stack.CurrentSpec)
}

// ListStacks returns all known stacks from the store.
func (s *FakeStackStore) ListStacks() ([]types.Stack, error) {
	s.RLock()
	defer s.RUnlock()
	stacks := []types.Stack{}
	for _, key := range s.SortedIDs() {
		snapshot := s.stacks[key]
		if err := s.maybeTriggerAnError("ListStacks", snapshot.CurrentSpec); err != nil {
			return nil, err
		}
		stacks = append(stacks, fakeConstructStack(snapshot))
	}
	return stacks, nil
}

func (s *FakeStackStore) maybeTriggerAnError(operation string, spec types.StackSpec) error {
	key := s.constructErrorMark(operation)
	errorName, ok := spec.Annotations.Labels[key]
	if !ok {
		key := s.constructErrorMark("")
		errorName, ok = spec.Annotations.Labels[key]
		if !ok {
			return nil
		}
	}

	return s.labelErrors[errorName]
}

// SpecifyErrorTrigger associates an error to an errorKey so that when calls to interfaces.StackStore find a marked types.StackSpec an error is returned
func (s *FakeStackStore) SpecifyErrorTrigger(errorKey string, err error) {
	s.labelErrors[errorKey] = err
}

// SpecifyKeyPrefix provides prefix to generated ID's
func (s *FakeStackStore) SpecifyKeyPrefix(keyPrefix string) {
	s.keyPrefix = keyPrefix
}

func (s *FakeStackStore) constructErrorMark(operation string) string {
	if len(operation) == 0 {
		return s.keyPrefix + ".storeError"
	}
	return s.keyPrefix + "." + operation + ".storeError"
}

// MarkStackSpecForError marks a ConfigSpec to trigger an error when calls from interfaces.SwarmConfigBackend are configured for the errorKey.
// - All interfaces.SwarmConfigBackend calls may be triggered if len(ops)==0
// - Otherwise, ops may be any of the following: ListStacks, GetStack, AddStack, UpdateStack, DeleteStack, UpdateSnapshotStack, GetSnapshotStack
func (s *FakeStackStore) MarkStackSpecForError(errorKey string, spec *types.StackSpec, ops ...string) {

	if spec.Annotations.Labels == nil {
		spec.Annotations.Labels = make(map[string]string)
	}
	if len(ops) == 0 {
		key := s.constructErrorMark("")
		spec.Annotations.Labels[key] = errorKey
	} else {
		for _, operation := range ops {
			key := s.constructErrorMark(operation)
			spec.Annotations.Labels[key] = errorKey
		}
	}
}

// InternalAddStack adds types.Stack to storage without preconditions
func (s *FakeStackStore) InternalAddStack(id string, snapshot *interfaces.SnapshotStack) {
	s.stacks[id] = snapshot
	s.stacksByName[snapshot.Name] = id
}

// InternalGetStack retrieves types.Stack or nil from storage without preconditions
func (s *FakeStackStore) InternalGetStack(id string) *interfaces.SnapshotStack {
	stack, ok := s.stacks[id]
	if !ok {
		return nil
	}
	return stack
}

// InternalQueryStacks retrieves all types.Stack from storage while applying a transform
func (s *FakeStackStore) InternalQueryStacks(transform func(*interfaces.SnapshotStack) interface{}) []interface{} {
	result := make([]interface{}, 0)

	for _, key := range s.SortedIDs() {
		item := s.InternalGetStack(key)
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

// InternalDeleteStack removes types.Stack from storage without preconditions
func (s *FakeStackStore) InternalDeleteStack(id string) *interfaces.SnapshotStack {
	snapshot, ok := s.stacks[id]
	if !ok {
		return nil
	}
	delete(s.stacks, id)
	delete(s.stacksByName, snapshot.Name)
	return snapshot
}

// SortedIDs returns sorted Stack IDs
func (s *FakeStackStore) SortedIDs() []string {
	result := []string{}
	for key, value := range s.stacks {
		if value != nil {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}
