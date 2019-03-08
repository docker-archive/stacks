package router

import (
	"context"
	"testing"

	"github.com/docker/docker/errdefs"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/docker/stacks/pkg/client/fake"
	composeTypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"
)

var baseSpec = types.StackSpec{
	Metadata: types.Metadata{
		Name: "swarm-stack",
	},
	Services: []composeTypes.ServiceConfig{
		{
			Name:  "testservice",
			Image: "testimage",
		},
	},
}

var swarmStackCreate = types.StackCreate{
	Orchestrator: types.OrchestratorSwarm,
	Spec:         baseSpec,
}

var kubeStackCreate = types.StackCreate{
	Orchestrator: types.OrchestratorKubernetes,
	Spec:         baseSpec,
}

func TestUpdateNotFound(t *testing.T) {
	// Update operations should return a NotFound error for non-existent stacks
	router := NewStacksRouter()
	swarmBackend := fake.NewStackClient()
	router.RegisterBackend(types.OrchestratorSwarm, swarmBackend)
	err := router.StackUpdate(context.Background(), "nosuchid", types.Version{}, types.StackSpec{}, types.StackUpdateOptions{})
	require.Error(t, err)
	require.True(t, errdefs.IsNotFound(err))
}

func TestRouterMultipleBackendsUpdate(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	router := NewStacksRouter()
	swarmBackend := fake.NewStackClient()
	kubeBackend := fake.NewStackClient(fake.WithStartingID(5000)) // Set a separate starting ID to avoid ID conflicts

	router.RegisterBackend(types.OrchestratorSwarm, swarmBackend)
	router.RegisterBackend(types.OrchestratorKubernetes, kubeBackend)

	// Create a swarm stack.
	swarmResp, err := router.StackCreate(ctx, swarmStackCreate, types.StackCreateOptions{})
	require.NoError(err)
	require.NotEmpty(swarmResp.ID)

	// Create a kube stack.
	kubeResp, err := router.StackCreate(ctx, kubeStackCreate, types.StackCreateOptions{})
	require.NoError(err)
	require.NotEmpty(kubeResp.ID)

	// Update the swarm stack.
	stack, err := router.StackInspect(ctx, swarmResp.ID)
	require.NoError(err)
	newSpec := stack.Spec
	newSpec.Services[0].Image = "newimage"

	require.NoError(router.StackUpdate(ctx, swarmResp.ID, stack.Version, newSpec, types.StackUpdateOptions{}))

	// A second update over the same version should trigger an "update out of sequence" error
	err = router.StackUpdate(ctx, swarmResp.ID, stack.Version, newSpec, types.StackUpdateOptions{})
	require.Error(err)
	require.Contains(err.Error(), "update out of sequence")

}

func TestRouterMultipleBackendsCRD(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	router := NewStacksRouter()
	swarmBackend := fake.NewStackClient()
	kubeBackend := fake.NewStackClient(fake.WithStartingID(5000)) // Set a separate starting ID to avoid ID conflicts

	router.RegisterBackend(types.OrchestratorSwarm, swarmBackend)
	router.RegisterBackend(types.OrchestratorKubernetes, kubeBackend)

	// Create a swarm stack.
	swarmResp, err := router.StackCreate(ctx, swarmStackCreate, types.StackCreateOptions{})
	require.NoError(err)
	require.NotEmpty(swarmResp.ID)

	// Ensure it's created via the router API.
	stack, err := router.StackInspect(ctx, swarmResp.ID)
	require.NoError(err)
	assert.DeepEqual(t, stack.Spec, swarmStackCreate.Spec)
	require.Equal(stack.ID, swarmResp.ID)

	stacks, err := router.StackList(ctx, types.StackListOptions{})
	require.NoError(err)
	assert.DeepEqual(t, stacks[0].Spec, swarmStackCreate.Spec)
	require.Equal(stacks[0].ID, swarmResp.ID)

	// Ensure the swarm stack only shows up on the swarm backend.
	stack, err = swarmBackend.StackInspect(ctx, swarmResp.ID)
	require.NoError(err)
	assert.DeepEqual(t, stack.Spec, swarmStackCreate.Spec)
	require.Equal(stacks[0].ID, swarmResp.ID)

	stack, err = kubeBackend.StackInspect(ctx, swarmResp.ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(stack)

	// Now create a kube stack.
	kubeResp, err := router.StackCreate(ctx, kubeStackCreate, types.StackCreateOptions{})
	require.NoError(err)
	require.NotEmpty(kubeResp.ID)

	// Ensure the kube stack only shows up on the kube backend.
	stack, err = kubeBackend.StackInspect(ctx, kubeResp.ID)
	require.NoError(err)
	assert.DeepEqual(t, stack.Spec, kubeStackCreate.Spec)
	require.Equal(stack.ID, kubeResp.ID)

	stack, err = swarmBackend.StackInspect(ctx, kubeResp.ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(stack)

	// We should be able to list both stacks through the router.
	stacks, err = router.StackList(ctx, types.StackListOptions{})
	require.NoError(err)
	require.Len(stacks, 2)

	looking := map[string]types.StackCreate{
		swarmResp.ID: swarmStackCreate,
		kubeResp.ID:  kubeStackCreate,
	}

	for _, stack := range stacks {
		create, ok := looking[stack.ID]
		require.True(ok)
		assert.DeepEqual(t, stack.Spec, create.Spec)
		delete(looking, stack.ID)
	}
	require.Empty(looking)

	// Delete the kube stack via the router.
	require.NoError(router.StackDelete(ctx, kubeResp.ID))

	// Ensure the kube stack was removed
	stacks, err = router.StackList(ctx, types.StackListOptions{})
	require.NoError(err)
	require.Len(stacks, 1)
	require.Equal(stacks[0].ID, swarmResp.ID)

	// The stack does not exist on any backend at this point
	stack, err = router.StackInspect(ctx, kubeResp.ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(stack)

	stack, err = kubeBackend.StackInspect(ctx, kubeResp.ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(stack)

	stack, err = swarmBackend.StackInspect(ctx, kubeResp.ID)
	require.Error(err)
	require.True(errdefs.IsNotFound(err))
	require.Empty(stack)
}
