package substitution

import (
	"testing"

	composetypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/types"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestDoPortSubstitutions(t *testing.T) {
	spec := types.StackSpec{
		Services: composetypes.Services{
			composetypes.ServiceConfig{
				Ports: []composetypes.ServicePortConfig{
					{
						Variable: "${PORT1}",
					},
				},
			},
		},
		PropertyValues: []string{
			"PORT1=1000-1002",
		},
	}
	expectedPorts := []composetypes.ServicePortConfig{
		{
			Mode:     "ingress",
			Protocol: "tcp",
			Target:   1000,
		},
		{
			Mode:     "ingress",
			Protocol: "tcp",
			Target:   1001,
		},
		{
			Mode:     "ingress",
			Protocol: "tcp",
			Target:   1002,
		},
	}
	outspec, err := DoSubstitution(spec)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(outspec.Services[0].Ports, expectedPorts))
}

func TestDoVolumeSubstitutions(t *testing.T) {
	spec := types.StackSpec{
		Services: composetypes.Services{
			composetypes.ServiceConfig{
				Volumes: []composetypes.ServiceVolumeConfig{
					{
						Target: "${VOL1}",
					},
				},
			},
		},
		PropertyValues: []string{
			"VOL1=/host/data/configs:/etc/configs/:ro",
		},
	}
	expectedVolumes := []composetypes.ServiceVolumeConfig{
		{
			Type:     "bind",
			Source:   "/host/data/configs",
			Target:   "/etc/configs/",
			ReadOnly: true,
		},
	}
	outspec, err := DoSubstitution(spec)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(outspec.Services[0].Volumes, expectedVolumes))
}
