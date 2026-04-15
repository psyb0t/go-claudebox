# go-claudebox
I am testing this bitch
Go client for the [claudebox](https://github.com/psyb0t/docker-claude-code) API. Lets you run Claude Code prompts, manage files, and check server status from Go.

## Features

- Run prompts (sync, fire-and-forget, resume sessions)
- Upload, download, list, and delete workspace files
- Check health, status, and cancel running jobs
- Bearer token auth
- Zero external dependencies — stdlib only
- Strict linting with zero `//nolint` directives

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
    "log"

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

    // Upload a file
    _, err = c.WriteFile(context.Background(), "notes.txt", []byte("hello"))
    if err != nil {
        log.Fatal(err)
    }

    // Read it back
    data, err := c.ReadFile(context.Background(), "notes.txt")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(data))
}
```

## API

| Method | What |
|---|---|
| `New(baseURL, ...Option)` | Create client |
| `Health(ctx)` | `GET /health` |
| `Status(ctx)` | `GET /status` — busy workspaces |
| `Run(ctx, *RunRequest)` | `POST /run` — execute prompt |
| `Cancel(ctx, workspace)` | `POST /run/cancel` |
| `ListFiles(ctx, path)` | `GET /files` or `GET /files/{path}` |
| `ReadFile(ctx, path)` | `GET /files/{path}` — raw bytes |
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
| `Effort` | `effort` | low, medium, high |
| `OutputFormat` | `outputFormat` | e.g. json-verbose |
| `NoContinue` | `noContinue` | Don't auto-continue |
| `Resume` | `resume` | Resume a previous session |
| `FireAndForget` | `fireAndForget` | Start and return immediately |

### Error handling

Non-2xx responses return `*claudebox.APIError`:

```go
var apiErr *claudebox.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("HTTP %d: %s\n", apiErr.StatusCode, apiErr.Body)
}
```

## License

MIT. Do whatever.
