---
phase: 87-capability-based-session-authorization
plan: 02
subsystem: capability
tags: [security, capability, crypto, hmac, keystore, joincode]

# Dependency graph
requires:
  - 87-01 test skeletons (phase87_wave1 build-tag gated)
provides:
  - HMAC-SHA256 capability.Sign and capability.Verify
  - Capability Claims type and WithClaims/ClaimsFromContext helpers
  - KeyStore interface + FileKeyStore with GenerateKey / LoadOrGenerate bootstrap
  - JoinCodeManager with TOCTOU-safe single-use Exchange and injected clock
  - Sentinel errors: ErrMalformedToken, ErrInvalidSignature, ErrMalformedClaims, ErrCodeNotFound, ErrCodeExpired
affects:
  - 87-03-webserver-enforcement (imports internal/capability)
  - 87-04-daemon-api (imports internal/capability)
  - 87-05-frontend-ui (consumes cap URLs and join codes via daemon IPC)
  - 87-06-web-pages-integration (consumes /join exchange → cap URL redirect)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Stdlib-only HMAC-SHA256 token: base64url(claimsJSON).base64url(sig), constant-time via hmac.Equal"
    - "Source-inspection constant-time regression guard (TestVerify_ConstantTimeComparison reads capability.go)"
    - "Unexported ctxKey struct{} for context value collision safety"
    - "Plain sync.Mutex (not RWMutex) for TOCTOU-safe read+delete Exchange"
    - "Injected now func() time.Time clock seam exposed to tests via package-local export_test.go helper"
    - "os.WriteFile mode 0600 mirroring internal/daemon/engine.go:132 settings persistence"
    - "Missing-file-is-not-error Load: returns (nil, nil) so LoadOrGenerate can bootstrap"

key-files:
  created:
    - internal/capability/capability.go
    - internal/capability/errors.go
    - internal/capability/context.go
    - internal/capability/keystore.go
    - internal/capability/joincode.go
    - internal/capability/export_test.go
  modified:
    - internal/capability/capability_test.go
    - internal/capability/capability_fuzz_test.go
    - internal/capability/keystore_test.go
    - internal/capability/joincode_test.go

key-decisions:
  - "Used an unexported ctxKey struct{} (not a package-level string) so no code outside the capability package can ever read or write claims under a colliding key"
  - "Placed SetClockForTest in a new export_test.go file (only compiled during go test) rather than adding a production-accessible setter, so the clock-injection seam is hermetically sealed to test builds"
  - "TestVerify_ConstantTimeComparison reads capability.go as a text file and asserts hmac.Equal is present and 'bytes.Equal' literal is absent — this forced a doc-comment rewrite (removed a mention of bytes.Equal in the prose) but correctly guards against a future regression where a maintainer replaces hmac.Equal with bytes.Equal 'for simplicity'"
  - "TestVerify_RejectsMalformedClaimsJSON builds a token by hand (HMAC-sign a non-JSON payload with the same key) so the signature passes and the test exercises the json.Unmarshal failure path specifically"
  - "Base32 NoPadding encoding of 5 random bytes yields exactly 8 characters with no conditional handling — clean XXXX-XXXX split without length checks"
  - "Exchange deletes expired entries as a side effect before returning ErrCodeExpired, so the next Exchange on the same code returns ErrCodeNotFound (verified by the TTL test)"

patterns-established:
  - "Test build tags (phase87_wave1) are removed as the first step of the un-tagging wave, after production code passes the compile gate — prevents accidentally breaking go test ./... during implementation"
  - "Source-grep regression guards in Go tests for invariants that cannot be tested behaviourally (e.g. constant-time comparison)"

requirements-completed: [SEC-01, SEC-02, SEC-03, SEC-04, SEC-05]

# Metrics
duration: 12min
completed: 2026-04-20
---

# Phase 87 Plan 02: Capability Core Summary

**Pure-Go stdlib-only capability token subsystem: HMAC-SHA256 Sign/Verify with constant-time comparison, 32-byte FileKeyStore with atomic 0600 persistence, and TOCTOU-safe single-use join-code manager with injectable clock — all 21 unit tests GREEN and FuzzVerify runs 30s with zero crashers.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-20T16:28:36Z
- **Completed:** 2026-04-20T16:40:12Z
- **Tasks:** 2
- **Files created:** 6 (5 production `.go` + 1 test helper)
- **Files modified:** 4 (Wave 0 test skeletons un-tagged and un-skipped)

## Accomplishments

- Implemented the `internal/capability` package from scratch with pure Go stdlib (`crypto/hmac`, `crypto/sha256`, `crypto/rand`, `encoding/base64`, `encoding/base32`, `encoding/json`, `sync`, `time`, `os`, `path/filepath`) — no new `go.mod` dependencies.
- `Sign` produces a two-segment base64url token (`b64Payload.b64Sig`) and `Verify` parses, constant-time-checks, and decodes it, returning typed sentinel errors (`ErrMalformedToken`, `ErrInvalidSignature`, `ErrMalformedClaims`) for the three failure modes.
- `FileKeyStore` persists a 32-byte signing key to `capability.key` with mode 0600 using the same `os.WriteFile` idiom as `internal/daemon/engine.go:132`. `Load` returns `(nil, nil)` on first run so `LoadOrGenerate` can bootstrap.
- `JoinCodeManager` issues `XXXX-XXXX` base32 codes (~40 bits of entropy) with a configurable TTL and atomically consumes them in `Exchange` under a plain `sync.Mutex` — never `RWMutex`, per RESEARCH Pitfall 4. An injected `now func() time.Time` seam (exposed to tests via `export_test.go`'s `SetClockForTest`) lets the TTL test advance simulated time without real sleeps.
- `WithClaims` / `ClaimsFromContext` use an unexported `ctxKey struct{}` to prevent cross-package collisions. Zero-value Claims and missing-context paths both return `(Claims{}, false)`.
- Un-tagged all four Wave-0 test skeletons (`phase87_wave1` build tag removed from `capability_test.go`, `capability_fuzz_test.go`, `keystore_test.go`, `joincode_test.go`), un-skipped all 21 tests, and filled in the two stubbed bodies: `TestVerify_RejectsMalformedClaimsJSON` (builds a signed non-JSON token by hand) and `TestVerify_ConstantTimeComparison` (source-grep assertion on `capability.go`).
- FuzzVerify runs 30s with 158k–761k execs per run (variable under concurrent fuzzer load), 45–47 new interesting inputs, and zero crashers across multiple invocations.

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement capability.Sign/Verify, Claims ctx helpers, and sentinel errors** — `dd1c15e` (feat)
2. **Task 2: Implement FileKeyStore, JoinCodeManager, and unblock keystore+joincode tests** — `6d2cf8f` (feat)

**Plan metadata commit:** _(appended in final step below)_

## Files Created/Modified

**Created:**
- `internal/capability/capability.go` — `Claims` struct (JSON field order SID/Perms/IAT/GrantID/V per D-03), `Sign`, `Verify`. Uses `base64.RawURLEncoding`, `hmac.Equal`, and wraps sentinel errors with `fmt.Errorf("%w: ...", ...)`.
- `internal/capability/errors.go` — Five sentinel errors with `capability: ...` prefix.
- `internal/capability/context.go` — `WithClaims` / `ClaimsFromContext` with unexported `ctxKey struct{}`.
- `internal/capability/keystore.go` — `KeyStore` interface, `FileKeyStore`, `GenerateKey`, `LoadOrGenerate`.
- `internal/capability/joincode.go` — `JoinCodeManager` with plain `sync.Mutex`, injected `now` clock, `Issue`/`Exchange`.
- `internal/capability/export_test.go` — `SetClockForTest` method on `JoinCodeManager` (only compiled during `go test`).

**Modified (un-tagged and un-skipped):**
- `internal/capability/capability_test.go` — 9 tests GREEN; `TestVerify_RejectsMalformedClaimsJSON` and `TestVerify_ConstantTimeComparison` now have real bodies.
- `internal/capability/capability_fuzz_test.go` — `FuzzVerify` harness active; seed token signs cleanly.
- `internal/capability/keystore_test.go` — 6 tests GREEN (round-trip with 0600 check, missing-file, corrupt-length, GenerateKey length, first-run generate+save, second-run reload).
- `internal/capability/joincode_test.go` — 6 tests GREEN (format regex, single-use, double-use rejection, TTL expiry via injected clock, 100-goroutine TOCTOU atomicity, unknown-code rejection).

## Decisions Made

- **`ctxKey struct{}` unexported:** Chose a struct type rather than a string constant so the value cannot be constructed from outside the package under a colliding key. This is the canonical Go idiom for context values that must be hermetic to one package.
- **`export_test.go` over a production setter:** `SetClockForTest` lives in `export_test.go` (only compiled during `go test`) rather than being an exported production method. Production code cannot import this helper by construction, so the clock remains sealed to real `time.Now` in shipped binaries.
- **Source-grep constant-time regression guard:** `TestVerify_ConstantTimeComparison` reads `capability.go` as a text file and asserts both (a) `hmac.Equal` appears, and (b) the literal `bytes.Equal` does NOT appear. This is a structural test, not a behavioural test — behavioural timing tests are flaky under Go's scheduler — and it correctly forced the production code to carry zero `bytes.Equal` text, including prose.
- **Handmade token for `TestVerify_RejectsMalformedClaimsJSON`:** Built a token manually by HMAC-signing the bytes `"not valid json {{{"` with the test key and base64url-concatenating. This exercises the specific code path where the HMAC check passes but `json.Unmarshal` fails, which no mutated-valid-token test could reliably target.
- **Plain `sync.Mutex` for JoinCodeManager:** A `sync.RWMutex` read lock would allow two goroutines to see the entry as valid before either deletes it — producing two successful exchanges from one code. The 100-goroutine `TestJoinCodeManager_ConcurrentExchangeIsAtomic` would catch this; using `sync.Mutex` is the design that passes.
- **Delete expired entries in `Exchange`:** When the TTL has elapsed, `Exchange` deletes the entry before returning `ErrCodeExpired` so the subsequent `Exchange` sees `ErrCodeNotFound`. This avoids a background sweeper; the map stays small because codes are short-lived.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Test Infrastructure] Added `export_test.go` for clock injection**
- **Found during:** Task 2 (TTL expiry test implementation).
- **Issue:** The plan specified "For the TTL expiry test, build manager and set m.now to a closure that returns start.Add(6*time.Minute)." However, the Wave-0 test files are in the external test package `capability_test` (not `package capability`), so they cannot directly set the unexported `now` field. Without an injection seam the TTL test would have to either sleep 5 minutes (unacceptable) or require moving tests to the internal package (broader change).
- **Fix:** Added `internal/capability/export_test.go` containing `SetClockForTest(now func() time.Time)` on `*JoinCodeManager`. The file name suffix `_test.go` means the method only exists during `go test` compilation, preserving the hermetic boundary of the production API.
- **Files modified:** `internal/capability/export_test.go` (new).
- **Commit:** `6d2cf8f`.

**2. [Rule 1 - Bug Fix] Removed `bytes.Equal` mention from capability.go doc comment**
- **Found during:** Task 1 (first test run).
- **Issue:** `TestVerify_ConstantTimeComparison` uses `strings.Contains(src, "bytes.Equal")` which matched the literal `bytes.Equal` token in my prose doc-comment line "All comparisons that touch the signature bytes use hmac.Equal — never bytes.Equal or `==` — to avoid...". The test correctly flagged this — the guard is intended to be absolute.
- **Fix:** Rewrote the doc-comment paragraph to describe the invariant without naming `bytes.Equal`: "Signature comparison uses hmac.Equal for constant-time behaviour — never a non-constant-time byte comparison — to avoid leaking key material through timing side channels."
- **Files modified:** `internal/capability/capability.go`.
- **Commit:** Amended into Task 1 commit (`dd1c15e`) — the commit itself was made after this fix was applied, so only a single Task 1 commit exists.

## Issues Encountered

- **Pre-existing `security-review/` package setup failure** (unchanged from Plan 01): `go test ./...` surfaces "found packages relay and webserver in /security-review" at repo scan. This is documented in `87-01-SUMMARY.md` as out-of-scope reference material. No action taken.

## User Setup Required

None — no external service configuration, secrets, or manual steps required. The `capability.key` file is not created at this stage; it will be generated at daemon startup by Plan 04 when `LoadOrGenerate` is first called in production.

## Next Phase Readiness

- **Plan 03 (webserver enforcement) is unblocked:** `capability.Sign`, `capability.Verify`, `capability.Claims`, `capability.WithClaims`, `capability.ClaimsFromContext`, `capability.JoinCodeManager`, and all sentinel errors are now importable from `github.com/scottkw/agenthub/internal/capability`.
- **Plan 04 (daemon API) is unblocked:** `capability.GenerateKey`, `capability.LoadOrGenerate`, `capability.NewFileKeyStore`, `capability.NewJoinCodeManager` are ready to wire into daemon startup.
- **Plan gate green:** All 21 capability unit tests PASS; FuzzVerify runs 30s with zero crashers; `go test ./... -count=1` passes for every package except the pre-existing security-review scaffold.
- **No blockers.** The plan's `<success_criteria>` are all satisfied:
  - 21/21 unit tests PASS ✓
  - FuzzVerify 30s with zero panics ✓
  - No `math/rand`, `bytes.Equal` on sig, `sync.RWMutex` in JoinCodeManager, `exp` claim, or `alg` field anywhere in the package ✓
  - No existing test regresses (security-review was already failing before this plan) ✓
  - `internal/capability/capability.key` does not exist in the repo ✓

## Self-Check: PASSED

Verification run:

```
$ go test ./internal/capability/ -count=1 -v 2>&1 | grep -E "^(---|PASS|FAIL|ok)" | tail -25
--- PASS: TestSign_RoundTrip (0.00s)
--- PASS: TestVerify_RejectsTamperedPayload (0.00s)
--- PASS: TestVerify_RejectsTamperedSignature (0.00s)
--- PASS: TestVerify_RejectsWrongKey (0.00s)
--- PASS: TestVerify_RejectsMalformedTokenSegmentCount (0.00s)
--- PASS: TestVerify_RejectsMalformedBase64 (0.00s)
--- PASS: TestVerify_RejectsMalformedClaimsJSON (0.00s)
--- PASS: TestVerify_ConstantTimeComparison (0.00s)
--- PASS: TestClaims_Context_RoundTrip (0.00s)
--- PASS: TestJoinCodeManager_IssueFormat (0.00s)
--- PASS: TestJoinCodeManager_ExchangeSucceedsOnce (0.00s)
--- PASS: TestJoinCodeManager_ExchangeRejectsDoubleUse (0.00s)
--- PASS: TestJoinCodeManager_ExchangeExpiresAfterTTL (0.00s)
--- PASS: TestJoinCodeManager_ConcurrentExchangeIsAtomic (0.00s)
--- PASS: TestJoinCodeManager_ExchangeRejectsUnknownCode (0.00s)
--- PASS: TestFileKeyStore_RoundTrip (0.00s)
--- PASS: TestFileKeyStore_MissingFileReturnsNilNil (0.00s)
--- PASS: TestFileKeyStore_CorruptLengthReturnsError (0.00s)
--- PASS: TestGenerateKey_Length32 (0.00s)
--- PASS: TestLoadOrGenerate_GeneratesOnFirstRun (0.00s)
--- PASS: TestLoadOrGenerate_ReloadsOnSecondRun (0.00s)
--- PASS: FuzzVerify/seed#0 (0.00s)
PASS
ok  	github.com/scottkw/agenthub/internal/capability	0.011s
```

All 6 created files verified present on disk:
- `internal/capability/capability.go` FOUND
- `internal/capability/errors.go` FOUND
- `internal/capability/context.go` FOUND
- `internal/capability/keystore.go` FOUND
- `internal/capability/joincode.go` FOUND
- `internal/capability/export_test.go` FOUND

Both task commits found in git log:
- `dd1c15e` — feat(87-02): implement capability.Sign/Verify, Claims ctx helpers, and sentinel errors
- `6d2cf8f` — feat(87-02): implement FileKeyStore, JoinCodeManager, and unblock keystore+joincode tests

Acceptance criteria re-verified:
- `grep -q "hmac.Equal" internal/capability/capability.go` — PASS
- `grep -q "base64.RawURLEncoding" internal/capability/capability.go` — PASS
- `grep -q "bytes.Equal" internal/capability/capability.go` — (empty, no matches) — PASS
- `grep -q "sync.Mutex" internal/capability/joincode.go` — PASS
- `grep -q "sync.RWMutex" internal/capability/joincode.go` — (empty) — PASS
- `grep -q "os.WriteFile" internal/capability/keystore.go` — PASS
- `grep -q "0600" internal/capability/keystore.go` — PASS
- `grep -rq '"math/rand"' internal/capability/` — (empty) — PASS
- Zero `t.Skip` calls in any capability test file — PASS
- 5 production Go files under `internal/capability/` (capability.go, context.go, errors.go, joincode.go, keystore.go) — PASS

---
*Phase: 87-capability-based-session-authorization*
*Completed: 2026-04-20*
