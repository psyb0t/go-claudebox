package claudebox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFiles(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/files", r.URL.Path)

			_ = json.NewEncoder(w).Encode(
				ListFilesResponse{
					Path: "/",
					Entries: []FileEntry{
						{Name: "foo.txt", Type: "file", Size: 42},
						{Name: "subdir", Type: "dir"},
					},
				},
			)
		},
	)

	resp, err := c.ListFiles(
		context.Background(), "",
	)
	require.NoError(t, err)
	assert.Equal(t, "/", resp.Path)
	require.Len(t, resp.Entries, 2)
	assert.Equal(t, "foo.txt", resp.Entries[0].Name)
	assert.Equal(t, "file", resp.Entries[0].Type)
	assert.Equal(t, int64(42), resp.Entries[0].Size)
	assert.Equal(t, "dir", resp.Entries[1].Type)
}

func TestListFilesPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath string
	}{
		{"subdir", "subdir", "/files/subdir"},
		{"leading slash stripped", "/mydir", "/files/mydir"},
		{"nested", "a/b/c", "/files/a/b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testServer(t,
				func(w http.ResponseWriter, r *http.Request) {
					assert.True(t,
						strings.HasSuffix(
							r.URL.Path, tt.wantPath,
						),
						"path %q should end with %q",
						r.URL.Path, tt.wantPath,
					)

					_ = json.NewEncoder(w).Encode(
						ListFilesResponse{Path: tt.input},
					)
				},
			)

			_, err := c.ListFiles(
				context.Background(), tt.input,
			)
			require.NoError(t, err)
		})
	}
}

func TestListFilesInvalidJSON(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `not json`),
	)

	_, err := c.ListFiles(context.Background(), "")
	require.Error(t, err)
}

func TestListFiles404(t *testing.T) {
	c, _ := testServer(t,
		errorHandler(t,
			http.StatusNotFound,
			`{"detail":"not found: nope"}`,
		),
	)

	_, err := c.ListFiles(
		context.Background(), "nope",
	)
	require.Error(t, err)

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t,
		http.StatusNotFound, apiErr.StatusCode,
	)
}

func TestReadFile(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.True(t,
				strings.HasSuffix(
					r.URL.Path, "/files/test.txt",
				),
			)

			w.Header().Set(
				"Content-Type", "text/plain",
			)
			_, _ = w.Write([]byte("file content here"))
		},
	)

	resp, err := c.ReadFile(
		context.Background(), "test.txt",
	)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "text/plain", resp.ContentType)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "file content here", string(data))
}

func TestReadFileContentLength(t *testing.T) {
	content := "hello world"

	c, _ := testServer(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(
				"Content-Type",
				"application/octet-stream",
			)
			_, _ = w.Write([]byte(content))
		},
	)

	resp, err := c.ReadFile(
		context.Background(), "file.bin",
	)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t,
		"application/octet-stream",
		resp.ContentType,
	)
	assert.Equal(t,
		int64(len(content)), resp.ContentLength,
	)
}

func TestReadFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath string
	}{
		{"simple", "test.txt", "/files/test.txt"},
		{"nested", "src/main.go", "/files/src/main.go"},
		{"deep", "a/b/c/d.txt", "/files/a/b/c/d.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testServer(t,
				func(w http.ResponseWriter, r *http.Request) {
					assert.True(t,
						strings.HasSuffix(
							r.URL.Path, tt.wantPath,
						),
						"path %q should end with %q",
						r.URL.Path, tt.wantPath,
					)

					_, _ = w.Write([]byte("ok"))
				},
			)

			resp, err := c.ReadFile(
				context.Background(), tt.input,
			)
			require.NoError(t, err)
			_ = resp.Body.Close()
		})
	}
}

func TestReadFile404(t *testing.T) {
	c, _ := testServer(t,
		errorHandler(t,
			http.StatusNotFound,
			`{"detail":"not found: missing.txt"}`,
		),
	)

	_, err := c.ReadFile(
		context.Background(), "missing.txt",
	)
	require.Error(t, err)

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t,
		http.StatusNotFound, apiErr.StatusCode,
	)
}

func TestReadFileBinary(t *testing.T) {
	binary := []byte{0x00, 0x01, 0xFF, 0xFE}

	c, _ := testServer(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(
				"Content-Type",
				"application/octet-stream",
			)
			_, _ = w.Write(binary)
		},
	)

	resp, err := c.ReadFile(
		context.Background(), "data.bin",
	)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, binary, data)
}

func TestWriteFile(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			assert.True(t,
				strings.HasSuffix(
					r.URL.Path, "/files/out.txt",
				),
			)

			// WriteFile sends raw bytes, not JSON
			assert.NotEqual(t,
				"application/json",
				r.Header.Get("Content-Type"),
			)

			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, "hello world", string(body))

			_ = json.NewEncoder(w).Encode(
				WriteFileResponse{
					Status: "ok",
					Path:   "/workspaces/out.txt",
					Size:   11,
				},
			)
		},
	)

	resp, err := c.WriteFile(
		context.Background(),
		"out.txt",
		[]byte("hello world"),
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "/workspaces/out.txt", resp.Path)
	assert.Equal(t, 11, resp.Size)
}

func TestWriteFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath string
	}{
		{"simple", "out.txt", "/files/out.txt"},
		{"nested", "a/b/c.txt", "/files/a/b/c.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testServer(t,
				func(w http.ResponseWriter, r *http.Request) {
					assert.True(t,
						strings.HasSuffix(
							r.URL.Path, tt.wantPath,
						),
					)

					_ = json.NewEncoder(w).Encode(
						WriteFileResponse{
							Status: "ok",
							Path:   tt.input,
							Size:   4,
						},
					)
				},
			)

			_, err := c.WriteFile(
				context.Background(),
				tt.input,
				[]byte("data"),
			)
			require.NoError(t, err)
		})
	}
}

func TestWriteFileInvalidJSON(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `not json`),
	)

	_, err := c.WriteFile(
		context.Background(), "f.txt", []byte("x"),
	)
	require.Error(t, err)
}

func TestDeleteFile(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.True(t,
				strings.HasSuffix(
					r.URL.Path, "/files/old.txt",
				),
			)

			_ = json.NewEncoder(w).Encode(
				DeleteFileResponse{
					Status: "ok",
					Path:   "/workspaces/old.txt",
				},
			)
		},
	)

	resp, err := c.DeleteFile(
		context.Background(), "old.txt",
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "/workspaces/old.txt", resp.Path)
}

func TestDeleteFileErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{
			"not found",
			http.StatusNotFound,
			`{"detail":"not found: gone.txt"}`,
		},
		{
			"directory",
			http.StatusBadRequest,
			`{"detail":"cannot delete directories"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testServer(t,
				errorHandler(t, tt.status, tt.body),
			)

			_, err := c.DeleteFile(
				context.Background(), "target",
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

func TestDeleteFileInvalidJSON(t *testing.T) {
	c, _ := testServer(t,
		jsonHandler(t, `not json`),
	)

	_, err := c.DeleteFile(
		context.Background(), "f.txt",
	)
	require.Error(t, err)
}

func TestFilePath(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{[]string{"a", "b", "c.txt"}, "a/b/c.txt"},
		{[]string{"a/b", "c.txt"}, "a/b/c.txt"},
		{[]string{"file.txt"}, "file.txt"},
	}

	for _, tt := range tests {
		assert.Equal(t,
			tt.want, FilePath(tt.parts...),
		)
	}
}
