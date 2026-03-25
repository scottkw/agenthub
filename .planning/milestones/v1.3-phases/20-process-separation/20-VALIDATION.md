---
phase: 20
slug: process-separation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-23
---

# Phase 20 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (go1.26.1) |
| **Config file** | None — standard `go test` |
| **Quick run command** | `go test -race ./internal/daemon/... .` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./internal/daemon/... .`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 20-01-01 | 01 | 1 | DAEMON-01 | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemon` | ❌ W0 | ⬜ pending |
| 20-01-02 | 01 | 1 | DAEMON-01 | integration | `go test -race ./internal/daemon/... -run TestRunDaemon` | ❌ W0 | ⬜ pending |
| 20-01-03 | 01 | 1 | DAEMON-03 | unit | `go test -race . -run TestShutdownSessionSurvive` | ❌ W0 | ⬜ pending |
| 20-01-04 | 01 | 1 | DAEMON-03 | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemonAlreadyRunning` | ❌ W0 | ⬜ pending |
| 20-01-05 | 01 | 1 | DAEMON-04 | integration | `go test -race . -run TestListSessions` | ✅ existing | ⬜ pending |
| 20-01-06 | 01 | 1 | DAEMON-04 | unit | `go test -race . -run TestAppStructFields` | ❌ W0 | ⬜ pending |
| 20-01-07 | 01 | 1 | DAEMON-05 | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemon_NoSocket` | ❌ W0 | ⬜ pending |
| 20-01-08 | 01 | 1 | DAEMON-05 | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemon_AlreadyRunning` | ❌ W0 | ⬜ pending |
| 20-01-09 | 01 | 1 | DAEMON-05 | unit | `go test -race ./internal/daemon/... -run TestEnsureDaemon_Timeout` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/process.go` — EnsureDaemon, startDetachedDaemon, RunDaemon
- [ ] `internal/daemon/process_unix.go` — SysProcAttr Setsid (build tag `!windows`)
- [ ] `internal/daemon/process_windows.go` — SysProcAttr CREATE_NEW_PROCESS_GROUP (build tag `windows`)
- [ ] `internal/daemon/process_test.go` — TestEnsureDaemon_*, TestRunDaemon
- [ ] `internal/daemon/types.go` additions — RelayPortResponse, WebServerStartRequest, WebServerStatusResponse
- [ ] `internal/daemon/api.go` additions — relay port route, webserver routes
- [ ] `internal/daemon/client.go` additions — GetRelayPort, StartWebServer, StopWebServer, GetWebServerStatus, ToggleWebServing

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Sessions survive GUI window close | DAEMON-03 | Requires Wails GUI lifecycle | 1. Start app, create session 2. Close window 3. Run `agenthub list` — session must still appear |
| GUI reconnects to daemon on reopen | DAEMON-03 | Requires Wails GUI lifecycle | 1. Close GUI window 2. Reopen app 3. Previous sessions must be visible |
| GUI relay port works after reconnect | DAEMON-01 | Requires full Wails+relay stack | 1. Start app 2. Close/reopen 3. Verify relay connection in browser dev tools |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
