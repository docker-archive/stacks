package fakes

import (
	"fmt"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/types"
)

// GenerateStackFixtures creates some types.Stack fixtures
// as well as marking the EVEN ones as belonging
// to types.StackLabel
func GenerateStackFixtures(n int, label string) []types.Stack {
	fixtures := make([]types.Stack, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%dstack", i)
		imageName := fmt.Sprintf("%dimage", i)
		spec := GetTestStackSpec(specName, imageName)
		fixtures[i] = types.Stack{
			Spec: spec,
		}
		if i%2 == 0 {
			fixtures[i].Spec.Annotations.Labels = make(map[string]string)
			fixtures[i].Spec.Annotations.Labels[types.StackLabel] = label
		}

		i = i + 1
	}
	return fixtures
}

// GetTestStackSpec creates a minimal types.StackSpec
func GetTestStackSpec(name, image string) types.StackSpec {

	ncr := GetTestNetworkRequest(name, image)

	spec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
		Services: []swarm.ServiceSpec{
			GetTestServiceSpec(name+"service", image),
		},
		Configs: []swarm.ConfigSpec{
			GetTestConfigSpec(name + "config"),
		},
		Secrets: []swarm.SecretSpec{
			GetTestSecretSpec(name + "secret"),
		},
		Networks: map[string]dockerTypes.NetworkCreate{
			ncr.Name + "network": ncr.NetworkCreate,
		},
	}

	return spec
}

// GetTestStack creates a minimal type.Stack
func GetTestStack(name, image string) types.Stack {
	stackSpec := GetTestStackSpec(name, image)
	return types.Stack{
		Spec: stackSpec,
	}
}

// GenerateServiceFixtures creates some swarm.Service fixtures
// as well as marking the EVEN ones as belonging
// to types.StackLabel
func GenerateServiceFixtures(n int, label string) []swarm.Service {
	fixtures := make([]swarm.Service, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%dservice", i)
		imageName := fmt.Sprintf("%dimage", i)
		spec := GetTestServiceSpec(specName, imageName)
		fixtures[i] = swarm.Service{
			Spec: spec,
		}
		if i%2 == 0 {
			fixtures[i].Spec.Annotations.Labels = make(map[string]string)
			fixtures[i].Spec.Annotations.Labels[types.StackLabel] = label
		}

		i = i + 1
	}
	return fixtures
}

// GetTestServiceSpec creates a minimal swarm.ServiceSpec
func GetTestServiceSpec(name, image string) swarm.ServiceSpec {

	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: image,
			},
		},
	}

	return spec
}

// GetTestService creates a minimal swarm.Service
func GetTestService(name, image string) swarm.Service {
	serviceSpec := GetTestServiceSpec(name, image)
	return swarm.Service{
		Spec: serviceSpec,
	}
}

// GenerateConfigFixtures creates some swarm.Config fixtures
// as well as marking the EVEN ones as belonging
// to types.StackLabel
func GenerateConfigFixtures(n int, label string) []swarm.Config {
	fixtures := make([]swarm.Config, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%dconfig", i)
		spec := GetTestConfigSpec(specName)
		fixtures[i] = swarm.Config{
			Spec: spec,
		}
		if i%2 == 0 {
			fixtures[i].Spec.Annotations.Labels = make(map[string]string)
			fixtures[i].Spec.Annotations.Labels[types.StackLabel] = label
		}

		i = i + 1
	}
	return fixtures
}

// GetTestConfigSpec creates a minimal swarm.ConfigSpec
func GetTestConfigSpec(name string) swarm.ConfigSpec {

	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
	}

	return spec
}

// GetTestConfig creates a minimal swarm.Config
func GetTestConfig(name string) swarm.Config {
	configSpec := GetTestConfigSpec(name)
	return swarm.Config{
		Spec: configSpec,
	}
}

// GenerateSecretFixtures creates some swarm.Secret fixtures
// as well as marking the EVEN ones as belonging
// to types.StackLabel
func GenerateSecretFixtures(n int, label string) []swarm.Secret {
	fixtures := make([]swarm.Secret, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%dsecret", i)
		spec := GetTestSecretSpec(specName)
		fixtures[i] = swarm.Secret{
			Spec: spec,
		}
		if i%2 == 0 {
			fixtures[i].Spec.Annotations.Labels = make(map[string]string)
			fixtures[i].Spec.Annotations.Labels[types.StackLabel] = label
		}

		i = i + 1
	}
	return fixtures
}

// GetTestSecretSpec creates a minimal swarm.SecretSpec
func GetTestSecretSpec(name string) swarm.SecretSpec {

	spec := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
	}

	return spec
}

// GetTestSecret creates a minimal swarm.Secret
func GetTestSecret(name string) swarm.Secret {
	secretSpec := GetTestSecretSpec(name)
	return swarm.Secret{
		Spec: secretSpec,
	}
}

// GenerateNetworkFixtures creates some dockerTypes.NetworkCreateRequest
// fixtures as well as marking the EVEN ones as belonging
// to types.StackLabel
func GenerateNetworkFixtures(n int, label string) []dockerTypes.NetworkCreateRequest {
	fixtures := make([]dockerTypes.NetworkCreateRequest, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%dnetwork", i)
		driverName := fmt.Sprintf("%ddriver", i)
		nr := GetTestNetworkRequest(specName, driverName)
		fixtures[i] = nr
		if i%2 == 0 {
			fixtures[i].NetworkCreate.Labels = make(map[string]string)
			fixtures[i].NetworkCreate.Labels[types.StackLabel] = label
		}

		i = i + 1
	}
	return fixtures
}

// GetTestNetworkRequest creates a minimal dockerTypes.NetworkCreateRequest
func GetTestNetworkRequest(name, driver string) dockerTypes.NetworkCreateRequest {

	networkCreate := dockerTypes.NetworkCreate{
		Driver:     driver,
		IPAM:       &network.IPAM{},
		ConfigFrom: &network.ConfigReference{},
	}

	return dockerTypes.NetworkCreateRequest{
		Name:          name,
		NetworkCreate: networkCreate,
	}
}
