package router

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/stacks/pkg/client/fake"

	"github.com/docker/stacks/pkg/types"
)

var swarmStackCreate = types.StackSpec{
	Annotations: swarm.Annotations{
		Name: "swarm-stack",
	},
	Services: []swarm.ServiceSpec{
		{
			Annotations: swarm.Annotations{
				Name: "testservice",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image: "testimage",
				},
			},
		},
	},
}

var kubeStackCreate = types.StackSpec{
	Annotations: swarm.Annotations{
		Name: "kube-stack",
	},
	Services: []swarm.ServiceSpec{
		{
			Annotations: swarm.Annotations{
				Name: "testservice",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image: "testimage",
				},
			},
		},
	},
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
	newSpec.Services[0].TaskTemplate.ContainerSpec.Image = "newimage"

	require.NoError(router.StackUpdate(ctx, swarmResp.ID, types.Version{Index: stack.Version.Index}, newSpec, types.StackUpdateOptions{}))

	// A second update over the same version should trigger an "update out of sequence" error
	err = router.StackUpdate(ctx, swarmResp.ID, types.Version{Index: stack.Version.Index}, newSpec, types.StackUpdateOptions{})
	require.Error(err)
	require.Contains(err.Error(), "update out of sequence")

}
