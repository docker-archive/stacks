package client

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/docker/stacks/pkg/types"

	"github.com/docker/docker/api/types/filters"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestStackListServerError(t *testing.T) {
	ctx := context.Background()
	opts := types.StackListOptions{}
	s := Settings{
		Client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}
	cli, err := NewClientWithSettings(s)
	assert.NilError(t, err)
	_, err = cli.StackList(ctx, opts)
	assert.ErrorContains(t, err, "Server error")
}

func TestStackListEmpty(t *testing.T) {
	ctx := context.Background()
	filters := filters.NewArgs()
	filters.Add("label", "label1")
	opts := types.StackListOptions{
		Filters: filters,
	}
	s := Settings{
		Client: newMockClient(func(req *http.Request) (*http.Response, error) {
			query := req.URL.Query()
			if val, ok := query["filters"]; !ok || val[0] != `{"label":{"label1":true}}` {
				return nil, fmt.Errorf("missing query parameters - found: %v", val[0])
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString("[]")),
			}, nil
		}),
	}
	cli, err := NewClientWithSettings(s)
	assert.NilError(t, err)
	res, err := cli.StackList(ctx, opts)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(res, 0))
}
