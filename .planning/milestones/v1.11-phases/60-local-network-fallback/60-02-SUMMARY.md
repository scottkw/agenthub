---
phase: 60-local-network-fallback
plan: "02"
subsystem: daemon
tags: [local-mode, password, wails-bindings, fallback]
dependency_graph:
  requires: [60-01]
  provides: [daemon-password-api, local-mode-fallback, wails-password-mode-bindings]
  affects: [app.go, internal/daemon]
tech_stack:
  added: [crypto/rand, encoding/base64]
  patterns: [mode-aware-web-server-start, password-once-per-daemon-lifetime, wails-binding-delegation]
key_files:
  created: []
  modified:
    - internal/daemon/types.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - internal/daemon/process.go
    - internal/daemon/api_test.go
    - app.go
    - cmd_cli.go
decisions:
  - "Password generated once per daemon lifetime using crypto/rand + base64url, stored in API struct"
  - "AutoStartWebServer now accepts mode+password; idempotency preserved"
  - "handleWebServerStart resolves LAN IP server-side when local mode and IP is empty"
  - "cmd_cli.go updated to pass mode/password (Rule 3 deviation — blocking compile fix)"
metrics:
  duration_minutes: 25
  completed_date: "2026-04-09"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 7
---

# Phase 60 Plan 02: Daemon Password Integration & Local Mode Fallback Summary

**One-liner:** Daemon generates a crypto/rand password at startup, falls back to local mode when Tailscale is absent, and exposes password/mode to the frontend via Wails bindings.

## What Was Built

Connected the Plan 01 webserver local mode infrastructure to the daemon lifecycle:

1. **Daemon types extended** (`internal/daemon/types.go`): `WebServerStartRequest` gained `Mode` and `Password` fields; `WebServerStatusResponse` gained `Mode`.

2. **API password storage + endpoint** (`internal/daemon/api.go`):
   - `API` struct has `localPassword string` field (mutex-protected)
   - `SetLocalPassword(pwd string)` method for runDaemonCore to call at startup
   - `GET /webserver/local-password` route + `handleGetLocalPassword` handler
   - `AutoStartWebServer` signature extended to `(ip string, port int, fqdn, mode, password string)`
   - `handleWebServerStart` passes `Mode`/`Password` to `webserver.Config`; resolves LAN IP server-side when `mode == "local" && ip == ""`
   - `handleWebServerStatus` returns `Mode` from the running `WebServer`

3. **Client method** (`internal/daemon/client.go`):
   - `StartWebServer` updated to 5-arg signature
   - `GetLocalNetworkPassword()` added — calls `GET /webserver/local-password`

4. **Password generation + local mode fallback** (`internal/daemon/process.go`):
   - `generateLocalPassword()` generates 16 bytes via `crypto/rand`, encodes as base64url (~22 chars)
   - `runDaemonCore` generates password before web server start, calls `api.SetLocalPassword`
   - Mode-aware auto-start: Tailscale connected → tailscale mode; else → `webserver.GetLANIP()` + local mode

5. **Wails bindings** (`app.go`):
   - `GetLocalNetworkPassword() string` — delegates to `client.GetLocalNetworkPassword()`
   - `GetWebServerMode() string` — delegates to `client.GetWebServerStatus().Mode`
   - `StartWebServer` updated to try Tailscale mode first, fall back to local mode

6. **Tests** (`internal/daemon/api_test.go`):
   - `TestGetLocalPassword` — verifies SetLocalPassword + endpoint returns correct value
   - `TestGetLocalPassword_TailscaleMode` — verifies empty password when not in local mode
   - `TestAutoStartWebServer_AlreadyRunning` updated to new 5-arg signature

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated cmd_cli.go to compile with new StartWebServer signature**
- **Found during:** Task 2 — `go build ./...` failed
- **Issue:** `cmdWebStart` in `cmd_cli.go` called `client.StartWebServer(h.IP, port, h.Domain)` with old 3-arg signature
- **Fix:** Updated call to `client.StartWebServer(h.IP, port, h.Domain, "tailscale", "")` — CLI always uses Tailscale mode (requires health check before calling)
- **Files modified:** `cmd_cli.go`
- **Commit:** 85b7ee0

## Verification Results

```
go build ./...                          EXIT 0
go test ./internal/daemon/... -v        60 tests PASS
go test ./internal/webserver/... -v     30 tests PASS
TestGetLocalPassword                    PASS
TestGetLocalPassword_TailscaleMode      PASS
TestRunDaemonCore_CancelledContext      PASS (log shows local mode auto-start)
```

## Known Stubs

None — all functionality is fully wired. Password is generated and stored; endpoint returns it; Wails bindings delegate to client methods.

## Threat Flags

None — no new network endpoints beyond the `GET /webserver/local-password` route which is on the Unix socket (local-only, not exposed over TCP).

## Commits

| Task | Commit | Message |
|------|--------|---------|
| Task 1 | e73e0cd | feat(60-02): extend daemon types, API, client with mode/password + local-password endpoint |
| Task 2 | 85b7ee0 | feat(60-02): password generation, local mode fallback in runDaemonCore, Wails bindings |

## Self-Check: PASSED

Files modified exist and are correctly updated. Commits e73e0cd and 85b7ee0 exist in git log.
