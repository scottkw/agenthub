package webserver

// Tests for the capability-gated web-share chat routes:
//   GET /api/chat/{id}/history?cap=TOKEN
//   GET /api/chat/{id}/export?cap=TOKEN
//
// These tests live in package webserver (internal) so they can use the
// existing test helpers (testServer, issueCapFor, capTestKey).
//
// Every test exercises at least one of the security invariants from
// T-151-04:
//   - no ?cap=    → 401 Unauthorized
//   - bad ?cap=   → 401 Unauthorized
//   - cap for wrong session → 403 Forbidden
//   - valid cap for matching session → 200 with provider data

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// wireChatProvider installs both the history and export providers (IN-04 split
// the single callback in two) on ws so the cap-gated chat routes respond with
// the supplied history bytes and markdown string for sessionID. Any other
// session ID returns found == false. Use to wire a WebServer for chat-route
// tests without importing the daemon or relay packages.
func wireChatProvider(ws *WebServer, sessionID string, historyJSON []byte, markdown string) {
	ws.SetChatHistoryProvider(func(id string) ([]byte, bool, error) {
		if id != sessionID {
			return nil, false, nil
		}
		return historyJSON, true, nil
	})
	ws.SetChatExportProvider(func(id string) (string, bool, error) {
		if id != sessionID {
			return "", false, nil
		}
		return markdown, true, nil
	})
}

// knownHistoryJSON is a small pre-marshaled JSON array used by tests that only
// need to confirm the provider bytes arrive unchanged at the caller.
var knownHistoryJSON = func() []byte {
	type msg struct {
		Content string `json:"content"`
	}
	b, _ := json.Marshal([]msg{{Content: "hello"}, {Content: "world"}})
	return b
}()

const knownMarkdown = "# Chat Thread: sess-web\n\n## alice (2026-01-01T00:00:00Z)\n\nhello world\n\n---\n\n"

// TestChatWeb_ValidCap_History verifies that a valid cap for the correct
// session returns 200 and the provider's JSON bytes.
func TestChatWeb_ValidCap_History(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-web")
	wireChatProvider(ws, "sess-web", knownHistoryJSON, knownMarkdown)

	token := issueCapFor(t, ws, "sess-web", "read,write")

	resp, err := client.Get(ws.BaseURL() + "/api/chat/sess-web/history?cap=" + token)
	if err != nil {
		t.Fatalf("GET history: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d; body=%s", resp.StatusCode, body)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q; want application/json prefix", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != string(knownHistoryJSON) {
		t.Errorf("body = %q; want %q", body, knownHistoryJSON)
	}
}

// TestChatWeb_NoCapReturns401 verifies that a request without a ?cap=
// parameter is rejected with 401 (PERSIST-02 / T-151-04).
func TestChatWeb_NoCapReturns401(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-web")
	wireChatProvider(ws, "sess-web", knownHistoryJSON, knownMarkdown)

	for _, path := range []string{
		"/api/chat/sess-web/history",
		"/api/chat/sess-web/export",
	} {
		resp, err := client.Get(ws.BaseURL() + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET %s (no cap): expected 401, got %d", path, resp.StatusCode)
		}
	}
}

// TestChatWeb_InvalidCapReturns401 verifies that a garbage ?cap= token is
// rejected with 401 (T-151-04 / SEC-03 analogous behavior).
func TestChatWeb_InvalidCapReturns401(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-web")
	wireChatProvider(ws, "sess-web", knownHistoryJSON, knownMarkdown)

	for _, path := range []string{
		"/api/chat/sess-web/history?cap=not-a-valid-token",
		"/api/chat/sess-web/export?cap=garbage",
	} {
		resp, err := client.Get(ws.BaseURL() + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET %s (invalid cap): expected 401, got %d", path, resp.StatusCode)
		}
	}
}

// TestChatWeb_WrongSessionCapReturns403 verifies that a valid cap minted for
// a different session is rejected with 403 (T-151-04 — session isolation).
func TestChatWeb_WrongSessionCapReturns403(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-A")
	ws.EnableSession("sess-B")
	wireChatProvider(ws, "sess-A", knownHistoryJSON, knownMarkdown)

	// Cap issued for sess-A. Request targets sess-B → 403.
	tokenA := issueCapFor(t, ws, "sess-A", "read,write")

	for _, path := range []string{
		"/api/chat/sess-B/history?cap=" + tokenA,
		"/api/chat/sess-B/export?cap=" + tokenA,
	} {
		resp, err := client.Get(ws.BaseURL() + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("GET %s (wrong-session cap): expected 403, got %d", path, resp.StatusCode)
		}
	}
}

// TestChatWeb_ValidCap_Export verifies that a valid cap for the correct session
// returns 200 with Content-Type text/markdown and a Content-Disposition
// attachment filename.
func TestChatWeb_ValidCap_Export(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-web")
	wireChatProvider(ws, "sess-web", knownHistoryJSON, knownMarkdown)

	token := issueCapFor(t, ws, "sess-web", "read,write")

	resp, err := client.Get(ws.BaseURL() + "/api/chat/sess-web/export?cap=" + token)
	if err != nil {
		t.Fatalf("GET export: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d; body=%s", resp.StatusCode, body)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/markdown") {
		t.Errorf("Content-Type = %q; want to contain text/markdown", ct)
	}
	cd := resp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition = %q; want to contain attachment", cd)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != knownMarkdown {
		t.Errorf("body = %q; want %q", body, knownMarkdown)
	}
}

// TestChatWeb_ProviderSessionNotFound returns 404 when the provider reports
// ok == false for the requested session (session has no chat store).
func TestChatWeb_ProviderSessionNotFound(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	// Enable and issue a valid cap for "sess-found" but provider returns
	// ok==false for all IDs (simulates session with no chat store).
	ws.EnableSession("sess-found")
	ws.SetChatHistoryProvider(func(string) ([]byte, bool, error) { return nil, false, nil })
	ws.SetChatExportProvider(func(string) (string, bool, error) { return "", false, nil })

	token := issueCapFor(t, ws, "sess-found", "read,write")

	for _, path := range []string{
		"/api/chat/sess-found/history?cap=" + token,
		"/api/chat/sess-found/export?cap=" + token,
	} {
		resp, err := client.Get(ws.BaseURL() + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s (no store): expected 404, got %d", path, resp.StatusCode)
		}
	}
}

// TestChatWeb_ProviderInternalError returns 500 (not 404) when the provider
// reports an internal error on a session that exists (WR-03 — an internal
// serialization/export failure must not be masked as "session not found").
func TestChatWeb_ProviderInternalError(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-err")
	// found == true but err != nil: an existing session whose history/export
	// failed internally. Both routes must surface 500, so wire both providers.
	ws.SetChatHistoryProvider(func(string) ([]byte, bool, error) {
		return nil, true, errors.New("marshal blew up")
	})
	ws.SetChatExportProvider(func(string) (string, bool, error) {
		return "", true, errors.New("export blew up")
	})

	token := issueCapFor(t, ws, "sess-err", "read,write")

	for _, path := range []string{
		"/api/chat/sess-err/history?cap=" + token,
		"/api/chat/sess-err/export?cap=" + token,
	} {
		resp, err := client.Get(ws.BaseURL() + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("GET %s (provider error): expected 500, got %d", path, resp.StatusCode)
		}
	}
}

// TestChatWeb_RequireCapabilityIsWiredForBothRoutes confirms that
// requireCapability is registered for both routes by checking that the
// routes are listed in the mux (via the 401 without-cap response).
// This is a belt-and-suspenders structural test: the critical enforcement
// is in the negative tests above; this test ensures neither route is
// accidentally left ungated after a refactor.
func TestChatWeb_RequireCapabilityIsWiredForBothRoutes(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-gate")
	wireChatProvider(ws, "sess-gate", knownHistoryJSON, knownMarkdown)

	// Without a cap, both routes must return 401 (not 404).
	for _, path := range []string{
		"/api/chat/sess-gate/history",
		"/api/chat/sess-gate/export",
	} {
		resp, err := client.Get(ws.BaseURL() + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Errorf("GET %s (no cap) returned 404 — route is not registered (check setupRoutes)", path)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET %s (no cap): expected 401, got %d", path, resp.StatusCode)
		}
	}
}
