---
phase: "165"
plan: "03"
subsystem: app/wails-bridge
tags: [tailscale, funnel, wails, app-binding, tdd, propagation]
dependency_graph:
  requires: [165-02]
  provides: [App.SetSessionFunnel, DaemonClient.SetSessionFunnel, SessionInfo.FunnelActive, ListSessions-FunnelActive-propagation, testAppWithAPI-helper, appTestFakeFunnelClient-seam]
  affects: [app.go, internal/daemon/client.go, app_test.go, internal/daemon/client_test.go, TESTING.md]
tech_stack:
  added: []
  patterns: [thin-delegation bound method, nil-guard (mirrors ToggleWebServing/SetSessionBrowse), copy-loop field propagation (mirrors HomeDir/BrowseEnabled), appTestFakeFunnelClient ETag read-modify-write fake, testAppWithAPI helper exposing daemon.API for WebServer injection]
key_files:
  created: []
  modified:
    - app.go
    - internal/daemon/client.go
    - app_test.go
    - internal/daemon/client_test.go
    - TESTING.md
decisions:
  - App.SetSessionFunnel mirrors ToggleWebServing/SetSessionBrowse shape exactly (nil-guard + delegate; no Funnel logic in app.go)
  - SessionInfo.FunnelActive carries json:"funnelActive" without omitempty (false must serialize so frontend poll detects expiry — same rule as BrowseEnabled)
  - Propagation test uses testAppWithAPI + appTestFakeFunnelClient so SetSessionFunnel can succeed end-to-end without live tailscaled daemon
  - TESTING.md traceability rows added for app_test.go and internal/daemon/client_test.go covering FNL-01
metrics:
  duration: "~8m"
  completed: "2026-06-30T15:48:22Z"
  tasks_completed: 1
  files_changed: 5
requirements: [FNL-01]
status: complete
---

# Phase 165 Plan 03: Wails Bridge — SetSessionFunnel + FunnelActive Propagation Summary

**One-liner:** `App.SetSessionFunnel` Wails bound method + `DaemonClient.SetSessionFunnel` client delegation + `SessionInfo.FunnelActive bool` mirror field with copy-loop propagation in `App.ListSessions`, completing the FNL-01 enable path end-to-end.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| RED | TDD — add failing tests | 4270eac4 | app_test.go, internal/daemon/client_test.go |
| GREEN | App.SetSessionFunnel + DaemonClient.SetSessionFunnel + FunnelActive | a9bfe492 | app.go, internal/daemon/client.go, app_test.go |
| docs | TESTING.md traceability update | 4d7faab4 | TESTING.md |

## TDD Gate Compliance

- RED gate: `test(165-03)` commit `4270eac4` — 6 compile errors (App.SetSessionFunnel undefined ×2, DaemonClient.SetSessionFunnel undefined ×2, SessionInfo.FunnelActive undefined ×2)
- GREEN gate: `feat(165-03)` commit `a9bfe492` — all 4 new tests pass; full suite green
- REFACTOR gate: not required (code is clean as written)

## Test Results

```
=== RUN   TestListSessions_PropagatesHomeDirAndBrowseEnabled    --- PASS
=== RUN   TestApp_SetSessionFunnel_NilClient                    --- PASS
=== RUN   TestListSessions_PropagatesFunnelActive               --- PASS
ok  github.com/scottkw/agenthub                                 0.058s
=== RUN   TestDaemonClient_SetSessionFunnel                     --- PASS
=== RUN   TestDaemonClient_SetSessionFunnel_ErrorStatus         --- PASS
ok  github.com/scottkw/agenthub/internal/daemon                 0.042s
```

## Key Implementation Details

### `DaemonClient.SetSessionFunnel` (internal/daemon/client.go)

Exact mirror of `ToggleWebServing` / `SetSessionBrowse` — one-line delegation via `c.doJSON`:
```go
func (c *DaemonClient) SetSessionFunnel(sessionID string, enabled bool, expiresIn int) error {
    return c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/funnel",
        SetSessionFunnelRequest{Enabled: enabled, ExpiresIn: expiresIn}, nil)
}
```
Response body (`FunnelURL`) discarded at this layer (GUI retrieves it via `IssueCapabilities`).

### `App.SetSessionFunnel` (app.go)

Thin Wails bound method mirroring `ToggleWebServing` / `SetSessionBrowse`:
- Nil-guards `a.client` → returns `fmt.Errorf("daemon not connected")` (T-165-14)
- Delegates to `a.client.SetSessionFunnel(sessionID, enabled, expiresIn)`
- No Funnel logic (Architectural Responsibility Map: "Thin bridge; no logic")

### `SessionInfo.FunnelActive` (app.go)

Added to `SessionInfo` struct with `json:"funnelActive"` (NOT `omitempty`):
```go
FunnelActive bool `json:"funnelActive"`
```
Copy-loop in `App.ListSessions` propagates from daemon `SessionInfo`:
```go
FunnelActive: s.FunnelActive,
```
Omitting this would silently drop the flag to false — the documented HomeDir-class UAT bug (T-165-15).

### `appTestFakeFunnelClient` + `testAppWithAPI` (app_test.go)

- `testAppWithAPI` — returns `(*App, *daemon.API, *daemon.SessionEngine)` so tests can inject a WebServer via `api.SetWebServerForTest(ws)`.
- `appTestFakeFunnelClient` — stateful `webserver.FunnelClientForTest` double with ETag read-modify-write semantics; mirrors `daemonFakeFunnelClient` from 165-02. Allows `TestListSessions_PropagatesFunnelActive` to call `SetSessionFunnel(true)` end-to-end without a live tailscaled daemon.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestListSessions_PropagatesFunnelActive initial design failed**

- **Found during:** RED → GREEN transition (test run)
- **Issue:** Original test called `app.SetSessionFunnel(id, true, 0)` against `testApp()` (no web server). Handler returns 400 "web server not running", causing test failure even after GREEN implementation.
- **Fix:** Redesigned test to use new `testAppWithAPI` helper + `appTestFakeFunnelClient` (mirrors 165-02 pattern). Injected WebServer with fake funnel client via `api.SetWebServerForTest(ws)` so `handleSetSessionFunnel` can succeed end-to-end.
- **Files modified:** app_test.go
- **Commit:** a9bfe492

## Known Stubs

None. `SessionInfo.FunnelActive` propagates the real daemon value. `App.SetSessionFunnel` delegates to the real daemon endpoint established in 165-02.

## Threat Flags

None. This plan adds a thin delegation layer with no new trust boundaries or security-relevant surfaces beyond what 165-02 already established.

## Self-Check: PASSED

- `app.go` — FOUND (SessionInfo.FunnelActive field, App.SetSessionFunnel, FunnelActive copy in ListSessions)
- `internal/daemon/client.go` — FOUND (DaemonClient.SetSessionFunnel)
- `app_test.go` — FOUND (testAppWithAPI, appTestFakeFunnelClient, TestApp_SetSessionFunnel_NilClient, TestListSessions_PropagatesFunnelActive)
- `internal/daemon/client_test.go` — FOUND (TestDaemonClient_SetSessionFunnel, TestDaemonClient_SetSessionFunnel_ErrorStatus)
- `TESTING.md` — FOUND (Phase 165-03 note + FNL-01 traceability rows for app_test.go + client_test.go)
- `4270eac4` (RED commit) — FOUND
- `a9bfe492` (GREEN commit) — FOUND
- `4d7faab4` (TESTING.md docs commit) — FOUND
- All target tests pass: TestApp_SetSessionFunnel_NilClient, TestDaemonClient_SetSessionFunnel, TestDaemonClient_SetSessionFunnel_ErrorStatus, TestListSessions_PropagatesFunnelActive
- `go build ./...` — PASSED
- `bash tests/check-traceability-paths.sh` — OK (exits 0)
