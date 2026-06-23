---
phase: 149-google-antigravity-agent
plan: 01
subsystem: api
tags: [go, pty, daemon, status, cli-detection, windows-path, agy, google-antigravity]

# Dependency graph
requires:
  - phase: 148-session-tab-chevron
    provides: stable Go backend with knownCLIs, platformExtraBins, and status detector
provides:
  - knownCLIs entry {Name: "agy", DisplayName: "Google Antigravity"} in internal/pty/detect.go
  - %LOCALAPPDATA%\agy\bin Windows PATH entry in internal/daemon/path_windows.go
  - DefaultAgyPatterns + agy case in PatternsForCLI in internal/status/detector.go
  - Extended unit tests for all three surfaces
affects:
  - 149-02 (frontend badge — agentBadge.ts agy case, CSS token)
  - 149-03 (TESTING.md traceability, Suite Manifest count update)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED/GREEN per task: test commit (failing) then feat commit (passing)"
    - "Go switch over if/else for PatternsForCLI dispatch — extensible for future agents"
    - "Status patterns marked [ASSUMED] with explicit tuning-deferred comment when source is screenshots not live data"

key-files:
  created: []
  modified:
    - internal/pty/detect.go
    - internal/pty/detect_test.go
    - internal/daemon/path_windows.go
    - internal/daemon/path_windows_test.go
    - internal/status/detector.go
    - internal/status/detector_test.go

key-decisions:
  - "Binary key is 'agy' not 'antigravity' — exec.LookPath requires the real binary name (D-09)"
  - "Windows dir is 'agy' not 'Antigravity' — installer uses %LOCALAPPDATA%\\agy\\bin per RESEARCH Fact 1"
  - "DefaultAgyPatterns Working left empty — no reliable working indicator from screenshots; FallbackPatterns default (Running) applies during work; tuning deferred to post-M-15 live access (D-13)"
  - "PatternsForCLI converted from if/else to switch — extensible pattern for future agent additions"

patterns-established:
  - "New agent pattern: CLISpec row in knownCLIs + Windows PATH entry + PatternSet in one plan"

requirements-completed: [AGENT-01]

# Metrics
duration: 25min
completed: 2026-06-23
---

# Phase 149 Plan 01: Google Antigravity Agent Backend Summary

**agy/Google Antigravity wired into Go backend: knownCLIs detection, %LOCALAPPDATA%\\agy\\bin Windows PATH, and minimal idle/waiting PatternSet — all surfaces covered via TDD with six task commits**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-06-23T04:00:00Z
- **Completed:** 2026-06-23T04:25:00Z
- **Tasks:** 3 (each with RED + GREEN commits)
- **Files modified:** 6

## Accomplishments

- Added `{Name: "agy", DisplayName: "Google Antigravity"}` to `knownCLIs` — flows to GUI/CLI/web detection via DetectCLIs/DetectCLI
- Added `filepath.Join(local, "agy", "bin")` to `platformExtraBins()` — service-mode daemon on Windows can discover agy.exe
- Added `DefaultAgyPatterns()` (Idle: `>\s*$`, Waiting: `[y/n]` variants) and `case "agy"` in `PatternsForCLI` switch — agy sessions classify as idle rather than perma-running

## Task Commits

Each task followed TDD RED then GREEN:

1. **Task 1 RED: detect tests (failing)** - `8cd42d01` (test)
2. **Task 1 GREEN: agy to knownCLIs** - `19d3a274` (feat)
3. **Task 2 RED: Windows PATH test (failing)** - `c3fc54a3` (test)
4. **Task 2 GREEN: agy\bin to platformExtraBins** - `6567ad6c` (feat)
5. **Task 3 RED: status pattern tests (failing)** - `74579215` (test)
6. **Task 3 GREEN: DefaultAgyPatterns + PatternsForCLI switch** - `762e8be1` (feat)

## Files Created/Modified

- `internal/pty/detect.go` — Added agy CLISpec row to knownCLIs
- `internal/pty/detect_test.go` — Updated TestKnownCLIs (5 entries), added TestDetectCLIs_FindsAgy and TestDetectCLI_AgyNotFound
- `internal/daemon/path_windows.go` — Added filepath.Join(local, "agy", "bin") inside LOCALAPPDATA block
- `internal/daemon/path_windows_test.go` — Added TestPlatformExtraBins_WindowsIncludesAgyBin; extended LocalAppDataEmpty to also assert no agy\bin when empty
- `internal/status/detector.go` — Added DefaultAgyPatterns(), converted PatternsForCLI to switch with case "agy"
- `internal/status/detector_test.go` — Added newAgyDetector helper, TestDetector_AgyIdle, TestDetector_AgyWaiting, TestPatternsForCLI_AgyNotFallback

## Decisions Made

- Binary key is `agy` (not `antigravity`) — `exec.LookPath` requires the real executable name; using `antigravity` would break detection on all platforms (D-09)
- Windows directory is `agy` (not `Antigravity`) — installer canonical path is `%LOCALAPPDATA%\agy\bin\agy.exe` per RESEARCH Fact 1
- `DefaultAgyPatterns.Working` left empty intentionally — no reliable working indicator from TUI screenshots; conservative FallbackPatterns Running default applies; full tuning deferred to post-M-15 live sampling per CONTEXT D-13 and RESEARCH Open Question 1
- `PatternsForCLI` refactored from `if/else` to `switch` — cleaner pattern that scales for future agents

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01 complete: Go backend fully wired for agy detection, Windows PATH, and status classification
- Plan 02 ready: frontend agentBadge.ts `agy` case + CSS token (#ff9e64) can now reference the canonical Name "agy"
- Plan 03 (TESTING.md) will add AGENT-01 traceability row and Suite Manifest count update

## Self-Check

Files created/modified:

- `internal/pty/detect.go` — found and modified
- `internal/pty/detect_test.go` — found and modified
- `internal/daemon/path_windows.go` — found and modified
- `internal/daemon/path_windows_test.go` — found and modified
- `internal/status/detector.go` — found and modified
- `internal/status/detector_test.go` — found and modified

Commits verified:

- 8cd42d01 (test RED pty) — present
- 19d3a274 (feat GREEN pty) — present
- c3fc54a3 (test RED daemon) — present
- 6567ad6c (feat GREEN daemon) — present
- 74579215 (test RED status) — present
- 762e8be1 (feat GREEN status) — present

## Self-Check: PASSED

---
*Phase: 149-google-antigravity-agent*
*Completed: 2026-06-23*
