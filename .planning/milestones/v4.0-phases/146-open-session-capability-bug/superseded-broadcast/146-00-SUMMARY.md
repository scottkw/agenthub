---
phase: 146-open-session-capability-bug
plan: "00"
subsystem: testing
tags: [go, typescript, vitest, tdd, capability, join-code, remote-session]

# Dependency graph
requires:
  - phase: 146-open-session-capability-bug
    provides: PATTERNS.md and VALIDATION.md defining FIX-03 contract shapes

provides:
  - "RED Go tests for WebServer.SetJoinCodeIssuer + ro_join_code/rw_join_code embed (TestSessionsMeta_EmbedJoinCodes, TestSessionsMeta_NilIssuer)"
  - "RED Go tests for api.mintSessionJoinCodes — distinct codes, grant registration, perms assertions (TestMintSessionJoinCodes)"
  - "Updated RB-03 allowed-keys map in TestSessionsMeta_NoCapInResponse (ro_join_code + rw_join_code added, sensitiveKeys unchanged)"
  - "RED TS source-inspection tests for handleOpenRemoteSession exchange-then-open flow (App.open-remote.test.tsx)"
  - "RED TS tests for adaptRemoteSession roJoinCode/rwJoinCode pass-through (remoteAdapter.test.ts)"

affects: [146-01, 146-02, 146-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wave 0 TDD RED-gate: tests reference undefined production symbols to lock contract before implementation"
    - "Source-inspection pattern (App.tsx?raw) for testing callbacks that are hard to mount in full DOM"
    - "mintCodesTestSetup: mirror of issueCapsTestSetup for join-code mint tests"

key-files:
  created:
    - internal/webserver/sessions_meta_embed_test.go
    - internal/daemon/mint_join_codes_test.go
    - frontend/src/components/__tests__/App.open-remote.test.tsx
    - frontend/src/lib/__tests__/remoteAdapter.test.ts
  modified:
    - internal/webserver/sessions_meta_test.go

key-decisions:
  - "mintCodesTestSetup mirrors issueCapsTestSetup exactly — same config pattern, same capability state setup, no new infrastructure"
  - "remoteAdapter.test.ts is a new file (did not exist); plan said 'extend' but creating is correct per plan's files_modified list"
  - "App.open-remote.test.tsx uses source-inspection (raw import) not DOM rendering — established pattern from App.fileBrowserMode.test.tsx"
  - "RED state confirmed: Go vet shows SetJoinCodeIssuer + mintSessionJoinCodes undefined; vitest shows 6 failing assertions"

patterns-established:
  - "Wave 0 TDD gate: Go test files in same package as production code reference undefined symbols — compile failure IS the RED state"
  - "Frontend source-inspection: import App.tsx?raw + slice at handler name to assert handler body contract"

requirements-completed: [FIX-03]

# Metrics
duration: 25min
completed: 2026-06-22
---

# Phase 146 Plan 00: Wave 0 RED Tests for FIX-03 Join-Code Embed Summary

**5 new test files (3 Go + 2 TS) that lock ro_join_code/rw_join_code field-shape contract and exchange-then-open flow before any production implementation — all RED until Waves 1+2**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-06-22T00:00:00Z
- **Completed:** 2026-06-22T00:25:00Z
- **Tasks:** 2
- **Files modified:** 5 (4 created, 1 updated)

## Accomplishments

- Wrote `TestSessionsMeta_EmbedJoinCodes` + `TestSessionsMeta_NilIssuer` in `sessions_meta_embed_test.go` — RED (reference undefined `ws.SetJoinCodeIssuer`)
- Wrote `TestMintSessionJoinCodes` in `mint_join_codes_test.go` — RED (reference undefined `api.mintSessionJoinCodes`), asserts distinct codes, grant registration, RO "read" / RW "read,write" perms
- Updated `sessions_meta_test.go` allowed-keys map: added `ro_join_code` and `rw_join_code`; RB-03 sensitiveKeys blacklist (cap/token/grant/grants/content/key/signing_key/hmac) unchanged
- Created `App.open-remote.test.tsx` with 5 source-inspection assertions — all RED (current handler is bare `BrowserOpenURL(url)`)
- Created `remoteAdapter.test.ts`: 2 RED join-code pass-through tests + 7 GREEN existing-contract tests (url/hostname/webEnabled/filtering)

## Task Commits

1. **Task 1: Write Go owner-side scaffold tests (webserver + daemon)** - `76d8e41b` (test)
2. **Task 2: Write frontend viewer-side scaffold tests (open-remote + adapter)** - `53fe8d71` (test)

## Files Created/Modified

- `internal/webserver/sessions_meta_embed_test.go` - New: TestSessionsMeta_EmbedJoinCodes + TestSessionsMeta_NilIssuer (RED)
- `internal/daemon/mint_join_codes_test.go` - New: TestMintSessionJoinCodes with mintCodesTestSetup (RED)
- `internal/webserver/sessions_meta_test.go` - Updated: ro_join_code + rw_join_code added to RB-03 allowed-keys map
- `frontend/src/components/__tests__/App.open-remote.test.tsx` - New: 5 RED source-inspection assertions for handleOpenRemoteSession
- `frontend/src/lib/__tests__/remoteAdapter.test.ts` - New: RED join-code pass-through tests + GREEN existing-contract tests

## Decisions Made

- Used `mintCodesTestSetup` as mirror of `issueCapsTestSetup` — same pattern, zero new infrastructure
- `remoteAdapter.test.ts` created as new file (did not exist prior); plan said "extend" which is correct since the file is listed in `files_modified`
- Source-inspection pattern (`App.tsx?raw`) chosen over DOM mounting — established pattern, hermetic, no wailsjs stub machinery needed
- RED state verified via `go vet` (undefined symbol errors) and `pnpm test --run` (6 failing assertions)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Known Stubs

None — this is a test-only plan. No production code was modified; no stubs introduced.

## Threat Flags

None — this plan creates only test files. No new network endpoints, auth paths, or schema changes introduced.

## Self-Check

- [x] `internal/webserver/sessions_meta_embed_test.go` exists and contains `TestSessionsMeta_EmbedJoinCodes`
- [x] `internal/daemon/mint_join_codes_test.go` exists and contains `TestMintSessionJoinCodes`
- [x] `frontend/src/components/__tests__/App.open-remote.test.tsx` exists and contains `handleOpenRemoteSession`
- [x] `frontend/src/lib/__tests__/remoteAdapter.test.ts` exists and contains `roJoinCode`
- [x] `go vet ./internal/webserver/...` reports `SetJoinCodeIssuer undefined` (RED confirmed)
- [x] `go vet ./internal/daemon/...` reports `mintSessionJoinCodes undefined` (RED confirmed)
- [x] `pnpm test -- App.open-remote remoteAdapter --run` reports 6 failing tests (RED confirmed)
- [x] Commits `76d8e41b` and `53fe8d71` exist in git log

## Self-Check: PASSED

## Next Phase Readiness

Wave 0 complete. All test contracts locked. Waves 1+2 can now implement production code to make these tests pass:
- Wave 1 (Plan 01): Add `SetJoinCodeIssuer` + `mintSessionJoinCodes` + `ro_join_code`/`rw_join_code` fields to Go production code
- Wave 2 (Plan 02): Rewrite `handleOpenRemoteSession` + extend `adaptRemoteSession` in TypeScript

---
*Phase: 146-open-session-capability-bug*
*Completed: 2026-06-22*
