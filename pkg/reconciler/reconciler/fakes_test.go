package reconciler

// this file contains fakes used to test the reconciler

import (
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/controller/backend"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

// fakeReconcilerClient is a fake implementing the ReconcilerClient interface,
// which is used to test the reconciler. fakeReconcilerClient only implements
// the strict subset of features needed to make the reconciler go. Most
// notably, it has a half-ass implementation of Filters that only works for
// stack ID labels.
type fakeReconcilerClient struct {
	backend.DefaultStacksBackend

	// alias to fake store features
	FakeStackStore interfaces.FakeStackStoreAPI

	// alias to fake services features
	FakeServiceStore *interfaces.FakeServiceStore

	// alias to fake secrets features
	FakeSecretStore interfaces.FakeFeatures

	// alias to fake configs features
	FakeConfigStore interfaces.FakeFeatures

	// alias to fake networks features
	FakeNetworkStore interfaces.FakeFeatures
}

type fakeSwarmBackend struct {
	*interfaces.FakeServiceStore
	*interfaces.FakeSecretStore
	*interfaces.FakeConfigStore
	*interfaces.FakeNetworkStore
}

func (*fakeSwarmBackend) Info() swarm.Info {
	return swarm.Info{}
}

func (*fakeSwarmBackend) GetNode(id string) (swarm.Node, error) {
	return swarm.Node{}, interfaces.FakeUnimplemented
}

func (*fakeSwarmBackend) GetTasks(dockerTypes.TaskListOptions) ([]swarm.Task, error) {
	return []swarm.Task{}, interfaces.FakeUnimplemented
}

func (*fakeSwarmBackend) GetTask(string) (swarm.Task, error) {
	return swarm.Task{}, interfaces.FakeUnimplemented
}

func newFakeReconcilerClient() *fakeReconcilerClient {

	fakeStacks := interfaces.NewFakeStackStore()
	fakeServices := interfaces.NewFakeServiceStore()
	fakeSecrets := interfaces.NewFakeSecretStore()
	fakeConfigs := interfaces.NewFakeConfigStore()
	fakeNetworks := interfaces.NewFakeNetworkStore()

	fakeBackend := fakeSwarmBackend{
		FakeServiceStore: fakeServices,
		FakeSecretStore:  fakeSecrets,
		FakeConfigStore:  fakeConfigs,
		FakeNetworkStore: fakeNetworks,
	}

	defaultBackend := backend.NewDefaultStacksBackend(fakeStacks, &fakeBackend)

	return &fakeReconcilerClient{
		DefaultStacksBackend: *defaultBackend,
		FakeStackStore:       fakeStacks,
		FakeServiceStore:     fakeServices,
		FakeSecretStore:      fakeSecrets,
		FakeConfigStore:      fakeConfigs,
		FakeNetworkStore:     fakeNetworks,
	}
}

// CreateStack creates a new stack if the stack is valid.
func (f *fakeReconcilerClient) CreateStack(stackSpec types.StackSpec) (types.StackCreateResponse, error) {
	resp, err := f.DefaultStacksBackend.CreateStack(stackSpec)
	return resp, err
}

// nolint: gocyclo
// GenerateStackDependencies creates a new stack if the stack is valid.
func (f *fakeReconcilerClient) GenerateStackDependencies(stackID string) (interfaces.SnapshotStack, error) {
	snapshot, err := f.FakeStackStore.GetSnapshotStack(stackID)

	if err != nil {
		return interfaces.SnapshotStack{}, err
	}

	stackSpec, _ := interfaces.CopyStackSpec(snapshot.CurrentSpec)
	configs := make([]interfaces.SnapshotResource, len(stackSpec.Configs))
	services := make([]interfaces.SnapshotResource, len(stackSpec.Services))
	secrets := make([]interfaces.SnapshotResource, len(stackSpec.Secrets))
	networks := make([]interfaces.SnapshotResource, len(stackSpec.Networks))

	for index, secret := range stackSpec.Secrets {
		secret, _ = interfaces.CopySecretSpec(secret)
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
		config = *interfaces.CopyConfigSpec(config)
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
		network, _ = interfaces.CopyNetworkCreate(network)
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
		service, _ = interfaces.CopyServiceSpec(service)
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
		CurrentSpec: stackSpec,
		Services:    services,
		Configs:     configs,
		Networks:    networks,
		Secrets:     secrets,
	}

	err = f.FakeStackStore.UpdateSnapshotStack(stackID, result, result.Version.Index)

	// FIXME: adjust version for return
	return result, err
}
