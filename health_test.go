package claudebox

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestHealth(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet ||
				r.URL.Path != "/health" {
				t.Errorf(
					"unexpected request: %s %s",
					r.Method, r.URL.Path,
				)
			}

			_ = json.NewEncoder(w).Encode(
				HealthResponse{Status: "ok"},
			)
		},
	)

	resp, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if resp.Status != "ok" {
		t.Errorf("got status %q, want ok", resp.Status)
	}
}

func TestStatus(t *testing.T) {
	c, _ := testServer(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/status" {
				t.Errorf(
					"unexpected path: %s",
					r.URL.Path,
				)
			}

			_ = json.NewEncoder(w).Encode(
				StatusResponse{
					BusyWorkspaces: []string{
						"/workspaces/foo",
					},
				},
			)
		},
	)

	resp, err := c.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.BusyWorkspaces) != 1 ||
		resp.BusyWorkspaces[0] != "/workspaces/foo" {
		t.Errorf(
			"unexpected busy workspaces: %v",
			resp.BusyWorkspaces,
		)
	}
}
