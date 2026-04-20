---
phase: 87-capability-based-session-authorization
plan: 04
subsystem: daemon
tags: [security, daemon, ipc, wails, capability, hmac, joincode]

# Dependency graph
requires:
  - 87-02 capability.Sign/Verify, Claims, FileKeyStore, NewJoinCodeManager
  - 87-03 WebServer.{SetSigningKey, SetJoinCodes, AddGrant, ClearGrants, IsSessionEnabled}
provides:
  - API.BootstrapCapabilityState — loads-or-generates capability.key at daemon startup
  - API.signingKey / signingKeyMu — atomic HMAC key swap state on *API (not *SessionEngine)
  - API.joinCodes — the single JoinCodeManager shared with WebServer.joinCodes
  - API.issueCapabilitiesForSession — mints read + read,write cap pair with join codes
  - API.runSessionExitCleanup — synchronous onExit cleanup for test injection
  - IPC POST /sessions/{id}/capabilities → handleIssueCapabilities
  - IPC POST /join/exchange → handleExchangeJoinCode (410/404/500 error mapping)
  - IPC POST /capability/regenerate-key → handleRegenerateSigningKey (D-16 panic button)
  - DaemonClient.{IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey}
  - Wails bindings App.{IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey, GetCapabilityQRCode}
  - SEC-01 closure: handleCreateSession no longer auto-enables web serving
  - D-15 / Pitfall 1 closure: onExit and toggle-off both call ClearGrants
affects:
  - 87-05-frontend-ui — Plan 05 calls IssueCapabilities after ToggleWebServing(true), renders readUrl/writeUrl
  - 87-06-web-pages-integration — Plan 06 /join page POSTs to /join/exchange and 303-redirects to returned URL
  - 87-VERIFICATION phase gate

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dedicated signingKeyMu sync.RWMutex (separate from a.mu) so capability-lookup hot path doesn't serialize with WebServer operations"
    - "Bootstrap-before-Serve: BootstrapCapabilityState runs BEFORE AutoStartWebServer in runDaemonCore so requireCapability sees a non-nil key on first request (Pitfall 3)"
    - "Short-TTL JoinCodeManager substitution for clock injection across packages — daemon tests use a 50ms TTL + time.Sleep(200ms) since SetClockForTest lives in capability/export_test.go (unreachable from package daemon)"
    - "runSessionExitCleanupForTest: test-only alias on *API that invokes the production cleanup synchronously, letting tests exercise the full ClearGrants+DisableSession path without waiting for the 10-second time.AfterFunc grace timer"
    - "Live HTTPS probeGrant helper in api_test.go: mints a fresh token, hits /api/sessions/{id}/info through the running WebServer, observes grant activeness via 200 vs 403"
    - "Two-commit plan split via file-level revert + diff-replay: task 1 removes auto-enable + adds bootstrap + clears grants; task 2 adds IPC handlers + types + client + Wails bindings on top"

key-files:
  created: []
  modified:
    - internal/daemon/api.go  # struct fields, registerRoutes, BootstrapCapabilityState, AutoStartWebServer/handleWebServerStart wiring, handleCreateSession (remove auto-enable), handleWebServe (clear grants on disable), issueCapabilitiesForSession, 3 new handlers
    - internal/daemon/api_test.go  # 8 new tests + probeGrant + configureCapabilityStateForTest + newLoopbackTLSListener + sameBytes + extractCapToken
    - internal/daemon/types.go  # IssueCapabilitiesResponse, ExchangeJoinCodeRequest, ExchangeJoinCodeResponse
    - internal/daemon/client.go  # IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey typed methods
    - internal/daemon/process.go  # BootstrapCapabilityState wired into runDaemonCore before AutoStartWebServer
    - internal/daemon/engine.go  # gofmt whitespace only
    - internal/daemon/engine_test.go  # gofmt whitespace only
    - internal/daemon/path.go  # gofmt whitespace only
    - app.go  # IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey, GetCapabilityQRCode Wails bindings
    - frontend/src/wailsjs/go/main/App.d.ts  # TypeScript stubs for 4 new Wails bindings + IssueCapabilitiesResponse interface
    - frontend/src/wailsjs/go/main/App.js  # Runtime bridge stubs for 4 new Wails bindings

key-decisions:
  - "signingKey lives on *API, not *SessionEngine (RESEARCH Open Question 1 resolved): capability is a webserver concern, SessionEngine has no web references; placing on *API matches the existing Config.Password plumbing pattern"
  - "Separate signingKeyMu from a.mu: capability-verify is the hot path; serializing it behind the webServer mutex would bottleneck every gated request on unrelated operations"
  - "Toggle-on returns 204 (not IssueCapabilitiesResponse): frontend calls POST /sessions/{id}/capabilities as a separate step (plan §4). DaemonClient.ToggleWebServing discards response body anyway; attaching the cap response would be dead weight. Matches plan's decision after commit dcec027"
  - "runSessionExitCleanup extracted out of the onExit closure so tests can invoke it synchronously without waiting 10 seconds for time.AfterFunc. Production path is unchanged — the closure just calls the extracted method inside AfterFunc"
  - "Short-TTL JoinCodeManager for expiry test: SetClockForTest is export_test.go-only and cannot be called from package daemon. Switched TestIPCHandlers_ExpiredCodeReturns410 to construct a NewJoinCodeManager(50*time.Millisecond) and sleep 200ms. Real behaviour is unchanged for production code"
  - "issueCapabilitiesForSession is task 87-04-01 scope (plan §5 action step), not task 87-04-02 — task 87-04-02 only wraps it in an IPC handler. Tests that exercise the helper are split: TestHandleWebServe_ToggleOffClearsGrants uses it directly (task 1), TestIPCHandlers_CapabilityRoundTrip uses the HTTP route (task 2)"
  - "GetCapabilityQRCode encodes the join-code URL (D-09), not the raw ?cap= URL. Photographing the QR is worthless after 5 minutes or first exchange — this is the intended defence against leaked QR images"
  - "Regenerate-key handler does NOT call ws.ClearGrants: the signature check alone suffices to invalidate all outstanding caps (Perms/GrantID etc. are all signed). Leaving grants stale is acceptable; they clear when sessions end"
  - "handleExchangeJoinCode uses errors.Is for sentinel mapping (matches capability package's fmt.Errorf(%w,...) convention). Ordering: ErrCodeExpired→410, ErrCodeNotFound→404, other→500. Body-decode failure and empty code→400"
  - "Plan 04 committed as two commits (one per task) via the file-level revert + diff-replay technique: first commit restricted Task 1 production code, second commit added Task 2 on top. Preserves atomic-per-task commit semantics even though api.go/api_test.go are shared"

patterns-established:
  - "Startup-time wiring order (D-04): configDir known → BootstrapCapabilityState → NewWebServer → SetSigningKey + SetJoinCodes → Start(). Future capability consumers must respect this order"
  - "Test-only *ForTest alias on *API: runSessionExitCleanupForTest exposes an internal helper for synchronous test invocation without widening the production API surface"
  - "Live-TLS probeGrant helper pattern: self-signed cert via GenerateSelfSignedCert('127.0.0.1') + InsecureSkipVerify HTTPS client, hitting a gated route to observe grant-list state without reaching into unexported WebServer fields"

requirements-completed: [SEC-01, SEC-02, SEC-03, SEC-04, SEC-05]

# Metrics
duration: 22min
completed: 2026-04-20
---

# Phase 87 Plan 04: Daemon API Summary

**Wires the daemon IPC + engine startup to the capability subsystem: removes the SEC-01 auto-enable bug from handleCreateSession, bootstraps capability.key (D-04) before the webserver accepts its first request (Pitfall 3), issues read + read,write capabilities with 5-minute join codes on POST /sessions/{id}/capabilities (D-07 / D-09), clears all session grants on toggle-off and onExit (D-15 / Pitfall 1), and adds 3 IPC endpoints + 4 Wails bindings (IssueCapabilities / ExchangeJoinCode / RegenerateSigningKey / GetCapabilityQRCode) so the GUI can drive the full share flow. All 8 Plan 04 tests GREEN; full daemon suite + internal packages + root package regression-clean; frontend pnpm build clean.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-04-20T17:09:02Z
- **Completed:** 2026-04-20T17:30:43Z
- **Tasks:** 2
- **Files modified:** 11 (8 production Go + 1 test + 2 frontend Wails stubs)

## Accomplishments

- **SEC-01 closed at the daemon layer:** `handleCreateSession` no longer calls `ws.EnableSession(id)` after creating a session. Creating a session while the web server is running leaves the session web-disabled until the user explicitly toggles web-serving ON — the D-06 grant gesture. `TestHandleCreateSession_NoAutoEnable` regression-locks this behaviour; `TestCreateSession_AutoWebEnable` (the old test that asserted auto-enable) was inverted to assert the correct post-fix behaviour.
- **Signing key bootstrapped before first request:** `API.BootstrapCapabilityState` calls `capability.LoadOrGenerate(NewFileKeyStore(configDir))` at daemon startup and stores the 32-byte key under `api.signingKeyMu`. `runDaemonCore` now runs this call BEFORE `AutoStartWebServer`, so `requireCapability` sees a non-nil key on the first request (Pitfall 3). `AutoStartWebServer` and `handleWebServerStart` both wire `ws.SetSigningKey(key)` + `ws.SetJoinCodes(a.joinCodes)` immediately after `NewWebServer` and BEFORE `ws.Start()`.
- **D-15 grant clearance on toggle-off:** `handleWebServe`'s disable path now calls `ws.ClearGrants(id)` alongside `ws.DisableSession(id)`, permanently invalidating previously-issued capabilities for the session.
- **Pitfall 1 grant clearance on session exit:** `handleCreateSession`'s `onExit` callback now calls `ws.ClearGrants(sessionID)` alongside `ws.DisableSession(sessionID)` inside the 10-second grace `time.AfterFunc`. Extracted into `runSessionExitCleanup` so tests can invoke it synchronously via `runSessionExitCleanupForTest`.
- **`issueCapabilitiesForSession` helper:** Mints the read + read,write cap pair (D-07), registers both `grant_id`s via `ws.AddGrant`, signs with the bootstrapped key, and issues single-use 5-minute join codes for each (D-09/D-11). Returns four strings (readURL, writeURL, readCode, writeCode). Used by `handleIssueCapabilities` (task 87-04-02).
- **Three IPC endpoints (task 87-04-02):**
  - `POST /sessions/{id}/capabilities` → `handleIssueCapabilities`: returns `IssueCapabilitiesResponse{readUrl, writeUrl, readCode, writeCode}`. 400 when `ws == nil` or session not web-enabled.
  - `POST /join/exchange` → `handleExchangeJoinCode`: consumes a single-use code, verifies the underlying token, returns `ExchangeJoinCodeResponse{url}`. Error→HTTP mapping: `ErrCodeExpired`→410 Gone, `ErrCodeNotFound`→404, bad body/empty code→400, verify failure or internal error→500, web server not running→400.
  - `POST /capability/regenerate-key` → `handleRegenerateSigningKey`: rotates the HMAC key, saves to disk, updates `api.signingKey` and `ws.signingKey` atomically. All previously-issued caps become invalid (D-16 panic button, intended blast radius).
- **Three typed DaemonClient methods** (`IssueCapabilities`, `ExchangeJoinCode`, `RegenerateSigningKey`) following the existing three-line `doJSON` delegation pattern.
- **Four Wails bindings on `*App`:** `IssueCapabilities`, `ExchangeJoinCode`, `RegenerateSigningKey`, and `GetCapabilityQRCode` (encodes the join-code URL per D-09, not the raw capability token).
- **Frontend Wails stubs registered:** `App.d.ts` + `App.js` now export the 4 new bindings with a matching `IssueCapabilitiesResponse` TypeScript interface, unblocking Plan 05 frontend work.

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove auto-enable, add signing key startup, wire grant issuance/clearance on toggle, fix onExit grant cleanup** — `b2871ee` (feat)
2. **Task 2: Add IssueCapabilities / ExchangeJoinCode / RegenerateSigningKey IPC handlers, types, client methods, and Wails bindings** — `b2e2105` (feat)

**Plan metadata commit:** _(appended in final step below)_

## Files Created/Modified

**Created:**
- _(none — this plan only modifies existing files)_

**Modified:**
- `internal/daemon/api.go` — Added `signingKeyMu sync.RWMutex`, `signingKey []byte`, `joinCodes *capability.JoinCodeManager` fields to `*API`. Added `BootstrapCapabilityState`, `CurrentSigningKey`, `JoinCodes` accessors. Added 3 new route registrations (`POST /sessions/{id}/capabilities`, `POST /join/exchange`, `POST /capability/regenerate-key`). `AutoStartWebServer` and `handleWebServerStart` now call `ws.SetSigningKey + ws.SetJoinCodes` before `ws.Start()`. Removed auto-enable block from `handleCreateSession`. `onExit` now calls `runSessionExitCleanup` which clears grants + disables the session. `handleWebServe` disable path calls `ws.ClearGrants(id)`. Added `issueCapabilitiesForSession` helper, `runSessionExitCleanup` + `runSessionExitCleanupForTest`, `handleIssueCapabilities`, `handleExchangeJoinCode`, `handleRegenerateSigningKey`. New imports: `crypto/rand`, `encoding/hex`, `errors`, `github.com/scottkw/agenthub/internal/capability`.
- `internal/daemon/api_test.go` — Added 8 new tests (5 Task 1 + 3 Task 2), plus helpers `configureCapabilityStateForTest`, `probeGrant` (live-HTTPS grant probe), `newLoopbackTLSListener`, `sameBytes`, `extractCapToken`. Inverted the old `TestCreateSession_AutoWebEnable` to `TestHandleCreateSession_NoAutoEnable`.
- `internal/daemon/types.go` — Added `IssueCapabilitiesResponse`, `ExchangeJoinCodeRequest`, `ExchangeJoinCodeResponse` (camelCase JSON tags per project convention).
- `internal/daemon/client.go` — Added `IssueCapabilities`, `ExchangeJoinCode`, `RegenerateSigningKey` typed methods following the existing three-line `doJSON` pattern.
- `internal/daemon/process.go` — `runDaemonCore` now calls `api.BootstrapCapabilityState()` between `NewAPI` and `AutoStartWebServer`; a bootstrap failure is fatal.
- `internal/daemon/engine.go`, `engine_test.go`, `path.go` — gofmt whitespace-only touchups (no behavioural changes).
- `app.go` — Added 4 Wails bindings: `IssueCapabilities`, `ExchangeJoinCode`, `RegenerateSigningKey`, `GetCapabilityQRCode`. All follow the existing `a.client == nil` nil-check + delegation pattern.
- `frontend/src/wailsjs/go/main/App.d.ts` — Registered 4 new bindings + the `IssueCapabilitiesResponse` TypeScript interface.
- `frontend/src/wailsjs/go/main/App.js` — Registered 4 new runtime bridge stubs.

## Decisions Made

- **`signingKey` lives on `*API`, not `*SessionEngine` (RESEARCH Open Question 1 RESOLVED):** The capability subsystem is a webserver concern — `SessionEngine` has no web references and placing capability state on `*API` matches the existing `Config.Password` pattern. `SessionEngine.configDir` is read by `BootstrapCapabilityState` only as a path input.
- **Separate `signingKeyMu` lock (not sharing `a.mu`):** Every capability-gated request performs a signing-key lookup; serializing that hot path behind the `webServer` mutex would create needless contention with unrelated operations like `localPassword` updates or `webServer` lifecycle changes. Dedicated `sync.RWMutex` keeps the capability path lock-local.
- **Toggle-on returns 204 (no capabilities in response body):** The plan locked this after commit `dcec027`. The frontend calls `POST /sessions/{id}/capabilities` as a separate step, so attaching `IssueCapabilitiesResponse` to the toggle-on response would be dead weight that `DaemonClient.ToggleWebServing` discards anyway. Cleaner separation-of-concerns: toggle controls enablement, capabilities endpoint mints tokens.
- **`runSessionExitCleanup` extracted from `onExit` closure:** The production path is unchanged — `time.AfterFunc(10s, runSessionExitCleanup)`. The extraction lets `TestOnExit_ClearsGrants` call the cleanup synchronously without a 10-second sleep. Test-only alias `runSessionExitCleanupForTest` makes the intent explicit at the call site.
- **Short-TTL JoinCodeManager for expiry test (not clock injection):** `SetClockForTest` is defined in `capability/export_test.go` which is only compiled into the capability package's test binary — the daemon test package cannot call it. Rather than exposing clock injection on the production API, `TestIPCHandlers_ExpiredCodeReturns410` constructs a fresh `NewJoinCodeManager(50 * time.Millisecond)` and sleeps 200ms. Production code path is unchanged.
- **`issueCapabilitiesForSession` is Task 87-04-01 scope:** Plan line 222-254 places this helper in task 1's action step. Task 2 only adds the HTTP route + handler that wraps it. This keeps the two-commit split clean: task 1 has the shipping capability minting helper (verifiable by toggle-off grant clearance test), task 2 adds the public IPC surface.
- **`GetCapabilityQRCode` encodes the join-code URL (D-09), not the raw `?cap=` URL:** A photographed/screenshot'd QR is worthless after 5 minutes or first exchange. This is the entire reason the D-09 join-code flow exists; encoding the cap directly would defeat the mitigation.
- **Regenerate-key handler does not call `ws.ClearGrants`:** The signature check alone suffices — any outstanding token will fail `hmac.Equal` against the new key. Clearing grants would be belt-and-suspenders but tightens coupling to `WebServer` internals beyond `SetSigningKey`. Stale grants clear naturally when sessions end.
- **`handleExchangeJoinCode` uses `errors.Is` for sentinel mapping:** Matches the `fmt.Errorf("%w: ...", ErrCodeExpired)` wrapping convention in `capability/joincode.go`. Direct `==` comparison would miss wrapped errors.
- **Two-commit plan split via revert + diff-replay:** `api.go` and `api_test.go` contain both tasks' changes intermixed at the file level. To preserve atomic-per-task commit semantics, I used `git checkout` to temporarily revert the 5 purely-task-2 files (types.go, client.go, app.go, App.d.ts, App.js), manually removed task 2 code from api.go and task 2 tests from api_test.go, committed task 1, then restored task 2 bits and committed task 2. Each commit independently passes `go build ./...` and its own acceptance tests.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Inverted `TestCreateSession_AutoWebEnable` to `TestHandleCreateSession_NoAutoEnable`**
- **Found during:** Task 1 (removing the auto-enable block from `handleCreateSession`).
- **Issue:** The pre-existing test asserted that creating a session while the web server is running auto-enables it (`info.WebEnabled == true`). SEC-01 explicitly removes that behaviour — keeping the old test would cause a failing assertion every time. The old test name also no longer matches the post-fix semantics.
- **Fix:** Renamed the test to `TestHandleCreateSession_NoAutoEnable` (matching the plan's `<behavior>` bullet) and inverted the assertion: it now verifies `ws.IsSessionEnabled(cr.ID) == false` and `info.WebEnabled == false`. This becomes the positive SEC-01 regression lock.
- **Files modified:** `internal/daemon/api_test.go`.
- **Commit:** `b2871ee` (Task 1).

**2. [Rule 3 - Blocking] Short-TTL JoinCodeManager substitute for `SetClockForTest` cross-package access**
- **Found during:** Task 2 (`TestIPCHandlers_ExpiredCodeReturns410`).
- **Issue:** The plan's action step said to advance the clock via `api.joinCodes.SetClockForTest(...)`. But `SetClockForTest` lives in `capability/export_test.go`, which is compiled only into the capability package's test binary — unreachable from `package daemon` tests. Without adjustment the expired-code test could not compile.
- **Fix:** Switched the test to construct a fresh `NewJoinCodeManager(50 * time.Millisecond)` and `time.Sleep(200 * time.Millisecond)` past the TTL. Equivalent behaviour, slightly slower (~200ms vs instant), but no new production API surface and no export_test.go widening.
- **Files modified:** `internal/daemon/api_test.go`.
- **Commit:** `b2e2105` (Task 2).

**3. [Rule 2 - Test Infrastructure] Added `runSessionExitCleanupForTest` alias**
- **Found during:** Task 1 (`TestOnExit_ClearsGrants`).
- **Issue:** The production onExit closure uses `time.AfterFunc(10*time.Second, ...)`, so a naive test would need to sleep 11 seconds. The plan's `<behavior>` bullet suggested "use a 0-second grace for test if possible via injected delay, or use existing test helper pattern".
- **Fix:** Extracted the 10-second-delayed cleanup body into a named method `runSessionExitCleanup(sessionID)` on `*API`. Added `runSessionExitCleanupForTest` alias for explicit test-call-site intent. Production code is unchanged — the closure still calls `runSessionExitCleanup` via `AfterFunc`.
- **Files modified:** `internal/daemon/api.go`.
- **Commit:** `b2871ee` (Task 1).

**4. [Rule 2 - Test Infrastructure] Live-HTTPS `probeGrant` helper in `api_test.go`**
- **Found during:** Task 1 (`TestHandleWebServe_ToggleOffClearsGrants`, `TestOnExit_ClearsGrants`).
- **Issue:** `webserver.WebServer.isGrantActive` is unexported, so daemon tests cannot check grant-activeness directly. The plan's approach — mint a fresh token via the API and observe its fate through the capability-gated HTTP route — is the correct "black-box" test style and requires a live TLS listener on the WebServer.
- **Fix:** Added `probeGrant(t, ws, key, sessionID, grantID)` helper that signs a synthetic token, hits `/api/sessions/{id}/info?cap=...` through the real TLS listener with `InsecureSkipVerify`, and returns true iff the request gets 200. The toggle-off and onExit tests both construct a live self-signed TLS listener via `newLoopbackTLSListener(t)` before using probeGrant.
- **Files modified:** `internal/daemon/api_test.go`.
- **Commit:** `b2871ee` (Task 1).

---

**Total deviations:** 4 auto-fixed (1 inverted stale test, 1 blocking helper-unreachable, 2 test-infrastructure additions).
**Impact on plan:** All 4 deviations preserve the plan's intended behaviour and test coverage. No production-API scope creep; the two new `*ForTest` surfaces (`runSessionExitCleanupForTest`) and the test helper (`probeGrant`) are test-only. The expired-code test takes ~200ms instead of instant, which is still well under the 30-second per-task sampling target.

## Issues Encountered

- **`SetClockForTest` cross-package inaccessibility** — resolved via short-TTL substitution (Deviation 2). No blocker.
- **Interleaved Task 1 / Task 2 edits on `api.go` + `api_test.go`** — the two tasks share several files, so mid-implementation the files contained both tasks' changes. To honour the "commit each task atomically" guideline, I split via file-level revert + diff-replay (documented in "Decisions Made"). Each commit independently builds and passes its respective acceptance tests.

## User Setup Required

None — no external service configuration, secrets, or manual steps required. Daemon startup will automatically generate `~/.config/agenthub/capability.key` (mode 0600) on first run after this plan ships.

## Next Phase Readiness

- **Plan 05 (frontend UI) is unblocked.** The 4 Wails bindings are live on `*App` and registered in the frontend stubs:
  - `IssueCapabilities(sessionID)` → `{readUrl, writeUrl, readCode, writeCode}` — Plan 05 calls this after `ToggleWebServing(sessionID, true)` succeeds, renders the two URLs with Copy/Open/QR buttons.
  - `ExchangeJoinCode(code)` → URL — used if the native app ever implements a "join via code" flow (currently Plan 06 handles this server-side).
  - `RegenerateSigningKey()` — wired to the Settings > Security "Regenerate Signing Key" destructive button Plan 05 builds (D-16).
  - `GetCapabilityQRCode(joinURL)` → base64 PNG — accepts the join-code URL from IssueCapabilities, returns a QR Plan 05 renders inline.
- **Plan 06 (web pages integration) is unblocked.** The daemon's `POST /join/exchange` handler implements the full error-status contract Plan 06's web-layer `/join/exchange` form POST needs. Plan 06 can either proxy to the daemon (via the socket IPC) or mirror the handler inside the webserver using the shared `ws.joinCodes` (which is the same `JoinCodeManager` the daemon populated via `SetJoinCodes`).
- **Plan gate green:**
  - 8/8 Plan 04 tests PASS (TestHandleCreateSession_NoAutoEnable, TestHandleWebServe_ToggleOnEnablesSession, TestHandleWebServe_ToggleOffClearsGrants, TestOnExit_ClearsGrants, TestStartup_LoadsOrGeneratesSigningKey, TestIPCHandlers_CapabilityRoundTrip, TestIPCHandlers_ExpiredCodeReturns410, TestIPCHandlers_RegenerateSigningKey_SwapsKey).
  - Full daemon suite `go test ./internal/daemon/ -count=1` GREEN (~6.5s).
  - All internal packages `go test ./internal/... -count=1` GREEN.
  - Root package `go test . -count=1` GREEN (~40s).
  - Frontend `cd frontend && pnpm build` GREEN (TypeScript + Vite).
  - `gofmt -l` clean on all modified Go files.
  - `go vet ./internal/...` clean.
- **Static-grep acceptance criteria (Task 1):**
  - `grep -q "capability.LoadOrGenerate" internal/daemon/api.go` ✓
  - `grep -q "capability.NewFileKeyStore" internal/daemon/api.go` ✓
  - `grep -q "SetSigningKey" internal/daemon/api.go` ✓
  - `grep -q "SetJoinCodes" internal/daemon/api.go` ✓
  - `! grep -qE 'ws\.EnableSession\(id\)\s*$' internal/daemon/api.go` ✓ (auto-enable block removed)
  - `grep -c "ClearGrants" internal/daemon/api.go` = 2 ✓
  - `grep -q "issueCapabilitiesForSession" internal/daemon/api.go` ✓
  - `grep -q "AddGrant" internal/daemon/api.go` ✓
  - `grep -q '"read,write"' internal/daemon/api.go` ✓
  - `grep -qE "signingKey\s+\[\]byte" internal/daemon/api.go` ✓ (gofmt aligned the whitespace but field exists on `*API`)
- **Static-grep acceptance criteria (Task 2):**
  - `grep -q "IssueCapabilitiesResponse" internal/daemon/types.go` ✓
  - `grep -q "ExchangeJoinCodeRequest" internal/daemon/types.go` ✓
  - `grep -q "handleIssueCapabilities" internal/daemon/api.go` ✓
  - `grep -q "handleExchangeJoinCode" internal/daemon/api.go` ✓
  - `grep -q "handleRegenerateSigningKey" internal/daemon/api.go` ✓
  - `grep -q "POST /sessions/{id}/capabilities" internal/daemon/api.go` ✓
  - `grep -q "POST /join/exchange" internal/daemon/api.go` ✓
  - `grep -q "POST /capability/regenerate-key" internal/daemon/api.go` ✓
  - Client 3 methods ✓, App 4 bindings ✓.
- **No blockers.** The plan's `<success_criteria>` are all satisfied.

## Self-Check: PASSED

All 11 modified files verified present on disk:
- `internal/daemon/api.go` FOUND
- `internal/daemon/api_test.go` FOUND
- `internal/daemon/types.go` FOUND
- `internal/daemon/client.go` FOUND
- `internal/daemon/process.go` FOUND
- `internal/daemon/engine.go` FOUND
- `internal/daemon/engine_test.go` FOUND
- `internal/daemon/path.go` FOUND
- `app.go` FOUND
- `frontend/src/wailsjs/go/main/App.d.ts` FOUND
- `frontend/src/wailsjs/go/main/App.js` FOUND

Both task commits found in git log:
- `b2871ee` — feat(87-04): remove auto-enable, wire capability signing key + join-code startup, clear grants on toggle-off and session exit
- `b2e2105` — feat(87-04): add IssueCapabilities / ExchangeJoinCode / RegenerateSigningKey IPC handlers, client methods, and Wails bindings

Test run re-verified (all 8 Plan 04 tests PASS):

```
=== RUN   TestHandleCreateSession_NoAutoEnable
--- PASS: TestHandleCreateSession_NoAutoEnable (0.02s)
=== RUN   TestHandleWebServe_ToggleOnEnablesSession
--- PASS: TestHandleWebServe_ToggleOnEnablesSession (0.01s)
=== RUN   TestHandleWebServe_ToggleOffClearsGrants
--- PASS: TestHandleWebServe_ToggleOffClearsGrants (0.04s)
=== RUN   TestOnExit_ClearsGrants
--- PASS: TestOnExit_ClearsGrants (0.02s)
=== RUN   TestStartup_LoadsOrGeneratesSigningKey
--- PASS: TestStartup_LoadsOrGeneratesSigningKey (0.00s)
=== RUN   TestIPCHandlers_CapabilityRoundTrip
--- PASS: TestIPCHandlers_CapabilityRoundTrip (0.02s)
=== RUN   TestIPCHandlers_ExpiredCodeReturns410
--- PASS: TestIPCHandlers_ExpiredCodeReturns410 (0.22s)
=== RUN   TestIPCHandlers_RegenerateSigningKey_SwapsKey
--- PASS: TestIPCHandlers_RegenerateSigningKey_SwapsKey (0.02s)
PASS
ok  	github.com/scottkw/agenthub/internal/daemon	0.369s
```

Full suite `go test ./internal/... . -count=1` PASS (all 12 packages green; pre-existing security-review/ scaffold failure is out-of-scope).

---
*Phase: 87-capability-based-session-authorization*
*Completed: 2026-04-20*
