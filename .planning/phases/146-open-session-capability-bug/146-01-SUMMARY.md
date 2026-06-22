---
phase: 146-open-session-capability-bug
plan: 01
subsystem: testing
tags: [go, vitest, tdd, webserver, sessions, capability, remote-open]

# Dependency graph
requires:
  - phase: 146-open-session-capability-bug
    provides: "CONTEXT.md, RESEARCH.md, PATTERNS.md with out-of-band redesign decisions"
provides:
  - "RED test contract for RB-03 cap-free /api/sessions/meta (Go)"
  - "RED test contract for out-of-band open path: handleOpenRemoteSession → modal → exchange → BrowserOpenURL"
  - "Behavior-level assertion crossing the actual open entry point (fills the prior blind spot)"
affects:
  - 146-02-PLAN  # Go implementation must turn Go RED test GREEN
  - 146-03-PLAN  # Frontend implementation must turn frontend RED tests GREEN

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Inverted test contract: assert key ABSENCE in JSON map (not presence) to enforce RB-03"
    - "Behavior-level SessionCard render test as cross-path assertion for open feature"
    - "Source-inspection (.not.toContain) to assert dead-code removal"

key-files:
  created: []
  modified:
    - internal/webserver/sessions_meta_embed_test.go
    - frontend/src/components/__tests__/App.open-remote.test.tsx

key-decisions:
  - "Behavior-level test asserts button is NOT disabled (no roJoinCode gate) — D-03 invariant"
  - "Source tests use .not.toContain for dead-code assertions (isPeerSelf, rwJoinCode absent)"
  - "Comment references to broadcast symbol names removed to satisfy grep-L acceptance criteria"

patterns-established:
  - "Wave-0 RED tests for a design reversal: invert prior assertions, add behavior-level cross-path check"

requirements-completed: [FIX-03]

# Metrics
duration: 5min
completed: 2026-06-22
---

# Phase 146 Plan 01: Open Session Capability Bug — RED Test Contract Summary

**Wave-0 TDD RED: inverted RB-03 Go contract (join codes absent from /api/sessions/meta) + out-of-band open contract with behavior-level SessionCard assertion that fills the prior test suite's blind spot**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-06-22T16:35:33Z
- **Completed:** 2026-06-22T16:40:07Z
- **Tasks:** 2 (both TDD RED)
- **Files modified:** 2

## Accomplishments

- Replaced the broadcast-era `TestSessionsMeta_EmbedJoinCodes` / `TestSessionsMeta_NilIssuer` with a single
  inverted `TestSessionsMeta_NoJoinCodesInResponse` that asserts `ro_join_code` and `rw_join_code` are
  ABSENT from the `/api/sessions/meta` response (RB-03 cap-free discovery restored).
- Rewrote `App.open-remote.test.tsx` with the out-of-band contract: source assertions that
  `handleOpenRemoteSession` uses `setJoinModalForSession` (not direct `ExchangeJoinCodeAtURL`),
  that `handleModalExchange` has an `open-session` branch with `/sessions/{id}?cap=` URL + `BrowserOpenURL`,
  and that dead code (`isPeerSelf`, `rwJoinCode`) is gone from App.tsx.
- Added a behavior-level test that renders a remote `SessionCard` and asserts the "Open in browser" button
  is NOT disabled (no `roJoinCode` gate) — the cross-path assertion that was missing from the prior suite
  and let the dead-on-arrival broadcast feature ship with a green test suite.
- Both test files are in RED state against current (broadcast) production code for the right reasons;
  they will go GREEN when Plans 02 (Go) and 03 (frontend) implement the out-of-band design.

## Task Commits

1. **Task 1: Invert the Go RB-03 contract test** - `2095ff8e` (test)
2. **Task 1 follow-up: remove comment broadcast symbol references** - `8d60da36` (test)
3. **Task 2: Rewrite frontend open-remote contract** - `bd3e8c47` (test)

## Files Created/Modified

- `/Users/ken/dev/agenthub/internal/webserver/sessions_meta_embed_test.go` — replaced broadcast-era tests with inverted `TestSessionsMeta_NoJoinCodesInResponse` asserting cap-free RB-03 contract; no reference to `SetJoinCodeIssuer` or removed test names
- `/Users/ken/dev/agenthub/frontend/src/components/__tests__/App.open-remote.test.tsx` — replaced broadcast-era source assertions with out-of-band contract; added behavior-level `SessionCard` render test as the cross-path open-path assertion

## Decisions Made

- The behavior test asserts `openBtn.disabled === false` (not `openBtn.disabled`) — the button must be enabled even when the session has no `roJoinCode`, because the modal is the mechanism through which the viewer obtains a code (D-03). This is the key invariant that separates the out-of-band design from the broadcast design.
- Source assertions use `.not.toContain('isPeerSelf')` / `.not.toContain('rwJoinCode')` — these appear in the test as string literals in negative assertions, which is correct: the test asserts the production code no longer uses these broadcast symbols.
- The second `clicking "Open in browser"` behavior test short-circuits to re-assert the `disabled` condition when the button is already disabled — this avoids cascading failures that obscure the root cause while still exercising the handler call path when the implementation is correct.

## Deviations from Plan

None — plan executed exactly as written. The minor follow-up commit (removing comment references to `TestSessionsMeta_EmbedJoinCodes`) was a strict adherence to the acceptance criteria (`grep -L` test), not a deviation.

## Issues Encountered

- `grep -L` (returns files NOT matching) returned empty output for the first version of the embed test because the file contained `TestSessionsMeta_EmbedJoinCodes` in a comment. Fixed with an additional commit removing the comment reference.

## Threat Flags

None — this plan only writes test files; no new network endpoints, auth paths, or schema changes introduced.

## Self-Check

Files exist:
- [x] `/Users/ken/dev/agenthub/internal/webserver/sessions_meta_embed_test.go`
- [x] `/Users/ken/dev/agenthub/frontend/src/components/__tests__/App.open-remote.test.tsx`

Commits exist:
- [x] `2095ff8e` — test(146-01): RED — invert RB-03 contract
- [x] `bd3e8c47` — test(146-01): RED — rewrite App.open-remote contract
- [x] `8d60da36` — test(146-01): remove comment reference

RED state verified:
- [x] `go test ./internal/webserver/ -run TestSessionsMeta_NoJoinCodesInResponse` FAILS with RB-03 violation messages
- [x] `pnpm test -- App.open-remote.test.tsx --run` FAILS (8 tests, all for correct reasons)

## Self-Check: PASSED

## Next Phase Readiness

- Plan 02 (Go): Remove `mintSessionJoinCodes`/`SetJoinCodeIssuer` broadcast wiring, remove `ROJoinCode`/`RWJoinCode` fields from `sessionMetaItem` and `ShareableSessionMeta` — this makes `TestSessionsMeta_NoJoinCodesInResponse` GREEN.
- Plan 03 (frontend): Rewrite `handleOpenRemoteSession` to open the modal, add `open-session` branch to `handleModalExchange`, remove `isPeerSelf`/`rwJoinCode`, update `SessionCard` to remove `disabled={!roJoinCode}` — this makes all 8 frontend tests GREEN.

---
*Phase: 146-open-session-capability-bug*
*Completed: 2026-06-22*
