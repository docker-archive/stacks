package types

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
)

const (
	// StackEventType is the value of Type in an events.Message for stacks
	StackEventType = "stack"
	// StackLabel is a label on objects indicating the stack that it belongs to
	StackLabel = "com.docker.stacks.stack_id"
)

// Stack represents a Stack with Engine API types.
type Stack struct {
	ID string
	swarm.Meta
	Spec StackSpec
}

// StackSpec represents a StackSpec with Engine API types.
type StackSpec struct {
	Annotations swarm.Annotations
	Services    []swarm.ServiceSpec
	// Networks is a map of name -> types.NetworkCreate. It's like this because
	// Networks don't have a Spec, they're defined in terms of the
	// NetworkCreate type only.
	Networks map[string]types.NetworkCreate
	Secrets  []swarm.SecretSpec
	Configs  []swarm.ConfigSpec
	// There are no "Volumes" in a StackSpec -- Swarm has no concept of
	// volumes
}

// StackCreateOptions is input to the Create operation for a Stack
type StackCreateOptions struct {
	EncodedRegistryAuth string
}

// StackUpdateOptions is input to the Update operation for a Stack
type StackUpdateOptions struct {
	EncodedRegistryAuth string
}

// StackListOptions is input to the List operation for a Stack
type StackListOptions struct {
	Filters filters.Args
}

// Version represents the internal object version.
type Version struct {
	Index uint64 `json:",omitempty"`
}

// StackList is the output for Stack listing
type StackList struct {
	Items []Stack `json:"items"`
}

// OrchestratorChoice This field specifies which orchestrator the stack is deployed on.
type OrchestratorChoice string

const (
	// OrchestratorSwarm defines the OrchestratorChoice valud for Swarm
	OrchestratorSwarm = "swarm"

	// OrchestratorKubernetes defines the OrchestratorChoice valud for Kubernetes
	OrchestratorKubernetes = "kubernetes"

	// OrchestratorNone defines the OrchestratorChoice valud for no orchestrator (basic containers)
	OrchestratorNone = "none"
)

// StackTask This contains a summary of the Stacks task
type StackTask struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Image        string `json:"image"`
	NodeID       string `json:"node_id"`
	DesiredState string `json:"desired_state"`
	CurrentState string `json:"current_state"`
	Err          string `json:"err"`
}

// StackCreateResponse is the response type of the Create Stack
// operation.
type StackCreateResponse struct {
	ID string
}
