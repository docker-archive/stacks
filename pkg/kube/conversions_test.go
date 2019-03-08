package kube

import (
	"fmt"
	"testing"
	"time"

	"github.com/docker/compose-on-kubernetes/api/compose/v1beta2"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	composetypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"
)

var (
	composeSecond        = composetypes.Duration(time.Second)
	second               = time.Second
	userID        int64  = 40523
	retries       uint64 = 5
	replicas      uint64 = 2
	parallelism   uint64 = 3

	testStackSpec = types.StackSpec{
		Metadata: types.Metadata{
			Name: "test-stack",
			Labels: map[string]string{
				"stackkey": "value",
			},
		},
		Collection: "namespace1",
		Services: []composetypes.ServiceConfig{
			{
				Name:    "service1",
				Image:   "image1",
				CapAdd:  []string{"CAP_SYS_ADMIN"},
				CapDrop: []string{"CAP_WHAT_LOL"},
				Command: []string{"execute", "this"},
				Configs: []composetypes.ServiceConfigObjConfig{
					{
						Source: "config1",
					},
					{
						Source: "config2",
					},
				},
				Deploy: composetypes.DeployConfig{
					Mode:     "replicated",
					Replicas: &replicas,
					Labels: map[string]string{
						"deploykey": "deployval",
					},
					UpdateConfig: &composetypes.UpdateConfig{
						Parallelism: &parallelism,
					},
					Resources: composetypes.Resources{
						Limits: &composetypes.Resource{
							NanoCPUs:    "1",
							MemoryBytes: composetypes.UnitBytes(52),
						},
						Reservations: &composetypes.Resource{
							NanoCPUs:    "2",
							MemoryBytes: composetypes.UnitBytes(53),
						},
					},
					RestartPolicy: &composetypes.RestartPolicy{
						Condition: "condition1",
					},
					Placement: composetypes.Placement{
						Constraints: []string{
							// NOTE: assertions will fail unless we order
							// the constraints in the order of arch, OS,
							// hostname, label
							fmt.Sprintf("%s!=x86", swarmArch),
							fmt.Sprintf("%s==linux", swarmOs),
							fmt.Sprintf("%s!=node1.local", swarmHostname),
							fmt.Sprintf("%slabel1==value1", swarmLabelPrefix),
						},
					},
				},
				StopGracePeriod: &composeSecond,
				HealthCheck: &composetypes.HealthCheckConfig{
					Test: composetypes.HealthCheckTest{
						"/test",
						"arg",
					},
					Timeout:  &composeSecond,
					Interval: &composeSecond,
					Retries:  &retries,
				},
				Labels: map[string]string{
					"servicekey": "servicelabel",
				},
				User: fmt.Sprintf("%d", userID),
			},
			{
				Name:  "service2",
				Image: "image2",
				Secrets: []composetypes.ServiceSecretConfig{
					{
						Source: "secret1",
					},
				},
			},
		},
		Secrets: map[string]composetypes.SecretConfig{
			"secret1": {
				Name: "secret1",
				External: composetypes.External{
					Name:     "somename",
					External: true,
				},
				Labels: map[string]string{
					"secretkey": "value",
				},
			},
		},
		Configs: map[string]composetypes.ConfigObjConfig{
			"config1": {
				Name: "config1",
				External: composetypes.External{
					External: true,
				},
				Labels: map[string]string{
					"configkey": "value",
				},
			},
			"config2": {
				Name: "config2",
			},
		},
	}

	testKubeStackSpec = v1beta2.StackSpec{
		Services: []v1beta2.ServiceConfig{
			{
				Name:    "service1",
				Image:   "image1",
				CapAdd:  []string{"CAP_SYS_ADMIN"},
				CapDrop: []string{"CAP_WHAT_LOL"},
				Command: []string{"execute", "this"},
				Deploy: v1beta2.DeployConfig{
					Mode:     "replicated",
					Replicas: &replicas,
					Labels: map[string]string{
						"deploykey": "deployval",
					},
					UpdateConfig: &v1beta2.UpdateConfig{
						Parallelism: &parallelism,
					},
					Resources: v1beta2.Resources{
						Limits: &v1beta2.Resource{
							NanoCPUs:    "1",
							MemoryBytes: int64(52),
						},
						Reservations: &v1beta2.Resource{
							NanoCPUs:    "2",
							MemoryBytes: int64(53),
						},
					},
					RestartPolicy: &v1beta2.RestartPolicy{
						Condition: "condition1",
					},
					Placement: v1beta2.Placement{
						Constraints: &v1beta2.Constraints{
							Architecture: &v1beta2.Constraint{
								Operator: "!=",
								Value:    "x86",
							},
							OperatingSystem: &v1beta2.Constraint{
								Operator: "==",
								Value:    "linux",
							},
							Hostname: &v1beta2.Constraint{
								Operator: "!=",
								Value:    "node1.local",
							},
							MatchLabels: map[string]v1beta2.Constraint{
								"label1": {
									Operator: "==",
									Value:    "value1",
								},
							},
						},
					},
				},

				StopGracePeriod: &second,
				Configs: []v1beta2.ServiceConfigObjConfig{
					{
						Source: "config1",
					},
					{
						Source: "config2",
					},
				},
				HealthCheck: &v1beta2.HealthCheckConfig{
					Test: []string{
						"/test",
						"arg",
					},
					Timeout:  &second,
					Interval: &second,
					Retries:  &retries,
				},
				Labels: map[string]string{
					"servicekey": "servicelabel",
				},
				User: &userID,
			},
			{
				Name:  "service2",
				Image: "image2",
				Secrets: []v1beta2.ServiceSecretConfig{
					{
						Source: "secret1",
					},
				},
			},
		},
		Secrets: map[string]v1beta2.SecretConfig{
			"secret1": {
				Name: "secret1",
				External: v1beta2.External{
					Name:     "somename",
					External: true,
				},
				Labels: map[string]string{
					"secretkey": "value",
				},
			},
		},
		Configs: map[string]v1beta2.ConfigObjConfig{
			"config1": {
				Name: "config1",
				External: v1beta2.External{
					External: true,
				},
				Labels: map[string]string{
					"configkey": "value",
				},
			},
			"config2": {
				Name: "config2",
			},
		},
	}

	testKubeStack = v1beta2.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stack",
			Namespace: "namespace1",
			Annotations: map[string]string{
				"stackkey": "value",
			},
			ResourceVersion: "16",
		},
		Spec: &testKubeStackSpec,
	}

	testStack = types.Stack{
		ID:   "kube_namespace1_test-stack",
		Spec: testStackSpec,
		Version: types.Version{
			Index: uint64(16),
		},
		Orchestrator: types.OrchestratorKubernetes,
	}
)

func TestFromStackSpec(t *testing.T) {
	resp := FromStackSpec(testStackSpec)
	// Hardcode the version as we are not actually aware of it just from the stack spec
	resp.ObjectMeta.ResourceVersion = "16"
	assert.DeepEqual(t, *resp, testKubeStack)
}

func TestConvertFromKubeStack(t *testing.T) {
	resp, err := ConvertFromKubeStack(&testKubeStack)
	require.NoError(t, err)
	assert.DeepEqual(t, resp, testStack)
}
