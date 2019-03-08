package store

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/swarmkit/api"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	composetypes "github.com/docker/stacks/pkg/compose/types"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/types"
)

// TestMarshalUnmarshal tests that pass a pair of stack objects to
// MarshalStacks and then passing the resulting proto.Message to
// UnmarshalStacks results in getting the same objects back out.
func TestMarshalUnmarshal(t *testing.T) {
	// so, this is gonna seem stupid as hell, but the marshaling of time in
	// proto messages is weird. To make this test pass, we're going first
	// create timestamps as proto types
	c := gogotypes.TimestampNow()
	u := gogotypes.TimestampNow()

	// we know this conversion will succeed, so throw away the error result
	ct, _ := gogotypes.TimestampFromProto(c)
	ut, _ := gogotypes.TimestampFromProto(u)

	// we don't have to fully fill this in -- we're testing proto marshalling,
	// not JSON marshalling. just add some canned data
	stack := &types.Stack{
		ID: "someID",
		Version: types.Version{
			Index: 1,
		},
		Spec: types.StackSpec{
			Metadata: types.Metadata{
				Name: "someName",
				Labels: map[string]string{
					"key": "value",
				},
			},
			Services: composetypes.Services{
				{
					Name: "bar",
				},
			},
			Collection: "something",
		},
		Orchestrator: types.OrchestratorSwarm,
	}
	swarmStack := &interfaces.SwarmStack{
		ID: "someID",
		Meta: swarm.Meta{
			CreatedAt: ct,
			UpdatedAt: ut,
			Version: swarm.Version{
				Index: 1,
			},
		},
		Spec: interfaces.SwarmStackSpec{
			Annotations: swarm.Annotations{
				Name: "someName",
				Labels: map[string]string{
					"key": "value",
				},
			},
			Services: []swarm.ServiceSpec{
				{
					Annotations: swarm.Annotations{Name: "bar"},
				},
			},
		},
	}

	msg, err := MarshalStacks(stack, swarmStack)
	require.NoError(t, err, "error marshalling stacks")

	// now pack the message into a Resource object
	resource := &api.Resource{
		ID: "someID",
		Meta: api.Meta{
			Version: api.Version{
				Index: 1,
			},
			CreatedAt: c,
			UpdatedAt: u,
		},
		Payload: msg,
	}

	// NOTE(dperny): because of the way marshaling json works (or perhaps
	// doesn't work), we should only have 1 ServiceConfig or ServiceSpec in the
	// stacks, else the order may get scrambled up and make this comparison
	// difficult.
	unstack, unswarm, err := UnmarshalStacks(resource)
	require.NoError(t, err, "error unmarshalling stacks")
	assert.Equal(t, stack, unstack)
	assert.Equal(t, swarmStack, unswarm)
}
