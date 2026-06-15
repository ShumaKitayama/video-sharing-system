package apitest

import (
	"net/http"
	"testing"
)

// TestHealth verifies the liveness probe always reports ok.
func TestHealth(t *testing.T) {
	requireDB(t)

	resp, body := doJSONURL(t, newClient(t), http.MethodGet, testServer.URL+"/health", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Status string `json:"status"`
	}
	decodeData(t, body, &data)
	if data.Status != "ok" {
		t.Fatalf("GET /health: expected status ok, got %q", data.Status)
	}
}

// TestReady verifies the readiness probe reports the database and storage as
// reachable when both dependencies are healthy.
func TestReady(t *testing.T) {
	requireDB(t)

	resp, body := doJSONURL(t, newClient(t), http.MethodGet, testServer.URL+"/ready", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /ready: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Status   string `json:"status"`
		Database string `json:"database"`
		Storage  string `json:"storage"`
	}
	decodeData(t, body, &data)
	if data.Status != "ready" || data.Database != "ok" || data.Storage != "ok" {
		t.Fatalf("GET /ready: unexpected payload: %+v", data)
	}
}
