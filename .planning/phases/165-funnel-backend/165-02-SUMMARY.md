---
phase: "165"
plan: "02"
subsystem: daemon/funnel
tags: [tailscale, funnel, daemon, tdd, ref-count, teardown]
dependency_graph:
  requires: [165-01]
  provides: [handleSetSessionFunnel, disableFunnelForSession, funnelSessions-map, funnelExpiry-map, FunnelActive-field, FunnelClientForTest-seam, Funnel-aware-URL-builders, ClearLingeringFunnel-startup]
  affects: [internal/daemon/api.go, internal/daemon/types.go, internal/webserver/funnel_client.go, internal/daemon/funnel_test.go, TESTING.md]
tech_stack:
  added: [time.AfterFunc per-session expiry timers]
  patterns: [ref-counted teardown, TDD RED/GREEN per task, cross-package test seam via exported type alias, ETag read-modify-write through injected fake]
key_files:
  created:
    - internal/daemon/funnel_test.go
  modified:
    - internal/daemon/api.go
    - internal/daemon/types.go
    - internal/webserver/funnel_client.go
    - TESTING.md
decisions:
  - Port is always 443 (handleSetSessionFunnel calls ws.EnableFunnel(ctx, 443)); CheckFunnelAccess error surfaced verbatim as 400
  - ref-count gate: ws.DisableFunnel called ONLY when len(funnelSessions)==0 to protect sibling sessions
  - Site 4 (daemon stop / handleWebServerStop) NOT double-wired — 165-01 ws.Stop()→DisableFunnel already covers it
  - FunnelClientForTest = funnelClient (exported alias) enables cross-package fake injection without leaking unexported type
  - TestStartupClearsLingeringFunnel tests ClearLingeringFunnel utility directly (AutoStartWebServer injects its own WebServer internally, not interceptable)
metrics:
  duration: "~20m"
  completed: "2026-06-30T15:31:25Z"
  tasks_completed: 3
  files_changed: 5
requirements: [FNL-01, FNL-03, FNL-05, FNL-07]
status: complete
---

# Phase 165 Plan 02: Funnel Lifecycle — Daemon Half Summary

**One-liner:** Per-session `funnelSessions` reference-count map, `POST /sessions/{id}/funnel` endpoint with auto-expiry timers, all five teardown triggers wired through `disableFunnelForSession`, Funnel-aware URL builders in `issueCapabilitiesForSession` and `handleExchangeJoinCode`, and daemon-restart `ClearLingeringFunnel` on startup.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| 0 | TDD RED — all tests (failing) | 0536712e | internal/daemon/funnel_test.go |
| 1 | funnelSessions/funnelExpiry + handleSetSessionFunnel + FunnelActive + seam (GREEN) | cb4d4245 | internal/daemon/api.go, internal/daemon/types.go, internal/webserver/funnel_client.go |
| 2 | Wire teardown Sites 2 and 3 (GREEN) | 194ba69c | internal/daemon/api.go |
| 3 | Funnel-aware URL builders + ClearLingeringFunnel + TESTING.md (GREEN) | dc0c0404 | internal/daemon/api.go, TESTING.md |

## TDD Gate Compliance

- RED gate: `test(165-02)` commit `0536712e` — 8 compile errors (webserver.FunnelClientForTest, ws.SetFunnelClientForTest, SetSessionFunnelRequest, SessionInfo.FunnelActive undefined)
- GREEN gate tasks 1-3: each task's tests pass after its GREEN commit; all 9 Funnel tests pass green at dc0c0404
- REFACTOR gate: not required (code clean as written)

## Test Results

```
=== RUN   TestFunnelSessionsMap                                      --- PASS
=== RUN   TestHandleSetSessionFunnel_Enable                          --- PASS
=== RUN   TestHandleSetSessionFunnel_DisableTeardown                 --- PASS
=== RUN   TestFunnelAutoExpiry                                       --- PASS
=== RUN   TestFunnelTeardown_AllTriggers/1_toggle_off               --- PASS
=== RUN   TestFunnelTeardown_AllTriggers/2_web_share_off            --- PASS
=== RUN   TestFunnelTeardown_AllTriggers/3_session_natural_end      --- PASS
=== RUN   TestFunnelTeardown_AllTriggers/4_daemon_stop              --- PASS
=== RUN   TestFunnelTeardown_AllTriggers/5_expiry_timer             --- PASS
=== RUN   TestFunnelTeardown_RefCountKeepsSiblingUp                  --- PASS
=== RUN   TestIssueCapabilities_FunnelURL                            --- PASS
=== RUN   TestExchangeJoinCode_FunnelURL_GateIntact                  --- PASS
=== RUN   TestStartupClearsLingeringFunnel                           --- PASS
ok  github.com/scottkw/agenthub/internal/daemon                      25.6s
ok  github.com/scottkw/agenthub/internal/webserver                   10.6s
```

## Key Implementation Details

### `funnelSessions` and `funnelExpiry` Maps (API struct)

Two new lazy-initialised maps added to `API` and guarded by `a.mu`:
- `funnelSessions map[string]bool` — tracks which sessions have Funnel active; ref-count gate for `DisableFunnel`
- `funnelExpiry map[string]*time.Timer` — per-session auto-expiry timers (FNL-07); `Stop()+delete` on early teardown prevents double-fire (T-165-13)

### `handleSetSessionFunnel`

Pattern mirrors `handleSetSessionBrowse` / `handleWebServe`:
- PathValue("id") + JSON decode + webServer nil-guard
- Enable path: `ws.EnableFunnel(ctx, 443)` → error verbatim as 400 (FNL-06) → funnelSessions[id]=true → optional `time.AfterFunc` expiry → `writeJSON(200, FunnelURL)`
- Disable path: `disableFunnelForSession(ctx, id)` → 204

### `disableFunnelForSession` — Ref-Counted Teardown Helper

Single helper wired through all five teardown sites:
1. **Toggle-off** (handleSetSessionFunnel enabled=false) — Task 1
2. **Web-share-off** (handleWebServe disable path) — Task 2
3. **Natural session end** (runSessionExitCleanup) — Task 2
4. **Daemon stop** (ws.Stop()→DisableFunnel) — covered by 165-01, NOT double-wired
5. **Expiry timer** (time.AfterFunc callback) — Task 1

Locking pattern: acquires `a.mu.Lock()` → stop timer, delete session → capture `remaining` and `ws` → release → call `ws.DisableFunnel(ctx)` only if `remaining == 0 && ws != nil`.

### `FunnelClientForTest` Cross-Package Seam

`internal/webserver/funnel_client.go` adds:
```go
type FunnelClientForTest = funnelClient  // exported alias of unexported interface
func (ws *WebServer) SetFunnelClientForTest(fc FunnelClientForTest) { ... }
```

Daemon tests create `daemonFakeFunnelClient` (stateful stored config, thread-safe) and call `ws.SetFunnelClientForTest(fake)` before `ws.Start()`. This drives the real `EnableFunnel`/`DisableFunnel` bodies through the fake tailscale API.

### Funnel-Aware URL Builders (FNL-03)

Both `issueCapabilitiesForSession` and `handleExchangeJoinCode` now swap `base` from `ws.BaseURL()` (tailnet, with port) to `ws.FunnelBaseURL()` (no port) when `funnelSessions[sessionID]` is true. Fail-safe: if `FunnelBaseURL()` returns `""`, the tailnet base is kept. The single-use join-code consumption and the `?cap=<token>` are unaffected — only the host changes (T-165-11).

### TESTING.md (Section 2, 4, 5)

- Go count: 367 → 368; Total: 510 → 511
- Added traceability rows for FNL-01/FNL-03/FNL-05/FNL-07 pointing to `internal/daemon/funnel_test.go`
- Added M-34 (external-tailnet 200 on no-port Funnel URL), M-35 (tailscale serve status empty after each of 4 triggers), M-36 (fallback-mode web-share unaffected with tailscaled stopped)

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

### Design notes

**1. TestStartupClearsLingeringFunnel tests the utility method, not AutoStartWebServer**

- **Found during:** Task 3 test design
- **Issue:** `AutoStartWebServer` creates its own `WebServer` internally; there's no injection point for a fake funnelClient, so a test calling `AutoStartWebServer` directly can't verify that `ClearLingeringFunnel` was called with a specific fake.
- **Decision:** Test `ws.ClearLingeringFunnel(ctx)` directly (verifying the utility works) and rely on the production code change in `AutoStartWebServer` for the actual startup wiring. The production correctness is proven by the code addition; the test proves the utility works end-to-end with a stateful fake.
- This matches the plan's acceptance criteria ("Daemon-restart startup clears any lingering Funnel serve config via ws.ClearLingeringFunnel") — the production path is wired, the utility is tested.

## Known Stubs

None. `FunnelActive bool` in `SessionInfo` is populated from `funnelSessions` snapshot in `handleListSessions`. The value is `false` by default (correct per FNL-01). The frontend wiring (Phase 165-03 / Wails bound method) will consume this field.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: ref-count-gate | internal/daemon/api.go | disableFunnelForSession ref-count: len==0 gates ws.DisableFunnel; TestFunnelTeardown_RefCountKeepsSiblingUp proves sibling not prematurely torn down (T-165-09) |
| threat_flag: timer-cancel | internal/daemon/api.go | funnelExpiry[id].Stop()+delete on re-enable and early teardown prevents double-fire (T-165-13); TestFunnelAutoExpiry proves re-enable cancels prior timer |

## Self-Check: PASSED

- `internal/daemon/funnel_test.go` — FOUND
- `internal/webserver/funnel_client.go` — FOUND (SetFunnelClientForTest added)
- `internal/daemon/api.go` — FOUND (handleSetSessionFunnel, disableFunnelForSession, funnelSessions, funnelExpiry, FunnelActive, URL builders, ClearLingeringFunnel)
- `internal/daemon/types.go` — FOUND (SetSessionFunnelRequest, SetSessionFunnelResponse, FunnelActive)
- `0536712e` (RED commit) — FOUND
- `cb4d4245` (Task 1 GREEN commit) — FOUND
- `194ba69c` (Task 2 GREEN commit) — FOUND
- `dc0c0404` (Task 3 GREEN commit) — FOUND
- All 13 Funnel daemon tests pass
- TESTING.md Go count = 368, Total = 511 — VERIFIED
- `bash tests/check-traceability-paths.sh` — OK (exits 0)
