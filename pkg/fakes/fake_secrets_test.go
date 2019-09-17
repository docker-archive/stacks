package fakes

import (
	"fmt"
	"reflect"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/stretchr/testify/require"
)

func TestUpdateFakeSecretStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeSecretStore()
	store.SpecifyKeyPrefix("TestUpdateFakeSecretStore")
	store.SpecifyErrorTrigger("TestUpdateFakeSecretStore", FakeUnimplemented)

	secret1 := GetTestSecret("secret1")
	secret2 := GetTestSecret("secret2")

	id1, err := store.CreateSecret(secret1.Spec)
	require.NoError(err)

	asecret, err := store.GetSecret(id1)
	require.NoError(err)
	require.Equal(asecret.ID, id1)
	require.True(reflect.DeepEqual(asecret.Spec, secret1.Spec))

	updateErr :=
		store.UpdateSecret(id1, asecret.Version.Index, secret2.Spec)
	require.NoError(updateErr)

	// index out of whack
	updateErr =
		store.UpdateSecret(id1, asecret.Version.Index, secret2.Spec)
	require.Error(updateErr)

	// id missing
	updateErr =
		store.UpdateSecret("123.456", asecret.Version.Index, secret2.Spec)
	require.Error(updateErr)

	asecret, err = store.GetSecret(id1)
	require.NoError(err)
	require.Equal(asecret.ID, id1)
	require.True(reflect.DeepEqual(asecret.Spec, secret2.Spec))

	asecret, err = store.GetSecret("123.456")
	require.Error(err)

	// double creation
	_, err = store.CreateSecret(secret1.Spec)
	require.True(errdefs.IsAlreadyExists(err))
	require.Error(err)
}

func TestIsolationFakeSecretStore(t *testing.T) {
	taintKey := "foo"
	taintValue := "bar"
	require := require.New(t)
	store := NewFakeSecretStore()

	fixtures := GenerateSecretFixtures(1, "TestIsolationFakeSecretStore")
	spec := &fixtures[0].Spec

	id, err := store.CreateSecret(*spec)
	require.NoError(err)
	secret1, _ := store.GetSecret(id)

	// 1. Isolation from creation argument

	require.True(reflect.DeepEqual(*spec, secret1.Spec))
	spec.Annotations.Labels[taintKey] = taintValue
	require.False(reflect.DeepEqual(*spec, secret1.Spec))

	// 2. Isolation between repeated calls to GetSecret

	secretTaint, taintErr := store.GetSecret(id)
	require.NoError(taintErr)
	secretTaint.Spec.Annotations.Labels[taintKey] = taintValue

	require.False(reflect.DeepEqual(secret1.Spec, secretTaint.Spec))

	// 3. Isolation from Update argument (using now changed spec)

	err = store.UpdateSecret(id, 1, *spec)
	require.NoError(err)
	secretUpdated, _ := store.GetSecret(id)

	require.True(reflect.DeepEqual(*spec, secretUpdated.Spec))
	delete(spec.Annotations.Labels, taintKey)
	require.False(reflect.DeepEqual(*spec, secretUpdated.Spec))

}

func TestSpecifiedErrorsFakeSecretStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeSecretStore()
	store.SpecifyKeyPrefix("SpecifiedError")
	store.SpecifyErrorTrigger("SpecifiedError", FakeUnimplemented)

	fixtures := GenerateSecretFixtures(10, "TestSpecifiedErrorsFakeSecretStore")

	var id string
	var err error

	// 0. Leaving untouched

	// 1. forced creation failure
	store.MarkSecretSpecForError("SpecifiedError", &fixtures[1].Spec, "CreateSecret")

	_, err = store.CreateSecret(fixtures[1].Spec)
	require.True(errdefs.IsNotImplemented(err))
	require.Error(err)

	// 2. forced get failure after good create
	store.MarkSecretSpecForError("SpecifiedError", &fixtures[2].Spec, "GetSecret")

	id, err = store.CreateSecret(fixtures[2].Spec)
	require.NoError(err)
	_, err = store.GetSecret(id)
	require.Error(err)

	// 3. forced update failure using untainted #0
	store.MarkSecretSpecForError("SpecifiedError", &fixtures[3].Spec, "UpdateSecret")

	id, err = store.CreateSecret(fixtures[3].Spec)
	require.NoError(err)
	_, err = store.GetSecret(id)
	require.NoError(err)

	err = store.UpdateSecret(id, 1, fixtures[0].Spec)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 4. acquired update failure using tainted #3
	id, err = store.CreateSecret(fixtures[4].Spec)
	require.NoError(err)

	// normal update using #0
	err = store.UpdateSecret(id, 1, fixtures[0].Spec)
	require.NoError(err)

	// tainted update using tainted #3
	err = store.UpdateSecret(id, 2, fixtures[3].Spec)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 5. forced remove failure
	store.MarkSecretSpecForError("SpecifiedError", &fixtures[5].Spec, "RemoveSecret")

	id, err = store.CreateSecret(fixtures[5].Spec)
	require.NoError(err)

	err = store.RemoveSecret(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 6. acquired remove failure using tainted #5
	id, err = store.CreateSecret(fixtures[6].Spec)
	require.NoError(err)

	// update #6 using tainted #5
	err = store.UpdateSecret(id, 1, fixtures[5].Spec)
	require.NoError(err)

	err = store.RemoveSecret(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 7. forced query failure
	store.MarkSecretSpecForError("SpecifiedError", &fixtures[7].Spec, "GetSecrets")

	_, err = store.CreateSecret(fixtures[7].Spec)
	require.NoError(err)

	_, err = store.GetSecrets(dockerTypes.SecretListOptions{})
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 8. force failures by manipulating raw datastructures
	id, err = store.CreateSecret(fixtures[8].Spec)
	require.NoError(err)

	rawSecret := store.InternalGetSecret(id)
	store.MarkSecretSpecForError("SpecifiedError", &rawSecret.Spec)

	err = store.RemoveSecret(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	_, err = store.GetSecret(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	err = store.UpdateSecret(id, 1, fixtures[0].Spec)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// Perform a little raw API test coverage
	pointer := store.InternalDeleteSecret(id)
	require.True(pointer == rawSecret)

	pointer = store.InternalDeleteSecret(id)
	require.Nil(pointer)

	pointer = store.InternalGetSecret(id)
	require.Nil(pointer)

}

func TestCRDFakeSecretStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeSecretStore()

	// Assert the store is empty
	_, filterErr := store.GetSecrets(dockerTypes.SecretListOptions{
		Filters: filters.NewArgs(filters.Arg("a", "b")),
	})
	require.Error(filterErr)

	secrets, err := store.GetSecrets(dockerTypes.SecretListOptions{
		Filters: filters.NewArgs(interfaces.StackLabelArg("Testing123")),
	})
	require.NoError(err)
	require.Empty(secrets)

	secret, err := store.GetSecret("doesntexist")
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(secret)

	// Add three items
	fixtures := GenerateSecretFixtures(4, "TestCRDFakeSecretStore")
	for i := 0; i < 3; i++ {
		id, err := store.CreateSecret(fixtures[i].Spec)
		require.NoError(err, fmt.Sprintf("failed to add fixture %d", i))
		require.NotNil(id)
	}

	// Assert we can list the three items and fetch them individually
	secrets, err = store.GetSecrets(dockerTypes.SecretListOptions{})
	require.NoError(err)
	require.NotNil(secrets)
	require.Len(secrets, 3)

	found := make(map[string]struct{})
	for _, secret := range secrets {
		found[secret.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, id := range []string{"SEC_1", "SEC_2", "SEC_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		secret, err = store.GetSecret(id)
		require.NoError(err)
		require.Equal(secret.ID, id)

		// special test feature
		secret, err = store.GetSecret(secret.Spec.Annotations.Name)
		require.NoError(err)
		require.Equal(secret.ID, id)
	}

	// Assert that the StackLabels on even specs are found
	secretsFilter, errFilter := store.GetSecrets(dockerTypes.SecretListOptions{
		Filters: filters.NewArgs(interfaces.StackLabelArg("TestCRDFakeSecretStore")),
	})
	require.NoError(errFilter)
	require.Len(secretsFilter, 2)

	// Remove second secret
	require.NoError(store.RemoveSecret(secrets[1].ID))

	// Remove second secret again
	require.Error(store.RemoveSecret(secrets[1].ID))

	secretsPointers := store.InternalQuerySecrets(nil)
	require.NotEmpty(secretsPointers)

	idFunction := func(i *swarm.Secret) interface{} { return i }
	secretsPointers = store.InternalQuerySecrets(idFunction)
	require.NotEmpty(secretsPointers)

	// Assert we can list the two items and fetch them individually
	secrets2, err2 := store.GetSecrets(dockerTypes.SecretListOptions{})
	require.NoError(err2)
	require.NotNil(secrets2)
	require.Len(secrets2, 2)

	for _, id := range []string{"SEC_1", "SEC_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		secret, err = store.GetSecret(id)
		require.NoError(err)
		require.Equal(secret.ID, id)
	}

	// Add a new secret
	id, err := store.CreateSecret(fixtures[3].Spec)
	require.NoError(err)
	require.NotNil(id)

	// Ensure that the deleted secret is not present
	secret, err = store.GetSecret(secrets[1].ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))

	// Ensure the expected list of secrets is present
	secrets, err = store.GetSecrets(dockerTypes.SecretListOptions{})
	require.NoError(err)
	require.NotNil(secrets)
	require.Len(secrets, 3)

	found = make(map[string]struct{})
	for _, secret := range secrets {
		found[secret.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, name := range []string{"SEC_1", "SEC_3", "SEC_4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}
}
