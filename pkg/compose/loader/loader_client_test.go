package loader

import (
	"strings"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestComposeWithEnv(t *testing.T) {
	input, err := LoadComposefile([]string{"../tests/fixtures/default-env-file/docker-compose.yml"})
	assert.NilError(t, err)
	stack, err := ParseComposeInput(*input)
	assert.NilError(t, err)
	assert.Check(t, is.Len(stack.Spec.Services, 1))
	assert.Check(t, is.Len(stack.Spec.PropertyValues, 4))
	// TODO - deeper inspection of the results, etc.

	input, err = LoadComposefile([]string{"../tests/fixtures/environment-interpolation-with-defaults/docker-compose.yml"})
	assert.NilError(t, err)
	stack, err = ParseComposeInput(*input)
	assert.NilError(t, err)
	assert.Check(t, is.Len(stack.Spec.Services, 1))
	assert.Check(t, is.Len(stack.Spec.PropertyValues, 3))
	allProperties := strings.Join(stack.Spec.PropertyValues, "\n")
	assert.Check(t, strings.Contains(allProperties, "="), allProperties)
	// TODO - deeper inspection of the results, etc.

	// This file is a v1 compose file so skipping for now
	/*
		input, err = LoadComposefile([]string{"../tests/fixtures/environment-interpolation/docker-compose.yml"})
		assert.NilError(t, err)
		stack, err = ParseComposeInput(*input)
		assert.NilError(t, err)
		assert.Check(t, is.Len(stack.Spec.Services, 1))
		assert.Check(t, is.Len(stack.Spec.PropertyValues, 4))
		// TODO - deeper inspection of the results, default values, etc.
	*/

	input, err = LoadComposefile([]string{"../tests/fixtures/tagless-image/docker-compose.yml"})
	assert.NilError(t, err)
	stack, err = ParseComposeInput(*input)
	assert.NilError(t, err)
	assert.Check(t, is.Len(stack.Spec.Services, 1))
	assert.Check(t, is.Len(stack.Spec.PropertyValues, 1))
	// TODO - deeper inspection of the results, default values, etc.

	input, err = LoadComposefile([]string{"../tests/fixtures/unicode-environment/docker-compose.yml"})
	assert.NilError(t, err)
	stack, err = ParseComposeInput(*input)
	assert.NilError(t, err)
	assert.Check(t, is.Len(stack.Spec.Services, 1))
	assert.Check(t, is.Len(stack.Spec.PropertyValues, 1))
	// TODO - deeper inspection of the results, default values, etc.

	// This file is a v1 compose file so skipping for now
	/*
		input, err = LoadComposefile([]string{"../tests/fixtures/volume-path-interpolation/docker-compose.yml"})
		assert.NilError(t, err)
		stack, err = ParseComposeInput(*input)
		assert.NilError(t, err)
		assert.Check(t, is.Len(stack.Spec.Services, 1))
		assert.Check(t, is.Len(stack.Spec.PropertyValues, 1))
		// TODO - deeper inspection of the results, default values, etc.
	*/
}
