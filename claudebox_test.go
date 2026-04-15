package claudebox

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
