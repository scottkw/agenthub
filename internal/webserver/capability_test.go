// Package webserver capability-enforcement tests activated by Plan 03.
//
// These nine tests assert the boundary contract SEC-02..SEC-05 from
// REQUIREMENTS.md. Plan 01 authored them as RED skeletons under build-tag
// phase87_wave2; Plan 03 un-tags the file and fills in real bodies against
// the production WebServer + capability.Verify path.
package webserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/relay"
)

// capTestKey is a deterministic 32-byte signing key used across all tests so
// an individual test can mint a token against the same key it installed on
// the server via ws.SetSigningKey.
var capTestKey = func() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}()

// issueCapFor builds a signed token for sessionID with perms, and registers
// the grant_id on ws so requireCapability's grant-list check passes.
func issueCapFor(t *testing.T, ws *WebServer, sessionID, perms string) string {
	t.Helper()
	claims := capability.Claims{
		SID:     sessionID,
		Perms:   perms,
		IAT:     time.Now().Unix(),
		GrantID: "grant-" + sessionID + "-" + perms,
		V:       1,
	}
	token, err := capability.Sign(claims, capTestKey)
	if err != nil {
		t.Fatalf("capability.Sign: %v", err)
	}
	ws.AddGrant(sessionID, claims.GrantID)
	return token
}

// TestSecurity_UnauthenticatedClientCannotEnumerateSessions is the inverted
// form of the security-review's "can enumerate" scaffold. SEC-02: GET
// /api/sessions without a valid cap must return 401.
func TestSecurity_UnauthenticatedClientCannotEnumerateSessions(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-authz")

	resp, err := client.Get(ws.BaseURL() + "/api/sessions")
	if err != nil {
		t.Fatalf("GET /api/sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without capability, got %d", resp.StatusCode)
	}
}

// TestSecurity_WrongSessionCapRejected covers SEC-03: a capability bound to
// session A must NOT grant access to session B's WebSocket upgrade.
func TestSecurity_WrongSessionCapRejected(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-A")
	ws.SetSigningKey(capTestKey)
	// Also enable sess-B so the wrong-cap path is what triggers the 403, not
	// a missing web-enable state.
	ws.EnableSession("sess-B")

	// Cap bound to sess-A.
	tokenA := issueCapFor(t, ws, "sess-A", "read,write")

	// Attempt GET /sessions/sess-B/ws with sess-A cap → expect 403.
	reqURL := ws.BaseURL() + "/sessions/sess-B/ws?cap=" + tokenA
	resp, err := client.Get(reqURL)
	if err != nil {
		t.Fatalf("GET /sessions/sess-B/ws: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for cross-session cap, got %d", resp.StatusCode)
	}
}

// TestSecurity_ReadOnlyParamCannotGrantWrite covers SEC-04: the legacy
// ?readonly= query param has been removed from the write-gate path (D-23).
// A caller with a read-only cap cannot write regardless of the readonly
// query string value.
func TestSecurity_ReadOnlyParamCannotGrantWrite(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-ro")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-ro", "read")

	// Dial with ?readonly=0 (explicit attempt to override) and the read cap.
	// Phase 88: set Origin header to ws.BaseURL() so the requireAllowedOrigin
	// middleware passes — only the write-gate assertion is the focus here.
	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())
	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-ro/ws?cap="+token+"&readonly=0",
		headers,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Send MsgInput — it must be discarded by the server since perms=="read"
	// (SEC-04 / D-24 source from claims.Perms, not the query string).
	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame([]byte("x\n"))); err != nil {
		t.Fatalf("write input frame: %v", err)
	}

	readPipeMustTimeout(t, inputReader, 300*time.Millisecond, "read cap with readonly=0 must not grant write")
}

// TestSecurity_ReadOnlyCapabilityBlocksMsgInput covers SEC-05: the relay
// (via sub.ReadOnly sourced from claims.Perms) drops MsgInput frames from a
// subscriber whose capability lacks write permission.
func TestSecurity_ReadOnlyCapabilityBlocksMsgInput(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-block")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-block", "read")

	// Phase 88: set Origin header so requireAllowedOrigin passes.
	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())
	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-block/ws?cap="+token,
		headers,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame([]byte("should-be-blocked\n"))); err != nil {
		t.Fatalf("write input frame: %v", err)
	}

	readPipeMustTimeout(t, inputReader, 300*time.Millisecond, "read cap must block MsgInput at relay")
}

// TestSecurity_ReconnectWithoutReadonlyStillBlocked covers the SEC-05
// regression that motivated the phase: a read-only cap still blocks write
// even when the client reconnects WITHOUT any ?readonly= parameter.
func TestSecurity_ReconnectWithoutReadonlyStillBlocked(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-reconnect")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-reconnect", "read")

	// Intentionally omit any ?readonly= parameter.
	// Phase 88: set Origin header so requireAllowedOrigin passes.
	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())
	conn := dialWebServerWS(t, client,
		ws.BaseURL(),
		"/sessions/sess-reconnect/ws?cap="+token,
		headers,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame([]byte("ghost-write\n"))); err != nil {
		t.Fatalf("write input frame: %v", err)
	}

	readPipeMustTimeout(t, inputReader, 300*time.Millisecond, "reconnect without readonry flag must still block write")
}

// TestCapability_MissingCapReturns401 covers the missing-?cap= path: no
// token at all returns 401.
func TestCapability_MissingCapReturns401(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("s-1")

	resp, err := client.Get(ws.BaseURL() + "/api/sessions")
	if err != nil {
		t.Fatalf("GET /api/sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without ?cap=, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "capability required") {
		t.Errorf("expected body 'capability required', got %q", string(body))
	}
}

// TestCapability_InvalidSignatureReturns401 covers the bad-signature path:
// a syntactically valid token with a wrong sig also returns 401.
func TestCapability_InvalidSignatureReturns401(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("s-1")

	// Build a token signed with a DIFFERENT key — segments decode cleanly but
	// the HMAC won't match the server's key.
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = 0xFF
	}
	claims := capability.Claims{SID: "s-1", Perms: "read,write", IAT: time.Now().Unix(), GrantID: "g-1", V: 1}
	token, err := capability.Sign(claims, wrongKey)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	resp, err := client.Get(ws.BaseURL() + "/api/sessions?cap=" + token)
	if err != nil {
		t.Fatalf("GET /api/sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for bad signature, got %d", resp.StatusCode)
	}
}

// TestCapability_RevokedGrantReturns403 covers the grant-list path: valid
// sig, matching SID, but grant_id no longer in the session's active set
// returns 403.
func TestCapability_RevokedGrantReturns403(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("s-revoke")

	token := issueCapFor(t, ws, "s-revoke", "read,write")

	// First call should succeed (grant active).
	resp, err := client.Get(ws.BaseURL() + "/api/sessions?cap=" + token)
	if err != nil {
		t.Fatalf("GET /api/sessions (active grant): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with active grant, got %d", resp.StatusCode)
	}

	// Clear grants — simulates toggle-off (D-15).
	ws.ClearGrants("s-revoke")

	// Second call with the same token should 403.
	resp2, err := client.Get(ws.BaseURL() + "/api/sessions?cap=" + token)
	if err != nil {
		t.Fatalf("GET /api/sessions (revoked): %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 after grant revocation, got %d", resp2.StatusCode)
	}
}

// TestCapability_ValidCapReturnsSession is the happy-path test: a cap with
// valid sig, matching SID, and active grant_id returns 200 with a
// single-item list (D-18 — handleListSessions returns only the bound
// session).
func TestCapability_ValidCapReturnsSession(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == "sess-ok" {
			return "OK Session", "claude", "running", "host-1"
		}
		return id, "", "", ""
	})
	ws.EnableSession("sess-ok")
	// Enable an extra session so a broken implementation that enumerates all
	// enabled sessions would return two items — this test catches D-18
	// regressions (the cap must constrain the response to just its bound
	// session).
	ws.EnableSession("sess-other")

	token := issueCapFor(t, ws, "sess-ok", "read,write")

	resp, err := client.Get(ws.BaseURL() + "/api/sessions?cap=" + token)
	if err != nil {
		t.Fatalf("GET /api/sessions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var items []sessionListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 session in response (D-18), got %d: %+v", len(items), items)
	}
	if items[0].ID != "sess-ok" {
		t.Errorf("expected id=sess-ok, got %q", items[0].ID)
	}
	if items[0].Name != "OK Session" {
		t.Errorf("expected name='OK Session', got %q", items[0].Name)
	}
}

// TestEndToEnd_CapabilityFlow wires the full Phase 87 read-only / read-write
// contract end-to-end against a running WebServer:
//  1. GET /api/sessions/{id}/info with a read cap returns perms == "read"
//     (D-19 / D-23 — terminal.html reads this to suppress the caret).
//  2. GET /api/sessions/{id}/info with a read,write cap returns perms ==
//     "read,write".
//  3. GET /dashboard is publicly accessible (no ?cap= required) — it is the
//     landing page (D-17), not a session-enumeration endpoint.
//  4. After ClearGrants, the info endpoint returns 403 — mirrors toggle-off
//     revocation (D-15) across the info endpoint rather than just the
//     WebSocket upgrade.
//
// This is the SEC-04 UI-level regression lock: the browser now has only one
// input for read-only state (the signed perms claim), and it is server-verified.
func TestEndToEnd_CapabilityFlow(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)

	sid := "e2e-sess"
	ws.EnableSession(sid)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == sid {
			return "E2E Session", "claude", "running", "host-e2e"
		}
		return id, "", "", ""
	})

	// Issue a read-only cap and a read,write cap, both bound to sid.
	roClaims := capability.Claims{
		SID: sid, Perms: "read", IAT: time.Now().Unix(),
		GrantID: "grant-ro", V: 1,
	}
	rwClaims := capability.Claims{
		SID: sid, Perms: "read,write", IAT: time.Now().Unix(),
		GrantID: "grant-rw", V: 1,
	}
	roTok, err := capability.Sign(roClaims, capTestKey)
	if err != nil {
		t.Fatalf("sign ro: %v", err)
	}
	rwTok, err := capability.Sign(rwClaims, capTestKey)
	if err != nil {
		t.Fatalf("sign rw: %v", err)
	}
	ws.AddGrant(sid, roClaims.GrantID)
	ws.AddGrant(sid, rwClaims.GrantID)

	// 1. Info endpoint with read cap returns perms == "read".
	{
		resp, err := client.Get(ws.BaseURL() + "/api/sessions/" + sid + "/info?cap=" + roTok)
		if err != nil {
			t.Fatalf("GET info (ro): %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("info (ro): expected 200, got %d", resp.StatusCode)
		}
		var info struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Perms string `json:"perms"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			t.Fatalf("decode info (ro): %v", err)
		}
		if info.Perms != "read" {
			t.Errorf("info.perms (ro) = %q, want %q", info.Perms, "read")
		}
		if info.ID != sid {
			t.Errorf("info.id (ro) = %q, want %q", info.ID, sid)
		}
	}

	// 2. Info endpoint with read,write cap returns perms == "read,write".
	{
		resp, err := client.Get(ws.BaseURL() + "/api/sessions/" + sid + "/info?cap=" + rwTok)
		if err != nil {
			t.Fatalf("GET info (rw): %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("info (rw): expected 200, got %d", resp.StatusCode)
		}
		var info struct {
			Perms string `json:"perms"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			t.Fatalf("decode info (rw): %v", err)
		}
		if info.Perms != "read,write" {
			t.Errorf("info.perms (rw) = %q, want %q", info.Perms, "read,write")
		}
	}

	// 3. /dashboard is publicly accessible (no ?cap=).
	{
		resp, err := client.Get(ws.BaseURL() + "/dashboard")
		if err != nil {
			t.Fatalf("GET /dashboard: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("/dashboard: expected 200, got %d", resp.StatusCode)
		}
	}

	// 4. ClearGrants revokes access — info endpoint returns 403.
	ws.ClearGrants(sid)
	{
		resp, err := client.Get(ws.BaseURL() + "/api/sessions/" + sid + "/info?cap=" + roTok)
		if err != nil {
			t.Fatalf("GET info (revoked): %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("info (revoked): expected 403, got %d", resp.StatusCode)
		}
	}
}

// TestRequireFilesRead exercises the requireFilesRead wrapper as a unit. The
// wrapper composes requireCapability (HMAC + grant + session-enabled checks)
// with an additional capability.HasPerm(claims.Perms, PermFilesRead) gate. The
// wrapper is defined in Phase 118 but not yet mounted on any route — Phase
// 119 will attach it to /api/files/list, /stat, /read. The unit test therefore
// installs the wrapper on a one-off mux against the *WebServer test instance
// and drives requests through that mux via httptest.
//
// Source: 118-03-PLAN.md Task 2; RESEARCH.md Validation Architecture FS-11/FS-13
// option (a); PITFALLS.md Pitfall 4 (separate wrapper, never modify shared
// requireCapability).
func TestRequireFilesRead(t *testing.T) {
	const sid = "sess-fr"

	// newHarness wires up a *WebServer with capTestKey, an enabled session,
	// and a one-off mux that mounts a sentinel handler under requireFilesRead.
	// The sentinel records whether it ran. ws is also returned so the caller
	// can mint tokens via issueCapFor.
	newHarness := func(t *testing.T) (ws *WebServer, mux *http.ServeMux, sentinelRan *bool) {
		t.Helper()
		ws, _ = testServer(t)
		ws.SetSigningKey(capTestKey)
		ws.EnableSession(sid)
		ran := false
		sentinelRan = &ran
		mux = http.NewServeMux()
		mux.HandleFunc("/test/files", ws.requireFilesRead(func(w http.ResponseWriter, r *http.Request) {
			ran = true
			// Echo perms so the happy-path assertion can confirm the inner
			// handler did receive the verified claims context.
			claims, ok := capability.ClaimsFromContext(r.Context())
			if !ok {
				http.Error(w, "claims not in context", http.StatusInternalServerError)
				return
			}
			_, _ = io.WriteString(w, "OK perms="+claims.Perms)
		}))
		return ws, mux, sentinelRan
	}

	// 1. Pass-through when files.read present.
	t.Run("pass-through when files.read present", func(t *testing.T) {
		ws, mux, ran := newHarness(t)
		token := issueCapFor(t, ws, sid, "read,write,files.read")

		req := httptest.NewRequest(http.MethodGet, "/test/files?cap="+token, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 with files.read in perms, got %d (body=%q)", rec.Code, rec.Body.String())
		}
		if !*ran {
			t.Fatal("sentinel handler was not called even though files.read present")
		}
		if !strings.Contains(rec.Body.String(), "perms=read,write,files.read") {
			t.Errorf("expected echoed perms in body, got %q", rec.Body.String())
		}
	})

	// 2. 403 when files.read missing (read,write only).
	t.Run("403 when files.read missing", func(t *testing.T) {
		ws, mux, ran := newHarness(t)
		token := issueCapFor(t, ws, sid, "read,write")

		req := httptest.NewRequest(http.MethodGet, "/test/files?cap="+token, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 without files.read, got %d (body=%q)", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "files.read") {
			t.Errorf("expected body to contain literal \"files.read\" so frontend can surface message, got %q", rec.Body.String())
		}
		if *ran {
			t.Error("sentinel handler ran on a 403 path — requireFilesRead failed to short-circuit")
		}
	})

	// 3. 401 from requireCapability takes priority over 403 from
	// requireFilesRead — confirms the wrappers chain in the documented order
	// (HMAC verify first, then HasPerm).
	t.Run("401 path takes priority over 403", func(t *testing.T) {
		ws, mux, ran := newHarness(t)
		// Sign with a DIFFERENT key — segments decode cleanly but the HMAC
		// won't match the server's key, so requireCapability returns 401
		// BEFORE requireFilesRead's HasPerm check runs.
		wrongKey := make([]byte, 32)
		for i := range wrongKey {
			wrongKey[i] = 0xFF
		}
		claims := capability.Claims{
			SID: sid, Perms: "read,write,files.read",
			IAT: time.Now().Unix(), GrantID: "grant-wrong-key", V: 1,
		}
		token, err := capability.Sign(claims, wrongKey)
		if err != nil {
			t.Fatalf("capability.Sign: %v", err)
		}
		ws.AddGrant(sid, claims.GrantID)

		req := httptest.NewRequest(http.MethodGet, "/test/files?cap="+token, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 (requireCapability rejects bad sig before requireFilesRead runs), got %d", rec.Code)
		}
		if *ran {
			t.Error("sentinel handler ran on a 401 path")
		}
	})

	// 4. Viewer token (read only) gets 403 with "files.read" in body — the
	// default read-only cap must NOT grant file-browser access (FS-13).
	t.Run("viewer token (read only) gets 403", func(t *testing.T) {
		ws, mux, ran := newHarness(t)
		token := issueCapFor(t, ws, sid, "read")

		req := httptest.NewRequest(http.MethodGet, "/test/files?cap="+token, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for viewer token, got %d (body=%q)", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "files.read") {
			t.Errorf("expected body to contain \"files.read\" for viewer 403, got %q", rec.Body.String())
		}
		if *ran {
			t.Error("sentinel handler ran on a viewer-403 path")
		}
	})
}

// TestRequireCapability_UnchangedByPhase118 is a source-inspection guard that
// pins the invariant from Pitfall 4: the existing requireCapability function
// body MUST NOT mention "files.read" — that check belongs in the SEPARATE
// requireFilesRead wrapper (T-118-14). Adding the check to requireCapability
// would break every existing terminal/relay/plugin route.
func TestRequireCapability_UnchangedByPhase118(t *testing.T) {
	data, err := os.ReadFile("capability_mw.go")
	if err != nil {
		t.Fatalf("read capability_mw.go: %v", err)
	}
	src := string(data)
	// Extract the requireCapability function body — bounded at the closing
	// brace at column 0, which is the canonical end of a Go top-level func.
	// Bounding at the next "\nfunc " would erroneously include the docstring
	// of any function that follows.
	idx := strings.Index(src, "func (ws *WebServer) requireCapability(")
	if idx < 0 {
		t.Fatal("capability_mw.go must declare func (ws *WebServer) requireCapability")
	}
	rest := src[idx:]
	end := strings.Index(rest, "\n}\n")
	if end < 0 {
		// Tolerate EOF without trailing newline.
		end = strings.Index(rest, "\n}")
	}
	if end < 0 {
		t.Fatal("could not locate closing brace of requireCapability")
	}
	body := rest[:end+2] // include the closing "}"
	if strings.Contains(body, "files.read") {
		t.Errorf("requireCapability body must NOT mention \"files.read\" (T-118-14 / Pitfall 4) — use the separate requireFilesRead wrapper instead. Body:\n%s", body)
	}
}

// readPipeMustTimeout asserts that no bytes arrive on r within timeout. The
// positive signal for a blocked write path is a timeout: if even a single
// byte reaches the PTY pipe, the server failed to filter the input.
//
// This differs from capability_test_helpers.go:readPipeWithTimeout — that
// helper asserts bytes DID arrive within the timeout. We need the inverse
// here (nothing arrives) and want a clear fail message tied to the security
// invariant being checked.
func readPipeMustTimeout(t *testing.T, r *io.PipeReader, timeout time.Duration, assertion string) {
	t.Helper()
	done := make(chan []byte, 1)
	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			errCh <- err
			return
		}
		done <- buf[:n]
	}()

	select {
	case data := <-done:
		t.Fatalf("%s: unexpected bytes on PTY pipe: %q", assertion, string(data))
	case err := <-errCh:
		t.Fatalf("%s: pipe read error: %v", assertion, err)
	case <-time.After(timeout):
		// Desired outcome — nothing arrived.
	}
}
