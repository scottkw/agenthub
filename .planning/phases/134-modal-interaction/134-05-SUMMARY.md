---
phase: 134-modal-interaction
plan: "05"
subsystem: frontend/hub-modal
tags: [react, modal, css, animation, cap-gate, accessibility]
dependency_graph:
  requires: ["134-01", "134-02", "134-04"]
  provides: []
  affects:
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/App.tsx
    - frontend/src/style.css
tech_stack:
  added: []
  patterns:
    - "MODAL-06 remote cap gate in handleCardClick (returns early, calls onRequestRemoteCap)"
    - "onRegisterCapAcquired pattern for cross-component callback registration via ref"
    - "intent discriminator on joinModalForSession state ('files' | 'hub-modal')"
    - "capAcquiredRef pattern (avoids stale closures in handleModalExchange)"
    - "root-scope CSS keyframes with no-preference-guarded animation assignments (Phase 133 template)"
key_files:
  modified:
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/App.tsx
    - frontend/src/style.css
decisions:
  - "HubPanel uses a React fragment wrapper so HubModal renders outside .hub div (overlay covers full surface)"
  - "pendingSourceRectRef captures card rect at cap-request time for use in auto-open after exchange"
  - "intent discriminator uses undefined-as-files default so existing file-browse callers need no changes"
  - "capAcquiredRef rather than state avoids stale-closure issues in handleModalExchange's useCallback"
  - "Keyframes at root scope per plan spec; only animation: assignments inside no-preference guard"
metrics:
  duration: "< 10 minutes"
  completed: "2026-06-17"
  tasks: 3
  files_modified: 4
---

# Phase 134 Plan 05: Final Integration Summary

**One-liner:** HubPanel gains MODAL-06 remote cap gate + HubModal render; App.tsx wires cap-acquired dispatch with intent discriminator; style.css gets all modal CSS with root-scope keyframes and token-only colors.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | HubPanel modal state + MODAL-06 cap gate + HubModal render | 7cf8a19c | HubPanel.tsx, HubPanel.test.tsx |
| 2 | App.tsx wiring — modal props + remote cap-acquired signal | 1edef361 | App.tsx |
| 3 | style.css — modal CSS, grow/shrink keyframes, reduced-motion guards | f178df95 | style.css |

## What Was Built

### Task 1: HubPanel Modal Integration

HubPanel received 7 new props (`relayPort`, `terminalTheme`, `pluginConfig`, `remoteCapsCached`, `onRequestRemoteCap`, `onRegisterCapAcquired`) and full modal state management:

- `modalState` / `setModalState` — tracks open modal session+sourceRect
- `pendingModalSessionId` / `pendingSourceRectRef` — tracks remote session awaiting cap
- `handleCardClick` — gates remote sessions via `remoteCapsCached`; calls `onRequestRemoteCap` for uncapped remotes (returns early, never opens modal directly)
- `handleCapAcquired` — callback registered with App.tsx; finds session, uses stored rect, opens modal
- `onCardClick={handleCardClick}` threaded to SessionCardGrid
- `<HubModal>` rendered in a React fragment when `modalState && relayPort !== undefined`

HubPanel.test.tsx extended with 4 source-inspection MODAL-06 assertions (all 41 tests pass).

### Task 2: App.tsx Cap-Acquired Signal

- Added `intent?: 'files' | 'hub-modal'` field to `joinModalForSession` state type
- Added `capAcquiredRef` to hold HubPanel's registered cap-acquired callback
- Updated `handleModalExchange` to branch on `pending.intent`: `'hub-modal'` calls `capAcquiredRef.current?.(pending.id)`; anything else (including `undefined`) continues to call `handleOpenFileBrowser` (Phase 122/130 path unregressed)
- Passed `relayPort/terminalTheme/pluginConfig/remoteCapsCached/onRequestRemoteCap/onRegisterCapAcquired` to `<HubPanel>`
- `onRequestRemoteCap` sets `intent: 'hub-modal'` to distinguish from file-browse path

### Task 3: style.css Modal CSS

Added all Phase 134 modal CSS (268 lines):

- `.hub-modal-overlay`: position fixed, inset 0, z-index 200, rgba(0,0,0,0.6) scrim
- `.hub-modal`: min(1100px/750px) panel, var(--hub-surface-elevated), border, box-shadow
- Header/body/tail/respond/send-btn/close-btn/error-banner rules (all var(--hub-*) tokens, no hardcoded hex)
- `@keyframes hub-modal-grow/shrink/overlay-in/overlay-out` at ROOT scope
- `animation:` assignments inside `@media (prefers-reduced-motion: no-preference)`
- Reduced-motion fallback: `animation: none; transition: none; opacity: 1` on both modal and overlay

## Verification Results

- Full frontend suite: **1686 tests, 104 test files — all pass**
- `pnpm exec tsc --noEmit` — **clean**
- 134-01 CSS contract tests (style.hub.modal.test.ts) — **all 14 now green**
- Prior Hub CSS contract tests (style.hub.test.ts, 48 tests) — **no regressions**
- HubPanel MODAL-06 source-inspection tests — **4 new, all pass**

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. All modal CSS is wired. HubModal renders correctly via HubPanel with real relayPort/theme/pluginConfig from App.tsx.

## Threat Surface Scan

No new threat surface introduced beyond what was planned. All STRIDE mitigations from the plan's threat register are implemented:

- T-134-05-01: MODAL-06 cap gate enforced in handleCardClick — remote modal only opens when `remoteCapsCached.has(session.id)` (or after successful exchange)
- T-134-05-02: Intent discriminator in handleModalExchange routes correctly — 'hub-modal' never calls handleOpenFileBrowser
- T-134-05-04: `onClose={() => setModalState(null)}` unmounts HubModal on close — no accumulating WS connections

## Self-Check: PASSED

- HubPanel.tsx: FOUND
- App.tsx: FOUND
- style.css: FOUND
- 134-05-SUMMARY.md: FOUND
- Task 1 commit 7cf8a19c: FOUND
- Task 2 commit 1edef361: FOUND
- Task 3 commit f178df95: FOUND
