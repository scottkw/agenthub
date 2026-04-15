---
phase: 78
slug: tui-remote-qr
status: draft
nyquist_compliant: false
wave_0_complete: false
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
| 78-01-01 | 01 | 0 | TUI-07 | — | N/A | unit stub | `go test ./internal/tui/... -run TestRemoteSessions -count=1` | ❌ W0 | ⬜ pending |
| 78-01-02 | 01 | 1 | TUI-07 | — | N/A | unit | `go test ./internal/tui/... -run TestListEntryBuild -count=1` | ❌ W0 | ⬜ pending |
| 78-01-03 | 01 | 1 | TUI-07 | — | N/A | unit | `go test ./internal/tui/... -run TestCursorSkipsDividers -count=1` | ❌ W0 | ⬜ pending |
| 78-01-04 | 01 | 1 | TUI-07 | — | N/A | unit | `go test ./internal/tui/... -run TestSelectionStableAcrossRefresh -count=1` | ❌ W0 | ⬜ pending |
| 78-02-01 | 02 | 1 | TUI-10 | — | N/A | unit | `go test ./internal/tui/... -run TestQRRender -count=1` | ❌ W0 | ⬜ pending |
| 78-02-02 | 02 | 1 | TUI-10 | — | N/A | unit | `go test ./internal/tui/... -run TestQRDecodeRoundTrip -count=1` | ❌ W0 | ⬜ pending |
| 78-02-03 | 02 | 1 | TUI-10 | — | N/A | unit | `go test ./internal/tui/... -run TestQROverlayLifecycle -count=1` | ❌ W0 | ⬜ pending |
| 78-03-01 | 03 | 2 | TUI-07, TUI-10 | — | N/A | integration | `go test ./internal/tui/... -run TestTUIRemoteAndQR -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Per-plan task rows are finalized by the planner. Checker enforces that every task cites an `<automated>` command or a Wave 0 dependency.*

---

## Wave 0 Requirements

- [ ] `internal/tui/remote_test.go` — unit stubs exercising `buildListEntries`, cursor navigation over dividers, and selection restoration by identity after refresh (TUI-07)
- [ ] `internal/tui/qr_test.go` — unit stubs covering QR render dimensions, quiet-zone presence, and decode round-trip verification of the rendered ASCII (TUI-10)
- [ ] `internal/tui/integration_test.go` — scripted bubbletea program test wiring remote-sessions fetch + QR overlay open/close flow (TUI-07, TUI-10)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| QR readability with a real phone camera | TUI-10 | End-to-end usability check — automated decode proves encoding correctness but not visual scan-ability in diverse terminals | Open TUI, select a session, press `q`, scan with iOS/Android camera in Terminal.app, iTerm2, Alacritty, and kitty; URL must decode and open in a browser |
| Remote peer parity vs GUI | TUI-07 | Cross-UI visual diff — ensures grouping/ordering matches the GUI Remote Sessions panel | With a live tailnet, open both GUI Remote Sessions and TUI side-by-side; confirm same peers, same groups, same per-session status |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
