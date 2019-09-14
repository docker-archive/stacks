package fakes

import (
	"fmt"
	"time"

	dockerTypes "github.com/docker/docker/api/types"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/types"
)

// FakeReconcilerClient is a fake implementing the BackendClient interface,
// which is used to test the reconciler.
type FakeReconcilerClient struct {
	FakeStackStore
	FakeServiceStore
	FakeSecretStore
	FakeConfigStore
	FakeNetworkStore
}

// Info call of the SwarmResourceBackend - unused
func (*FakeReconcilerClient) Info() swarm.Info {
	return swarm.Info{}
}

// GetNode calls of the SwarmResourceBackend - unused
func (*FakeReconcilerClient) GetNode(id string) (swarm.Node, error) {
	return swarm.Node{}, FakeUnimplemented
}

// GetTasks calls of the SwarmResourceBackend - unused
func (*FakeReconcilerClient) GetTasks(dockerTypes.TaskListOptions) ([]swarm.Task, error) {
	return []swarm.Task{}, FakeUnimplemented
}

// GetTask calls of the SwarmResourceBackend - unused
func (*FakeReconcilerClient) GetTask(string) (swarm.Task, error) {
	return swarm.Task{}, FakeUnimplemented
}

// SubscribeToEvents subscribes to events - unused
func (*FakeReconcilerClient) SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{}) {
	return nil, nil
}

// UnsubscribeFromEvents subscribes to events - unused
func (*FakeReconcilerClient) UnsubscribeFromEvents(events chan interface{}) {

}

// NewFakeReconcilerClient creates a BackendClient using the
// FIVE fake storage interfaces for stack, service, secret, network, config
func NewFakeReconcilerClient() *FakeReconcilerClient {

	return &FakeReconcilerClient{
		FakeStackStore:   *NewFakeStackStore(),
		FakeServiceStore: *NewFakeServiceStore(),
		FakeSecretStore:  *NewFakeSecretStore(),
		FakeConfigStore:  *NewFakeConfigStore(),
		FakeNetworkStore: *NewFakeNetworkStore(),
	}
}

// CreateStack creates a new stack if the stack is valid.
func (f *FakeReconcilerClient) CreateStack(stackSpec types.StackSpec) (types.StackCreateResponse, error) {
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

// GenerateStackDependencies creates a new stack if the stack is valid.
// nolint: gocyclo
func (f *FakeReconcilerClient) GenerateStackDependencies(stackID string) error {
	snapshot, err := f.FakeStackStore.GetSnapshotStack(stackID)

	if err != nil {
		return err
	}

	stackSpec := CopyStackSpec(snapshot.CurrentSpec)

	for _, secret := range stackSpec.Secrets {
		secret = *CopySecretSpec(secret)
		if secret.Annotations.Labels == nil {
			secret.Annotations.Labels = map[string]string{}
		}
		secret.Annotations.Labels[types.StackLabel] = stackID
		_, secretError := f.CreateSecret(secret)

		if secretError != nil {
			err = secretError
			break
		}
	}
	if err != nil {
		return err
	}

	for _, config := range stackSpec.Configs {
		config = *CopyConfigSpec(config)
		if config.Annotations.Labels == nil {
			config.Annotations.Labels = map[string]string{}
		}
		config.Annotations.Labels[types.StackLabel] = stackID
		_, configError := f.CreateConfig(config)

		if configError != nil {
			err = configError
			break
		}
	}
	if err != nil {
		return err
	}

	for name, network := range stackSpec.Networks {
		network = *CopyNetworkCreate(network)
		if network.Labels == nil {
			network.Labels = map[string]string{}
		}
		network.Labels[types.StackLabel] = stackID
		networkCreate := dockerTypes.NetworkCreateRequest{
			Name:          name,
			NetworkCreate: network,
		}
		_, networkError := f.CreateNetwork(networkCreate)

		if networkError != nil {
			err = networkError
			break
		}
	}
	if err != nil {
		return err
	}

	for _, service := range stackSpec.Services {
		service = *CopyServiceSpec(service)
		if service.Annotations.Labels == nil {
			service.Annotations.Labels = map[string]string{}
		}
		service.Annotations.Labels[types.StackLabel] = stackID
		_, serviceError := f.CreateService(service, "", false)

		if serviceError != nil {
			err = serviceError
			break
		}
	}
	if err != nil {
		return err
	}

	return nil
}
