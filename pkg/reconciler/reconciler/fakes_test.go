package reconciler

// this file contains fakes used to test the reconciler

import (
	"fmt"
	"time"

	dockerTypes "github.com/docker/docker/api/types"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/fakes"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

// fakeReconcilerClient is a fake implementing the ReconcilerClient interface,
// which is used to test the reconciler. fakeReconcilerClient only implements
// the strict subset of features needed to make the reconciler go. Most
// notably, it has a half-ass implementation of Filters that only works for
// stack ID labels.
type fakeReconcilerClient struct {
	fakes.FakeStackStore
	fakes.FakeServiceStore
	fakes.FakeSecretStore
	fakes.FakeConfigStore
	fakes.FakeNetworkStore
}

func (*fakeReconcilerClient) Info() swarm.Info {
	return swarm.Info{}
}

func (*fakeReconcilerClient) GetNode(id string) (swarm.Node, error) {
	return swarm.Node{}, fakes.FakeUnimplemented
}

func (*fakeReconcilerClient) GetTasks(dockerTypes.TaskListOptions) ([]swarm.Task, error) {
	return []swarm.Task{}, fakes.FakeUnimplemented
}

func (*fakeReconcilerClient) GetTask(string) (swarm.Task, error) {
	return swarm.Task{}, fakes.FakeUnimplemented
}

func (*fakeReconcilerClient) SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{}) {
	return nil, nil
}

func (*fakeReconcilerClient) UnsubscribeFromEvents(events chan interface{}) {

}

func newFakeReconcilerClient() *fakeReconcilerClient {

	return &fakeReconcilerClient{
		FakeStackStore:   *fakes.NewFakeStackStore(),
		FakeServiceStore: *fakes.NewFakeServiceStore(),
		FakeSecretStore:  *fakes.NewFakeSecretStore(),
		FakeConfigStore:  *fakes.NewFakeConfigStore(),
		FakeNetworkStore: *fakes.NewFakeNetworkStore(),
	}
}

// CreateStack creates a new stack if the stack is valid.
func (f *fakeReconcilerClient) CreateStack(stackSpec types.StackSpec) (types.StackCreateResponse, error) {
	if stackSpec.Annotations.Name == "" {
		return types.StackCreateResponse{}, fmt.Errorf("StackSpec contains no name")
	}

	id, err := f.FakeStackStore.AddStack(stackSpec)
	if err != nil {
		return types.StackCreateResponse{}, fmt.Errorf("unable to store stack: %s", err)
	}

	return types.StackCreateResponse{
		ID: id,
	}, err
}

// nolint: gocyclo
// GenerateStackDependencies creates a new stack if the stack is valid.
func (f *fakeReconcilerClient) GenerateStackDependencies(stackID string) (interfaces.SnapshotStack, error) {
	snapshot, err := f.FakeStackStore.GetSnapshotStack(stackID)

	if err != nil {
		return interfaces.SnapshotStack{}, err
	}

	stackSpec := fakes.CopyStackSpec(snapshot.CurrentSpec)
	configs := make([]interfaces.SnapshotResource, len(stackSpec.Configs))
	services := make([]interfaces.SnapshotResource, len(stackSpec.Services))
	secrets := make([]interfaces.SnapshotResource, len(stackSpec.Secrets))
	networks := make([]interfaces.SnapshotResource, len(stackSpec.Networks))

	for index, secret := range stackSpec.Secrets {
		secret = *fakes.CopySecretSpec(secret)
		if secret.Annotations.Labels == nil {
			secret.Annotations.Labels = map[string]string{}
		}
		secret.Annotations.Labels[types.StackLabel] = stackID
		secretResp, secretError := f.CreateSecret(secret)

		if secretError != nil {
			err = secretError
			break
		}
		secrets[index] = interfaces.SnapshotResource{
			ID:   secretResp,
			Name: secret.Annotations.Name,
		}
	}
	if err != nil {
		return interfaces.SnapshotStack{}, err
	}

	for index, config := range stackSpec.Configs {
		config = *fakes.CopyConfigSpec(config)
		if config.Annotations.Labels == nil {
			config.Annotations.Labels = map[string]string{}
		}
		config.Annotations.Labels[types.StackLabel] = stackID
		configResp, configError := f.CreateConfig(config)

		if configError != nil {
			err = configError
			break
		}
		configs[index] = interfaces.SnapshotResource{
			ID:   configResp,
			Name: config.Annotations.Name,
		}
	}
	if err != nil {
		return interfaces.SnapshotStack{}, err
	}

	for name, network := range stackSpec.Networks {
		network = *fakes.CopyNetworkCreate(network)
		if network.Labels == nil {
			network.Labels = map[string]string{}
		}
		network.Labels[types.StackLabel] = stackID
		networkCreate := dockerTypes.NetworkCreateRequest{
			Name:          name,
			NetworkCreate: network,
		}
		networkResp, networkError := f.CreateNetwork(networkCreate)

		if networkError != nil {
			err = networkError
			break
		}
		networks = append(networks, interfaces.SnapshotResource{
			ID:   networkResp,
			Name: name,
		})
	}
	if err != nil {
		return interfaces.SnapshotStack{}, err
	}

	for index, service := range stackSpec.Services {
		service = *fakes.CopyServiceSpec(service)
		if service.Annotations.Labels == nil {
			service.Annotations.Labels = map[string]string{}
		}
		service.Annotations.Labels[types.StackLabel] = stackID
		serviceResp, serviceError := f.CreateService(service, "", false)

		if serviceError != nil {
			err = serviceError
			break
		}
		services[index] = interfaces.SnapshotResource{
			ID:   serviceResp.ID,
			Name: service.Annotations.Name,
		}
	}
	if err != nil {
		return interfaces.SnapshotStack{}, err
	}

	result := interfaces.SnapshotStack{
		SnapshotResource: interfaces.SnapshotResource{
			ID:   snapshot.ID,
			Meta: snapshot.Meta,
			Name: snapshot.Name,
		},
		CurrentSpec: *stackSpec,
		Services:    services,
		Configs:     configs,
		Networks:    networks,
		Secrets:     secrets,
	}

	err = f.FakeStackStore.UpdateSnapshotStack(stackID, result, result.Version.Index)

	// FIXME: adjust version for return
	return result, err
}
