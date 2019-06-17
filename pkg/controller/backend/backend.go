package backend

import (
	"fmt"

	"github.com/docker/stacks/pkg/interfaces"
)

// DefaultStacksBackend implements the interfaces.StacksBackend interface, which serves as the
// API handler for the Stacks APIs.
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
func (b *DefaultStacksBackend) CreateStack(stackSpec interfaces.StackSpec) (string, error) {

	// FIXME: Structural Preconditions?
	if stackSpec.Annotations.Name == "" {
		return "", fmt.Errorf("StackSpec contains no name")
	}
	stack := interfaces.Stack{
		Spec: stackSpec,
	}

	id, err := b.stackStore.AddStack(stack)
	if err != nil {
		return "", fmt.Errorf("unable to store stack: %s", err)
	}

	return id, err
}

// GetStack retrieves a stack by its ID.
func (b *DefaultStacksBackend) GetStack(id string) (interfaces.Stack, error) {
	stack, err := b.stackStore.GetStack(id)
	if err != nil {
		return interfaces.Stack{}, fmt.Errorf("unable to retrieve stack %s: %s", id, err)
	}

	return stack, err
}

// ListStacks lists all stacks.
func (b *DefaultStacksBackend) ListStacks() ([]interfaces.Stack, error) {
	return b.stackStore.ListStacks()
}

// UpdateStack updates a stack.
func (b *DefaultStacksBackend) UpdateStack(id string, spec interfaces.StackSpec, version uint64) error {
	return b.stackStore.UpdateStack(id, spec, version)
}

// DeleteStack deletes a stack.
func (b *DefaultStacksBackend) DeleteStack(id string) error {
	return b.stackStore.DeleteStack(id)
}

/*
// FIXME:  DELETE
func (b *DefaultStacksBackend) convertToSwarmStackSpec(spec types.StackSpec) (interfaces.StackSpec, error) {
	// Substitute variables with desired property values
	substitutedSpec, err := substitution.DoSubstitution(spec)
	if err != nil {
		return interfaces.StackSpec{}, err
	}

	namespace := convert.NewNamespace(spec.Metadata.Name)

	services, err := convert.Services(namespace, substitutedSpec, b.swarmBackend)
	if err != nil {
		return interfaces.StackSpec{}, fmt.Errorf("failed to convert services : %s", err)
	}

	configs, err := convert.Configs(namespace, substitutedSpec.Configs)
	if err != nil {
		return interfaces.StackSpec{}, fmt.Errorf("failed to convert configs: %s", err)
	}

	secrets, err := convert.Secrets(namespace, substitutedSpec.Secrets)
	if err != nil {
		return interfaces.StackSpec{}, fmt.Errorf("failed to convert secrets: %s", err)
	}

	serviceNetworks := getServicesDeclaredNetworks(substitutedSpec.Services)
	networkCreates, _ := convert.Networks(namespace, substitutedSpec.Networks, serviceNetworks)
	// TODO: validate external networks?

	stackSpec := interfaces.StackSpec{
		Annotations: swarm.Annotations{
			Name:   spec.Metadata.Name,
			Labels: spec.Metadata.Labels,
		},
		Services: services,
		Configs:  configs,
		Secrets:  secrets,
		Networks: networkCreates,
	}

	return stackSpec, nil
}

func getServicesDeclaredNetworks(serviceConfigs []composetypes.ServiceConfig) map[string]struct{} {
	serviceNetworks := map[string]struct{}{}
	for _, serviceConfig := range serviceConfigs {
		if len(serviceConfig.Networks) == 0 {
			serviceNetworks["default"] = struct{}{}
			continue
		}
		for network := range serviceConfig.Networks {
			serviceNetworks[network] = struct{}{}
		}
	}
	return serviceNetworks
}
*/
// TODO: rewrite if needed
/*
func validateExternalNetworks(
	ctx context.Context,
	client dockerclient.NetworkAPIClient,
	externalNetworks []string,
) error {
	for _, networkName := range externalNetworks {
		if !container.NetworkMode(networkName).IsUserDefined() {
			// Networks that are not user defined always exist on all nodes as
			// local-scoped networks, so there's no need to inspect them.
			continue
		}
		network, err := client.NetworkInspect(ctx, networkName, types.NetworkInspectOptions{})
		switch {
		case dockerclient.IsErrNotFound(err):
			return errors.Errorf("network %q is declared as external, but could not be found. You need to create a swarm-scoped network before the stack is deployed", networkName)
		case err != nil:
			return err
		case network.Scope != "swarm":
			return errors.Errorf("network %q is declared as external, but it is not in the right scope: %q instead of \"swarm\"", networkName, network.Scope)
		}
	}
	return nil
}
*/
