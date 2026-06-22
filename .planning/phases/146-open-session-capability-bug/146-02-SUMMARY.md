---
phase: 146-open-session-capability-bug
plan: 02
subsystem: go-backend
tags: [go, webserver, daemon, tailnet, tdd, capability, remote-open, security]

# Dependency graph
requires:
  - phase: 146-open-session-capability-bug
    plan: 01
    provides: "RED tests — TestSessionsMeta_NoJoinCodesInResponse (inverted RB-03 contract)"
provides:
  - "Broadcast wiring fully removed from production Go"
  - "RB-03 restored: GET /api/sessions/meta is cap-free (id/name/cli_type/status/url only)"
  - "TestSessionsMeta_NoJoinCodesInResponse GREEN"
  - "mintSessionJoinCodes and SetJoinCodeIssuer symbols deleted from codebase"
affects:
  - 146-03-PLAN  # frontend GREEN implementation next

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Surgical delete of superseded broadcast wiring: remove method + field + setter + wiring calls as a unit"
    - "Inverted TDD GREEN: remove the production code that caused the RED test to fail"

key-files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/daemon/api.go
    - internal/tailnet/sessions.go
    - internal/webserver/sessions_meta_test.go
  deleted:
    - internal/daemon/mint_join_codes_test.go

key-decisions:
  - "mintSessionJoinCodes removed as a complete unit (method + both wiring sites + field + setter)"
  - "issueCapabilitiesForSession and IssueCapabilities untouched — owner-side minting preserved for out-of-band flow"
  - "Comment text updated to avoid symbol name matches in grep-based verification"

requirements-completed: [FIX-03]

# Metrics
duration: 8min
completed: 2026-06-22
---

# Phase 146 Plan 02: Open Session Capability Bug — Go GREEN Summary

**Remove broadcast join-code wiring (mintSessionJoinCodes + SetJoinCodeIssuer + ROJoinCode/RWJoinCode fields), restore RB-03 cap-free /api/sessions/meta, and make the Plan 01 RED test GREEN**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-06-22
- **Completed:** 2026-06-22
- **Tasks:** 2
- **Files modified:** 3 (production), 1 (test), 1 deleted

## Accomplishments

- Deleted `mintSessionJoinCodes` method from `internal/daemon/api.go` (broadcast-only extraction of `issueCapabilitiesForSession`).
- Deleted both `ws.SetJoinCodeIssuer(a.mintSessionJoinCodes)` wiring calls from `api.go` (auto-start site at L458 and `handleWebServerStart` site at L995).
- Deleted `joinCodeIssuer` struct field and `SetJoinCodeIssuer` setter from `internal/webserver/server.go`.
- Removed `ROJoinCode`/`RWJoinCode` fields from `sessionMetaItem` in `server.go`.
- Removed `ROJoinCode`/`RWJoinCode` fields from `ShareableSessionMeta` in `internal/tailnet/sessions.go`.
- Restored `handleSessionsMeta` to cap-free discovery: builds items with `id`/`name`/`cli_type`/`status`/`url` only; no embed block.
- Reverted broadcast allow-list entries (`ro_join_code`, `rw_join_code`) from `TestSessionsMeta_NoCapInResponse` in `sessions_meta_test.go`.
- Deleted `internal/daemon/mint_join_codes_test.go` (tested the removed `mintSessionJoinCodes`; cap issuance covered by existing `TestIssueCapabilities_*`).
- `TestSessionsMeta_NoJoinCodesInResponse` (Plan 01 RED) is now GREEN.
- All three Go packages pass: `webserver`, `daemon`, `tailnet`.

## Task Commits

1. **Task 1: Remove broadcast wiring, restore cap-free discovery payload** — `7d02ac20`
2. **Task 2: Reconcile Go tests — GREEN RB-03 contract, delete broadcast-only test** — `ac7fd684`

## Files Created/Modified/Deleted

- `/Users/ken/dev/agenthub/internal/webserver/server.go` — removed `joinCodeIssuer` field, `SetJoinCodeIssuer` setter, `ROJoinCode`/`RWJoinCode` on `sessionMetaItem`, embed block in `handleSessionsMeta`; restored cap-free comment
- `/Users/ken/dev/agenthub/internal/daemon/api.go` — removed both `SetJoinCodeIssuer` wiring calls and entire `mintSessionJoinCodes` method; `issueCapabilitiesForSession` untouched
- `/Users/ken/dev/agenthub/internal/tailnet/sessions.go` — removed `ROJoinCode`/`RWJoinCode` from `ShareableSessionMeta`; restored cap-free RB-03 comment
- `/Users/ken/dev/agenthub/internal/webserver/sessions_meta_test.go` — reverted allowed-key map to 5 cap-free keys only (removed `ro_join_code`/`rw_join_code` entries and broadcast comment)
- `/Users/ken/dev/agenthub/internal/daemon/mint_join_codes_test.go` — DELETED (broadcast-only test)

## Decisions Made

- Comment text for the deleted broadcast fields uses "broadcast join-code fields" rather than repeating the exact symbol names (`ROJoinCode`, `RWJoinCode`) to satisfy the grep-based acceptance criteria without leaving misleading breadcrumbs. The comment still explains why the fields are gone and references CONTEXT.md D-10.
- `issueCapabilitiesForSession` and the owner-side cap-minting path (`IssueCapabilities`) are entirely untouched — the out-of-band flow (Plan 03 frontend + Share modal copy affordance) depends on these.

## Deviations from Plan

None — plan executed exactly as written. The comment text adjustment (using "broadcast join-code fields" in prose instead of repeating symbol names) is a minor cosmetic choice, not a behavioral deviation.

## Threat Flags

None — this plan only deletes code; it removes a security violation (credential broadcast) and introduces no new network endpoints, auth paths, or schema changes.

## Self-Check

Files verified:
- [x] `internal/webserver/server.go` — modified (no joinCodeIssuer field, no SetJoinCodeIssuer, no ROJoinCode/RWJoinCode)
- [x] `internal/daemon/api.go` — modified (no mintSessionJoinCodes, no SetJoinCodeIssuer wiring)
- [x] `internal/tailnet/sessions.go` — modified (no ROJoinCode/RWJoinCode on ShareableSessionMeta)
- [x] `internal/webserver/sessions_meta_test.go` — modified (cap-free allowed-key set)
- [x] `internal/daemon/mint_join_codes_test.go` — DELETED (confirmed absent)

Commits verified:
- [x] `7d02ac20` — fix(146-02): remove broadcast wiring
- [x] `ac7fd684` — fix(146-02): reconcile Go tests

Verification:
- [x] `go build ./...` PASS
- [x] `go test ./internal/webserver/ ./internal/daemon/ ./internal/tailnet/` all PASS
- [x] `TestSessionsMeta_NoJoinCodesInResponse` PASS (GREEN)
- [x] `broadcast_symbols_in_prod=0`
- [x] `issueCapabilitiesForSession` still present (count=1)
- [x] `mint_join_codes_test.go` deleted
- [x] `grep -c 'ro_join_code' sessions_meta_test.go` == 0

## Self-Check: PASSED
