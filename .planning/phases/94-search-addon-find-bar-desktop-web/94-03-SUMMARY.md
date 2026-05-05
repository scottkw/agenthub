---
phase: 94-search-addon-find-bar-desktop-web
plan: 03
subsystem: ui
tags: [phase-94, search, desktop-ui, findbar, xterm, react, addon-search, wave-2]

# Dependency graph
requires:
  - phase: 94-01
    provides: vendored @xterm/addon-search@0.16.0 + 13 RED scaffolds (5 turned GREEN here)
  - phase: 94-02
    provides: daemon.SearchConfig + Wails models.ts SearchConfig class + nested PluginSettings.searchConfig field
provides:
  - Desktop FindBar component (controlled, BEM-classed, TokyoNight)
  - isXtermFocused() helper (T-94-03 mitigation gate)
  - SearchAddon hot-swap lifecycle in TerminalPanel (symmetric with webgl/clipboard arms)
  - Focus-conditioned Cmd-F (Mac) / Ctrl-F (Win/Linux) window keydown listener
  - 100ms input debounce (catastrophic-regex DoS guard — RESEARCH Pitfall #5)
  - Toggle persistence via fire-and-forget SetPluginSettings round-trip (Pitfall #2 — no mid-open re-sync)
  - Theme.selectionBackground match highlight via decorations-undefined invariant (138-theme-safe)
affects:
  - 94-04 (perf wave: FindBar.cancel test will source-inspect handleSearchClose path)
  - 94-05 (web parity wave: web/terminal.html find bar mirrors this desktop contract)
  - 99 (PUI-03 disclosure for default regex/case/word options under Settings — independent surface)

# Tech tracking
tech-stack:
  added: []  # all packages already present (heroicons, addon-search vendored in 94-01)
  patterns:
    - "Controlled-component find bar: parent owns SearchAddon + state; component renders + emits"
    - "Hot-swap arm symmetric with Phase 93 webgl/clipboard pattern (single useEffect, specific-key dep array)"
    - "Focus-conditioned global keydown listener using a pure activeElement-contains helper"
    - "Decorations-undefined invariant for theme-aware xterm match highlight (no per-theme branching)"
    - "Source-inspection tests for runtime paths jsdom cannot exercise (canvas/WebGL/keyboard global)"

key-files:
  created:
    - frontend/src/lib/isXtermFocused.ts
    - frontend/src/components/FindBar/FindBar.tsx
    - frontend/src/components/FindBar/style.css (placeholder co-location stub)
    - frontend/src/components/__tests__/TerminalPanel.search.test.tsx
  modified:
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/style.css (Phase 94 — Find bar block + position:relative on .terminal-session-container)
    - frontend/src/lib/__tests__/isXtermFocused.test.ts (RED → GREEN)
    - frontend/src/components/FindBar/__tests__/FindBar.focus.test.tsx (RED → GREEN)
    - frontend/src/components/FindBar/__tests__/FindBar.dismiss.test.tsx (RED → GREEN)
    - frontend/src/components/FindBar/__tests__/FindBar.matchCount.test.tsx (RED → GREEN)
    - frontend/src/components/FindBar/__tests__/FindBar.persistence.test.tsx (RED → GREEN)
    - frontend/src/components/FindBar/__tests__/FindBar.visual.test.tsx (RED → GREEN)

key-decisions:
  - "FindBar lives in subdirectory frontend/src/components/FindBar/ rather than at frontend/src/components/FindBar.tsx — co-locates __tests__/ alongside the component, matches the existing scaffold layout from 94-01."
  - "FindBar is fully controlled (zero search state owned). Parent (TerminalPanel) owns SearchAddon + matchInfo + searchOptions + searchQuery. Avoids prop-drilling surprises and keeps SearchAddon co-located with the React state it drives."
  - "focusSeq counter prop (vs imperative ref handle) — re-pressing Cmd-F while bar already open bumps the seq, FindBar's mount useEffect re-fires and re-focuses input. Simpler than useImperativeHandle; matches React idiom."
  - "Nav arrow disabled state when matchCount === 0 — UI-SPEC §'Icon Buttons / DISABLED state' (acceptable interpretation of 'disabled at first/last match' since 0 matches is the only no-navigation state worth blocking)."
  - "Three new TerminalPanel.search.test.tsx tests as a separate file (rather than extending TerminalPanel.hot-swap.test.tsx) — keeps SearchAddon assertions discoverable by future Phase 94-05 web-parity work."

patterns-established:
  - "Decorations-undefined invariant: SearchAddon.findNext/findPrevious are NEVER passed a `decorations:` field, leaving xterm to use theme.selectionBackground automatically across all 138 themes (SRC-04). Enforced by source-inspection tests in both FindBar.visual and TerminalPanel.search."
  - "Three-arm hot-swap symmetry: webgl + clipboard + search hot-swap from one useEffect with specific dep keys (pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search) — adding more addons in the future follows the same pattern."
  - "Auth/state cleanup pyramid: hot-swap-off arm + mount cleanup + window listener removal — every addon added must dispose in three places."
  - "Cancel-on-close pattern (Pitfall #10): handleSearchClose clears the debounce timer + clearDecorations() + resets local state + returns focus to .xterm-helper-textarea."

requirements-completed: [SRC-01, SRC-02, SRC-04]

# Metrics
duration: 31min
completed: 2026-05-05
---

# Phase 94 Plan 03: Desktop FindBar UI + TerminalPanel Integration Summary

**Desktop find-bar: focus-conditioned Cmd-F opens a controlled `<FindBar>` overlay; SearchAddon hot-swaps alongside webgl/clipboard; toggles persist via SetPluginSettings; xterm theme.selectionBackground highlights via decorations-undefined invariant.**

## Performance

- **Duration:** ~31 min
- **Started:** 2026-05-05T14:31:00Z (approx, post worktree branch check)
- **Completed:** 2026-05-05T15:02:08Z
- **Tasks:** 2 (both auto, both TDD-style with RED scaffolds turned GREEN)
- **Files created:** 4 (isXtermFocused.ts, FindBar.tsx, FindBar/style.css, TerminalPanel.search.test.tsx)
- **Files modified:** 8 (TerminalPanel.tsx, style.css, 6 test scaffolds RED→GREEN)
- **New lines:** ~1232 (Task 1: 839, Task 2: 393)

## Accomplishments

- Desktop SRC-01 + SRC-02 + SRC-04 closed (web parity remains for Plan 94-05).
- 5 of 13 phase-94 RED scaffolds turned GREEN (isXtermFocused + FindBar.focus + FindBar.dismiss + FindBar.matchCount + FindBar.persistence + FindBar.visual = 6 files; 23+ assertions).
- 21 new TerminalPanel.search.test.tsx source-inspection assertions pin every Phase 94 invariant for future regression detection.
- T-94-03 (browser-find pre-emption) mitigated in code via `isXtermFocused(containerRef.current)` gate. Verified by source-inspection assertion.
- T-94-04 (regex DoS) mitigated by 100ms debounce + addon's default 1000-result highlight cap. handleSearchClose cancels pending debounce + clearDecorations.
- Decorations-undefined invariant for SRC-04 enforced by TWO source-inspection tests (FindBar.visual + TerminalPanel.search) — no `decorations:` site exists in either file. Theme.selectionBackground works automatically across all 138 themes.
- Zero new colors, zero new spacing tokens, zero new typography sizes — UI-SPEC visual contract honored verbatim.

## Task Commits

Each task was committed atomically:

1. **Task 1: FindBar component + isXtermFocused helper + Phase 94 CSS** — `6115329` (feat)
2. **Task 2: Wire SearchAddon + FindBar into TerminalPanel** — `7a239c2` (feat)

_Plan 94-03 has no separate metadata commit — orchestrator owns STATE.md / ROADMAP.md after the wave completes._

## Files Created/Modified

**Created:**
- `frontend/src/lib/isXtermFocused.ts` — pure activeElement-inside-container helper (14 lines)
- `frontend/src/components/FindBar/FindBar.tsx` — controlled find-bar React component (238 lines)
- `frontend/src/components/FindBar/style.css` — placeholder co-location stub (CSS lives in global style.css)
- `frontend/src/components/__tests__/TerminalPanel.search.test.tsx` — 21 source-inspection assertions

**Modified:**
- `frontend/src/components/TerminalPanel.tsx` — added SearchAddon imports, refs, state, hot-swap arm, mount cleanup, Cmd-F window listener, 5 callback handlers, conditional FindBar render (468 lines, was 308)
- `frontend/src/style.css` — appended Phase 94 — Find bar block (~130 lines of BEM); added position:relative to .terminal-session-container
- `frontend/src/lib/__tests__/isXtermFocused.test.ts` — RED → GREEN (4 tests)
- `frontend/src/components/FindBar/__tests__/FindBar.focus.test.tsx` — RED → GREEN (4 tests)
- `frontend/src/components/FindBar/__tests__/FindBar.dismiss.test.tsx` — RED → GREEN (3 tests)
- `frontend/src/components/FindBar/__tests__/FindBar.matchCount.test.tsx` — RED → GREEN (4 tests)
- `frontend/src/components/FindBar/__tests__/FindBar.persistence.test.tsx` — RED → GREEN (5 tests)
- `frontend/src/components/FindBar/__tests__/FindBar.visual.test.tsx` — RED → GREEN (8 tests)

## Decisions Made

1. **FindBar at `components/FindBar/FindBar.tsx`, not `components/FindBar.tsx`** — co-locates `__tests__/` with the component (matches Wave 0 scaffold layout). UI-SPEC line 447 says the file path is `frontend/src/components/FindBar.tsx` — the planner explicitly noted (94-03 Action Step 0) that the subdirectory layout is acceptable as long as the same WebGLRecoveryBanner content pattern is followed.
2. **Fully controlled FindBar** — UI-SPEC §"Component Inventory" line 447 states the contract; FindBar owns no search state. Parent (TerminalPanel) owns SearchAddon + matchInfo + searchOptions + searchQuery. The only React state inside FindBar is the inputRef for auto-focus.
3. **`focusSeq` prop counter** for Cmd-F-while-already-open re-focus — bumped by TerminalPanel's keydown handler whenever Cmd-F fires (whether bar is opening or already open). FindBar's `useEffect(() => inputRef.current?.focus(), [focusSeq])` re-fires on bump. Simpler than `useImperativeHandle`.
4. **Nav arrow `disabled` when matchCount === 0** — UI-SPEC §"DISABLED state" specifies "at first or last match" but the only no-navigation state worth blocking with `disabled` styling is the no-results state; SearchAddon wraps natively at first/last. This is the tightest interpretation of the visual disabled affordance.
5. **`new daemon.PluginSettings({...pluginConfig, searchConfig: opts})` in handleSearchOptionsChange** — pluginConfig is a class instance from Wails models.ts. Spreading it into a plain object would lose the SearchConfig sub-class shape; constructing a new PluginSettings instance keeps the wire-format invariant intact for the SetPluginSettings RPC.
6. **TerminalPanel.search.test.tsx as a separate file** (not appended to TerminalPanel.hot-swap.test.tsx) — keeps Phase 94 SearchAddon assertions discoverable by future 94-05 web-parity grep + matches existing convention of feature-scoped test files.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Frontend node_modules missing in worktree**
- **Found during:** Pre-Task-1 setup (worktree was freshly created with no symlinked node_modules).
- **Issue:** `pnpm exec vitest` and `tsc` could not resolve any imports; addon-search package not installed.
- **Fix:** Ran `pnpm --filter ./frontend install` (2 seconds, all packages already cached).
- **Files modified:** none (install populates node_modules; not committed).
- **Verification:** `pnpm exec tsc --noEmit` exits 0; `pnpm exec vitest run` exits with only pre-existing/expected-RED failures.
- **Committed in:** N/A (install side-effect, no source changes).

**2. [Rule 2 - Missing critical] Added `position: relative` to `.terminal-session-container`**
- **Found during:** Task 1 (CSS authoring).
- **Issue:** UI-SPEC §"Layout and Positioning" requires the find-bar to absolutely-position inside `.terminal-session-container`, but the existing CSS rule had only `padding: 8px; overflow: hidden` — without `position: relative`, the absolutely-positioned find-bar would anchor to the next ancestor with positioning (likely `.terminal-container` two levels up, or the viewport), causing the bar to appear in the wrong place.
- **Fix:** Added `position: relative` to the existing `.terminal-session-container` block with a Phase 94 comment marker.
- **Files modified:** `frontend/src/style.css`.
- **Verification:** `grep -c "position: relative" frontend/src/style.css` ≥ 1 inside the .terminal-session-container block; visual UAT will be run by 94-VERIFICATION.md.
- **Committed in:** `6115329` (Task 1 commit).

**3. [Rule 1 - Bug] Fixed visual.test.tsx assertion for `spellcheck`**
- **Found during:** First Task-1 vitest run (one of 23 tests failed).
- **Issue:** Test asserted `input.spellcheck === false` (property), but jsdom does not reflect the React-rendered `spellCheck={false}` prop onto the property — it only sets the HTML attribute `spellcheck="false"`.
- **Fix:** Changed assertion to `input.getAttribute('spellcheck') === 'false'`. Documented why in test comment.
- **Files modified:** `frontend/src/components/FindBar/__tests__/FindBar.visual.test.tsx`.
- **Verification:** `pnpm exec vitest run src/components/FindBar/__tests__/FindBar.visual.test.tsx` — 8/8 pass.
- **Committed in:** `6115329` (Task 1 commit).

**4. [Rule 1 - Bug] Loosened source-inspection regex for conditional FindBar render**
- **Found during:** First Task-2 vitest run.
- **Issue:** Test used `/findBarOpen\s*&&\s*pluginConfig\?\.search\s*&&\s*<FindBar/` but the actual JSX has `findBarOpen && pluginConfig?.search && (\n  <FindBar` — the parenthesis on the next line broke the match.
- **Fix:** Updated regex to `/findBarOpen\s*&&\s*pluginConfig\?\.search\s*&&\s*\(?\s*<FindBar/` — optionally matches the opening `(` and trailing whitespace.
- **Files modified:** `frontend/src/components/__tests__/TerminalPanel.search.test.tsx`.
- **Verification:** Re-ran; 21/21 source-inspection tests pass.
- **Committed in:** `7a239c2` (Task 2 commit).

---

**Total deviations:** 4 auto-fixed (1 blocking, 1 missing critical, 2 test assertion bugs in newly-authored code).
**Impact on plan:** All four were necessary to complete the planned scope. Three were authoring slips fixed in the same commit they were introduced; one (node_modules install) was a worktree-environment concern that did not change source. No scope creep.

## Issues Encountered

- **jsdom `Not implemented: HTMLCanvasElement's getContext()` warnings** during vitest runs — expected; xterm.js mounts a `<canvas>` for renderer-init. These are not failures; they appear in stdout of any test that imports `TerminalPanel` (even via `?raw` source-inspection — vitest still resolves the component module's transitive imports). Did not affect any assertion.
- **Pre-existing Sidebar.test.tsx 20 jsdom localStorage failures** (per memory note `feedback_verify_test_env_before_declaring_failure.md` and the parallel-execution prompt) — predate Phase 94 and are out of scope. Not addressed.
- **`FindBar.cancel.test.tsx` and `FindBar.themeMatrix.test.tsx` remain RED** — per success criteria, these are reserved for Plans 94-04 (cancel-on-close source-inspection) and 94-05 (web parity SearchAddon construction). Cancel-on-close behavior IS implemented in this plan's `handleSearchClose` (clearTimeout + clearDecorations) — only the test scaffold turn-GREEN is deferred to 94-04 to keep that wave's deliverable clean.

## Test Outcomes

- **23 tests** turned GREEN across 5 RED scaffolds (Task 1).
- **5 tests** turned GREEN in FindBar.persistence.test.tsx (Task 2).
- **21 source-inspection tests** added in TerminalPanel.search.test.tsx (all GREEN).
- **49 total Phase-94 assertions added/turned GREEN** in this plan.
- **tsc --noEmit:** exits 0.
- **Full vitest suite:** 580 passed / 22 failed; the 22 failures are 20 pre-existing Sidebar localStorage tests + 2 expected RED scaffolds for plans 94-04/94-05.

## Threat Mitigation Status

| Threat ID | Disposition | Status After This Plan |
|-----------|-------------|------------------------|
| T-94-03 (Tampering — Cmd-F intercepts when not focused) | mitigate | **DONE** — `isXtermFocused(containerRef.current)` gate in TerminalPanel keydown handler; verified by source-inspection assertion. |
| T-94-04 (DoS — regex backtracking) | mitigate | **DONE (desktop)** — 100ms debounce in handleSearchQueryChange + handleSearchClose cancels pending debounce + clearDecorations. Default highlightLimit=1000 inherited (RESEARCH §"SearchAddon API Contract"). FindBar.cancel test will pin this in Plan 94-04. |
| T-94-02 (toggle race vs PluginsSection) | accept | UNCHANGED — pre-existing race; daemon settings:plugins event re-syncs. |
| T-94-05 (search query exfiltration) | accept | UNCHANGED — query never leaves browser; daemon receives only SearchConfig booleans. |

## Next Phase Readiness

- **Plan 94-04 (perf wave):** all source-inspection hooks for FindBar.cancel are in place — handleSearchClose contains both `clearTimeout(debounceTimerRef.current)` and `searchAddonRef.current?.clearDecorations()`. The FindBar.cancel test scaffold can grep `TerminalPanel.tsx?raw` for these strings and turn GREEN with no production code changes.
- **Plan 94-05 (web parity):** desktop contract is locked. Web `web/terminal.html` + `web/assets/terminal.js` should mirror the same pattern: `new SearchAddon()` (no options), `addon.onDidChangeResults(e => updateCount(e))`, no `decorations:` ever passed to findNext/findPrevious. The FindBar.themeMatrix test scaffold will source-inspect both files for the `decorations:` invariant.
- **Plan 99 (PUI-03):** advanced disclosure for default regex/case/word options is unblocked — `daemon.SearchConfig` is already round-tripping; PluginsSection just needs a `<details>` block that reads/writes `pluginConfig.searchConfig` via SetPluginSettings (same pattern as the toggle row already in PluginsSection.tsx).

## Self-Check: PASSED

- `frontend/src/lib/isXtermFocused.ts` — exists ✓
- `frontend/src/components/FindBar/FindBar.tsx` — exists ✓
- `frontend/src/components/FindBar/style.css` — exists ✓
- `frontend/src/components/__tests__/TerminalPanel.search.test.tsx` — exists ✓
- Commit `6115329` (Task 1) — found in git log ✓
- Commit `7a239c2` (Task 2) — found in git log ✓
- 23 isXtermFocused/FindBar GREEN tests pass; 21 TerminalPanel.search source-inspection tests pass; 5 FindBar.persistence tests pass — verified by direct vitest invocation ✓
- tsc --noEmit exits 0 — verified ✓
- FindBar.cancel.test.tsx and FindBar.themeMatrix.test.tsx remain RED — verified ✓
- No `decorations:` substring exists in FindBar.tsx or TerminalPanel.tsx — verified by grep ✓

---
*Phase: 94-search-addon-find-bar-desktop-web*
*Plan: 03*
*Completed: 2026-05-05*
