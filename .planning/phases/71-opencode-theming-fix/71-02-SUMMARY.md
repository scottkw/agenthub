---
phase: 71-opencode-theming-fix
plan: 02
subsystem: daemon
tags: [go, opencode, env-injection, xterm-theme, tui-config, tdd]

# Dependency graph
requires:
  - phase: 71-01
    provides: "RED-state test stubs (TestCreateSession_OpenCodeEnv, TestOpenCodeTUIConfig) and spyBackend test double"
provides:
  - "ensureOpenCodeTUIConfig helper writing managed opencode-tui.json with system theme"
  - "Per-CLI env injection in CreateSession: OPENCODE_TUI_CONFIG for opencode sessions only"
  - "daemonConfigDir() mirroring app.go configDir() for internal package use"
  - "GREEN tests for both Wave 0 test stubs"
affects: [71-03, 71-04]

# Tech tracking
tech-stack:
  added: []
  patterns: [per-agent env injection via CreateRequest.Env, managed config file for external CLI]

key-files:
  created: []
  modified:
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go

key-decisions:
  - "Duplicated configDir() as daemonConfigDir() in daemon package since internal packages cannot import main"
  - "Used cli string comparison (not cliPath) for opencode detection, matching existing pattern in status/detector.go"
  - "Guarded injection with e.opencodeTUIConfig != '' so tests constructing engines without init don't inject empty env var"
  - "File overwritten on every engine startup (idempotent); content is hardcoded constant, not user data"

patterns-established:
  - "Per-agent env injection: check cli name in CreateSession, append to env []string, pass via CreateRequest.Env"
  - "Managed config file: ensureOpenCodeTUIConfig writes static JSON to daemonConfigDir at engine init"

requirements-completed: [THM-05]

# Metrics
duration: 4min
completed: 2026-04-13
---

# Phase 71 Plan 02: OpenCode Theming Implementation Summary

**Per-CLI env injection forcing OpenCode's system theme via managed tui.json and OPENCODE_TUI_CONFIG env var in CreateSession**

## Performance

- **Duration:** 4m 18s
- **Started:** 2026-04-13T19:15:01Z
- **Completed:** 2026-04-13T19:19:19Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- ensureOpenCodeTUIConfig writes `{"$schema":"https://opencode.ai/tui.json","theme":"system"}` to managed config file at engine startup
- CreateSession injects OPENCODE_TUI_CONFIG env var for opencode CLI sessions only; non-opencode CLIs (claude, codex, gemini) verified to NOT receive it
- Both Wave 0 RED test stubs (TestCreateSession_OpenCodeEnv, TestOpenCodeTUIConfig) are now GREEN
- Full Go test suite (8 packages, 200+ tests) passes with zero regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Add ensureOpenCodeTUIConfig helper and opencodeTUIConfig field** - `aa9544f` (feat)
2. **Task 2: Inject OPENCODE_TUI_CONFIG env var in CreateSession for opencode CLI** - `dc73386` (feat)

## Files Created/Modified
- `internal/daemon/engine.go` - Added daemonConfigDir(), ensureOpenCodeTUIConfig(), opencodeTUIConfig struct field, per-agent env injection in CreateSession
- `internal/daemon/engine_test.go` - Updated TestOpenCodeTUIConfig from RED stub to GREEN with idempotency; updated TestCreateSession_OpenCodeEnv with value verification and codex negative assertion

## Decisions Made
- Duplicated configDir() as daemonConfigDir() in daemon package -- internal packages cannot import main; 6-line function, acceptable duplication per Chesterton's Fence (app.go version serves GUI startup, daemon version serves engine init)
- Used `cli == "opencode"` (raw CLI name) not cliPath (resolved executable) for detection -- matches existing pattern in internal/status/detector.go line 86
- Guarded with `e.opencodeTUIConfig != ""` to prevent empty env var injection in test scenarios where engines are constructed without full init
- File content is a hardcoded constant overwritten on every startup -- acceptable since this is a managed file, not user configuration

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Threat Surface Scan

No new threat surfaces introduced beyond what the plan's threat model already covers. All three mitigations verified:
- T-71-01: daemonConfigDir uses os.UserConfigDir() with no user input in path computation
- T-71-02: OPENCODE_TUI_CONFIG only injected when cli == "opencode" (test asserts claude and codex do NOT receive it)
- T-71-03: Content is hardcoded constant `{"$schema":"...","theme":"system"}\n`

## Next Phase Readiness
- Plans 71-03 and 71-04 (Wave 2) can proceed: the backend env injection and managed config file are complete
- The managed opencode-tui.json is written to OS-standard config dir (macOS: ~/Library/Application Support/agenthub/, Linux: ~/.config/agenthub/)
- Full functional verification (OpenCode session respecting theme changes) deferred to Plan 71-04 UAT

## Self-Check: PASSED

- FOUND: internal/daemon/engine.go
- FOUND: internal/daemon/engine_test.go
- FOUND: 71-02-SUMMARY.md
- FOUND: aa9544f (Task 1 commit)
- FOUND: dc73386 (Task 2 commit)

---
*Phase: 71-opencode-theming-fix*
*Completed: 2026-04-13*
