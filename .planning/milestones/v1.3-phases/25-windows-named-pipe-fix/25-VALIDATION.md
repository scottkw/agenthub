---
phase: 25
slug: windows-named-pipe-fix
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-24
---

# Phase 25 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test toolchain |
| **Quick run command** | `go test ./internal/daemon/ -run TestSocket -count=1` |
| **Full suite command** | `go test ./internal/daemon/ -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/ -run TestSocket -count=1`
- **After every plan wave:** Run `go test ./internal/daemon/ -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 25-01-01 | 01 | 1 | DAEMON-05 | unit | `go test ./internal/daemon/ -run TestIsWindowsNamedPipe -count=1` | ❌ W0 | ⬜ pending |
| 25-01-02 | 01 | 1 | DAEMON-05 | unit | `go test ./internal/daemon/ -run TestCleanupStaleSocket -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/socket_windows_test.go` — build-tagged test stubs for Windows named pipe dial
- [ ] Existing `internal/daemon/socket_test.go` — verify Unix tests still pass unmodified

*Existing infrastructure covers Go test framework; only new test files needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Windows named pipe cleanup in production | DAEMON-05 | Requires Windows OS with running daemon | Start daemon on Windows, verify no duplicate spawn on reconnect |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
