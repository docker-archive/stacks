package fakes

import (
	"fmt"
	"sort"
	"sync"

	"github.com/containerd/typeurl"
	gogotypes "github.com/gogo/protobuf/types"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/types"
)

// FakeSecretStore contains the subset of Backend APIs for swarm.SecretSpec
type FakeSecretStore struct {
	mu          sync.Mutex
	curID       int
	labelErrors map[string]error
	keyPrefix   string

	secrets       map[string]*swarm.Secret
	secretsByName map[string]string
}

func init() {
	typeurl.Register(&swarm.SecretSpec{}, "github.com/docker/swarm/SecretSpec")
}

// CopySecretSpec duplicates the swarm.SecretSpec
func CopySecretSpec(spec swarm.SecretSpec) (swarm.SecretSpec, error) {
	var payload *gogotypes.Any
	var err error
	payload, err = typeurl.MarshalAny(&spec)
	if err != nil {
		return swarm.SecretSpec{}, err
	}
	iface, err := typeurl.UnmarshalAny(payload)
	if err != nil {
		return swarm.SecretSpec{}, err
	}
	return *iface.(*swarm.SecretSpec), nil
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

func (f *FakeSecretStore) newID(objType string) string {
	index := f.curID
	f.curID++
	return fmt.Sprintf("id_%s_%v", objType, index)
}

// GetSecrets implements the GetSecrets method of the BackendClient,
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
		// if we're filtering on stack ID, and this secret doesn't match, then
		// we should skip this secret
		if hasFilter && secret.Spec.Annotations.Labels[types.StackLabel] != stackID {
			continue
		}
		// otherwise, we should append this secret to the set
		if err := f.causeAnError(nil, "GetSecrets", secret.Spec); err != nil {
			return nil, err
		}
		secrets = append(secrets, *secret)
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
		return swarm.Secret{}, FakeNotFound
	}

	if err := f.causeAnError(nil, "GetSecret", secret.Spec); err != nil {
		return swarm.Secret{}, FakeUnavailable
	}
	return *secret, nil
}

// CreateSecret creates a swarm secret.
func (f *FakeSecretStore) CreateSecret(spec swarm.SecretSpec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.causeAnError(nil, "CreateSecret", spec); err != nil {
		return "", FakeInvalidArg
	}

	if _, ok := f.secretsByName[spec.Annotations.Name]; ok {
		return "", FakeInvalidArg
	}
	copied, err := CopySecretSpec(spec)
	if err != nil {
		return "", err
	}

	// otherwise, create a secret object
	secret := &swarm.Secret{
		ID: f.newID("secret"),
		Meta: swarm.Meta{
			Version: swarm.Version{
				Index: uint64(1),
			},
		},
		Spec: copied,
	}

	f.secretsByName[spec.Annotations.Name] = secret.ID
	f.secrets[secret.ID] = secret

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
		return FakeNotFound
	}

	if version != secret.Meta.Version.Index {
		return FakeInvalidArg
	}

	copied, err := CopySecretSpec(spec)
	secret.Spec = copied
	secret.Meta.Version.Index = secret.Meta.Version.Index + 1
	return err
}

// RemoveSecret deletes the config
func (f *FakeSecretStore) RemoveSecret(idOrName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := f.resolveID(idOrName)

	secret, ok := f.secrets[id]
	if !ok {
		return FakeNotFound
	}

	if err := f.causeAnError(nil, "RemoveSecret", secret.Spec); err != nil {
		return err
	}

	delete(f.secrets, secret.ID)
	delete(f.secretsByName, secret.Spec.Annotations.Name)

	return nil
}

func (f *FakeSecretStore) causeAnError(err error, operation string, spec swarm.SecretSpec) error {
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
func (f *FakeSecretStore) SpecifyError(errorKey string, err error) {
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

// MarkInputForError mark swarm.SecretSpec with potential errors
func (f *FakeSecretStore) MarkInputForError(errorKey string, input interface{}, ops ...string) {

	spec := input.(*swarm.SecretSpec)
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

// DirectAdd adds swarm.SecretSpec to storage without preconditions
func (f *FakeSecretStore) DirectAdd(id string, iface interface{}) {
	var secret *swarm.Secret = iface.(*swarm.Secret)
	f.secrets[id] = secret
	f.secretsByName[secret.Spec.Annotations.Name] = id
}

// DirectGet retrieves swarm.SecretSpec or nil from storage without preconditions
func (f *FakeSecretStore) DirectGet(id string) interface{} {
	secret, ok := f.secrets[id]

	if !ok {
		return &swarm.Secret{}
	}
	return secret
}

// DirectAll retrieves all swarm.SecretSpec from storage while applying a transform
func (f *FakeSecretStore) DirectAll(transform func(interface{}) interface{}) []interface{} {
	result := make([]interface{}, 0, len(f.secrets))
	for _, item := range f.secrets {
		if transform == nil {
			result = append(result, item)
		} else {
			result = append(result, transform(item))
		}
	}
	return result
}

// DirectDelete removes swarm.SecretSpec from storage without preconditions
func (f *FakeSecretStore) DirectDelete(id string) interface{} {
	secret, ok := f.secrets[id]
	if !ok {
		return &swarm.Secret{}
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
