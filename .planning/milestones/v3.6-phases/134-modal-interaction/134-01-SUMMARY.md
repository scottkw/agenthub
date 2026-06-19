---
phase: 134-modal-interaction
plan: 01
subsystem: testing
tags: [vitest, source-inspection, css-contract, hub-modal, tdd, wave-0]

# Dependency graph
requires:
  - phase: 133-attention-pulse
    provides: "Hub attention CSS pattern (hub-attn-pulse) used as template for modal animation guard"
provides:
  - "HubModal.test.tsx: source-inspection RED tests for MODAL-01/02/03 (role=dialog, Escape, routing)"
  - "HubInteractiveModal.test.tsx: source-inspection RED tests for MODAL-03/05 (TerminalPanel props, isActive gate)"
  - "HubBriefingModal.test.tsx: source-inspection RED tests for MODAL-04 (tail fetch, maxLength, Send Response)"
  - "style.hub.modal.test.ts: CSS contract RED tests for overlay z-index:200, keyframes, reduced-motion guard, token-only panel"
affects: [134-02, 134-03, 134-04, 134-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Source-inspection via ?raw import for components that cannot be DOM-rendered (xterm canvas absence in jsdom)"
    - "CSS block-finder pattern: indexOf('.class') + indexOf('}', idx) → slice → assert substring"
    - "Intentional RED state: test files reference non-existent implementation as Nyquist compliance strategy"

key-files:
  created:
    - frontend/src/components/Hub/HubModal.test.tsx
    - frontend/src/components/Hub/HubInteractiveModal.test.tsx
    - frontend/src/components/Hub/HubBriefingModal.test.tsx
    - frontend/src/components/__tests__/style.hub.modal.test.ts
  modified: []

key-decisions:
  - "Wave-0 test-first strategy: test files created before implementation so downstream plans have a red→green target (Nyquist compliance)"
  - "All component tests use ?raw import source inspection — no render() or xterm import (canvas APIs absent in jsdom)"
  - "CSS test uses readFileSync + indexOf block-finder, not bare .includes(), to prevent self-invalidating comments from passing"
  - "style.hub.modal.test.ts created as separate file (not appended to style.hub.test.ts) per PLAN.md spec for discoverability"

patterns-established:
  - "Hub modal test pattern: ?raw import + describe/it blocks named by MODAL-0x requirement ID"
  - "CSS contract block-finder: indexOf('.hub-modal-overlay') + indexOf('}', idx) → slice → assert"

requirements-completed: [MODAL-01, MODAL-02, MODAL-03, MODAL-04, MODAL-05, MODAL-06]

# Metrics
duration: 5min
completed: 2026-06-17
---

# Phase 134 Plan 01: Modal Test Scaffolding Summary

**Wave-0 source-inspection test files for HubModal/HubInteractiveModal/HubBriefingModal and modal CSS contract — 4 RED test files, 13 failing assertions against not-yet-implemented targets**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-06-17T14:00:00Z
- **Completed:** 2026-06-17T14:04:05Z
- **Tasks:** 2
- **Files modified:** 4 (all created)

## Accomplishments

- Created 3 component source-inspection test files in `frontend/src/components/Hub/` using `?raw` import pattern — all correctly fail with "Failed to resolve import" because target `.tsx` files do not yet exist
- Created 1 CSS contract test file in `frontend/src/components/__tests__/` using `readFileSync` + `indexOf` block-finder pattern — 12 assertions fail (CSS not yet written), 1 passes (existing reduced-motion block already has `animation: none`)
- Pre-existing suite remained green: 1639 tests pass, 13 new intentionally-failing tests added

## Task Commits

1. **Task 1+2: Component tests + CSS contract test** - `00b899d9` (test)

**Plan metadata:** (see final commit below)

## Files Created/Modified

- `frontend/src/components/Hub/HubModal.test.tsx` - Source inspection for MODAL-01/02/03: role=dialog, aria-modal, transformOrigin, Escape, stopImmediatePropagation, cardFocusRef, isAttentionStatus routing to HubBriefingModal/HubInteractiveModal
- `frontend/src/components/Hub/HubInteractiveModal.test.tsx` - Source inspection for MODAL-03/05: TerminalPanel props (sessionId, isActive, relayPort, theme, pluginConfig) + isActive phase gate regex
- `frontend/src/components/Hub/HubBriefingModal.test.tsx` - Source inspection for MODAL-04: GetSessionTailLines, RelayClient, sendInput, onOpen, maxLength={4096}, responseText.trim(), Send Response copy
- `frontend/src/components/__tests__/style.hub.modal.test.ts` - CSS contract for overlay (position:fixed/inset:0/z-index:200), panel (var tokens/no hex), 4 keyframes, reduced-motion guard, send-btn accent, tail bg

## Decisions Made

- Tasks 1 and 2 committed together in one atomic commit — both are test-only scaffolding with no production code changes; splitting into two commits would add overhead without clarity benefit
- CSS test uses `indexOf('.hub-modal {')` (with space+brace) to precisely match the panel class, not a prefix that could match `.hub-modal__*` selectors

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None — vitest run command format required `pnpm exec vitest run` (not `pnpm test --run` which gives "Unknown option: 'run'"). Matched existing test patterns correctly.

## Known Stubs

None — this plan creates test files only. No production code stubs introduced.

## Threat Flags

None — test-only artifacts, no runtime surface added.

## Next Phase Readiness

- All 4 Wave-0 test files exist at their VALIDATION.md-specified locations
- Each fails for the correct reason (missing implementation targets, not malformed tests)
- Phase 134 Plans 02–05 can now implement components/CSS against these RED targets
- When implementation is complete, these tests flip GREEN automatically — no test changes needed

---
*Phase: 134-modal-interaction*
*Completed: 2026-06-17*
