package kube

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	composetypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"
)

// NOTE: The conversions in this package are required to create or update
// stacks, but should be replaced through a tighter integration of the
// docker/stacks and docker/compose-on-kubernetes projects.

// NOTE: majority of file contents were adapted for the v1beta2 version from:
// https://github.com/docker/compose-on-kubernetes/blob/master/internal/conversions/v1alpha3.go

var constraintEquals = regexp.MustCompile(`([\w\.]*)\W*(==|!=)\W*([\w\.]*)`)

const (
	swarmOs          = "node.platform.os"
	swarmArch        = "node.platform.arch"
	swarmHostname    = "node.hostname"
	swarmLabelPrefix = "node.labels."
)

// FromStackSpec converts a StackSpec to a v1beta2.Stack
func FromStackSpec(spec types.StackSpec) *v1beta2.Stack {
	namespace := spec.Collection
	if namespace == "" {
		namespace = "default"
	}
	return &v1beta2.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Metadata.Name,
			Namespace:   namespace,
			Annotations: spec.Metadata.Labels,
		},
		Spec: &v1beta2.StackSpec{
			Services: fromComposeServices(spec.Services),
			Secrets:  fromComposeSecrets(spec.Secrets),
			Configs:  fromComposeConfigs(spec.Configs),
		},
	}
}

func fromComposeServices(s []composetypes.ServiceConfig) []v1beta2.ServiceConfig {
	res := []v1beta2.ServiceConfig{}
	for _, service := range s {
		res = append(res, fromComposeServiceConfig(service))
	}

	return res
}

func fromComposeSecrets(s map[string]composetypes.SecretConfig) map[string]v1beta2.SecretConfig {
	if s == nil {
		return nil
	}
	m := map[string]v1beta2.SecretConfig{}
	for key, value := range s {
		m[key] = v1beta2.SecretConfig{
			Name: value.Name,
			File: value.File,
			External: v1beta2.External{
				Name:     value.External.Name,
				External: value.External.External,
			},
			Labels: value.Labels,
		}
	}
	return m
}

func fromComposeConfigs(s map[string]composetypes.ConfigObjConfig) map[string]v1beta2.ConfigObjConfig {
	if s == nil {
		return nil
	}
	m := map[string]v1beta2.ConfigObjConfig{}
	for key, value := range s {
		m[key] = v1beta2.ConfigObjConfig{
			Name: value.Name,
			File: value.File,
			External: v1beta2.External{
				Name:     value.External.Name,
				External: value.External.External,
			},
			Labels: value.Labels,
		}
	}
	return m
}

func fromComposeServiceConfig(s composetypes.ServiceConfig) v1beta2.ServiceConfig {
	var userID *int64
	if s.User != "" {
		numerical, err := strconv.Atoi(s.User)
		if err == nil {
			unixUserID := int64(numerical)
			userID = &unixUserID
		}
	}
	return v1beta2.ServiceConfig{
		Name:    s.Name,
		CapAdd:  s.CapAdd,
		CapDrop: s.CapDrop,
		Command: s.Command,
		Configs: fromComposeServiceConfigs(s.Configs),
		Deploy: v1beta2.DeployConfig{
			Mode:          s.Deploy.Mode,
			Replicas:      s.Deploy.Replicas,
			Labels:        s.Deploy.Labels,
			UpdateConfig:  fromComposeUpdateConfig(s.Deploy.UpdateConfig),
			Resources:     fromComposeResources(s.Deploy.Resources),
			RestartPolicy: fromComposeRestartPolicy(s.Deploy.RestartPolicy),
			Placement:     fromComposePlacement(s.Deploy.Placement),
		},
		Entrypoint:      s.Entrypoint,
		Environment:     s.Environment,
		ExtraHosts:      s.ExtraHosts,
		Hostname:        s.Hostname,
		HealthCheck:     fromComposeHealthcheck(s.HealthCheck),
		Image:           s.Image,
		Ipc:             s.Ipc,
		Labels:          s.Labels,
		Pid:             s.Pid,
		Ports:           fromComposePorts(s.Ports),
		Privileged:      s.Privileged,
		ReadOnly:        s.ReadOnly,
		Secrets:         fromComposeServiceSecrets(s.Secrets),
		StdinOpen:       s.StdinOpen,
		StopGracePeriod: composetypes.ConvertDurationPtr(s.StopGracePeriod),
		Tmpfs:           s.Tmpfs,
		Tty:             s.Tty,
		User:            userID,
		Volumes:         fromComposeServiceVolumeConfig(s.Volumes),
		WorkingDir:      s.WorkingDir,
	}

}

func fromComposePorts(ports []composetypes.ServicePortConfig) []v1beta2.ServicePortConfig {
	if ports == nil {
		return nil
	}
	p := make([]v1beta2.ServicePortConfig, len(ports))
	for i, port := range ports {
		p[i] = v1beta2.ServicePortConfig{
			Mode:      port.Mode,
			Target:    port.Target,
			Published: port.Published,
			Protocol:  port.Protocol,
		}
	}
	return p
}

func fromComposeServiceSecrets(secrets []composetypes.ServiceSecretConfig) []v1beta2.ServiceSecretConfig {
	if secrets == nil {
		return nil
	}
	c := make([]v1beta2.ServiceSecretConfig, len(secrets))
	for i, secret := range secrets {
		c[i] = v1beta2.ServiceSecretConfig{
			Source: secret.Source,
			Target: secret.Target,
			UID:    secret.UID,
			Mode:   secret.Mode,
		}
	}
	return c
}

func fromComposeServiceConfigs(configs []composetypes.ServiceConfigObjConfig) []v1beta2.ServiceConfigObjConfig {
	if configs == nil {
		return nil
	}
	c := make([]v1beta2.ServiceConfigObjConfig, len(configs))
	for i, config := range configs {
		c[i] = v1beta2.ServiceConfigObjConfig{
			Source: config.Source,
			Target: config.Target,
			UID:    config.UID,
			Mode:   config.Mode,
		}
	}
	return c
}

func fromComposeHealthcheck(h *composetypes.HealthCheckConfig) *v1beta2.HealthCheckConfig {
	if h == nil {
		return nil
	}
	return &v1beta2.HealthCheckConfig{
		Test:     h.Test,
		Timeout:  composetypes.ConvertDurationPtr(h.Timeout),
		Interval: composetypes.ConvertDurationPtr(h.Interval),
		Retries:  h.Retries,
	}
}

func fromComposePlacement(p composetypes.Placement) v1beta2.Placement {
	return v1beta2.Placement{
		Constraints: fromComposeConstraints(p.Constraints),
	}
}

func fromComposeConstraints(s []string) *v1beta2.Constraints {
	if len(s) == 0 {
		return nil
	}
	constraints := &v1beta2.Constraints{}
	for _, constraint := range s {
		matches := constraintEquals.FindStringSubmatch(constraint)
		if len(matches) == 4 {
			key := matches[1]
			operator := matches[2]
			value := matches[3]
			constraint := &v1beta2.Constraint{
				Operator: operator,
				Value:    value,
			}
			switch {
			case key == swarmOs:
				constraints.OperatingSystem = constraint
			case key == swarmArch:
				constraints.Architecture = constraint
			case key == swarmHostname:
				constraints.Hostname = constraint
			case strings.HasPrefix(key, swarmLabelPrefix):
				if constraints.MatchLabels == nil {
					constraints.MatchLabels = map[string]v1beta2.Constraint{}
				}
				constraints.MatchLabels[strings.TrimPrefix(key, swarmLabelPrefix)] = *constraint
			}
		}
	}
	return constraints
}

func fromComposeResources(r composetypes.Resources) v1beta2.Resources {
	return v1beta2.Resources{
		Limits:       fromComposeResourcesResource(r.Limits),
		Reservations: fromComposeResourcesResource(r.Reservations),
	}
}

func fromComposeResourcesResource(r *composetypes.Resource) *v1beta2.Resource {
	if r == nil {
		return nil
	}
	return &v1beta2.Resource{
		MemoryBytes: int64(r.MemoryBytes),
		NanoCPUs:    r.NanoCPUs,
	}
}

func fromComposeUpdateConfig(u *composetypes.UpdateConfig) *v1beta2.UpdateConfig {
	if u == nil {
		return nil
	}
	return &v1beta2.UpdateConfig{
		Parallelism: u.Parallelism,
	}
}

func fromComposeRestartPolicy(r *composetypes.RestartPolicy) *v1beta2.RestartPolicy {
	if r == nil {
		return nil
	}
	return &v1beta2.RestartPolicy{
		Condition: r.Condition,
	}
}

func fromComposeServiceVolumeConfig(vs []composetypes.ServiceVolumeConfig) []v1beta2.ServiceVolumeConfig {
	if vs == nil {
		return nil
	}
	volumes := []v1beta2.ServiceVolumeConfig{}
	for _, v := range vs {
		volumes = append(volumes, v1beta2.ServiceVolumeConfig{
			Type:     v.Type,
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}
	return volumes
}
