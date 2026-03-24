---
phase: 24
slug: cli-polish
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-24
---

# Phase 24 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib) |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `cd /Users/ken/dev/agenthub && go test ./cmd/agenthub-cli/... -count=1` |
| **Full suite command** | `cd /Users/ken/dev/agenthub && go test ./... -count=1` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/agenthub-cli/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 24-01-01 | 01 | 1 | POLISH-01 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdList_JSON -count=1` | ❌ W0 | ⬜ pending |
| 24-01-02 | 01 | 1 | POLISH-01 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdWebStatus_JSON -count=1` | ❌ W0 | ⬜ pending |
| 24-01-03 | 01 | 1 | POLISH-01 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdHealth_JSON -count=1` | ❌ W0 | ⬜ pending |
| 24-01-04 | 01 | 1 | POLISH-01 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdDaemon_Status -count=1` | ❌ W0 | ⬜ pending |
| 24-02-01 | 02 | 1 | POLISH-02 | unit | `go test ./cmd/agenthub-cli/... -run TestCmdSettings -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/agenthub-cli/main_test.go` — tests for `--json` flag on cmdList, cmdWebStatus, cmdHealth
- [ ] `cmd/agenthub-cli/main_test.go` — tests for cmdSettings
- [ ] `cmd/agenthub-cli/cmd_daemon_test.go` — tests for daemon status subcommand

*Existing test infrastructure covers framework needs — no new installs required.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| JSON output piped to jq | POLISH-01 | End-to-end requires running daemon | `agenthub list --json \| jq .` with daemon running |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
