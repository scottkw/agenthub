---
phase: 168-bug-fix-settings-polish
plan: 08
subsystem: ui
tags: [react, wails, share-modal, state-sync]

# Dependency graph
requires:
  - phase: 168-bug-fix-settings-polish
    provides: "Plan 05's single lifted SessionShareModal instance at App.tsx level (footer no longer toggles web-sharing directly)"
provides:
  - "onShareEnabledChange callback on SessionShareModal, closing the last recorded App-level state-sync gap for the modal's own toggle"
  - "Regression proof that the footer StatusBar pill tracks live web-share state for every toggle path (warned, un-warned, ON, OFF)"
affects: [168-bug-fix-settings-polish, UX-02, footer-statusbar]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Host-notification callback pattern: a controlled child (SessionShareModal) reports server-truth state changes to its host (App.tsx) via an optional callback prop, invoked only after the underlying RPC resolves successfully (preserves the WR-01 success-gate)."

key-files:
  created: []
  modified:
    - frontend/src/components/Hub/SessionShareModal.tsx
    - frontend/src/App.tsx
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - frontend/src/components/__tests__/StatusBar.shareSession.test.tsx
    - TESTING.md

key-decisions:
  - "Chose Option A (callback) over Option B (poll reconciliation) per the plan's pre-decided rationale: the CR-02 poll only runs while the modal is open from a non-Hub tab, so it cannot cover the Hub-card-open case, and would race the immediate callback."
  - "onShareEnabledChange fires only inside handleShareToggle's success path (after await ToggleWebServing), not inside the warning-guard early return — avoids a double-set with App's existing handleShellWebShareConfirm on the un-warned first-time shell path."

requirements-completed: [UX-02]

coverage:
  - id: D1
    description: "SessionShareModal exposes an optional onShareEnabledChange(sessionId, enabled) prop, invoked from handleShareToggle after a successful ToggleWebServing for both ON and OFF transitions on the already-warned/non-shell path."
    requirement: UX-02
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — Phase 168-08 (gap): onShareEnabledChange notifies App on modal toggle in the already-warned path > warned shell path: toggling ON calls ToggleWebServing(id, true) and onShareEnabledChange(id, true); no banner"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — Phase 168-08 (gap): onShareEnabledChange notifies App on modal toggle in the already-warned path > warned shell path: toggling OFF (already shared) calls ToggleWebServing(id, false) and onShareEnabledChange(id, false)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The un-warned first-time shell path (banner shown) does NOT call onShareEnabledChange, avoiding a double-set with App's handleShellWebShareConfirm."
    requirement: UX-02
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — Phase 168-08 (gap): onShareEnabledChange notifies App on modal toggle in the already-warned path > un-warned first-time shell path: toggling ON shows the banner and does NOT call onShareEnabledChange (no double-set with handleShellWebShareConfirm)"
        status: pass
    human_judgment: false
  - id: D3
    description: "App.tsx's single <SessionShareModal> render wires onShareEnabledChange to setWebEnabled, so webEnabled[sessionId] — and therefore the footer StatusBar pill — tracks the modal toggle for every path (footer-opened + Hub-card-opened, warned + un-warned)."
    requirement: UX-02
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/StatusBar.shareSession.test.tsx#App.tsx wiring — footer \"Share Session\" opens the lifted Share modal (D-14) > wires onShareEnabledChange on the <SessionShareModal> render to setWebEnabled, closing the UX-02 / #115 footer pill drift"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live observable truth: dismiss the shell warning once, then toggle 'Share the session' ON/OFF from the modal — footer pill reads 'WEB ON'/'WEB OFF' with no drift."
    requirement: UX-02
    verification: []
    human_judgment: true
    rationale: "Requires a live daemon session with a real shell CLI and manual dismissal of the one-time shell-web-share warning, matching TESTING.md Test 4's rerun scope — not reproducible in the unit-test harness which mocks ToggleWebServing. Deferred to live-UAT re-check per the plan's <verification> section."

# Metrics
duration: 20min
completed: 2026-07-02
status: complete
---

# Phase 168 Plan 08: Footer StatusBar Pill Web-Share Drift Fix Summary

**SessionShareModal gains an onShareEnabledChange(sessionId, enabled) callback, wired in App.tsx to setWebEnabled, so the footer pill tracks server-truth web-share state on every modal toggle — closing the last recorded gap in the UX-02 (#115) "no more button↔modal↔pill drift" goal.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-02T12:53:50Z
- **Tasks:** 2/2 completed
- **Files modified:** 5

## Accomplishments
- SessionShareModal's `handleShareToggle` now notifies its host via `onShareEnabledChange(session.id, next)` for both ON and OFF transitions, fired only after `ToggleWebServing` resolves successfully (preserves the WR-01 success-gate) — the un-warned first-time shell path is untouched (App's `handleShellWebShareConfirm` already covers it, so no double-set).
- App.tsx's single `<SessionShareModal>` render wires the new prop to `setWebEnabled`, so `webEnabled[sessionId]` — and the footer pill it drives — now tracks server truth for every toggle path: footer-opened, Hub-card-opened, warned, and un-warned.
- Regression coverage locks the fix: a mounted-modal test proves the warned-path toggle notifies ON then OFF and the un-warned path does not; a source-inspection test proves the App-level wiring exists.
- TESTING.md updated per the standing convention (Suite Manifest note + 2 new UX-02 traceability rows); `check-traceability-paths.sh` passes; vitest count unchanged (141 files, both extended in place).

## Task Commits

Each task was committed atomically:

1. **Task 1: Add onShareEnabledChange callback to SessionShareModal and wire it to App.setWebEnabled** - `2129a423` (fix)
2. **Task 2: Add warned-path regression test + App wiring source-inspection + TESTING.md updates** - `a1b609ff` (test)

**Plan metadata:** (this commit, docs: complete plan)

## Files Created/Modified
- `frontend/src/components/Hub/SessionShareModal.tsx` - New optional `onShareEnabledChange?: (sessionId, enabled) => void` prop; invoked from `handleShareToggle` after a successful `ToggleWebServing`, for both ON and OFF.
- `frontend/src/App.tsx` - The single `<SessionShareModal>` render (App.tsx level, gated on `shareModalSession`) now passes `onShareEnabledChange={(sessionId, enabled) => setWebEnabled((prev) => ({ ...prev, [sessionId]: enabled }))}`.
- `frontend/src/components/__tests__/SessionShareModal.test.tsx` - New describe block: warned-path ON/OFF notify assertions + un-warned no-notify assertion.
- `frontend/src/components/__tests__/StatusBar.shareSession.test.tsx` - New source-inspection test asserting the App.tsx `<SessionShareModal>` render site wires `onShareEnabledChange` to `setWebEnabled`.
- `TESTING.md` - New Suite Manifest note (168-08 gap closure) + 2 new UX-02 traceability rows for the extended test files.

## Decisions Made
- Chose Option A (callback) over Option B (poll reconciliation) per the plan's pre-decided rationale — the CR-02 poll only runs while the modal is open from a non-Hub tab and would race an immediate callback.
- `onShareEnabledChange` invoked strictly after `await ToggleWebServing` resolves (success path), never inside the warning-guard early return, to avoid a double-set with `handleShellWebShareConfirm`.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - `pnpm exec tsc --noEmit` was clean on the first pass, and both target test runs (`SessionShareModal StatusBar.shareSession` and the full 141-file vitest suite) passed without iteration.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

This closes the sole diagnosed UAT gap recorded in `168-UAT.md` for Phase 168. All static verification (tsc, vitest full suite, traceability-paths script) is green. The one remaining item — live confirmation that the footer pill visibly updates on a real daemon session after dismissing the shell warning — is deferred to a live-UAT re-check per the plan's `<verification>` section (TESTING.md Test 4 rerun), consistent with how this same class of live-daemon behavior was verified for the rest of Phase 168.

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-02*

## Self-Check: PASSED

All created/modified files confirmed on disk; both task commit hashes (`2129a423`, `a1b609ff`) confirmed in git log.
