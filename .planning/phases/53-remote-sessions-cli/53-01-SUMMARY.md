---
phase: 53-remote-sessions-cli
plan: 01
subsystem: cli
tags: [remote-sessions, tailnet, cli, tabwriter, json]

# Dependency graph
requires:
  - phase: 50-tailscale-peer-discovery
    provides: "tailnet.Peer struct, ListTailnetPeers daemon client method, DefaultProbePort"
  - phase: 52-remote-sessions-gui-panel
    provides: "fetchRemoteSessions pattern in app.go, /api/sessions endpoint shape"
provides:
  - "cmd_remote.go with shared CLI remote session helpers (parseRemoteID, resolveRemotePeer, fetchPeerSessions)"
  - "CLIRemoteSession struct for CLI use"
  - "cmdList with HOST column grouping local and remote sessions"
  - "--local flag and --json structured output with local/remote arrays"
affects: [53-02, remote-session-attach, remote-session-kill]

# Tech tracking
tech-stack:
  added: []
  patterns: [fetchPeerSessionsWithClient for testable HTTP fetching, listOutput struct for JSON grouping]

key-files:
  created: [cmd_remote.go, cmd_remote_test.go]
  modified: [cmd_cli.go, cmd_cli_test.go]

key-decisions:
  - "fetchPeerSessionsWithClient internal helper for httptest injection — avoids TLS complexity in tests"
  - "listOutput struct with local/remote JSON structure instead of flat array — enables grouped display"
  - "Silent error handling for ListTailnetPeers and fetchPeerSessions — graceful degradation matches app.go pattern"

patterns-established:
  - "fetchPeerSessionsWithClient pattern: separate production TLS client from test-injectable client"
  - "listRemoteGroup pattern: group remote sessions by peer hostname for CLI display"

requirements-completed: [REM-04]

# Metrics
duration: 10min
completed: 2026-04-07
---

# Phase 53 Plan 01: Remote Session CLI Helpers & List Command Summary

**Shared remote session helpers (parseRemoteID, resolveRemotePeer, fetchPeerSessions) and enhanced cmdList with HOST column grouping local and remote sessions**

## Performance

- **Duration:** 10 min (615s)
- **Started:** 2026-04-07T20:58:11Z
- **Completed:** 2026-04-07T21:08:26Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Created cmd_remote.go with CLIRemoteSession struct and three shared helpers for CLI remote session operations
- Enhanced cmdList with HOST column showing "(local)" for local sessions and peer hostname for remote sessions
- Added --local flag to skip remote discovery and --json with structured local/remote output
- Comprehensive test coverage: 15 new/updated tests all passing with race detector

## Task Commits

Each task was committed atomically:

1. **Task 1: Create cmd_remote.go with shared remote session helpers** - `cc3f7b4` (feat)
2. **Task 2: Update cmdList to show remote sessions grouped by host** - `05988e6` (feat)

## Files Created/Modified
- `cmd_remote.go` - CLIRemoteSession struct, parseRemoteID, resolveRemotePeer, fetchPeerSessions helpers
- `cmd_remote_test.go` - Tests for parseRemoteID, resolveRemotePeer, fetchPeerSessions (5 test functions)
- `cmd_cli.go` - Enhanced cmdList with HOST column, --local flag, remote session grouping, listOutput JSON struct
- `cmd_cli_test.go` - Updated existing JSON tests, added WithHostColumn, LocalFlag, JSON_WithHostField tests

## Decisions Made
- Used fetchPeerSessionsWithClient internal helper pattern so tests can inject httptest.NewTLSServer's client without dealing with production TLS configuration
- Changed JSON output from flat SessionInfo array to listOutput struct with local/remote grouping — breaking change for --json consumers but more useful for remote-aware tooling
- Silent error handling for ListTailnetPeers and fetchPeerSessions errors — matches existing GetRemoteSessions pattern in app.go for graceful degradation

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- cmd_remote.go helpers (parseRemoteID, resolveRemotePeer) ready for use by 53-02 (remote attach/kill commands)
- listOutput struct and listRemoteGroup type available for future CLI commands that need remote session data

## Self-Check: PASSED

All 4 files exist. Both commit hashes verified. No stubs found.

---
*Phase: 53-remote-sessions-cli*
*Completed: 2026-04-07*
