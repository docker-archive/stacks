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

func TestUpdateFakeConfigStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeConfigStore()
	store.SpecifyKeyPrefix("TestUpdateFakeConfigStore")
	store.SpecifyErrorTrigger("TestUpdateFakeConfigStore", FakeUnimplemented)

	config1 := GetTestConfig("config1")
	config2 := GetTestConfig("config2")

	id1, err := store.CreateConfig(config1.Spec)
	require.NoError(err)

	aconfig, err := store.GetConfig(id1)
	require.NoError(err)
	require.Equal(aconfig.ID, id1)
	require.True(reflect.DeepEqual(aconfig.Spec, config1.Spec))

	updateErr :=
		store.UpdateConfig(id1, aconfig.Version.Index, config2.Spec)
	require.NoError(updateErr)

	// index out of whack
	updateErr =
		store.UpdateConfig(id1, aconfig.Version.Index, config2.Spec)
	require.Error(updateErr)

	// id missing
	updateErr =
		store.UpdateConfig("123.456", aconfig.Version.Index, config2.Spec)
	require.Error(updateErr)

	aconfig, err = store.GetConfig(id1)
	require.NoError(err)
	require.Equal(aconfig.ID, id1)
	require.True(reflect.DeepEqual(aconfig.Spec, config2.Spec))

	aconfig, err = store.GetConfig("123.456")
	require.Error(err)

	// double creation
	_, err = store.CreateConfig(config1.Spec)
	require.True(errdefs.IsInvalidParameter(err))
	require.Error(err)
}

func TestIsolationFakeConfigStore(t *testing.T) {
	taintKey := "foo"
	taintValue := "bar"
	require := require.New(t)
	store := NewFakeConfigStore()

	fixtures := GenerateConfigFixtures(1, "TestIsolationFakeConfigStore")
	spec := &fixtures[0].Spec

	id, err := store.CreateConfig(*spec)
	require.NoError(err)
	config1, _ := store.GetConfig(id)

	// 1. Isolation from creation argument

	require.True(reflect.DeepEqual(*spec, config1.Spec))
	spec.Annotations.Labels[taintKey] = taintValue
	require.False(reflect.DeepEqual(*spec, config1.Spec))

	// 2. Isolation between repeated calls to GetConfig

	configTaint, taintErr := store.GetConfig(id)
	require.NoError(taintErr)
	configTaint.Spec.Annotations.Labels[taintKey] = taintValue

	require.False(reflect.DeepEqual(config1.Spec, configTaint.Spec))

	// 3. Isolation from Update argument (using now changed spec)

	err = store.UpdateConfig(id, 1, *spec)
	require.NoError(err)
	configUpdated, _ := store.GetConfig(id)

	require.True(reflect.DeepEqual(*spec, configUpdated.Spec))
	delete(spec.Annotations.Labels, taintKey)
	require.False(reflect.DeepEqual(*spec, configUpdated.Spec))

}

func TestSpecifiedErrorsFakeConfigStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeConfigStore()
	store.SpecifyKeyPrefix("SpecifiedError")
	store.SpecifyErrorTrigger("SpecifiedError", FakeUnimplemented)

	fixtures := GenerateConfigFixtures(10, "TestSpecifiedErrorsFakeConfigStore")

	var id string
	var err error

	// 0. Leaving untouched

	// 1. forced creation failure
	store.MarkConfigSpecForError("SpecifiedError", &fixtures[1].Spec, "CreateConfig")

	_, err = store.CreateConfig(fixtures[1].Spec)
	require.True(errdefs.IsUnavailable(err))
	require.Error(err)

	// 2. forced get failure after good create
	store.MarkConfigSpecForError("SpecifiedError", &fixtures[2].Spec, "GetConfig")

	id, err = store.CreateConfig(fixtures[2].Spec)
	require.NoError(err)
	_, err = store.GetConfig(id)
	require.Error(err)

	// 3. forced update failure using untainted #0
	store.MarkConfigSpecForError("SpecifiedError", &fixtures[3].Spec, "UpdateConfig")

	id, err = store.CreateConfig(fixtures[3].Spec)
	require.NoError(err)
	_, err = store.GetConfig(id)
	require.NoError(err)

	err = store.UpdateConfig(id, 1, fixtures[0].Spec)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 4. acquired update failure using tainted #3
	id, err = store.CreateConfig(fixtures[4].Spec)
	require.NoError(err)

	// normal update using #0
	err = store.UpdateConfig(id, 1, fixtures[0].Spec)
	require.NoError(err)

	// tainted update using tainted #3
	err = store.UpdateConfig(id, 2, fixtures[3].Spec)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 5. forced remove failure
	store.MarkConfigSpecForError("SpecifiedError", &fixtures[5].Spec, "RemoveConfig")

	id, err = store.CreateConfig(fixtures[5].Spec)
	require.NoError(err)

	err = store.RemoveConfig(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 6. acquired remove failure using tainted #5
	id, err = store.CreateConfig(fixtures[6].Spec)
	require.NoError(err)

	// update #6 using tainted #5
	err = store.UpdateConfig(id, 1, fixtures[5].Spec)
	require.NoError(err)

	err = store.RemoveConfig(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 7. forced query failure
	store.MarkConfigSpecForError("SpecifiedError", &fixtures[7].Spec, "GetConfigs")

	_, err = store.CreateConfig(fixtures[7].Spec)
	require.NoError(err)

	_, err = store.GetConfigs(dockerTypes.ConfigListOptions{})
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 8. force failures by manipulating raw datastructures
	id, err = store.CreateConfig(fixtures[8].Spec)
	require.NoError(err)

	rawConfig := store.InternalGetConfig(id)
	store.MarkConfigSpecForError("SpecifiedError", &rawConfig.Spec)

	err = store.RemoveConfig(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	_, err = store.GetConfig(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	err = store.UpdateConfig(id, 1, fixtures[0].Spec)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// Perform a little raw API test coverage
	pointer := store.InternalDeleteConfig(id)
	require.True(pointer == rawConfig)

	pointer = store.InternalDeleteConfig(id)
	require.Nil(pointer)

	pointer = store.InternalGetConfig(id)
	require.Nil(pointer)

}

func TestCRDFakeConfigStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeConfigStore()

	// Assert the store is empty
	_, filterErr := store.GetConfigs(dockerTypes.ConfigListOptions{
		Filters: filters.NewArgs(filters.Arg("a", "b")),
	})
	require.Error(filterErr)

	configs, err := store.GetConfigs(dockerTypes.ConfigListOptions{
		Filters: filters.NewArgs(interfaces.StackLabelArg("Testing123")),
	})
	require.NoError(err)
	require.Empty(configs)

	config, err := store.GetConfig("doesntexist")
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(config)

	// Add three items
	fixtures := GenerateConfigFixtures(4, "TestCRDFakeConfigStore")
	for i := 0; i < 3; i++ {
		id, err := store.CreateConfig(fixtures[i].Spec)
		require.NoError(err, fmt.Sprintf("failed to add fixture %d", i))
		require.NotNil(id)
	}

	// Assert we can list the three items and fetch them individually
	configs, err = store.GetConfigs(dockerTypes.ConfigListOptions{})
	require.NoError(err)
	require.NotNil(configs)
	require.Len(configs, 3)

	found := make(map[string]struct{})
	for _, config := range configs {
		found[config.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, id := range []string{"CFG_1", "CFG_2", "CFG_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		config, err = store.GetConfig(id)
		require.NoError(err)
		require.Equal(config.ID, id)

		// special test feature
		config, err = store.GetConfig(config.Spec.Annotations.Name)
		require.NoError(err)
		require.Equal(config.ID, id)
	}

	// Assert that the StackLabels on even specs are found
	configsFilter, errFilter := store.GetConfigs(dockerTypes.ConfigListOptions{
		Filters: filters.NewArgs(interfaces.StackLabelArg("TestCRDFakeConfigStore")),
	})
	require.NoError(errFilter)
	require.Len(configsFilter, 2)

	// Remove second config
	require.NoError(store.RemoveConfig(configs[1].ID))

	// Remove second config again
	require.Error(store.RemoveConfig(configs[1].ID))

	configsPointers := store.InternalQueryConfigs(nil)
	require.NotEmpty(configsPointers)

	idFunction := func(i *swarm.Config) interface{} { return i }
	configsPointers = store.InternalQueryConfigs(idFunction)
	require.NotEmpty(configsPointers)

	// Assert we can list the two items and fetch them individually
	configs2, err2 := store.GetConfigs(dockerTypes.ConfigListOptions{})
	require.NoError(err2)
	require.NotNil(configs2)
	require.Len(configs2, 2)

	for _, id := range []string{"CFG_1", "CFG_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		config, err = store.GetConfig(id)
		require.NoError(err)
		require.Equal(config.ID, id)
	}

	// Add a new config
	id, err := store.CreateConfig(fixtures[3].Spec)
	require.NoError(err)
	require.NotNil(id)

	// Ensure that the deleted config is not present
	config, err = store.GetConfig(configs[1].ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))

	// Ensure the expected list of configs is present
	configs, err = store.GetConfigs(dockerTypes.ConfigListOptions{})
	require.NoError(err)
	require.NotNil(configs)
	require.Len(configs, 3)

	found = make(map[string]struct{})
	for _, config := range configs {
		found[config.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, name := range []string{"CFG_1", "CFG_3", "CFG_4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}
}
