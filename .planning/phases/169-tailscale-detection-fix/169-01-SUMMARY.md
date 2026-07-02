---
phase: 169-tailscale-detection-fix
plan: 01
subsystem: infra
tags: [go, tailscale, cli-fallback, exec.CommandContext, ipnstate, testing]

# Dependency graph
requires:
  - phase: 168-bug-fix-settings-polish
    provides: stable webserver package baseline (no direct code dependency, just sequencing)
provides:
  - CLI `tailscale status --json` fallback in checkHealth, engaged only on SDK read error
  - cliStatusFunc injectable test seam mirroring statusFunc/prefsFunc
  - Non-admin macOS accounts (macsys sameuserproof unreadable) now report Connected via CLI fallback
affects: [tailscale-detection, funnel-backend, native-notifications]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CLI-fallback injectable func-type (cliStatusFunc) mirroring the existing statusFunc/prefsFunc test-seam idiom"
    - "Additive fallback branch inside an existing err != nil arm — never touches the success path, preserving byte-for-byte SC2 compatibility"

key-files:
  created: []
  modified:
    - internal/webserver/tailscale.go
    - internal/webserver/tailscale_test.go
    - TESTING.md

key-decisions:
  - "cliStatusFunc reuses tailscale.com/ipn/ipnstate.Status (no new local struct) so the fallback's field-mapping logic is a drop-in mirror of the SDK-success path"
  - "Fallback fires on ANY SDK error, on ALL platforms — no error-string classification, no runtime.GOOS gate (D-03/D-04)"
  - "AcceptDNS stays at its false zero value in the fallback path — no CLI prefs read added (D-02)"
  - "CLI spawn bounded solely by the existing ctx deadline via exec.CommandContext — no new timeout knob (D-07)"

patterns-established:
  - "Injectable CLI-fallback seam (cliStatusFunc) for testability without a live daemon or real binary"

requirements-completed: [FIX-05]

coverage:
  - id: D1
    description: "checkHealth reconstructs Connected/IP/HasCerts/Domain from `tailscale status --json` when the SDK read fails (SC1, FIX-05)"
    requirement: "FIX-05"
    verification:
      - kind: unit
        ref: "internal/webserver/tailscale_test.go#TestCheckHealth_CLIFallback_Success"
        status: pass
    human_judgment: false
  - id: D2
    description: "CLI fallback is never invoked and behavior is byte-for-byte unchanged when the SDK read succeeds (SC2)"
    requirement: "FIX-05"
    verification:
      - kind: unit
        ref: "internal/webserver/tailscale_test.go#TestCheckHealth_CLIFallback_NotInvokedOnSDKSuccess"
        status: pass
    human_judgment: false
  - id: D3
    description: "When both the SDK and CLI fallback fail, checkHealth returns the pre-existing not-connected state (D-05)"
    requirement: "FIX-05"
    verification:
      - kind: unit
        ref: "internal/webserver/tailscale_test.go#TestCheckHealth_CLIFallback_AlsoFails"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live confirmation on a real non-admin macOS account (macsys sameuserproof unreadable) that the GUI now reports Connected instead of installed-but-not-connected"
    verification: []
    human_judgment: true
    rationale: "Requires a real macsys Tailscale install and a genuine non-admin macOS user account to reproduce the sameuserproof permission failure that triggers the SDK read error — cannot be reproduced in a headless Go test or CI runner. Tracked as M-45 (TESTING.md Category W)."

# Metrics
duration: 13min
completed: 2026-07-02
status: complete
---

# Phase 169 Plan 01: Tailscale Detection Fix Summary

**CLI `tailscale status --json` fallback in checkHealth engages only on SDK read error, closing Issue #120 (non-admin macOS accounts unable to see Connected status) without touching the SDK-success path.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-07-02T13:20:02-05:00 (plan-creation baseline)
- **Completed:** 2026-07-02T13:32:27-05:00
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added a new injectable `cliStatusFunc` type and `runTailscaleStatusCLI` real implementation that spawns `tailscale status --json` via `exec.CommandContext` with a fixed argument vector, unmarshaling into the already-imported `ipnstate.Status` type
- Extended `checkHealth`'s `err != nil` arm with an additive CLI-fallback branch that reconstructs the full `TailscaleHealth` struct (Connected, IP, HasCerts, Domain) from CLI output — fires on any SDK error, on all platforms, using the already-resolved binary path (honoring custom paths per D-06)
- Wired the real CLI func through both public entrypoints (`CheckHealth`, `CheckHealthWithCustomPath`)
- Added three unit tests covering SC1 (fallback success), SC2 (fallback not invoked on SDK success), and D-05 (fallback also fails → not-connected state preserved)
- Registered FIX-05 in TESTING.md: §2 suite-manifest extension note, §4 traceability row, and a new §5 Category W with manual item M-45 for the live non-admin macOS verification

## Task Commits

Each task was committed atomically:

1. **Task 1: Add CLI status fallback to checkHealth (tailscale.go)** - `608af809` (feat)
2. **Task 2: Fallback unit tests + TESTING.md registration** - `c93d0774` (test)

_Note: no plan-metadata commit issued separately here — see the final metadata commit below._

## Files Created/Modified
- `internal/webserver/tailscale.go` - new `cliStatusFunc` injectable type, `runTailscaleStatusCLI` real implementation, additive fallback branch inside `checkHealth`'s error arm, both public wrappers updated
- `internal/webserver/tailscale_test.go` - all existing `checkHealth(...)` call sites updated with a trailing `nil` cli arg; three new fallback tests added
- `TESTING.md` - FIX-05 traceability row (§4), suite-manifest extension note (§2), new Category W with M-45 manual item (§5)

## Decisions Made
- Reused `tailscale.com/ipn/ipnstate.Status` for CLI JSON unmarshal instead of inventing a local struct — keeps the fallback's field-mapping logic a direct mirror of the existing SDK-success Step 4 mapping, per the plan's explicit prohibition against introducing a new struct
- Named the real implementation `runTailscaleStatusCLI` (module-level func) wrapped by `realCLIStatusFunc()` (returns the `cliStatusFunc`-typed value) to mirror the existing `realPrefsFunc(lc)` factory pattern in the same file

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 169 was the last open v4.2 phase (FIX-05 / Issue #120). All code-level acceptance criteria pass: `go build`, `go test`, `go vet`, and `bash tests/check-traceability-paths.sh` all exit 0; the SDK-success path (Steps 3-4) is untouched, verified by both the unchanged code diff and the SC2 unit test.
- **Live verification (M-45) still required** before this phase can be considered fully closed: a real macsys Tailscale install on a non-admin macOS account. This is release-time UAT, tracked alongside the other deferred v4.2 manual items (Phase 166 M-37..M-40, Phase 167 M-41) in STATE.md's Deferred Items table.
- With all 5 v4.2 phases (165-169) now code-complete, the milestone is ready for `/gsd-verify-work 169` (live M-45 confirmation, when a non-admin macOS test account is available) followed by milestone-level closeout review.

---
*Phase: 169-tailscale-detection-fix*
*Completed: 2026-07-02*

## Self-Check: PASSED

- FOUND: `.planning/phases/169-tailscale-detection-fix/169-01-SUMMARY.md`
- FOUND: commit `608af809` (Task 1)
- FOUND: commit `c93d0774` (Task 2)
- FOUND: commit `ad14d116` (this SUMMARY)
