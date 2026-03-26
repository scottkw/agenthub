---
phase: 27
slug: unified-entrypoint
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-25
audited: 2026-03-25
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
| 27-01-01 | 01 | 1 | ROUTE-01 | unit | `go test -run TestDispatchNoArgs` | ✅ | ✅ green |
| 27-01-02 | 01 | 1 | ROUTE-02 | unit | `go test -run TestDispatchFlagRouting` | ✅ | ✅ green |
| 27-01-03 | 01 | 1 | ROUTE-03 | unit | `go test -run TestCmdDaemon_ServiceActions` | ✅ | ✅ green |
| 27-01-04 | 01 | 1 | CLI-01 | unit | `go test -run TestCmd` | ✅ | ✅ green |
| 27-01-05 | 01 | 1 | CLI-02 | unit | `go test -run TestCmdDaemon` | ✅ | ✅ green |
| 27-01-06 | 01 | 1 | CLI-03 | unit | `go test -run TestCmdAttach` | ✅ | ✅ green |
| 27-01-07 | 01 | 1 | CLI-04 | unit | `go test -run TestDispatchHelp` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Dispatch routing tests (TestDispatchNoArgs, TestDispatchFlagRouting) — ROUTE-01, ROUTE-02, ROUTE-03
- [x] Help output test (TestDispatchHelp, TestDispatchHelpShort) — CLI-04
- [x] Existing CLI/daemon/attach tests migrated from `cmd/agenthub-cli/` to root package

*All requirements have automated test coverage.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GUI launches on no-args | ROUTE-01 | Requires windowing system | Run `agenthub` on desktop, confirm Wails window opens |
| PTY attach with resize | CLI-03 | Requires active agent + terminal | Run `agenthub attach <id>`, resize terminal, confirm output reflows |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-03-25

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All 7 requirements (ROUTE-01, ROUTE-02, ROUTE-03, CLI-01, CLI-02, CLI-03, CLI-04) have automated test coverage in root package. Tests verified green with `go test -count=1 -run "TestDispatch|TestCmd|TestAttach|TestRunCLI" .`
