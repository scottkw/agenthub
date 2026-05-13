---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
plan: "107-04"
subsystem: ui
tags: [react, vitest, session-exit, ExitToast, tab-management, SHELL-12]

# Dependency graph
requires:
  - phase: 107-02
    provides: "daemon normalizes PTY exit-code -1→0 so frontend branch on ===0 is sound"
provides:
  - "session:exit handler branches on exitCode===0 — clean exits close tab immediately, no ExitToast"
  - "11-test Vitest suite locking UI-SPEC §4 SHELL-12 five-assertion contract"
affects: [ExitToast rendering, tab focus adjacency, session lifecycle UX]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Early-return guard pattern: branch at top of event handler before any state mutation"
    - "Source-inspection test pattern (?raw import) for React event handler structural contracts"

key-files:
  created:
    - frontend/src/components/__tests__/App.shellExit.test.tsx
  modified:
    - frontend/src/App.tsx

key-decisions:
  - "Early-return before setSessionExits ensures ExitToast never flashes for clean exits — no render cycle sees the state"
  - "countdown setInterval block removed entirely (was gated on exitCode===0, now unreachable after early-return)"
  - "countdownTimers ref and autoCloseRef retained — still used by handleCloseTab cleanup; no hygiene removal"
  - "Source-inspection tests (App.tsx?raw) used per established codebase pattern rather than full component mounting"

patterns-established:
  - "Event handler branching: early-return at handler top for fast-path; state mutations only on slow path"

requirements-satisfied: [SHELL-12 frontend half]
requirements-completed: [SHELL-12]

# Metrics
duration: 3min
completed: 2026-05-13
---

# Phase 107 Plan 04: SHELL-12 Frontend — Auto-Close Tab on Exit-Code 0 Summary

**session:exit handler gains exitCode===0 early-return that closes the tab immediately via handleCloseTabRef, skipping setSessionExits and ExitToast entirely for clean shell exits**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-05-13T04:50:23Z
- **Completed:** 2026-05-13T04:53:23Z
- **Tasks:** 1 (TDD: RED + GREEN)
- **Files modified:** 2

## Accomplishments

- Early-return branch inserted at the top of the `session:exit` event handler in App.tsx: `if (data.exitCode === 0) { void handleCloseTabRef.current?.(data.sessionId); return }`
- Dead countdown block (setInterval + autoCloseRef logic) removed — it was entirely gated on `exitCode === 0`, now unreachable after the early-return
- `countdown: -1` constant used for non-zero exit path (was dynamic ternary `data.exitCode === 0 ? 5 : -1`, now always -1 since exit-code 0 exits early)
- Non-zero exit path (ExitToast) fully preserved and unchanged
- 11-test Vitest suite created with 5 UI-SPEC §4 SHELL-12 assertions + 6 infrastructure invariants
- Full suite: 888/888 green; `pnpm tsc --noEmit` clean

## Task Commits

1. **Task 1 (RED): Failing SHELL-12 test suite** - `0172181` (test)
2. **Task 1 (GREEN): Branch session:exit on exitCode===0** - `1fdbf81` (feat)

**Plan metadata:** (docs commit — this summary + state updates)

## Files Created/Modified

- `frontend/src/components/__tests__/App.shellExit.test.tsx` — 11-test source-inspection suite for SHELL-12 five-assertion contract
- `frontend/src/App.tsx` — session:exit handler refactored: early-return for exit-code 0, countdown block removed

## Decisions Made

- **Early-return over if/else:** Matches UI-SPEC §2 exactly; guarantees `setSessionExits` is never called for exit-code 0 regardless of future changes below the branch
- **Source-inspection tests:** Following the pattern established by App.exit.test.tsx, App.nav.test.tsx, App.shellWebShare.test.tsx — avoids the extensive Wails-runtime mock harness required for full App mounting
- **Retain autoCloseRef/countdownTimers:** Both refs still referenced outside the deleted block (countdownTimers cleanup in handleCloseTab, autoCloseRef potentially used by other paths). Left intact per plan instruction; flagged for hygiene pass after phase ships.

## Deviations from Plan

None — plan executed exactly as written. The implementation followed the UI-SPEC §2 locked shape verbatim.

## Issues Encountered

- During full suite run, `NewSessionModal.test.tsx` showed 7 failures — these are from 107-03's parallel wave work (NewSessionModal.tsx was already modified in the working directory by 107-03's agent). Confirmed out-of-scope: reverting my App.tsx changes had no effect on those failures. By the time of final verification, 107-03 had committed its changes and all 888 tests passed.

## Known Stubs

None — no hardcoded empty values or placeholder text introduced.

## Threat Flags

None — this change reduces attack surface by removing countdown timer infrastructure from the event handler. No new network endpoints or auth paths introduced.

## Follow-up Items

- `autoCloseRef` remains defined but is now only referenced in the mount-time `GetAutoCloseSession().then(...)` hydration call and no longer inside the `session:exit` handler. If SHELL-12's non-zero exit path never needs countdown behavior, `autoCloseRef` and `countdownTimers` could be removed in a hygiene pass after Phase 107 ships. Note: `countdownTimers` cleanup in `handleCloseTab` is still a valid no-op safety measure.

## Self-Check: PASSED

- `frontend/src/components/__tests__/App.shellExit.test.tsx` — FOUND
- `frontend/src/App.tsx` (modified) — FOUND
- Commit `0172181` — FOUND (test RED)
- Commit `1fdbf81` — FOUND (feat GREEN)
- `grep -c "data.exitCode === 0" frontend/src/App.tsx` — 1 (only the early-return branch)
- `grep -c "countdown: data.exitCode === 0" frontend/src/App.tsx` — 0 (old ternary removed)
- `pnpm tsc --noEmit` — clean (no output)
- `pnpm test` — 888/888 green

## Next Phase Readiness

- SHELL-12 frontend half complete. Pairs with 107-02 daemon normalization.
- 107-01 (shell-path backend) and 107-03 (NewSessionModal + SettingsTab frontend) are the remaining wave-1 plans.
- UAT: spawn shell, type `exit` → tab vanishes, no toast. `exit 2` → ExitToast appears as before.

---
*Phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling*
*Completed: 2026-05-13*
