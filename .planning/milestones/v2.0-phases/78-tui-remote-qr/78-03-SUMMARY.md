---
phase: 78
plan: "03"
subsystem: tui
tags: [tui, integration-test, remote-sessions, qr-overlay, bubble-tea]
dependency_graph:
  requires: [78-01, 78-02]
  provides: [integration-test-tui-remote-qr]
  affects: [internal/tui]
tech_stack:
  added: []
  patterns: [scripted-bubbletea-model-test, lipgloss-url-wrapping-awareness]
key_files:
  created:
    - internal/tui/integration_test.go
  modified: []
decisions:
  - "URL overlay assertions check 'tail.ts.net' substring rather than full hostname because lipgloss word-wraps URLs at hyphens across lines, splitting 'laptop-work.tail.ts.net' into two rendered lines"
  - "Full URL correctness validated via m.qrURL field (not overlay string) to avoid lipgloss wrapping false negatives"
metrics:
  duration: "~3 minutes"
  completed: "2026-04-15"
  tasks: 1
  files: 1
---

# Phase 78 Plan 03: Integration Tests for Remote Sessions and QR Overlay Summary

**One-liner:** Two scripted Bubble Tea model integration tests exercising the complete TUI-07 + TUI-10 end-to-end flow: remote session loading, unified list, divider-skip navigation, QR overlay open/close lifecycle, and blocked remote operations.

## What Was Built

Plan 03 adds `internal/tui/integration_test.go` with two comprehensive integration tests that validate Plans 01 and 02 working together without a live daemon or tailnet.

### Test 1: `TestTUIRemoteAndQR_FullFlow`

Exercises the complete Phase 78 lifecycle in a single scripted model test:

1. **Session loading:** Sends `sessionsMsg` (2 local) then `remoteSessionsMsg` (1 peer, 2 sessions)
2. **Unified list:** Asserts `len(unifiedList) == 5` (2 local + 1 divider + 2 remote) and `unifiedList[2].kind == entryDivider`
3. **View content:** Asserts header shows `"2 local, 2 remote"`, divider shows `"Remote:"` and `"laptop-work"`, all 4 session names visible
4. **Divider-skip navigation:** Two `j` presses from index 0 lands on index 3 (skips divider at 2)
5. **QR open on remote:** `q` on remote session sets `qrSession`, `qrContent`, and correct pre-built URL
6. **Overlay content:** `renderQROverlay()` contains session name in title and URL domain portion
7. **QR close via Esc:** `Esc` clears `qrSession`
8. **Jump to top:** `g` returns `selected` to 0
9. **QR open on local:** `q` on local session opens overlay with `webStatus.URL/sessions/ID` URL
10. **QR close via q:** `q` while overlay open closes it (per UI-SPEC)
11. **Q quits:** `Q` returns non-nil quit command

### Test 2: `TestTUIRemoteAndQR_BlockedOperations`

Tests guard rails for remote sessions and missing web server:

1. **Kill blocked:** `d` on remote session sets toast `"Cannot kill remote session"`, `modal == modalNone`
2. **Rename blocked:** `r` on remote session sets toast `"Cannot rename remote session"`, `editing == false`
3. **QR without web server:** `q` on local session with `webStatus.Running=false` sets toast `"Web serving not enabled for this session"`, `qrSession == nil`

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | aefaa18 | test(78-03): add end-to-end integration tests for remote sessions and QR overlay |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Lipgloss word-wraps URL at hyphen in renderQROverlay output**
- **Found during:** Task 1 verification
- **Issue:** The test plan specified `strings.Contains(overlay, "laptop-work.tail.ts.net")` but lipgloss wraps the URL `https://laptop-work.tail.ts.net:7443/sessions/r1` at the hyphen, placing `"laptop-"` on one line and `"work.tail.ts.net:7443/sessions/r1"` on the next. The combined string `"laptop-work"` never appears in the overlay output.
- **Root cause:** lipgloss `Width(overlayWidth-2)` triggers word-wrap at hyphens when the URL approaches the inner content width. The URL is 51 chars; overlay inner width is ~51 chars so it wraps exactly at the hyphen.
- **Fix:** Changed overlay assertion to check `"tail.ts.net"` (which appears whole on a single line) and rely on `m.qrURL` for full URL correctness. This avoids lipgloss wrapping false negatives without weakening the test's intent.
- **Files modified:** `internal/tui/integration_test.go`
- **Commit:** aefaa18 (fixed before commit)

## Known Stubs

None — this plan adds tests only; no new production code with stubs.

## Threat Flags

None — integration tests operate on mocked in-process data. No new trust boundaries introduced.

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| internal/tui/integration_test.go exists with package tui | FOUND |
| Contains TestTUIRemoteAndQR_FullFlow | FOUND |
| Contains TestTUIRemoteAndQR_BlockedOperations | FOUND |
| Commit aefaa18 exists | FOUND |
| `go test ./internal/tui/... -run "TestTUIRemoteAndQR" -count=1` | PASS (2 tests) |
| `go test ./internal/tui/... -count=1` | PASS (76 tests) |
| `go test ./... -count=1` | PASS (all packages) |
