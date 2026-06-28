---
type: quick
quick_id: 260628-pt8
slug: when-the-chat-sidebar-slides-in-and-out-
date: 2026-06-28
files_modified:
  - frontend/src/style.css
  - frontend/src/components/Hub/chatToggleOverlap.test.ts
  - TESTING.md
---

# Quick Task: Chat toggle button should slide with the sidebar, not jump

## Problem

When the chat drawer opens/closes it slides via `.chat-panel { transition: transform 220ms ease-out }`
(style.css ~6694). The floating chat toggle button relocates between its closed position
(`.hub-modal__chat-toggle { right: 12px }`, ~6031) and its open position
(`.chat-panel--open ~ .hub-modal__chat-toggle { right: calc(var(--chat-panel-width, 360px) + 12px) }`, ~6060),
but the toggle has **no `transition`** declared, so its `right` value changes instantly — the button
teleports to the new spot while the drawer is still mid-slide. Visually jarring / out of sync.

## Fix

Add a `right` transition to the base `.hub-modal__chat-toggle` rule that mirrors the drawer's slide
exactly (`220ms ease-out`), so the toggle glides in lockstep with the opening/closing drawer on all
three surfaces that render it (Hub interactive modal, web-share guest, terminal-tab chat host — they
all share this one selector via the general-sibling combinator).

## Tasks

### Task 1: Add the slide transition to the toggle + source-gate test
- **style.css**: in the base `.hub-modal__chat-toggle` rule (~6028), add
  `transition: right 220ms ease-out;` — matching `.chat-panel`'s `transition: transform 220ms ease-out`
  so the button and drawer move together. Do NOT touch the `.chat-panel--open ~ ...` offset rule or the
  closed-state `right: 12px` (both must stay exactly as-is — they are covered by existing source-gate
  tests (b) and (c) in chatToggleOverlap.test.ts).
- **chatToggleOverlap.test.ts**: add test `(d)` asserting the base `.hub-modal__chat-toggle` rule body
  declares a `transition` on `right` (e.g. matches `/transition\s*:\s*right\b/`) so the slide can't
  silently regress. Reuse the existing base-rule capture pattern from test (c).
- **verify**: `cd frontend && npx vitest run src/components/Hub/chatToggleOverlap.test.ts` passes;
  `npx tsc --noEmit` clean.

### Task 2: Register the test change in TESTING.md (standing regression convention)
- Add a Suite Manifest `> Note:` entry documenting this quick task: no new test files; EXTENDED
  `frontend/src/components/Hub/chatToggleOverlap.test.ts` with test (d) (toggle `right` transition gate).
  No new traceability row needed (polish under existing CHAT-FIX-01 / CHAT-LAYOUT-02 toggle behavior).
- **verify**: `bash tests/check-traceability-paths.sh` exits 0.

## Done when
- The chat toggle button animates its `right` offset over 220ms ease-out, sliding with the drawer
  instead of jumping, on open and close.
- chatToggleOverlap.test.ts (a)/(b)/(c) still pass and new (d) passes; tsc clean; traceability green.
