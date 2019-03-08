package kube

import (
	"fmt"
	"strconv"
	"time"

	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	composetypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"
)

// NOTE: The conversions in this package are required to inspect or list
// stacks, but should be replaced through a tighter integration of the
// docker/stacks and docker/compose-on-kubernetes projects.

// ConvertFromKubeStack converts a compose/v1beta2.Stack to a Stack.
func ConvertFromKubeStack(kubeStack *v1beta2.Stack) (types.Stack, error) {
	namespace := kubeStack.ObjectMeta.Namespace
	name := kubeStack.ObjectMeta.Name

	stackSpec := convertFromKubeStackSpec(kubeStack.Spec, kubeStack.ObjectMeta)
	res := types.Stack{
		ID:           getKubeStackID(namespace, name),
		Orchestrator: types.OrchestratorKubernetes,
		Spec:         stackSpec,
	}

	// Parse the object version to a uint64 - should be possible for any
	// conformant kubernetes distribution.
	version, err := strconv.ParseUint(kubeStack.ObjectMeta.ResourceVersion, 10, 64)
	if err != nil {
		return types.Stack{}, fmt.Errorf("unable to parse stack version %s as uint64: %s", kubeStack.ObjectMeta.ResourceVersion, err)
	}
	res.Version.Index = version

	return res, nil
}

// ConvertFromKubeStacks converts a []v1beta2.Stack to equivalent
// Stacks.
func ConvertFromKubeStacks(kubeStacks []v1beta2.Stack) ([]types.Stack, error) {
	res := []types.Stack{}
	for _, kubeStack := range kubeStacks {
		stack, err := ConvertFromKubeStack(&kubeStack)
		if err != nil {
			return []types.Stack{}, fmt.Errorf("unable to convert kubernetes stack: %s", err)
		}

		res = append(res, stack)
	}

	return res, nil
}

func convertFromKubeStackSpec(s *v1beta2.StackSpec, objectMeta metav1.ObjectMeta) types.StackSpec {
	return types.StackSpec{
		Metadata: types.Metadata{
			Name: objectMeta.Name,
			// Stack Labels are equal to Kubernetes Annotations
			Labels: objectMeta.Annotations,
		},

		Collection: objectMeta.Namespace,
		Configs:    convertFromKubeConfigs(s.Configs),
		Secrets:    convertFromKubeSecrets(s.Secrets),
		Services:   convertFromKubeServices(s.Services),
	}
}

func convertFromKubeServices(services []v1beta2.ServiceConfig) composetypes.Services {
	res := composetypes.Services{}
	for _, service := range services {
		res = append(res, convertFromKubeService(service))
	}
	return res
}

func fromKubePorts(ports []v1beta2.ServicePortConfig) []composetypes.ServicePortConfig {
	if ports == nil {
		return nil
	}
	p := make([]composetypes.ServicePortConfig, len(ports))
	for i, port := range ports {
		p[i] = composetypes.ServicePortConfig{
			Mode:      port.Mode,
			Target:    port.Target,
			Published: port.Published,
			Protocol:  port.Protocol,
		}
	}
	return p
}

func convertFromKubeService(s v1beta2.ServiceConfig) composetypes.ServiceConfig {
	userID := ""
	if s.User != nil {
		userID = fmt.Sprintf("%d", *s.User)
	}

	return composetypes.ServiceConfig{
		Name:    s.Name,
		CapAdd:  s.CapAdd,
		CapDrop: s.CapDrop,
		Command: s.Command,
		Configs: fromKubeServiceConfigs(s.Configs),
		Deploy: composetypes.DeployConfig{
			Mode:          s.Deploy.Mode,
			Replicas:      s.Deploy.Replicas,
			Labels:        s.Deploy.Labels,
			UpdateConfig:  fromKubeUpdateConfig(s.Deploy.UpdateConfig),
			Resources:     fromKubeResources(s.Deploy.Resources),
			RestartPolicy: fromKubeRestartPolicy(s.Deploy.RestartPolicy),
			Placement:     fromKubePlacement(s.Deploy.Placement),
		},
		Entrypoint:      s.Entrypoint,
		Environment:     s.Environment,
		ExtraHosts:      s.ExtraHosts,
		Hostname:        s.Hostname,
		HealthCheck:     fromKubeHealthCheck(s.HealthCheck),
		Image:           s.Image,
		Ipc:             s.Ipc,
		Labels:          s.Labels,
		Pid:             s.Pid,
		Ports:           fromKubePorts(s.Ports),
		Privileged:      s.Privileged,
		ReadOnly:        s.ReadOnly,
		Secrets:         fromKubeServiceSecrets(s.Secrets),
		StdinOpen:       s.StdinOpen,
		StopGracePeriod: fromDuration(s.StopGracePeriod),
		Tmpfs:           s.Tmpfs,
		Tty:             s.Tty,
		User:            userID,
		Volumes:         fromKubeServiceVolumeConfig(s.Volumes),
		WorkingDir:      s.WorkingDir,
	}
}

func convertFromKubeSecrets(s map[string]v1beta2.SecretConfig) map[string]composetypes.SecretConfig {
	if s == nil {
		return nil
	}
	m := map[string]composetypes.SecretConfig{}
	for key, value := range s {
		m[key] = composetypes.SecretConfig{
			Name: value.Name,
			File: value.File,
			External: composetypes.External{
				Name:     value.External.Name,
				External: value.External.External,
			},
			Labels: value.Labels,
		}
	}
	return m
}

func convertFromKubeConfigs(s map[string]v1beta2.ConfigObjConfig) map[string]composetypes.ConfigObjConfig {
	if s == nil {
		return nil
	}
	m := map[string]composetypes.ConfigObjConfig{}
	for key, value := range s {
		m[key] = composetypes.ConfigObjConfig{
			Name: value.Name,
			File: value.File,
			External: composetypes.External{
				Name:     value.External.Name,
				External: value.External.External,
			},
			Labels: value.Labels,
		}
	}

	return m
}

func fromKubeServiceSecrets(secrets []v1beta2.ServiceSecretConfig) []composetypes.ServiceSecretConfig {
	if secrets == nil {
		return nil
	}
	c := make([]composetypes.ServiceSecretConfig, len(secrets))
	for i, secret := range secrets {
		c[i] = composetypes.ServiceSecretConfig{
			Source: secret.Source,
			Target: secret.Target,
			UID:    secret.UID,
			Mode:   secret.Mode,
		}
	}
	return c
}

func fromKubeServiceConfigs(configs []v1beta2.ServiceConfigObjConfig) []composetypes.ServiceConfigObjConfig {
	if configs == nil {
		return nil
	}

	c := make([]composetypes.ServiceConfigObjConfig, len(configs))
	for i, config := range configs {
		c[i] = composetypes.ServiceConfigObjConfig{
			Source: config.Source,
			Target: config.Target,
			UID:    config.UID,
			Mode:   config.Mode,
		}
	}
	return c
}

func fromKubeUpdateConfig(u *v1beta2.UpdateConfig) *composetypes.UpdateConfig {
	if u == nil {
		return nil
	}
	return &composetypes.UpdateConfig{
		Parallelism: u.Parallelism,
	}
}

func fromKubePlacement(p v1beta2.Placement) composetypes.Placement {
	return composetypes.Placement{
		Constraints: fromKubeConstraints(p.Constraints),
	}
}

func fromKubeConstraints(s *v1beta2.Constraints) []string {
	if s == nil {
		return nil
	}

	constraints := []string{}
	if s.Architecture != nil {
		constraints = append(constraints, fmt.Sprintf("%s%s%s", swarmArch, s.Architecture.Operator, s.Architecture.Value))
	}

	if s.OperatingSystem != nil {
		constraints = append(constraints, fmt.Sprintf("%s%s%s", swarmOs, s.OperatingSystem.Operator, s.OperatingSystem.Value))
	}

	if s.Hostname != nil {
		constraints = append(constraints, fmt.Sprintf("%s%s%s", swarmHostname, s.Hostname.Operator, s.Hostname.Value))
	}

	if s.MatchLabels != nil {
		for label, constraint := range s.MatchLabels {
			key := fmt.Sprintf("%s%s", swarmLabelPrefix, label)
			constraints = append(constraints, fmt.Sprintf("%s%s%s", key, constraint.Operator, constraint.Value))
		}
	}

	return constraints
}

func fromKubeRestartPolicy(r *v1beta2.RestartPolicy) *composetypes.RestartPolicy {
	if r == nil {
		return nil
	}
	return &composetypes.RestartPolicy{
		Condition: r.Condition,
	}
}

func fromKubeResources(r v1beta2.Resources) composetypes.Resources {
	return composetypes.Resources{
		Limits:       fromKubeResourcesResource(r.Limits),
		Reservations: fromKubeResourcesResource(r.Reservations),
	}
}

func fromKubeResourcesResource(r *v1beta2.Resource) *composetypes.Resource {
	if r == nil {
		return nil
	}
	return &composetypes.Resource{
		MemoryBytes: composetypes.UnitBytes(r.MemoryBytes),
		NanoCPUs:    r.NanoCPUs,
	}
}

func fromKubeServiceVolumeConfig(vs []v1beta2.ServiceVolumeConfig) []composetypes.ServiceVolumeConfig {
	if vs == nil {
		return nil
	}
	volumes := []composetypes.ServiceVolumeConfig{}
	for _, v := range vs {
		volumes = append(volumes, composetypes.ServiceVolumeConfig{
			Type:     v.Type,
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}
	return volumes
}

func fromKubeHealthCheck(h *v1beta2.HealthCheckConfig) *composetypes.HealthCheckConfig {
	if h == nil {
		return nil
	}
	return &composetypes.HealthCheckConfig{
		Test:     h.Test,
		Timeout:  fromDuration(h.Timeout),
		Interval: fromDuration(h.Interval),
		Retries:  h.Retries,
	}
}

func fromDuration(d *time.Duration) *composetypes.Duration {
	if d == nil {
		return nil
	}

	r := composetypes.Duration(*d)
	return &r
}
