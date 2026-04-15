package claudebox

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/psyb0t/ctxerrors"
)

// FileEntry is a single item in a directory listing.
type FileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size,omitempty"`
}

// ListFilesResponse is the response from GET /files or
// GET /files/{path} on a directory.
type ListFilesResponse struct {
	Path    string      `json:"path"`
	Entries []FileEntry `json:"entries"`
}

// ListFiles lists files at the given path
// (empty = workspace root).
func (c *Client) ListFiles(
	ctx context.Context,
	dirPath string,
) (*ListFilesResponse, error) {
	endpoint := "/files"

	if dirPath != "" {
		endpoint = "/files/" +
			strings.TrimLeft(dirPath, "/")
	}

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

	var v ListFilesResponse
	if err := json.NewDecoder(resp.Body).Decode(
		&v,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	return &v, nil
}

// ReadFile downloads a file and returns its contents
// as bytes.
func (c *Client) ReadFile(
	ctx context.Context,
	filePath string,
) ([]byte, error) {
	endpoint := "/files/" +
		strings.TrimLeft(filePath, "/")

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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ctxerrors.Wrap(
			err, "read file body",
		)
	}

	return data, nil
}

// WriteFileResponse is the response from
// PUT /files/{path}.
type WriteFileResponse struct {
	Status string `json:"status"`
	Path   string `json:"path"`
	Size   int    `json:"size"`
}

// WriteFile uploads content to the given path.
func (c *Client) WriteFile(
	ctx context.Context,
	filePath string,
	content []byte,
) (*WriteFileResponse, error) {
	endpoint := c.baseURL + "/files/" +
		strings.TrimLeft(filePath, "/")

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut,
		endpoint,
		bytes.NewReader(content),
	)
	if err != nil {
		return nil, ctxerrors.Wrap(
			err, "create request",
		)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ctxerrors.Wrap(
			err, "execute request",
		)
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var v WriteFileResponse
	if err := json.NewDecoder(resp.Body).Decode(
		&v,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	return &v, nil
}

// DeleteFileResponse is the response from
// DELETE /files/{path}.
type DeleteFileResponse struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// DeleteFile deletes a file at the given path.
func (c *Client) DeleteFile(
	ctx context.Context,
	filePath string,
) (*DeleteFileResponse, error) {
	endpoint := "/files/" +
		strings.TrimLeft(filePath, "/")

	resp, err := c.do(
		ctx, http.MethodDelete, endpoint, nil,
	)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	var v DeleteFileResponse
	if err := json.NewDecoder(resp.Body).Decode(
		&v,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode response",
		)
	}

	return &v, nil
}

// FilePath joins path segments for use with file
// operations.
func FilePath(parts ...string) string {
	return path.Join(parts...)
}
