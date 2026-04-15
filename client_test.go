package claudebox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthTokenSent(t *testing.T) {
	c, _ := testServerWithToken(
		t, "secret123",
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t,
				"Bearer secret123",
				r.Header.Get("Authorization"),
			)

			_ = json.NewEncoder(w).Encode(
				HealthResponse{Status: "ok"},
			)
		},
	)

	_, err := c.Health(context.Background())
	require.NoError(t, err)
}

func TestNoTokenByDefault(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t,
				r.Header.Get("Authorization"),
			)

			_ = json.NewEncoder(w).Encode(
				HealthResponse{Status: "ok"},
			)
		},
	)

	_, err := c.Health(context.Background())
	require.NoError(t, err)
}

func TestAPIError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			"unauthorized",
			http.StatusUnauthorized,
			`{"detail":"unauthorized"}`,
		},
		{
			"not found",
			http.StatusNotFound,
			`{"detail":"not found"}`,
		},
		{
			"conflict",
			http.StatusConflict,
			`{"detail":"workspace busy"}`,
		},
		{
			"bad request",
			http.StatusBadRequest,
			`{"detail":"path outside root"}`,
		},
		{
			"internal server error",
			http.StatusInternalServerError,
			`{"detail":"boom"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testServer(t,
				errorHandler(t, tt.status, tt.body),
			)

			_, err := c.Status(context.Background())
			require.Error(t, err)

			var apiErr *APIError
			require.True(t,
				errors.As(err, &apiErr),
				"expected *APIError, got %T", err,
			)
			assert.Equal(t, tt.status, apiErr.StatusCode)
			assert.Contains(t, apiErr.Body, "detail")
			assert.Contains(t,
				apiErr.Error(),
				"claudebox: HTTP",
			)
		})
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c := New(
		"http://localhost",
		WithHTTPClient(custom),
	)

	assert.Equal(t, custom, c.httpClient)
}

func TestContextCancellation(t *testing.T) {
	c, _ := testServer(
		t,
		func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		},
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	_, err := c.Health(ctx)
	require.Error(t, err)
}
