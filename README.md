# go-claudebox

Go client for the [claudebox](https://github.com/psyb0t/docker-claudebox) API. Lets you run Claude Code prompts, manage files, and check server status from Go.

## Features

- Run prompts (sync, async, fire-and-forget, resume sessions)
- Async job management (start, poll, cancel by run ID)
- Full verbose output with typed turns, tool calls, and tool results
- Upload, download, list, and delete workspace files
- Check health, status, and cancel running jobs
- Bearer token auth
- **Mockable** — `Claudebox` interface for easy testing
- Minimal dependencies — [ctxerrors](https://github.com/psyb0t/ctxerrors), [common-go](https://github.com/psyb0t/common-go), [testify](https://github.com/stretchr/testify), [godotenv](https://github.com/joho/godotenv)
- Integration tests against a live claudebox instance (`go test -tags=real`)
- Strict linting

## Install

```bash
go get github.com/psyb0t/go-claudebox@latest
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "os"
    "time"

    claudebox "github.com/psyb0t/go-claudebox"
)

func main() {
    c := claudebox.New("http://localhost:8080", claudebox.WithToken("my-secret"))

    // Check health
    h, err := c.Health(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(h.Status)

    // Run a prompt
    resp, err := c.Run(context.Background(), &claudebox.RunRequest{
        Prompt: "list all files in the project",
        Model:  "sonnet",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.Result)
    fmt.Printf("cost: $%.4f, turns: %d\n", resp.TotalCostUSD, resp.NumTurns)

    // Run with verbose output (includes tool call history)
    verbose, err := c.Run(context.Background(), &claudebox.RunRequest{
        Prompt:       "read main.go and explain it",
        Model:        "haiku",
        OutputFormat: "json-verbose",
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, turn := range verbose.Turns {
        for _, block := range turn.Content {
            switch block.Type {
            case "tool_use":
                fmt.Printf("[%s] %s\n", block.Name, string(block.Input))
            case "tool_result":
                fmt.Printf("  → %s\n", block.Content)
            case "text":
                fmt.Println(block.Text)
            }
        }
    }

    // Run async — returns immediately
    async, err := c.RunAsync(context.Background(), &claudebox.RunRequest{
        Prompt:    "refactor the entire codebase",
        Workspace: "myproject",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("started run %s\n", async.RunID)

    // Cancel by run ID (if needed):
    // _, _ = c.CancelRun(context.Background(), async.RunID)

    // Poll for result
    for {
        res, err := c.RunResult(context.Background(), async.RunID)
        if err != nil {
            log.Fatal(err)
        }
        if res.Status == "running" {
            time.Sleep(5 * time.Second)
            continue
        }
        if res.Status == "completed" {
            fmt.Println(res.Result.Result)
        }
        if res.Status == "failed" {
            fmt.Printf("failed: %s\n", res.Error)
        }
        break
    }

    // Upload a file
    _, err = c.WriteFile(context.Background(), "notes.txt", []byte("hello"))
    if err != nil {
        log.Fatal(err)
    }

    // Read it back (streaming — caller closes Body)
    file, err := c.ReadFile(context.Background(), "notes.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Body.Close()
    fmt.Printf("type: %s, size: %d\n", file.ContentType, file.ContentLength)
    io.Copy(os.Stdout, file.Body)
}
```

## Mocking

The `Claudebox` interface makes testing straightforward:

```go
type mockClient struct {
    runFunc func(ctx context.Context, req *claudebox.RunRequest) (*claudebox.RunResponse, error)
}

func (m *mockClient) Run(ctx context.Context, req *claudebox.RunRequest) (*claudebox.RunResponse, error) {
    return m.runFunc(ctx, req)
}

func (m *mockClient) Health(context.Context) (*claudebox.HealthResponse, error) {
    return &claudebox.HealthResponse{Status: "ok"}, nil
}

// ... implement other methods as needed

func TestMyService(t *testing.T) {
    mock := &mockClient{
        runFunc: func(_ context.Context, req *claudebox.RunRequest) (*claudebox.RunResponse, error) {
            return &claudebox.RunResponse{
                Result:  "mocked response",
                IsError: false,
            }, nil
        },
    }

    svc := NewMyService(mock) // your code accepts claudebox.Claudebox
    // test away
}
```

## API

| Method | What |
|---|---|
| `New(baseURL, ...Option)` | Create client |
| `Health(ctx)` | `GET /health` |
| `Status(ctx)` | `GET /status` — busy workspaces + async runs |
| `Run(ctx, *RunRequest)` | `POST /run` — execute prompt (sync) |
| `RunAsync(ctx, *RunRequest)` | `POST /run` — start async job |
| `RunResult(ctx, runID)` | `GET /run/result` — poll async result |
| `Cancel(ctx, workspace)` | `POST /run/cancel` — by workspace |
| `CancelRun(ctx, runID)` | `POST /run/cancel` — by run ID |
| `ListFiles(ctx, path)` | `GET /files` or `GET /files/{path}` |
| `ReadFile(ctx, path)` | `GET /files/{path}` — streaming download |
| `WriteFile(ctx, path, content)` | `PUT /files/{path}` |
| `DeleteFile(ctx, path)` | `DELETE /files/{path}` |

### Options

| Option | What |
|---|---|
| `WithToken(token)` | Set Bearer token for auth |
| `WithHTTPClient(hc)` | Override default `http.Client` (default: 10min timeout) |

### RunRequest fields

| Field | JSON | What |
|---|---|---|
| `Prompt` | `prompt` | The prompt to run |
| `Workspace` | `workspace` | Target workspace |
| `Model` | `model` | Model to use (sonnet, opus, haiku) |
| `SystemPrompt` | `systemPrompt` | Override system prompt |
| `AppendSystemPrompt` | `appendSystemPrompt` | Append to system prompt |
| `JSONSchema` | `jsonSchema` | Constrain output to schema |
| `Effort` | `effort` | low, medium, high, max |
| `OutputFormat` | `outputFormat` | json (default) or json-verbose |
| `NoContinue` | `noContinue` | Don't auto-continue |
| `Resume` | `resume` | Resume a previous session |
| `FireAndForget` | `fireAndForget` | Start and return immediately |

### AsyncRunResponse fields

Returned by `RunAsync`:

| Field | JSON | What |
|---|---|---|
| `RunID` | `runId` | Unique run identifier for polling |
| `Workspace` | `workspace` | Resolved workspace path |
| `Status` | `status` | Always "running" |

### RunResultResponse fields

Returned by `RunResult`:

| Field | JSON | What |
|---|---|---|
| `RunID` | `runId` | The run identifier |
| `Workspace` | `workspace` | Workspace path (non-completed) |
| `Status` | `status` | running, completed, cancelled, failed |
| `Error` | `error` | Error message (failed only) |
| `Result` | — | Full `*RunResponse` (completed only) |

Results are purged server-side after first read (except running). Unread results expire after 6 hours.

### RunResponse fields

| Field | JSON | What |
|---|---|---|
| `RunID` | `runId` | Run identifier (set for async results) |
| `Type` | `type` | Always "result" |
| `Subtype` | `subtype` | "success" or "error" |
| `Result` | `result` | The response text |
| `IsError` | `isError` | Whether the run errored |
| `NumTurns` | `numTurns` | Number of conversation turns |
| `DurationMs` | `durationMs` | Total duration in ms |
| `DurationAPIMs` | `durationApiMs` | API call duration in ms |
| `StopReason` | `stopReason` | Why the run stopped (e.g. "end_turn") |
| `SessionID` | `sessionId` | Session ID for resuming |
| `TotalCostUSD` | `totalCostUsd` | Total cost in USD |
| `UUID` | `uuid` | Unique run identifier |
| `FastModeState` | `fastModeState` | Fast mode state ("off", "on") |
| `Usage` | `usage` | Token usage (see Usage) |
| `ModelUsage` | `modelUsage` | Per-model stats map (see ModelStats) |
| `Turns` | `turns` | Conversation turns (json-verbose only) |
| `System` | `system` | Session metadata (json-verbose only) |
| `PermissionDenials` | `permissionDenials` | Any permission denials |

### Usage fields

| Field | What |
|---|---|
| `InputTokens` | Input token count |
| `OutputTokens` | Output token count |
| `CacheCreationInputTokens` | Tokens used to create cache |
| `CacheReadInputTokens` | Tokens read from cache |
| `ServerToolUse` | Web search/fetch counters (`WebSearchRequests`, `WebFetchRequests`) |
| `ServiceTier` | Service tier (e.g. "standard") |
| `CacheCreation` | Cache creation breakdown (`Ephemeral1hInputTokens`, `Ephemeral5mInputTokens`) |
| `InferenceGeo` | Inference region (e.g. "us-east-1") |
| `Iterations` | Per-iteration breakdown (raw JSON) |
| `Speed` | Speed tier (e.g. "standard") |

### ModelStats fields

Per-model usage in `ModelUsage` map (keyed by model ID like `"claude-haiku-4-5-20251001"`):

| Field | What |
|---|---|
| `InputTokens` | Input tokens for this model |
| `OutputTokens` | Output tokens for this model |
| `CacheReadInputTokens` | Tokens read from cache |
| `CacheCreationInputTokens` | Tokens used to create cache |
| `WebSearchRequests` | Web search requests made |
| `CostUSD` | Cost in USD for this model |
| `ContextWindow` | Context window size |
| `MaxOutputTokens` | Max output tokens |

### Turn and ContentBlock

Verbose output includes `[]Turn`, each with `Role` ("assistant" or "tool_result") and `[]ContentBlock`.

Content block types:
- **text**: `Text` field set
- **tool_use**: `ID`, `Name`, `Input` (json.RawMessage) set
- **tool_result**: `ToolUseID`, `IsError`, `Content` set. Optionally `Truncated`, `TotalLength`, `SHA256` for large results.

### ReadFileResponse

`ReadFile` returns a streaming response instead of buffering the entire file in memory:

| Field | Type | What |
|---|---|---|
| `ContentType` | `string` | MIME type (e.g. "text/plain", "application/octet-stream") |
| `ContentLength` | `int64` | File size in bytes, or -1 if unknown |
| `Body` | `io.ReadCloser` | Streaming file data — caller must close |

### Error handling

Non-2xx responses return `*claudebox.APIError`:

```go
var apiErr *claudebox.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("HTTP %d: %s\n", apiErr.StatusCode, apiErr.Body)
}
```

## Testing

Unit tests run without any external dependencies:

```bash
make test
# or: go test -race ./...
```

Integration tests run against a live claudebox instance. Create `.env.test` with your instance details:

```bash
CLAUDEBOX_URL=http://localhost:8080
CLAUDEBOX_TOKEN=your-api-token
```

Then run:

```bash
make test-with-real
# or: go test -race -tags=real -timeout=5m ./...
```

The integration tests verify every response field is properly deserialized — token counts, model usage, cost, turns with tool calls, system info, content types, the works.

## License

MIT. Do whatever.
