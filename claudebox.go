// Package claudebox provides a Go client for the
// claudebox API — a runtime harness for Claude Code
// running in Docker containers.
//
// The Claudebox interface enables mocking for tests.
// Use New to create a concrete *Client.
package claudebox

import "context"

// Claudebox is the interface for all claudebox API
// operations. Implement this for mocking in tests.
type Claudebox interface {
	// Health checks if the server is up.
	Health(ctx context.Context) (*HealthResponse, error)

	// Status returns currently busy workspaces.
	Status(ctx context.Context) (*StatusResponse, error)

	// Run executes a prompt via POST /run.
	Run(
		ctx context.Context,
		req *RunRequest,
	) (*RunResponse, error)

	// Cancel kills a running process in the given
	// workspace (empty = default).
	Cancel(
		ctx context.Context,
		workspace string,
	) (*CancelResponse, error)

	// ListFiles lists files at the given path
	// (empty = workspace root).
	ListFiles(
		ctx context.Context,
		dirPath string,
	) (*ListFilesResponse, error)

	// ReadFile downloads a file. The caller must close
	// the returned ReadFileResponse.Body when done.
	ReadFile(
		ctx context.Context,
		filePath string,
	) (*ReadFileResponse, error)

	// WriteFile uploads content to the given path.
	WriteFile(
		ctx context.Context,
		filePath string,
		content []byte,
	) (*WriteFileResponse, error)

	// DeleteFile deletes a file at the given path.
	DeleteFile(
		ctx context.Context,
		filePath string,
	) (*DeleteFileResponse, error)
}

// compile-time check
var _ Claudebox = (*Client)(nil)
