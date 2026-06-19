---
phase: 134-modal-interaction
plan: "04"
subsystem: frontend/hub-modal
tags: [react, modal, animation, accessibility, focus-management]
dependency_graph:
  requires: ["134-01", "134-03"]
  provides: ["134-05"]
  affects: ["frontend/src/components/Hub/HubModal.tsx"]
tech_stack:
  added: []
  patterns:
    - "grow/shrink animation phase machine (entering → open → exiting)"
    - "stopImmediatePropagation Escape guard (Pitfall 6)"
    - "cardFocusRef focus-return on unmount"
    - "transformOrigin from sourceRect.center for shared-element grow animation"
key_files:
  created:
    - frontend/src/components/Hub/HubModal.tsx
  modified:
    - frontend/src/components/Hub/HubModal.test.tsx
decisions:
  - "HubModal routes to HubBriefingModal when isAttentionStatus=true, HubInteractiveModal otherwise"
  - "Interactive child receives isOpen={phase==='open'} (fit-safe timing guard — Pitfall 1)"
  - "Escape uses stopImmediatePropagation to prevent Hub card menu from double-firing"
  - "cardFocusRef captures document.activeElement at mount and restores focus in effect cleanup"
metrics:
  duration: "< 5 minutes"
  completed: "2026-06-17"
  tasks: 1
  files: 2
---

# Phase 134 Plan 04: HubModal Shell Summary

**One-liner:** HubModal shell with overlay, grow/shrink animation phase machine (entering→open→exiting), Escape+click-outside dismissal, cardFocusRef focus-return, transformOrigin from sourceRect center, and isAttentionStatus routing to HubInteractiveModal/HubBriefingModal.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | HubModal shell — overlay, animation phases, Escape/click-outside, focus return, header strip, type routing | 9bd8f1f3 | HubModal.tsx (new), HubModal.test.tsx (fixed import) |

## Verification

- HubModal.test.tsx: 9/9 tests GREEN (MODAL-01, MODAL-02, MODAL-03 assertions)
- `pnpm exec tsc --noEmit`: no errors
- `isAttentionStatus` present (non-comment): 2 occurrences
- `phase === 'open'` guard: 1 occurrence (interactive child fit-safe timing)
- No hardcoded hex values in HubModal.tsx
- All required literal strings present: `Session terminal:`, `Briefing:`, `needs input`, `Needs attention`, `Close modal`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed HubModal.test.tsx import path**
- **Found during:** Task 1 (test run)
- **Issue:** HubModal.test.tsx in `frontend/src/components/Hub/` imported `'../HubModal.tsx?raw'` (resolves to `frontend/src/components/HubModal.tsx`) instead of `'./HubModal.tsx?raw'` (same directory)
- **Fix:** Changed import to `'./HubModal.tsx?raw'`
- **Files modified:** frontend/src/components/Hub/HubModal.test.tsx
- **Commit:** 9bd8f1f3 (bundled with task commit)

## Known Stubs

None — HubModal correctly wires to real HubInteractiveModal and HubBriefingModal leaf components from 134-03. No placeholder data.

## Threat Surface Scan

No new threat surface beyond the plan's documented threat model (T-134-04-01 through T-134-04-03). All mitigations implemented:
- session.name rendered as React text content (no dangerouslySetInnerHTML)
- Escape calls stopImmediatePropagation
- document keydown listener removed in useEffect cleanup

## Self-Check: PASSED

- [x] frontend/src/components/Hub/HubModal.tsx — EXISTS
- [x] Commit 9bd8f1f3 — EXISTS (verified by git log)
- [x] HubModal.test.tsx 9/9 GREEN — VERIFIED
- [x] TypeScript noEmit — CLEAN
