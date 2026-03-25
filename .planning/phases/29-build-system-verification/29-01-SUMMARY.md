---
phase: 29-build-system-verification
plan: 01
subsystem: testing
tags: [ci, github-actions, race-detector, bash, build-script]

# Dependency graph
requires:
  - phase: 28-cli-package-removal
    provides: unified agenthub binary in project root (no cmd/agenthub-cli/)
provides:
  - CI-ready build-script tests with portable path resolution (BASH_SOURCE)
  - Race-enabled Go test step on all 4 CI matrix legs
  - Build-script verification step on ubuntu-latest CI leg
affects: [future CI changes, future build system changes]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "BASH_SOURCE[0] for portable shell script path resolution"
    - "go test -race for race-condition detection in CI"
    - "if: runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest' for platform-specific CI steps"

key-files:
  created: []
  modified:
    - tests/build-script.test.sh
    - .github/workflows/build.yml

key-decisions:
  - "Build-script test step runs on ubuntu-latest only — build.sh uses docker for Linux cross-compile, only that runner has bash and the required toolchain detection paths"
  - "BASH_SOURCE over $0 — correct for sourced scripts and direct bash invocations, handles edge cases where $0 is 'bash'"

patterns-established:
  - "BASH_SOURCE[0] pattern: SCRIPT_DIR=$(cd $(dirname ${BASH_SOURCE[0]}) && pwd) for portable test script paths"

requirements-completed: [BUILD-01, BUILD-02, BUILD-03]

# Metrics
duration: 8min
completed: 2026-03-25
---

# Phase 29 Plan 01: Build System Verification Summary

**Portable BASH_SOURCE path resolution in build-script tests and race-enabled CI workflow with ubuntu-latest build-script verification**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-25T00:00:00Z
- **Completed:** 2026-03-25T00:08:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Replaced hardcoded `/Users/ken/dev/agenthub/build.sh` with `BASH_SOURCE[0]`-based portable path — test now runs from any CWD or CI runner
- All 35 build-script tests pass with the portable path
- Added `-race` flag to `go test ./...` in CI — all 6 packages pass race-clean locally (194 tests across all packages)
- Added `Run build script tests` CI step on ubuntu-latest only with correct `if:` condition

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix build-script test path and verify locally** - `3b34c1c` (fix)
2. **Task 2: Add race flag and build-script step to CI workflow** - `aa9c9f4` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `tests/build-script.test.sh` - Line 6 comment updated; lines 10-11 replace hardcoded path with BASH_SOURCE resolution
- `.github/workflows/build.yml` - "Run Go tests (all platforms)" renamed and gets -race; new "Run build script tests" step added after Go tests, before Build with Wails

## Decisions Made
- Build-script test step restricted to `runner.os == 'Linux' && matrix.build.os == 'ubuntu-latest'` — `build.sh` relies on bash features and the Linux runner has the necessary environment without macOS/Windows complications
- Used `BASH_SOURCE[0]` (not `$0`) for portable script self-location — `$0` can be "bash" when run as `bash script.sh`, while `BASH_SOURCE[0]` always points to the script file

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- BUILD-01, BUILD-02, BUILD-03 requirements satisfied
- CI workflow is hardened with race detection and build-script verification
- No blockers for future phases

---
*Phase: 29-build-system-verification*
*Completed: 2026-03-25*
