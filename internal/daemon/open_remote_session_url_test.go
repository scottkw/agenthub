package daemon

// Phase 146-05 — Go test for the daemon read path: GET /api/remote-files/caps/{sessionID}/open-url.
//
// Behavior-level tests (not source-grep) per WR-02 plan requirement.
// Exercises handleRemoteSessionOpenURL via the HTTP API directly:
//   - Held cap: deposit then call → expect JSON {"url": "baseURL/sessions/sessionID?cap=TOKEN"}
//   - No cap: absent sessionID → expect 404 + JSON {"error": "no cap registered for session"}

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// TestRemoteSessionOpenURL_HeldCap verifies that when a cap has been deposited
// for a session, the open-url handler returns HTTP 200 with a JSON body
// {"url": "baseURL/sessions/sessionID?cap=TOKEN"}.
func TestRemoteSessionOpenURL_HeldCap(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	api, srv := proxyTestAPI(t, upstream)

	// Deposit a cap with a fixed https baseURL and token.
	// proxyRemoteFiles tests use upstream.URL (http://...) but RemoteCapStore.Put
	// validates https://; use a hardcoded peer URL to match the handler's composition.
	if err := api.remoteCaps.Put("sess-1", "https://peer:8443", "TOK"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	resp, err := http.Get(srv.URL + "/api/remote-files/caps/sess-1/open-url")
	if err != nil {
		t.Fatalf("GET open-url: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200; got %d; body=%s", resp.StatusCode, body)
	}

	var got map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	wantURL := "https://peer:8443/sessions/sess-1?cap=TOK"
	if got["url"] != wantURL {
		t.Fatalf("url mismatch:\n  got  %q\n  want %q", got["url"], wantURL)
	}
}

// TestRemoteSessionOpenURL_NoCap verifies that when no cap is deposited for a
// session, the open-url handler returns HTTP 404 with a JSON body containing
// "no cap registered for session" (mirrors proxyRemoteFiles 404 shape).
func TestRemoteSessionOpenURL_NoCap(t *testing.T) {
	upstream, closeUp := mockRemoteWebserver(t)
	defer closeUp()
	_, srv := proxyTestAPI(t, upstream)

	resp, err := http.Get(srv.URL + "/api/remote-files/caps/sess-x/open-url")
	if err != nil {
		t.Fatalf("GET open-url: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404; got %d; body=%s", resp.StatusCode, body)
	}

	var got map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if got["error"] != "no cap registered for session" {
		t.Fatalf("error mismatch: got %q; want %q", got["error"], "no cap registered for session")
	}
}
