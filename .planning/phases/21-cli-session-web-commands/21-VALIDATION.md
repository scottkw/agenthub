---
phase: 21
slug: cli-session-web-commands
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-23
---

# Phase 21 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib) |
| **Config file** | none — `go test ./...` discovers all `*_test.go` files |
| **Quick run command** | `go test ./cmd/agenthub-cli/... ./internal/daemon/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -count=1 -timeout 60s` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/agenthub-cli/... ./internal/daemon/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -count=1 -timeout 60s`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 21-01-01 | 01 | 1 | CLI-01 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdNew -v` | ❌ W0 | ⬜ pending |
| 21-01-02 | 01 | 1 | CLI-02 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdList -v` | ❌ W0 | ⬜ pending |
| 21-01-03 | 01 | 1 | CLI-03 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdKill -v` | ❌ W0 | ⬜ pending |
| 21-01-04 | 01 | 1 | CLI-04 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdRename -v` | ❌ W0 | ⬜ pending |
| 21-02-01 | 02 | 1 | WEB-01 | unit (mock) | `go test ./cmd/agenthub-cli/... -run TestCmdWebStart -v` | ❌ W0 | ⬜ pending |
| 21-02-02 | 02 | 1 | WEB-02 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdWebStatus -v` | ❌ W0 | ⬜ pending |
| 21-02-03 | 02 | 1 | WEB-03 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdServe -v` | ❌ W0 | ⬜ pending |
| 21-02-04 | 02 | 1 | WEB-04 | unit (mock) | `go test ./cmd/agenthub-cli/... -run TestCmdHealth -v` | ❌ W0 | ⬜ pending |
| 21-02-05 | 02 | 1 | WEB-05 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdQR -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/agenthub-cli/main.go` — CLI entry point (new file)
- [ ] `cmd/agenthub-cli/main_test.go` — test stubs for CLI-01 through WEB-05
- [ ] Test helper that starts `testDaemon(t)` from CLI package scope (or import from `internal/daemon` testhelper)

*Existing `internal/daemon` test infrastructure is complete and passing. The gaps are only the new CLI package.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GUI tab bar reflects `agenthub rename` | CLI-03 | Requires GUI rendering | 1. Run `agenthub rename <id> newname` 2. Check GUI tab bar shows "newname" |
| Session appears in both CLI list and GUI | CLI-01 | Requires GUI rendering | 1. Run `agenthub new claude .` 2. Check `agenthub list` shows session 3. Check GUI shows session |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
