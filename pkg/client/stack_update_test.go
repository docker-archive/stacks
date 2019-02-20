package client

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/docker/stacks/pkg/types"

	"gotest.tools/assert"
)

func TestStackUpdateServerError(t *testing.T) {
	ctx := context.Background()
	id := "dummy"
	version := types.Version{Index: 123}
	s := Settings{
		Client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}
	cli, err := NewClientWithSettings(s)
	assert.NilError(t, err)
	err = cli.StackUpdate(ctx, id, version, types.StackSpec{}, types.StackUpdateOptions{})
	assert.ErrorContains(t, err, "Server error")
}

func TestStackUpdateEmpty(t *testing.T) {
	// id string, version types.Version, spec types.StackSpec) error {
	ctx := context.Background()
	id := "dummy"
	version := types.Version{Index: 123}
	s := Settings{
		Client: newMockClient(func(req *http.Request) (*http.Response, error) {
			query := req.URL.Query()
			if val, ok := query["version"]; !ok || val[0] != "123" {
				return nil, fmt.Errorf("missing version parameter- found: %v", val[0])
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString("")),
			}, nil
		}),
	}
	cli, err := NewClientWithSettings(s)
	assert.NilError(t, err)
	err = cli.StackUpdate(ctx, id, version, types.StackSpec{}, types.StackUpdateOptions{})
	assert.NilError(t, err)
}
