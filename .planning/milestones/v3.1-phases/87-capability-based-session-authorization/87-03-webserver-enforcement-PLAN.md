---
phase: 87
plan: 03
type: execute
wave: 2
depends_on: [1, 2]
files_modified:
  - internal/webserver/capability_mw.go
  - internal/webserver/server.go
  - internal/webserver/capability_test.go
  - internal/webserver/capability_test_helpers.go
autonomous: true
requirements:
  - SEC-02
  - SEC-03
  - SEC-04
  - SEC-05
tags:
  - security
  - webserver
  - middleware
  - websocket

must_haves:
  truths:
    - "GET /api/sessions without a valid ?cap= token returns HTTP 401 with plain-text body 'capability required'"
    - "GET /api/sessions with a valid cap returns JSON containing ONLY the single session the cap is bound to (D-18)"
    - "GET /sessions/{id}/ws with no cap or a cap for a different session ID returns 401 or 403"
    - "GET /sessions/{id}/ws with a read-only cap sets Subscriber.ReadOnly = true regardless of ?readonly query string"
    - "Relay drops MsgInput frames from read-only subscribers (existing check) fed by cap perms (SEC-05)"
    - "Reconnect without ?readonly=1 using the same read-only cap still blocks MsgInput"
    - "WebServer exposes AddGrant / ClearGrants / isGrantActive for Plan 04 to call on toggle-on/off"
    - "SetSigningKey swap is race-free and applies to subsequent requireCapability checks"
  artifacts:
    - path: internal/webserver/capability_mw.go
      provides: "requireCapability middleware, signingKey accessor, grant-list read method"
      contains: "func .* requireCapability"
    - path: internal/webserver/server.go
      provides: "grants map field, AddGrant/ClearGrants/isGrantActive methods, signingKey field + SetSigningKey, route wrapping with requireCapability, handleListSessions single-session response, handleWSSRelay readonly from cap perms, handleTerminalPage wrapped"
      contains: "requireCapability"
  key_links:
    - from: internal/webserver/server.go handleWSSRelay
      to: capability.ClaimsFromContext
      via: "context-plumbed claims from requireCapability middleware"
      pattern: "capability\\.ClaimsFromContext"
    - from: internal/webserver/server.go setupRoutes
      to: "mux.HandleFunc wrapping"
      via: "ws.requireCapability(ws.handleListSessions) etc."
      pattern: "requireCapability\\("
    - from: internal/webserver/server.go Subscriber.ReadOnly
      to: 'claims.Perms == "read"'
      via: "server-bound read-only per D-24"
      pattern: 'Perms == "read"'
---

<objective>
Wire the capability package (Plan 02) into `internal/webserver/server.go`: add the `requireCapability` middleware, wrap the four guarded routes (`GET /api/sessions`, `GET /api/sessions/{id}/info`, `GET /sessions/{id}`, `GET /sessions/{id}/ws`), rewrite `handleListSessions` to return only the cap-bound session (D-18), rewrite `handleWSSRelay` to source `Subscriber.ReadOnly` from `claims.Perms` (D-24), and add the grant list state (`AddGrant`, `ClearGrants`, `isGrantActive`) that Plan 04 will populate on toggle-on.

Purpose: Deliver SEC-02 (listing gated), SEC-03 (per-session cap binding), SEC-04 (read-only is server-bound), SEC-05 (MsgInput rejected at relay). This is the HTTP-layer enforcement boundary.

Output: One new middleware file, one modified server.go, Wave 0 webserver tests all GREEN. Phase 87's read-only bypass regression test passes.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-CONTEXT.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md
@internal/webserver/server.go
@internal/webserver/auth.go
@internal/webserver/server_test.go
@internal/relay/hub.go
@internal/capability/capability.go
@internal/capability/context.go

<interfaces>
Consuming `internal/capability` (from Plan 02):

```go
capability.Verify(token string, key []byte) (Claims, error)
capability.ClaimsFromContext(ctx) (Claims, bool)
capability.WithClaims(ctx, c) context.Context
capability.ErrInvalidSignature, ErrMalformedToken, ErrMalformedClaims
type capability.Claims{ SID, Perms, IAT, GrantID, V }
```

New WebServer exports (consumed by Plan 04):
```go
func (ws *WebServer) AddGrant(sessionID, grantID string)
func (ws *WebServer) ClearGrants(sessionID string)
func (ws *WebServer) SetSigningKey(key []byte) // race-safe swap
```

Internal (used by middleware and handlers):
```go
func (ws *WebServer) isGrantActive(sessionID, grantID string) bool
func (ws *WebServer) requireCapability(next http.HandlerFunc) http.HandlerFunc
```

WebServer struct additions (alongside existing webEnabled at server.go:54):
```go
grants     map[string]map[string]struct{} // sessionID -> set of active grant_ids
signingKey []byte                          // guarded by ws.mu; swapped via SetSigningKey
```
</interfaces>

<security_ordering>
Middleware ordering (CRITICAL per RESEARCH Pitfall 5):
1. basicAuthMiddleware wraps the whole mux at startLocal (server.go:201) — runs FIRST
2. requireCapability wraps individual mux.HandleFunc registrations — runs SECOND
3. handleWSSRelay runs ONLY if both pass; websocket.Accept is inside the wrapped handler

NEVER place requireCapability inside the handler (after websocket.Accept) — upgrade completes before the check fires.
</security_ordering>
</context>

<tasks>

<task type="auto" tdd="true">
  <id>87-03-01</id>
  <name>Task 1: Add signingKey + grants state and SetSigningKey/AddGrant/ClearGrants/isGrantActive methods on WebServer</name>
  <files>internal/webserver/server.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/internal/webserver/server.go (full — especially lines 52-115 for struct and EnableSession pattern)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 316-415 server.go edits; lines 864-895 Pattern A mutex-guarded map)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 413-451 Pattern 4 grant list; Open Question 3 RegenerateSigningKey)
  </read_first>
  <behavior>
    - WebServer struct grows two fields: `grants map[string]map[string]struct{}` and `signingKey []byte`, both guarded by the existing `ws.mu`.
    - Constructor initializes grants to empty map; signingKey defaults to nil until `SetSigningKey` is called (Plan 04 wires daemon to call this at startup).
    - `AddGrant(sid, gid)` acquires ws.mu.Lock, lazily inits inner map, adds grantID, unlocks — mirrors EnableSession at server.go:82-86.
    - `ClearGrants(sid)` acquires ws.mu.Lock, deletes sid from outer map, unlocks — mirrors DisableSession at server.go:88-92.
    - `isGrantActive(sid, gid)` acquires ws.mu.RLock (read-only), returns membership — mirrors IsSessionEnabled at server.go:94-99.
    - `SetSigningKey(key)` acquires ws.mu.Lock, assigns `ws.signingKey = key`, unlocks. Called by daemon at startup and by RegenerateSigningKey handler. signingKey accessor `ws.currentSigningKey()` returns a copy under RLock to avoid races with concurrent Set.
  </behavior>
  <action>
    Edit `internal/webserver/server.go`:

    1. In the WebServer struct (after `webEnabled map[string]bool` at approximately line 54), add:
       ```go
       grants     map[string]map[string]struct{} // D-14: sessionID -> active grant_ids
       signingKey []byte                          // D-04/D-16: guarded by ws.mu; swapped via SetSigningKey
       ```

    2. In the WebServer constructor (the function that returns a new WebServer — likely `NewWebServer` or inline in `startLocal`/`startTailscale`), initialize: `grants: make(map[string]map[string]struct{})`. Do NOT initialize signingKey — leave nil so a missing SetSigningKey call panics early via currentSigningKey nil-check.

    3. Add five new methods in server.go (place them alongside EnableSession/DisableSession/IsSessionEnabled to preserve visual grouping):
       ```go
       func (ws *WebServer) AddGrant(sessionID, grantID string) {
           ws.mu.Lock()
           if ws.grants[sessionID] == nil {
               ws.grants[sessionID] = make(map[string]struct{})
           }
           ws.grants[sessionID][grantID] = struct{}{}
           ws.mu.Unlock()
       }

       func (ws *WebServer) ClearGrants(sessionID string) {
           ws.mu.Lock()
           delete(ws.grants, sessionID)
           ws.mu.Unlock()
       }

       func (ws *WebServer) isGrantActive(sessionID, grantID string) bool {
           ws.mu.RLock()
           defer ws.mu.RUnlock()
           if ws.grants[sessionID] == nil { return false }
           _, ok := ws.grants[sessionID][grantID]
           return ok
       }

       func (ws *WebServer) SetSigningKey(key []byte) {
           ws.mu.Lock()
           ws.signingKey = key
           ws.mu.Unlock()
       }

       func (ws *WebServer) currentSigningKey() []byte {
           ws.mu.RLock()
           defer ws.mu.RUnlock()
           // Return the slice header; callers must not mutate contents.
           // The backing array is only reassigned by SetSigningKey (never mutated in place).
           return ws.signingKey
       }
       ```

    4. Verify the file compiles: `go build ./internal/webserver/...`.

    5. Do NOT yet write the middleware or wrap routes — those are task 87-03-02.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go build ./internal/webserver/... && grep -c "func (ws \*WebServer) AddGrant\|func (ws \*WebServer) ClearGrants\|func (ws \*WebServer) isGrantActive\|func (ws \*WebServer) SetSigningKey\|func (ws \*WebServer) currentSigningKey" internal/webserver/server.go | grep -q "^5$" && grep -q 'grants.*map\[string\]map\[string\]struct{}' internal/webserver/server.go && grep -q 'signingKey.*\[\]byte' internal/webserver/server.go && go test ./internal/webserver/ -count=1 2>&1 | tee /tmp/ws-base.log ; ! grep -q FAIL /tmp/ws-base.log</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/webserver/...` succeeds
    - `grep -q "grants.*map\[string\]map\[string\]struct{}" internal/webserver/server.go` succeeds
    - `grep -q "signingKey.*\[\]byte" internal/webserver/server.go` succeeds
    - All 5 methods present: AddGrant, ClearGrants, isGrantActive, SetSigningKey, currentSigningKey
    - Existing `go test ./internal/webserver/ -count=1` still passes (no regression)
    - grants map initialized in constructor (`grep -q "grants: make" internal/webserver/server.go`)
  </acceptance_criteria>
  <done>WebServer struct carries signingKey + grants. Five accessor methods exist. Package still compiles and existing tests still pass.</done>
</task>

<task type="auto" tdd="true">
  <id>87-03-02</id>
  <name>Task 2: Create requireCapability middleware, wrap routes, rewrite handleListSessions + handleWSSRelay, activate Wave 0 SEC tests</name>
  <files>internal/webserver/capability_mw.go, internal/webserver/server.go, internal/webserver/capability_test.go, internal/webserver/capability_test_helpers.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/internal/webserver/server.go (full)
    - /Users/ken/dev/agenthub/internal/webserver/auth.go (full — middleware factory template)
    - /Users/ken/dev/agenthub/internal/webserver/server_test.go (full — existing helpers + patterns)
    - /Users/ken/dev/agenthub/internal/webserver/capability_test.go (created in Plan 01 — 9 RED tests to activate)
    - /Users/ken/dev/agenthub/internal/webserver/capability_test_helpers.go (created in Plan 01)
    - /Users/ken/dev/agenthub/internal/relay/hub.go (full — Subscriber.ReadOnly field)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 371-411 Pattern 3 middleware; lines 487-509 Pattern 6 readonly rewrite; lines 511-526 Pattern 7 auto-enable removal NOTE — that removal is in Plan 04 not here; lines 595-604 Pitfall 5 middleware placement)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 267-313 capability_mw.go; lines 316-425 server.go edit list; lines 938-952 Pattern E middleware factory; lines 429-467 test activation)
  </read_first>
  <behavior>
    GREEN these Wave 0 tests (from internal/webserver/capability_test.go):
    - TestCapability_MissingCapReturns401 — GET /api/sessions without ?cap — expect 401 body "capability required"
    - TestCapability_InvalidSignatureReturns401 — GET /api/sessions?cap=tampered — expect 401
    - TestCapability_RevokedGrantReturns403 — valid cap whose grant_id is not in WS.grants — expect 403 body "capability has been revoked"
    - TestCapability_ValidCapReturnsSession — valid cap + active grant — returns JSON list containing ONLY claims.SID
    - TestSecurity_UnauthenticatedClientCannotEnumerateSessions (SEC-02) — assert 401
    - TestSecurity_WrongSessionCapRejected (SEC-03) — cap for session A, GET /sessions/B/ws — expect 403 "capability does not match session"
    - TestSecurity_ReadOnlyParamCannotGrantWrite (SEC-04) — dial /sessions/{id}/ws?cap=<readcap>&readonly=0 — Subscriber.ReadOnly should still be true (sourced from cap perms)
    - TestSecurity_ReadOnlyCapabilityBlocksMsgInput (SEC-05) — dial with read-only cap, send MsgInput frame, assert PTY pipe receives 0 bytes via readPipeWithTimeout
    - TestSecurity_ReconnectWithoutReadonlyStillBlocked (SEC-05 regression) — dial with read-only cap, no readonly query string at all, MsgInput still dropped
  </behavior>
  <action>
    1. Create `internal/webserver/capability_mw.go` (remove any build tag from capability_test_helpers.go and capability_test.go: delete the `//go:build phase87_wave2` first lines). Contents:
       ```go
       package webserver

       import (
           "errors"
           "net/http"

           "github.com/kenscott/agenthub/internal/capability" // verify module path via go.mod
       )

       // requireCapability returns a middleware that validates the ?cap= query token,
       // enforces session binding (claims.SID must match PathValue "id" when present),
       // and checks that the grant_id is still active. On success, attaches claims to
       // the request context so downstream handlers can read Perms.
       //
       // Ordering: this wrapper MUST sit on the outside of the handler (see RESEARCH
       // Pitfall 5) so the 401/403 is returned before any WebSocket upgrade fires.
       func (ws *WebServer) requireCapability(next http.HandlerFunc) http.HandlerFunc {
           return func(w http.ResponseWriter, r *http.Request) {
               token := r.URL.Query().Get("cap")
               if token == "" {
                   http.Error(w, "capability required", http.StatusUnauthorized)
                   return
               }
               key := ws.currentSigningKey()
               if key == nil {
                   http.Error(w, "capability required", http.StatusUnauthorized)
                   return
               }
               claims, err := capability.Verify(token, key)
               if err != nil {
                   // Collapse all verify failures to 401 — do not distinguish
                   // malformed from bad-sig from bad-claims (information disclosure).
                   _ = err
                   http.Error(w, "capability required", http.StatusUnauthorized)
                   return
               }
               if pathID := r.PathValue("id"); pathID != "" && claims.SID != pathID {
                   http.Error(w, "capability does not match session", http.StatusForbidden)
                   return
               }
               if !ws.isGrantActive(claims.SID, claims.GrantID) {
                   http.Error(w, "capability has been revoked", http.StatusForbidden)
                   return
               }
               // Also gate on web-enabled for the session — toggle-off disables access
               // even if a cap is otherwise valid (defense in depth beyond grant clear).
               if !ws.IsSessionEnabled(claims.SID) {
                   http.Error(w, "capability has been revoked", http.StatusForbidden)
                   return
               }
               ctx := capability.WithClaims(r.Context(), claims)
               next(w, r.WithContext(ctx))
           }
       }

       // Ensure errors package is referenced so imports tidy cleanly if we wire error typing later.
       var _ = errors.New
       ```

       (Remove the trailing `var _ = errors.New` if errors is used elsewhere; goal is an `errors` import available for downstream typed wrapping.)

    2. Edit `internal/webserver/server.go`. In `setupRoutes` (or wherever the four routes are currently registered, near lines 265-287), replace:
       ```go
       mux.HandleFunc("GET /api/sessions", ws.handleListSessions)
       // ->
       mux.HandleFunc("GET /api/sessions", ws.requireCapability(ws.handleListSessions))

       // Info route (D-19) — add if absent, or wrap if already present:
       mux.HandleFunc("GET /api/sessions/{id}/info", ws.requireCapability(ws.handleSessionInfo))

       // Terminal page (D-17 says dashboard is landing; per-session page remains but is cap-gated):
       mux.HandleFunc("GET /sessions/{id}", ws.requireCapability(ws.handleTerminalPage))

       // WebSocket relay:
       mux.HandleFunc("GET /sessions/{id}/ws", ws.requireCapability(ws.handleWSSRelay))
       ```

       Remove the existing `if !ws.IsSessionEnabled(...) { http.NotFound(...) }` pre-check inside the old `/sessions/{id}` closure — the middleware already enforces it.

    3. Rewrite `handleListSessions` (currently server.go:304-320, loops over all enabled sessions) — make it return ONLY the cap-bound session (D-18):
       ```go
       func (ws *WebServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
           claims, ok := capability.ClaimsFromContext(r.Context())
           if !ok {
               http.Error(w, "capability required", http.StatusUnauthorized)
               return
           }
           items := make([]sessionListItem, 0, 1)
           if ws.IsSessionEnabled(claims.SID) && ws.sessionResolver != nil {
               name, cliType, st, hostname := ws.sessionResolver(claims.SID)
               if name == "" { name = claims.SID }
               items = append(items, sessionListItem{
                   ID: claims.SID, Name: name, CLIType: cliType, Status: st, Hostname: hostname,
               })
           }
           w.Header().Set("Content-Type", "application/json")
           _ = json.NewEncoder(w).Encode(items)
       }
       ```

    4. Rewrite `handleWSSRelay` readonly source (D-24). Currently at server.go:385-391:
       ```go
       // REMOVE:
       // readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"

       // REPLACE WITH:
       claims, _ := capability.ClaimsFromContext(r.Context())
       readonly := claims.Perms == "read"  // server-bound (SEC-04); client cannot override
       ```

       The `sub.ReadOnly = readonly` assignment at server.go:411-415 stays structurally identical — only the source changes.

       Do NOT remove `OriginPatterns: []string{"*"}` — that is Phase 88's work per CONTEXT D-22 (phase coordination). Only update the adjacent comment to: `// Phase 87: capability check has already passed here. Origin allowlist arrives in Phase 88 (WebSocket Handshake Security).`

    5. Add a minimal `handleSessionInfo` handler if one does not already exist. Its body reads claims from context, returns JSON `{id, name, perms}` where `perms` comes directly from `claims.Perms` — this is what the terminal page uses to decide caret suppression (UI-SPEC Surface 5, D-23).

    6. Remove the `//go:build phase87_wave2` build tags from both `internal/webserver/capability_test.go` and `internal/webserver/capability_test_helpers.go`.

    7. Un-skip each of the 9 tests in `capability_test.go`. For each test, use the helper to build a WebServer with a freshly-generated signing key, call `ws.SetSigningKey(key)`, then `ws.EnableSession(sid)` and `ws.AddGrant(sid, grantID)` to set up a valid capability flow. Then `capability.Sign(Claims{SID:sid, Perms:"read" or "read,write", ..., V:1}, key)` to produce the token, construct URL with `?cap=<token>`, assert status/body behavior.

    8. For the MsgInput rejection tests, use `testServerWithHub` helper, dial via `dialWebServerWS`, send a `MsgInput` frame via the websocket API, then `readPipeWithTimeout(t, ptyPipe, "", 200*time.Millisecond)` — assert NO bytes arrive.

    9. Run: `go test ./internal/webserver/ -count=1 -v -run 'TestCapability|TestSecurity'`. All 9 must pass.

    Anti-patterns:
    - NEVER place `requireCapability` inside the handler after `websocket.Accept` — Pitfall 5
    - NEVER distinguish error messages between malformed-token and invalid-sig in the 401 body (information disclosure)
    - NEVER remove `OriginPatterns: ["*"]` — that is Phase 88 scope per D-22
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/webserver/ -count=1 -v -run 'TestCapability|TestSecurity|TestWS|TestHandle' 2>&1 | tee /tmp/ws-cap.log ; ! grep -q FAIL /tmp/ws-cap.log && grep -q 'requireCapability' internal/webserver/server.go && grep -q 'claims.Perms == "read"' internal/webserver/server.go && ! grep -q 'readonly.*readonly=1\|Get("readonly") == "1"' internal/webserver/server.go && grep -q 'capability.WithClaims' internal/webserver/capability_mw.go && ! grep -qE 'OriginPatterns.*\[\]string\{\}' internal/webserver/server.go</automated>
  </verify>
  <acceptance_criteria>
    - `go test ./internal/webserver/ -count=1 -run 'TestCapability|TestSecurity'` exits 0 with all 9 Wave 0 SEC tests PASS
    - `grep -q "requireCapability" internal/webserver/server.go` succeeds (routes wrapped)
    - `grep -q 'claims.Perms == "read"' internal/webserver/server.go` succeeds (D-24)
    - `grep -q 'Get("readonly") == "1"' internal/webserver/server.go` fails (old path removed)
    - `grep -q "capability.ClaimsFromContext" internal/webserver/server.go` succeeds (handler reads claims)
    - `grep -q "capability required" internal/webserver/capability_mw.go` succeeds
    - `grep -q "capability does not match session" internal/webserver/capability_mw.go` succeeds
    - `grep -q "capability has been revoked" internal/webserver/capability_mw.go` succeeds
    - `grep -q "OriginPatterns:" internal/webserver/server.go` still succeeds (NOT removed — Phase 88)
    - `grep -q "//go:build phase87_wave2" internal/webserver/capability*.go` fails (tags removed)
    - `go build ./...` succeeds (full compile)
    - `go test ./... -count=1` passes (no cross-package regression)
  </acceptance_criteria>
  <done>Webserver HTTP + WS layer is capability-gated. All 9 SEC tests pass. ?readonly=1 query parameter no longer influences write access. OriginPatterns still "*" awaiting Phase 88.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| tailnet peer → HTTP | Every /api/sessions, /sessions/{id}/*, and /sessions/{id}/ws request crosses this boundary; capability gate is the sole authz |
| HTTP layer → relay layer | Subscriber.ReadOnly is set from claims.Perms — untrusted client cannot influence |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-87-01 | Elevation of Privilege | /api/sessions enumeration | mitigate | requireCapability middleware on route; handleListSessions returns only bound session (D-18); TestSecurity_UnauthenticatedClientCannotEnumerateSessions, TestCapability_ValidCapReturnsSession in task 87-03-02 |
| T-87-02 | Elevation of Privilege | cross-session cap reuse on /sessions/{id}/ws | mitigate | requireCapability checks claims.SID == r.PathValue("id"); TestSecurity_WrongSessionCapRejected in task 87-03-02 |
| T-87-04 | Elevation of Privilege | read-only bypass via ?readonly omission | mitigate | D-24 rewrites readonly source from query string to claims.Perms; TestSecurity_ReadOnlyParamCannotGrantWrite + TestSecurity_ReconnectWithoutReadonlyStillBlocked in task 87-03-02 |
| T-87-07 | Elevation of Privilege | revoked grant replay | mitigate | isGrantActive check on every request; cleared-grants yields 403; TestCapability_RevokedGrantReturns403 in task 87-03-02 |
| T-87-08 | Information Disclosure | distinguishing verify-failure reasons | mitigate | All verify failures collapse to single 401 body "capability required"; no malformed-vs-sig-vs-claims leak |
</threat_model>

<verification>
- 9 Wave 0 SEC + capability tests green
- `go test ./... -count=1` passes (no regression)
- Static-grep gate: zero occurrences of `readonly=1` in handleWSSRelay write path
- Middleware ordering: basicAuthMiddleware (startLocal:201) runs before requireCapability (wrapping individual routes)
- `OriginPatterns: []string{"*"}` intentionally retained (Phase 88 removes it)
</verification>

<success_criteria>
- 9 new Wave 0 webserver tests PASS
- handleListSessions returns single-item JSON array for valid cap (D-18)
- handleWSSRelay sources Subscriber.ReadOnly from claims.Perms (D-24)
- requireCapability returns 401 for missing/invalid cap, 403 for wrong session or revoked grant
- No test in the existing webserver suite regresses
</success_criteria>

<output>
After completion, create `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-03-SUMMARY.md` documenting: which routes are now capability-gated, the exact 401/403 body strings, the Subscriber.ReadOnly wire-up change, and confirmation that Phase 88's Origin work is NOT encroached upon.
</output>
