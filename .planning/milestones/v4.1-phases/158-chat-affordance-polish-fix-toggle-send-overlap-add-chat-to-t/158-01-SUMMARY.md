---
phase: 158-chat-affordance-polish-fix-toggle-send-overlap-add-chat-to-t
plan: "01"
subsystem: ui
tags: [css, chat, hub-modal, web-share, source-gate, vitest]

requires:
  - phase: 154-session-chat-ui-desktop
    provides: HubInteractiveModal chat toggle + ChatPanel DOM structure (sibling order that the combinator matches)
  - phase: 155-session-chat-web-share-parity
    provides: WebShareSessionView uses same .hub-modal__chat-toggle + .chat-panel--open classes (unscoped rule corrects both surfaces)

provides:
  - CSS relocation rule .chat-panel--open ~ .hub-modal__chat-toggle { right: 372px } — toggle moves clear of the 360px chat drawer while open
  - vitest source-gate chatToggleOverlap.test.ts — 3 assertions: selector present, right:372px in relocation block, right:12px in base block
  - TESTING.md registration: Section 2 manifest (vitest 129→130, total 505→506), Section 4 CHAT-FIX-01 traceability row, Section 5 M-29 manual item

affects:
  - 158-02-PLAN.md (terminal-tab chat affordance, CHAT-PARITY-01)
  - Any future phase touching HubInteractiveModal or WebShareSessionView CSS

tech-stack:
  added: []
  patterns:
    - "CSS general-sibling combinator for state-driven layout: .state-class ~ .affected-element { override }"
    - "Unscoped relocation rule corrects multiple surfaces sharing the same CSS classes without markup change"
    - "readFileSync source-gate pattern (style.css) consistent with style.hub.test.ts"

key-files:
  created:
    - frontend/src/components/Hub/chatToggleOverlap.test.ts
  modified:
    - frontend/src/style.css
    - TESTING.md

key-decisions:
  - "Use general-sibling combinator (.chat-panel--open ~ .hub-modal__chat-toggle) not a scoped parent selector — ChatPanel renders BEFORE the toggle as a same-parent sibling, and leaving the rule unscoped corrects the identical latent overlap on the web-share surface for free (parity-positive)"
  - "right: 372px = 360px drawer width + 12px gutter, preserving the same 12px clearance the toggle has in its closed-state position"
  - "readFileSync pattern over ?raw import for CSS source-gates — Vite/vitest ?raw on CSS returns empty string in jsdom; readFileSync is the established pattern in this codebase (style.hub.test.ts)"

patterns-established:
  - "CSS source-gates for layout rules: readFileSync + regex on selector block, not render-based geometry assertions"

requirements-completed: [CHAT-FIX-01]

coverage:
  - id: D1
    description: "CSS relocation rule .chat-panel--open ~ .hub-modal__chat-toggle { right: 372px } ships in style.css and no markup changes"
    requirement: CHAT-FIX-01
    verification:
      - kind: unit
        ref: frontend/src/components/Hub/chatToggleOverlap.test.ts#(a) stylesheet contains the relocation selector
        status: pass
      - kind: unit
        ref: frontend/src/components/Hub/chatToggleOverlap.test.ts#(b) relocation rule body sets right: 372px
        status: pass
      - kind: unit
        ref: frontend/src/components/Hub/chatToggleOverlap.test.ts#(c) base .hub-modal__chat-toggle rule still sets right: 12px
        status: pass
    human_judgment: false
  - id: D2
    description: "Toggle relocates clear of the composer Send/Inject button (rendered geometric non-overlap) and still closes the drawer when clicked"
    requirement: CHAT-FIX-01
    verification: []
    human_judgment: true
    rationale: "JSDOM performs no layout; rendered pixel overlap cannot be measured in vitest. Source-gate only proves the CSS rule's presence and offset. Visual confirmation requires a live browser with a real layout engine (M-29)."

duration: 10min
completed: 2026-06-27
status: complete
---

# Phase 158 Plan 01: Chat Toggle / Send Overlap Fix Summary

**Additive CSS general-sibling rule relocates .hub-modal__chat-toggle to right:372px while the chat drawer is open, clearing the Send/Inject button; 3-assertion vitest source-gate proves rule presence and offset**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-06-27T15:36:00Z
- **Completed:** 2026-06-27T15:46:44Z
- **Tasks:** 2
- **Files modified:** 3 (style.css, chatToggleOverlap.test.ts [new], TESTING.md)

## Accomplishments

- Added `.chat-panel--open ~ .hub-modal__chat-toggle { right: 372px }` to `frontend/src/style.css` after the `:hover` rule — toggle now relocates 12px left of the 360px drawer's left edge while the drawer is open, no longer obscuring the composer Send/Inject button
- The rule is intentionally unscoped (no `.hub-modal__body--interactive` parent) so it corrects the identical latent overlap on the web-share surface (WebShareSessionView.tsx) at no extra cost — parity-positive
- Created `frontend/src/components/Hub/chatToggleOverlap.test.ts`: 3 source-gate assertions using `readFileSync` (consistent with `style.hub.test.ts`) — all 3 pass
- Registered in TESTING.md: Section 2 delta note + count bump (vitest 129→130, total 505→506); Section 4 CHAT-FIX-01 traceability row; Section 5 Category Q / M-29 manual item

## Task Commits

1. **Task 1: Add toggle-relocation CSS rule + vitest source-gate** - `415be4b7` (fix)
2. **Task 2: Register CHAT-FIX-01 in TESTING.md** - `0efa8239` (docs)

## Files Created/Modified

- `frontend/src/style.css` — additive relocation rule `.chat-panel--open ~ .hub-modal__chat-toggle { right: 372px }` with CHAT-FIX-01 comment
- `frontend/src/components/Hub/chatToggleOverlap.test.ts` — new vitest source-gate (3 assertions)
- `TESTING.md` — Section 2 manifest delta + count bump; Section 4 CHAT-FIX-01 row; Section 5 M-29 Category Q

## Decisions Made

- Used general-sibling combinator rather than a scoped parent selector because ChatPanel always renders BEFORE the toggle button as same-parent siblings in both HubInteractiveModal and WebShareSessionView. Leaving the rule unscoped corrects the web-share surface for free.
- `right: 372px` = 360px drawer width + 12px gutter (same clearance as the closed-state 12px margin).
- Used `readFileSync` pattern for CSS source-gate, not `?raw` import — confirmed that Vite/vitest returns empty string for `style.css?raw` in jsdom environment; `readFileSync` is the established pattern in this codebase.

## Deviations from Plan

**1. [Rule 1 - Bug] Changed CSS import strategy from `?raw` to `readFileSync`**
- **Found during:** Task 1 verification (test run)
- **Issue:** Plan specified `import css from '../../style.css?raw'` but vitest/jsdom returns empty string `''` for CSS `?raw` imports; all 3 assertions failed
- **Fix:** Switched to `import { readFileSync } from 'fs'; const css = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')` — matching the established pattern in `style.hub.test.ts`
- **Files modified:** `frontend/src/components/Hub/chatToggleOverlap.test.ts`
- **Verification:** All 3 source-gate tests pass after fix
- **Committed in:** 415be4b7 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - Bug, CSS import strategy mismatch)
**Impact on plan:** Necessary correction — the `?raw` approach works for `.tsx` files but not for CSS in jsdom. Fix aligns the test with the established codebase pattern and has no scope impact.

## Issues Encountered

None beyond the CSS `?raw` import behavior documented under Deviations.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- CHAT-FIX-01 complete; the toggle/Send overlap is fixed with source proof
- 158-02 (CHAT-PARITY-01) can proceed: add "Chat" affordance to terminal session tabs (tab-bar parity with Hub modal and web-share)
- M-29 manual verification item recorded for UAT sign-off

## Self-Check: PASSED

- `frontend/src/components/Hub/chatToggleOverlap.test.ts` — EXISTS
- `frontend/src/style.css` — EXISTS and contains `.chat-panel--open ~ .hub-modal__chat-toggle`
- Commit 415be4b7 — EXISTS (fix: add toggle-relocation CSS rule + vitest source-gate)
- Commit 0efa8239 — EXISTS (docs: register CHAT-FIX-01 in TESTING.md)
- `bash tests/check-traceability-paths.sh` — PASSES (OK: all traceability paths exist)

---
*Phase: 158-chat-affordance-polish-fix-toggle-send-overlap-add-chat-to-t*
*Completed: 2026-06-27*
