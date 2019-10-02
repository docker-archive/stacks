package interfaces

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/server/router/network"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/types"
)

// StacksBackend is the backend handler for Stacks within the engine.
// It is consumed by the API handlers, and by the Reconciler.
type StacksBackend interface {
	CreateStack(spec types.StackSpec) (types.StackCreateResponse, error)
	GetStack(id string) (types.Stack, error)
	GetSnapshotStack(id string) (SnapshotStack, error)
	ListStacks() ([]types.Stack, error)
	UpdateStack(id string, spec types.StackSpec, version uint64) error
	UpdateSnapshotStack(id string, spec SnapshotStack, version uint64) (SnapshotStack, error)
	DeleteStack(id string) error
}

// SwarmResourceBackend is a subset of the swarm.Backend interface,
// combined with the network.ClusterBackend interface. It includes all
// methods required to validate, provision and update manipulate Swarm
// stacks and their referenced resources.
type SwarmResourceBackend interface {
	// Info isn't actually in the swarm.Backend interface, but it is defined on
	// the Cluster object, which provides the rest of the implementation
	Info() swarm.Info
	GetNode(id string) (swarm.Node, error)
	GetTasks(dockerTypes.TaskListOptions) ([]swarm.Task, error)
	GetTask(string) (swarm.Task, error)
}

// SwarmNetworkBackend is a subset of the swarm.Backend interface,
// combined with the network.ClusterBackend interface. It includes all
// methods required to validate, provision and update manipulate Swarm
// Networks and their referenced resources.
type SwarmNetworkBackend interface {
	network.ClusterBackend
}

// SwarmServiceBackend is a subset of the swarm.Backend interface,
// It includes all methods required to validate, provision and
// update manipulate Swarm Services and their referenced resources.
type SwarmServiceBackend interface {
	GetServices(dockerTypes.ServiceListOptions) ([]swarm.Service, error)
	GetService(idOrName string, insertDefaults bool) (swarm.Service, error)
	CreateService(swarm.ServiceSpec, string, bool) (*dockerTypes.ServiceCreateResponse, error)
	UpdateService(string, uint64, swarm.ServiceSpec, dockerTypes.ServiceUpdateOptions, bool) (*dockerTypes.ServiceUpdateResponse, error)
	RemoveService(string) error
}

// SwarmConfigBackend is a subset of the swarm.Backend interface,
// It includes all methods required to validate, provision and
// update manipulate Swarm Configs and their referenced resources.
type SwarmConfigBackend interface {
	GetConfigs(opts dockerTypes.ConfigListOptions) ([]swarm.Config, error)
	CreateConfig(s swarm.ConfigSpec) (string, error)
	RemoveConfig(id string) error
	GetConfig(id string) (swarm.Config, error)
	UpdateConfig(idOrName string, version uint64, spec swarm.ConfigSpec) error
}

// SwarmSecretBackend is a subset of the swarm.Backend interface,
// It includes all methods required to validate, provision and
// update manipulate Swarm Secrets and their referenced resources.
type SwarmSecretBackend interface {
	GetSecrets(opts dockerTypes.SecretListOptions) ([]swarm.Secret, error)
	CreateSecret(s swarm.SecretSpec) (string, error)
	RemoveSecret(idOrName string) error
	GetSecret(id string) (swarm.Secret, error)
	UpdateSecret(idOrName string, version uint64, spec swarm.SecretSpec) error
}

// BackendClient is the full interface used by the Stacks Reconciler to
// consume Docker Events and act upon swarmkit resources. In the engine
// runtime, it is implemented directly by the docker/daemon.Daemon
// object. In the standalone test runtime, the BackendAPIClientShim
// allows a normal engine API to be used in its place.
type BackendClient interface {
	StacksBackend

	SwarmResourceBackend
	SwarmNetworkBackend
	SwarmConfigBackend
	SwarmSecretBackend
	SwarmServiceBackend

	// SubscribeToEvents and UnsubscribeFromEvents are part of the
	// system.Backend interface.
	SubscribeToEvents(since, until time.Time, ef filters.Args) ([]events.Message, chan interface{})
	UnsubscribeFromEvents(chan interface{})
}

// StackStore defines an interface to an arbitrary store which is able
// to perform CRUD operations for all objects required by the Stacks
// Controller.
type StackStore interface {
	AddStack(types.StackSpec) (string, error)
	UpdateStack(string, types.StackSpec, uint64) error
	UpdateSnapshotStack(string, SnapshotStack, uint64) (SnapshotStack, error)

	DeleteStack(string) error

	GetStack(id string) (types.Stack, error)
	GetSnapshotStack(id string) (SnapshotStack, error)

	ListStacks() ([]types.Stack, error)
}

// SnapshotStack - a stored version of a stack with types.StackSpec and ID's of created Resources
type SnapshotStack struct {
	SnapshotResource
	CurrentSpec types.StackSpec
	Services    []SnapshotResource
	Networks    []SnapshotResource
	Secrets     []SnapshotResource
	Configs     []SnapshotResource
}

// SnapshotResource - identifying information of a created Resource
type SnapshotResource struct {
	ID string
	swarm.Meta
	Name string
}

// StackLabelArg constructs the filters.KeyValuePair for API usage
func StackLabelArg(stackID string) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("%s=%s", types.StackLabel, stackID))
}

// temporary constant arguments in order to track their uses
var (
	DefaultGetServiceArg2    = false
	DefaultCreateServiceArg2 = ""
	DefaultCreateServiceArg3 = false
	DefaultUpdateServiceArg4 = dockerTypes.ServiceUpdateOptions{}
	DefaultUpdateServiceArg5 = false
)

// ReconcileKind is an enumeration the kind of resource held in a types.Stack
type ReconcileKind = string

const (
	// ReconcileConfig indicates that the Resource is swarm.Config
	ReconcileConfig ReconcileKind = events.ConfigEventType

	// ReconcileNetwork indicates that the Resource is dockerTypes.NetworkResource
	ReconcileNetwork ReconcileKind = events.NetworkEventType

	// ReconcileSecret indicates that the Resource is swarm.Secret
	ReconcileSecret ReconcileKind = events.SecretEventType

	// ReconcileService indicates that the Resource is swarm.Service
	ReconcileService ReconcileKind = events.ServiceEventType

	// ReconcileStack indicates that the Resource is types.Stack
	ReconcileStack ReconcileKind = types.StackEventType
)

// ReconcileKinds maps all the ReconcileKind enumerations for comparisons
var ReconcileKinds = map[ReconcileKind]struct{}{ReconcileStack: {}, ReconcileNetwork: {}, ReconcileSecret: {}, ReconcileConfig: {}, ReconcileService: {}}

// ReconcileState is an enumeration for reconciliation operations
type ReconcileState string

const (
	// ReconcileSkip defines the ReconcileState for Skipping
	ReconcileSkip ReconcileState = "SKIP"

	// ReconcileCreate defines the ReconcileState for Creating
	ReconcileCreate ReconcileState = "CREATE"

	// ReconcileCompare defines the ReconcileState for Compare
	ReconcileCompare ReconcileState = "COMPARE"

	// ReconcileSame defines the ReconcileState for No operation
	ReconcileSame ReconcileState = "SAME"

	// ReconcileUpdate defines the ReconcileState for Update
	ReconcileUpdate ReconcileState = "UPDATE"

	// ReconcileDelete defines the ReconcileState for Delete
	ReconcileDelete ReconcileState = "DELETE"
)

// ReconcileResource is part of the reconciliation datastructure for Stack resources
type ReconcileResource struct {
	SnapshotResource
	Mark    ReconcileState
	Kind    ReconcileKind
	StackID string
	Config  interface{}
}
