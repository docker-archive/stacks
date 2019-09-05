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
	"github.com/docker/stacks/pkg/types"
	"github.com/stretchr/testify/require"
)

func generateServiceFixtures(n int, label string) []swarm.Service {
	fixtures := make([]swarm.Service, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%dservice", i)
		imageName := fmt.Sprintf("%dimage", i)
		spec := getTestServiceSpec(specName, imageName)
		fixtures[i] = swarm.Service{
			Spec: spec,
		}
		if i%2 == 0 {
			fixtures[i].Spec.Annotations.Labels = make(map[string]string)
			fixtures[i].Spec.Annotations.Labels[types.StackLabel] = label
		}

		i = i + 1
	}
	return fixtures
}

func getTestServiceSpec(name, image string) swarm.ServiceSpec {

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

func getTestService(name, image string) swarm.Service {
	serviceSpec := getTestServiceSpec(name, image)
	return swarm.Service{
		Spec: serviceSpec,
	}
}

func TestUpdateFakeServiceStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeServiceStore()
	store.SpecifyKeyPrefix("TestUpdateFakeServiceStore")
	store.SpecifyErrorTrigger("TestUpdateFakeServiceStore", FakeUnimplemented)

	service1 := getTestService("service1", "image1")
	service2 := getTestService("service2", "image2")

	resp, err := store.CreateService(service1.Spec,
		interfaces.DefaultCreateServiceArg2,
		interfaces.DefaultCreateServiceArg3)
	require.NoError(err)
	id := resp.ID

	aservice, err := store.GetService(id, interfaces.DefaultGetServiceArg2)
	require.NoError(err)
	require.Equal(aservice.ID, id)
	require.True(reflect.DeepEqual(aservice.Spec, service1.Spec))

	_, updateErr :=
		store.UpdateService(id, aservice.Version.Index, service2.Spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.NoError(updateErr)

	// index out of whack
	_, updateErr =
		store.UpdateService(id, aservice.Version.Index, service2.Spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.Error(updateErr)

	// id missing
	_, updateErr =
		store.UpdateService("123.456", aservice.Version.Index, service2.Spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.Error(updateErr)

	aservice, err = store.GetService(id, interfaces.DefaultGetServiceArg2)
	require.NoError(err)
	require.Equal(aservice.ID, id)
	require.True(reflect.DeepEqual(aservice.Spec, service2.Spec))

	aservice, err = store.GetService("123.456", interfaces.DefaultGetServiceArg2)
	require.Error(err)

	// double creation
	_, err = store.CreateService(service1.Spec,
		interfaces.DefaultCreateServiceArg2,
		interfaces.DefaultCreateServiceArg3)
	require.True(errdefs.IsInvalidParameter(err))
	require.Error(err)

}

func TestIsolationFakeServiceStore(t *testing.T) {
	taintKey := "foo"
	taintValue := "bar"
	require := require.New(t)
	store := NewFakeServiceStore()

	fixtures := generateServiceFixtures(1, "TestIsolationFakeServiceStore")
	spec := &fixtures[0].Spec

	resp, err := store.CreateService(*spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.NoError(err)
	service1, _ := store.GetService(resp.ID, interfaces.DefaultGetServiceArg2)

	// 1. Isolation from creation argument

	require.True(reflect.DeepEqual(*spec, service1.Spec))
	spec.Annotations.Labels[taintKey] = taintValue
	require.False(reflect.DeepEqual(*spec, service1.Spec))

	// 2. Isolation between repeated calls to GetService

	serviceTaint, taintErr := store.GetService(resp.ID, interfaces.DefaultGetServiceArg2)
	require.NoError(taintErr)
	serviceTaint.Spec.Annotations.Labels[taintKey] = taintValue

	require.False(reflect.DeepEqual(service1.Spec, serviceTaint.Spec))

	// 3. Isolation from Update argument (using now changed spec)

	_, err = store.UpdateService(resp.ID, 1, *spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.NoError(err)
	serviceUpdated, _ := store.GetService(resp.ID, interfaces.DefaultGetServiceArg2)

	require.True(reflect.DeepEqual(*spec, serviceUpdated.Spec))
	delete(spec.Annotations.Labels, taintKey)
	require.False(reflect.DeepEqual(*spec, serviceUpdated.Spec))

}

func TestSpecifiedErrorsFakeServiceStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeServiceStore()
	store.SpecifyKeyPrefix("SpecifiedError")
	store.SpecifyErrorTrigger("SpecifiedError", FakeUnimplemented)

	fixtures := generateServiceFixtures(10, "TestSpecifiedErrorsFakeServiceStore")

	var err error
	var resp *dockerTypes.ServiceCreateResponse

	// 0. Leaving untouched

	// 1. forced creation failure
	store.MarkServiceSpecForError("SpecifiedError", &fixtures[1].Spec, "CreateService")

	_, err = store.CreateService(fixtures[1].Spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.True(errdefs.IsUnavailable(err))
	require.Error(err)

	// 2. forced get failure after good create
	store.MarkServiceSpecForError("SpecifiedError", &fixtures[2].Spec, "GetService")

	resp, err = store.CreateService(fixtures[2].Spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.NoError(err)
	_, err = store.GetService(resp.ID, interfaces.DefaultGetServiceArg2)
	require.Error(err)

	// 3. forced update failure using untainted #0
	store.MarkServiceSpecForError("SpecifiedError", &fixtures[3].Spec, "UpdateService")

	resp, err = store.CreateService(fixtures[3].Spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.NoError(err)
	_, err = store.GetService(resp.ID, interfaces.DefaultGetServiceArg2)
	require.NoError(err)

	_, err = store.UpdateService(resp.ID, 1, fixtures[0].Spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 4. acquired update failure using tainted #3
	resp, err = store.CreateService(fixtures[4].Spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.NoError(err)

	// normal update using #0
	_, err = store.UpdateService(resp.ID, 1, fixtures[0].Spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.NoError(err)

	// tainted update using tainted #3
	_, err = store.UpdateService(resp.ID, 2, fixtures[3].Spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 5. forced remove failure
	store.MarkServiceSpecForError("SpecifiedError", &fixtures[5].Spec, "RemoveService")

	resp, err = store.CreateService(fixtures[5].Spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.NoError(err)

	err = store.RemoveService(resp.ID)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 6. acquired remove failure using tainted #5
	resp, err = store.CreateService(fixtures[6].Spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.NoError(err)

	// update #6 using tainted #5
	_, err = store.UpdateService(resp.ID, 1, fixtures[5].Spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.NoError(err)

	err = store.RemoveService(resp.ID)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 7. forced query failure
	store.MarkServiceSpecForError("SpecifiedError", &fixtures[7].Spec, "GetServices")

	_, err = store.CreateService(fixtures[7].Spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.NoError(err)

	_, err = store.GetServices(dockerTypes.ServiceListOptions{})
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// 8. force failures by manipulating raw datastructures
	resp, err = store.CreateService(fixtures[8].Spec, interfaces.DefaultCreateServiceArg2, interfaces.DefaultCreateServiceArg3)
	require.NoError(err)

	rawService := store.InternalGetService(resp.ID)
	store.MarkServiceSpecForError("SpecifiedError", &rawService.Spec)

	err = store.RemoveService(resp.ID)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	_, err = store.GetService(resp.ID, interfaces.DefaultGetServiceArg2)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	_, err = store.UpdateService(resp.ID, 1, fixtures[0].Spec, interfaces.DefaultUpdateServiceArg4, interfaces.DefaultUpdateServiceArg5)
	require.Error(err)
	require.True(err == FakeUnimplemented)

	// Perform a little raw API test coverage
	pointer := store.InternalDeleteService(resp.ID)
	require.True(pointer == rawService)

	pointer = store.InternalDeleteService(resp.ID)
	require.Nil(pointer)

	pointer = store.InternalGetService(resp.ID)
	require.Nil(pointer)

}

func TestCRDFakeServiceStore(t *testing.T) {
	require := require.New(t)
	store := NewFakeServiceStore()

	// Assert the store is empty
	_, filterErr := store.GetServices(dockerTypes.ServiceListOptions{
		Filters: filters.NewArgs(filters.Arg("a", "b")),
	})
	require.Error(filterErr)

	services, err := store.GetServices(dockerTypes.ServiceListOptions{
		Filters: filters.NewArgs(interfaces.StackLabelArg("Testing123")),
	})
	require.NoError(err)
	require.Empty(services)

	service, err := store.GetService("doesntexist", interfaces.DefaultGetServiceArg2)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(service)

	// Add three items
	fixtures := generateServiceFixtures(4, "TestCRDFakeServiceStore")
	for i := 0; i < 3; i++ {
		resp, err := store.CreateService(fixtures[i].Spec,
			interfaces.DefaultCreateServiceArg2,
			interfaces.DefaultCreateServiceArg3)
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

	for _, id := range []string{"SVC_1", "SVC_2", "SVC_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		service, err = store.GetService(id, interfaces.DefaultGetServiceArg2)
		require.NoError(err)
		require.Equal(service.ID, id)
	}

	// Remove second service
	require.NoError(store.RemoveService(services[1].ID))

	// Remove second service again
	require.Error(store.RemoveService(services[1].ID))

	servicesPointers := store.InternalQueryServices(nil)
	require.NotEmpty(servicesPointers)

	idFunction := func(i *swarm.Service) interface{} { return i }
	servicesPointers = store.InternalQueryServices(idFunction)
	require.NotEmpty(servicesPointers)

	// Assert we can list the three items and fetch them individually
	services2, err2 := store.GetServices(dockerTypes.ServiceListOptions{})
	require.NoError(err2)
	require.NotNil(services2)
	require.Len(services2, 2)

	for _, id := range []string{"SVC_1", "SVC_3"} {
		require.Contains(found, id, fmt.Sprintf("ID %s not found", id))
		service, err = store.GetService(id, interfaces.DefaultGetServiceArg2)
		require.NoError(err)
		require.Equal(service.ID, id)
	}

	// Add a new service
	resp, err := store.CreateService(fixtures[3].Spec,
		interfaces.DefaultCreateServiceArg2,
		interfaces.DefaultCreateServiceArg3)
	require.NoError(err)
	id := resp.ID
	require.NotNil(id)

	// Ensure that the deleted service is not present
	service, err = store.GetService(services[1].ID, interfaces.DefaultGetServiceArg2)
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

	for _, name := range []string{"SVC_1", "SVC_3", "SVC_4"} {
		require.Contains(found, name, fmt.Sprintf("name %s not found", name))
	}
}
