package webserver_test

// Wave 0 RED tests for Phase 146 FIX-03 — join-code embed in sessions/meta.
//
// These tests assert the contract for SetJoinCodeIssuer + the ro_join_code /
// rw_join_code fields returned by GET /api/sessions/meta. They are intentionally
// RED until Wave 1 adds:
//   - sessionMetaItem.ROJoinCode / RWJoinCode fields
//   - WebServer.joinCodeIssuer field
//   - WebServer.SetJoinCodeIssuer setter
//   - handleSessionsMeta calling ws.joinCodeIssuer when non-nil
//
// Field-shape contract fixed here (before any implementation diverges):
//   ro_join_code  string `json:"ro_join_code,omitempty"`
//   rw_join_code  string `json:"rw_join_code,omitempty"`

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestSessionsMeta_EmbedJoinCodes asserts that when a joinCodeIssuer is wired,
// GET /api/sessions/meta embeds ro_join_code and rw_join_code in each item.
//
// RED: ws.SetJoinCodeIssuer does not exist until Wave 1.
func TestSessionsMeta_EmbedJoinCodes(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == "sess-a" {
			return "Session Alpha", "claude", "running", "host-a.ts.net"
		}
		return "", "", "", ""
	})
	ws.EnableSession("sess-a")

	// Wire a deterministic fake issuer. Production waves will use api.mintSessionJoinCodes.
	// This setter does NOT yet exist — the test is RED until Wave 1 adds it.
	ws.SetJoinCodeIssuer(func(id string) (string, string, error) {
		return "ro-code-" + id, "rw-code-" + id, nil
	})

	resp, err := client.Get(ws.BaseURL() + "/api/sessions/meta")
	if err != nil {
		t.Fatalf("GET /api/sessions/meta: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d: %v", len(items), items)
	}

	item := items[0]

	// Assert ro_join_code and rw_join_code are present and correct.
	roCode, ok := item["ro_join_code"]
	if !ok {
		t.Error("item missing ro_join_code field")
	} else if roCode != "ro-code-sess-a" {
		t.Errorf("ro_join_code: got %q, want %q", roCode, "ro-code-sess-a")
	}

	rwCode, ok := item["rw_join_code"]
	if !ok {
		t.Error("item missing rw_join_code field")
	} else if rwCode != "rw-code-sess-a" {
		t.Errorf("rw_join_code: got %q, want %q", rwCode, "rw-code-sess-a")
	}
}

// TestSessionsMeta_NilIssuer verifies that when no joinCodeIssuer is wired,
// GET /api/sessions/meta returns items without ro_join_code / rw_join_code
// (the handler must not panic in degraded mode).
//
// RED: the fields do not yet exist in sessionMetaItem; once Wave 1 adds them
// with omitempty, this test becomes GREEN automatically.
func TestSessionsMeta_NilIssuer(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == "sess-b" {
			return "Session Beta", "codex", "idle", "host-b.ts.net"
		}
		return "", "", "", ""
	})
	ws.EnableSession("sess-b")
	// Deliberately do NOT call ws.SetJoinCodeIssuer — degraded mode.

	resp, err := client.Get(ws.BaseURL() + "/api/sessions/meta")
	if err != nil {
		t.Fatalf("GET /api/sessions/meta: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 in nil-issuer mode, got %d", resp.StatusCode)
	}

	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]

	// ro_join_code / rw_join_code must be absent or empty (omitempty with empty string).
	if v, ok := item["ro_join_code"]; ok && v != "" {
		t.Errorf("nil-issuer: ro_join_code should be absent/empty, got %q", v)
	}
	if v, ok := item["rw_join_code"]; ok && v != "" {
		t.Errorf("nil-issuer: rw_join_code should be absent/empty, got %q", v)
	}

	// Required fields must still be present.
	if item["id"] == "" {
		t.Error("expected non-empty id in nil-issuer response")
	}
	if item["url"] == "" {
		t.Error("expected non-empty url in nil-issuer response")
	}
}
