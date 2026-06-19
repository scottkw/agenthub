---
phase: 132-unified-grid-mini-preview-named-groups
fixed_at: 2026-06-16T18:25:00Z
review_path: .planning/phases/132-unified-grid-mini-preview-named-groups/132-REVIEW.md
iteration: 1
findings_in_scope: 9
fixed: 9
skipped: 0
status: all_fixed
---

# Phase 132: Code Review Fix Report

**Fixed at:** 2026-06-16T18:25:00Z
**Source review:** `.planning/phases/132-unified-grid-mini-preview-named-groups/132-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 9 (CR-01, CR-02, CR-03, WR-01, WR-02, WR-03, WR-04, WR-05, IN-01, IN-03)
- Fixed: 9
- Skipped: 0

## Fixed Issues

### CR-01: ANSI Regex Fails to Strip OSC 8 Hyperlinks with Escaped-String Terminator

**Files modified:** `internal/daemon/engine.go`, `internal/daemon/engine_test.go`
**Commit:** `0a6357e6`
**Applied fix:** Split the OSC branch of `ansiEscape` into two separate alternations: BEL-terminated (`[^\x07\x1b]*\x07`) and ST-terminated (`[^\x1b]*\x1b\\`). The original single branch used `[^\x07\x1b]*` which stopped at the first `\x1b` in both arms, leaving the `\x5c` backslash of the ST terminator as a visible literal character. Updated comment block. Added `TestGetSessionTailLines_StripsOSC8Hyperlink` asserting that OSC 8 hyperlinks with ST terminator are fully stripped including the URL and no trailing backslash artifact.

---

### CR-02: `handleGetSessionTailLines` — No Upper Bound on `n` at the HTTP Layer

**Files modified:** `internal/daemon/api.go`, `internal/daemon/api_test.go`
**Commit:** `254eea97`
**Applied fix:** Added `if n > 20 { n = 20 }` clamp immediately after parsing `n` from the query string, mirroring the clamp in `app.go` (lines 438–443). Added `TestHandleGetSessionTailLines_ClampN` which creates a session with 30 lines of content, requests `n=1000`, and asserts the response contains at most 20 lines.

---

### CR-03: `usePreviewPoller` Overwrites Entire Tails Map — Stopped-Session Previews Flicker

**Files modified:** `frontend/src/components/Hub/HubPanel.tsx`, `frontend/src/components/Hub/HubPanel.test.tsx`
**Commit:** `707c9597` (combined with WR-04)
**Applied fix:** Changed `setTails(new Map(...))` to a functional update that merges results: only overwrites a session's entry if `lines.length > 0` or if no prior value exists. Stopped/killed sessions retain their last-seen snapshot. Also fixed WR-04 in the same commit (see below).

---

### WR-01: "Other (default)" Menu Item Shown for Already-Ungrouped Sessions

**Files modified:** `frontend/src/components/Hub/SessionCard.tsx`, `frontend/src/components/Hub/SessionCard.test.tsx`
**Commit:** `ef90246b`
**Applied fix:** Wrapped the "Other (default)" button in `{isInNamedGroup && (...)}` so it only appears when the session is in a named group (where it functions as "move back to Other"). Removed the separate "Remove from group" section that followed — it was redundant (same `handleAssign('__other__')` call). Updated two tests: `"shows 'Remove from group' only..."` replaced with `WR-01: shows "Other (default)" only when session IS in a named group`; `"does NOT show 'Remove from group'..."` updated to also assert "Other (default)" is absent for ungrouped sessions.

---

### WR-02: `hub-card--dragging` CSS Class Referenced in TSX but Not Defined

**Files modified:** `frontend/src/style.css`
**Commit:** `5526a409` (combined with WR-03)
**Applied fix:** Added `.hub-card--dragging { opacity: 0.5; cursor: grabbing; }` before the drag-handle rules block. Colorblind-safe: opacity change and cursor shape are the primary visual cues, not color. Includes `COLORBLIND-SAFE` comment. Matches UI-SPEC drag-state treatment (`opacity: 0.5` while dragging).

---

### WR-03: `hub-card__menu-item--header` and `hub-card__menu-item--group` Missing from `style.css`

**Files modified:** `frontend/src/style.css`
**Commit:** `5526a409` (combined with WR-02)
**Applied fix:** Added `.hub-card__menu-item--header` (dimmed/uppercase/non-interactive "Move to group" label) and `.hub-card__menu-item--group` (20px left-indent under header) rules after `.hub-card__menu-item--sub`. The header gets `pointer-events: none; cursor: default; color: var(--hub-text-muted)` to visually distinguish it from clickable items.

---

### WR-04: `usePreviewPoller` Includes Remote Session IDs in `sessionIdKey` Dep

**Files modified:** `frontend/src/components/Hub/HubPanel.tsx`, `frontend/src/components/Hub/HubPanel.test.tsx`
**Commit:** `707c9597` (combined with CR-03)
**Applied fix:** Changed `sessionIdKey` to filter local sessions only before joining IDs: `.filter((s) => !s.hostname || s.hostname === '').map((s) => s.id).join(',')`. Remote session ID changes (from 30s remote polls) no longer trigger the preview poller effect to restart its interval. Added test `CR-03: does NOT call GetSessionTailLines for remote sessions when remote session ID changes (WR-04)` asserting only 2 calls (one per local session) with no calls for remote sessions.

---

### WR-05: `sidebarCollapsed` Initialization Reads `localStorage` Without Try/Catch

**Files modified:** `frontend/src/components/Hub/HubPanel.tsx`
**Commit:** `8254052c`
**Applied fix:** Wrapped the lazy initializer in a `try/catch` block that returns `false` on error. Matches the pattern used by `loadGroups()` in `hubGroups.ts` (line 17–21). A `SecurityError` from disabled storage no longer crashes Hub mount.

---

### IN-01: `GetSessionTailLines` Returns `nil` When Session Has No Hub

**Files modified:** `internal/daemon/engine.go`, `internal/daemon/engine_test.go`
**Commit:** `dd6272a8`
**Applied fix:** Changed `return nil` to `return []string{}` in the `!ok` early-exit path. Updated `TestGetSessionTailLines_UnknownSession` to assert non-nil, len 0 (previously asserted nil). Updated doc comment and section comment block to reflect the change.

---

### IN-03: `adaptRemoteSession` Uses `createdAt = now` — Shows Misleading "0m" Uptime

**Files modified:** `frontend/src/components/Hub/SessionCard.tsx`, `frontend/src/components/Hub/SessionCard.test.tsx`
**Commit:** `23cccd90`
**Applied fix:** Added a remote-session guard to `timeText`: when `hostname && hostname !== ''`, set `timeText = ''` (blank) instead of calling `formatUptime(createdAt)` which would show meaningless near-zero time. Also updated `makeSession()` test helper default from `hostname: 'local-machine.local'` to `hostname: ''` (the production value for local sessions), which is the correct baseline for local-session tests.

---

## Validation

All required test suites were run after all fixes were applied:

- `go build ./...` — PASS
- `go test ./internal/daemon/... -count=1` — PASS (full suite, 9s)
- `vitest run` (full frontend suite) — PASS (99 files, 1598 tests)

---

_Fixed: 2026-06-16T18:25:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
