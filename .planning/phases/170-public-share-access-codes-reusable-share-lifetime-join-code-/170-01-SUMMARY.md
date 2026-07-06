---
phase: 170-public-share-access-codes-reusable-share-lifetime-join-code-
plan: 01
subsystem: auth
tags: [go, capability, join-code, funnel, http]

# Dependency graph
requires: []
provides:
  - "JoinCodeManager.IssueReusable(token, ttl) — mints a code that survives repeated Exchange calls"
  - "JoinCodeManager.Revoke(code) — idempotent immediate invalidation"
  - "Exchange's success-path delete is conditional on entry.reusable, leaving single-use behavior unchanged"
  - "internal/webserver/join_test.go proving the reusable contract at the public /join/exchange HTTP boundary"
affects: [170-02, 171]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Reusable vs single-use join codes distinguished by a bool field on joinEntry, not a second data structure"
    - "Public-HTTP-boundary test file pattern (join_test.go) mirroring the existing joinExchangeTestServer harness style used elsewhere in webserver_test"

key-files:
  created:
    - internal/webserver/join_test.go
  modified:
    - internal/capability/joincode.go
    - internal/capability/joincode_test.go
    - TESTING.md

key-decisions:
  - "IssueReusable reuses Issue's exact crypto/rand + joinCodeEncoding path — no second RNG or alphabet, preserving 40-bit entropy (T-170-06)"
  - "Exchange's expiry-path delete stays unconditional for BOTH classes; only the success-path delete is gated on !entry.reusable — reusable codes are not immortal, they still expire"
  - "Lookup + expiry-check + conditional-delete all stay under the single existing mutex hold — TOCTOU guarantee preserved for both code classes"

patterns-established:
  - "joinExchangeTestServer test helper in join_test.go mirrors testServer/capForSession conventions from server_test.go but adds SetJoinCodes wiring and returns the signing key (no getter exists on WebServer, by design)"

requirements-completed: [FNL-08]

coverage:
  - id: D1
    description: "IssueReusable mints a code that survives repeated Exchange calls until its per-code TTL elapses or Revoke is called"
    requirement: "FNL-08"
    verification:
      - kind: unit
        ref: "internal/capability/joincode_test.go#TestJoinCodeManager_IssueReusable_MultiExchange"
        status: pass
    human_judgment: false
  - id: D2
    description: "Revoke(code) makes a previously-valid reusable code return ErrCodeNotFound on the next Exchange"
    requirement: "FNL-08"
    verification:
      - kind: unit
        ref: "internal/capability/joincode_test.go#TestJoinCodeManager_Revoke"
        status: pass
      - kind: unit
        ref: "internal/capability/joincode_test.go#TestJoinCodeManager_Revoke_UnknownCodeIsNoOp"
        status: pass
    human_judgment: false
  - id: D3
    description: "A reusable code past its per-call TTL returns ErrCodeExpired (and is deleted), same as single-use codes"
    requirement: "FNL-08"
    verification:
      - kind: unit
        ref: "internal/capability/joincode_test.go#TestJoinCodeManager_ReusableExpiresAfterTTL"
        status: pass
    human_judgment: false
  - id: D4
    description: "The existing single-use Issue/Exchange contract is unchanged — all 6 pre-existing joincode tests pass unmodified"
    requirement: "FNL-08"
    verification:
      - kind: unit
        ref: "internal/capability/joincode_test.go (6 pre-existing tests: IssueFormat, ExchangeSucceedsOnce, ExchangeRejectsDoubleUse, ExchangeExpiresAfterTTL, ConcurrentExchangeIsAtomic, ExchangeRejectsUnknownCode)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The public /join/exchange HTTP handler resolves a reusable read code twice in a row (two 303 redirects to /sessions/{id}?cap=...), never /join?error="
    requirement: "FNL-08"
    verification:
      - kind: integration
        ref: "internal/webserver/join_test.go#TestJoinExchange_ReusableCodeSurvivesTwoExchanges"
        status: pass
      - kind: integration
        ref: "internal/webserver/join_test.go#TestJoinExchange_SingleUseCodeStillFailsOnSecondExchange"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-07-05
status: complete
---

# Phase 170 Plan 01: Reusable Join-Code Primitive Summary

**JoinCodeManager gains IssueReusable/Revoke + a reusable-conditional Exchange delete, proven at both the unit layer and the public /join/exchange HTTP boundary — closing the FNL-08 UAT dead-end where a public share code resolved only once.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-07-05T19:30:44-05:00 (first task commit)
- **Completed:** 2026-07-05T19:34:56-05:00 (last task commit)
- **Tasks:** 3
- **Files modified:** 3 (+ TESTING.md docs update)

## Accomplishments
- `internal/capability/joincode.go`: added `reusable bool` on `joinEntry`, `IssueReusable(token, ttl)` (same crypto/rand + base32 path as `Issue`, no new RNG surface), `Revoke(code)` (idempotent), and made `Exchange`'s success-path delete conditional on `!entry.reusable` while keeping the expiry-path delete unconditional and the whole lookup+delete sequence under one mutex hold.
- `internal/capability/joincode_test.go`: extended with 4 new tests (`IssueReusable_MultiExchange`, `Revoke`, `Revoke_UnknownCodeIsNoOp`, `ReusableExpiresAfterTTL` via `SetClockForTest`) — all 6 pre-existing single-use tests left untouched and still green.
- `internal/webserver/join_test.go` (new file — no dedicated join-exchange test file existed before): proves the reusable contract at the actual public HTTP boundary a recipient hits — POSTing the same reusable read-only code to `/join/exchange` twice yields two 303 redirects to `/sessions/{id}?cap=<rTok>`, plus a negative guard that single-use `Issue` codes still fail on their second exchange.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add reusable + per-code-TTL semantics to JoinCodeManager** - `70f2f8ad` (feat)
2. **Task 2: Unit tests for reusable + per-code-TTL behavior** - `fa35b928` (test)
3. **Task 3: Public /join/exchange reusable double-exchange test (new webserver test file)** - `1618255b` (test)

**Docs (deviation, standing convention):** `cc864779` (docs: register join_test.go in TESTING.md)

## Files Created/Modified
- `internal/capability/joincode.go` - `reusable` field, `IssueReusable`, `Revoke`, conditional-delete `Exchange`
- `internal/capability/joincode_test.go` - 4 new tests covering the reusable contract
- `internal/webserver/join_test.go` (new) - public `/join/exchange` reusable double-exchange + single-use negative-guard tests
- `TESTING.md` - Suite Manifest count (374→375 Go / 527→528 total), FNL-08 traceability rows, dated Note entry

## Decisions Made
- IssueReusable deliberately reuses Issue's exact 5-byte crypto/rand + `joinCodeEncoding` path rather than introducing any new RNG or alphabet — a locked constraint from the plan (T-170-06), preserving the ~40-bit entropy budget.
- Exchange's conditional delete is the entire single-use-vs-reusable distinction: expiry-path delete is unconditional for both classes (reusable codes are not immortal), only the success-path delete is gated on `!entry.reusable`.
- The lookup+expiry-check+delete sequence stays under the single existing `sync.Mutex` hold for both code classes — the TOCTOU guarantee documented in the original struct comment is preserved verbatim, just extended to describe two code classes instead of one.
- `internal/webserver/join_test.go`'s helper (`joinExchangeTestServer`) returns the signing key alongside the `*WebServer`, since `SetSigningKey` intentionally has no getter — tests must capture the key at generation time to sign tokens `capability.Verify` inside the handler will accept.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Registered new test file in TESTING.md per project standing convention**
- **Found during:** Post-Task-3 review of `/Users/ken/dev/agenthub/CLAUDE.md`'s Regression Test Convention (Standing Rule)
- **Issue:** The project's root `CLAUDE.md` mandates that every new test file be added to TESTING.md's Suite Manifest counts and Requirement→Test Traceability Map (Section 4), with `tests/check-traceability-paths.sh` passing before commit. This plan added `internal/webserver/join_test.go` (new file) but the plan itself did not call out the TESTING.md update.
- **Fix:** Updated Suite Manifest Go count 374→375 (Total 527→528), added two FNL-08 rows to the Requirement→Test Traceability Map (one for the extended `joincode_test.go`, one for the new `join_test.go`), and added a dated Note entry following the existing convention.
- **Files modified:** `TESTING.md`
- **Verification:** `bash tests/check-traceability-paths.sh` → `OK: all traceability paths exist`
- **Committed in:** `cc864779` (separate docs commit, not folded into a task commit, to keep the traceability update auditable independent of the code change)

---

**Total deviations:** 1 auto-fixed (1 missing critical — project convention compliance)
**Impact on plan:** No scope creep; this is a project-wide standing rule (CLAUDE.md) that applies to every phase adding test files, not new functionality.

## Issues Encountered
None - all three tasks executed cleanly on the first attempt; every verification command specified in the plan passed as written (`gofmt -l`, `go build`, `go test -run JoinCode -v`, `go test -run TestJoinExchange -v`, `go test -race -short`).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `IssueReusable`/`Revoke` and the reusable-Exchange HTTP behavior are now proven and available for 170-02 (daemon-side minting of the public-share join code, tied to Funnel enable/expiry) and frontend display work in later waves.
- T-170-01 (read-only-only invariant) is honored in this plan's own test fixtures (`signReadOnly` mints `Perms: "read"` exclusively) but full enforcement — refusing to ever mint a write-scoped reusable code from the real mint call site — is explicitly deferred to 170-02, per the plan's threat model.
- No blockers.

---
*Phase: 170-public-share-access-codes-reusable-share-lifetime-join-code-*
*Completed: 2026-07-05*

## Self-Check: PASSED

All created/modified files found on disk; all 5 commits (70f2f8ad, fa35b928, 1618255b, cc864779, 5f47f3e7) verified present in git log.
