---
phase: 113-ipad-terminal-touch-scroll
plan: 01
subsystem: frontend
tags: [frontend, ipad, terminal, xterm, touch-events, ui-03, ui-04]
requires:
  - frontend/src/components/TerminalPanel.tsx (existing mount useEffect at ~line 195)
  - @xterm/xterm 6.0.0 (Terminal.scrollLines public API)
provides:
  - frontend/src/lib/touchScrollHandler.ts → attachTouchScroll(container, term) => cleanup
  - .terminal-session-container { touch-action: pan-y } CSS rule
affects:
  - iPad Safari + iPad Chrome terminal scrollback (now drivable by single-finger drag)
  - OSC 8 link tap path (preserved — no preventDefault on sub-threshold taps)
tech-stack:
  added: []
  patterns:
    - Pure-function helper module under frontend/src/lib (matches openLink.ts, urlSafety.ts, webglProbe.ts)
    - ?raw source-grep tests for React + CSS wiring (matches existing TerminalPanel.test.tsx pattern)
    - mocked event-shape objects for jsdom (jsdom#1508 workaround) — capture handlers off addEventListener spy
key-files:
  created:
    - frontend/src/lib/touchScrollHandler.ts
    - frontend/src/lib/__tests__/touchScrollHandler.test.ts
    - frontend/src/components/__tests__/TerminalPanel.touchscroll.test.tsx
    - .planning/phases/113-ipad-terminal-touch-scroll/113-VERIFICATION.md
  modified:
    - frontend/src/components/TerminalPanel.tsx (import + 1 new useEffect, 8 lines added)
    - frontend/src/style.css (touch-action: pan-y on .terminal-session-container, 5 lines added)
decisions:
  - "Option B (explicit touch handlers) + CSS companion, per RESEARCH; Option A alone rejected (xterm v6 content not in a natively scrollable overflow region)"
  - "Cell height read LIVE on every touchmove from term._core._renderService.dimensions.css.cell.height so SHIFT+= / SHIFT+- font changes are honored — never cache"
  - "8px tap-vs-drag threshold; touchend NEVER preventDefaults so OSC 8 WebLinksAddon click path stays intact"
  - "touchmove registered passive:false (required for preventDefault on confirmed scroll); other listeners passive:true"
  - "Handler attaches to React-owned outer containerRef <div>, not anything inside .xterm, so it survives term.dispose() across session re-mounts"
  - "No new dependency — pure DOM addEventListener and existing xterm public API are sufficient"
  - "No REFACTOR commit needed — readCellHeight helper was extracted directly in GREEN step"
metrics:
  duration: "~30 min wall-clock"
  completed: 2026-05-18
  tasks: 4
  files: 6
  commits: 5
---

# Phase 113 Plan 01: iPad terminal touch-scroll Summary

**One-liner:** Adds a 117-line `attachTouchScroll` helper that wires single-finger touch drag to `xterm.Terminal.scrollLines()` on the React-owned terminal container, with a `touch-action: pan-y` CSS companion — restoring iPad scrollback that xterm.js v6 silently broke when it swapped to vscode's `SmoothScrollableElement`.

## What was built

1. **`frontend/src/lib/touchScrollHandler.ts`** (new) — pure module exporting `attachTouchScroll(container: HTMLElement, term: Terminal): () => void`. Tracks a single touch identifier, accumulates `Δy`, divides by live cell height (read from the same private xterm path used at `TerminalPanel.tsx:29`), and calls `term.scrollLines(-lines)` — negating because finger-down reveals older content. Multi-touch (`e.touches.length !== 1`) releases tracking so iOS handles pinch. `touchend` never `preventDefault`s — sub-threshold (<8px) taps reach the WebLinksAddon click handler.

2. **`frontend/src/lib/__tests__/touchScrollHandler.test.ts`** (new) — 10 unit tests covering: returns cleanup function; positive Δy → negative scrollLines; negative Δy → positive scrollLines; sub-cell-height drag is a no-op; sub-threshold tap leaves touchend alone; multi-touch bails without preventDefault; confirmed scroll DOES preventDefault; cleanup removes all 4 listeners; live cell-height read; 17px fallback when `_core` is undefined. Mocked container captures handlers off `addEventListener` spy calls (jsdom#1508 workaround per RESEARCH Open Q3).

3. **`frontend/src/components/TerminalPanel.tsx`** (edited) — added `import { attachTouchScroll } from '../lib/touchScrollHandler'` and a new `useEffect` with `[sessionId]` dep array immediately after the existing mount useEffect (so `termRef.current` is populated). The new effect attaches to `containerRef.current` (React-owned outer div, persists across `term.dispose()`) and returns the helper's cleanup.

4. **`frontend/src/style.css`** (edited) — added `touch-action: pan-y` to `.terminal-session-container`. Signals to iOS Safari that we claim single-finger vertical drag, suppressing competing page-pan; multi-touch (pinch-zoom) still falls through to the browser per the `touch-action` spec.

5. **`frontend/src/components/__tests__/TerminalPanel.touchscroll.test.tsx`** (new) — 5 source-grep tests using the existing `?raw` + `readFileSync(style.css)` pattern: import is present, `attachTouchScroll(` is called, the effect closes with `}, [sessionId])`, CSS rule contains `touch-action: pan-y`, CSS rule does NOT contain `touch-action: none`.

6. **`.planning/phases/113-ipad-terminal-touch-scroll/113-VERIFICATION.md`** (new) — full verification matrix: automated results (10+5+922 tests + clean tsc), macOS desktop-Chrome wheel-scroll smoke (manual-on-macOS), 5 iPad UAT items (all `human_needed`, physical hardware required), cross-surface parity note, source-grep gate table, gap closure path.

## Test counts

- `touchScrollHandler.test.ts`: **10/10** pass (Task 1 GREEN — commit `c9a8506`)
- `TerminalPanel.touchscroll.test.tsx`: **5/5** pass (Task 2 GREEN — commit `8f06df7`)
- Full frontend suite (`pnpm test`): **922/922** pass across 62 test files (Task 3 — no regressions)
- TypeScript typecheck (`npx tsc --noEmit`): **clean** (no errors)
- Lint: no `lint` script defined in `frontend/package.json` — gate not applicable, not introduced by this plan.

## Source-grep gates (all PASS at commit `821d7e0`)

| Gate | Result |
|------|--------|
| `scrollLines(-lines)` in `touchScrollHandler.ts` | 1 match ✓ |
| `passive: false` in `touchScrollHandler.ts` | 1 match ✓ |
| `attachTouchScroll` in `TerminalPanel.tsx` (excluding comments) | 2 matches (import + call site) ✓ |
| `touch-action:\s*pan-y` in `style.css` | matches ✓ |
| `touch-action:\s*none` inside `.terminal-session-container` block | 0 matches ✓ |

## Commits (chronological)

| # | Hash | Type | Description |
|---|------|------|-------------|
| 1 | `5745c02` | test | add failing tests for attachTouchScroll (RED) |
| 2 | `c9a8506` | feat | implement attachTouchScroll for iPad terminal scroll (GREEN) |
| 3 | `49e88d2` | test | add failing source-grep tests for touch wiring (RED) |
| 4 | `8f06df7` | feat | wire touch-scroll handler + touch-action CSS in TerminalPanel (GREEN) |
| 5 | `821d7e0` | docs | author VERIFICATION.md UAT scaffold |

TDD RED→GREEN cycle is visible for both Task 1 (commits 1→2) and Task 2 (commits 3→4). No REFACTOR commits were needed — the `readCellHeight` helper was extracted directly in the GREEN step.

## Deviations from Plan

None. The plan was executed exactly as written. Notes on minor decisions inside scope:

- **`pnpm lint` from Task 3 verify is N/A.** The frontend `package.json` defines `dev`, `build`, `preview`, `test`, `test:coverage` — there is no `lint` script. Substituted `npx tsc --noEmit` as the equivalent static-analysis gate (TypeScript compilation is the closest available correctness check; ESLint is not wired into pnpm scripts for this package). This is not new — it's a pre-existing project state. No commit needed for Task 3.
- **REFACTOR step for Task 1 was skipped.** The plan permitted skipping when not needed (`<action>` text: "skip if not needed"). The `readCellHeight(term)` helper was extracted directly in the GREEN implementation, so no separate refactor pass produced any diff.
- **Untracked items remain in working tree** (`.claire/`, `.claude/`, `bin/`, `node_modules/`, `screenshots/`). All five pre-existed before this plan (per the session-start `gitStatus` snapshot) and are unrelated to Phase 113. Left untouched per the destructive-git prohibition; they are out-of-scope from this plan's modifications.

## Verification status

- **Automated (macOS executor):** ✅ DONE — 10 + 5 + 922 unit/source-grep tests green, tsc clean.
- **Manual smoke (macOS executor — desktop Chrome wheel scroll):** ⏳ pending operator. Documented in `113-VERIFICATION.md §2` with exact steps. **Not executed by this executor** because it requires a Wails dev build session + visual confirmation; the relevant code path (xterm internal `SmoothScrollableElement`) is untouched by this plan and the unit suite confirms no React-level regression around mount/unmount of the new effect.
- **Manual UAT (physical iPad):** ⏳ pending operator. 5 `human_needed` UAT items in `113-VERIFICATION.md §3`. Macintosh hardware cannot perform these per `CONTEXT.md` and project memory `feedback_check_github_issues_during_uat`.

## Known follow-ups

1. **Physical iPad UAT must be completed before Phase 113 can be marked DONE** (`/gsd-verify-work`). 5 `human_needed` UAT items documented.
2. **v3.3 UAT-04 (iPad tap-on-link) carry-over** is probed by UAT-04 in `113-VERIFICATION.md`. Per RESEARCH Open Q2, two acceptable outcomes:
   - Fixed as side-effect (the 8px threshold + no-preventDefault-on-tap path lets the WebLinksAddon click handler fire) → bonus repair, will be noted here after iPad UAT.
   - Still broken but not regressed → carry-over remains independent of Phase 113.
   - **Status:** unknown until physical iPad UAT is run.
3. **Desktop Chrome wheel-scroll smoke** (`113-VERIFICATION.md §2`) is a 30-second visual check. Code paths involved are unchanged by this plan (xterm internal `SmoothScrollableElement` was not touched), so the risk is low — but the smoke remains a documented gate.

## Requirements satisfied

| Requirement | Where satisfied |
|-------------|-----------------|
| **UI-03** — single-finger drag scrolls scrollback on iPad Safari + Chrome | Task 1 handler (commit `c9a8506`) + Task 2 wiring (commit `8f06df7`) + Task 4 UAT scaffold (commit `821d7e0`). Physical iPad UAT pending. |
| **UI-04** — desktop wheel scroll unchanged; OSC 8 tap path preserved | Task 1 Test 5 (no preventDefault on tap) + Task 2 (CSS `pan-y`, no `touch-action: none`) + Task 3 (full-suite green, no regression) + Task 4 (desktop wheel smoke + UAT-04 carry-over probe). |

## Self-Check: PASSED

**Files asserted created/modified — all present:**

- ✓ `frontend/src/lib/touchScrollHandler.ts` — FOUND
- ✓ `frontend/src/lib/__tests__/touchScrollHandler.test.ts` — FOUND
- ✓ `frontend/src/components/__tests__/TerminalPanel.touchscroll.test.tsx` — FOUND
- ✓ `frontend/src/components/TerminalPanel.tsx` — FOUND (modified)
- ✓ `frontend/src/style.css` — FOUND (modified)
- ✓ `.planning/phases/113-ipad-terminal-touch-scroll/113-VERIFICATION.md` — FOUND

**Commits asserted exist — all present in `git log`:**

- ✓ `5745c02` — test(113-01): add failing tests for attachTouchScroll
- ✓ `c9a8506` — feat(113-01): implement attachTouchScroll for iPad terminal scroll
- ✓ `49e88d2` — test(113-01): add failing source-grep tests for touch wiring
- ✓ `8f06df7` — feat(113-01): wire touch-scroll handler + touch-action CSS in TerminalPanel
- ✓ `821d7e0` — docs(113-01): author VERIFICATION.md UAT scaffold
