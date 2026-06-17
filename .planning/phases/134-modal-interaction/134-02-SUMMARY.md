---
phase: 134-modal-interaction
plan: "02"
subsystem: frontend/hub
tags: [modal, card-click, stopPropagation, keyboard-a11y, coexistence]
dependency_graph:
  requires: ["134-01"]
  provides: ["onCardClick contract emitted from SessionCard, threaded through SessionCardGrid"]
  affects: ["frontend/src/components/Hub/SessionCard.tsx", "frontend/src/components/Hub/SessionCardGrid.tsx"]
tech_stack:
  added: []
  patterns: ["source-inspection TDD via ?raw import", "defense-in-depth event guard with .closest()", "stopPropagation for coexistence"]
key_files:
  created: []
  modified:
    - "frontend/src/components/Hub/SessionCard.tsx"
    - "frontend/src/components/Hub/SessionCardGrid.tsx"
    - "frontend/src/components/Hub/SessionCard.test.tsx"
decisions:
  - "Defense-in-depth: both article onClick guard AND button stopPropagation protect coexistence (T-134-02-01)"
  - "onCardClick fires on Enter and Space key events per A11Y-02 partial requirement"
metrics:
  duration: "~4 minutes"
  completed: "2026-06-17T14:10:28Z"
  tasks: 2
  files_modified: 3
---

# Phase 134 Plan 02: SessionCard onCardClick + stopPropagation Fixes Summary

Wire the card-click → modal trigger contract through SessionCard and SessionCardGrid, fixing both missing stopPropagation calls so the Phase 131 re-attach Open button coexists with the new card-click modal interaction.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Failing source-inspection tests for onCardClick + stopPropagation | 631f9369 | SessionCard.test.tsx |
| 1 (GREEN) | SessionCard onCardClick prop + body click/keyboard + stopPropagation fixes | 7e1ee7cd | SessionCard.tsx |
| 2 | SessionCardGrid threads onCardClick through both render paths | 32bcc7b6 | SessionCardGrid.tsx |

## What Was Built

**SessionCard.tsx:**
- Added `onCardClick?: (session: SessionInfo, rect: DOMRect) => void` to `SessionCardProps`
- Added article `onClick` handler with defense-in-depth guards: returns early if click target is inside `.hub-card__open`, `.hub-card__menu-btn`, `.hub-card__menu`, `.InlineSessionName input`, or if `isDragging` is true; otherwise fires `onCardClick` with the session and `getBoundingClientRect()`
- Added article `onKeyDown` handler: fires `onCardClick` on `Enter` or `' '` (Space) after `e.preventDefault()` (A11Y-02 partial)
- **Fixed Open button**: `onClick={() => onOpenSession(id, name, cli)}` → `onClick={(e) => { e.stopPropagation(); onOpenSession(id, name, cli) }}` — Phase 131 coexistence guarantee
- **Fixed menu button**: `onClick={() => setMenuOpen((p) => !p)}` → `onClick={(e) => { e.stopPropagation(); setMenuOpen((p) => !p) }}` — Phase 131 coexistence guarantee
- Added 7 source-inspection tests using `?raw` import confirming all click-contract invariants

**SessionCardGrid.tsx:**
- Added `onCardClick?: (session: SessionInfo, rect: DOMRect) => void` to `SessionCardGridProps`
- Destructured `onCardClick` in component
- Passed `onCardClick={onCardClick}` to `<SessionCard>` in both render paths:
  - Named-group render path (Phase 132, `groupDefs` non-empty)
  - workDir-group render path (Phase 131 fallback)

## Acceptance Criteria Verification

- `grep -c "e.stopPropagation()" SessionCard.tsx` (filtered: no comments) = **2** ✓
- `grep -c "onCardClick={onCardClick}" SessionCardGrid.tsx` = **2** (both render paths) ✓
- `SessionCardGridProps` source contains `onCardClick?: (session: SessionInfo, rect: DOMRect) => void` ✓
- `pnpm exec tsc --noEmit` = no errors ✓
- SessionCard tests: **92/92 pass** (85 pre-existing + 7 new source-inspection) ✓
- SessionCardGrid tests: **37/37 pass** ✓
- Full suite: 100 files / 1646 tests pass; 4 failing test files are **intentionally red** from 134-01 scaffolding (HubModal, HubInteractiveModal, HubBriefingModal, style.hub.modal) — their targets are implemented in plans 134-03/04

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — this plan emits only the click contract. No modal component is mounted; the `onCardClick` callback is wired from `HubPanel` in plan 134-05.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. All changes are frontend event-handling only.

## Self-Check: PASSED

- SessionCard.tsx: FOUND
- SessionCardGrid.tsx: FOUND
- SessionCard.test.tsx: FOUND
- Commit 631f9369 (RED tests): FOUND
- Commit 7e1ee7cd (GREEN impl): FOUND
- Commit 32bcc7b6 (Grid thread): FOUND
