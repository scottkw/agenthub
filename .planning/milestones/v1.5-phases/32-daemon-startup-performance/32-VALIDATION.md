---
phase: 32
slug: daemon-startup-performance
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-26
---

# Phase 32 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test tooling |
| **Quick run command** | `go test ./internal/daemon/... -run TestPoll -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/... -run TestPoll -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 32-01-01 | 01 | 1 | PERF-01 | unit | `go test ./internal/daemon/... -run TestPollImmediate -count=1` | ❌ W0 | ⬜ pending |
| 32-01-02 | 01 | 1 | PERF-02 | unit | `go test ./internal/daemon/... -run TestPollInterval -count=1` | ❌ W0 | ⬜ pending |
| 32-02-01 | 02 | 1 | PERF-03 | unit | `go test ./internal/daemon/... -run TestPathAugment -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/poll_test.go` — stubs for PERF-01, PERF-02 (pollSessionStatus timing)
- [ ] `internal/daemon/path_test.go` — stubs for PERF-03 (PATH augmentation)

*Existing Go test infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| launchd service PATH resolution | PERF-03 | Requires actual launchd environment | Install as service, verify agent discovery |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
