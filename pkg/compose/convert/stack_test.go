package convert

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"

	"github.com/docker/stacks/pkg/compose/loader"
	composetypes "github.com/docker/stacks/pkg/compose/types"
)

func TestLoadStacks(t *testing.T) {
	// Switch to a better reference
	input, err := loader.LoadComposefile([]string{"../tests/fixtures/compatibility-mode/docker-compose.yml"})
	assert.NilError(t, err)
	stack, err := loader.ParseComposeInput(*input)
	assert.NilError(t, err)

	namespace := NewNamespace("dummy")

	// TODO - figure out how to map theses...
	// the compose file used above doesn't have secrets/config so we'll need to wire this up further...
	secrets := []*swarm.SecretReference{}
	configs := []*swarm.ConfigReference{}

	secretlist, err := Secrets(
		namespace,
		stack.Spec.Secrets,
	)
	assert.NilError(t, err)
	assert.Check(t, is.Len(secretlist, 0))

	cfglist, err := Configs(
		namespace,
		stack.Spec.Configs,
	)
	assert.NilError(t, err)
	assert.Check(t, is.Len(cfglist, 0))

	volumes := []composetypes.ServiceVolumeConfig{}
	networks := map[string]struct{}{}
	for _, service := range stack.Spec.Services {
		res, err := Service(namespace,
			service,
			stack.Spec.Networks,
			stack.Spec.Volumes,
			secrets,
			configs,
		)
		assert.NilError(t, err)
		// Spot check the service itself
		assert.Check(t, is.Contains(res.Annotations.Name, "foo"))
		volumes = append(volumes, service.Volumes...)
		for name := range service.Networks {
			networks[name] = struct{}{}
		}
	}

	nwlist, externalNames := Networks(
		namespace,
		stack.Spec.Networks,
		networks,
	)
	assert.Check(t, is.Len(nwlist, 1), stack.Spec.Networks)
	assert.Check(t, is.Len(externalNames, 0))

	mounts, err := Volumes(
		volumes,
		stack.Spec.Volumes,
		namespace,
	)
	assert.NilError(t, err)
	assert.Check(t, is.Len(mounts, 1))
}
