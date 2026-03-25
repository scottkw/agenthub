---
phase: 27
slug: unified-entrypoint
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-25
---

# Phase 27 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test infrastructure |
| **Quick run command** | `go test -count=1 -run TestCmd ./...` |
| **Full suite command** | `go test -count=1 ./...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -count=1 -run TestCmd ./...`
- **After every plan wave:** Run `go test -count=1 ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 27-01-01 | 01 | 1 | ROUTE-01 | unit | `go test -run TestDispatchNoArgs` | ❌ W0 | ⬜ pending |
| 27-01-02 | 01 | 1 | ROUTE-02 | unit | `go test -run TestDispatchCLI` | ❌ W0 | ⬜ pending |
| 27-01-03 | 01 | 1 | ROUTE-03 | unit | `go test -run TestDispatchDaemon` | ❌ W0 | ⬜ pending |
| 27-01-04 | 01 | 1 | CLI-01 | unit | `go test -run TestCmd` | ✅ | ⬜ pending |
| 27-01-05 | 01 | 1 | CLI-02 | unit | `go test -run TestCmdDaemon` | ✅ | ⬜ pending |
| 27-01-06 | 01 | 1 | CLI-03 | unit | `go test -run TestCmdAttach` | ✅ | ⬜ pending |
| 27-01-07 | 01 | 1 | CLI-04 | unit | `go test -run TestHelp` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Dispatch routing tests (TestDispatchNoArgs, TestDispatchCLI, TestDispatchDaemon) — stubs for ROUTE-01, ROUTE-02, ROUTE-03
- [ ] Help output test (TestHelp) — stub for CLI-04
- [ ] Existing CLI/daemon/attach tests migrate from `cmd/agenthub-cli/` to root package

*Existing test infrastructure covers CLI-01, CLI-02, CLI-03 requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GUI launches on no-args | ROUTE-01 | Requires windowing system | Run `agenthub` on desktop, confirm Wails window opens |
| PTY attach with resize | CLI-03 | Requires active agent + terminal | Run `agenthub attach <id>`, resize terminal, confirm output reflows |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
