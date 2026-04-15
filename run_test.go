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

func TestRun(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/run", r.URL.Path)
			assert.Equal(t,
				"application/json",
				r.Header.Get("Content-Type"),
			)

			var req RunRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, "say hello", req.Prompt)
			assert.Equal(t, "haiku", req.Model)
			assert.True(t, req.NoContinue)

			_, _ = w.Write([]byte(
				`{"type":"result",` +
					`"subtype":"success",` +
					`"result":"hello",` +
					`"isError":false,` +
					`"numTurns":1,` +
					`"durationMs":2500,` +
					`"durationApiMs":2300,` +
					`"stopReason":"end_turn",` +
					`"sessionId":"abc",` +
					`"totalCostUsd":0.001,` +
					`"uuid":"u-123",` +
					`"fastModeState":"off",` +
					`"usage":{"inputTokens":10,` +
					`"outputTokens":5,` +
					`"cacheCreationInputTokens":100,` +
					`"cacheReadInputTokens":200}}`,
			))
		},
	)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{
			Prompt:     "say hello",
			Model:      "haiku",
			NoContinue: true,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, "result", resp.Type)
	assert.Equal(t, "success", resp.Subtype)
	assert.Equal(t, "hello", resp.Result)
	assert.False(t, resp.IsError)
	assert.Equal(t, 1, resp.NumTurns)
	assert.Equal(t, int64(2500), resp.DurationMs)
	assert.Equal(t, int64(2300), resp.DurationAPIMs)
	assert.Equal(t, "end_turn", resp.StopReason)
	assert.Equal(t, "abc", resp.SessionID)
	assert.Equal(t, "u-123", resp.UUID)
	assert.Equal(t, "off", resp.FastModeState)
	assert.Equal(t, 10, resp.Usage.InputTokens)
	assert.Equal(t, 5, resp.Usage.OutputTokens)
	assert.Equal(t,
		100, resp.Usage.CacheCreationInputTokens,
	)
	assert.Equal(t,
		200, resp.Usage.CacheReadInputTokens,
	)
	assert.NotNil(t, resp.Raw())
}

func TestRunWithAllFields(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			var req RunRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, "myproject", req.Workspace)
			assert.Equal(t, "be brief", req.SystemPrompt)
			assert.Equal(t,
				"end with OK", req.AppendSystemPrompt,
			)
			assert.Equal(t,
				`{"type":"object"}`, req.JSONSchema,
			)
			assert.Equal(t, "low", req.Effort)
			assert.Equal(t,
				"json-verbose", req.OutputFormat,
			)
			assert.Equal(t, "sess-123", req.Resume)
			assert.True(t, req.FireAndForget)

			_, _ = w.Write([]byte(
				`{"type":"result","result":"ok"}`,
			))
		},
	)

	_, err := c.Run(
		context.Background(),
		&RunRequest{
			Prompt:             "test",
			Workspace:          "myproject",
			SystemPrompt:       "be brief",
			AppendSystemPrompt: "end with OK",
			JSONSchema:         `{"type":"object"}`,
			Effort:             "low",
			OutputFormat:       "json-verbose",
			Resume:             "sess-123",
			FireAndForget:      true,
		},
	)
	require.NoError(t, err)
}

func TestRunMinimalResponse(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t,
			`{"type":"result","result":"hi"}`,
		),
	)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{Prompt: "hi"},
	)
	require.NoError(t, err)

	assert.Equal(t, "hi", resp.Result)
	assert.Nil(t, resp.System)
	assert.Nil(t, resp.Turns)
	assert.Nil(t, resp.ModelUsage)
	assert.Nil(t, resp.PermissionDenials)
	assert.Nil(t, resp.Usage.ServerToolUse)
	assert.Nil(t, resp.Usage.CacheCreation)
}

func TestRunIsErrorTrue(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t,
			`{"type":"result",`+
				`"subtype":"error",`+
				`"isError":true,`+
				`"result":"something broke"}`,
		),
	)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{Prompt: "test"},
	)
	require.NoError(t, err)
	assert.True(t, resp.IsError)
	assert.Equal(t, "error", resp.Subtype)
	assert.Equal(t, "something broke", resp.Result)
}

func TestRunInvalidJSON(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `{not json`),
	)

	_, err := c.Run(
		context.Background(),
		&RunRequest{Prompt: "test"},
	)
	require.Error(t, err)
}

func TestRunPermissionDenials(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t,
			`{"type":"result","result":"ok",`+
				`"permissionDenials":[`+
				`{"tool":"Bash","reason":"denied"}`+
				`]}`,
		),
	)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{Prompt: "test"},
	)
	require.NoError(t, err)
	require.Len(t, resp.PermissionDenials, 1)

	var denial struct {
		Tool   string `json:"tool"`
		Reason string `json:"reason"`
	}

	require.NoError(t,
		json.Unmarshal(
			resp.PermissionDenials[0], &denial,
		),
	)
	assert.Equal(t, "Bash", denial.Tool)
}

func TestRunVerbose(t *testing.T) {
	verboseJSON := `{
		"type":"result",
		"subtype":"success",
		"isError":false,
		"durationMs":10225,
		"durationApiMs":9982,
		"numTurns":4,
		"result":"done",
		"stopReason":"end_turn",
		"sessionId":"sess-v",
		"totalCostUsd":0.017,
		"uuid":"u-verbose",
		"fastModeState":"off",
		"usage":{
			"inputTokens":18,
			"outputTokens":909,
			"cacheCreationInputTokens":6251,
			"cacheReadInputTokens":46279,
			"serverToolUse":{
				"webSearchRequests":2,
				"webFetchRequests":1
			},
			"serviceTier":"standard",
			"cacheCreation":{
				"ephemeral1hInputTokens":6251,
				"ephemeral5mInputTokens":42
			},
			"inferenceGeo":"us-east-1",
			"speed":"standard"
		},
		"modelUsage":{
			"claude-haiku-4-5-20251001":{
				"inputTokens":18,
				"outputTokens":909,
				"cacheReadInputTokens":46279,
				"cacheCreationInputTokens":6251,
				"webSearchRequests":2,
				"costUSD":0.017,
				"contextWindow":200000,
				"maxOutputTokens":32000
			}
		},
		"turns":[
			{
				"role":"assistant",
				"content":[{
					"type":"tool_use",
					"id":"toolu_01A",
					"name":"Bash",
					"input":{"command":"ls -la"}
				}]
			},
			{
				"role":"tool_result",
				"content":[{
					"type":"tool_result",
					"toolUseId":"toolu_01A",
					"isError":false,
					"content":"file1.txt\nfile2.txt"
				}]
			},
			{
				"role":"assistant",
				"content":[{
					"type":"tool_use",
					"id":"toolu_01B",
					"name":"Read",
					"input":{"file_path":"file1.txt"}
				}]
			},
			{
				"role":"tool_result",
				"content":[{
					"type":"tool_result",
					"toolUseId":"toolu_01B",
					"isError":false,
					"content":"hello world",
					"truncated":true,
					"totalLength":5000,
					"sha256":"abc123"
				}]
			},
			{
				"role":"assistant",
				"content":[{
					"type":"text",
					"text":"done"
				}]
			}
		],
		"system":{
			"sessionId":"sess-v",
			"model":"claude-haiku-4-5-20251001",
			"cwd":"/workspaces",
			"tools":["Bash","Read","Write"]
		}
	}`

	c, _ := testServer(t,
		jsonHandler(t, verboseJSON),
	)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{
			Prompt:       "test verbose",
			OutputFormat: "json-verbose",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 4, resp.NumTurns)

	// turns
	require.Len(t, resp.Turns, 5)

	// turn 0: assistant tool_use
	assert.Equal(t, "assistant", resp.Turns[0].Role)
	require.Len(t, resp.Turns[0].Content, 1)

	b0 := resp.Turns[0].Content[0]
	assert.Equal(t, "tool_use", b0.Type)
	assert.Equal(t, "toolu_01A", b0.ID)
	assert.Equal(t, "Bash", b0.Name)
	require.NotNil(t, b0.Input)

	var input struct {
		Command string `json:"command"`
	}
	require.NoError(t,
		json.Unmarshal(b0.Input, &input),
	)
	assert.Equal(t, "ls -la", input.Command)

	// turn 1: tool_result
	assert.Equal(t,
		"tool_result", resp.Turns[1].Role,
	)

	tr := resp.Turns[1].Content[0]
	assert.Equal(t, "tool_result", tr.Type)
	assert.Equal(t, "toolu_01A", tr.ToolUseID)
	assert.False(t, tr.IsError)
	assert.Equal(t,
		"file1.txt\nfile2.txt", tr.Content,
	)

	// turn 3: truncated tool_result
	tr3 := resp.Turns[3].Content[0]
	assert.True(t, tr3.Truncated)
	assert.Equal(t, 5000, tr3.TotalLength)
	assert.Equal(t, "abc123", tr3.SHA256)

	// turn 4: assistant text
	last := resp.Turns[4].Content[0]
	assert.Equal(t, "text", last.Type)
	assert.Equal(t, "done", last.Text)

	// system
	require.NotNil(t, resp.System)
	assert.Equal(t, "sess-v", resp.System.SessionID)
	assert.Equal(t,
		"claude-haiku-4-5-20251001",
		resp.System.Model,
	)
	assert.Equal(t, "/workspaces", resp.System.Cwd)
	assert.Equal(t,
		[]string{"Bash", "Read", "Write"},
		resp.System.Tools,
	)

	// usage nested structs
	require.NotNil(t, resp.Usage.ServerToolUse)
	assert.Equal(t,
		2, resp.Usage.ServerToolUse.WebSearchRequests,
	)
	assert.Equal(t,
		1, resp.Usage.ServerToolUse.WebFetchRequests,
	)
	assert.Equal(t, "standard", resp.Usage.ServiceTier)

	require.NotNil(t, resp.Usage.CacheCreation)
	assert.Equal(t,
		6251,
		resp.Usage.CacheCreation.Ephemeral1hInputTokens,
	)
	assert.Equal(t,
		42,
		resp.Usage.CacheCreation.Ephemeral5mInputTokens,
	)
	assert.Equal(t,
		"us-east-1", resp.Usage.InferenceGeo,
	)
	assert.Equal(t, "standard", resp.Usage.Speed)

	// model usage
	ms, ok := resp.ModelUsage["claude-haiku-4-5-20251001"]
	require.True(t, ok)
	assert.Equal(t, 18, ms.InputTokens)
	assert.Equal(t, 909, ms.OutputTokens)
	assert.Equal(t, 0.017, ms.CostUSD)
	assert.Equal(t, 200000, ms.ContextWindow)
	assert.Equal(t, 32000, ms.MaxOutputTokens)
	assert.Equal(t, 2, ms.WebSearchRequests)
}

func TestRunVerboseMultipleBlocksPerTurn(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `{
			"type":"result","result":"ok",
			"turns":[{
				"role":"assistant",
				"content":[
					{"type":"text","text":"let me check"},
					{"type":"tool_use","id":"t1","name":"Bash",
					 "input":{"command":"ls"}},
					{"type":"tool_use","id":"t2","name":"Read",
					 "input":{"file_path":"x.go"}}
				]
			}]
		}`),
	)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{Prompt: "test"},
	)
	require.NoError(t, err)
	require.Len(t, resp.Turns, 1)

	blocks := resp.Turns[0].Content
	require.Len(t, blocks, 3)
	assert.Equal(t, "text", blocks[0].Type)
	assert.Equal(t, "let me check", blocks[0].Text)
	assert.Equal(t, "tool_use", blocks[1].Type)
	assert.Equal(t, "Bash", blocks[1].Name)
	assert.Equal(t, "tool_use", blocks[2].Type)
	assert.Equal(t, "Read", blocks[2].Name)
}

func TestRunVerboseToolResultError(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `{
			"type":"result","result":"failed",
			"turns":[{
				"role":"tool_result",
				"content":[{
					"type":"tool_result",
					"toolUseId":"t1",
					"isError":true,
					"content":"command not found: foobar"
				}]
			}]
		}`),
	)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{Prompt: "test"},
	)
	require.NoError(t, err)

	tr := resp.Turns[0].Content[0]
	assert.True(t, tr.IsError)
	assert.Equal(t,
		"command not found: foobar", tr.Content,
	)
}

func TestCancel(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t,
				"myproject",
				r.URL.Query().Get("workspace"),
			)

			_ = json.NewEncoder(w).Encode(
				CancelResponse{
					Status:    "ok",
					Workspace: "/workspaces/myproject",
				},
			)
		},
	)

	resp, err := c.Cancel(
		context.Background(), "myproject",
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t,
		"/workspaces/myproject", resp.Workspace,
	)
}

func TestCancelWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		wantQuery string
	}{
		{
			"default (empty)",
			"",
			"",
		},
		{
			"simple",
			"myproject",
			"workspace=myproject",
		},
		{
			"needs escaping",
			"my project/sub dir",
			"workspace=my+project%2Fsub+dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testServer(t,
				func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t,
						tt.wantQuery, r.URL.RawQuery,
					)

					_ = json.NewEncoder(w).Encode(
						CancelResponse{Status: "ok"},
					)
				},
			)

			_, err := c.Cancel(
				context.Background(), tt.workspace,
			)
			require.NoError(t, err)
		})
	}
}

func TestCancelErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			"not found",
			http.StatusNotFound,
			`{"detail":"no running process"}`,
		},
		{
			"unauthorized",
			http.StatusUnauthorized,
			`{"detail":"unauthorized"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testServer(t,
				errorHandler(t, tt.status, tt.body),
			)

			_, err := c.Cancel(
				context.Background(), "ws",
			)
			require.Error(t, err)

			var apiErr *APIError
			require.True(t,
				errors.As(err, &apiErr),
			)
			assert.Equal(t,
				tt.status, apiErr.StatusCode,
			)
		})
	}
}

func TestCancelInvalidJSON(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `not json`),
	)

	_, err := c.Cancel(context.Background(), "ws")
	require.Error(t, err)
}

func TestRunHTTPErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"conflict", http.StatusConflict},
		{"unauthorized", http.StatusUnauthorized},
		{"internal error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testServer(t,
				errorHandler(t, tt.status,
					`{"detail":"error"}`,
				),
			)

			_, err := c.Run(
				context.Background(),
				&RunRequest{Prompt: "test"},
			)
			require.Error(t, err)

			var apiErr *APIError
			require.True(t,
				errors.As(err, &apiErr),
			)
			assert.Equal(t,
				tt.status, apiErr.StatusCode,
			)
		})
	}
}
