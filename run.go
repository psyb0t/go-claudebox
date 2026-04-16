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
	RunID         string                `json:"runId,omitempty"`
	Type          string                `json:"type"`
	Subtype       string                `json:"subtype"`
	Result        string                `json:"result"`
	IsError       bool                  `json:"isError"`
	NumTurns      int                   `json:"numTurns"`
	DurationMs    int64                 `json:"durationMs"`
	DurationAPIMs int64                 `json:"durationApiMs"`
	StopReason    string                `json:"stopReason"`
	SessionID     string                `json:"sessionId"`
	TotalCostUSD  float64               `json:"totalCostUsd"`
	UUID          string                `json:"uuid"`
	FastModeState string                `json:"fastModeState"`
	Usage         Usage                 `json:"usage"`
	ModelUsage    map[string]ModelStats `json:"modelUsage,omitempty"`
	Turns         []Turn                `json:"turns,omitempty"`
	System        *SystemInfo           `json:"system,omitempty"`

	// PermissionDenials lists any permission denials
	// that occurred during the run.
	PermissionDenials []json.RawMessage `json:"permissionDenials,omitempty"`

	raw json.RawMessage
}

// Raw returns the full unparsed JSON response body.
func (r *RunResponse) Raw() json.RawMessage {
	return r.raw
}

// Usage holds token usage stats.
type Usage struct {
	InputTokens              int               `json:"inputTokens"`
	OutputTokens             int               `json:"outputTokens"`
	CacheCreationInputTokens int               `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int               `json:"cacheReadInputTokens"`
	ServerToolUse            *ServerToolUse    `json:"serverToolUse,omitempty"`
	ServiceTier              string            `json:"serviceTier,omitempty"`
	CacheCreation            *CacheCreation    `json:"cacheCreation,omitempty"`
	InferenceGeo             string            `json:"inferenceGeo,omitempty"`
	Iterations               []json.RawMessage `json:"iterations,omitempty"`
	Speed                    string            `json:"speed,omitempty"`
}

// ServerToolUse holds server-side tool usage counters.
type ServerToolUse struct {
	WebSearchRequests int `json:"webSearchRequests"`
	WebFetchRequests  int `json:"webFetchRequests"`
}

// CacheCreation holds cache creation token breakdown.
type CacheCreation struct {
	Ephemeral1hInputTokens int `json:"ephemeral1hInputTokens"`
	Ephemeral5mInputTokens int `json:"ephemeral5mInputTokens"`
}

// ModelStats holds per-model usage and cost info.
type ModelStats struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	WebSearchRequests        int     `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"` //nolint:tagliatelle // server sends costUSD
	ContextWindow            int     `json:"contextWindow"`
	MaxOutputTokens          int     `json:"maxOutputTokens"`
}

// Turn represents a conversation turn in verbose output.
type Turn struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock is a single block within a turn.
// The Type field determines which other fields are set.
//
// Type "text": Text is set.
// Type "tool_use": ID, Name, Input are set.
// Type "tool_result": ToolUseID, IsError, Content,
// and optionally Truncated/TotalLength/SHA256 are set.
type ContentBlock struct {
	Type string `json:"type"`

	// text block fields
	Text string `json:"text,omitempty"`

	// tool_use block fields
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// tool_result block fields
	ToolUseID   string `json:"toolUseId,omitempty"`
	IsError     bool   `json:"isError,omitempty"`
	Content     string `json:"content,omitempty"`
	Truncated   bool   `json:"truncated,omitempty"`
	TotalLength int    `json:"totalLength,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

// SystemInfo holds session metadata from verbose output.
type SystemInfo struct {
	SessionID string   `json:"sessionId"`
	Model     string   `json:"model"`
	Cwd       string   `json:"cwd"`
	Tools     []string `json:"tools"`
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

// asyncRunRequest wraps RunRequest with the async flag
// for internal use by RunAsync.
type asyncRunRequest struct {
	RunRequest
	Async bool `json:"async"`
}

// AsyncRunResponse is returned by POST /run when async
// mode is enabled.
type AsyncRunResponse struct {
	RunID     string `json:"runId"`
	Workspace string `json:"workspace"`
	Status    string `json:"status"`
}

// RunAsync starts an async run and returns immediately.
// Poll with RunResult to get the outcome.
func (c *Client) RunAsync(
	ctx context.Context,
	req *RunRequest,
) (*AsyncRunResponse, error) {
	body := asyncRunRequest{
		RunRequest: *req,
		Async:      true,
	}

	resp, err := c.do(
		ctx, http.MethodPost, "/run", &body,
	)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var v AsyncRunResponse
	if err := json.NewDecoder(resp.Body).Decode(
		&v,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	return &v, nil
}

// RunResultResponse is the response from
// GET /run/result. Check Status to determine the
// outcome:
//   - "running": still in progress
//   - "completed": Result is populated
//   - "cancelled": run was cancelled
//   - "failed": Error contains the failure message
type RunResultResponse struct {
	RunID     string `json:"runId"`
	Workspace string `json:"workspace,omitempty"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`

	// Result is set when Status is "completed".
	Result *RunResponse `json:"-"`
}

// decodeRunResult parses raw JSON into a
// RunResultResponse, handling both in-progress status
// responses and completed full results.
func decodeRunResult(raw []byte) (
	*RunResultResponse, error,
) {
	var probe struct {
		RunID     string `json:"runId"`
		Workspace string `json:"workspace"`
		Status    string `json:"status"`
		Error     string `json:"error"`
	}

	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	rr := &RunResultResponse{
		RunID:     probe.RunID,
		Workspace: probe.Workspace,
		Status:    probe.Status,
		Error:     probe.Error,
	}

	if probe.Status == "running" ||
		probe.Status == "cancelled" ||
		probe.Status == "failed" {
		return rr, nil
	}

	// completed — decode full result
	var result RunResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode result",
		)
	}

	result.raw = raw
	rr.Status = "completed"
	rr.Result = &result

	return rr, nil
}

// RunResult polls for the result of an async run.
// Results are purged after first read (except running).
func (c *Client) RunResult(
	ctx context.Context,
	runID string,
) (*RunResultResponse, error) {
	endpoint := "/run/result?runId=" +
		url.QueryEscape(runID)

	resp, err := c.do(
		ctx, http.MethodGet, endpoint, nil,
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

	return decodeRunResult(raw)
}

// CancelResponse is the response from POST /run/cancel.
type CancelResponse struct {
	Status    string `json:"status"`
	RunID     string `json:"runId,omitempty"`
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

// CancelRun cancels a running async job by run ID.
func (c *Client) CancelRun(
	ctx context.Context,
	runID string,
) (*CancelResponse, error) {
	endpoint := "/run/cancel?runId=" +
		url.QueryEscape(runID)

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

	var cr CancelResponse
	if err := json.NewDecoder(resp.Body).Decode(
		&cr,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	return &cr, nil
}
