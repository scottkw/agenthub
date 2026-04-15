---
phase: 78
plan: "02"
subsystem: tui
tags: [tui, qr-code, keybindings, bubble-tea, overlay]
dependency_graph:
  requires: [78-01]
  provides: [qr-overlay, q-qr-keybinding, Q-quit-keybinding, help-hints-updated]
  affects: [internal/tui]
tech_stack:
  added: []
  patterns: [qr-modal-overlay, key-dispatch-priority-4, qr-no-color-styling]
key_files:
  created:
    - internal/tui/qr.go
  modified:
    - internal/tui/keys.go
    - internal/tui/help.go
    - internal/tui/model.go
    - internal/tui/update.go
    - internal/tui/view.go
    - internal/tui/update_test.go
    - internal/tui/view_test.go
    - internal/tui/help_test.go
decisions:
  - "QR code block rendered without lipgloss color styles -- ToSmallString(false) uses ANSI resets internally; color styling corrupts half-block chars"
  - "Terminal minimum 55x25 enforced before opening QR overlay per UI-SPEC to prevent overflow"
  - "handleQRKey swallows all keys except esc/q (close) and Q/ctrl+c (quit) matching kill-confirm pattern"
  - "qrRows variable removed -- overlay renders each QR line individually without needing total row count"
metrics:
  duration: "~25 minutes"
  completed: "2026-04-15"
  tasks: 3
  files: 8
---

# Phase 78 Plan 02: QR Code Overlay Summary

**One-liner:** ASCII QR overlay triggered by `q` on web-served sessions using go-qrcode ToSmallString, with q->QR/Q->Quit key reassignment, terminal size guard, and 10 new unit tests.

## What Was Built

Plan 02 satisfies TUI-10 by adding the QR code overlay modal to the Bubble Tea TUI. The `q` key was reassigned from Quit to QR trigger; Quit moves to `Q` (Shift+Q) and `Ctrl+C`. The QR overlay follows the exact same modal pattern as kill-confirm and new-session overlays (rounded border, injectBorderTitle, lipgloss.Place centering).

### Task 1: Keybinding Reassignment and Help/Hint Updates

**keys.go:**
- `Quit` binding changed from `"q", "ctrl+c"` to `"Q", "ctrl+c"`
- New `QR` binding added: `key.WithKeys("q")`, `key.WithHelp("q", "QR code / URL")`

**help.go:**
- Sessions group: `formatBinding("q", "QR code / URL")` added after Enter/Attach
- General group: `formatBinding("Q, Ctrl+C", "Quit")` (was `"q, Ctrl+C"`)

**view.go:**
- `renderHintBar`: `"q QR ... Q Quit"` (was `"q Quit"`)
- `renderWebStatus`: quit hint changed to `"Q Quit"` (was `"q Quit"`)
- `renderFull`: `if m.qrSession != nil { return m.renderQROverlay() }` added after modal checks

**model.go:** Fields `qrSession *sessionRef`, `qrContent string`, `qrURL string` were already added by Plan 01; no changes needed.

### Task 2: QR Overlay Renderer and Key Handlers

**qr.go (new file):**
- `renderQROverlay()`: centered modal overlay with rounded border, border title `"QR: {session-name}"` in `BorderAccent` color, QR code block (NO lipgloss color styling — critical for ANSI half-block rendering), URL line in `FgAccent`, "Esc: close" hint in `FgMuted` centered
- Overlay width: `max(qr_cols+6, url_len+6, 50)` clamped to `min(terminal_cols-4, 80)` per UI-SPEC
- `sessionURL(entry listEntry) string`: returns web URL for local sessions (`webStatus.URL + /sessions/ + ID`) or remote sessions (pre-built `entry.remote.URL`), empty string if unavailable

**update.go:**
- Import `qrcode "github.com/skip2/go-qrcode"` added
- `handleQRKey()`: closes on `esc`/`q`, quits on `Q`/`ctrl+c`, swallows all other keys
- `entryID()`: returns unique string ID for list entry (used for display/reference)
- `handleKey` priority 4 inserted: `if m.qrSession != nil { return m.handleQRKey(msg) }` — between new-session modal (priority 3) and help overlay (priority 5)
- `handleMainKey` QR case: checks `sessionURL`, terminal size (55x25 min), calls `qrcode.New(url, qrcode.Medium)`, stores `qrContent = q.ToSmallString(false)`, sets `qrSession`/`qrURL`

### Task 3: Comprehensive Tests (10 new + 3 updated)

**update_test.go (8 new, 1 updated):**
- `TestUpdate_KeyQuit`: updated from `'q'` to `'Q'`
- `TestUpdate_QuitKeyReassignment`: verifies `q` doesn't quit, `Q` does
- `TestUpdate_QROpen`: session + webStatus.Running=true → qrSession set, qrContent non-empty, correct URL
- `TestUpdate_QRClose`: Esc clears qrSession/qrContent/qrURL
- `TestUpdate_QRNoURL`: webStatus.Running=false → toast "Web serving not enabled for this session"
- `TestUpdate_QRTerminalTooSmall`: 50x20 terminal → toast "Terminal too small to display QR code"
- `TestUpdate_QRSwallowsKeys`: j key swallowed while QR overlay open, selection unchanged
- `TestUpdate_QRQuitFromOverlay`: Q quits even from QR overlay

**view_test.go (2 new, 1 updated):**
- `TestView_HintBar`: updated for `"q QR"` and `"Q Quit"` (not `"q Quit"`); also asserts old `"q Quit"` absent
- `TestView_QROverlayContent`: real `qrcode.New()` call; asserts overlay contains title, URL, "Esc: close"

**help_test.go (1 new, 1 updated):**
- `TestHelpOverlay_ContainsBindings`: added `"QR code / URL"` to required list
- `TestHelp_QRBinding`: verifies "QR code / URL" and "Q, Ctrl+C" present, "q, Ctrl+C" absent

**Total: 74 tests pass (10 new + 3 updated + 61 unchanged)**

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | ce22765 | feat(78-02): reassign q->QR Q->Quit, update help/hints, add QR model fields and stub |
| 2 | 6af684d | feat(78-02): implement QR overlay renderer and key handlers |
| 3 | 845efb4 | test(78-02): add comprehensive QR overlay and key-reassignment tests |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Remove unused `qrRows` variable in qr.go**
- **Found during:** Task 2 verification (`go test` revealed build failure)
- **Issue:** `qrRows := len(qrLines)` declared but never used — Go compiler refuses to compile
- **Fix:** Removed `qrRows` variable; the overlay renders each QR line individually using `range qrLines`, so the total row count is not needed
- **Files modified:** `internal/tui/qr.go`
- **Commit:** 6af684d (fixed before commit)

**2. [Rule 1 - Bug] Test file was running against main repo, not worktree**
- **Found during:** Task 2 verification
- **Issue:** `cd /Users/ken/dev/agenthub && go test` was testing the MAIN repo's code (unchanged), not the worktree. Tests appeared to pass on old code.
- **Fix:** All test commands run without `cd` to stay in worktree directory. This revealed the real test failures that needed to be fixed in Task 3.
- **Impact:** `TestUpdate_KeyQuit` (expected `'q'` to quit — now tests `'Q'`) and `TestView_HintBar` (expected old `"q Quit"` hint) correctly failed and were updated in Task 3.

## Known Stubs

None — all QR overlay functionality is fully wired. The `renderQROverlay` stub from Task 1 was replaced by the full implementation in Task 2.

## Threat Flags

None beyond what the plan's threat model covered:
- T-78-06 (terminal size overflow): mitigated by 55x25 minimum check in `handleMainKey` before opening overlay
- QR content injection: URL always constructed from `webStatus.URL + /sessions/ + ID` (local) or pre-built HTTPS URL (remote) — not user-controlled

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| internal/tui/qr.go exists with renderQROverlay | FOUND |
| internal/tui/keys.go has QR binding and Q quit | FOUND |
| internal/tui/help.go has "QR code / URL" and "Q, Ctrl+C" | FOUND |
| internal/tui/update.go has handleQRKey, QR priority 4, QR trigger | FOUND |
| internal/tui/view.go dispatches to renderQROverlay, hint/web updated | FOUND |
| Commit ce22765 exists | FOUND |
| Commit 6af684d exists | FOUND |
| Commit 845efb4 exists | FOUND |
| `go build ./...` | PASS |
| `go test ./internal/tui/... -count=1` | PASS (74 tests) |
