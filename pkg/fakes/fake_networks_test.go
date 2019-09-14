package fakes

import (
	"fmt"
	"reflect"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/stretchr/testify/require"
)

func TestSimpleFakeNetworkStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeNetworkStore()
	store.SpecifyKeyPrefix("TestUpdateFakeNetworkStore")
	store.SpecifyErrorTrigger("TestUpdateFakeNetworkStore", FakeUnimplemented)

	network1 := GetTestNetworkRequest("network1", "driver1")

	id1, err := store.CreateNetwork(network1)
	require.NoError(err)

	anetwork, err := store.GetNetwork(id1)
	require.NoError(err)
	require.Equal(anetwork.ID, id1)
	anetwork.ID = ""
	comparison := store.TransformNetworkCreateRequest(network1)
	require.True(reflect.DeepEqual(anetwork, *comparison))

	// networks have no update, so the matching steps from
	// the other fake tests are not included here

	anetwork, err = store.GetNetwork("123.456")
	require.Error(err)

	// double creation
	_, err = store.CreateNetwork(network1)
	require.True(errdefs.IsInvalidParameter(err))
	require.Error(err)
}

func TestIsolationFakeNetworkStore(t *testing.T) {
	taintKey := "foo"
	taintValue := "bar"
	require := require.New(t)
	store := NewFakeNetworkStore()

	fixtures := GenerateNetworkFixtures(1, "TestIsolationFakeNetworkStore")
	spec := &fixtures[0]

	id, err := store.CreateNetwork(*spec)
	require.NoError(err)
	network1, _ := store.GetNetwork(id)

	// 1. Isolation from creation argument

	require.True(reflect.DeepEqual(spec.NetworkCreate.Labels, network1.Labels))
	spec.Labels[taintKey] = taintValue
	require.False(reflect.DeepEqual(spec.NetworkCreate.Labels, network1.Labels))

	// 2. Isolation between repeated calls to GetNetwork

	networkTaint, taintErr := store.GetNetwork(id)
	require.NoError(taintErr)
	networkTaint.Labels[taintKey] = taintValue

	require.False(reflect.DeepEqual(network1, networkTaint))

}

func TestSpecifiedErrorsFakeNetworkStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeNetworkStore()
	store.SpecifyKeyPrefix("SpecifiedError")
	store.SpecifyErrorTrigger("SpecifiedError", FakeUnimplemented)

	fixtures := GenerateNetworkFixtures(10, "TestSpecifiedErrorsFakeNetworkStore")

	var id string
	var err error

	// 0. Leaving untouched

	// 1. forced creation failure
	store.MarkNetworkCreateForError("SpecifiedError", &fixtures[1].NetworkCreate, "CreateNetwork")

	_, err = store.CreateNetwork(fixtures[1])
	require.True(errdefs.IsUnavailable(err))
	require.Error(err)

	// 2. forced get failure after good create
	store.MarkNetworkCreateForError("SpecifiedError", &fixtures[2].NetworkCreate, "GetNetwork")

	id, err = store.CreateNetwork(fixtures[2])
	require.NoError(err)
	_, err = store.GetNetwork(id)
	require.Error(err)

	// 3. forced update failure using untainted #0
	// UpdateNetwork does not exist

	// 4. acquired update failure using tainted #3
	// UpdateNetwork does not exist

	// 5. forced remove failure
	store.MarkNetworkCreateForError("SpecifiedError", &fixtures[5].NetworkCreate, "RemoveNetwork")

	id, err = store.CreateNetwork(fixtures[5])
	require.NoError(err)

	err = store.RemoveNetwork(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 6. acquired remove failure using tainted #5
	// UpdateNetwork does not exist

	// 7. forced query failure
	store.MarkNetworkCreateForError("SpecifiedError", &fixtures[7].NetworkCreate, "GetNetworks")

	_, err = store.CreateNetwork(fixtures[7])
	require.NoError(err)

	_, err = store.GetNetworks(filters.NewArgs())
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 8. force failures by manipulating raw datastructures
	id, err = store.CreateNetwork(fixtures[8])
	require.NoError(err)

	rawNetwork := store.InternalGetNetwork(id)
	store.MarkNetworkResourceForError("SpecifiedError", rawNetwork)

	err = store.RemoveNetwork(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	_, err = store.GetNetwork(id)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// Perform a little raw API test coverage
	pointer := store.InternalDeleteNetwork(id)
	require.True(pointer == rawNetwork)

	pointer = store.InternalDeleteNetwork(id)
	require.Nil(pointer)

	pointer = store.InternalGetNetwork(id)
	require.Nil(pointer)

	// 9. forced query failure
	store.MarkNetworkCreateForError("SpecifiedError", &fixtures[9].NetworkCreate, "GetNetworksByName")

	id, err = store.CreateNetwork(fixtures[9])
	require.NoError(err)

	getnetwork, geterr := store.GetNetwork(id)
	require.NoError(geterr)

	_, err = store.GetNetworksByName(getnetwork.Name)
	require.Error(err)
	require.True(err == FakeUnimplemented)
}

func TestCRDFakeNetworkStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeNetworkStore()

	// Assert the store is empty
	_, filterErr := store.GetNetworks(filters.NewArgs(filters.Arg("a", "b")))
	require.Error(filterErr)

	networks, err := store.GetNetworks(filters.NewArgs(interfaces.StackLabelArg("Testing123")))
	require.NoError(err)
	require.Empty(networks)

	network, err := store.GetNetwork("doesntexist")
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(network)

	// Add three items
	fixtures := GenerateNetworkFixtures(4, "TestCRDFakeNetworkStore")
	for i := 0; i < 3; i++ {
		id, err := store.CreateNetwork(fixtures[i])
		require.NoError(err, fmt.Sprintf("failed to add fixture %d", i))
		require.NotNil(id)
	}

	// Assert we can list the three items and fetch them individually
	networks, err = store.GetNetworks(filters.NewArgs())
	require.NoError(err)
	require.NotNil(networks)
	require.Len(networks, 3)

	found := make(map[string]struct{})
	for _, network := range networks {
		found[network.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, id := range []string{"NET_1", "NET_2", "NET_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		network, err = store.GetNetwork(id)
		require.NoError(err)
		require.Equal(network.ID, id)

		// special test feature
		network, err = store.GetNetwork(network.Name)
		require.NoError(err)
		require.Equal(network.ID, id)
	}

	// Assert that the StackLabels on even specs are found
	networksFilter, errFilter := store.GetNetworks(filters.NewArgs(interfaces.StackLabelArg("TestCRDFakeNetworkStore")))
	require.NoError(errFilter)
	require.Len(networksFilter, 2)

	// Remove second network
	require.NoError(store.RemoveNetwork(networks[1].ID))

	// Remove second network again
	require.Error(store.RemoveNetwork(networks[1].ID))

	networksPointers := store.InternalQueryNetworks(nil)
	require.NotEmpty(networksPointers)

	idFunction := func(i *dockerTypes.NetworkResource) interface{} { return i }
	networksPointers = store.InternalQueryNetworks(idFunction)
	require.NotEmpty(networksPointers)

	// Assert we can list the two items and fetch them individually
	networks2, err2 := store.GetNetworks(filters.NewArgs())
	require.NoError(err2)
	require.NotNil(networks2)
	require.Len(networks2, 2)

	networks2, err2 = store.GetNetworksByName(networks2[0].Name)
	require.NoError(err2)
	require.NotNil(networks2)
	require.Len(networks2, 1)

	for _, id := range []string{"NET_1", "NET_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		network, err = store.GetNetwork(id)
		require.NoError(err)
		require.Equal(network.ID, id)
	}

	// Add a new network
	id, err := store.CreateNetwork(fixtures[3])
	require.NoError(err)
	require.NotNil(id)

	// Ensure that the deleted network is not present
	network, err = store.GetNetwork(networks[1].ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))

	// Ensure the expected list of networks is present
	networks, err = store.GetNetworks(filters.NewArgs())
	require.NoError(err)
	require.NotNil(networks)
	require.Len(networks, 3)

	found = make(map[string]struct{})
	for _, network := range networks {
		found[network.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, name := range []string{"NET_1", "NET_3", "NET_4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}
}
