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

func TestParseComposeInputServerError(t *testing.T) {
	ctx := context.Background()
	s := Settings{
		Client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}
	cli, err := NewClientWithSettings(s)
	assert.NilError(t, err)
	_, err = cli.ParseComposeInput(ctx, types.ComposeInput{})
	assert.ErrorContains(t, err, "Server error")
}

func TestParseComposeInputEmpty(t *testing.T) {
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
	_, err = cli.ParseComposeInput(ctx, types.ComposeInput{})
	assert.NilError(t, err)
}
