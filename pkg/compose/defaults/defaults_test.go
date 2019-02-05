package defaults

import (
	"strings"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func recordMapping(results *[]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		*results = append(*results, name)
		return "", false
	}
}

func TestEscaped(t *testing.T) {
	found := []string{}
	result, err := RecordVariablesWithDefaults("$${foo}", recordMapping(&found))
	assert.NilError(t, err)
	assert.Check(t, is.Equal("${foo}", result))
	assert.Check(t, is.Len(found, 0))
}

func TestSubstituteNoMatch(t *testing.T) {
	found := []string{}
	result, err := RecordVariablesWithDefaults("foo", recordMapping(&found))
	assert.NilError(t, err)
	assert.Equal(t, "foo", result)
	assert.Check(t, is.Len(found, 0))
}

func TestInvalid(t *testing.T) {
	invalidTemplates := []string{
		"${",
		"$}",
		"${}",
		"${ }",
		"${ foo}",
		"${foo }",
		"${foo!}",
	}

	for _, template := range invalidTemplates {
		found := []string{}
		_, err := RecordVariablesWithDefaults(template, recordMapping(&found))
		assert.ErrorContains(t, err, "Invalid template")
		assert.Check(t, is.Len(found, 0))
	}
}

func TestNoValueNoDefault(t *testing.T) {
	for _, template := range []string{"This ${missing} var", "This ${BAR} var"} {
		found := []string{}
		result, err := RecordVariablesWithDefaults(template, recordMapping(&found))
		assert.NilError(t, err)
		assert.Check(t, is.Equal(template, result))
		assert.Check(t, is.Len(found, 1))
	}
}

func TestNoValueWithDefault(t *testing.T) {
	for _, template := range []string{"ok ${missing:-def}", "ok ${missing-def}"} {
		found := []string{}
		result, err := RecordVariablesWithDefaults(template, recordMapping(&found))
		assert.NilError(t, err)
		assert.Check(t, is.Equal(template, result))
		assert.Check(t, is.Len(found, 1))
		tmp := strings.Join(found, "\n")
		assert.Check(t, is.Contains(tmp, "="), tmp)
	}
}

func TestNonAlphanumericDefault(t *testing.T) {
	found := []string{}
	_, err := RecordVariablesWithDefaults("ok ${BAR:-/non:-alphanumeric}", recordMapping(&found))
	assert.NilError(t, err)
	assert.Check(t, is.Len(found, 1))
	tmp := strings.Join(found, "\n")
	assert.Check(t, is.Contains(tmp, "=/non:-alphanumeric"), tmp)
}

func TestMandatory(t *testing.T) {
	found := []string{}
	template := "not ok ${UNSET_VAR:?Mandatory Variable Unset}"
	_, err := RecordVariablesWithDefaults(template, recordMapping(&found))
	assert.NilError(t, err)
	assert.Check(t, is.Len(found, 1))
	tmp := strings.Join(found, "\n")
	assert.Check(t, !strings.Contains(tmp, "="), tmp)
}
