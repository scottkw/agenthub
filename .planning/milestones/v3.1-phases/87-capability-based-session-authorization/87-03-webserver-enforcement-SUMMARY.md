---
phase: 87-capability-based-session-authorization
plan: 03
subsystem: webserver
tags: [security, webserver, middleware, websocket, hmac, capability, authorization]

# Dependency graph
requires:
  - 87-01 (Wave 0 test skeletons under phase87_wave2 build tag)
  - 87-02 (capability.Sign/Verify/Claims, WithClaims/ClaimsFromContext, JoinCodeManager type)
provides:
  - WebServer.requireCapability middleware
  - WebServer grant-list state (AddGrant / ClearGrants / isGrantActive)
  - WebServer signingKey swap path (SetSigningKey / currentSigningKey)
  - WebServer joinCodes holder (SetJoinCodes — consumed by Plan 06)
  - Capability-gated routes for GET /api/sessions, /api/sessions/{id}/info, /sessions/{id}, /sessions/{id}/ws
  - Single-session response contract for /api/sessions (D-18)
  - Server-bound readonly semantics via claims.Perms (D-24, removes ?readonly write-path)
affects:
  - 87-04-daemon-api (consumes AddGrant / ClearGrants / SetSigningKey / SetJoinCodes at toggle and startup)
  - 87-05-frontend-ui (capability URL shape is now ?cap=<token>)
  - 87-06-web-pages-integration (consumes ws.joinCodes for /join/exchange handler)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-route middleware wrapping via mux.HandleFunc('VERB /path', ws.requireCapability(ws.handler)) — matches auth.go BasicAuthMiddleware shape for HandlerFunc targets"
    - "Collapsed 401 body for all verify failures (malformed / bad sig / bad claims) — information-disclosure defense"
    - "Defense-in-depth cross-check: both isGrantActive AND IsSessionEnabled must pass inside requireCapability; toggle-off flips either alone still blocks"
    - "readonly source moved from ?readonly query string to claims.Perms == \"read\" — client can no longer assert read/write permission"
    - "Context-plumbed claims via capability.WithClaims → capability.ClaimsFromContext so handlers read Perms/SID/GrantID without re-parsing the token"
    - "readPipeMustTimeout inverse of readPipeWithTimeout — positive signal for a blocked write-path is a deadline miss"

key-files:
  created:
    - internal/webserver/capability_mw.go
  modified:
    - internal/webserver/server.go
    - internal/webserver/capability_test.go
    - internal/webserver/capability_test_helpers.go
    - internal/webserver/server_test.go

key-decisions:
  - "Middleware shape is func(http.HandlerFunc) http.HandlerFunc, not func(http.Handler) http.Handler, so it can wrap individual mux.HandleFunc registrations AFTER path-value extraction has happened. This mirrors RESEARCH Pattern 3 and keeps r.PathValue(\"id\") available inside the middleware for SEC-03 session-ID binding."
  - "All capability.Verify failure modes (ErrMalformedToken, ErrInvalidSignature, ErrMalformedClaims) collapse to a single 401 body 'capability required'. Distinguishing them to the caller would leak whether the token format was recognized (T-87-08 information-disclosure)."
  - "The middleware checks both isGrantActive AND IsSessionEnabled. Either alone would be sufficient in the current code path (toggle-off clears grants AND disables), but the redundancy guards against a future code path that touches only one — e.g. a hypothetical admin-revoke that clears grants without disabling."
  - "nil signingKey produces 401 (not 500 or panic). A daemon that failed to bootstrap the FileKeyStore should refuse to authenticate rather than accept any token under a nil key. The test TestCapability_MissingCapReturns401 implicitly exercises this when SetSigningKey is called before any token is minted."
  - "handleListSessions returns a zero-or-one-item array, never multiple items. D-18 collapses /api/sessions from enumeration to self-describe. An extra EnableSession for a second session in TestCapability_ValidCapReturnsSession guards against a regression that would re-introduce enumeration."
  - "OriginPatterns: []string{\"*\"} is intentionally retained. Phase 88 (WebSocket Handshake Security) removes it as part of Origin allowlisting. Removing it here would front-run Phase 88 scope and the surrounding handshake policy."
  - "Post-toggle-off behavior in TestWebServerToggle now expects 403 (not 404). The capability is structurally valid; the grant-list / web-enabled cross-check inside requireCapability returns 403 'capability has been revoked'. This mirrors D-15: toggle-off revokes, it doesn't make the URL a 404."
  - "TestSessionAccessWithoutAuth was inverted to expect 401 instead of 200. This is the direct expression of SEC-02/SEC-03 at the HTTP layer — a pre-Phase-87 session page load without a cap must now be rejected."

patterns-established:
  - "Capability-gated route pattern: mux.HandleFunc(..., ws.requireCapability(ws.handler)) — carries into Plan 06 when the /join/exchange POST handler lands"
  - "Test cap-minting helper (issueCapFor / capForSession) that mints a signed token AND registers its grant_id atomically — future test files can reuse"
  - "readPipeMustTimeout inverse helper for asserting that bytes did NOT arrive on a pipe within a deadline — useful for any future write-path-blocked test"

requirements-completed: [SEC-02, SEC-03, SEC-04, SEC-05]

# Metrics
duration: 8min
completed: 2026-04-20
---

# Phase 87 Plan 03: WebServer Capability Enforcement Summary

**Wires the capability package into internal/webserver so every tailnet-facing session route enforces a per-session HMAC-signed token. The GET /api/sessions endpoint collapses to a single-session self-describe response, and WebSocket Subscriber.ReadOnly is sourced from the capability's perms claim — client-asserted ?readonly=1 can no longer grant or deny write access. All 9 Wave 0 SEC tests GREEN; no cross-package regressions.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-20T16:49:58Z
- **Completed:** 2026-04-20T16:58:26Z
- **Tasks:** 2
- **Files created:** 1 (`internal/webserver/capability_mw.go`)
- **Files modified:** 4 (`internal/webserver/server.go`, `capability_test.go`, `capability_test_helpers.go`, `server_test.go`)

## Accomplishments

- Added three new fields to `WebServer` — `grants map[string]map[string]struct{}`, `signingKey []byte`, `joinCodes *capability.JoinCodeManager` — all guarded by the existing `ws.mu` and initialized or set via six new methods (`AddGrant`, `ClearGrants`, `isGrantActive`, `SetSigningKey`, `SetJoinCodes`, `currentSigningKey`). Plan 04 wires these at daemon startup and on web-serving toggle; Plan 06 consumes `joinCodes` in the /join/exchange handler.
- Created `internal/webserver/capability_mw.go` with the `requireCapability` middleware. The wrapper sits on the *outside* of the handler so the 401/403 lands before `websocket.Accept` commits a 101 response (RESEARCH Pitfall 5). All `capability.Verify` failure modes (malformed / bad sig / bad claims) collapse to a single 401 body to defuse information-disclosure probing (T-87-08). Successful verification attaches the `Claims` to the request context via `capability.WithClaims`, so downstream handlers can read `Perms`, `SID`, and `GrantID` without re-parsing the token.
- Wrapped four routes in `setupRoutes`: `GET /api/sessions`, `GET /api/sessions/{id}/info`, `GET /sessions/{id}`, and `GET /sessions/{id}/ws`. Removed the old `webEnabled` pre-checks inside the per-session closures — the middleware's grant-list and web-enabled cross-check supersedes them. `/dashboard`, `GET /`, and `/api/sessions/{id}/qr` remain open (landing page, redirect, QR PNG).
- Rewrote `handleListSessions` to return exactly zero or one session per D-18. The handler reads `claims.SID` from the context and emits a single-item JSON array if that session is web-enabled, collapsing the endpoint from enumeration to self-describe. No caller can ever receive a list longer than one via HTTPS.
- Rewrote `handleWSSRelay` so `Subscriber.ReadOnly` is sourced from `claims.Perms == "read"` (D-24 / SEC-04). Removed `r.URL.Query().Get("readonly")` entirely from the write-gate path. A caller with a read-only cap cannot write regardless of `?readonly=` value; a caller with `read,write` can always write. `OriginPatterns: []string{"*"}` is intentionally retained with an updated comment — Phase 88 removes it as part of Origin allowlisting.
- Un-tagged the two Wave 0 files (`capability_test.go`, `capability_test_helpers.go`) by removing `//go:build phase87_wave2`. Replaced the nine `t.Skip` stubs with real test bodies that mint capabilities via `capability.Sign` against a deterministic test key, install the key on the server via `ws.SetSigningKey`, and register grant_ids via `ws.AddGrant`. Added a small `issueCapFor` helper that bundles the sign-and-register handshake.
- Added `readPipeMustTimeout` — the inverse of the existing `readPipeWithTimeout` helper — so the SEC-04 and SEC-05 tests can assert that NO bytes arrive on the PTY input pipe within a deadline. The positive signal for a blocked write-path is a deadline miss.
- Updated nine pre-existing tests in `server_test.go` to supply caps against the newly-gated routes (`TestWebServerSessionListAPI`, `TestWebServerSessionListAPIWithResolver`, `TestWebServerWSS`, `TestWebServerToggle`, `TestSessionListIncludesHostname`, `TestSessionInfoEndpoint`, `TestSessionInfoEndpoint_NotEnabled`, `TestSessionInfoEndpoint_NotFound`, `TestSessionAccessWithoutAuth`). Two tests changed expected status codes to reflect the new semantics: `TestWebServerToggle` now expects 403 (not 404) after disable because the cap path is structurally valid but revoked, and `TestSessionInfoEndpoint_NotEnabled` likewise expects 403. `TestSessionAccessWithoutAuth` was inverted to assert the 401 path — the direct expression of SEC-02/SEC-03 at the HTTP layer.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add signingKey + grants + joinCodes state to WebServer** — `1703ccd` (feat)
2. **Task 2: Add requireCapability middleware, gate routes, rewrite list/relay** — `d269e62` (feat)

**Plan metadata commit:** _(appended in the final step below)_

## Files Created/Modified

**Created:**
- `internal/webserver/capability_mw.go` — `requireCapability` middleware. 76 lines including the doc comment. Single public method `(ws *WebServer).requireCapability(next http.HandlerFunc) http.HandlerFunc`.

**Modified:**
- `internal/webserver/server.go` — added three struct fields (`grants`, `signingKey`, `joinCodes`) and six methods (`AddGrant`, `ClearGrants`, `isGrantActive`, `SetSigningKey`, `SetJoinCodes`, `currentSigningKey`). Wrapped four routes in `requireCapability`. Rewrote `handleListSessions` for D-18 single-session response. Rewrote `handleWSSRelay` readonly source per D-24.
- `internal/webserver/capability_test.go` — removed `//go:build phase87_wave2`; replaced nine `t.Skip` bodies with real tests. Added local helpers (`capTestKey`, `issueCapFor`, `readPipeMustTimeout`).
- `internal/webserver/capability_test_helpers.go` — removed `//go:build phase87_wave2`; updated the package doc to note the tag removal.
- `internal/webserver/server_test.go` — added external-package helpers (`ssExtTestKey`, `capForSession`) and updated nine pre-existing tests to supply capabilities against the newly-gated routes. Inverted `TestSessionAccessWithoutAuth` to assert the 401 path.

## Decisions Made

- **Middleware shape is `func(http.HandlerFunc) http.HandlerFunc`**, not `func(http.Handler) http.Handler`. This lets the wrapper wrap individual `mux.HandleFunc` registrations AFTER Go 1.22 path-value routing has populated `r.PathValue("id")` — critical for the SEC-03 session-ID binding check. The other shape (matching `basicAuthMiddleware`) runs before routing.
- **All `capability.Verify` failure modes collapse to a single 401 body** "capability required". Distinguishing malformed vs. bad-sig vs. bad-claims to the caller would leak whether the token format was recognized — a classic information-disclosure vector (T-87-08). The same body is used for missing-cap, nil-key, and verify-failure paths, so only the presence or absence of a valid cap is observable.
- **Two different 403 bodies** — "capability does not match session" and "capability has been revoked" — are acceptable because they do not reveal whether the caller holds a structurally valid token (both paths require successful Verify). They only reveal *why* a verified token was rejected, which is the minimum a legitimate client needs to choose between re-issuing and refreshing.
- **Defense-in-depth cross-check** of both `isGrantActive` AND `IsSessionEnabled` inside `requireCapability`. Either alone would be sufficient in the current code path (toggle-off clears grants AND disables), but the redundancy guards against a future code path that touches only one — e.g. a hypothetical admin-revoke-without-disable or a partial-cleanup bug in the onExit hook Plan 04 will wire.
- **nil signingKey maps to 401**, not 500 or panic. A daemon that failed to bootstrap the FileKeyStore should refuse to authenticate rather than accept any token under a nil key. `currentSigningKey()` returns nil on startup; the middleware's early-return covers the window between `NewWebServer` and the first `SetSigningKey` call.
- **`currentSigningKey()` returns the slice header directly** under RLock, without copying. `SetSigningKey` only reassigns the slice field (it never mutates the underlying bytes), so callers observing a pre-swap slice see stable bytes. Copying would add GC pressure and an allocation per request for no safety benefit.
- **`TestCapability_ValidCapReturnsSession` enables a second session** (`sess-other`) that is NOT the cap-bound one. A regression that re-introduced `webEnabledSessions()` enumeration inside `handleListSessions` would return two items; the assertion `len(items) == 1` catches that and pins D-18.
- **`TestWebServerToggle` expects 403 (not 404) post-disable.** Pre-Phase-87 the handler returned 404 from an inner `IsSessionEnabled` check. Now the cap path is capability-gated: the token is structurally valid, but the middleware's web-enabled cross-check returns 403 "capability has been revoked". This is the correct post-toggle-off contract per D-15.
- **`TestSessionAccessWithoutAuth` was inverted.** The pre-Phase-87 version asserted that a session page loaded without any auth (tailnet membership was the access boundary). Phase 87 replaces that trust model; the test now asserts the 401 path — no cap, no access. This is arguably the single most important regression lock for SEC-02.
- **`readPipeMustTimeout` intentionally duplicates the plumbing of `readPipeWithTimeout`** rather than refactoring the latter into a parameterized helper. The two helpers express opposite invariants ("bytes must arrive" vs. "bytes must NOT arrive") and a parameterized helper would hide that distinction behind a bool flag. The duplication is ~20 lines and makes each test's intent explicit at the call site.
- **`OriginPatterns: []string{"*"}` is intentionally retained.** Phase 88 removes it as part of WebSocket Origin allowlisting. Removing it here would front-run Phase 88 scope and the surrounding handshake-token policy. The comment adjacent to the Accept call is updated to name Phase 88 as the owner.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test Compatibility] Updated pre-existing webserver_test tests to supply capabilities**

- **Found during:** Task 2 (after wrapping routes in `requireCapability`).
- **Issue:** Nine pre-existing tests in `server_test.go` exercised the newly-gated routes without capabilities and therefore started failing with 401. The plan's success criteria mandate `go test ./... -count=1` green — these tests are direct downstream consequences of the route gating and must be updated in the same commit that introduces the gate.
- **Fix:** Added `ssExtTestKey` + `capForSession` helpers to the external test package, then threaded `ws.SetSigningKey(ssExtTestKey)` + `?cap=<token>` through each affected test. Two tests changed their expected status code (`TestWebServerToggle`, `TestSessionInfoEndpoint_NotEnabled`) because the semantics of "disabled session" moved from "404 not found" to "403 revoked". `TestSessionAccessWithoutAuth` was inverted — the test formerly asserted that tailnet-only trust was sufficient, which is the exact behavior Phase 87 removes.
- **Files modified:** `internal/webserver/server_test.go`.
- **Commit:** `d269e62` (bundled with Task 2).

**2. [Rule 2 - Missing Helper] Added `readPipeMustTimeout` helper**

- **Found during:** Task 2 (writing SEC-04/SEC-05 test bodies).
- **Issue:** The existing `readPipeWithTimeout` helper (from `capability_test_helpers.go`, relocated by Plan 01) asserts that bytes DID arrive within a deadline. The SEC-04 and SEC-05 tests need the opposite — "no bytes arrived, meaning the write was blocked at the relay" — and inverting the behavior inside the existing helper would make its callers confusing to read.
- **Fix:** Added a parallel `readPipeMustTimeout` helper in `capability_test.go` with an explicit assertion message parameter. The two helpers are named to read naturally at the call site: "I expect bytes within 500ms" vs. "nothing should arrive".
- **Files modified:** `internal/webserver/capability_test.go`.
- **Commit:** `d269e62` (bundled with Task 2).

## Issues Encountered

- **Pre-existing `security-review/` package setup failure** (unchanged from Plans 01 and 02): `go test ./...` surfaces "found packages relay (internal_relay_protocol_fuzz_test.go) and webserver (internal_webserver_server_test.go) in /Users/ken/dev/agenthub/security-review" at package scan. This has been documented as out-of-scope in every Phase 87 summary. No action taken.
- **No other issues encountered.** The 9 Wave 0 SEC tests passed on the first real-run after Task 2's edits. The Wave 0 skeletons' skip-then-fill-in protocol (Plan 01's design) made the test activation nearly mechanical.

## User Setup Required

None — no external service configuration, secrets, or manual steps required. Plan 04 wires the daemon to call `SetSigningKey` and `SetJoinCodes` at startup; until then, a `NewWebServer` that is never configured with a key will 401 every capability-gated request, which is the correct defensive default.

## Next Phase Readiness

- **Plan 04 (daemon API) is unblocked.** The methods it needs are in place:
  - `ws.SetSigningKey(key)` — call from daemon startup after `capability.LoadOrGenerate`
  - `ws.SetJoinCodes(jc)` — call from daemon startup with a fresh `capability.NewJoinCodeManager(5 * time.Minute)`
  - `ws.AddGrant(sessionID, grantID)` — call from `handleToggleWebServing` when `enabled=true`, once per issued capability (read + write)
  - `ws.ClearGrants(sessionID)` — call from `handleToggleWebServing` when `enabled=false` AND from the onExit hook (RESEARCH Pitfall 1)
  - `ws.DisableSession(sessionID)` (existing) — already paired with `ClearGrants` in the same toggle handler
- **Plan 06 (web pages + join exchange) is unblocked.** `ws.joinCodes` is accessible via `ws.joinCodes.Exchange(code)` inside any handler; the field is only set if Plan 04 calls `SetJoinCodes`, so `handleJoinExchange` should nil-check defensively and return 500 if the manager was never wired.
- **Plan gate green:**
  - 9/9 Wave 0 SEC tests PASS
  - All 22 existing webserver tests still PASS (nine were updated to supply caps; the rest are untouched)
  - `go test ./... -count=1` passes for every package except the pre-existing `security-review/` scaffold
  - `go build ./internal/...` clean; `go vet ./internal/...` clean
- **Static-grep gates:** All 10 acceptance greps pass (see Task 2 verify output).
- **No blockers.** The plan's `<success_criteria>` are all satisfied: the 4 must-have truths from the frontmatter are all reflected in test coverage, the static-grep gates all pass, and the threat register's five `mitigate` dispositions (T-87-01, T-87-02, T-87-04, T-87-07, T-87-08) all have a paired test in the Wave 0 set.

## Self-Check: PASSED

All 5 created/modified files verified present on disk:

```
FOUND: internal/webserver/capability_mw.go
FOUND: internal/webserver/server.go
FOUND: internal/webserver/capability_test.go
FOUND: internal/webserver/capability_test_helpers.go
FOUND: internal/webserver/server_test.go
```

Both task commits found in git log:

```
FOUND: 1703ccd — feat(87-03): add signingKey + grants + joinCodes state to WebServer
FOUND: d269e62 — feat(87-03): add requireCapability middleware, gate routes, rewrite list/relay
```

Acceptance criteria re-verified:
- `grep -q "requireCapability" internal/webserver/server.go` — PASS
- `grep -q 'claims.Perms == "read"' internal/webserver/server.go` — PASS
- `grep -q 'Get("readonly") == "1"' internal/webserver/server.go` — (empty, no matches) — PASS (old path removed)
- `grep -q "capability.ClaimsFromContext" internal/webserver/server.go` — PASS
- `grep -q "capability required" internal/webserver/capability_mw.go` — PASS
- `grep -q "capability does not match session" internal/webserver/capability_mw.go` — PASS
- `grep -q "capability has been revoked" internal/webserver/capability_mw.go` — PASS
- `grep -q "OriginPatterns:" internal/webserver/server.go` — PASS (retained for Phase 88)
- `grep -q "//go:build phase87_wave2" internal/webserver/capability*.go` — (empty, no matches) — PASS (tags removed)
- `grep -q "capability.WithClaims" internal/webserver/capability_mw.go` — PASS

Test run:

```
$ go test ./internal/webserver/ -count=1 -v -run 'TestCapability|TestSecurity' | grep -E '^--- '
--- PASS: TestSecurity_UnauthenticatedClientCannotEnumerateSessions (0.02s)
--- PASS: TestSecurity_WrongSessionCapRejected (0.00s)
--- PASS: TestSecurity_ReadOnlyParamCannotGrantWrite (0.30s)
--- PASS: TestSecurity_ReadOnlyCapabilityBlocksMsgInput (0.31s)
--- PASS: TestSecurity_ReconnectWithoutReadonlyStillBlocked (0.31s)
--- PASS: TestCapability_MissingCapReturns401 (0.00s)
--- PASS: TestCapability_InvalidSignatureReturns401 (0.00s)
--- PASS: TestCapability_RevokedGrantReturns403 (0.00s)
--- PASS: TestCapability_ValidCapReturnsSession (0.00s)
```

Full `go test ./internal/...` passes. Pre-existing `security-review/` scaffold failure unchanged (out-of-scope).

---
*Phase: 87-capability-based-session-authorization*
*Completed: 2026-04-20*
