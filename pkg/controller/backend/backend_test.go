package backend

import (
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	composeTypes "github.com/docker/stacks/pkg/compose/types"
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
	resp, err := b.CreateStack(types.StackCreate{
		Spec: types.StackSpec{
			Metadata: types.Metadata{
				Name: "teststack",
			},
			Collection: "test1",
		},
		Orchestrator: types.OrchestratorSwarm,
	})
	require.NoError(err)

	// Inspect the stack
	stack, err := b.GetStack(resp.ID)
	require.NoError(err)

	stack.Spec.Collection = "test1"

	err = b.UpdateStack(stack.ID, stack.Spec, stack.Version.Index)
	require.NoError(err)

	stack.Spec.Collection = "test2"
	err = b.UpdateStack(stack.ID, stack.Spec, stack.Version.Index)
	require.Error(err)
	require.Contains(err.Error(), "out of sequence")

	stack, err = b.GetStack(stack.ID)
	require.NoError(err)
	require.Equal(stack.Spec.Collection, "test1")
}

func TestStacksBackendInvalidCreate(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	// Attempt to create a stack with an invalid orchestrator type.
	_, err := b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorNone,
	})
	require.Error(err)
	require.Contains(err.Error(), "invalid orchestrator type")

	_, err = b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorKubernetes,
	})
	require.Error(err)
	require.Contains(err.Error(), "invalid orchestrator type")

	_, err = b.CreateStack(types.StackCreate{
		Orchestrator: "foobar",
	})
	require.Error(err)
	require.Contains(err.Error(), "invalid orchestrator type")

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
	stack1Spec := types.StackSpec{
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service1",
				Image: "image1",
			},
		},
	}

	resp, err := b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorSwarm,
		Spec:         stack1Spec,
	})
	require.NoError(err)
	require.Equal("1", resp.ID)

	// Create another stack
	stack2Spec := types.StackSpec{
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service2",
				Image: "image2",
			},
		},
	}

	resp, err = b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorSwarm,
		Spec:         stack2Spec,
	})
	require.NoError(err)
	require.Equal("2", resp.ID)

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
		require.Equal(image, stack.Spec.Services[0].Image)
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
		Services: []composeTypes.ServiceConfig{
			{
				Name:  "service3",
				Image: "image3",
			},
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
		Metadata: types.Metadata{
			Name: "teststack",
			Labels: map[string]string{
				"key": "value",
			},
		},
		Services: []composeTypes.ServiceConfig{
			{
				Name:       "test_service",
				Image:      "testimage",
				CapAdd:     []string{"CAP_SYS_ADMIN"},
				Privileged: true,
				Ports: []composeTypes.ServicePortConfig{
					{
						Target:    8888,
						Published: 80,
					},
				},
				Secrets: []composeTypes.ServiceSecretConfig{
					{
						Source: "test_secret1",
					},
					{
						Source: "test_secret2",
					},
				},
				Configs: []composeTypes.ServiceConfigObjConfig{
					{
						Source: "test_config1",
					},
					{
						Source: "test_config2",
					},
				},
			},
		},
		Configs: map[string]composeTypes.ConfigObjConfig{
			"test_config1": {
				Name: "test_config1",
				External: composeTypes.External{
					External: true,
				},
			},
			"test_config2": {
				Name: "test_config2",
				External: composeTypes.External{
					External: true,
				},
			},
		},
		Secrets: map[string]composeTypes.SecretConfig{
			"test_secret1": {
				Name: "test_secret1",
				External: composeTypes.External{
					External: true,
				},
			},
			"test_secret2": {
				Name: "test_secret2",
				External: composeTypes.External{
					External: true,
				},
			},
		},
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

	resp, err := b.CreateStack(types.StackCreate{
		Orchestrator: types.OrchestratorSwarm,
		Spec:         stackSpec,
	})
	require.NoError(err)

	swarmStack, err := b.GetSwarmStack(resp.ID)
	require.NoError(err)
	require.Equal(swarmStack.ID, resp.ID)

	stack, err := b.GetStack(resp.ID)
	require.NoError(err)
	require.Equal(stack.ID, resp.ID)
	assert.DeepEqual(t, stack.Spec, stackSpec)

	assertStackEquality(t, swarmStack, stack)
}

func assertStackEquality(t *testing.T, swarmStack interfaces.SwarmStack, stack types.Stack) {
	require := require.New(t)

	require.Equal(swarmStack.ID, stack.ID)
	require.Equal(len(swarmStack.Spec.Services), len(stack.Spec.Services))

	assert.DeepEqual(t, swarmStack.Spec.Annotations.Labels, stack.Spec.Metadata.Labels)

	// External secrets should not be populated as part of the SwarmStack. They
	// are expected to be created independently.
	stackSecretCount := 0
	for _, secret := range stack.Spec.Secrets {
		if secret.External.External {
			continue
		}
		stackSecretCount++
	}
	require.Equal(len(swarmStack.Spec.Secrets), stackSecretCount)

	// External configs should not be populated as part of the
	// SwarmStack. They are expected to be created independently.
	stackConfigCount := 0
	for _, config := range stack.Spec.Configs {
		if config.External.External {
			continue
		}
		stackConfigCount++
	}
	require.Equal(len(swarmStack.Spec.Configs), stackConfigCount)

	// If a service does not specify any network, a "default" network
	// will be created during conversion.
	swarmNetworkCount := 0
	for networkName := range swarmStack.Spec.Networks {
		if strings.HasSuffix(networkName, "default") {
			continue
		}
		swarmNetworkCount++
	}
	require.Equal(swarmNetworkCount, len(stack.Spec.Networks))

	for i := 0; i < len(swarmStack.Spec.Services); i++ {
		assertServiceEquality(t, swarmStack.Spec.Services[i], stack.Spec.Services[i])
	}

	for _, secret := range swarmStack.Spec.Secrets {
		stackSecret, ok := stack.Spec.Secrets[secret.Name]
		require.True(ok)
		require.False(stackSecret.External.External)
		assertSecretEquality(t, secret, stackSecret)
	}

	for _, config := range swarmStack.Spec.Configs {
		stackConfig, ok := stack.Spec.Configs[config.Name]
		require.True(ok)
		require.False(stackConfig.External.External)
		assertConfigEquality(t, config, stackConfig)
	}

	for networkName, network := range swarmStack.Spec.Networks {
		if strings.HasSuffix(networkName, "_default") {
			continue
		}

		stackNetwork, ok := stack.Spec.Networks[networkName]
		require.True(ok)
		assertNetworkEquality(t, network, stackNetwork)
	}
}

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
