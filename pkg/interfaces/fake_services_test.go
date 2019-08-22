package interfaces

import (
	"fmt"
	"reflect"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/stretchr/testify/require"
)

func generateServiceFixtures(n int) []swarm.Service {
	fixtures := make([]swarm.Service, n)
	var i int
	for i < n {
		fixtures[i] = swarm.Service{
			Spec: getTestServiceSpecs(fmt.Sprintf("%dservice", i), fmt.Sprintf("%dimage", i)),
		}
		i = i + 1
	}
	return fixtures
}

func getTestServiceSpecs(name, image string) swarm.ServiceSpec {

	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: image,
			},
		},
	}

	return spec
}

func getTestServices(name, image string) swarm.Service {
	serviceSpec := getTestServiceSpecs(name, image)
	return swarm.Service{
		Spec: serviceSpec,
	}
}

func TestUpdateFakeServiceStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeServiceStore()

	service1 := getTestServices("service1", "image1")
	service2 := getTestServices("service2", "image2")

	resp, err := store.CreateService(service1.Spec,
		DefaultCreateServiceArg2,
		DefaultCreateServiceArg3)
	require.NoError(err)
	id := resp.ID

	service, err := store.GetService(id, DefaultGetServiceArg2)
	require.NoError(err)
	require.Equal(service.ID, id)
	require.True(reflect.DeepEqual(service.Spec, service1.Spec))

	_, updateErr :=
		store.UpdateService(id, service.Version.Index, service2.Spec,
			dockerTypes.ServiceUpdateOptions{}, false)
	require.NoError(updateErr)

	service, err = store.GetService(id, DefaultGetServiceArg2)
	require.NoError(err)
	require.Equal(service.ID, id)
	require.True(reflect.DeepEqual(service.Spec, service2.Spec))

}

func TestCRDFakeServiceStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeServiceStore()

	// Assert the store is empty
	services, err := store.GetServices(dockerTypes.ServiceListOptions{})
	require.NoError(err)
	require.Empty(services)

	service, err := store.GetService("doesntexist", DefaultGetServiceArg2)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(service)

	// Add three items
	fixtures := generateServiceFixtures(4)
	for i := 0; i < 3; i++ {
		resp, err := store.CreateService(fixtures[i].Spec,
			DefaultCreateServiceArg2,
			DefaultCreateServiceArg3)
		require.NoError(err, fmt.Sprintf("failed to add fixture %d", i))
		id := resp.ID
		require.NotNil(id)
	}

	// Assert we can list the three items and fetch them individually
	services, err = store.GetServices(dockerTypes.ServiceListOptions{})
	require.NoError(err)
	require.NotNil(services)
	require.Len(services, 3)

	found := make(map[string]struct{})
	for _, service := range services {
		found[service.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, id := range []string{"id_service_1", "id_service_2", "id_service_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		service, err = store.GetService(id, DefaultGetServiceArg2)
		require.NoError(err)
		require.Equal(service.ID, id)
	}

	// Remove second service
	require.NoError(store.RemoveService(services[1].ID))

	// Assert we can list the three items and fetch them individually
	services2, err2 := store.GetServices(dockerTypes.ServiceListOptions{})
	require.NoError(err2)
	require.NotNil(services2)
	require.Len(services2, 2)

	for _, id := range []string{"id_service_1", "id_service_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		service, err = store.GetService(id, DefaultGetServiceArg2)
		require.NoError(err)
		require.Equal(service.ID, id)
	}

	// Add a new service
	resp, err := store.CreateService(fixtures[3].Spec,
		DefaultCreateServiceArg2,
		DefaultCreateServiceArg3)
	require.NoError(err)
	id := resp.ID
	require.NotNil(id)

	// Ensure that the deleted service is not present
	service, err = store.GetService(services[1].ID, DefaultGetServiceArg2)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))

	// Ensure the expected list of services is present
	services, err = store.GetServices(dockerTypes.ServiceListOptions{})
	require.NoError(err)
	require.NotNil(services)
	require.Len(services, 3)

	found = make(map[string]struct{})
	for _, service := range services {
		found[service.ID] = struct{}{}
	}
	require.Len(found, 3)

	for _, name := range []string{"id_service_1", "id_service_3", "id_service_4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}
}
