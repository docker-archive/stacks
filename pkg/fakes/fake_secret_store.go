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
 *   fake_secret_store.go implementation is a customized-but-duplicate of
 *   fake_service_store.go, fake_config_store.go, fake_network_store.go and
 *   fake_stack_store.go.
 *
 *   fake_secret_store.go represents the interfaces.SwarmSecretBackend portions
 *   of the interfaces.BackendClient.
 *
 *   reconciler.fakeReconcilerClient exposes extra API to direct control
 *   of the internals of the implementation for testing.
 *
 *   SortedIDs() []string
 *   InternalDeleteSecret(id string) *swarm.Secret
 *   InternalQuerySecrets(transform func(*swarm.Secret) interface{}) []interface
 *   InternalGetSecret(id string) *swarm.Secret
 *   InternalAddSecret(id string, secret *swarm.Secret)
 *   MarkSecretSpecForError(errorKey string, *swarm.SecretSpec, ops ...string)
 *   SpecifyKeyPrefix(keyPrefix string)
 *   SpecifyErrorTrigger(errorKey string, err error)
 */

// FakeSecretStore contains the subset of Backend APIs SwarmSecretBackend
type FakeSecretStore struct {
	mu          sync.Mutex
	curID       int
	labelErrors map[string]error
	keyPrefix   string

	secrets       map[string]*swarm.Secret
	secretsByName map[string]string
}

// These type registrations are for TESTING in order to create deep copies
func init() {
	typeurl.Register(&swarm.SecretSpec{}, "github.com/docker/swarm/SecretSpec")
	typeurl.Register(&swarm.Secret{}, "github.com/docker/swarm/Secret")
}

// CopySecretSpec duplicates the swarm.SecretSpec
func CopySecretSpec(spec swarm.SecretSpec) *swarm.SecretSpec {
	// any errors ought to be impossible xor panicked in devel
	payload, _ := typeurl.MarshalAny(&spec)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*swarm.SecretSpec)
}

// CopySecret duplicates the swarm.Secret
func CopySecret(spec swarm.Secret) *swarm.Secret {
	// any errors ought to be impossible xor panicked in devel
	payload, _ := typeurl.MarshalAny(&spec)
	iface, _ := typeurl.UnmarshalAny(payload)
	return iface.(*swarm.Secret)
}

// NewFakeSecretStore creates a new FakeSecretStore
func NewFakeSecretStore() *FakeSecretStore {
	return &FakeSecretStore{
		// Don't start from ID 0, to catch any uninitialized types.
		curID:         1,
		secrets:       map[string]*swarm.Secret{},
		secretsByName: map[string]string{},
		labelErrors:   map[string]error{},
	}
}

// resolveID takes a value that might be an ID or and figures out which it is,
// returning the ID
func (f *FakeSecretStore) resolveID(key string) string {
	id, ok := f.secretsByName[key]
	if !ok {
		return key
	}
	return id
}

func (f *FakeSecretStore) newID() string {
	index := f.curID
	f.curID++
	if len(f.keyPrefix) == 0 {
		return fmt.Sprintf("SEC_%v", index)
	}
	return fmt.Sprintf("%s_SEC_%v", f.keyPrefix, index)
}

// GetSecrets implements the GetSecrets method of the SwarmSecretBackend,
// returning a list of secrets. It only supports 1 kind of filter, which is
// a filter for stack ID.
func (f *FakeSecretStore) GetSecrets(opts dockerTypes.SecretListOptions) ([]swarm.Secret, error) {
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

	secrets := []swarm.Secret{}

	for _, key := range f.SortedIDs() {
		secret := f.secrets[key]

		// if we're filtering on stack ID, and this secret doesn't
		// match, then we should skip this secret
		if hasFilter && secret.Spec.Annotations.Labels[types.StackLabel] != stackID {
			continue
		}
		// otherwise, we should append this secret to the set
		if err := f.maybeTriggerAnError("GetSecrets", secret.Spec); err != nil {
			return nil, err
		}
		secrets = append(secrets, *CopySecret(*secret))
	}

	return secrets, nil
}

// GetSecret gets a swarm secret
func (f *FakeSecretStore) GetSecret(idOrName string) (swarm.Secret, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	secret, ok := f.secrets[id]
	if !ok {
		return swarm.Secret{}, errdefs.NotFound(fmt.Errorf("secret %s not found", id))
	}

	if err := f.maybeTriggerAnError("GetSecret", secret.Spec); err != nil {
		return swarm.Secret{}, err
	}
	return *CopySecret(*secret), nil
}

// CreateSecret creates a swarm secret.
func (f *FakeSecretStore) CreateSecret(spec swarm.SecretSpec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.maybeTriggerAnError("CreateSecret", spec); err != nil {
		return "", err
	}

	if _, ok := f.secretsByName[spec.Annotations.Name]; ok {
		return "", FakeInvalidArg
	}

	copied := CopySecretSpec(spec)

	// otherwise, create a secret object
	secret := &swarm.Secret{
		ID: f.newID(),
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: uint64(1),
			},
		},
		Spec: *copied,
	}

	f.InternalAddSecret(secret.ID, secret)

	return secret.ID, nil
}

// UpdateSecret updates the secret to the provided spec.
func (f *FakeSecretStore) UpdateSecret(
	idOrName string,
	version uint64,
	spec swarm.SecretSpec,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)
	secret, ok := f.secrets[id]
	if !ok {
		return errdefs.NotFound(fmt.Errorf("secret %s not found", id))
	}

	if version != secret.Meta.Version.Index {
		return FakeInvalidArg
	}

	if err := f.maybeTriggerAnError("UpdateSecret", secret.Spec); err != nil {
		return err
	}

	if err := f.maybeTriggerAnError("UpdateSecret", spec); err != nil {

		return err
	}

	copied := CopySecretSpec(spec)
	secret.Spec = *copied
	secret.Meta.Version.Index = secret.Meta.Version.Index + 1
	return nil
}

// RemoveSecret deletes the secret
func (f *FakeSecretStore) RemoveSecret(idOrName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	secret := f.InternalGetSecret(id)
	if secret == nil {
		return errdefs.NotFound(fmt.Errorf("secret %s not found", id))
	}

	if err := f.maybeTriggerAnError("RemoveSecret", secret.Spec); err != nil {
		return err
	}

	f.InternalDeleteSecret(id)

	return nil
}

// utility function for interfaces.SwarmSecretBackend calls to trigger an error
func (f *FakeSecretStore) maybeTriggerAnError(operation string, spec swarm.SecretSpec) error {
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

// SpecifyErrorTrigger associates an error to an errorKey so that when calls interfaces.SwarmSecretBackend find a marked swarm.SecretSpec an error is returned
func (f *FakeSecretStore) SpecifyErrorTrigger(errorKey string, err error) {
	f.labelErrors[errorKey] = err
}

// SpecifyKeyPrefix provides prefix to generated ID's
func (f *FakeSecretStore) SpecifyKeyPrefix(keyPrefix string) {
	f.keyPrefix = keyPrefix
}

func (f *FakeSecretStore) constructErrorMark(operation string) string {
	if len(operation) == 0 {
		return f.keyPrefix + ".secretError"
	}
	return f.keyPrefix + "." + operation + ".secretError"
}

// MarkSecretSpecForError marks a swarm.SecretSpec to trigger an error when calls from interfaces.SwarmSecretBackend are configured for the errorKey.
// - All interfaces.SwarmSecretBackend calls may be triggered if len(ops)==0
// - Otherwise, ops may be any of the following: GetSecrets, GetSecret, CreateSecret, UpdateSecret, RemoveSecret
func (f *FakeSecretStore) MarkSecretSpecForError(errorKey string, spec *swarm.SecretSpec, ops ...string) {

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

// InternalAddSecret adds swarm.Secret to storage without preconditions
func (f *FakeSecretStore) InternalAddSecret(id string, secret *swarm.Secret) {
	f.secrets[id] = secret
	f.secretsByName[secret.Spec.Annotations.Name] = id
}

// InternalGetSecret retrieves swarm.Secret or nil from storage without preconditions
func (f *FakeSecretStore) InternalGetSecret(id string) *swarm.Secret {
	secret, ok := f.secrets[id]
	if !ok {
		return nil
	}
	return secret
}

// InternalQuerySecrets retrieves all swarm.Secret from storage while applying a transform
func (f *FakeSecretStore) InternalQuerySecrets(transform func(*swarm.Secret) interface{}) []interface{} {
	result := make([]interface{}, 0)

	for _, key := range f.SortedIDs() {
		item := f.InternalGetSecret(key)
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

// InternalDeleteSecret removes swarm.SecretSpec from storage without preconditions
func (f *FakeSecretStore) InternalDeleteSecret(id string) *swarm.Secret {
	secret, ok := f.secrets[id]
	if !ok {
		return nil
	}
	delete(f.secrets, id)
	delete(f.secretsByName, secret.Spec.Annotations.Name)
	return secret
}

// SortedIDs returns sorted Secret IDs
func (f *FakeSecretStore) SortedIDs() []string {
	result := []string{}
	for key, value := range f.secrets {
		if value != nil {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}
