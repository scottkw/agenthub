---
phase: 05-qr-codes-status-indicators
plan: "04"
subsystem: status
tags: [go, regex, ansi, relay, status-detector]

# Dependency graph
requires:
  - phase: 02-session-registry-websocket-relay
    provides: relay.ParseFrame, MakeOutputFrame, MsgOutput constant
  - phase: 05-qr-codes-status-indicators
    provides: status.Detector, Watch(), HubLike interface
provides:
  - "Watch() strips relay framing before feeding detector — binary 0x01 prefix no longer pollutes rolling tail"
  - "reANSI covers OSC sequences — \x1b]...\x07 stripped before pattern matching"
  - "Scrollback snapshot MsgOutput prefix stripped before Feed()"
  - "TestWatch_IdleTransition, TestWatch_NonOutputFrameIgnored, TestStripANSI_OSC tests"
affects: [status-indicators, tab-dot, UAT-test-2]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "relay.ParseFrame used as guard in Watch() loop — only MsgOutput frames reach detector"
    - "OSC regex alternation: \\x1b\\][^\\x07\\x1b]*(?:\\x07|\\x1b\\\\) covers both BEL and ST termination"

key-files:
  created: []
  modified:
    - internal/status/detector.go
    - internal/status/detector_test.go

key-decisions:
  - "reANSI extended with OSC alternation using character class exclusion [^\\x07\\x1b]* for correct greedy matching without backtracking hazard"
  - "Scrollback snapshot: strip only leading MsgOutput byte — correct for common case where snapshot is one large concatenated blob"
  - "Watch() uses relay.ParseFrame() guard: err != nil || msgType != MsgOutput → continue"

patterns-established:
  - "Frame stripping pattern: always call relay.ParseFrame() and check msgType before feeding raw bytes to non-relay consumers"

requirements-completed: [STAT-01, STAT-02]

# Metrics
duration: 8min
completed: 2026-03-18
---

# Phase 5 Plan 04: Status Detector Gap Closure Summary

**Fixed status dot blue-stuck UAT failure: Watch() now strips relay framing via ParseFrame() and reANSI covers OSC sequences Claude Code emits before the prompt**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-18T20:25:00Z
- **Completed:** 2026-03-18T20:33:00Z
- **Tasks:** 1 (TDD)
- **Files modified:** 2

## Accomplishments
- Extended `reANSI` to match OSC sequences (`\x1b]...\x07` BEL-terminated and `\x1b]...\x1b\` ST-terminated) that Claude Code emits for window title updates immediately before the idle prompt
- Fixed `Watch()` loop: each frame parsed via `relay.ParseFrame()`, non-`MsgOutput` frames skipped, only payload fed to detector — eliminates binary framing byte `0x01` from rolling tail
- Fixed scrollback snapshot feed: strips leading `MsgOutput` byte before `Feed()` so initial status seed is clean
- Added three new tests: `TestWatch_IdleTransition`, `TestWatch_NonOutputFrameIgnored`, `TestStripANSI_OSC`

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix Watch() frame stripping and extend reANSI for OSC sequences** - `319bfed` (feat)

**Plan metadata:** (docs commit follows)

_Note: TDD task — RED phase confirmed TestStripANSI_OSC failed (OSC not stripped); GREEN phase fixed all three issues._

## Files Created/Modified
- `internal/status/detector.go` - Extended reANSI, fixed Watch() loop frame stripping, fixed scrollback snapshot stripping
- `internal/status/detector_test.go` - Added TestWatch_IdleTransition, TestWatch_NonOutputFrameIgnored, TestStripANSI_OSC

## Decisions Made
- OSC regex uses `[^\x07\x1b]*` character class exclusion rather than `.*?` lazy match — avoids potential backtracking on long sequences and correctly handles both terminator forms
- Scrollback snapshot: strip only the leading byte if it equals `MsgOutput` — safe for the common case (single large concatenated blob); live-stream fix via `ParseFrame()` is the critical correctness path

## Deviations from Plan

None - plan executed exactly as written. TDD RED phase revealed that `TestWatch_IdleTransition` and `TestWatch_NonOutputFrameIgnored` already passed before the fix (the `0x01` prefix at frame start doesn't interfere with `❯\s*$` matching at end); `TestStripANSI_OSC` correctly failed as expected for the OSC regex gap.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Status detector correctly classifies idle state from framed Claude Code output — UAT test 2 ("dot stays blue") should now pass
- All three gap-closure fixes (05-04, 05-05, 05-06) can be re-verified with the UAT harness
- Full test suite green with race detector

---
*Phase: 05-qr-codes-status-indicators*
*Completed: 2026-03-18*
