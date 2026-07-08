---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
plan: 05
subsystem: infra
tags: [go, slog, wails, daemon-client, exit-poller, diagnostics]

requires:
  - phase: 175-01
    provides: "Live timed diagnosis of BUG-03 (#126) — VERDICT DISPROVED, selecting this plan's fix branch"
  - phase: 175-02
    provides: "shouldContinuePolling pure helper extraction + app_poll_test.go regression pin for the 300s deadline"
provides:
  - "Diagnostic logging around the RESEARCH-flagged candidate stall site (daemon client HTTP round-trip) at its app.go call site"
  - "Structured, non-silent terminal-exit logging for every pollSessionStatus branch that does not emit session:exit (daemon-gone, session-removed, deadline-expiry)"
affects: [175-07]

tech-stack:
  added: []
  patterns:
    - "log/slog structured Warn logging at terminal/exit branches only, no per-tick spam"

key-files:
  created: []
  modified:
    - app.go

key-decisions:
  - "Implemented the DISPROVED branch only: kept the fixed 300s exit-poll deadline (shouldContinuePolling / maxPollWindow) unchanged, per the 175-01 live diagnosis VERDICT"
  - "Instrumented the ListSessions() call site in app.go (elapsed-time threshold warn) instead of editing internal/daemon/client.go, honoring this plan's declared files_modified scope"
  - "Added slog.Warn to all three non-exit-emitting terminal branches of pollSessionStatus: daemon-gone, session-removed, and deadline-expiry"

patterns-established:
  - "Terminal-exit-only structured logging: log once on the way out of a polling loop, never per-tick"

requirements-completed: [BUG-03]

coverage:
  - id: D1
    description: "175-01 DISPROVED verdict honored — fixed 300s exit-poll deadline left unchanged (no behavior change to shouldContinuePolling)"
    requirement: BUG-03
    verification:
      - kind: unit
        ref: "app_poll_test.go#TestShouldContinuePolling (unchanged, still GREEN — proves no accidental behavior drift)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Diagnostic logging added around the suspect daemon-client stall path (ListSessions round-trip) at its app.go call site"
    requirement: BUG-03
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... clean; grep -n slog app.go shows the instrumentation"
        status: pass
    human_judgment: true
    rationale: "The logging is a diagnostic instrument for a future recurrence, not a testable behavior change; there is no live-reproducible stall to assert against in this session (BUG-03 did not reproduce in 175-01's live run)."
  - id: D3
    description: "Every non-exit-emitting terminal branch of pollSessionStatus (daemon-gone, session-removed, deadline-expiry) now logs sessionID + reason exactly once, closing the silent-give-up failure mode"
    requirement: BUG-03
    verification:
      - kind: other
        ref: "code inspection: app.go pollSessionStatus — 3 slog.Warn call sites, one per terminal branch, no per-tick logging"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-07-08
status: complete
---

# Phase 175 Plan 05: BUG-03 Exit-Poll Diagnosis-Gated Fix Summary

**Implemented the 175-01 DISPROVED branch: kept the fixed 300s exit-poll deadline unchanged and added structured slog diagnostics around the suspect daemon-client stall path plus every non-exit-emitting terminal branch of `pollSessionStatus`, closing the silent-give-up failure mode without a behavior change.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-08T14:17:10-05:00 (immediately after 175-04's plan-metadata commit)
- **Completed:** 2026-07-08T14:22:23-05:00
- **Tasks:** 2 completed
- **Files modified:** 1 (`app.go`)

## Branch Selection (gating decision)

The 175-01 live timed diagnosis VERDICT is **DISPROVED**: a shared session that lived
**past** the fixed 300s exit-poll window (`app.go` `pollSessionStatus`, `maxPollWindow = 300 *
time.Second`) still auto-closed its tab on exit in the live repro (case 2: >5min shared session,
exited, tab closed normally). BUG-03 did not reproduce across baseline, >5-min shared, or <5-min
shared cases in that session.

Per the plan's branch-selection gate, this plan implements the **DISPROVED branch only**:

- The deadline seam (`shouldContinuePolling`, `maxPollWindow`) is **unchanged** — no removal, no
  re-arming. `TestShouldContinuePolling` (pinned in 175-02) required zero edits and stays GREEN,
  proving no accidental behavior drift.
- Diagnostic logging was added instead of a behavioral fix: (1) around the RESEARCH-flagged
  candidate hang site — the un-timeout'd `http.Client` in `internal/daemon/client.go` — measured
  at its `app.go` call site (this plan's `files_modified` does not include `client.go`, so no
  edits were made there); (2) at every terminal exit of `pollSessionStatus` that does not emit
  `session:exit`, so a future recurrence of BUG-03 is directly diagnosable instead of silent.

The **CONFIRMED branch** (remove/re-arm the deadline) was **NOT** implemented — that would have
been a blind behavioral change contradicted by the live diagnosis.

## Accomplishments

- Added `log/slog`-based instrumentation in `pollSessionStatus` timing each `a.client.ListSessions()`
  call; a `slog.Warn` fires if a single round-trip exceeds 2s, naming `sessionId`, `elapsed`, and
  `err` — directly targeting the DIAGNOSIS's flagged candidate stall site without touching
  `internal/daemon/client.go`.
- Added `slog.Warn` at all three non-exit-emitting terminal branches of `pollSessionStatus`:
  - `daemon-gone` (5 consecutive `ListSessions` errors)
  - `session-removed` (session no longer present in `ListSessions`)
  - `deadline-expiry` (the exit-poll window elapsed without a stopped state or the session
    disappearing) — a DISPROVED-branch-only case, now non-silent since the deadline itself
    remains in place.
- No per-tick log spam: each branch logs exactly once, on the way out.

## Task Commits

Each task was committed atomically:

1. **Task 1: Apply the diagnosis-selected BUG-03 fix (DISPROVED branch — instrumentation, deadline unchanged)** - `040eb2b3` (fix)
2. **Task 2: Add a debuggable signal so a future watch failure is never silent** - `e70e720c` (fix)

_Note: This plan is `tdd="true"` on Task 1, but since the DISPROVED branch changes no pure logic
(only adds side-effect logging around the daemon call), the RED/GREEN/REFACTOR cycle did not apply
— `TestShouldContinuePolling` remained the correct regression guard unmodified, per the explicit
branch-selection instruction to only touch `app_poll_test.go` if pure logic changed._

## Files Created/Modified

- `app.go` - Added `log/slog` import; `pollSessionStatus` now (a) times each `ListSessions()`
  call and warns on a >2s round-trip (candidate daemon-client stall instrumentation), and (b) logs
  a structured reason (`sessionId` + `reason`) on each of its three non-`session:exit` terminal
  exits (`daemon-gone`, `session-removed`, `deadline-expiry`). `shouldContinuePolling` and
  `maxPollWindow` (300s) are byte-for-byte unchanged.

## Decisions Made

- Implemented the DISPROVED branch exclusively — the fixed 300s deadline stays in place; no
  CONFIRMED-branch code (deadline removal/re-arm) was written.
- Kept the instrumentation entirely inside `app.go`, scoped to this plan's declared
  `files_modified` (`app.go`, `app_poll_test.go`); did not edit `internal/daemon/client.go` even
  though it is the DIAGNOSIS's named candidate hang site.
- Left `app_poll_test.go` untouched since no pure logic in `shouldContinuePolling` changed — the
  existing `TestShouldContinuePolling` GREEN result is itself the proof that this plan's fix
  introduced no deadline-behavior regression.

## Deviations from Plan

None - plan executed exactly as written per the DISPROVED branch instructions supplied in the
execution prompt (`CRITICAL_BRANCH_SELECTION`), which take precedence over the PLAN.md's default
(CONFIRMED-branch-shaped) task language.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The threat-model mitigation `T-175-05-01` (poller with no deadline, CONFIRMED-branch DoS
  concern) does not apply — the deadline was never removed.
- `T-175-05-02` (new diagnostic log lines, information disclosure) is satisfied: all new log
  lines contain only `sessionId` + fixed cause strings/timings — no capability tokens, no session
  content.
- Per the plan's `<verification>` section, the live timed repro re-run (shared session >5min →
  exit → tab closes) should be captured as a new M-NN manual-UAT item in 175-07's TESTING.md
  reconciliation, along with a note that BUG-03 was **non-reproducible** in the 175-01 live run
  and this plan's deliverable is the diagnostic instrumentation, not a behavior change.
- Ready for 175-06 (BUG-02 WS close-reason fix) and 175-07 (final wave: TESTING.md
  reconciliation + new manual UAT items).

---
*Phase: 175-web-share-remote-viewer-windowing-bug-fixes*
*Completed: 2026-07-08*

## Self-Check: PASSED

- FOUND: app.go
- FOUND: .planning/phases/175-web-share-remote-viewer-windowing-bug-fixes/175-05-SUMMARY.md
- FOUND commit: 040eb2b3
- FOUND commit: e70e720c
