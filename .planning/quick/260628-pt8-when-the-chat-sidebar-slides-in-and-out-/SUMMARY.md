---
quick_id: 260628-pt8
slug: when-the-chat-sidebar-slides-in-and-out-
date: 2026-06-28
status: complete
tags: [css, animation, chat, polish]
key_files:
  modified:
    - frontend/src/style.css
    - frontend/src/components/Hub/chatToggleOverlap.test.ts
    - TESTING.md
commits:
  - d2053f7b
  - 5e7c27c8
duration: ~4 minutes
---

# Quick Task 260628-pt8: Chat toggle button should slide with the sidebar, not jump

## One-liner

Added `transition: right 220ms ease-out` to `.hub-modal__chat-toggle` base rule so the toggle glides in lockstep with the drawer slide instead of teleporting.

## What Was Done

### Task 1 — CSS fix + regression gate (commit d2053f7b)

**frontend/src/style.css** (~line 6028): Added `transition: right 220ms ease-out;` to the base `.hub-modal__chat-toggle` rule. This mirrors the drawer's `transition: transform 220ms ease-out` so both elements animate at the same speed and easing — the button now glides right when the drawer opens and glides back left when it closes, on all three surfaces that render it (Hub interactive modal, web-share guest, terminal-tab chat host).

**frontend/src/components/Hub/chatToggleOverlap.test.ts**: Added test `(d)` asserting the base `.hub-modal__chat-toggle` rule body matches `/transition\s*:\s*right\b/`, reusing the base-rule capture from test `(c)`. All four tests (a)/(b)/(c)/(d) pass.

### Task 2 — TESTING.md Suite Manifest note (commit 5e7c27c8)

Added a `> Note:` entry at the top of the Suite Manifest notes block documenting: no new test files, extension of `chatToggleOverlap.test.ts` with test `(d)`, CSS-only change in `style.css`. Counts unchanged (366 Go / 132 vitest / 9 Playwright / 509 total).

## Verification

- `npx vitest run src/components/Hub/chatToggleOverlap.test.ts` — 4/4 passed
- `npx tsc --noEmit` — clean
- `bash tests/check-traceability-paths.sh` — exits 0

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- style.css base toggle rule has `transition: right 220ms ease-out;` — FOUND
- chatToggleOverlap.test.ts test (d) added — FOUND
- TESTING.md Suite Manifest note added — FOUND
- Commits d2053f7b and 5e7c27c8 exist — VERIFIED
