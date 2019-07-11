package backend

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/mocks"
	"github.com/docker/stacks/pkg/types"
)

func TestStacksBackendUpdateOutOfSequence(t *testing.T) {
	// This test ensures that we cannot globber changes by performing updates
	// with invalid versions.
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	// Create a stack with a valid StackCreate
	response, err := b.CreateStack(types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "teststack",
		},
	})
	require.NoError(err)

	// Inspect the stack
	stack, err := b.GetStack(response.ID)
	require.NoError(err)

	stack.Spec.Annotations.Name = "test1"

	err = b.UpdateStack(stack.ID, stack.Spec, stack.Version.Index)
	require.NoError(err)

	stack.Spec.Annotations.Name = "test2"
	err = b.UpdateStack(stack.ID, stack.Spec, stack.Version.Index)
	require.Error(err)
	require.Contains(err.Error(), "out of sequence")

	stack, err = b.GetStack(stack.ID)
	require.NoError(err)
	require.Equal(stack.Spec.Annotations.Name, "test1")
}

func TestStacksBackendInvalidCreate(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	_, err := b.CreateStack(types.StackSpec{})
	require.Error(err)
	require.Contains(err.Error(), "contains no name")

	// Ensure no stacks were created
	stacks, err := b.ListStacks()
	require.NoError(err)
	require.Empty(stacks)
}

func TestStacksBackendCRUD(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	// Create a stack with a valid StackCreate
	service1Spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "service1",
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "image1",
			},
		},
	}

	service2Spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "service2",
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "image2",
			},
		},
	}

	service3Spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "service3",
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "image3",
			},
		},
	}

	stack1Spec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "stack1",
		},
		Services: []swarm.ServiceSpec{
			service1Spec,
		},
	}

	response, err := b.CreateStack(stack1Spec)
	require.NoError(err)
	require.Equal("1", response.ID)

	// Create another stack
	stack2Spec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "stack2",
		},
		Services: []swarm.ServiceSpec{
			service2Spec,
		},
	}

	response, err = b.CreateStack(stack2Spec)
	require.NoError(err)
	require.Equal("2", response.ID)

	// List both stacks
	stacks, err := b.ListStacks()
	require.NoError(err)
	require.Len(stacks, 2)

	found := map[string]string{
		"service1": "image1",
		"service2": "image2",
	}

	for _, stack := range stacks {
		require.Len(stack.Spec.Services, 1)
		serviceName := stack.Spec.Services[0].Name
		image, ok := found[serviceName]
		require.True(ok)
		require.Equal(image, stack.Spec.Services[0].TaskTemplate.ContainerSpec.Image)
		delete(found, serviceName)
	}
	require.Empty(found)

	// Get a stack by ID
	stack, err := b.GetStack("1")
	require.NoError(err)
	require.True(reflect.DeepEqual(stack.Spec, stack1Spec))
	require.Equal(stack.ID, "1")
	// TODO: require.Equal(stack.Orchestrator, types.OrchestratorSwarm)

	// Update a stack
	stack3Spec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "stack3",
		},
		Services: []swarm.ServiceSpec{
			service3Spec,
		},
	}

	stack2, err := b.GetStack("2")
	require.NoError(err)
	err = b.UpdateStack("2", stack3Spec, stack2.Version.Index)
	require.NoError(err)

	// Get the updated stack by ID
	stack, err = b.GetStack("2")
	require.NoError(err)
	require.True(reflect.DeepEqual(stack.Spec, stack3Spec))
	require.Equal(stack.ID, "2")

	// Remove a stack
	require.NoError(b.DeleteStack("2"))
	_, err = b.GetStack("2")
	require.Error(err)
	require.Contains(err.Error(), "stack not found")
}

// TODO: we need a large variety of tests at this level
func TestStackBackendSwarmSimpleConversion(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	// Create a stack, and retrieve the stored SwarmStack
	stackSpec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: "teststack",
			Labels: map[string]string{
				"key": "value",
			},
		},
		Services: []swarm.ServiceSpec{
			{
				Annotations: swarm.Annotations{
					Name: "test_service",
				},
				//				CapAdd:     []string{"CAP_SYS_ADMIN"},
				//				Privileged: true,
				EndpointSpec: &swarm.EndpointSpec{
					Ports: []swarm.PortConfig{
						{
							TargetPort:    8888,
							PublishedPort: 80,
						},
					},
				},
				TaskTemplate: swarm.TaskSpec{
					ContainerSpec: &swarm.ContainerSpec{
						Image: "testimage",
					},
				},
				// Secrets: []swarm.SecretReference{
				// 	{
				// 		Source: "test_secret1",
				// 	},
				// 	{
				// 		Source: "test_secret2",
				// 	},
				// },
				// Configs: []swarm.SecretReference{
				// 	{
				// 		Source: "test_config1",
				// 	},
				// 	{
				// 		Source: "test_config2",
				// 	},
				// },
			},
		},
		// Configs: map[string]swarm.Config{
		// 	"test_config1": {
		// 		Name: "test_config1",
		// 		External: composeTypes.External{
		// 			External: true,
		// 		},
		// 	},
		// 	"test_config2": {
		// 		Name: "test_config2",
		// 		External: composeTypes.External{
		// 			External: true,
		// 		},
		// 	},
		// },
		// Secrets: map[string]swarm.SecretSpec{
		// 	"test_secret1": {
		// 		Name: "test_secret1",
		// 		External: composeTypes.External{
		// 			External: true,
		// 		},
		// 	},
		// 	"test_secret2": {
		// 		Name: "test_secret2",
		// 		External: composeTypes.External{
		// 			External: true,
		// 		},
		// 	},
		// },
	}

	backendClient.EXPECT().GetSecrets(gomock.Any()).Return([]swarm.Secret{
		{
			Spec: swarm.SecretSpec{
				Annotations: swarm.Annotations{
					Name: "test_secret1",
				},
			},
		},
		{
			Spec: swarm.SecretSpec{
				Annotations: swarm.Annotations{
					Name: "test_secret2",
				},
			},
		},
	}, nil)
	backendClient.EXPECT().GetConfigs(gomock.Any()).Return([]swarm.Config{
		{
			Spec: swarm.ConfigSpec{
				Annotations: swarm.Annotations{
					Name: "test_config1",
				},
			},
		},
		{
			Spec: swarm.ConfigSpec{
				Annotations: swarm.Annotations{
					Name: "test_config2",
				},
			},
		},
	}, nil)

	response, err := b.CreateStack(stackSpec)
	require.NoError(err)

	stack, err := b.GetStack(response.ID)
	require.NoError(err)
	require.Equal(stack.ID, response.ID)
	assert.DeepEqual(t, stack.Spec, stackSpec)
}

/*
func assertServiceEquality(t *testing.T, swarmServiceSpec swarm.ServiceSpec, stackServiceSpec composeTypes.ServiceConfig) {
	assert := tassert.New(t)
	assert.Equal(swarmServiceSpec.Annotations.Name, stackServiceSpec.Name)
	assert.Equal(swarmServiceSpec.TaskTemplate.ContainerSpec.Image, stackServiceSpec.Image)
	assert.Equal(swarmServiceSpec.TaskTemplate.ContainerSpec.Hostname, stackServiceSpec.Hostname)
	assert.Equal(len(swarmServiceSpec.EndpointSpec.Ports), len(stackServiceSpec.Ports))
	for i := 0; i < len(stackServiceSpec.Ports); i++ {
		assert.Equal(stackServiceSpec.Ports[i].Published, swarmServiceSpec.EndpointSpec.Ports[i].PublishedPort)
		assert.Equal(stackServiceSpec.Ports[i].Target, swarmServiceSpec.EndpointSpec.Ports[i].TargetPort)
	}
}

func assertSecretEquality(t *testing.T, swarmSecretSpec swarm.SecretSpec, stackSecretSpec composeTypes.SecretConfig) {
	assert := tassert.New(t)

	assert.Equal(swarmSecretSpec.Name, stackSecretSpec.Name)
	assert.Equal(swarmSecretSpec.Labels, stackSecretSpec.Labels)
}

func assertConfigEquality(t *testing.T, swarmConfigSpec swarm.ConfigSpec, stackConfigSpec composeTypes.ConfigObjConfig) {
	assert := tassert.New(t)

	assert.Equal(swarmConfigSpec.Name, stackConfigSpec.Name)
	assert.Equal(swarmConfigSpec.Labels, stackConfigSpec.Labels)
}

func assertNetworkEquality(t *testing.T, swarmNetworkSpec dockerTypes.NetworkCreate, stackNetworkSpec composeTypes.NetworkConfig) {
	assert := tassert.New(t)

	assert.Equal(swarmNetworkSpec.Driver, stackNetworkSpec.Driver)
	assert.Equal(swarmNetworkSpec.Options, stackNetworkSpec.DriverOpts)
	assert.Equal(swarmNetworkSpec.IPAM.Driver, stackNetworkSpec.Ipam.Driver)
}
*/
