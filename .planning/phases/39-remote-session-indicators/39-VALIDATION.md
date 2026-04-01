---
phase: 39
slug: remote-session-indicators
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-01
---

# Phase 39 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (backend), vitest (frontend — N/A for terminal.html) |
| **Config file** | go.mod (Go), frontend/vitest.config.ts (Vitest) |
| **Quick run command** | `go test ./internal/webserver/... ./internal/daemon/... && go test -run TestCmdAttach -v .` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/... ./internal/daemon/... && go test -run TestCmdAttach -v .`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 39-01-01 | 01 | 1 | RMTE-01 | unit | `go test ./internal/webserver/ -run TestSessionInfo` | ❌ W0 | ⬜ pending |
| 39-01-02 | 01 | 1 | RMTE-01 | integration | `go test ./internal/webserver/ -run TestTerminalSessionInfo` | ❌ W0 | ⬜ pending |
| 39-02-01 | 02 | 1 | RMTE-02 | unit | `go test -run TestCmdAttach_Banner -v .` | ❌ W0 | ⬜ pending |
| 39-02-02 | 02 | 1 | RMTE-02 | unit | `go test -run TestCmdAttach_Detach -v .` | ✅ exists | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Tests for `handleSessionInfo` endpoint in `internal/webserver/server_test.go`
- [ ] Tests for CLI attach banner output in `cmd_attach_test.go`

*Existing test infrastructure covers framework needs — no new framework install required.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Web status bar visible with metadata | RMTE-01 | Requires browser + running web server | Build, start web server, open session in browser, verify status bar shows name/agent/hostname |
| Status bar updates within 3s on disconnect | RMTE-01 | Requires daemon kill during browser session | Open web terminal, kill daemon, observe status bar change within 3 seconds |
| Terminal viewport fills correctly with status bar | RMTE-01 | Visual verification of terminal dimensions | Open web terminal, verify no gap or overflow with status bar present |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
