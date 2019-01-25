package types

import (
	"github.com/docker/stacks/pkg/compose/types"
)

// Stack represents a runtime instantiation of a Docker Compose based application
type Stack struct {
	Spec           StackSpec          `json:"spec"`
	StackResources StackResources     `json:"stack_resources"`
	Orchestrator   OrchestratorChoice `json:"orchestrator"`
	Status         StackStatus        `json:"stack_status"`

	// TODO - temporary (not in swagger)
	ID string
}

// StackCreate is input to the Create operation for a Stack
type StackCreate struct {
	Spec         StackSpec          `json:"spec"`
	Orchestrator OrchestratorChoice `json:"orchestrator"`
}

// StackList is the output for Stack listing
type StackList struct {
	Items []Stack `json:"items"`
}

// StackSpec defines the desired state of Stack
type StackSpec struct {
	Services       []types.ServiceConfig            `json:"services,omitempty"`
	Secrets        map[string]types.SecretConfig    `json:"secrets,omitempty"`
	Configs        map[string]types.ConfigObjConfig `json:"configs,omitempty"`
	Networks       []types.NetworkConfig            `json:"networks,omitempty"`
	Volumes        []types.VolumeConfig             `json:"volumes,omitempty"`
	StackImage     string                           `json:"stack_image,omitempty"`
	PropertyValues []string                         `json:"property_values,omitempty"`
	Collection     string                           `json:"collection,omitempty"`
}

// StackResources links to the running instances of the StackSpec
type StackResources struct {
	Services []StackResource `json:"services:omitempty"`
	Configs  []StackResource `json:"configs:omitempty"`
	Secrets  []StackResource `json:"secrets:omitempty"`
	Networks []StackResource `json:"networks:omitempty"`
	Volumes  []StackResource `json:"volumes:omitempty"`
}

// StackResource contains a link to a single instance of the spec
// For example, when a Service is run on basic containers, the ID would
// contain the container ID.  When the Service is running on Swarm the ID would be
// a Swarm Service ID.  When mapped to kubernetes, it would map to a Deployment or
// DaemonSet ID.
type StackResource struct {
	Orchestrator OrchestratorChoice `json:"orchestrator"`
	Kind         string             `json:"kind"`
	ID           string             `json:"id"`
}

// StackStatus defines the observed state of Stack
type StackStatus struct {
	Message       string `json:"message"`
	Phase         string `json:"phase"`
	OverallHealth string `json:"overall_health"`
	LastUpdated   string `json:"last_updated"`
}

// StackTaskList contains a summary of the underlying tasks that make up this Stack
type StackTaskList struct {
	CurrentTasks []StackTask `json:"current_tasks"`
	PastTasks    []StackTask `json:"past_tasks"`
}

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

// OrchestratorChoice This field specifies which orchestrator the stack is deployed on.
type OrchestratorChoice string
