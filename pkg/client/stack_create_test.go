package client

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/docker/stacks/pkg/types"

	"gotest.tools/assert"
)

func TestCreateStackServerError(t *testing.T) {
	ctx := context.Background()
	s := Settings{
		Client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}
	cli, err := NewClientWithSettings(s)
	assert.NilError(t, err)
	_, err = cli.StackCreate(ctx, types.StackSpec{}, types.StackCreateOptions{})
	assert.ErrorContains(t, err, "Server error")
}

func TestCreateStackEmpty(t *testing.T) {
	ctx := context.Background()
	s := Settings{
		Client: newMockClient(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString("{}")),
			}, nil
		}),
	}
	cli, err := NewClientWithSettings(s)
	assert.NilError(t, err)
	_, err = cli.StackCreate(ctx, types.StackSpec{}, types.StackCreateOptions{})
	assert.NilError(t, err)
}
