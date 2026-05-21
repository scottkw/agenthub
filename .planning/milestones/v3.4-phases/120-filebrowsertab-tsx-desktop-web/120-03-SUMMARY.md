---
phase: 120
plan: 03
subsystem: frontend-ui
tags: [react, vitest, ui, accessibility, keyboard-nav, file-browser]
requires:
  - frontend/src/lib/filesApi.ts (Wave 1 stub — FileEntry shape)
  - frontend/src/lib/filesTypes.ts (Wave 1 stub — SortKey, SortDir, BreadcrumbSegment)
  - frontend/src/lib/humanSize.ts (Wave 1 stub — humanSize formatter)
provides:
  - BreadcrumbBar (path navigation chrome)
  - FileListPane (listbox + column headers + keyboard nav)
  - FileRow (leaf row with icon + name + size + mtime)
  - StatusLine (item count + filter input)
  - sortEntries (pure comparator with directories-sticky rule)
  - defaultSortDir (per-column default direction)
affects:
  - frontend/src/style.css (appended ~280 lines under "File Browser Tab (Phase 120)" section)
tech-stack:
  added:
    - None (no new runtime dependencies — uses existing @heroicons/react)
  patterns:
    - Pure-function sort comparator with TDD coverage (sortEntries)
    - React useEffect interval keyed on prop (BreadcrumbBar refresh ticker — RESEARCH Pitfall 4)
    - Auto-focus pattern via useEffect + ref (StatusLine filter input)
    - useMemo'd filter+sort pipeline (FileListPane displayEntries)
    - role=listbox / role=option / role=columnheader ARIA composition
    - data-testid taxonomy from 120-UI-SPEC §"Test Hooks"
key-files:
  created:
    - frontend/src/components/FileBrowser/sortEntries.ts
    - frontend/src/components/FileBrowser/BreadcrumbBar.tsx
    - frontend/src/components/FileBrowser/StatusLine.tsx
    - frontend/src/components/FileBrowser/FileListPane.tsx
    - frontend/src/components/FileBrowser/FileRow.tsx
    - frontend/src/components/FileBrowser/__tests__/sortEntries.test.ts
    - frontend/src/components/FileBrowser/__tests__/BreadcrumbBar.test.tsx
    - frontend/src/components/FileBrowser/__tests__/FileListPane.test.tsx
    - frontend/src/lib/filesApi.ts (Wave 1 stub)
    - frontend/src/lib/filesTypes.ts (Wave 1 stub)
    - frontend/src/lib/humanSize.ts (Wave 1 stub)
  modified:
    - frontend/src/style.css (appended File Browser Tab section, ~280 new lines, no existing rules changed)
decisions:
  - "Empty mtime entries sink to the bottom of the modified-asc sort group and rise to the top under desc — implemented by biasing the bare comparator to put empties last, then relying on the existing dir-reversal for desc semantics. Keeps the comparator pure and single-direction-aware."
  - "PAGE_SIZE constant fixed at 10 for PgUp/PgDn. UI-SPEC suggests viewport÷row-height but the listbox container has no ref-stable height in unit tests; the constant is a safe baseline and Plan 04 can swap to viewport-relative once the layout is wired."
  - "FileRow size column renders em-dash for both directories AND files with size==0 because the Phase 118 List endpoint leaves size=0 (lazy-stat per CONTEXT D-03). The em-dash is the v3.4 placeholder; Plan 04 wires a per-row stat call when a row is selected and re-renders with humanSize."
  - "BreadcrumbBar refresh ticker uses setInterval keyed on the refreshedAt prop (RESEARCH Pitfall 4). The interval cleans up on unmount and on prop change so we never run two intervals concurrently."
  - "Stubbed Wave-1 dependencies (filesApi.ts, filesTypes.ts, humanSize.ts) locally because Wave 1 was running in parallel; the stubs match Plan 02's <interfaces> contract byte-for-byte so the upcoming merge replaces them cleanly with the richer Plan 02 implementations."
metrics:
  duration: "~25 minutes (single-pass execution; no checkpoints, no deviations)"
  completed: "2026-05-20"
  tasks_completed: 3
  files_created: 11
  files_modified: 1
  new_tests: 35 (11 sortEntries + 7 BreadcrumbBar + 17 FileListPane)
  total_frontend_tests_passing: 957
---

# Phase 120 Plan 03: BreadcrumbBar + FileListPane + FileRow + StatusLine + sortEntries Summary

## One-liner

Five purely presentational React components plus a pure sort comparator that together form the list-side surface of the FileBrowserTab (UI-02 / UI-03 / UI-04 / UI-05 / UI-12) with 35 vitest cases covering directories-sticky sort, breadcrumb navigation, keyboard map, and ARIA semantics.

## What was built

### Pure modules

**`sortEntries.ts`** — pure function `sortEntries(entries, sortKey, sortDir)` returning a new array with directories always at the top, sorted within each group by name (case-insensitive), size (numeric), or modified (RFC3339 lexical with empty-mtime sink). `defaultSortDir(key)` returns asc for name, desc for size and modified. 11 vitest cases.

### React components

**`BreadcrumbBar.tsx`** — top chrome of the file browser tab.
- `<nav aria-label="Path">` with an `<ol>` of segments; first is non-clickable text ("session/"), middle segments are `<button>` firing `onNavigateTo(pathFromCwd)`, last segment is text with `aria-current="page"`.
- Right side: `ArrowPathIcon` button labeled "Refresh directory listing" + live "Last refreshed Ns ago" text that ticks every 5s via setInterval keyed on the `refreshedAt` prop (RESEARCH Pitfall 4: re-key, don't recompute).
- Format: <5s → "just now", <60s → seconds, ≥60s → minutes. 7 vitest cases.

**`StatusLine.tsx`** — bottom chrome with item count + inline filter.
- `<div role="status" aria-live="polite">` wrapper.
- Item count copy is pluralized ("1 item" vs "N items").
- Filter input (`<input type="search" role="searchbox" aria-label="Filter files in current directory">`) renders only when `filterActive=true`; auto-focuses via `useEffect + ref` so the user can type immediately after pressing `/`. Escape fires `onFilterDismiss()`.
- When filter is active and non-empty: shows "N of M items" left and "Filtering: "q" — N matches" right.

**`FileListPane.tsx`** — listbox + column headers + keyboard navigation (UI-12).
- Root `<div tabIndex=0 onKeyDown={...}>` so the listbox is keyboard-focusable.
- Column-header row of three `<button role="columnheader" aria-sort=...>` for NAME / SIZE / MODIFIED; the active column shows a `ChevronUpIcon` (asc) or `ChevronDownIcon` (desc).
- Filter-then-sort pipeline via `useMemo`: case-insensitive substring filter (when `filter` non-empty) → `sortEntries`.
- Keyboard: ArrowDown/Up, Home/End, PageDown/Up (page size 10), Enter on a directory → `onNavigateInto`, Backspace → `onNavigateUp`, `/` → `preventDefault` + `onFilterActivate`.
- Renders truncation banner when `truncated=true`; renders a no-match row when filter is active but yields zero results. 17 vitest cases.

**`FileRow.tsx`** — leaf row component.
- `<li role="option" aria-selected={isSelected}>` with composed `aria-label` ("{name}, {kind}, {size}, modified {mtime}") for SR announcement.
- Heroicon glyph picks file type — Folder / Document / DocumentText / Photo / Link with an ExclamationTriangle overlay for broken-symlink hint.
- Three-channel selection signal per UI-SPEC colorblind contract: 3px `#7aa2f7` left border + `aria-selected=true` + inverted `#1e2030` background + bold name. All three are CSS-driven so the row only needs `--selected` modifier class.
- Size column shows em-dash for directories AND for files with `size==0` (lazy-stat is a Plan 04 concern).

### CSS

`frontend/src/style.css` gained ~280 lines under a `/* ─── File Browser Tab (Phase 120) ─── */` section. All hex tokens reuse the existing TokyoNight palette already in the stylesheet (`#1a1b26`, `#16161e`, `#1e2030`, `#292e42`, `#7aa2f7`, `#c0caf5`, `#a9b1d6`, `#9aa5ce`, `#565f89`, `#f7768e`, `#f59e0b`) — no new colors introduced. Skeleton shimmer animation is gated by `@media (prefers-reduced-motion: no-preference)` per UI-SPEC §Motion.

## Verification

| Check | Result |
| ----- | ------ |
| `pnpm test` (full suite) | 957/957 PASS |
| `pnpm test -- --run src/components/FileBrowser/__tests__/` | 35/35 PASS |
| `pnpm exec tsc --noEmit` | clean (0 errors) |
| `role="listbox"` in FileListPane | 1 |
| `role="option"` in FileRow | 1 |
| `role="columnheader"` in FileListPane | 1 |
| `aria-sort=` in FileListPane | 1 |
| `aria-current="page"` in BreadcrumbBar | 1 |
| `aria-label="Refresh directory listing"` in BreadcrumbBar | 1 |
| `aria-label="Filter files in current directory"` in StatusLine | 1 |
| `data-testid="file-browser-` count in FileListPane + FileRow + BreadcrumbBar + StatusLine | 9 unique testids |
| `dangerouslySetInnerHTML` in any FileBrowser component | 0 |
| New hex tokens beyond UI-SPEC §Color palette | 0 (subset relationship verified) |
| Auto-fix attempts | 1 (mtime sort direction logic — fixed before commit) |

### data-testid coverage (UI-SPEC §"Test Hooks")

Present in this plan's components:
- `file-browser-breadcrumb`
- `file-browser-breadcrumb-segment-{n}` (templated)
- `file-browser-refresh`
- `file-browser-list`
- `file-browser-list-scroll`
- `file-browser-row-{name}` (templated)
- `file-browser-col-{name|size|modified}`
- `file-browser-truncated`
- `file-browser-no-match`
- `file-browser-status`
- `file-browser-filter`

Not yet present (delivered by Plan 04 — preview pane + tab root):
- `file-browser-tab`
- `file-browser-preview*`
- `file-browser-download`
- `file-browser-empty`
- `file-browser-permission-denied`
- `file-browser-network-error`
- `file-browser-over-cap`
- `file-browser-binary`
- `file-browser-broken-symlink`
- `file-browser-loading`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] mtime sort direction logic was incorrect on first attempt**
- **Found during:** Task 1 first test run
- **Issue:** The mtime comparator tried to bias empty-mtime placement using the `sortDir` argument, but the result was then reversed AGAIN by the generic dir-reversal step, flipping the empty-mtime to the wrong end.
- **Fix:** Made the mtime comparator direction-agnostic — empty entries always sort to the bottom under bare ascending order. The single `.reverse()` step then naturally rises them to the top of their group for desc. Cleaner contract, fewer special cases.
- **Files modified:** `frontend/src/components/FileBrowser/sortEntries.ts`
- **Commit:** 96f6533 (included in the GREEN commit; RED→GREEN cycle was atomic for a single feature)

### Parallel-execution accommodation

This plan was executed in a Wave 2 worktree while Wave 1 (Plan 02 — `filesApi.ts`, `filesTypes.ts`, `humanSize.ts`) was running concurrently. To allow Plan 03 components to compile and unit-test independently, I authored minimal stubs of those three modules in this worktree under a separate `chore(120-03): stub Wave-1 type deps` commit (75c849b). The stubs match Plan 02's `<interfaces>` block byte-for-byte (FileEntry shape, SortKey/SortDir/BreadcrumbSegment unions, humanSize signature). When Wave 1 merges, Plan 02's richer implementations supersede the stubs cleanly — no behavior change in the consumers.

## Decisions Made

1. **Empty mtime semantics:** Bare comparator places empties last; reversal handles desc-rise. Documented in sortEntries.ts header comment.
2. **PAGE_SIZE = 10:** Static constant for PgUp/PgDn. UI-SPEC suggests viewport÷row-height but the container has no test-stable ref height. Plan 04 can swap to a measured value when the layout is mounted in the real DOM.
3. **size=0 rendering:** Em-dash for both directories and files with size=0. The Phase 118 List endpoint leaves size=0 (lazy-stat per CONTEXT D-03); Plan 04 will wire a per-row stat call when a row is selected.
4. **StatusLine has no dedicated test file** (plan stipulated): visible-only-when-active behavior is covered by Plan 05's e2e suite, and the filter ownership lives in FileBrowserTab (Plan 04), so unit-testing the empty container has no value.
5. **No '/' document-level handler in FileListPane:** Plan 04 owns the document-level focus-conditional listener (`isXtermFocused`-style gating). FileListPane only handles `/` when the listbox itself has focus.

## Known Stubs

None that block the plan goal. The three Wave-1 dependency stubs (`filesApi.ts`, `filesTypes.ts`, `humanSize.ts`) are by design — Plan 02 supersedes them. The richer FilesApiClient class, FilesApiError class, and PreviewState discriminated union from Plan 02 are NOT referenced by any Plan 03 component, so the absence of those exports in the stubs causes no compile breakage in this plan's scope.

## Commits

- 75c849b: `chore(120-03): stub Wave-1 type deps for parallel execution`
- 96f6533: `feat(120-03): add sortEntries pure comparator with TDD coverage`
- 6458be4: `feat(120-03): add BreadcrumbBar + StatusLine components`
- b0a648a: `feat(120-03): add FileListPane + FileRow with keyboard navigation`

## Self-Check: PASSED

- [x] `frontend/src/components/FileBrowser/sortEntries.ts` exists
- [x] `frontend/src/components/FileBrowser/BreadcrumbBar.tsx` exists
- [x] `frontend/src/components/FileBrowser/StatusLine.tsx` exists
- [x] `frontend/src/components/FileBrowser/FileListPane.tsx` exists
- [x] `frontend/src/components/FileBrowser/FileRow.tsx` exists
- [x] All three test files exist and pass (35/35 in this plan; 957/957 frontend total)
- [x] tsc --noEmit clean
- [x] All four commits present in `git log` (75c849b, 96f6533, 6458be4, b0a648a)
- [x] No modifications to STATE.md / ROADMAP.md
- [x] No `dangerouslySetInnerHTML`, no new hex tokens, no `fetch()` in components
