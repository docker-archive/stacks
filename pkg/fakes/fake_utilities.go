package fakes

import (
	"fmt"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/types"
)

// InjectStackID adds types.StackLabel to all specs
func InjectStackID(spec *types.StackSpec, stackID string) {

	for i := range spec.Services {
		if spec.Services[i].Annotations.Labels == nil {
			spec.Services[i].Annotations.Labels = map[string]string{}
		}
		spec.Services[i].Annotations.Labels[types.StackLabel] = stackID
	}
	for i := range spec.Configs {
		if spec.Configs[i].Annotations.Labels == nil {
			spec.Configs[i].Annotations.Labels = map[string]string{}
		}
		spec.Configs[i].Annotations.Labels[types.StackLabel] = stackID
	}
	for i := range spec.Secrets {
		if spec.Secrets[i].Annotations.Labels == nil {
			spec.Secrets[i].Annotations.Labels = map[string]string{}
		}
		spec.Secrets[i].Annotations.Labels[types.StackLabel] = stackID
	}
	networks := map[string]dockerTypes.NetworkCreate{}
	for k, v := range spec.Networks {
		if v.Labels == nil {
			v.Labels = map[string]string{}
		}
		v.Labels[types.StackLabel] = stackID
		networks[k] = v
	}
	spec.Networks = networks
}

// InjectForcedRemoveError adds types.StackLabel to all specs
func InjectForcedRemoveError(cli *FakeReconcilerClient, spec *types.StackSpec, removeError string) {

	for i := range spec.Services {
		cli.FakeServiceStore.MarkServiceSpecForError(removeError, &spec.Services[i], "RemoveService")
	}
	for i := range spec.Configs {
		cli.FakeConfigStore.MarkConfigSpecForError(removeError, &spec.Configs[i], "RemoveConfig")
	}
	for i := range spec.Secrets {
		cli.FakeSecretStore.MarkSecretSpecForError(removeError, &spec.Secrets[i], "RemoveSecret")
	}
	networks := map[string]dockerTypes.NetworkCreate{}
	for k, v := range spec.Networks {
		cli.FakeNetworkStore.MarkNetworkCreateForError(removeError, &v, "RemoveNetwork")
		networks[k] = v
	}
	spec.Networks = networks
}

// InjectForcedUpdateError adds types.StackLabel to all specs
func InjectForcedUpdateError(cli *FakeReconcilerClient, spec *types.StackSpec, removeError string) {

	for i := range spec.Services {
		cli.FakeServiceStore.MarkServiceSpecForError(removeError, &spec.Services[i], "UpdateService")
	}
	for i := range spec.Configs {
		cli.FakeConfigStore.MarkConfigSpecForError(removeError, &spec.Configs[i], "UpdateConfig")
	}
	for i := range spec.Secrets {
		cli.FakeSecretStore.MarkSecretSpecForError(removeError, &spec.Secrets[i], "UpdateSecret")
	}
	networks := map[string]dockerTypes.NetworkCreate{}
	for k, v := range spec.Networks {
		cli.FakeNetworkStore.MarkNetworkCreateForError(removeError, &v, "UpdateNetwork")
		networks[k] = v
	}
	spec.Networks = networks
}

// InjectForcedGetError marks all specs with Get* errors
func InjectForcedGetError(cli *FakeReconcilerClient, spec *types.StackSpec, removeError string) {

	for i := range spec.Services {
		cli.FakeServiceStore.MarkServiceSpecForError(removeError, &spec.Services[i], "GetService")
	}
	for i := range spec.Configs {
		cli.FakeConfigStore.MarkConfigSpecForError(removeError, &spec.Configs[i], "GetConfig")
	}
	for i := range spec.Secrets {
		cli.FakeSecretStore.MarkSecretSpecForError(removeError, &spec.Secrets[i], "GetSecret")
	}
	networks := map[string]dockerTypes.NetworkCreate{}
	for k, v := range spec.Networks {
		cli.FakeNetworkStore.MarkNetworkCreateForError(removeError, &v, "GetNetwork")
		networks[k] = v
	}
	spec.Networks = networks
}

// InjectForcedGetResourcesError marks all specs with Get errors
func InjectForcedGetResourcesError(cli *FakeReconcilerClient, spec *types.StackSpec, removeError string) {

	for i := range spec.Services {
		cli.FakeServiceStore.MarkServiceSpecForError(removeError, &spec.Services[i], "GetServices")
	}
	for i := range spec.Configs {
		cli.FakeConfigStore.MarkConfigSpecForError(removeError, &spec.Configs[i], "GetConfigs")
	}
	for i := range spec.Secrets {
		cli.FakeSecretStore.MarkSecretSpecForError(removeError, &spec.Secrets[i], "GetSecrets")
	}
	networks := map[string]dockerTypes.NetworkCreate{}
	for k, v := range spec.Networks {
		cli.FakeNetworkStore.MarkNetworkCreateForError(removeError, &v, "GetNetworks")
		networks[k] = v
	}
	spec.Networks = networks
}

// GenerateStackFixtures creates some types.Stack fixtures
// as well as marking the EVEN ones as belonging
// to types.StackLabel
func GenerateStackFixtures(n int, label string) []types.Stack {
	fixtures := make([]types.Stack, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%dstack", i)
		spec := GetTestStackSpec(specName)
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

// GetTestStackSpec creates a full types.StackSpec
func GetTestStackSpec(name string) types.StackSpec {
	return GetTestStackSpecWithMultipleSpecs(1, name)
}

// GetTestStackSpecWithMultipleSpecs creates a full types.StackSpec with multiple specs
func GetTestStackSpecWithMultipleSpecs(n int, name string) types.StackSpec {

	serviceFixtures := GenerateServiceFixtures(n, name+"service", name+"service")
	configFixtures := GenerateConfigFixtures(n, name+"config", name+"config")
	secretFixtures := GenerateSecretFixtures(n, name+"secret", name+"secret")
	networkFixtures := GenerateNetworkFixtures(n, name+"network", name+"network")

	services := []swarm.ServiceSpec{}
	configs := []swarm.ConfigSpec{}
	secrets := []swarm.SecretSpec{}
	networks := map[string]dockerTypes.NetworkCreate{}

	for _, service := range serviceFixtures {
		services = append(services, service.Spec)
	}
	for _, config := range configFixtures {
		configs = append(configs, config.Spec)
	}
	for _, secret := range secretFixtures {
		secrets = append(secrets, secret.Spec)
	}
	for _, ncr := range networkFixtures {
		networks[ncr.Name+"network"] = ncr.NetworkCreate
	}

	spec := types.StackSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
		Services: services,
		Configs:  configs,
		Secrets:  secrets,
		Networks: networks,
	}

	return spec
}

// GetTestStack creates a minimal type.Stack
func GetTestStack(name string) types.Stack {
	stackSpec := GetTestStackSpec(name)
	return types.Stack{
		Spec: stackSpec,
	}
}

// GenerateServiceFixtures creates some swarm.Service fixtures
// as well as marking the EVEN ones as belonging
// to types.StackLabel
func GenerateServiceFixtures(n int, name, label string) []swarm.Service {
	fixtures := make([]swarm.Service, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%s%dservice", name, i)
		imageName := fmt.Sprintf("%s%dimage", name, i)
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
func GenerateConfigFixtures(n int, name, label string) []swarm.Config {
	fixtures := make([]swarm.Config, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%s%dconfig", name, i)
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
func GenerateSecretFixtures(n int, name, label string) []swarm.Secret {
	fixtures := make([]swarm.Secret, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%s%dsecret", name, i)
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
func GenerateNetworkFixtures(n int, name, label string) []dockerTypes.NetworkCreateRequest {
	fixtures := make([]dockerTypes.NetworkCreateRequest, n)
	var i int
	for i < n {
		specName := fmt.Sprintf("%s%dnetwork", name, i)
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
