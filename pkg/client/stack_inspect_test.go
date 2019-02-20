package client

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"gotest.tools/assert"
)

func TestStackInspectServerError(t *testing.T) {
	ctx := context.Background()
	id := "dummy"
	s := Settings{
		Client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}
	cli, err := NewClientWithSettings(s)
	assert.NilError(t, err)
	_, err = cli.StackInspect(ctx, id)
	assert.ErrorContains(t, err, "Server error")
}

func TestStackInspectEmpty(t *testing.T) {
	ctx := context.Background()
	id := "dummy"
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
	_, err = cli.StackInspect(ctx, id)
	assert.NilError(t, err)
}
