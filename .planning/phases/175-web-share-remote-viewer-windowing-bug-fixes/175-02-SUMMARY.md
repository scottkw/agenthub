---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
plan: 02
subsystem: testing
tags: [go, tdd, red-scaffold, websocket, vt-emulator, exit-poll, clock-injection]

requires:
  - phase: 175
    provides: BUG-02/03/04 root-cause research from 175-RESEARCH.md
provides:
  - Pure injectable-clock helper shouldContinuePolling() extracted from app.go exit-poll loop
  - RED (skip-guarded) Go test asserting web-guest WS close on hub.Done() carries a code + reason (BUG-02)
  - RED (skip-guarded) Go test asserting reconnect after ring-wrap-past-alt-screen-enter reconstructs correct buffer mode (BUG-04)
affects: [175-04, 175-05, 175-06]

tech-stack:
  added: []
  patterns:
    - "Injectable-clock pure helper (pollStart, now, maxWindow) for time-based logic under test"
    - "RED scaffolding via t.Skip(\"RED until 175-XX ...\") markers referencing the implementing plan"

key-files:
  created:
    - app_poll_test.go
    - internal/webserver/session_ended_test.go
    - internal/relay/scrollback_altscreen_test.go
  modified:
    - app.go

key-decisions:
  - "RED tests use t.Skip markers (not failing assertions) so the suite stays green while scaffolding — later plans remove the Skip and implement"
  - "shouldContinuePolling extraction is strictly behavior-preserving (still a fixed 300s window from poll start); no observable change until 175-05"

patterns-established:
  - "Skip-guarded RED tests: each carries a \"RED until 175-NN\" reason naming the plan that turns it GREEN"
  - "Time logic tested via a pure helper with an injected clock rather than wall-clock sleeps"

requirements-completed: [BUG-02, BUG-03, BUG-04]

coverage:
  - id: D1
    description: "app.go exit-poll deadline math extracted into pure injectable-clock helper shouldContinuePolling()"
    requirement: BUG-03
    verification:
      - kind: unit
        ref: "app_poll_test.go#TestShouldContinuePolling (in-window / at-deadline / zero-window)"
        status: pass
    human_judgment: false
  - id: D2
    description: "RED scaffold: web-guest WS close on hub.Done() must carry a close code + reason (BUG-02)"
    requirement: BUG-02
    verification:
      - kind: unit
        ref: "internal/webserver/session_ended_test.go#TestSessionEnd_HubDone_CarriesCloseReason (skip-guarded RED until 175-06)"
        status: unknown
    human_judgment: false
    rationale: "Intentionally skip-guarded RED; 175-06 removes the Skip and turns it GREEN"
  - id: D3
    description: "RED scaffold: reconnect after ring-wrap-past-alt-screen-enter reconstructs correct buffer mode (BUG-04)"
    requirement: BUG-04
    verification:
      - kind: unit
        ref: "internal/relay/scrollback_altscreen_test.go#TestScrollbackAltScreenReplay (skip-guarded RED until 175-04)"
        status: unknown
    human_judgment: false
    rationale: "Intentionally skip-guarded RED; 175-04 adds the live per-hub VT emulator + RenderSnapshot and turns it GREEN"

duration: 6min
completed: 2026-07-08
status: complete
---

# Phase 175 / Plan 02: Wave-0 Test Scaffolding Summary

**Extracted the exit-poll deadline into a pure injectable-clock helper (`shouldContinuePolling`) and laid down skip-guarded RED Go tests for BUG-02 (WS close reason) and BUG-04 (alt-screen reconnect replay) that Waves 1–2 must turn GREEN.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-07-08T18:05Z
- **Completed:** 2026-07-08T18:11Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Behavior-preserving extraction of `app.go:386`'s inline `time.Now().Before(deadline)` loop condition into the pure, side-effect-free `shouldContinuePolling(pollStart, now, maxWindow)` helper (app.go:379-401), with unit tests for in-window / at-deadline / zero-window cases.
- BUG-02 RED scaffold: `TestSessionEnd_HubDone_CarriesCloseReason` asserting the web-guest WS close carries a code + reason on `hub.Done()`; skip-guarded until 175-06.
- BUG-04 RED scaffold: `TestScrollbackAltScreenReplay` asserting reconnect after the raw ring wraps past `ESC[?1049h` reconstructs the correct buffer mode; skip-guarded until 175-04.

## Task Commits

1. **Task 1: extract exit-poll deadline into pure helper** - `9841549d` (feat)
2. **Task 2: RED scaffold for BUG-02 WS close-on-session-end reason** - `591e4a66` (test)
3. **Task 3: RED scaffold for BUG-04 reconnect alt-screen replay** - `bdb42ff1` (test)

## Files Created/Modified
- `app.go` - Added `shouldContinuePolling` pure helper; `pollSessionStatus` loop now calls it
- `app_poll_test.go` - Unit tests for the helper (GREEN)
- `internal/webserver/session_ended_test.go` - BUG-02 RED scaffold (skip-guarded)
- `internal/relay/scrollback_altscreen_test.go` - BUG-04 RED scaffold + supporting scrollback tests

## Decisions Made
- RED tests are skip-guarded (`t.Skip("RED until 175-NN ...")`) rather than hard-failing, so `go test ./...` and CI stay green while scaffolding. Each Skip names the plan that unskips it.
- The helper extraction is strictly behavior-preserving; the fixed-300s-window bug (BUG-03) is left intact for 175-05 to fix, gated on the 175-01 diagnosis.

## Deviations from Plan
None - plan executed as written.

## Issues Encountered
The executor subagent hit a transient API 500 immediately after committing Task 3, before writing this SUMMARY.md and updating tracking. All three task commits were verified present and correct; `go build ./...` passes and `go test ./...` is green (RED scaffolds skip). SUMMARY.md + tracking were closed out by the orchestrator via the safe-resume path.

## User Setup Required
None.

## Next Phase Readiness
- 175-04 can now implement the live per-hub VT emulator to turn `TestScrollbackAltScreenReplay` GREEN.
- 175-06 can now implement the WS close reason to turn `TestSessionEnd_HubDone_CarriesCloseReason` GREEN.
- 175-05 (BUG-03) will replace the fixed-window logic inside/around `shouldContinuePolling`, gated on the 175-01 live diagnosis.

---
*Phase: 175-web-share-remote-viewer-windowing-bug-fixes*
*Completed: 2026-07-08*
