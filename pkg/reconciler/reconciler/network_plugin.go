package reconciler

import (
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

type activeNetwork struct {
	interfaces.SnapshotResource
	network dockerTypes.NetworkResource
	stackID string
}

type initializationNetwork struct {
	cli interfaces.BackendClient
}

type algorithmNetwork struct {
	initializationNetwork
	requestedResource *interfaces.ReconcileResource
	stackID           string
	stackSpec         types.StackSpec
	goals             map[string]*interfaces.ReconcileResource
}

func (a activeNetwork) getSnapshot() interfaces.SnapshotResource {
	return a.SnapshotResource
}

func (a activeNetwork) getStackID() string {
	return a.stackID
}

func (i initializationNetwork) getActiveResource(resource interfaces.ReconcileResource) (activeResource, error) {
	network, err := i.cli.GetNetwork(resource.ID)
	if err != nil {
		return activeNetwork{}, err
	}
	return i.wrapNetwork(network), nil
}

func (i initializationNetwork) getSnapshotResourceNames(snapshot interfaces.SnapshotStack) []string {
	result := make([]string, 0, len(snapshot.Networks))
	for _, snapshotResource := range snapshot.Networks {
		result = append(result, snapshotResource.Name)
	}
	return result
}

func (i initializationNetwork) wrapNetwork(network dockerTypes.NetworkResource) activeResource {
	stackID, ok := network.Labels[types.StackLabel]
	if !ok {
		stackID = ""
	}

	return activeNetwork{
		SnapshotResource: interfaces.SnapshotResource{
			ID:   network.ID,
			Name: network.Name,
		},
		network: network,
		stackID: stackID,
	}
}

func (i initializationNetwork) getKind() interfaces.ReconcileKind {
	return interfaces.ReconcileNetwork
}

func (i initializationNetwork) createPlugin(snapshot interfaces.SnapshotStack, requestedResource *interfaces.ReconcileResource) algorithmPlugin {
	return newAlgorithmPluginNetwork(i, snapshot, requestedResource)
}

func newInitializationSupportNetwork(cli interfaces.BackendClient) initializationNetwork {
	return initializationNetwork{
		cli: cli,
	}
}

func newAlgorithmPluginNetwork(networkInit initializationNetwork, snapshot interfaces.SnapshotStack, requestedResource *interfaces.ReconcileResource) *algorithmNetwork {
	result := algorithmNetwork{
		requestedResource:     requestedResource,
		initializationNetwork: networkInit,
		stackID:               snapshot.ID,
		stackSpec:             snapshot.CurrentSpec,
		goals:                 map[string]*interfaces.ReconcileResource{},
	}

	for _, resource := range snapshot.Networks {
		result.goals[resource.Name] = transform(resource, &result)
	}

	return &result
}

func (a *algorithmNetwork) lookupSpecifiedResource(name string) interface{} {
	return a.lookupNetworkSpec(name)
}

func (a *algorithmNetwork) getRequestedResource() *interfaces.ReconcileResource {
	return a.requestedResource
}

func (a *algorithmNetwork) reconcile(stack interfaces.SnapshotStack) (interfaces.SnapshotStack, error) {
	return reconcileResource(stack, a)
}

func (a *algorithmNetwork) lookupNetworkSpec(name string) *dockerTypes.NetworkCreateRequest {
	for networkName, networkSpec := range a.stackSpec.Networks {
		if name == networkName {
			return &dockerTypes.NetworkCreateRequest{
				Name:          name,
				NetworkCreate: networkSpec,
			}
		}
	}
	return nil
}

func (a *algorithmNetwork) getGoalResources() []*interfaces.ReconcileResource {
	result := make([]*interfaces.ReconcileResource, 0, len(a.goals))
	for _, networkResource := range a.goals {
		result = append(result, networkResource)
	}
	return result
}

func (a *algorithmNetwork) getSpecifiedResourceNames() []string {
	result := make([]string, 0, len(a.stackSpec.Networks))
	for networkName := range a.stackSpec.Networks {
		result = append(result, networkName)
	}
	return result
}

func (a *algorithmNetwork) getActiveResources() ([]activeResource, error) {
	networks, err := a.cli.GetNetworks(stackLabelFilter(a.stackID))
	if err != nil {
		return []activeResource{}, err
	}
	result := make([]activeResource, 0, len(networks))
	for _, network := range networks {
		result = append(result, a.wrapNetwork(network))
	}
	return result, nil
}

func (a *algorithmNetwork) getGoalResource(name string) *interfaces.ReconcileResource {
	for _, networkResource := range a.goals {
		if name == networkResource.Name {
			return networkResource
		}
	}
	return nil
}

func (a *algorithmNetwork) addCreateResourceGoal(specName string) *interfaces.ReconcileResource {
	// returning nil networkCreateRequest will generate a panic but that
	// is a bug in the calling code
	networkCreateRequest := a.lookupNetworkSpec(specName)
	resource := &interfaces.ReconcileResource{
		SnapshotResource: interfaces.SnapshotResource{
			Name: networkCreateRequest.Name,
		},
		Config: networkCreateRequest,
		Kind:   a.getKind(),
	}
	a.goals[specName] = resource
	return resource
}

func (a *algorithmNetwork) addRemoveResourceGoal(activeResource activeResource) *interfaces.ReconcileResource {
	activeNetwork := activeResource.(activeNetwork)
	resource := &interfaces.ReconcileResource{
		SnapshotResource: activeNetwork.SnapshotResource,
		Kind:             a.getKind(),
		Config:           a.lookupNetworkSpec(activeNetwork.Name),
	}
	a.goals[activeNetwork.Name] = resource
	return resource
}

func (a *algorithmNetwork) storeGoals(previous interfaces.SnapshotStack) (interfaces.SnapshotStack, error) {
	goalNetworks := []interfaces.SnapshotResource{}
	for _, resource := range a.goals {
		if resource.Mark == interfaces.ReconcileDelete {
			continue
		}
		goalNetworks = append(goalNetworks, resource.SnapshotResource)
	}

	// Simple copy + override
	updated := previous
	updated.Networks = goalNetworks

	current, err := a.cli.UpdateSnapshotStack(a.stackID,
		updated,
		updated.Meta.Version.Index)
	if err != nil {
		return previous, err
	}

	return current, nil
}

func (a *algorithmNetwork) hasSameConfiguration(resource interfaces.ReconcileResource, actual activeResource) bool {
	// FIXME: Since Networks cannot be updated, is it still useful to
	// compare the original configuration and the current configuration
	/*
		createRequest := resource.Config.(*dockerTypes.NetworkCreateRequest)
		networkResource := actual.(activeNetwork).network
	*/
	return true
}

func (a *algorithmNetwork) createResource(resource *interfaces.ReconcileResource) error {
	networkCreateRequest := resource.Config.(*dockerTypes.NetworkCreateRequest)
	if networkCreateRequest.NetworkCreate.Labels == nil {
		networkCreateRequest.NetworkCreate.Labels = map[string]string{}
	}
	networkCreateRequest.NetworkCreate.Labels[types.StackLabel] = a.stackID
	id, err := a.cli.CreateNetwork(*networkCreateRequest)
	if err != nil {
		return err
	}
	resource.ID = id
	return nil
}

func (a *algorithmNetwork) deleteResource(resource *interfaces.ReconcileResource) error {
	err := a.cli.RemoveNetwork(resource.ID)
	// Ignore not found error
	if err != nil && !errdefs.IsNotFound(err) {
		return err
	}
	resource.ID = ""
	return nil
}

func (a *algorithmNetwork) updateResource(resource interfaces.ReconcileResource) error {
	return nil
}
