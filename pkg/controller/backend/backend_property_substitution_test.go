package backend

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/stacks/pkg/compose/loader"
	"github.com/docker/stacks/pkg/compose/template"
	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/mocks"
	"github.com/docker/stacks/pkg/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestStacksBackendPropertySubstitutionBasic(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	backendClient := mocks.NewMockBackendClient(ctrl)
	b := NewDefaultStacksBackend(interfaces.NewFakeStackStore(), backendClient)

	// TODO - make this loop over multiple fixtures that follow
	//        the pattern of a compose v2+ file with variables in
	//        a value.env file.

	// Load up the test data
	input, err := loader.LoadComposefile([]string{"../../compose/tests/fixtures/default-env-file/docker-compose.yml"})
	require.NoError(err)
	valueBytes, err := ioutil.ReadFile("../../compose/tests/fixtures/default-env-file/values.env")
	require.NoError(err)
	values := strings.Split(strings.TrimSpace(string(valueBytes)), "\n")

	// Parse it to get the stack create payload
	stackCreate, err := b.ParseComposeInput(*input)
	require.NoError(err)

	// Verify the variables were properly detected
	require.Len(stackCreate.Spec.PropertyValues, len(values), "Expected %d properties, found: %#v", len(values), stackCreate.Spec.PropertyValues)
	// Set the desired property values and orchestrator
	stackCreate.Spec.PropertyValues = values
	stackCreate.Orchestrator = types.OrchestratorSwarm

	// Create the stack
	resp, err := b.CreateStack(*stackCreate)
	require.NoError(err)
	require.Equal("1", resp.ID)

	// Get the swarm stack and verify values were substituted properly
	swarmStack, err := b.GetSwarmStack("1")
	require.NoError(err)

	// Serialize to json to do generic string matching for lingering variables
	data, err := json.MarshalIndent(swarmStack, "", "    ")
	require.NoError(err)

	matches := template.DefaultPattern.FindAllStringSubmatch(string(data), -1)
	require.Len(matches, 0, "%s\nFound remaining variables: %#v", string(data), matches)

	// TODO consider adding a golden referent pattern here to allow
	//      a checked in json serialization of the expected swarm resources
	//      so we can do deeper verification of the expected values
	//      without having to write a lot of custom go test code.
}
