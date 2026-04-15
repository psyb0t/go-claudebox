package claudebox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAuthTokenSent(t *testing.T) {
	c, _ := testServerWithToken(
		t, "secret123",
		func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer secret123" {
				t.Errorf(
					"got auth %q, want Bearer secret123",
					auth,
				)
			}

			_ = json.NewEncoder(w).Encode(
				HealthResponse{Status: "ok"},
			)
		},
	)

	_, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoTokenByDefault(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "" {
				t.Errorf(
					"expected no auth header, got %q",
					auth,
				)
			}

			_ = json.NewEncoder(w).Encode(
				HealthResponse{Status: "ok"},
			)
		},
	)

	_, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIError(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write(
				[]byte(`{"detail":"unauthorized"}`),
			)
		},
	)

	_, err := c.Status(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf(
			"got status %d, want 401",
			apiErr.StatusCode,
		)
	}

	if !strings.Contains(apiErr.Body, "unauthorized") {
		t.Errorf("got body %q", apiErr.Body)
	}

	if !strings.Contains(apiErr.Error(), "401") {
		t.Errorf(
			"Error() should contain status code: %s",
			apiErr.Error(),
		)
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c := New(
		"http://localhost",
		WithHTTPClient(custom),
	)

	if c.httpClient != custom {
		t.Error("custom http client not set")
	}
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
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
