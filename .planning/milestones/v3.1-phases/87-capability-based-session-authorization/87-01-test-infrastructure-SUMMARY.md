---
phase: 87-capability-based-session-authorization
plan: 01
subsystem: testing
tags: [capability, hmac, security, tdd, wave0, go-testing]

# Dependency graph
requires: []
provides:
  - RED test skeletons for the internal/capability package (Plan 02 targets)
  - RED test skeletons for webserver capability enforcement (Plan 03 targets)
  - Relocated self-signed TLS + WebSocket + pipe-read helpers in internal/webserver/
  - Fuzz harness for capability.Verify forgery resistance
  - Build-tag protocol (phase87_wave1, phase87_wave2) that lets Wave 0 land
    without breaking go test ./... on the main branch
affects:
  - 87-02-capability-core
  - 87-03-webserver-enforcement
  - 87-04-daemon-api
  - 87-05-frontend-ui
  - 87-06-landing-and-terminal

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Build-tag-gated RED skeletons: files compile with `go vet -tags <wave>` but are excluded from default `go test ./...` until the wave lands"
    - "Package-internal test helpers (package webserver) coexist with external package webserver_test file-set — no namespace collision"
    - "Fuzz entry points gated by build tag because f.Skip() is not supported"

key-files:
  created:
    - internal/capability/capability_test.go
    - internal/capability/keystore_test.go
    - internal/capability/joincode_test.go
    - internal/capability/capability_fuzz_test.go
    - internal/webserver/capability_test_helpers.go
    - internal/webserver/capability_test.go
  modified: []

key-decisions:
  - "Build-tag protocol uses phase87_wave1 for the capability package and phase87_wave2 for webserver so the two waves can merge independently without cross-package compile coupling"
  - "capability_test_helpers.go placed in package webserver (internal test helpers) rather than package webserver_test to avoid duplicating helper names already in server_test.go's external test package"
  - "FuzzVerify uses a build tag rather than f.Skip() because fuzz entry-point Skip is silently dropped by the fuzzer harness"
  - "Each RED skeleton references the exact production symbols Plan 02/03 will create (capability.Claims, capability.Sign, capability.Verify, capability.JoinCodeManager, etc.) — un-skipping is a one-line change per test, not a rewrite"

patterns-established:
  - "Wave-0 RED skeletons: every downstream plan has an already-authored test file its automated verify command can point at (Nyquist sampling rule)"
  - "Build-tag gated testing: production-symbol-referencing tests land before the production symbols exist without breaking go test ./... (merge-order independence)"

requirements-completed: [SEC-01, SEC-02, SEC-03, SEC-04, SEC-05]

# Metrics
duration: 8min
completed: 2026-04-20
---

# Phase 87 Plan 01: Test Infrastructure Summary

**Six build-tag-gated Go test files (21 unit tests + 1 fuzz harness + 9 integration tests) authored as RED skeletons so every downstream Phase 87 plan has an already-existing test-file target — Nyquist-sampling compliant Wave 0.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-20T16:16:05Z
- **Completed:** 2026-04-20T16:24:02Z
- **Tasks:** 2
- **Files modified:** 0
- **Files created:** 6

## Accomplishments

- Authored four test files under `internal/capability/` (gated by `//go:build phase87_wave1`) totaling 21 unit tests plus a `FuzzVerify` harness seeded with one known-good token. Coverage includes Sign/Verify round-trip, payload/signature tamper detection, wrong-key rejection, malformed-input classes, constant-time comparison sentinel, Claims<->context round-trip, FileKeyStore 0600 round-trip, missing-file-is-not-error, corrupt-length, GenerateKey length, LoadOrGenerate restart stability, JoinCodeManager format regex, single-use Exchange, double-use rejection, TTL expiry, 100-goroutine TOCTOU atomicity, and unknown-code rejection.
- Authored two test files under `internal/webserver/` (gated by `//go:build phase87_wave2`) totaling 9 SEC-01..SEC-05 integration test skeletons plus relocated helpers (`selfSignedTLSForTest`, `testServer`, `testServerWithHub`, `dialWebServerWS`, `readPipeWithTimeout`) from the security-review scaffold.
- Wired the build-tag protocol so `go test ./...` on main remains green (existing 22 webserver tests pass; new files excluded by build tag); `go vet -tags phase87_wave1 ./internal/capability/...` and `go vet -tags phase87_wave2 ./internal/webserver/...` both pass, confirming syntactic validity for the wave un-tagging Plan 02 and Plan 03 will perform.
- Satisfied every Nyquist sampling requirement from 87-VALIDATION.md: every downstream task's `<automated>` command points at a test file that already exists after this plan.

## Task Commits

Each task was committed atomically:

1. **Task 1: Create capability package RED test skeletons** — `a35e963` (test)
2. **Task 2: Relocate security-review test helpers and author webserver capability test skeletons** — `5ca1f3e` (test)

**Plan metadata commit:** _(appended in final step below)_

## Files Created/Modified

- `internal/capability/capability_test.go` — 9 tests: Sign/Verify round-trip, tamper (payload + sig), wrong key, malformed token/base64/claims, constant-time comparison sentinel, Claims context round-trip
- `internal/capability/keystore_test.go` — 6 tests: FileKeyStore Save/Load round-trip (0600 assertion), missing-file, corrupt-length, GenerateKey length, LoadOrGenerate first-run, LoadOrGenerate reload
- `internal/capability/joincode_test.go` — 6 tests: Issue format regex (`^[A-Z2-7]{4}-[A-Z2-7]{4}$`), Exchange once, double-use rejection (ErrCodeNotFound), TTL expiry (ErrCodeExpired), 100-goroutine atomic Exchange, unknown-code rejection
- `internal/capability/capability_fuzz_test.go` — `FuzzVerify` seeded with one known-good `Sign(Claims{SID:"s1",Perms:"read",V:1}, key)` token, deterministic 32-byte key, body asserts no panic on any mutation
- `internal/webserver/capability_test_helpers.go` — Relocated helpers from `security-review/internal_webserver_server_test.go:29-198` into `package webserver` (internal test helpers); no name collision with `server_test.go`'s external-package helpers
- `internal/webserver/capability_test.go` — 9 SEC integration skeletons: UnauthenticatedClientCannotEnumerateSessions (SEC-02), WrongSessionCapRejected (SEC-03), ReadOnlyParamCannotGrantWrite (SEC-04), ReadOnlyCapabilityBlocksMsgInput (SEC-05), ReconnectWithoutReadonlyStillBlocked (SEC-05 regression), MissingCapReturns401, InvalidSignatureReturns401, RevokedGrantReturns403, ValidCapReturnsSession

## Decisions Made

- **Build-tag protocol (`phase87_wave1`/`phase87_wave2`):** Chose file-level `//go:build` tags over individual `t.Skip()` because the test bodies reference symbols (`capability.Sign`, `capability.NewJoinCodeManager`, etc.) that do not exist yet. With bodies skipped but imports unresolved, the files would fail to compile. The build tag excludes the entire file from default compilation until the relevant wave implementation lands.
- **Separate tags per wave:** `phase87_wave1` for the capability package tests and `phase87_wave2` for the webserver tests lets Plan 02 and Plan 03 un-tag independently. If a single tag gated both, Plan 02 could not green the capability unit tests without forcing Plan 03's webserver tests to also compile (they depend on Plan 03 wiring).
- **Helpers in `package webserver` (not `package webserver_test`):** Avoids duplicating helper names already present in `server_test.go` (`selfSignedTLSForTest`, `testServer`). The two packages coexist in the same directory with no Go-level collision. Plan 03's `capability_test.go` lives in the same `package webserver` so it can call the relocated helpers as bare identifiers.
- **FuzzVerify gated by build tag rather than `f.Skip()`:** Fuzz entry-point `Skip` is silently dropped by `go test -fuzz`, masking regressions. The build tag is the only reliable way to exclude a fuzz file from default compilation.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- **Pre-existing FAIL in `security-review/` package** (unrelated to this plan): running `go test ./...` surfaces a "found packages relay and webserver in /security-review" setup error. This predates Plan 01 (confirmed via `git stash` + baseline run) — the directory contains two test-scaffold files with different package declarations and was never intended to be built. Out of scope per the executor deviation-rules scope boundary. Documented here rather than "fixed" because fixing it would modify `security-review/` which is gitignored reference material for the phase.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 02 (capability core) is unblocked: remove the `//go:build phase87_wave1` tag from the four `internal/capability/*_test.go` files and author `capability.go` + `keystore.go` + `joincode.go` to satisfy the references.
- Plan 03 (webserver enforcement) is unblocked: remove the `//go:build phase87_wave2` tag from the two `internal/webserver/capability_test*.go` files and wire the capability package into `WebServer` (signingKey/grants/joinCodes fields + `requireCapability` middleware).
- Every downstream SEC-XX task now has an automated verify target that points at an already-authored test file — the `VALIDATION.md` Nyquist compliance gate holds.
- No blockers. The plan's `<success_criteria>` all pass and the pre-existing `security-review/` package is a no-op for this phase.

## Self-Check: PASSED

All 6 created files verified present on disk. Both task commits (`a35e963`, `5ca1f3e`) found in git log. Plan `<success_criteria>` re-run: 4 capability test files, 2 webserver capability files, 6 total `phase87_wave` build tags, 0 production `.go` files under `internal/capability/`.

---
*Phase: 87-capability-based-session-authorization*
*Completed: 2026-04-20*
