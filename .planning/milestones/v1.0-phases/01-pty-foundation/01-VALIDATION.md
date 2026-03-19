---
phase: 1
slug: pty-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-17
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib) |
| **Config file** | none — Wave 0 installs |
| **Quick run command** | `go test ./internal/pty/... -v -timeout 30s` |
| **Full suite command** | `go test ./... -v -timeout 60s` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/pty/... -v -timeout 30s`
- **After every plan wave:** Run `go test ./... -v -timeout 60s`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 0 | CLI-01 | unit | `go test ./internal/pty/... -run TestDetectCLIs -v` | ❌ W0 | ⬜ pending |
| 1-01-02 | 01 | 0 | CLI-02 | integration | `go test ./internal/pty/... -run TestSpawnPTY -v` | ❌ W0 | ⬜ pending |
| 1-01-03 | 01 | 0 | TERM-06 | unit | `go test ./internal/pty/... -run TestResize -v` | ❌ W0 | ⬜ pending |
| 1-01-04 | 01 | 0 | TERM-07 | integration | `go test ./internal/pty/... -run TestKillClean -v` | ❌ W0 | ⬜ pending |
| 1-01-05 | 01 | 0 | SESS-01 | unit | `go test ./internal/pty/... -run TestSessionPersist -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `go.mod` — module definition (`go mod init`)
- [ ] `internal/pty/backend.go` — SessionBackend interface definition
- [ ] `internal/pty/native.go` — NativePTYBackend implementation
- [ ] `internal/pty/session.go` — Session struct
- [ ] `internal/pty/registry.go` — SessionRegistry with RWMutex
- [ ] `internal/pty/detect.go` — CLI detection via exec.LookPath
- [ ] `internal/pty/cleanup.go` — graceful shutdown + process group kill
- [ ] `internal/pty/win32input.go` — win32-input-mode parser (build tag: windows)
- [ ] `internal/pty/win32input_other.go` — no-op stub (build tag: !windows)
- [ ] `internal/pty/detect_test.go` — TestDetectCLIs
- [ ] `internal/pty/native_test.go` — TestSpawnPTY, TestResize, TestKillClean, TestSessionPersist
- [ ] `cmd/agenthub/main.go` — minimal main for manual smoke test
- [ ] Framework install: none needed — Go stdlib `testing` package

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real CLI interactive TUI renders correctly | CLI-02 | Requires actual Claude Code/Gemini CLI installed | Run `cmd/agenthub/main.go`, spawn a real CLI, verify TUI renders |
| Windows ConPTY resize race | TERM-06 | Requires Windows + ConPTY timing | On Windows, resize immediately after spawn, verify dimensions propagate |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
