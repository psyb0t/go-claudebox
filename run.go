package claudebox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/psyb0t/ctxerrors"
)

// RunRequest is the body for POST /run.
type RunRequest struct {
	Prompt             string `json:"prompt"`
	Workspace          string `json:"workspace,omitempty"`
	Model              string `json:"model,omitempty"`
	SystemPrompt       string `json:"systemPrompt,omitempty"`
	AppendSystemPrompt string `json:"appendSystemPrompt,omitempty"`
	JSONSchema         string `json:"jsonSchema,omitempty"`
	Effort             string `json:"effort,omitempty"`
	OutputFormat       string `json:"outputFormat,omitempty"`
	NoContinue         bool   `json:"noContinue,omitempty"`
	Resume             string `json:"resume,omitempty"`
	FireAndForget      bool   `json:"fireAndForget,omitempty"`
}

// RunResponse is the JSON response from POST /run.
type RunResponse struct {
	Result       string          `json:"result"`
	Usage        Usage           `json:"usage"`
	CostUSD      float64         `json:"costUsd"`
	Duration     float64         `json:"duration"`
	IsError      bool            `json:"isError"`
	SessionID    string          `json:"sessionId"`
	TotalCostUSD float64         `json:"totalCostUsd"`
	Turns        json.RawMessage `json:"turns,omitempty"`

	raw json.RawMessage
}

// Raw returns the full unparsed JSON response body.
func (r *RunResponse) Raw() json.RawMessage {
	return r.raw
}

// Usage holds token usage stats.
type Usage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	CacheRead    int `json:"cacheReadInputTokens"`
	CacheCreated int `json:"cacheCreationInputTokens"`
}

// Run executes a prompt via POST /run and returns the
// parsed response.
func (c *Client) Run(
	ctx context.Context,
	req *RunRequest,
) (*RunResponse, error) {
	resp, err := c.do(
		ctx, http.MethodPost, "/run", req,
	)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ctxerrors.Wrap(
			err, "read response",
		)
	}

	var rr RunResponse
	if err := json.Unmarshal(raw, &rr); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	rr.raw = raw

	return &rr, nil
}

// CancelResponse is the response from POST /run/cancel.
type CancelResponse struct {
	Status    string `json:"status"`
	Workspace string `json:"workspace"`
}

// Cancel kills a running process in the given workspace
// (empty = default).
func (c *Client) Cancel(
	ctx context.Context,
	workspace string,
) (*CancelResponse, error) {
	endpoint := "/run/cancel"

	if workspace != "" {
		endpoint += "?workspace=" +
			url.QueryEscape(workspace)
	}

	resp, err := c.do(
		ctx, http.MethodPost, endpoint, nil,
	)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var v CancelResponse
	if err := json.NewDecoder(resp.Body).Decode(
		&v,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	return &v, nil
}
