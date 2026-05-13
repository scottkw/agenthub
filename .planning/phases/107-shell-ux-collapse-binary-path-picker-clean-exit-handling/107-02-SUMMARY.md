---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
plan: "107-02"
subsystem: daemon
tags: [go, pty, exit-code, shell, normalization]

requires:
  - phase: 100-shell-session-support
    provides: "Natural-exit goroutine with existing -1→0 normalization at onExit path"

provides:
  - "ListSessions ExitCode field normalized: -1 is mapped to 0 before emission (SHELL-12)"
  - "6 regression tests protecting the normalization invariant at both emission sites"

affects:
  - "107-04: frontend can now trust exitCode in session:exit event is user-meaningful (0=clean, non-zero=error)"
  - "App.tsx session-exit handler: branch on exitCode===0 is safe"

tech-stack:
  added: []
  patterns:
    - "ListSessions normalization guard: apply `if ec == -1 { ec = 0 }` after reading s.ExitCode() whenever emitting to external consumers"
    - "Unit-test via registry injection: populate engine.registry directly with pre-configured pty.Session values to test ListSessions behavior without real PTY"

key-files:
  created: []
  modified:
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go

key-decisions:
  - "Apply -1→0 guard only at ListSessions emission site (not in Session.ExitCode() itself) to preserve the raw value for internal callers while normalizing at the API boundary"
  - "Use direct registry injection in tests (not real process spawning) for the ListSessions unit tests — faster, deterministic, no PTY required"
  - "onExit callback path tested with a real short-lived sh -c 'exit 0' process routed through the non-shell AI-CLI path (fakecli→/bin/sh) so isShellSession() is false and custom args are honored"

patterns-established:
  - "SHELL-12 normalization pattern: at every external ExitCode emission site, apply `if ec == -1 { ec = 0 }` — mirrors the conservative D-10 default from the natural-exit goroutine"

requirements-completed: [SHELL-12]

duration: 4min
completed: 2026-05-13
---

# Phase 107 Plan 02: SHELL-12 Backend Exit-Code Normalization Summary

**Closed the ListSessions -1→0 normalization gap: PTY-EOF exit code is now mapped to 0 at the ListSessions emission site, so the GUI never interprets a clean shell exit as "exited with error".**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-05-13T04:43:10Z
- **Completed:** 2026-05-13T04:46:54Z
- **Tasks:** 1 (TDD: RED + GREEN)
- **Files modified:** 2

## Accomplishments

- Inserted `if ec == -1 { ec = 0 }` guard at `engine.go:389` (ListSessions natural-exit emission block), mirroring the identical guard already present at line 334 (natural-exit goroutine before onExit).
- 6 regression tests covering: -1→0 normalization, non-zero preservation, zero preservation, state=stopped assertion, killed-session ExitCode=nil guard, and onExit callback normalization.
- Full daemon suite passes with `-race` flag; gofmt + go vet clean.
- Confirmed two `== -1` normalization sites in engine.go (goroutine at L334 + ListSessions at L389).

## Task Commits

1. **Task 1 RED: SHELL-12 regression tests** — `bb28e1c` (test)
2. **Task 1 GREEN: -1→0 normalization in ListSessions** — `70c9f12` (feat)

## Files Created/Modified

- `internal/daemon/engine.go` — Added 3-line normalization guard in ListSessions ExitCode emission block (~L387-389)
- `internal/daemon/engine_test.go` — Added 6 SHELL-12 regression tests plus 3 helper functions (`newExitedShell12Session`, `newKilledShell12Session`, `newBareEngine`, `findShell12Session`) and imported `status` package

## Decisions Made

- Applied the guard only at the ListSessions emission site (not in `Session.ExitCode()`) so the raw -1 value is preserved internally — callers that need to distinguish "not yet set" from "exited with code 0" can still use the raw ExitCode. The normalization applies only at the external API boundary.
- Used direct registry injection for the ListSessions unit tests instead of spawning real processes — this gives deterministic, platform-independent -1 exit-code values without depending on PTY behavior.

## Deviations from Plan

None — plan executed exactly as written. The implementation was a surgical 3-line change. No architectural decisions required, no missing dependencies, no bugs found in adjacent code.

## Issues Encountered

None.

## Known Stubs

None.

## Threat Flags

None — the change is a normalization guard inside the daemon's internal ListSessions method. No new network surface, no new auth paths, no new file access patterns.

## Note for 107-04 Executor

The daemon now guarantees: at the ListSessions ExitCode field AND the onExit callback, exitCode is the user-meaningful value — `0` for a clean/natural exit, non-zero for a real error. The frontend (107-04) can safely branch purely on `data.exitCode === 0` to decide whether to auto-close the tab silently or show the ExitToast.

## Next Phase Readiness

- 107-04 (frontend SHELL-12) can proceed: daemon boundary is correct.
- 107-01 API routes are needed before 107-04 can test the full flow end-to-end (GET/PATCH /settings/shell-path).

---
*Phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling*
*Completed: 2026-05-13*

## Self-Check

### Files Exist

- `internal/daemon/engine.go` — modified (guard at ListSessions)
- `internal/daemon/engine_test.go` — modified (6 SHELL-12 tests)

### Commits Exist

- `bb28e1c` — test(107-02): add failing tests for SHELL-12 -1→0 normalization
- `70c9f12` — feat(107-02): normalize PTY exit-code -1→0 in ListSessions emission path

## Self-Check: PASSED
