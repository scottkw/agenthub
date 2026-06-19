---
phase: 133-attention-pulse
fixed_at: 2026-06-16T21:33:00Z
review_path: .planning/phases/133-attention-pulse/133-REVIEW.md
iteration: 1
findings_in_scope: 9
fixed: 9
skipped: 0
status: all_fixed
---

# Phase 133: Code Review Fix Report

**Fixed at:** 2026-06-16T21:33:00Z
**Source review:** .planning/phases/133-attention-pulse/133-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 9
- Fixed: 9
- Skipped: 0

## Fixed Issues

### CR-01: FLIP capturePositions fires on EVERY render — not only before sort-driven re-renders

**Files modified:** `frontend/src/components/Hub/SessionCardGrid.tsx`, `frontend/src/components/Hub/SessionCardGrid.test.tsx`
**Commit:** `94fdc4a3` (combined with WR-02 and WR-03)
**Applied fix:** Removed the standalone always-running `capturePositions` useLayoutEffect (no dep array). Replaced with a single `useLayoutEffect([debouncedSortKey])` that calls `playFLIP()` after the DOM update and `capturePositions()` in the cleanup (before the next debouncedSortKey-triggered mutation). `prevPositions` is now never overwritten by unrelated renders (preview polls, filter changes), so `deltaY` is always measured against the correct pre-sort snapshot.

---

### CR-02: `.hub-card--attention` pulse animation suppressed by `.hub-card--dim` opacity

**Files modified:** `frontend/src/style.css`
**Commit:** `3e3e1aa2`
**Applied fix:** Added `opacity: 1` to `.hub-card--attention` rule. This enforces the dim/attention mutual-exclusion invariant at the CSS level — attention always wins opacity regardless of any co-applied modifier class. Self-documenting comment added.

---

### CR-03: Missing CSS rule for `.hub__group-sidebar-item__attn-badge--count`

**Files modified:** `frontend/src/style.css`, `frontend/src/components/__tests__/style.hub.test.ts`
**Commit:** `3e3e1aa2` (CSS rule), `812d69aa` (test assertion)
**Applied fix:** Added `.hub__group-sidebar-item__attn-badge--count { font-size: 11px; font-weight: 600; line-height: 1; color: var(--hub-attn-badge-text); }` after `.hub__group-sidebar-item__attn-badge svg` in style.css. Also added CSS contract assertion to style.hub.test.ts.

---

### WR-01: `handleSidebarToggle` writes to localStorage without try/catch

**Files modified:** `frontend/src/components/Hub/HubPanel.tsx`
**Commit:** `c85e2856`
**Applied fix:** Wrapped `localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(next))` in a try/catch block matching the existing guard on the getItem read side. Prevents SecurityError (private browsing / block-all-cookies) or QuotaExceededError from propagating out of the React state updater.

---

### WR-02: `capturePositions` skips under reduced-motion — `prevPositions` goes stale

**Files modified:** `frontend/src/components/Hub/SessionCardGrid.tsx`
**Commit:** `94fdc4a3`
**Applied fix:** Removed the `prefers-reduced-motion: reduce` early-return guard from `capturePositions`. `prevPositions` is now always kept in sync regardless of motion preference. Only `playFLIP` guards against the preference — measurement is unconditional. This prevents a mid-session preference change from reduce→no-preference animating to stale/wrong positions.

---

### WR-03: `sortSessionsForDisplay` runs on every render — cards reorder immediately before animation

**Files modified:** `frontend/src/components/Hub/SessionCardGrid.tsx`, `frontend/src/components/Hub/SessionCardGrid.test.tsx`
**Commit:** `94fdc4a3` (initial fix) + `74b155b6` (refined to fix filter regression)
**Applied fix:** The sort order is now debounce-gated. `groupByWorkDir` and `groupByNamedGroups` no longer call `sortSessionsForDisplay` internally. Inside the component:
1. `sortedOrder` (array of IDs) is memoized on `debouncedSortKey` — re-derives only when the 1s debounce settles.
2. `sortedSessions` applies the stable `sortedOrder` to the live `sessions` prop on every render — so filter/search changes apply immediately while the attention-float reordering waits for the debounce.

After the fix: card content (border/icon) updates live via `attentionIds`; card position updates only after 1s debounce. A regression (filter-change showing stale sorted set) was caught during testing and fixed in `74b155b6` using the two-memo `sortedOrder` approach.

Tests updated: renamed the attention-ordering test to pass `debouncedSortKey`; added WR-03 debounce-gate assertion confirming sort does not re-run on same key.

**Note: requires human verification** — the debounce-gate logic is a correctness change to observable timing behavior. Automated tests verify the memo boundary; visual behavior (1s delay before reorder, then FLIP animation) should be confirmed with a live UAT.

---

### WR-04: `NeedsInputBadge` uses dead Tailwind `w-3 h-3` class

**Files modified:** `frontend/src/components/Hub/GroupSidebar.tsx`
**Commit:** `aba644cc`
**Applied fix:** Removed `className="w-3 h-3"` from `PauseCircleIcon` in `NeedsInputBadge` and `className="w-4 h-4"` from `ChevronRightIcon`/`ChevronLeftIcon` in the sidebar toggle. All sizing is already handled by explicit CSS rules (`.hub__group-sidebar-item__needs-input-badge svg`, `.hub__group-sidebar-toggle svg`). Added clarifying inline comments pointing to those rules.

---

### IN-01: `.hub-card` lacks `position: relative`

**Files modified:** `frontend/src/style.css`, `frontend/src/components/__tests__/style.hub.test.ts`
**Commit:** `3e3e1aa2` (CSS), `812d69aa` (test assertion)
**Applied fix:** Added `position: relative` to `.hub-card` rule with comment explaining it anchors the absolutely-positioned `.hub-card__drag-handle` and `.hub-card__menu-btn`. Added CSS contract assertion to style.hub.test.ts.

---

### IN-02: `attentionIds` Set recreated on every render — referential identity lost

**Files modified:** `frontend/src/components/Hub/HubPanel.tsx`
**Commit:** `c85e2856`
**Applied fix:** Wrapped `attentionIds` derivation in `React.useMemo` keyed on `attentionSortKey`. Since `attentionSortKey` already encodes all `id:bit` changes, the Set keeps stable referential identity across unrelated renders (preview polls, filter changes). The `attentionSortKey` derivation is computed first so `attentionIds` can depend on it.

---

## Skipped Issues

None — all 9 findings were fixed.

---

_Fixed: 2026-06-16T21:33:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
