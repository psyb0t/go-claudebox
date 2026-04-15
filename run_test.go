package claudebox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestRun(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost ||
				r.URL.Path != "/run" {
				t.Errorf(
					"unexpected request: %s %s",
					r.Method, r.URL.Path,
				)
			}

			var req RunRequest

			_ = json.NewDecoder(r.Body).Decode(&req)

			if req.Prompt != "say hello" {
				t.Errorf(
					"unexpected prompt: %q",
					req.Prompt,
				)
			}

			if req.Model != "haiku" {
				t.Errorf(
					"unexpected model: %q",
					req.Model,
				)
			}

			if !req.NoContinue {
				t.Error("expected noContinue=true")
			}

			_, _ = w.Write([]byte(
				`{"result":"hello",` +
					`"usage":{"inputTokens":10,` +
					`"outputTokens":5},` +
					`"costUsd":0.001,` +
					`"sessionId":"abc"}`,
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
	if err != nil {
		t.Fatal(err)
	}

	if resp.Result != "hello" {
		t.Errorf(
			"got result %q, want hello",
			resp.Result,
		)
	}

	if resp.Usage.InputTokens != 10 {
		t.Errorf(
			"got input tokens %d, want 10",
			resp.Usage.InputTokens,
		)
	}

	if resp.Usage.OutputTokens != 5 {
		t.Errorf(
			"got output tokens %d, want 5",
			resp.Usage.OutputTokens,
		)
	}

	if resp.SessionID != "abc" {
		t.Errorf(
			"got session %q, want abc",
			resp.SessionID,
		)
	}

	if resp.Raw() == nil {
		t.Error("raw should not be nil")
	}
}

func TestRunWithAllFields(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			var req RunRequest

			_ = json.NewDecoder(r.Body).Decode(&req)

			if req.Workspace != "myproject" {
				t.Errorf(
					"workspace: got %q, want myproject",
					req.Workspace,
				)
			}

			if req.SystemPrompt != "be brief" {
				t.Errorf(
					"systemPrompt: got %q",
					req.SystemPrompt,
				)
			}

			if req.AppendSystemPrompt != "end with OK" {
				t.Errorf(
					"appendSystemPrompt: got %q",
					req.AppendSystemPrompt,
				)
			}

			if req.Effort != "low" {
				t.Errorf("effort: got %q", req.Effort)
			}

			if req.OutputFormat != "json-verbose" {
				t.Errorf(
					"outputFormat: got %q",
					req.OutputFormat,
				)
			}

			if req.Resume != "sess-123" {
				t.Errorf("resume: got %q", req.Resume)
			}

			if !req.FireAndForget {
				t.Error("expected fireAndForget=true")
			}

			_, _ = w.Write([]byte(`{"result":"ok"}`))
		},
	)

	_, err := c.Run(
		context.Background(),
		&RunRequest{
			Prompt:             "test",
			Workspace:          "myproject",
			SystemPrompt:       "be brief",
			AppendSystemPrompt: "end with OK",
			Effort:             "low",
			OutputFormat:       "json-verbose",
			Resume:             "sess-123",
			FireAndForget:      true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancel(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf(
					"unexpected method: %s",
					r.Method,
				)
			}

			ws := r.URL.Query().Get("workspace")
			if ws != "myproject" {
				t.Errorf(
					"unexpected workspace query: %q",
					ws,
				)
			}

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
	if err != nil {
		t.Fatal(err)
	}

	if resp.Status != "ok" {
		t.Errorf("got status %q", resp.Status)
	}
}

func TestCancelDefaultWorkspace(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "" {
				t.Errorf(
					"expected no query params, got %q",
					r.URL.RawQuery,
				)
			}

			_ = json.NewEncoder(w).Encode(
				CancelResponse{Status: "ok"},
			)
		},
	)

	_, err := c.Cancel(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIError409(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(
				`{"detail":"workspace busy, retry later"}`,
			))
		},
	)

	_, err := c.Run(
		context.Background(),
		&RunRequest{Prompt: "test"},
	)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf(
			"got %d, want 409",
			apiErr.StatusCode,
		)
	}
}
