package claudebox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/health", r.URL.Path)

			_ = json.NewEncoder(w).Encode(
				HealthResponse{Status: "ok"},
			)
		},
	)

	resp, err := c.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestHealthServerDown(t *testing.T) {
	c := New("http://127.0.0.1:1")

	_, err := c.Health(context.Background())
	require.Error(t, err)
}

func TestHealthInvalidJSON(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `{not json`),
	)

	_, err := c.Health(context.Background())
	require.Error(t, err)
}

func TestStatus(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/status", r.URL.Path)

			_ = json.NewEncoder(w).Encode(
				StatusResponse{
					BusyWorkspaces: []string{
						"/workspaces/foo",
						"/workspaces/bar",
					},
				},
			)
		},
	)

	resp, err := c.Status(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{
		"/workspaces/foo",
		"/workspaces/bar",
	}, resp.BusyWorkspaces)
}

func TestStatusEmpty(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `{"busyWorkspaces":[]}`),
	)

	resp, err := c.Status(context.Background())
	require.NoError(t, err)
	assert.Empty(t, resp.BusyWorkspaces)
}

func TestStatusInvalidJSON(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `{not json`),
	)

	_, err := c.Status(context.Background())
	require.Error(t, err)
}

func TestStatus401(t *testing.T) {
	c, _ := testServer(t,
		errorHandler(t,
			http.StatusUnauthorized,
			`{"detail":"unauthorized"}`,
		),
	)

	_, err := c.Status(context.Background())
	require.Error(t, err)

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t,
		http.StatusUnauthorized, apiErr.StatusCode,
	)
}
