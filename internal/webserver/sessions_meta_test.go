package webserver_test

// Wave 0 RED tests for Phase 130 RB-01 / RB-03.
//
// These tests assert the contract for GET /api/sessions/meta — the new
// metadata-only, open (no-cap) endpoint that lets tailnet peers discover
// which sessions are shareable without holding a session-scoped capability.
//
// The handler (handleSessionsMeta) does NOT exist yet; these tests are
// intentionally RED until Plan 130-03 creates the handler and registers the
// route in setupRoutes(). They encode:
//   - RB-01: returns web-enabled sessions as metadata (id, name, cli_type, status, url)
//   - RB-03: response contains NO cap, grant, content, or signing-key fields

import (
	"encoding/json"
	"net/http"
	"sort"
	"testing"
)

// TestSessionsMeta_ReturnsWebEnabledSessions verifies that GET /api/sessions/meta
// returns 200 + a JSON array with one item per web-enabled session, each with
// id, name, cli_type, status, and url fields populated by the sessionResolver.
func TestSessionsMeta_ReturnsWebEnabledSessions(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		switch id {
		case "sess-a":
			return "Session Alpha", "claude", "running", "host-a.ts.net"
		case "sess-b":
			return "Session Beta", "codex", "idle", "host-b.ts.net"
		}
		return "", "", "", ""
	})
	ws.EnableSession("sess-a")
	ws.EnableSession("sess-b")

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
	if len(items) != 2 {
		t.Fatalf("expected 2 session items, got %d: %v", len(items), items)
	}

	// Sort by id for deterministic assertion.
	sort.Slice(items, func(i, j int) bool {
		return items[i]["id"].(string) < items[j]["id"].(string)
	})

	a := items[0]
	if a["id"] != "sess-a" {
		t.Errorf("expected id=sess-a, got %q", a["id"])
	}
	if a["name"] != "Session Alpha" {
		t.Errorf("expected name='Session Alpha', got %q", a["name"])
	}
	if a["cli_type"] != "claude" {
		t.Errorf("expected cli_type=claude, got %q", a["cli_type"])
	}
	if a["status"] != "running" {
		t.Errorf("expected status=running, got %q", a["status"])
	}
	if a["url"] == "" {
		t.Error("expected url to be non-empty")
	}

	b := items[1]
	if b["id"] != "sess-b" {
		t.Errorf("expected id=sess-b, got %q", b["id"])
	}
	if b["name"] != "Session Beta" {
		t.Errorf("expected name='Session Beta', got %q", b["name"])
	}
	if b["cli_type"] != "codex" {
		t.Errorf("expected cli_type=codex, got %q", b["cli_type"])
	}
	if b["status"] != "idle" {
		t.Errorf("expected status=idle, got %q", b["status"])
	}
	if b["url"] == "" {
		t.Error("expected url to be non-empty")
	}
}

// TestSessionsMeta_NoCap verifies that GET /api/sessions/meta returns 200
// WITHOUT any cap header — the endpoint is open (tailnet-trusted via network
// binding, not application-layer cap). This contrasts with GET /api/sessions
// which requires a capability token.
func TestSessionsMeta_NoCap(t *testing.T) {
	ws, client := testServer(t)
	ws.EnableSession("sess1")

	// Request without any cap header or query param.
	resp, err := client.Get(ws.BaseURL() + "/api/sessions/meta")
	if err != nil {
		t.Fatalf("GET /api/sessions/meta (no cap): %v", err)
	}
	defer resp.Body.Close()

	// Must be 200 (open), not 401 (which /api/sessions returns without a cap).
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatalf("/api/sessions/meta returned 401 — the endpoint must be open (no cap required); got %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for open endpoint, got %d", resp.StatusCode)
	}
}

// TestSessionsMeta_EmptyWhenNoneEnabled verifies that GET /api/sessions/meta
// returns 200 + an empty JSON array `[]` (not null, not 404) when no sessions
// are web-enabled. The RB-04 honest-empty-state contract.
func TestSessionsMeta_EmptyWhenNoneEnabled(t *testing.T) {
	ws, client := testServer(t)
	// No sessions enabled.

	resp, err := client.Get(ws.BaseURL() + "/api/sessions/meta")
	if err != nil {
		t.Fatalf("GET /api/sessions/meta (empty): %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// Must be an empty slice (not nil — the JSON must be `[]`, not `null`).
	if items == nil {
		t.Fatal("expected non-nil empty slice (JSON `[]`), got null")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d: %v", len(items), items)
	}
}

// TestSessionsMeta_NoCapInResponse encodes the RB-03 no-enumeration security
// guarantee: the /api/sessions/meta response must contain ONLY the allowed
// metadata fields {id, name, cli_type, status, url}. It MUST NOT contain any
// cap, token, grant, grants, content, or signing-key field.
//
// The test decodes each item into map[string]any and asserts the key set
// equals exactly {id, name, cli_type, status, url}.
func TestSessionsMeta_NoCapInResponse(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey) // signing key is present (simulate production state)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == "secure-sess" {
			return "Secure Session", "claude", "running", "host.ts.net"
		}
		return "", "", "", ""
	})
	ws.EnableSession("secure-sess")

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
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]

	// RB-03: assert the EXACT allowed key set — any extra key is a violation.
	// Phase 146: ro_join_code and rw_join_code are intentionally allowed (they
	// are short-lived join codes, not raw cap tokens). They are NOT on the
	// sensitiveKeys blacklist below.
	allowed := map[string]bool{
		"id":           true,
		"name":         true,
		"cli_type":     true,
		"status":       true,
		"url":          true,
		"ro_join_code": true, // Phase 146: join-code for RO viewer access
		"rw_join_code": true, // Phase 146: join-code for RW owner access
	}
	for k := range item {
		if !allowed[k] {
			t.Errorf("RB-03 violation: /api/sessions/meta response contains forbidden key %q", k)
		}
	}

	// Assert all required keys are present.
	for k := range allowed {
		if _, ok := item[k]; !ok {
			t.Errorf("expected key %q in response, got: %v", k, item)
		}
	}

	// Explicit check for the most sensitive field names (defense-in-depth).
	sensitiveKeys := []string{"cap", "token", "grant", "grants", "content", "key", "signing_key", "hmac"}
	for _, k := range sensitiveKeys {
		if _, ok := item[k]; ok {
			t.Errorf("RB-03 violation: response contains sensitive key %q — must NEVER be returned by /api/sessions/meta", k)
		}
	}
}
