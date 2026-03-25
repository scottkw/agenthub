---
phase: 22
slug: cli-attach
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-24
---

# Phase 22 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./cmd/agenthub-cli/ -run TestCmdAttach -v -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/agenthub-cli/ -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 22-01-01 | 01 | 1 | CLI-05 | unit | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_MissingArgs -v` | ❌ W0 | ⬜ pending |
| 22-01-02 | 01 | 1 | CLI-05 | integration | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_ReceivesOutput -v` | ❌ W0 | ⬜ pending |
| 22-01-03 | 01 | 1 | CLI-06 | unit | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_CtrlCPassthrough -v` | ❌ W0 | ⬜ pending |
| 22-01-04 | 01 | 1 | CLI-06 | unit | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_ResizeForwarded -v` | ❌ W0 | ⬜ pending |
| 22-01-05 | 01 | 1 | CLI-07 | unit | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_DetachKey -v` | ❌ W0 | ⬜ pending |
| 22-01-06 | 01 | 1 | CLI-08 | integration | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_ScrollbackReplay -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/agenthub-cli/cmd_attach_test.go` — test stubs for CLI-05, CLI-06, CLI-07, CLI-08
- [ ] `go get golang.org/x/term` — promote from indirect to direct dep in go.mod

*Existing `go test` infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Terminal restore after SIGHUP | CLI-07 | Requires actual terminal + signal delivery | 1. `agenthub attach <id>` 2. `kill -HUP <attach-pid>` 3. Verify terminal echo/cursor restored |
| Visual corruption on resize | CLI-06 | Requires visual terminal inspection | 1. Attach to session 2. Resize terminal window 3. Verify no garbled output |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
