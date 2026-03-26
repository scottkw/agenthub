---
phase: 32-daemon-startup-performance
plan: 02
subsystem: daemon
tags: [go, path, nvm, volta, homebrew, service-mode, exec]

# Dependency graph
requires:
  - phase: 32-daemon-startup-performance plan 01
    provides: pollSessionStatus timing fix (PERF-01/PERF-02)
provides:
  - augmentServicePath function in internal/daemon/path.go
  - nvmActiveBin function in internal/daemon/path.go
  - PATH augmentation called as first line of runDaemonCore
affects: [daemon-startup, service-mode, exec-lookpath, agent-cli-resolution]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Runtime PATH augmentation at daemon startup using os.Setenv before exec.LookPath"
    - "nvm alias resolution via ~/.nvm/alias/default file + os.ReadDir scan"
    - "Silent skip of non-existent candidate dirs via os.Stat"

key-files:
  created:
    - internal/daemon/path.go
    - internal/daemon/path_test.go
  modified:
    - internal/daemon/process.go

key-decisions:
  - "Runtime PATH augmentation over install-time capture: avoids stale PATH when user switches nvm node versions"
  - "Separate path.go file over adding to process.go: keeps single responsibility, easier to test"
  - "os.Setenv on daemon process PATH (not cmd.Env): exec.LookPath reads process PATH, not child env"

patterns-established:
  - "PATH augmentation pattern: prepend candidates with os.Stat guard, join with PathListSeparator"
  - "nvm resolution pattern: read alias file, normalize v-prefix, scan versions/node/ directory"

requirements-completed: [PERF-03]

# Metrics
duration: 15min
completed: 2026-03-26
---

# Phase 32 Plan 02: Daemon PATH Augmentation Summary

**TDD implementation of runtime PATH augmentation at daemon startup so service-mode agents (nvm, Volta, Homebrew) resolve via exec.LookPath without shell init files**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-03-26T00:00:00Z
- **Completed:** 2026-03-26
- **Tasks:** 1 (TDD: RED + GREEN commits)
- **Files modified:** 3

## Accomplishments

- Created `internal/daemon/path.go` with `augmentServicePath` and `nvmActiveBin` functions
- `augmentServicePath` prepends `.volta/bin`, `/opt/homebrew/bin`, `/usr/local/bin`, `/home/linuxbrew/.linuxbrew/bin`, and the active nvm node bin to process PATH at daemon startup
- `nvmActiveBin` reads `~/.nvm/alias/default`, normalizes version prefix, scans `~/.nvm/versions/node/` to find the matching bin directory; handles both `"20"` and `"v20.11.0"` alias formats
- Non-existent directories silently skipped via `os.Stat`
- `augmentServicePath()` called as the first line of `runDaemonCore` in `process.go` — before `NewSessionEngine()` and any `exec.LookPath` usage
- 6 new tests all passing; full daemon package suite green

## Task Commits

TDD task produced two commits:

1. **RED phase (failing tests)** - `2d5687a` (test)
2. **GREEN phase (implementation)** - `50f05d4` (feat)

## Files Created/Modified

- `internal/daemon/path.go` - augmentServicePath and nvmActiveBin functions
- `internal/daemon/path_test.go` - 6 tests covering PATH augmentation and nvm resolution
- `internal/daemon/process.go` - Added `augmentServicePath()` as first line of runDaemonCore

## Decisions Made

- **Runtime augmentation over install-time capture:** Setting `EnvVars["PATH"]` in `newServiceConfig()` captures PATH at `agenthub daemon install` time; if user later switches nvm node versions, the service gets stale PATH. Runtime augmentation at startup is always current.
- **Separate `path.go` file:** Keeps single responsibility, easier to test in isolation.
- **`os.Setenv` on daemon process PATH:** `exec.LookPath` resolves against the daemon process's own `$PATH`, not the child process `cmd.Env`. Must set process env before `exec.Command` calls.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Pre-existing `TestPollSessionStatus_StopsOnErrored` failure in root package (60s timeout due to `time.Sleep(2s)` bug) — this is the PERF-01/PERF-02 bug Plan 01 addresses, unrelated to Plan 02 changes. All internal package tests pass cleanly.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- PATH augmentation complete; PERF-03 requirement satisfied
- Phase 32 plans complete: both pollSessionStatus timing fix (Plan 01) and service-mode PATH augmentation (Plan 02) are shipped
- Ready for phase verification (`/gsd:verify-work`)

---
*Phase: 32-daemon-startup-performance*
*Completed: 2026-03-26*
