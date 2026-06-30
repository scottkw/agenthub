---
phase: "165"
plan: "01"
subsystem: webserver/funnel
tags: [tailscale, funnel, webserver, tdd, origin-allowlist]
dependency_graph:
  requires: []
  provides: [funnelClient-interface, EnableFunnel, DisableFunnel, FunnelBaseURL, ClearLingeringFunnel, dual-origin-allowlist]
  affects: [internal/webserver/server.go, internal/webserver/origin_mw.go, internal/webserver/capability_mw.go]
tech_stack:
  added: [funnelClient injectable interface seam]
  patterns: [ETag read-modify-write, CheckFunnelAccess-before-lock, dual-origin allowlist, RWMutex-safe field access]
key_files:
  created:
    - internal/webserver/funnel_client.go
    - internal/webserver/funnel_test.go
  modified:
    - internal/webserver/server.go
    - internal/webserver/origin_mw.go
    - internal/webserver/capability_mw.go
    - TESTING.md
decisions:
  - Promote lc from stack-local in startTailscale() to WebServer struct field (by value, zero-value usable)
  - Access ws.listener directly inside ws.mu.Lock() rather than calling ws.Addr() to prevent RWMutex deadlock
  - CheckFunnelAccess + StatusWithoutPeers called BEFORE ws.mu.Lock() (blocking Unix-socket calls must not hold mutex)
  - funnelBaseURL cached as string rather than recomputed; hostname extracted from it in DisableFunnel via TrimPrefix/TrimSuffix
  - Secondary (Funnel) origin branch in requireAllowedOrigin is fail-closed when FunnelBaseURL()=="" (no Funnel active)
metrics:
  duration: "~6m (context resumption; original session ran longer)"
  completed: "2026-06-30T14:57:14Z"
  tasks_completed: 3
  files_changed: 6
requirements: [FNL-02, FNL-04, FNL-05, FNL-06]
status: complete
---

# Phase 165 Plan 01: Funnel Lifecycle — Webserver Half Summary

**One-liner:** Injectable `funnelClient` interface seam with `EnableFunnel`/`DisableFunnel`/`FunnelBaseURL`/`ClearLingeringFunnel` methods and dual-origin WebSocket allowlist for Tailscale Funnel guests.

## Tasks Completed

| # | Name | Commit | Files |
|---|------|--------|-------|
| 1 | funnelClient interface seam (RED) | 1312fd0e | internal/webserver/funnel_test.go |
| 2 | funnelClient interface seam (GREEN) | 38442225 | internal/webserver/funnel_client.go |
| 3 | EnableFunnel/DisableFunnel lifecycle (GREEN) | 38442225 | internal/webserver/server.go |
| 4 | Dual-origin allowlist (GREEN) | 38442225 | internal/webserver/origin_mw.go, internal/webserver/capability_mw.go |
| 5 | TESTING.md update | 38442225 | TESTING.md |

> Note: Tasks 2–5 (all GREEN work) were committed atomically per TDD protocol.

## TDD Gate Compliance

- RED gate: `test(165-01)` commit `1312fd0e` — all tests compile-fail (funnelClient type undefined)
- GREEN gate: `feat(165-01)` commit `38442225` — all 7 new tests pass; 367 Go tests pass total
- REFACTOR gate: not required (code clean as written)

## Test Results

```
=== RUN   TestFunnelClient_CompileSmoke                  --- PASS
=== RUN   TestEnableFunnelCallsSetServeConfig             --- PASS
=== RUN   TestEnableFunnel_PrereqCheckPreventsSetServeConfig --- PASS
=== RUN   TestEnableFunnel_FallbackModeSafe               --- PASS
=== RUN   TestWebServerStop_DisablesFunnel                --- PASS
=== RUN   TestRequireAllowedOrigin_FunnelOrigin           --- PASS
    --- PASS: TestRequireAllowedOrigin_FunnelOrigin/funnel_origin_passes
    --- PASS: TestRequireAllowedOrigin_FunnelOrigin/tailnet_origin_still_passes
    --- PASS: TestRequireAllowedOrigin_FunnelOrigin/absent_origin_rejected
    --- PASS: TestRequireAllowedOrigin_FunnelOrigin/mismatched_origin_rejected
=== RUN   TestOriginAllowedForWrite_FunnelOrigin          --- PASS
ok  github.com/scottkw/agenthub/internal/webserver        13.019s
```

## Key Implementation Details

### funnelClient Interface (FNL-02)

`internal/webserver/funnel_client.go` defines a narrow 3-method interface mirroring `statusFunc`/`prefsFunc` in `tailscale.go`. `*local.Client` satisfies it in production; `*fakeFunnelClient` (func fields) satisfies it in tests.

### struct fields added to WebServer

```go
lc            local.Client   // promoted from startTailscale() stack local
funnelClient  funnelClient   // injectable; set to &ws.lc by NewWebServer
funnelActive  bool
funnelBaseURL string
funnelPort    uint16
```

### Lock sequencing

`StatusWithoutPeers` + `ipn.CheckFunnelAccess` execute **before** `ws.mu.Lock()` — both involve blocking Unix-socket calls that must not hold the mutex. `GetServeConfig`/`SetServeConfig` and all field writes run **inside** the lock.

### Deadlock prevention

`ws.Addr()` calls `ws.mu.RLock()` — calling it inside `ws.mu.Lock()` would deadlock on `sync.RWMutex`. Instead `ws.listener` is accessed directly inside the locked section to extract the local port.

### Dual-origin allowlist (FNL-04)

`requireAllowedOrigin` checks tailnet URL first (primary), then `FunnelBaseURL()` as secondary. The secondary branch is inert (fail-closed) when `FunnelBaseURL()==""`. Same pattern applied to `originAllowedForWrite` (CSRF guard for write routes).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Deadlock in EnableFunnel calling ws.Addr() inside ws.mu.Lock()**

- **Found during:** Task 2 (GREEN implementation review)
- **Issue:** The 165-PATTERNS.md `EnableFunnel` pattern called `ws.Addr()` to get the local address. `ws.Addr()` calls `ws.mu.RLock()`. Called inside `ws.mu.Lock()`, this would deadlock immediately on `sync.RWMutex`.
- **Fix:** Access `ws.listener` directly inside the locked section: `ln := ws.listener; if ln == nil { return error }; _, localPort, _ := net.SplitHostPort(ln.Addr().String())`
- **Files modified:** internal/webserver/server.go
- **Commit:** 38442225

## Known Stubs

None. The secondary Funnel origin branch is inert until Plan 165-02 calls `ws.EnableFunnel()` from the daemon endpoint, but this is by design (not a stub — the guard condition `FunnelBaseURL()!=""` is the correct runtime gate).

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: origin-expansion | internal/webserver/origin_mw.go | New Funnel URL added to WebSocket origin allowlist; secondary branch is fail-closed (empty string excluded), exact byte match enforced (T-165-07: no prefix/substring widen) |

## Self-Check: PASSED

- `internal/webserver/funnel_client.go` — FOUND
- `internal/webserver/funnel_test.go` — FOUND
- `1312fd0e` (RED commit) — FOUND
- `38442225` (GREEN commit) — FOUND
- All 7 new funnel tests pass
- TESTING.md Go count = 367, Total = 510 — VERIFIED
- `bash tests/check-traceability-paths.sh` — OK
