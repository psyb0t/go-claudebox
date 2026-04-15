// Package claudebox provides a Go client for the
// claudebox direct API.
package claudebox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	commonhttp "github.com/psyb0t/common-go/http"
	"github.com/psyb0t/ctxerrors"
)

const defaultTimeout = 10 * time.Minute

// Client talks to a claudebox API server.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithToken sets the Bearer token for authenticated
// requests.
func WithToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// WithHTTPClient overrides the default http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a claudebox client. baseURL is the server
// root, e.g. "http://localhost:8080".
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

func (c *Client) do(
	ctx context.Context,
	method, endpoint string,
	body any,
) (*http.Response, error) {
	var r io.Reader

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, ctxerrors.Wrap(
				err, "marshal body",
			)
		}

		r = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(
		ctx, method,
		c.baseURL+endpoint, r,
	)
	if err != nil {
		return nil, ctxerrors.Wrap(
			err, "create request",
		)
	}

	if body != nil {
		req.Header.Set(
			commonhttp.HeaderContentType,
			commonhttp.ContentTypeJSON,
		)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ctxerrors.Wrap(
			err, "execute request",
		)
	}

	return resp, nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.token == "" {
		return
	}

	req.Header.Set(
		commonhttp.HeaderAuthorization,
		commonhttp.AuthSchemeBearer+c.token,
	)
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK &&
		resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	b, _ := io.ReadAll(resp.Body)

	return &APIError{
		StatusCode: resp.StatusCode,
		Body:       string(b),
	}
}

// APIError is returned when the server responds with a
// non-2xx status code.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf(
		"claudebox: HTTP %d: %s",
		e.StatusCode, e.Body,
	)
}
