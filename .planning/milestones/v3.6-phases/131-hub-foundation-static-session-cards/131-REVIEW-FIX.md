---
phase: 131-hub-foundation-static-session-cards
fixed_at: 2026-06-16T14:40:00Z
review_path: .planning/phases/131-hub-foundation-static-session-cards/131-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 131: Code Review Fix Report

**Fixed at:** 2026-06-16T14:40:00Z
**Source review:** .planning/phases/131-hub-foundation-static-session-cards/131-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7
- Fixed: 7
- Skipped: 0

## Fixed Issues

### CR-01: CSS class names in Hub components do not match style.css definitions

**Files modified:** `frontend/src/components/Hub/SessionCard.tsx`, `frontend/src/components/Hub/HubFilterBar.tsx`, `frontend/src/style.css`, `frontend/src/components/Hub/SessionCard.test.tsx`
**Commit:** 4a9a110
**Applied fix (Option A — align TSX to CSS):**
- SessionCard.tsx: renamed row divs to match CSS definitions
  - `hub-card__row hub-card__row--primary` → `hub-card__row1`
  - `hub-card__row hub-card__row--origin` → `hub-card__row2`
  - `hub-card__row hub-card__row--meta` → `hub-card__row3`
  - `hub-card__row hub-card__row--exit` → `hub-card__row4`
  - `hub-card__time` → `hub-card__uptime`
- HubFilterBar.tsx: outer wrapper div `hub-filter` → `hub__filter-bar` (matches CSS rule)
- style.css: added four missing CSS rules:
  - `.hub-card__status-indicator` (flex row for icon + label)
  - `.hub-card__status-label` (11px secondary text)
  - `.hub-filter__pills` (flex grouping container)
  - `.hub-filter__new-session` (New session button styling + hover)
- SessionCard.test.tsx: updated selectors from `hub-card__time` → `hub-card__uptime`

### CR-02: Double RenameSession RPC call on every Hub card rename

**Files modified:** `frontend/src/components/Hub/InlineSessionName.tsx`, `frontend/src/components/Hub/InlineSessionName.test.tsx`
**Commit:** b60c069
**Applied fix:**
- Removed the direct `RenameSession(id, trimmed)` call from `InlineSessionName.commitEdit`
- Component now fires `onRenamed?.(trimmed)` only; App.handleRenameTab owns the RPC
- `commitEdit` is now synchronous (no async/await); `onBlur` handler simplified
- Removed the `RenameSession` import (no longer used in this file)
- The `id` prop is still accepted (prefixed as `_id`) for backward-compatibility of the interface, but not used internally
- InlineSessionName.test.tsx updated: assert `onRenamed` fires and `RenameSession` is NOT called directly from this component

### WR-01: `deriveStatus` logic triplicated across three Hub files

**Files modified:** `frontend/src/lib/hubStatus.ts` (new), `frontend/src/components/Hub/SessionCard.tsx`, `frontend/src/components/Hub/HubFilterBar.tsx`, `frontend/src/components/Hub/HubPanel.tsx`
**Commit:** 389edfe
**Applied fix:**
- Created `frontend/src/lib/hubStatus.ts` exporting `HubStatus` type and `deriveHubStatus(s: SessionInfo): HubStatus`
- All three components now import `{ deriveHubStatus }` from the shared util
- Local `deriveStatus`/`deriveFilterStatus` functions removed from all three files
- TypeScript: `bucket as HubFilter` cast in `HubFilterBar.computeCounts` is safe because the `in` guard already ensures runtime safety; `HubStatus` is a superset of `HubFilter` (adds `'errored'`, excludes `'all'`)

### WR-02: `hubSessions` stale on Hub tab re-open; Hub error state never clears

**Files modified:** `frontend/src/App.tsx`
**Commit:** 93e2ac2
**Applied fix:**
- Added `setHubError(false)` immediately after the `activeId !== HUB_TAB.id` guard (before the first async `refresh()` call)
- `hubSessions` is deliberately NOT reset — avoids a flash-to-empty; the first `refresh()` populates it promptly
- This matches the REVIEW.md "at minimum" recommendation

### WR-03: `SessionCard` renders `tab__agent-badge` instead of text badge `hub-card__badge`

**Files modified:** `frontend/src/components/Hub/SessionCard.tsx`, `frontend/src/components/Hub/SessionCard.test.tsx`
**Commit:** 4a9a110 (same commit as CR-01 — both changes are in SessionCard.tsx)
**Applied fix:**
- Removed `agentBadgeModifier` helper function (no longer needed)
- CLI badge element changed from `<span className={badgeClass} aria-hidden="true">{cli}</span>` to `<span className="hub-card__badge">{cli}</span>`
- `aria-hidden` removed — text badge must be announced by screen readers (CLI name is the primary differentiator for colorblind users)
- SessionCard.test.tsx: updated CLI badge selector from `.tab__agent-badge--claude` to `.hub-card__badge`

### IN-01: `_workDir` unused destructure in `SessionCard`

**Files modified:** `frontend/src/components/Hub/SessionCard.tsx`
**Commit:** 4a9a110 (same commit as CR-01)
**Applied fix:**
- Removed `workDir: _workDir` from the destructuring block in `SessionCard`
- `session` is never spread in this component; the discard was noise

### IN-02: `hub-card__exit-chip` span creates duplicate screen-reader announcement

**Files modified:** `frontend/src/components/Hub/SessionCard.tsx`
**Commit:** 4a9a110 (same commit as CR-01)
**Applied fix:**
- Added `aria-hidden="true"` to the `<span className="hub-card__exit-chip">` element
- The exit code is already communicated in the card's `aria-label` (`${name}, Exited 1, ${cli}, ${originText}`)
- This prevents screen readers from announcing the exit code twice

---

## Verification

**TypeScript:** `pnpm exec tsc --noEmit` — zero errors
**Tests:** `pnpm vitest run src/components/Hub/ src/components/__tests__/style.hub.test.ts src/components/__tests__/App.hub.test.tsx` — 8 test files, 147 tests, all passed
**Class name parity:** CSS rules now have matching TSX class names for all Hub card rows, status indicator, filter bar wrapper, filter pills group, and new-session button

---

_Fixed: 2026-06-16T14:40:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
