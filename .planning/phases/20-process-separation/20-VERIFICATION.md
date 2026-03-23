---
phase: 20-process-separation
verified: 2026-03-23T17:00:00Z
status: passed
score: 13/13 must-haves verified
re_verification: false
---

# Phase 20: Process Separation Verification Report

**Phase Goal:** Separate the daemon from the GUI — RunDaemon entry point, EnsureDaemon auto-start, platform-specific process detach, and App refactored to thin DaemonClient shell.
**Verified:** 2026-03-23
**Status:** PASSED
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | RunDaemon() starts SessionEngine + API + relay server and shuts down cleanly on SIGTERM | VERIFIED | `process.go` lines 15-48: creates engine, calls api.StartRelay(), api.Start(), blocks on signal.NotifyContext, calls api.Stop() + engine.Manager().Shutdown() |
| 2 | EnsureDaemon() returns nil immediately when daemon is already running | VERIFIED | `process.go` lines 53-64: calls client.Health(); if healthy AND relay port > 0, returns nil |
| 3 | EnsureDaemon() spawns a detached daemon subprocess and polls until health OK | VERIFIED | `process.go` lines 66-85: calls startDetachedDaemon(exe), polls health + relay port every 50ms |
| 4 | EnsureDaemon() returns an error if daemon does not start within 3 seconds | VERIFIED | `process.go` line 85: `return fmt.Errorf("EnsureDaemon: daemon did not become ready within 3s")` |
| 5 | Daemon API serves relay port via GET /relay-port | VERIFIED | `api.go` line 49: `a.mux.HandleFunc("GET /relay-port", a.handleRelayPort)` — handler returns `RelayPortResponse{Port: a.relayPort}` |
| 6 | Daemon API serves web server lifecycle via POST /webserver/start, POST /webserver/stop, GET /webserver/status, POST /sessions/{id}/web-serve | VERIFIED | `api.go` lines 50-53: all 4 routes registered with substantive handlers |
| 7 | DaemonClient has typed methods GetRelayPort, StartWebServer, StopWebServer, GetWebServerStatus, ToggleWebServing | VERIFIED | `client.go` lines 98-133: all 5 methods present and implemented |
| 8 | main.go dispatches to daemon.RunDaemon() when os.Args[1] == "daemon" | VERIFIED | `main.go` lines 21-24: `if len(os.Args) > 1 && os.Args[1] == "daemon" { daemon.RunDaemon(); return }` |
| 9 | App struct holds exactly one daemon communication field: *daemon.DaemonClient | VERIFIED | `app.go` lines 31-35: struct has ctx, client *daemon.DaemonClient, trayInit — no engine, api, server, listener, webServer, mu, socketPath fields |
| 10 | App.startup() calls EnsureDaemon then wires DaemonClient — no in-process engine or API | VERIFIED | `app.go` lines 43-58: calls daemon.EnsureDaemon(socketPath), then daemon.NewDaemonClient(socketPath) |
| 11 | App.CreateSession delegates through DaemonClient (no direct engine call) | VERIFIED | `app.go` lines 88-96: `a.client.CreateSession(cli, name, workDir)` |
| 12 | App.GetRelayPort fetches port from daemon via client.GetRelayPort() | VERIFIED | `app.go` lines 183-189: `a.client.GetRelayPort()` |
| 13 | Frontend shows error banner when daemon connection fails | VERIFIED | `frontend/src/App.tsx` lines 60, 97-103, 246-333: daemonError state set in catch block; banner renders "Unable to connect to session daemon" with Retry Connection button |

**Score:** 13/13 truths verified

---

## Required Artifacts

### Plan 01 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/daemon/process.go` | RunDaemon entry point, EnsureDaemon helper | VERIFIED | Exports `RunDaemon()` and `EnsureDaemon(socketPath string) error`; substantive implementations (86 lines) |
| `internal/daemon/process_unix.go` | Unix-specific startDetachedDaemon with Setsid | VERIFIED | Build tag `!windows`, `Setsid: true`, `cmd.Process.Release()` |
| `internal/daemon/process_windows.go` | Windows-specific startDetachedDaemon with CREATE_NEW_PROCESS_GROUP | VERIFIED | Build tag `windows`, `CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP` |
| `internal/daemon/types.go` | RelayPortResponse, WebServerStartRequest, WebServerStatusResponse types | VERIFIED | All 5 new types present (RelayPortResponse, WebServerStartRequest, WebServerStartResponse, WebServerStatusResponse, WebServeRequest) |
| `internal/daemon/api.go` | New HTTP routes for relay port and web server | VERIFIED | 5 new routes registered; StartRelay(), handleRelayPort(), handleWebServerStart(), handleWebServerStop(), handleWebServerStatus(), handleWebServe() all implemented |
| `internal/daemon/client.go` | New client methods for relay port and web server | VERIFIED | GetRelayPort, StartWebServer, StopWebServer, GetWebServerStatus, ToggleWebServing all implemented |

### Plan 02 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `main.go` | Daemon subcommand dispatch | VERIFIED | `daemon.RunDaemon()` called on `os.Args[1] == "daemon"` |
| `app.go` | Thin DaemonClient-only App struct | VERIFIED | 3-field struct (ctx, client, trayInit); all ops delegate through `a.client.*`; shutdown() has no daemon teardown |
| `app_test.go` | Updated tests using out-of-process daemon pattern | VERIFIED | `testApp` function starts in-process daemon API, wires DaemonClient |
| `frontend/src/App.tsx` | Daemon error banner UI | VERIFIED | "Unable to connect to session daemon" text present; retryInit callback; daemonError state |

---

## Key Link Verification

### Plan 01 Key Links

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/daemon/process.go` | `internal/daemon/api.go` | RunDaemon creates API and calls api.Start | WIRED | `api := NewAPI(engine)` + `api.StartRelay()` + `api.Start(socketPath)` in RunDaemon() |
| `internal/daemon/process.go` | `internal/daemon/client.go` | EnsureDaemon probes health via NewDaemonClient | WIRED | `client := NewDaemonClient(socketPath); client.Health()` in EnsureDaemon() |
| `internal/daemon/api.go` | `internal/relay/server.go` | API creates relay.Server from engine and starts TCP listener | WIRED | `server := relay.NewServer(a.engine.Manager(), a.engine.Backend())` in StartRelay() |

### Plan 02 Key Links

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `main.go` | `internal/daemon/process.go` | os.Args[1]=="daemon" dispatches to RunDaemon | WIRED | `daemon.RunDaemon()` on line 22 |
| `app.go` | `internal/daemon/process.go` | startup calls EnsureDaemon before wiring client | WIRED | `daemon.EnsureDaemon(socketPath)` on line 47 |
| `app.go` | `internal/daemon/client.go` | All session ops delegate through a.client | WIRED | 11 methods confirmed: CreateSession, ListSessions, RenameSession, KillSession, GetSessionStatus, GetRelayPort, UpdateCLIPath, StartWebServer, StopWebServer, GetWebServerStatus, ToggleWebServing |
| `frontend/src/App.tsx` | `app.go` | init() calls GetRelayPort which goes through daemon | WIRED | `GetRelayPort()` called in init() and retryInit(); wired to `a.client.GetRelayPort()` in app.go |

---

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| DAEMON-01 | 20-01, 20-02 | Session management runs in a standalone daemon process separate from the GUI | SATISFIED | RunDaemon() is an independent process entry point; App struct has no in-process engine/API |
| DAEMON-03 | 20-02 | Sessions persist when all clients (GUI and CLI) disconnect | SATISFIED | shutdown() contains no daemon teardown (comment: "Daemon is an independent process — GUI does NOT stop it. Sessions persist after GUI exits (DAEMON-03)") |
| DAEMON-04 | 20-02 | GUI app connects to the daemon as a client; GUI and CLI see the same session pool | SATISFIED | App.startup() wires DaemonClient to existing daemon socket; ListSessions() fetches from daemon |
| DAEMON-05 | 20-01, 20-02 | Daemon auto-starts when any CLI command is run and no daemon is running | SATISFIED | EnsureDaemon() spawns detached subprocess when health check fails; called from App.startup() |

All 4 requirement IDs from PLAN frontmatter are accounted for. REQUIREMENTS.md confirms all 4 are marked Phase 20 / Complete.

---

## Anti-Patterns Found

Scanning key modified files for stubs, placeholders, empty implementations.

| File | Pattern | Severity | Assessment |
|------|---------|----------|------------|
| `app.go` line 14 | `"github.com/agenthub/agenthub/internal/webserver"` imported | Info | Used only for `webserver.TailscaleHealth` return type and `webserver.CheckHealth()` — Tailscale health check is legitimate GUI responsibility, not a daemon delegation concern |
| None | No TODO/FIXME/placeholder patterns found | — | Clean |
| None | No empty return stubs found | — | Clean |
| None | No console.log-only implementations | — | Clean |

The `webserver` import in `app.go` is not a violation of process separation. The `GetTailscaleStatus()` / `startHealthPoller()` methods directly call the Tailscale API for display purposes; they do not manage the web server lifecycle (that is fully delegated). This is expected behavior.

---

## Human Verification Required

### 1. Sessions survive GUI close and reopen

**Test:** Run `wails dev`, create a session, close the GUI window (or hide to tray), reopen it, verify the same session appears with its state intact.
**Expected:** Session is visible with unchanged state after GUI reopen.
**Why human:** Cannot automate Wails window lifecycle in tests; requires actual process lifecycle with `DAEMON-03` persistence semantics.

**Note:** The SUMMARY documents this was verified in Task 3 of Plan 02 — all 10 steps passed, including close/reopen session survival and daemon auto-restart after manual kill. This is recorded as human-approved in the summary.

---

## Test Results

All automated tests pass:

```
ok  github.com/agenthub/agenthub              2.172s
ok  github.com/agenthub/agenthub/internal/daemon  1.992s
ok  github.com/agenthub/agenthub/internal/pty     1.800s
ok  github.com/agenthub/agenthub/internal/relay   3.025s
ok  github.com/agenthub/agenthub/internal/status  1.955s
ok  github.com/agenthub/agenthub/internal/webserver 2.199s
```

`go build ./...` — exits 0
`GOOS=windows go vet ./internal/daemon/...` — exits 0

---

## Summary

Phase 20 goal is fully achieved. All structural requirements of process separation are present in the codebase:

1. `RunDaemon()` is a real entry point that creates engine + relay + API, blocks on signal, and shuts down cleanly.
2. `EnsureDaemon()` probes health (including relay readiness), spawns a detached subprocess when not running, and times out after 3 seconds with an error.
3. Platform-specific detach is implemented correctly: Unix uses `Setsid: true`, Windows uses `CREATE_NEW_PROCESS_GROUP`.
4. `App` struct is a thin shell with exactly 3 fields (ctx, client, trayInit) — no engine, API, relay, web server, or listener fields remain.
5. `shutdown()` contains no daemon teardown — sessions survive GUI close by design.
6. All 5 new API routes and 5 new DaemonClient methods are substantively implemented and wired.
7. Frontend error banner is in place with retry mechanism.
8. All 4 requirement IDs (DAEMON-01, DAEMON-03, DAEMON-04, DAEMON-05) are satisfied.
9. All tests pass with `-race`; Windows cross-compilation vets clean.

---

_Verified: 2026-03-23_
_Verifier: Claude (gsd-verifier)_
