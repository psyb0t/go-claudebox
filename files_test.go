package claudebox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListFiles(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf(
					"unexpected method: %s",
					r.Method,
				)
			}

			_ = json.NewEncoder(w).Encode(
				ListFilesResponse{
					Path: "/",
					Entries: []FileEntry{
						{
							Name: "foo.txt",
							Type: "file",
							Size: 42,
						},
					},
				},
			)
		},
	)

	resp, err := c.ListFiles(
		context.Background(), "",
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Entries) != 1 ||
		resp.Entries[0].Name != "foo.txt" {
		t.Errorf(
			"unexpected entries: %+v",
			resp.Entries,
		)
	}
}

func TestListFilesSubdir(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(
				r.URL.Path, "/files/subdir",
			) {
				t.Errorf(
					"unexpected path: %s",
					r.URL.Path,
				)
			}

			_ = json.NewEncoder(w).Encode(
				ListFilesResponse{
					Path:    "subdir",
					Entries: []FileEntry{},
				},
			)
		},
	)

	_, err := c.ListFiles(
		context.Background(), "subdir",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadFile(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(
				r.URL.Path, "/files/test.txt",
			) {
				t.Errorf(
					"unexpected path: %s",
					r.URL.Path,
				)
			}

			_, _ = w.Write([]byte("file content here"))
		},
	)

	data, err := c.ReadFile(
		context.Background(), "test.txt",
	)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "file content here" {
		t.Errorf("got %q", string(data))
	}
}

func TestWriteFile(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf(
					"unexpected method: %s",
					r.Method,
				)
			}

			if !strings.HasSuffix(
				r.URL.Path, "/files/out.txt",
			) {
				t.Errorf(
					"unexpected path: %s",
					r.URL.Path,
				)
			}

			body, _ := io.ReadAll(r.Body)
			if string(body) != "hello world" {
				t.Errorf(
					"unexpected body: %q",
					string(body),
				)
			}

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
	if err != nil {
		t.Fatal(err)
	}

	if resp.Status != "ok" {
		t.Errorf("got status %q", resp.Status)
	}

	if resp.Size != 11 {
		t.Errorf("got size %d", resp.Size)
	}
}

func TestDeleteFile(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
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
	if err != nil {
		t.Fatal(err)
	}

	if resp.Status != "ok" {
		t.Errorf("got status %q", resp.Status)
	}
}

func TestFilePath(t *testing.T) {
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"a", "b", "c.txt"}, "a/b/c.txt"},
		{[]string{"a/b", "c.txt"}, "a/b/c.txt"},
		{[]string{"file.txt"}, "file.txt"},
	}

	for _, tt := range tests {
		got := FilePath(tt.parts...)
		if got != tt.expected {
			t.Errorf(
				"FilePath(%v) = %q, want %q",
				tt.parts, got, tt.expected,
			)
		}
	}
}
