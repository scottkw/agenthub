---
phase: 75
slug: cli-status-bar
status: final
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-14
audited: 2026-04-15
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
| 75-01-01 | 01 | 1 | SB-04 | T-75-01 | MsgMeta server-only | unit | `go test ./internal/relay/ -run TestMakeMeta` | internal/relay/protocol_test.go | ✅ green |
| 75-01-02 | 01 | 1 | SB-04 | T-75-02 | BroadcastMeta non-blocking | unit | `go test ./internal/relay/ -run TestBroadcastMeta` | internal/relay/hub_test.go | ✅ green |
| 75-02-01 | 02 | 1 | SB-01 | T-75-03 | sanitize strips control chars | unit | `go test ./internal/statusbar/ -run TestBar_FormatContainsRequiredFields` | internal/statusbar/bar_test.go | ✅ green |
| 75-02-01b | 02 | 1 | SB-01 | T-75-03 | sanitize strips control chars (explicit) | unit | `go test ./internal/statusbar/ -run TestBar_SanitizeSessionName` | internal/statusbar/bar_test.go | ✅ green |
| 75-02-02 | 02 | 1 | SB-02 | — | DECSTBM scroll region set | unit | `go test ./internal/statusbar/ -run TestBar_ScrollRegionSetOnStart` | internal/statusbar/bar_test.go | ✅ green |
| 75-02-03 | 02 | 1 | SB-06 | — | Top position DECSTBM | unit | `go test ./internal/statusbar/ -run TestBar_TopPosition` | internal/statusbar/bar_test.go | ✅ green |
| 75-02-04 | 02 | 1 | SB-07 | T-75-04 | Stop clears bar + resets scroll region | unit | `go test ./internal/statusbar/ -run TestBar_StopClearsBarAndResetsScrollRegion` | internal/statusbar/bar_test.go | ✅ green |
| 75-02-04b | 02 | 1 | SB-07 | T-75-04 | Stop idempotent via sync.Once | unit | `go test ./internal/statusbar/ -run TestBar_StopIdempotent` | internal/statusbar/bar_test.go | ✅ green |
| 75-02-05 | 02 | 1 | SB-05 | — | Connection state rendered in bar | unit | `go test ./internal/statusbar/ -run TestBar_ConnStateDisplay` | internal/statusbar/bar_test.go | ✅ green |
| 75-03-01 | 03 | 2 | SB-03, SB-04 | T-75-05 | MsgMeta parse + viewer count update | unit | `go test -run TestWsOutputPump_MsgMeta -count=1 -timeout 30s` | cmd_attach_test.go | ✅ green |
| 75-03-02 | 03 | 2 | — | T-75-06 | lockedWriter serialises concurrent writes | unit | `go test -run TestLockedWriter_ConcurrentWrites -count=1 -timeout 30s` | cmd_attach_test.go | ✅ green |
| 75-03-03 | 03 | 2 | — | — | Unknown frame types silently ignored | unit | `go test -run TestWsOutputPump_IgnoresUnknownFrameTypes -count=1 -timeout 30s` | cmd_attach_test.go | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/relay/protocol_test.go` — existing file, TestMakeMeta tests added (Plan 01 Task 3)
- [x] `internal/relay/hub_test.go` — existing file, TestBroadcastMeta test added (Plan 01 Task 3)
- [x] `internal/statusbar/bar_test.go` — new file with 9 tests (Plan 02 Task 2)
- [x] `cmd_attach_test.go` — existing file, MsgMeta and lockedWriter tests added (Plan 03 Tasks 2-3)

*Existing Go test infrastructure covers framework needs — no new tooling required.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual scroll region correctness | SB-02 | Terminal rendering can only be verified visually | 1. Run `agenthub attach <session>` 2. Generate output exceeding terminal height 3. Verify status bar stays fixed at bottom, output scrolls normally |
| Clean terminal restore on detach | SB-07 | Terminal state restoration requires visual inspection | 1. Attach to session 2. Press Ctrl-\\ 3. Verify no leftover bar artifacts, cursor at correct position |
| Live viewer count across two attach clients | SB-04 | Requires two concurrent attach sessions; current unit tests only verify frame parsing and setter thread-safety | 1. `agenthub attach <id>` in terminal A 2. `agenthub attach <id>` in terminal B 3. Confirm A's bar updates to "2 viewers" within 1s; drops back when B detaches |
| Non-TTY bar suppression end-to-end | SB-03 | Requires piping actual attach output and inspecting raw bytes | 1. `agenthub attach <id> \| cat > /tmp/out.txt` 2. `grep -c $'\\x1b\\[' /tmp/out.txt` 3. Confirm zero bar ANSI sequences (only PTY content) |
| Connection state watcher exits on ctx.Done | T-75-07 | Watcher is an inline anonymous goroutine in `cmdAttachRemoteWithClient` (cmd_attach.go:295). Testing the ctx.Done exit would require extracting it into an exported function — refactor cost exceeds risk for a standard 3-line Go idiom. Verified by code review. | 1. Read cmd_attach.go:294-311 2. Confirm `select { case <-tick.C: ... case <-sigCtx.Done(): return }` pattern present 3. Confirm `defer tick.Stop()` prevents ticker leak |
| Connection state watcher triggers after 5s silence | SB-05 | Same inline-goroutine constraint as T-75-07. Bar rendering of "[reconnecting]" state is covered by `TestBar_ConnStateDisplay`; the watcher→setter wiring is not unit-tested. | 1. `agenthub attach <remote-id>` 2. Simulate network partition (e.g., `sudo pfctl` block) for >5s 3. Confirm bar shows "[reconnecting]" 4. Restore connectivity 5. Confirm state clears on next frame |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter
- [x] All mapped automated tests exist and pass (`go test ./internal/statusbar/... ./internal/relay/...` + targeted cmd_attach_test.go runs)

**Approval:** validated 2026-04-15

---

## Validation Audit 2026-04-15

| Metric | Count |
|--------|-------|
| Rows in original map | 8 |
| Rows confirmed passing | 8 |
| Mapping corrections | 1 (75-03-02 retargeted from SB-05/T-75-07 → T-75-06) |
| Rows added to map | 4 (75-02-01b, 75-02-04b, 75-02-05, 75-03-03 — tests existed but were not indexed) |
| Gaps found | 1 (T-75-07 watcher ctx.Done exit) |
| Resolved (automated) | 0 |
| Escalated (manual-only) | 1 (T-75-07 — inline anonymous goroutine, refactor cost exceeds risk) |
| Additional manual-only rows added | 3 (SB-04 two-client live, SB-03 non-TTY piping, SB-05 watcher-trigger integration) |

**Audit verdict:** Phase 75 remains `nyquist_compliant: true`. All unit-testable behaviors are covered by passing automated tests. Remaining gaps are either (a) inherently visual (terminal rendering) or (b) inline goroutines whose extraction cost exceeds the risk of the standard Go context-cancellation idiom they embody. These are documented in Manual-Only with explicit verification steps.
