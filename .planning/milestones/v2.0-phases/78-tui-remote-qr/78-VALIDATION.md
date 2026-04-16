---
phase: 78
slug: tui-remote-qr
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-15
---

# Phase 78 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — standard `_test.go` files |
| **Quick run command** | `go test ./internal/tui/... -run {TestName} -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30 seconds for package tests; ~90 seconds full suite |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/tui/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

> Populated by the planner from PLAN.md task list. Planner MUST emit an `<automated>` test command for every non-glue task. Rows below are placeholders to be replaced when PLAN.md is written.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 78-01-01 | 01 | 0 | TUI-07 | — | N/A | unit | `go test ./internal/tui/... -run "TestUpdate_RemoteSessionsMsg\|TestUpdate_UnifiedListEmpty" -count=1` | ✅ | ✅ green |
| 78-01-02 | 01 | 1 | TUI-07 | — | N/A | unit | `go test ./internal/tui/... -run "TestUpdate_RemoteSessionsMsg\|TestView_SessionListWithRemotes" -count=1` | ✅ | ✅ green |
| 78-01-03 | 01 | 1 | TUI-07 | — | N/A | unit | `go test ./internal/tui/... -run TestUpdate_NavigationSkipsDividers -count=1` | ✅ | ✅ green |
| 78-01-04 | 01 | 1 | TUI-07 | — | N/A | unit | `go test ./internal/tui/... -run TestUpdate_SelectionRestoredAfterRebuild -count=1` | ✅ | ✅ green |
| 78-02-01 | 02 | 1 | TUI-10 | — | N/A | unit | `go test ./internal/tui/... -run TestView_QROverlayContent -count=1` | ✅ | ✅ green |
| 78-02-02 | 02 | 1 | TUI-10 | — | N/A | unit | `go test ./internal/tui/... -run TestView_QROverlayContent -count=1` | ✅ | ✅ green |
| 78-02-03 | 02 | 1 | TUI-10 | — | N/A | unit | `go test ./internal/tui/... -run "TestUpdate_QROpen\|TestUpdate_QRClose\|TestUpdate_QRSwallowsKeys" -count=1` | ✅ | ✅ green |
| 78-03-01 | 03 | 2 | TUI-07, TUI-10 | — | N/A | integration | `go test ./internal/tui/... -run TestTUIRemoteAndQR -count=1` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Per-plan task rows are finalized by the planner. Checker enforces that every task cites an `<automated>` command or a Wave 0 dependency.*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

- [x] `internal/tui/update_test.go` — unit tests for remote sessions, unified list, navigation, QR overlay (TUI-07, TUI-10)
- [x] `internal/tui/view_test.go` — view rendering tests for dividers, remote rows, QR overlay, header counts (TUI-07, TUI-10)
- [x] `internal/tui/integration_test.go` — end-to-end integration tests for remote+QR flow (TUI-07, TUI-10)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| QR readability with a real phone camera | TUI-10 | End-to-end usability check — automated decode proves encoding correctness but not visual scan-ability in diverse terminals | Open TUI, select a session, press `q`, scan with iOS/Android camera in Terminal.app, iTerm2, Alacritty, and kitty; URL must decode and open in a browser |
| Remote peer parity vs GUI | TUI-07 | Cross-UI visual diff — ensures grouping/ordering matches the GUI Remote Sessions panel | With a live tailnet, open both GUI Remote Sessions and TUI side-by-side; confirm same peers, same groups, same per-session status |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-16

---

## Validation Audit 2026-04-16

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

### Audit Notes

All 8 task verification entries had stale placeholder test names from pre-execution planning (e.g., `TestRemoteSessions` instead of `TestUpdate_RemoteSessionsMsg`). Behavioral coverage was already 100% — the executor created tests with different names that fully cover all requirements. Updated `Automated Command` column to reference actual test names. 78 tests passing across 5 test files.

### Coverage Summary

| Requirement | Tests | Files |
|-------------|-------|-------|
| TUI-07 (remote sessions) | `TestUpdate_RemoteSessionsMsg`, `TestUpdate_NavigationSkipsDividers`, `TestUpdate_SelectionRestoredAfterRebuild`, `TestUpdate_KillRemoteBlocked`, `TestUpdate_RenameRemoteBlocked`, `TestUpdate_UnifiedListEmpty`, `TestView_HeaderRemoteCount`, `TestView_DividerRow`, `TestView_RemoteSessionRow`, `TestView_SessionListWithRemotes` | update_test.go, view_test.go |
| TUI-10 (QR code) | `TestUpdate_QROpen`, `TestUpdate_QRClose`, `TestUpdate_QRNoURL`, `TestUpdate_QRTerminalTooSmall`, `TestUpdate_QRSwallowsKeys`, `TestUpdate_QRQuitFromOverlay`, `TestUpdate_QuitKeyReassignment`, `TestView_QROverlayContent`, `TestView_HintBar`, `TestHelp_QRBinding` | update_test.go, view_test.go, help_test.go |
| TUI-07 + TUI-10 (integration) | `TestTUIRemoteAndQR_FullFlow`, `TestTUIRemoteAndQR_BlockedOperations` | integration_test.go |
