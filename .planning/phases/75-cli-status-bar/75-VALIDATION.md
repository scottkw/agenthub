---
phase: 75
slug: cli-status-bar
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-14
---

# Phase 75 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test toolchain |
| **Quick run command** | `go test ./internal/statusbar/...` |
| **Full suite command** | `go test ./internal/statusbar/... ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/statusbar/...`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 75-01-01 | 01 | 1 | SB-04 | T-75-01 | MsgMeta server-only | unit | `go test ./internal/relay/ -run TestMakeMeta` | internal/relay/protocol_test.go | ⬜ pending |
| 75-01-02 | 01 | 1 | SB-04 | T-75-02 | BroadcastMeta non-blocking | unit | `go test ./internal/relay/ -run TestBroadcastMeta` | internal/relay/hub_test.go | ⬜ pending |
| 75-02-01 | 02 | 1 | SB-01 | T-75-03 | sanitize strips control chars | unit | `go test ./internal/statusbar/ -run TestBar_FormatContainsRequiredFields` | internal/statusbar/bar_test.go | ⬜ pending |
| 75-02-02 | 02 | 1 | SB-02 | — | N/A | unit | `go test ./internal/statusbar/ -run TestBar_ScrollRegionSetOnStart` | internal/statusbar/bar_test.go | ⬜ pending |
| 75-02-03 | 02 | 1 | SB-06 | — | N/A | unit | `go test ./internal/statusbar/ -run TestBar_TopPosition` | internal/statusbar/bar_test.go | ⬜ pending |
| 75-02-04 | 02 | 1 | SB-07 | T-75-04 | Stop idempotent via Once | unit | `go test ./internal/statusbar/ -run TestBar_StopClearsBarAndResetsScrollRegion` | internal/statusbar/bar_test.go | ⬜ pending |
| 75-03-01 | 03 | 2 | SB-03, SB-04 | T-75-05 | MsgMeta from trusted relay | unit | `go test -run TestWsOutputPump_MsgMeta -count=1 -timeout 30s` | cmd_attach_test.go | ⬜ pending |
| 75-03-02 | 03 | 2 | SB-05 | T-75-07 | Watcher exits on ctx.Done | unit | `go test -run TestLockedWriter_ConcurrentWrites -count=1 -timeout 30s` | cmd_attach_test.go | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/relay/protocol_test.go` — existing file, add TestMakeMeta tests (Plan 01 Task 3)
- [ ] `internal/relay/hub_test.go` — existing file, add TestBroadcastMeta test (Plan 01 Task 3)
- [ ] `internal/statusbar/bar_test.go` — new file with 9 tests (Plan 02 Task 2)
- [ ] `cmd_attach_test.go` — existing file, add MsgMeta and lockedWriter tests (Plan 03 Tasks 2-3)

*Existing Go test infrastructure covers framework needs — no new tooling required.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual scroll region correctness | SB-02 | Terminal rendering can only be verified visually | 1. Run `agenthub attach <session>` 2. Generate output exceeding terminal height 3. Verify status bar stays fixed at bottom, output scrolls normally |
| Clean terminal restore on detach | SB-07 | Terminal state restoration requires visual inspection | 1. Attach to session 2. Press Ctrl-\\ 3. Verify no leftover bar artifacts, cursor at correct position |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
