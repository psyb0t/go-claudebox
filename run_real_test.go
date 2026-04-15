//go:build real

package claudebox

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealRun(t *testing.T) {
	c := realClient(t)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{
			Prompt:     "respond with just the word pong",
			Model:      "haiku",
			NoContinue: true,
			Effort:     "low",
		},
	)
	require.NoError(t, err)

	// top-level fields
	assert.Equal(t, "result", resp.Type)
	assert.Equal(t, "success", resp.Subtype)
	assert.False(t, resp.IsError)
	assert.Contains(t, resp.Result, "pong")
	assert.NotEmpty(t, resp.SessionID)
	assert.NotEmpty(t, resp.UUID)
	assert.Equal(t, "end_turn", resp.StopReason)
	assert.NotEmpty(t, resp.FastModeState)
	assert.Greater(t, resp.DurationMs, int64(0))
	assert.Greater(t, resp.DurationAPIMs, int64(0))
	assert.Greater(t, resp.TotalCostUSD, float64(0))
	assert.Greater(t, resp.NumTurns, 0)
	assert.NotNil(t, resp.Raw())

	// permissionDenials — should be present as empty
	// slice, not nil
	assert.NotNil(t, resp.PermissionDenials)

	// usage — token counts
	assert.Greater(t, resp.Usage.InputTokens, 0)
	assert.Greater(t, resp.Usage.OutputTokens, 0)
	assert.GreaterOrEqual(t,
		resp.Usage.CacheCreationInputTokens, 0,
	)
	assert.GreaterOrEqual(t,
		resp.Usage.CacheReadInputTokens, 0,
	)

	// usage — serverToolUse
	require.NotNil(t, resp.Usage.ServerToolUse,
		"serverToolUse should be present",
	)
	assert.GreaterOrEqual(t,
		resp.Usage.ServerToolUse.WebSearchRequests, 0,
	)
	assert.GreaterOrEqual(t,
		resp.Usage.ServerToolUse.WebFetchRequests, 0,
	)

	// usage — serviceTier
	assert.NotEmpty(t, resp.Usage.ServiceTier,
		"serviceTier should be set",
	)

	// usage — cacheCreation
	require.NotNil(t, resp.Usage.CacheCreation,
		"cacheCreation should be present",
	)
	assert.GreaterOrEqual(t,
		resp.Usage.CacheCreation.Ephemeral1hInputTokens, 0,
	)
	assert.GreaterOrEqual(t,
		resp.Usage.CacheCreation.Ephemeral5mInputTokens, 0,
	)

	// usage — speed
	assert.NotEmpty(t, resp.Usage.Speed,
		"speed should be set",
	)

	// usage — iterations (present even if empty)
	assert.NotNil(t, resp.Usage.Iterations)

	// modelUsage — per-model stats
	require.NotNil(t, resp.ModelUsage)
	require.Greater(t, len(resp.ModelUsage), 0)

	for model, ms := range resp.ModelUsage {
		assert.NotEmpty(t, model)
		assert.Greater(t, ms.InputTokens, 0)
		assert.Greater(t, ms.OutputTokens, 0)
		assert.GreaterOrEqual(t,
			ms.CacheReadInputTokens, 0,
		)
		assert.GreaterOrEqual(t,
			ms.CacheCreationInputTokens, 0,
		)
		assert.GreaterOrEqual(t,
			ms.WebSearchRequests, 0,
		)
		assert.Greater(t, ms.CostUSD, float64(0))
		assert.Greater(t, ms.ContextWindow, 0)
		assert.Greater(t, ms.MaxOutputTokens, 0)
	}

	// non-verbose — turns and system should be absent
	assert.Nil(t, resp.Turns)
	assert.Nil(t, resp.System)
}

func TestRealRunVerbose(t *testing.T) {
	c := realClient(t)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{
			Prompt: "run the command: echo real-test-ok. " +
				"then tell me what it said.",
			Model:        "haiku",
			NoContinue:   true,
			OutputFormat: "json-verbose",
			Effort:       "low",
		},
	)
	require.NoError(t, err)

	// top-level — same as non-verbose
	assert.Equal(t, "result", resp.Type)
	assert.Equal(t, "success", resp.Subtype)
	assert.False(t, resp.IsError)
	assert.Contains(t, resp.Result, "real-test-ok")
	assert.NotEmpty(t, resp.SessionID)
	assert.NotEmpty(t, resp.UUID)
	assert.Equal(t, "end_turn", resp.StopReason)
	assert.NotEmpty(t, resp.FastModeState)
	assert.Greater(t, resp.DurationMs, int64(0))
	assert.Greater(t, resp.DurationAPIMs, int64(0))
	assert.Greater(t, resp.TotalCostUSD, float64(0))
	assert.Greater(t, resp.NumTurns, 0)

	// usage — full check
	assert.Greater(t, resp.Usage.InputTokens, 0)
	assert.Greater(t, resp.Usage.OutputTokens, 0)
	require.NotNil(t, resp.Usage.ServerToolUse)
	assert.NotEmpty(t, resp.Usage.ServiceTier)
	require.NotNil(t, resp.Usage.CacheCreation)
	assert.NotEmpty(t, resp.Usage.Speed)
	assert.NotNil(t, resp.Usage.Iterations)

	// modelUsage
	require.NotNil(t, resp.ModelUsage)
	require.Greater(t, len(resp.ModelUsage), 0)

	for model, ms := range resp.ModelUsage {
		assert.NotEmpty(t, model)
		assert.Greater(t, ms.CostUSD, float64(0))
		assert.Greater(t, ms.ContextWindow, 0)
		assert.Greater(t, ms.MaxOutputTokens, 0)
	}

	// turns — must have all three block types
	require.NotEmpty(t, resp.Turns)

	hasToolUse := false
	hasToolResult := false
	hasText := false

	for _, turn := range resp.Turns {
		assert.NotEmpty(t, turn.Role)
		assert.Contains(t,
			[]string{"assistant", "tool_result"},
			turn.Role,
		)

		for _, block := range turn.Content {
			assert.NotEmpty(t, block.Type)

			switch block.Type {
			case "tool_use":
				hasToolUse = true
				assert.NotEmpty(t, block.ID,
					"tool_use must have id",
				)
				assert.NotEmpty(t, block.Name,
					"tool_use must have name",
				)
				require.NotNil(t, block.Input,
					"tool_use must have input",
				)

				var parsed map[string]any
				require.NoError(t,
					json.Unmarshal(block.Input, &parsed),
					"tool_use input must be valid JSON",
				)

				// tool_use should not have tool_result fields
				assert.Empty(t, block.ToolUseID)
				assert.Empty(t, block.Content)
				assert.Empty(t, block.Text)

			case "tool_result":
				hasToolResult = true
				assert.NotEmpty(t, block.ToolUseID,
					"tool_result must have toolUseId",
				)
				assert.NotEmpty(t, block.Content,
					"tool_result must have content",
				)

				// tool_result should not have tool_use fields
				assert.Empty(t, block.ID)
				assert.Empty(t, block.Name)
				assert.Nil(t, block.Input)
				assert.Empty(t, block.Text)

			case "text":
				hasText = true
				assert.NotEmpty(t, block.Text,
					"text block must have text",
				)

				// text should not have tool fields
				assert.Empty(t, block.ID)
				assert.Empty(t, block.Name)
				assert.Nil(t, block.Input)
				assert.Empty(t, block.ToolUseID)
				assert.Empty(t, block.Content)
			}
		}
	}

	assert.True(t, hasToolUse,
		"should have tool_use blocks",
	)
	assert.True(t, hasToolResult,
		"should have tool_result blocks",
	)
	assert.True(t, hasText,
		"should have text blocks",
	)

	// tool_use IDs should match tool_result toolUseIds
	toolUseIDs := map[string]bool{}
	toolResultIDs := map[string]bool{}

	for _, turn := range resp.Turns {
		for _, block := range turn.Content {
			if block.Type == "tool_use" {
				toolUseIDs[block.ID] = true
			}

			if block.Type == "tool_result" {
				toolResultIDs[block.ToolUseID] = true
			}
		}
	}

	for id := range toolResultIDs {
		assert.True(t, toolUseIDs[id],
			"tool_result references unknown tool_use id: %s",
			id,
		)
	}

	// system info
	require.NotNil(t, resp.System)
	assert.NotEmpty(t, resp.System.SessionID)
	assert.Equal(t,
		resp.SessionID, resp.System.SessionID,
		"system.sessionId should match top-level sessionId",
	)
	assert.NotEmpty(t, resp.System.Model)
	assert.Contains(t, resp.System.Model, "haiku",
		"model should contain haiku since we requested it",
	)
	assert.NotEmpty(t, resp.System.Cwd)
	require.NotEmpty(t, resp.System.Tools)
	assert.Contains(t, resp.System.Tools, "Bash",
		"tools should contain Bash",
	)
	assert.Contains(t, resp.System.Tools, "Read",
		"tools should contain Read",
	)
	assert.Contains(t, resp.System.Tools, "Write",
		"tools should contain Write",
	)
}

func TestRealRunIsError(t *testing.T) {
	c := realClient(t)

	resp, err := c.Run(
		context.Background(),
		&RunRequest{
			Prompt:     "test",
			Model:      "haiku",
			NoContinue: true,
			Resume:     "nonexistent-session-id-xyz",
		},
	)

	// server might return HTTP error or isError in body
	if err != nil {
		return
	}

	if resp.IsError {
		assert.Equal(t, "error", resp.Subtype)
	}
}

func TestRealRunWithJSONSchema(t *testing.T) {
	c := realClient(t)

	schema := `{
		"type": "object",
		"properties": {
			"greeting": {"type": "string"},
			"count": {"type": "integer"}
		},
		"required": ["greeting", "count"]
	}`

	resp, err := c.Run(
		context.Background(),
		&RunRequest{
			Prompt: "respond with JSON only: " +
				"a greeting and a number 1-10",
			Model:      "haiku",
			NoContinue: true,
			JSONSchema: schema,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "result", resp.Type)
}

func TestRealCancel404(t *testing.T) {
	c := realClient(t)

	_, err := c.Cancel(
		context.Background(),
		"nonexistent-workspace-xyz",
	)
	require.Error(t, err)
}
