package claudebox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer(
	t *testing.T,
	handler http.HandlerFunc,
) (*Client, *httptest.Server) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return New(srv.URL), srv
}

func testServerWithToken(
	t *testing.T,
	token string,
	handler http.HandlerFunc,
) (*Client, *httptest.Server) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return New(srv.URL, WithToken(token)), srv
}

func jsonHandler(
	t *testing.T,
	body string,
) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}
}

func errorHandler(
	t *testing.T,
	status int,
	body string,
) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestInterfaceMock(t *testing.T) {
	var client Claudebox = &mockClaudebox{
		healthFunc: func(
			_ context.Context,
		) (*HealthResponse, error) {
			return &HealthResponse{Status: "ok"}, nil
		},
		runFunc: func(
			_ context.Context,
			req *RunRequest,
		) (*RunResponse, error) {
			return &RunResponse{
				Result:  "mocked: " + req.Prompt,
				IsError: false,
			}, nil
		},
	}

	h, err := client.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", h.Status)

	resp, err := client.Run(
		context.Background(),
		&RunRequest{Prompt: "test"},
	)
	require.NoError(t, err)
	assert.Equal(t, "mocked: test", resp.Result)
	assert.False(t, resp.IsError)
}

func TestNewDefaults(t *testing.T) {
	c := New("http://localhost:8080")

	assert.Equal(t, "http://localhost:8080", c.baseURL)
	assert.Empty(t, c.token)
	require.NotNil(t, c.httpClient)
	assert.Equal(t, 10*time.Minute, c.httpClient.Timeout)
}

func TestNewTrimsTrailingSlash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:8080/", "http://localhost:8080"},
		{"http://localhost:8080///", "http://localhost:8080"},
		{"http://localhost:8080", "http://localhost:8080"},
	}

	for _, tt := range tests {
		c := New(tt.input)
		assert.Equal(t, tt.want, c.baseURL)
	}
}

func TestDoInvalidURL(t *testing.T) {
	c := New("://bad\x7furl")

	_, err := c.Health(context.Background())
	require.Error(t, err)
}

// mockClaudebox is a test double for the Claudebox
// interface. Only the funcs you set get called; nil
// funcs panic, surfacing unintended calls.
type mockClaudebox struct {
	healthFunc     func(context.Context) (*HealthResponse, error)
	statusFunc     func(context.Context) (*StatusResponse, error)
	runFunc        func(context.Context, *RunRequest) (*RunResponse, error)
	runAsyncFunc   func(context.Context, *RunRequest) (*AsyncRunResponse, error)
	runResultFunc  func(context.Context, string) (*RunResultResponse, error)
	cancelFunc     func(context.Context, string) (*CancelResponse, error)
	cancelRunFunc  func(context.Context, string) (*CancelResponse, error)
	listFilesFunc  func(context.Context, string) (*ListFilesResponse, error)
	readFileFunc   func(context.Context, string) (*ReadFileResponse, error)
	writeFileFunc  func(context.Context, string, []byte) (*WriteFileResponse, error)
	deleteFileFunc func(context.Context, string) (*DeleteFileResponse, error)
}

func (m *mockClaudebox) Health(
	ctx context.Context,
) (*HealthResponse, error) {
	return m.healthFunc(ctx)
}

func (m *mockClaudebox) Status(
	ctx context.Context,
) (*StatusResponse, error) {
	return m.statusFunc(ctx)
}

func (m *mockClaudebox) Run(
	ctx context.Context,
	req *RunRequest,
) (*RunResponse, error) {
	return m.runFunc(ctx, req)
}

func (m *mockClaudebox) RunAsync(
	ctx context.Context,
	req *RunRequest,
) (*AsyncRunResponse, error) {
	return m.runAsyncFunc(ctx, req)
}

func (m *mockClaudebox) RunResult(
	ctx context.Context,
	runID string,
) (*RunResultResponse, error) {
	return m.runResultFunc(ctx, runID)
}

func (m *mockClaudebox) Cancel(
	ctx context.Context,
	workspace string,
) (*CancelResponse, error) {
	return m.cancelFunc(ctx, workspace)
}

func (m *mockClaudebox) CancelRun(
	ctx context.Context,
	runID string,
) (*CancelResponse, error) {
	return m.cancelRunFunc(ctx, runID)
}

func (m *mockClaudebox) ListFiles(
	ctx context.Context,
	dirPath string,
) (*ListFilesResponse, error) {
	return m.listFilesFunc(ctx, dirPath)
}

func (m *mockClaudebox) ReadFile(
	ctx context.Context,
	filePath string,
) (*ReadFileResponse, error) {
	return m.readFileFunc(ctx, filePath)
}

func (m *mockClaudebox) WriteFile(
	ctx context.Context,
	filePath string,
	content []byte,
) (*WriteFileResponse, error) {
	return m.writeFileFunc(ctx, filePath, content)
}

func (m *mockClaudebox) DeleteFile(
	ctx context.Context,
	filePath string,
) (*DeleteFileResponse, error) {
	return m.deleteFileFunc(ctx, filePath)
}
