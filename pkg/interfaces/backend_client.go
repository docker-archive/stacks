package interfaces

import (
	"time"

	"github.com/docker/docker/api/server/router/network"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
)

// BackendClient is the interface used by the Stacks Reconciler to consume
// Docker Events and act upon swarmkit resources. In the engine runtime, it is
// implemented directly by the docker/daemon.Daemon object. In the standalone
// test runtime, the BackendAPIClientShim allows a normal engine API to be used
// in its place.
type BackendClient interface {
	// TODO: StackStore is a temporary interface until generic Stack CRUD
	// operations are made available through the swarm Backend.
	StackStore

	network.ClusterBackend

	// The following operations are a subset of the
	// swarm.Backend interface.
	GetServices(types.ServiceListOptions) ([]swarm.Service, error)
	GetService(idOrName string, insertDefaults bool) (swarm.Service, error)
	CreateService(swarm.ServiceSpec, string, bool) (*types.ServiceCreateResponse, error)
	UpdateService(string, uint64, swarm.ServiceSpec, types.ServiceUpdateOptions, bool) (*types.ServiceUpdateResponse, error)
	RemoveService(string) error
	GetTasks(types.TaskListOptions) ([]swarm.Task, error)
	GetTask(string) (swarm.Task, error)
	GetSecrets(opts types.SecretListOptions) ([]swarm.Secret, error)
	CreateSecret(s swarm.SecretSpec) (string, error)
	RemoveSecret(idOrName string) error
	GetSecret(id string) (swarm.Secret, error)
	UpdateSecret(idOrName string, version uint64, spec swarm.SecretSpec) error
	GetConfigs(opts types.ConfigListOptions) ([]swarm.Config, error)
	CreateConfig(s swarm.ConfigSpec) (string, error)
	RemoveConfig(id string) error
	GetConfig(id string) (swarm.Config, error)
	UpdateConfig(idOrName string, version uint64, spec swarm.ConfigSpec) error

	// SubscribeToEvents and UnsubscribeFromEvents are part of the
	// system.Backend interface
	SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{})
	UnsubscribeFromEvents(chan interface{})
}
