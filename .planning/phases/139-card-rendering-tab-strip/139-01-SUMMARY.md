---
phase: 139-card-rendering-tab-strip
plan: "01"
subsystem: test-scaffolding
tags: [tdd, card-rendering, tab-strip, red-scaffold, a2-verification]
dependency_graph:
  requires: []
  provides:
    - RED Go tests for GetSessionStyledTailLines + StyledTailLinesResponse (plans 03 must satisfy)
    - RED frontend tests for resolveColor (plan 04 must satisfy)
    - RED frontend tests for MiniPreview StyledSpan[] rendering (plan 04 must satisfy)
    - RED frontend tests for TabBar chevron/floor/rename/title (plan 02 must satisfy)
    - A2 verification: headless xterm write+serializeAsHTML works in jsdom (unblocks plan 04)
  affects:
    - internal/daemon/engine_test.go
    - internal/daemon/api_test.go
    - frontend/src/lib/vtColor.test.ts
    - frontend/src/components/Hub/MiniPreview.test.tsx
    - frontend/src/components/__tests__/TabBar.test.tsx
    - frontend/src/lib/xtermHeadless.verify.test.ts
    - frontend/scripts/verify-xterm-headless.mjs
tech_stack:
  added: []
  patterns:
    - "RED/GREEN TDD: tests authored before implementation"
    - "Go test helper reuse: makeTailHub seeds Hub scrollback for unit tests"
    - "vitest jsdom: headless xterm runs without term.open() in jsdom environment"
key_files:
  created:
    - frontend/src/lib/vtColor.test.ts
    - frontend/src/lib/xtermHeadless.verify.test.ts
    - frontend/scripts/verify-xterm-headless.mjs
  modified:
    - internal/daemon/engine_test.go
    - internal/daemon/api_test.go
    - frontend/src/components/Hub/MiniPreview.test.tsx
    - frontend/src/components/__tests__/TabBar.test.tsx
decisions:
  - "A2 fallback: @xterm/xterm 6.x requires DOM shims (Terminal is not a constructor in raw Node.js); vitest jsdom is the authoritative A2 artifact, not the .mjs script"
  - "TabBar chevron tests: source-inspection pattern (raw import) used for state/ResizeObserver checks; DOM-render pattern used for rename-at-floor and outer-div title assertions"
  - "MiniPreview tests: cast to 'any' for StyledSpan[][] prop until Plan 04 updates the prop signature — keeps tests self-contained without import coupling"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-20"
  tasks_completed: 3
  tasks_total: 3
  files_created: 3
  files_modified: 4
---

# Phase 139 Plan 01: Wave 0 Red Scaffolding + A2 Verification Summary

**One-liner:** RED test scaffold for StyledSpan styled-tail contract (Go) + resolveColor/MiniPreview/TabBar chevrons (TS) + A2 headless xterm PASS verdict.

## What Was Built

### Task 1: Go RED Tests for GetSessionStyledTailLines + HTTP Handler

Added 4 RED tests to `internal/daemon/engine_test.go`:

- **TestGetSessionStyledTailLines_ColorBold** — feeds `\x1b[1;32mgreen\x1b[0m\n`; asserts result contains spans with `FG:"ansi:2"` and `Bold:true` on the first span, and that reconstructed chars contain "green".
- **TestGetSessionStyledTailLines_TUI** — feeds `"aaaa\rbbbb\n"` (carriage-return overwrite); asserts exactly one non-empty row whose text is "bbbb" (regression guard for #96 — no doubled "aaaabbbb").
- **TestGetSessionStyledTailLines_Unknown** — calls with non-existent session ID; asserts non-nil empty `[][]StyledSpan{}` (no panic, no nil).

Added 1 RED test to `internal/daemon/api_test.go`:

- **TestHandleGetSessionStyledTailLines** — creates a session, calls `GET /sessions/{id}/styled-tail?n=5`, asserts 200 + JSON decodable into `StyledTailLinesResponse` with non-nil `Lines`; also asserts `?n=999` returns 200 (clamp defense).

**Verified RED state:** `go vet ./internal/daemon/` surfaces undefined `GetSessionStyledTailLines`, `StyledSpan`, `StyledTailLinesResponse` — expected, documented.

### Task 2: Frontend RED Tests (resolveColor, MiniPreview StyledSpan, TabBar chevrons)

**Created `frontend/src/lib/vtColor.test.ts`** — 12 unit tests for `resolveColor` contract:
- `''` + `isFg=true` → `theme.foreground`
- `''` + `isFg=false` → `undefined`
- `'ansi:2'` → `theme.green` (ANSI index 2)
- `'ansi:16'` → `theme.extendedAnsi[0]` (extended color)
- `'#abcdef'` → `'#abcdef'` (hex passthrough)
- RED: imports from `'../lib/vtColor'` which does not exist until Plan 04.

**Updated `frontend/src/components/Hub/MiniPreview.test.tsx`**:
- Migrated existing `string[]` tests to `StyledSpan[][]` shapes via `renderPreviewStyled` helper.
- Added 4 new StyledSpan tests: colored span with inline color from theme, bold span with `fontWeight: 'bold'`, no `.xterm` element guard (CARD-07), two-row grid renders hub-card__preview-line rows.
- RED: `MiniPreview` doesn't accept `StyledSpan[][] | theme` props until Plan 04.

**Updated `frontend/src/components/__tests__/TabBar.test.tsx`**:
- Added `describe('Phase 139 TAB-02: TabBar chevron overflow')` — 6 source-inspection tests checking `canScrollRight`, `canScrollLeft`, `aria-label="Scroll tabs right/left"`, ResizeObserver usage, `scrollBy` call.
- Added `describe('Phase 139 TAB-03: TabBar rename-at-floor via context menu')` — DOM-render test: right-click → Rename → onRename fires without requiring `.tab__name` double-click (floor accessibility).
- Added `describe('Phase 139 TAB-03: TabBar title on outer .tab div')` — 2 tests asserting outer `.tab` div has a non-null `title` attribute containing the full tab name "claude 1". Currently RED because `title` is only on `.tab__name` (inner span).

**Verified RED state:** 12 frontend tests fail as expected (vtColor missing, MiniPreview prop mismatch, TabBar missing chevron state + outer title).

### Task 3: A2 Assumption Verification

**Node.js probe (`frontend/scripts/verify-xterm-headless.mjs`)**: Confirmed `@xterm/xterm 6.x` does NOT work in raw Node.js — `Terminal is not a constructor` error. This is expected: `@xterm/xterm` requires DOM shims (window, document, canvas). The script exits 1 with a clear fallback message.

**vitest jsdom fallback (`frontend/src/lib/xtermHeadless.verify.test.ts`)**: 3 tests, all PASS:
1. `Terminal.write()` + `serializeAsHTML()` work without `term.open()` — A2 CORE TEST
2. `serializeAsHTML()` output contains `<span>` elements with colored text
3. `Terminal.dispose()` doesn't crash after headless write

**A2 Verdict: PASS** — The jsdom environment provides sufficient DOM shims for `@xterm/xterm` headless operation. `serializeAsHTML()` returns a non-empty HTML string containing "hello" without `term.open()` ever being called. The `HTMLCanvasElement.getContext()` warning in test output is non-blocking (xterm falls back gracefully when WebGL/canvas is unavailable).

**Implication for Plan 04:** The remote-tail path in `HubBriefingModal.tsx` can safely use `new Terminal() + term.write() + serializeAsHTML() + term.dispose()` without DOM attachment in production (Chromium WebView provides full DOM; jsdom confirms the pattern works in the test environment too).

## Verification Status

### Go RED state (per plan spec)
```
go vet ./internal/daemon/ 2>&1 | grep "GetSessionStyledTailLines\|StyledTailLinesResponse"
# → multiple undefined-symbol errors — RED as expected
```

### Frontend RED state (per plan spec)
```
pnpm test -- --run src/lib/vtColor.test.ts src/components/Hub/MiniPreview.test.tsx src/components/__tests__/TabBar.test.tsx
# → 12 failed | 29 passed — RED as expected
```

### A2 PASS
```
pnpm test -- --run src/lib/xtermHeadless.verify.test.ts
# → 3 passed — A2 VERIFIED
```

## Deviations from Plan

### Auto-adapted: A2 Verification Form

**Found during:** Task 3  
**Issue:** Plan 01 specified `scripts/verify-xterm-headless.mjs` as the primary A2 artifact, but `@xterm/xterm 6.x` is not constructable in raw Node.js (`Terminal is not a constructor`). The library requires DOM environment shims.  
**Fix:** Created both the `.mjs` script (now serves as a diagnostic/probe showing WHY Node.js alone doesn't work) AND the `src/lib/xtermHeadless.verify.test.ts` vitest test (the authoritative PASS artifact). This matches the plan's stated fallback: "if the script needs a DOM, create a vitest test."  
**Files modified:** `frontend/scripts/verify-xterm-headless.mjs` (probe), `frontend/src/lib/xtermHeadless.verify.test.ts` (authoritative)  
**Commits:** 765045e6

### No Other Deviations

The plan executed exactly as written for Tasks 1 and 2. No new packages added. Test structure matches the PATTERNS.md analogs exactly.

## A2 Verdict (Explicit)

**A2: PASS** — `@xterm/xterm Terminal.write()` + `SerializeAddon.serializeAsHTML()` work without `term.open()` in vitest jsdom environment.  
**Execution form:** `src/lib/xtermHeadless.verify.test.ts` (vitest, not Node.js script)  
**Reason for form choice:** `@xterm/xterm 6.x` requires DOM environment; raw Node.js lacks `window`/`document` globals needed by the Terminal constructor.  
**Production impact:** In the Wails WebView (Chromium), full DOM is available — the headless pattern works. In Plan 04's remote-tail implementation, `term.open()` must NOT be called (no container element needed; Pitfall 5 in RESEARCH.md).

## Known Stubs

None. This plan creates only test files and verification scripts — no production code stubs.

## Threat Flags

None. Test files only; no new network endpoints, auth paths, or schema changes introduced.

## Self-Check: PASSED

Files verified to exist:
- /Users/ken/dev/agenthub/frontend/src/lib/vtColor.test.ts ✓
- /Users/ken/dev/agenthub/frontend/src/lib/xtermHeadless.verify.test.ts ✓
- /Users/ken/dev/agenthub/frontend/scripts/verify-xterm-headless.mjs ✓
- /Users/ken/dev/agenthub/internal/daemon/engine_test.go (modified) ✓
- /Users/ken/dev/agenthub/internal/daemon/api_test.go (modified) ✓
- /Users/ken/dev/agenthub/frontend/src/components/Hub/MiniPreview.test.tsx (modified) ✓
- /Users/ken/dev/agenthub/frontend/src/components/__tests__/TabBar.test.tsx (modified) ✓

Commits verified:
- ed4cd051: test(139-01): add RED Go tests for GetSessionStyledTailLines + HTTP handler ✓
- eba29dd1: test(139-01): add RED frontend tests for resolveColor, MiniPreview StyledSpan, TabBar chevrons ✓
- 765045e6: test(139-01): A2 verification — headless xterm write+serializeAsHTML PASS (jsdom) ✓
