---
phase: 14-tailscale-health-check-infrastructure
plan: 01
subsystem: infra
tags: [tailscale, go, health-check, tdd, local-client, ipnstate]

# Dependency graph
requires: []
provides:
  - TailscaleHealth struct with Installed/Connected/HasCerts/IP/Domain fields
  - checkHealth internal function with injected statusFunc for testability
  - CheckHealth public function wrapping local.Client{}.StatusWithoutPeers
  - tailscale.com v1.96.3 promoted from indirect to direct dependency
affects:
  - 14-02 (uses TailscaleHealth and CheckHealth for app-level integration)
  - phase-15 (TLS binding depends on HasCerts/Domain from this health check)

# Tech tracking
tech-stack:
  added:
    - tailscale.com/client/local (v1.96.3, promoted to direct)
    - tailscale.com/ipn/ipnstate (v1.96.3, transitive via above)
  patterns:
    - Function injection pattern: checkHealth(ctx, statusFunc) for daemon-free unit testing
    - Public wrapper pattern: CheckHealth calls checkHealth with real local.Client{}
    - BackendState=="Running" as sole Connected signal (not partial states)
    - len(CertDomains)>0 as HasCerts signal (distinct from Connected)

key-files:
  created:
    - internal/webserver/tailscale.go
    - internal/webserver/tailscale_test.go
  modified:
    - go.mod (tailscale.com promoted from indirect to direct)
    - go.sum (updated by go mod tidy)

key-decisions:
  - "statusFunc type defined in tailscale.go (not test file) since both are package webserver — avoids duplicate type error"
  - "statusFunc type alias enables function injection without interface overhead"
  - "Connected = BackendState==\"Running\" only; all other states (Stopped, NeedsLogin, Starting) map to false"
  - "pre-existing TestHub_SlowClientDisconnected failure in internal/relay is out of scope — deferred"

patterns-established:
  - "TDD with function injection: write failing tests referencing internal function, then implement"
  - "statusFunc = func(ctx context.Context) (*ipnstate.Status, error) — canonical injection signature"

requirements-completed: [HEALTH-01, HEALTH-02, HEALTH-03]

# Metrics
duration: 15min
completed: 2026-03-20
---

# Phase 14 Plan 01: TailscaleHealth struct and CheckHealth function via statusFunc injection

**TailscaleHealth struct and CheckHealth() with injected statusFunc for daemon-free unit tests — 4/4 tests pass, tailscale.com v1.96.3 promoted to direct dependency**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-03-20T18:42:30Z
- **Completed:** 2026-03-20T18:57:00Z
- **Tasks:** 2 (TDD RED + TDD GREEN)
- **Files modified:** 4

## Accomplishments

- `TailscaleHealth` struct with 5 JSON-tagged fields covering all health dimensions
- `checkHealth` internal function accepting injected `statusFunc` — 4 tests exercise all 7 behavior cases
- `CheckHealth` public wrapper using zero-value `local.Client{}` — no configuration needed
- `go mod tidy` promotes `tailscale.com v1.96.3` from `// indirect` to direct dependency

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing tests for CheckHealth health states** - `5648add` (test)
2. **Task 2: Implement TailscaleHealth struct and CheckHealth to pass all tests** - `7185ac8` (feat)

_Note: TDD tasks have two commits — test (RED) then implementation (GREEN)_

## Files Created/Modified

- `internal/webserver/tailscale.go` - TailscaleHealth struct, statusFunc type, checkHealth and CheckHealth functions
- `internal/webserver/tailscale_test.go` - 4 test functions covering all health states via injected statusFunc
- `go.mod` - tailscale.com v1.96.3 promoted from indirect to direct
- `go.sum` - updated by go mod tidy

## Decisions Made

- `statusFunc` type defined in `tailscale.go` rather than the test file. Both files share `package webserver`, so defining it in both would cause a duplicate type error. The linter confirmed this by removing the definition from the test file after the initial RED commit.
- `Connected` uses strict equality `status.BackendState == "Running"`. All six non-Running states (NoState, NeedsLogin, NeedsMachineAuth, Stopped, Starting) correctly yield `Connected=false`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed duplicate statusFunc type from test file**
- **Found during:** Task 2 (implementation phase)
- **Issue:** Plan specified defining `statusFunc` in the test file (Task 1) and again in tailscale.go (Task 2). Since both files are in `package webserver` (not `_test`), duplicate type definition causes compile error.
- **Fix:** Removed `type statusFunc` from `tailscale_test.go`; defined once in `tailscale.go` where tests can access it via same-package visibility.
- **Files modified:** `internal/webserver/tailscale_test.go`
- **Verification:** `go vet ./internal/webserver/` passes; all 4 tests pass.
- **Committed in:** `7185ac8` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug fix for duplicate type)
**Impact on plan:** Necessary for correct compilation. No scope creep. Same-package test access unchanged.

## Issues Encountered

Pre-existing `TestHub_SlowClientDisconnected` failure in `internal/relay` — verified to be present before this plan's changes. Not caused by this work. Logged to deferred items.

## User Setup Required

None - no external service configuration required. All code runs without a live tailscaled daemon (tests use injected fake statusFunc).

## Self-Check: PASSED

- `internal/webserver/tailscale.go` — FOUND
- `internal/webserver/tailscale_test.go` — FOUND
- Task 1 commit `5648add` — FOUND
- Task 2 commit `7185ac8` — FOUND
- `grep -c "func TestCheckHealth"` = 4 — VERIFIED
- `go vet ./internal/webserver/` — PASS
- `go test ./internal/webserver/ -run TestCheckHealth -v` — 4/4 PASS

## Next Phase Readiness

- `TailscaleHealth` and `CheckHealth` are exported and ready for plan 02 (Wails method binding + background poller)
- No blockers for plan 02 execution

---
*Phase: 14-tailscale-health-check-infrastructure*
*Completed: 2026-03-20*
