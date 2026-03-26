---
phase: 31
slug: cli-arg-passthrough
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-25
---

# Phase 31 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `go test . -run "TestCmdNew\|TestSplitDashDash" -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test . -run "TestCmdNew|TestSplitDashDash" -v`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 31-01-01 | 01 | 1 | ARGS-01 | unit | `go test . -run TestSplitDashDash -v` | ❌ W0 | ⬜ pending |
| 31-01-02 | 01 | 1 | ARGS-01 | unit | `go test . -run TestCmdNew_WithExtraArgs -v` | ❌ W0 | ⬜ pending |
| 31-01-03 | 01 | 1 | ARGS-01 | regression | `go test . -run TestCmdNew_NoSeparator -v` | ❌ W0 | ⬜ pending |
| 31-01-04 | 01 | 1 | ARGS-01 | regression | `go test ./...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd_cli_test.go` — add `TestCmdNew_WithExtraArgs` (non-nil extraArgs, expects session ID)
- [ ] `cmd_cli_test.go` — add `TestCmdNew_NoSeparator` (nil extraArgs, backward compat)
- [ ] `cmd_cli_test.go` — add `TestSplitDashDash` (boundary conditions: no `--`, `--` mid-slice, trailing `--`, leading `--`)

*(No new test files required — add functions to existing files)*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
