---
phase: 23
slug: service-manager-integration
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-24
---

# Phase 23 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `testing` (Go stdlib) + `go test` |
| **Config file** | none — standard Go test runner |
| **Quick run command** | `go test ./internal/daemon/ ./cmd/agenthub-cli/ -run TestSvc -timeout 30s` |
| **Full suite command** | `go test ./... -timeout 120s` |
| **Estimated runtime** | ~30 seconds (quick), ~120 seconds (full) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/ ./cmd/agenthub-cli/ -timeout 30s`
- **After every plan wave:** Run `go test ./... -timeout 120s`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 23-01-01 | 01 | 1 | SVC-01 | unit | `go test ./internal/daemon/ -run TestServiceConfig -timeout 30s` | ❌ W0 | ⬜ pending |
| 23-01-02 | 01 | 1 | SVC-01 | unit | `go test ./internal/daemon/ -run TestServiceControl_Install -timeout 30s` | ❌ W0 | ⬜ pending |
| 23-01-03 | 01 | 1 | SVC-01 | unit | `go test ./internal/daemon/ -run TestServiceControl_Uninstall -timeout 30s` | ❌ W0 | ⬜ pending |
| 23-02-01 | 02 | 1 | SVC-03 | unit | `go test ./cmd/agenthub-cli/ -run TestCmdDaemon -timeout 30s` | ❌ W0 | ⬜ pending |
| 23-02-02 | 02 | 1 | SVC-02 | unit | `go test ./internal/daemon/ -run TestServiceConfig_RunAtLoad -timeout 30s` | ❌ W0 | ⬜ pending |
| 23-03-01 | 03 | 2 | SVC-01/02/03 | manual | N/A — requires OS service manager | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/service_test.go` — unit tests for `newServiceConfig`, `ServiceControl` install/uninstall
- [ ] `internal/daemon/service_test.go` — unit tests for `daemonSvc` Start/Stop lifecycle
- [ ] `cmd/agenthub-cli/cmd_daemon_test.go` — unit tests for `cmdDaemon` dispatch

*Note: Tests should mock `service.Control()` via dependency injection on a `ServiceController` interface.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `agenthub daemon install` registers macOS LaunchAgent | SVC-01 | Requires macOS launchd | Run `agenthub daemon install`, verify `~/Library/LaunchAgents/agenthub-daemon.plist` exists |
| Daemon auto-starts on login | SVC-02 | Requires reboot/re-login | Install, log out/in, check daemon is running via health check |
| `agenthub daemon uninstall` removes registration | SVC-01 | Requires macOS launchd | Run `agenthub daemon uninstall`, verify plist removed |
| `agenthub daemon start/stop` controls service | SVC-03 | Requires service manager | Install, stop via CLI, verify daemon not responding, start, verify responding |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
