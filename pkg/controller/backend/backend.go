package backend

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/events"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

// DefaultStacksBackend implements the interfaces.StacksBackend
// interface, which serves as the API handler for the Stacks APIs.
type DefaultStacksBackend struct {
	// stackStore is the underlying CRUD store of stacks.
	stackStore interfaces.StackStore

	// swarmBackend provides access to swarmkit operations on secrets
	// and configs, required for stack validation and conversion.
	swarmBackend interfaces.SwarmResourceBackend
}

// NewDefaultStacksBackend creates a new DefaultStacksBackend.
func NewDefaultStacksBackend(stackStore interfaces.StackStore, swarmBackend interfaces.SwarmResourceBackend) *DefaultStacksBackend {
	return &DefaultStacksBackend{
		stackStore:   stackStore,
		swarmBackend: swarmBackend,
	}
}

// CreateStack creates a new stack if the stack is valid.
func (b *DefaultStacksBackend) CreateStack(stackSpec types.StackSpec) (types.StackCreateResponse, error) {

	if stackSpec.Annotations.Name == "" {
		return types.StackCreateResponse{}, fmt.Errorf("StackSpec contains no name")
	}

	id, err := b.stackStore.AddStack(stackSpec)
	if err != nil {
		return types.StackCreateResponse{}, fmt.Errorf("unable to store stack: %s", err)
	}

	return types.StackCreateResponse{
		ID: id,
	}, err
}

// GetStack retrieves a stack by its ID.
func (b *DefaultStacksBackend) GetStack(id string) (types.Stack, error) {
	stack, err := b.stackStore.GetStack(id)
	if err != nil {
		return types.Stack{}, errdefs.NotFound(fmt.Errorf("unable to retrieve stack %s: %s", id, err))
	}

	return stack, err
}

// ListStacks lists all stacks.
func (b *DefaultStacksBackend) ListStacks() ([]types.Stack, error) {
	return b.stackStore.ListStacks()
}

// UpdateStack updates a stack.
func (b *DefaultStacksBackend) UpdateStack(id string, spec types.StackSpec, version uint64) error {
	return b.stackStore.UpdateStack(id, spec, version)
}

// DeleteStack deletes a stack.
func (b *DefaultStacksBackend) DeleteStack(id string) error {
	return b.stackStore.DeleteStack(id)
}

// GetNetworks forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetNetworks(filter filters.Args) ([]dockerTypes.NetworkResource, error) {
	return b.swarmBackend.GetNetworks(filter)
}

// GetNetwork forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetNetwork(name string) (dockerTypes.NetworkResource, error) {
	return b.swarmBackend.GetNetwork(name)
}

// GetNetworksByName forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetNetworksByName(name string) ([]dockerTypes.NetworkResource, error) {
	return b.swarmBackend.GetNetworksByName(name)
}

// CreateNetwork forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) CreateNetwork(nc dockerTypes.NetworkCreateRequest) (string, error) {
	return b.swarmBackend.CreateNetwork(nc)
}

// RemoveNetwork forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) RemoveNetwork(name string) error {
	return b.swarmBackend.RemoveNetwork(name)
}

// Info forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) Info() swarm.Info {
	return b.swarmBackend.Info()
}

// GetNode forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetNode(id string) (swarm.Node, error) {
	return b.swarmBackend.GetNode(id)
}

// GetServices forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetServices(opts dockerTypes.ServiceListOptions) ([]swarm.Service, error) {
	return b.swarmBackend.GetServices(opts)
}

// GetService forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetService(id string, insertDefaults bool) (swarm.Service, error) {
	return b.swarmBackend.GetService(id, insertDefaults)
}

// CreateService forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) CreateService(spec swarm.ServiceSpec, s string, bo bool) (*dockerTypes.ServiceCreateResponse, error) {
	return b.swarmBackend.CreateService(spec, s, bo)
}

// UpdateService forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) UpdateService(id string, version uint64, spec swarm.ServiceSpec, opts dockerTypes.ServiceUpdateOptions, bo bool) (*dockerTypes.ServiceUpdateResponse, error) {
	return b.swarmBackend.UpdateService(id, version, spec, opts, bo)
}

// RemoveService forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) RemoveService(id string) error {
	return b.swarmBackend.RemoveService(id)
}

// GetTasks forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetTasks(opts dockerTypes.TaskListOptions) ([]swarm.Task, error) {
	return b.swarmBackend.GetTasks(opts)
}

// GetTask forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetTask(id string) (swarm.Task, error) {
	return b.swarmBackend.GetTask(id)
}

// GetSecrets forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetSecrets(opts dockerTypes.SecretListOptions) ([]swarm.Secret, error) {
	return b.swarmBackend.GetSecrets(opts)
}

// CreateSecret forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) CreateSecret(s swarm.SecretSpec) (string, error) {
	return b.swarmBackend.CreateSecret(s)
}

// RemoveSecret forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) RemoveSecret(id string) error {
	return b.swarmBackend.RemoveSecret(id)
}

// GetSecret forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetSecret(id string) (swarm.Secret, error) {
	return b.swarmBackend.GetSecret(id)
}

// UpdateSecret forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) UpdateSecret(id string, version uint64, spec swarm.SecretSpec) error {
	return b.swarmBackend.UpdateSecret(id, version, spec)
}

// GetConfigs forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetConfigs(opts dockerTypes.ConfigListOptions) ([]swarm.Config, error) {
	return b.swarmBackend.GetConfigs(opts)
}

// CreateConfig forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) CreateConfig(s swarm.ConfigSpec) (string, error) {
	return b.swarmBackend.CreateConfig(s)
}

// RemoveConfig forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) RemoveConfig(id string) error {
	return b.swarmBackend.RemoveConfig(id)
}

// GetConfig forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) GetConfig(id string) (swarm.Config, error) {
	return b.swarmBackend.GetConfig(id)
}

// UpdateConfig forwards to the calls to the SwarmResourceBackend
func (b *DefaultStacksBackend) UpdateConfig(id string, version uint64, spec swarm.ConfigSpec) error {
	return b.swarmBackend.UpdateConfig(id, version, spec)
}

// SubscribeToEvents subscribes to events
func (b *DefaultStacksBackend) SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{}) {
	return nil, nil
}

// UnsubscribeFromEvents unsubscribes to events
func (b *DefaultStacksBackend) UnsubscribeFromEvents(events chan interface{}) {

}
