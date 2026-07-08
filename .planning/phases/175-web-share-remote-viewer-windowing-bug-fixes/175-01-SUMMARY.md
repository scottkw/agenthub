---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
plan: 01
subsystem: testing
tags: [bug-03, live-uat, diagnosis, exit-poll, human-verify]

requires:
  - phase: 175
    provides: BUG-03 root-cause research + exit-poll deadline hypothesis (175-RESEARCH.md)
provides:
  - Live timed BUG-03 (#126) diagnosis with a DISPROVED verdict
  - Fix-branch selection for 175-05 (diagnostic-logging pass, NOT deadline removal)
affects: [175-05, 175-07]

tech-stack:
  added: []
  patterns:
    - "Gate a code-fix plan on a live human diagnostic when the root cause cannot be confirmed statically"

key-files:
  created:
    - .planning/phases/175-web-share-remote-viewer-windowing-bug-fixes/175-01-DIAGNOSIS.md
  modified: []

key-decisions:
  - "DISPROVED: app.go:386's fixed 300s exit-poll deadline is NOT BUG-03's root cause — a >5-min shared session still auto-closed its tab"
  - "BUG-03 did not reproduce across baseline / >5-min shared / <5-min shared cases in this timed run"
  - "175-05 takes the DISPROVED branch: add a diagnostic-logging pass rather than a blind behavioral fix"

patterns-established:
  - "Human-verify checkpoint gates the downstream fix plan's branch selection"

requirements-completed: [BUG-03]

coverage:
  - id: D1
    description: "Live timed BUG-03 reproduction run + written DISPROVED verdict selecting 175-05's fix branch"
    requirement: BUG-03
    verification:
      - kind: manual_procedural
        ref: "175-01-DIAGNOSIS.md (operator-run timed live repro: baseline / >5min shared / <5min shared)"
        status: pass
    human_judgment: true
    rationale: "Exit-detection poller runs in the Wails Go process against a real daemon; only a live, wall-clock-timed human run can confirm/disprove the deadline hypothesis"

duration: 1min
completed: 2026-07-08
status: complete
---

# Phase 175 / Plan 01: BUG-03 Live Diagnosis Summary

**Live timed reproduction DISPROVED the exit-poll-deadline hypothesis — a >5-minute shared session still auto-closed its tab; BUG-03 did not reproduce, steering 175-05 to a diagnostic-logging pass instead of removing the deadline.**

## Performance

- **Duration:** ~1 min (verdict recording; live repro run by operator)
- **Tasks:** 1 (human-verify checkpoint)
- **Files modified:** 1 created (diagnosis doc)

## Accomplishments
- Operator ran the mandatory timed live repro (baseline unshared/immediate; shared >5 min; shared <5 min tie-breaker) against a live app + real daemon.
- All three cases auto-closed the tab on exit — **BUG-03 did not reproduce**, and the fixed 300s deadline at `app.go:386` is **not** the mechanism.
- Recorded a **DISPROVED** verdict in `175-01-DIAGNOSIS.md` and selected 175-05's fix branch: add a diagnostic-logging pass around the exit-detection / tab-close path (candidate hang site: un-timeout'd `http.Client` at `internal/daemon/client.go:37`), not a blind deadline removal.

## Task Commits
1. **Task 1: Live timed BUG-03 repro + verdict** — committed with this summary (diagnostic-only plan, no source changes).

## Files Created/Modified
- `175-01-DIAGNOSIS.md` — both timed-run outcomes + DISPROVED verdict + 175-05 branch selection.

## Decisions Made
- DISPROVED the deadline hypothesis based on the >5-min shared case auto-closing normally.
- 175-05 will ship instrumentation (diagnostic-logging), not a behavioral deadline change.

## Deviations from Plan
None — the plan explicitly anticipated a DISPROVED outcome and pre-specified the fallback branch.

## Issues Encountered
BUG-03 was non-reproducible in this timed run. This is itself a finding: the live-UAT gate (175-07 / M-3x) should record non-reproduction and treat the shipped instrumentation as the deliverable unless a reliable repro is later found.

## User Setup Required
None.

## Next Phase Readiness
- 175-05 is unblocked and now scoped to the DISPROVED (diagnostic-logging) branch.

---
*Phase: 175-web-share-remote-viewer-windowing-bug-fixes*
*Completed: 2026-07-08*
