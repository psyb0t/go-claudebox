package claudebox

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/psyb0t/ctxerrors"
)

// HealthResponse is the response from GET /health.
type HealthResponse struct {
	Status string `json:"status"`
}

// Health checks if the server is up.
func (c *Client) Health(
	ctx context.Context,
) (*HealthResponse, error) {
	resp, err := c.do(
		ctx, http.MethodGet, "/health", nil,
	)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var v HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(
		&v,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	return &v, nil
}

// RunInfo is a summary of an async run, returned by
// GET /status.
type RunInfo struct {
	RunID     string `json:"runId"`
	Workspace string `json:"workspace"`
	Status    string `json:"status"`
}

// StatusResponse is the response from GET /status.
type StatusResponse struct {
	BusyWorkspaces []string  `json:"busyWorkspaces"`
	Runs           []RunInfo `json:"runs,omitempty"`
}

// Status returns currently busy workspaces.
func (c *Client) Status(
	ctx context.Context,
) (*StatusResponse, error) {
	resp, err := c.do(
		ctx, http.MethodGet, "/status", nil,
	)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var v StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(
		&v,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	return &v, nil
}
