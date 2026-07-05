---
phase: 169-tailscale-detection-fix
plan: 01
subsystem: infra
tags: [tailscale, go, permissions, macos, health-check]

# Dependency graph
requires:
  - phase: 169-tailscale-detection-fix (169-01, original execution — HALTED)
    provides: The invalidated CLI-status-fallback code this plan reverts (CR-01 proved it hits the identical per-user macsys permission gate)
provides:
  - "Honest, permission-aware Tailscale health detection: distinguishes macsys daemon-running-but-unreadable from genuinely-down without ever reporting a false Connected"
  - "permProbeFunc injectable seam wired through CheckHealth/CheckHealthWithCustomPath"
  - "Additive PermissionLimited field on TailscaleHealth (json: permissionLimited)"
affects: [169-02, frontend Settings/Tailscale health indicator consumers]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Injectable seam idiom (statusFunc/prefsFunc/permProbeFunc) for testing OS-permission-dependent code without a live daemon or real 0640 root:admin file"
    - "Fail-safe error classification: unknown errors from a permission probe fall through to the pre-existing safe state, never to a new positive claim"

key-files:
  created: []
  modified:
    - internal/webserver/tailscale.go
    - internal/webserver/tailscale_test.go
    - TESTING.md

key-decisions:
  - "Reverted the invalidated CLI-status-fallback (cliStatusFunc/runTailscaleStatusCLI/realCLIStatusFunc) by direct code edit rather than git revert, since a metadata commit sits on top of the original 169-01 commits"
  - "permProbeFunc only fires within the err != nil arm of checkHealth — the SDK-success path (Steps 3-4) is byte-for-byte unchanged, proven by TestCheckHealth_FullyHealthy staying green"
  - "PermissionLimited=true sets DaemonUp=true and Installed=true (the honest, liveness-confirmed choice) but Connected/HasCerts/IP/Domain/AcceptDNS all stay at zero values — no false Connected under any circumstance"
  - "realPermProbe only observes os.Open failing with fs.ErrPermission; it never reads the sameuserproof file's bytes (info-disclosure boundary, T-169-02)"
  - "macsysDaemonAlive parses the ipnport symlink target strictly as an integer port via strconv.Atoi, never as a filesystem path (T-169-01 tampering mitigation)"
  - "Any error other than fs.ErrPermission/fs.ErrNotExist from the probe is treated as unknown-then-false (fail-safe) and logged at slog.Debug (addresses REVIEW IN-01 visibility gap)"

patterns-established:
  - "Pattern: OS-permission-boundary probes should observe (os.Open + errors.Is(fs.ErrPermission)) rather than classify error strings from a higher-level API — the underlying EACCES is otherwise swallowed and indistinguishable from other failures"

requirements-completed: [FIX-05]

coverage:
  - id: D1
    description: "CLI-status-fallback code (cliStatusFunc/runTailscaleStatusCLI/realCLIStatusFunc, os/exec + encoding/json imports, and the 3 CLIFallback tests) fully removed; pre-169-01 checkHealth signature restored (D-01)"
    requirement: "FIX-05"
    verification:
      - kind: unit
        ref: "internal/webserver/tailscale_test.go — full TestCheckHealth suite (all pre-existing tests) after revert"
        status: pass
      - kind: other
        ref: "grep -cE 'os/exec|runTailscaleStatusCLI|cliStatusFunc' internal/webserver/tailscale.go == 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "Permission-limited detection distinct from daemon-down: PermissionLimited=true only when the injected perm probe confirms macsys EACCES + liveness; Connected stays false (D-02/D-03/D-04, hard SC)"
    requirement: "FIX-05"
    verification:
      - kind: unit
        ref: "internal/webserver/tailscale_test.go#TestCheckHealth_PermissionLimited"
        status: pass
      - kind: unit
        ref: "internal/webserver/tailscale_test.go#TestCheckHealth_DaemonDown_NotPermissionLimited"
        status: pass
    human_judgment: false
  - id: D3
    description: "SDK-success path (Steps 3-4 of checkHealth) byte-for-byte unchanged — TestCheckHealth_FullyHealthy stays green (D-06/SC2)"
    requirement: "FIX-05"
    verification:
      - kind: unit
        ref: "internal/webserver/tailscale_test.go#TestCheckHealth_FullyHealthy"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live non-admin macsys acceptance: a real non-admin macOS account with a real 0640 root:admin sameuserproof file reports permissionLimited:true (never a false Connected) in the running app"
    requirement: "FIX-05"
    verification: []
    human_judgment: true
    rationale: "Cannot be exercised on this admin dev box or in CI — requires a real macsys (App Store) Tailscale install and a genuine non-admin macOS user account to reproduce the sameuserproof permission failure (M-45 in TESTING.md, human_judgment: true)."

duration: 35min
completed: 2026-07-05
status: complete
---

# Phase 169 Plan 01: Honest Permission-Aware Tailscale Detection Summary

**Reverted the invalidated CLI-status-fallback and replaced it with a darwin-only, liveness-confirmed macsys permission probe that reports `PermissionLimited=true` instead of ever risking a false `Connected`**

## Performance

- **Duration:** 35 min
- **Started:** 2026-07-05T00:00:00Z (approx, re-execution session)
- **Completed:** 2026-07-05
- **Tasks:** 2 (Task 2 executed as TDD: RED + GREEN)
- **Files modified:** 3 (tailscale.go, tailscale_test.go, TESTING.md)

## Accomplishments
- Fully removed the invalidated CLI-`status`-fallback code (`cliStatusFunc`, `runTailscaleStatusCLI`, `realCLIStatusFunc`, the `os/exec`/`encoding/json` imports, and the three `TestCheckHealth_CLIFallback_*` tests) — CR-01 proved this approach ran as the same OS user and hit the identical per-user macsys permission gate, so it could report a FALSE Connected on the exact case it targeted.
- Restored `checkHealth` and its public wrappers to the pre-169-01 4-arg shape as an explicit prerequisite checkpoint (Task 1), verified fully green before any new code was added.
- Added the injectable `permProbeFunc` seam (mirroring `statusFunc`/`prefsFunc`), `realPermProbe()` (darwin-only, globs `/Library/Tailscale/sameuserproof-*`, `os.Open`s each match to observe `fs.ErrPermission` without ever reading file contents, then confirms liveness via `macsysDaemonAlive`), and `macsysDaemonAlive()` (reads the `ipnport` symlink, parses it strictly as an integer port, TCP-dials `127.0.0.1:<port>` with a 1s timeout).
- Added the additive `PermissionLimited bool` field (`json:"permissionLimited"`) on `TailscaleHealth`; wired `realPermProbe()` through both `CheckHealth` and `CheckHealthWithCustomPath`.
- Proved via TDD (RED then GREEN) that `PermissionLimited=true` never coexists with `Connected=true`, and that a genuinely-down daemon (no macsys signature) is never misclassified as permission-limited.
- Updated `TESTING.md` (suite manifest note, FIX-05 traceability row, M-45 manual checklist) per the repo's standing regression-suite convention, replacing the stale CLI-fallback-era text with the honest-detection description.

## Task Commits

Each task was committed atomically:

1. **Task 1: Revert the invalidated CLI-status fallback (D-01)** - `a0e87398` (revert)
2. **Task 2 RED: Add failing tests for permission-limited detection** - `c42d238b` (test)
3. **Task 2 GREEN: Implement honest permission-aware macsys detection** - `b244d254` (feat)
4. **TESTING.md convention update** - `f3e5b12b` (docs)

_Note: Task 2 was tdd="true" and produced two commits (test → feat); no refactor commit was needed since GREEN passed cleanly on first implementation._

## Files Created/Modified
- `internal/webserver/tailscale.go` - Removed CLI-fallback code; added `permProbeFunc`, `realPermProbe()`, `macsysDaemonAlive()`, `macsysSharedDir` const, `PermissionLimited` field, and rewired both public wrappers
- `internal/webserver/tailscale_test.go` - Removed 3 CLIFallback tests; updated all `checkHealth(...)` call sites to the final 5-arg signature; added `TestCheckHealth_PermissionLimited` and `TestCheckHealth_DaemonDown_NotPermissionLimited`
- `TESTING.md` - Updated suite manifest note, FIX-05 traceability row, and Category W M-45 manual checklist item to describe the honest-detection approach instead of the reverted CLI fallback

## Decisions Made
- Reverted by direct code edit (not `git revert`) since a docs/metadata commit sits on top of the original 169-01 commits.
- `PermissionLimited=true` sets `DaemonUp=true`/`Installed=true` (honest, liveness-confirmed) but leaves `Connected`/`HasCerts`/`IP`/`Domain`/`AcceptDNS` at zero values — no false Connected under any circumstance (hard SC, unit-asserted).
- The probe never reads `sameuserproof` file contents — only observes whether `os.Open` returns `fs.ErrPermission` (T-169-02 info-disclosure boundary).
- The `ipnport` symlink target is parsed strictly as an integer via `strconv.Atoi`; it is never used as a filesystem path (T-169-01 tampering mitigation).
- Any error other than `fs.ErrPermission`/`fs.ErrNotExist` is treated as unknown-then-false (fail-safe, T-169-04) and logged at `slog.Debug` for visibility (addresses REVIEW IN-01).

## Deviations from Plan

None - plan executed exactly as written, including the TDD RED/GREEN gate sequence for Task 2. The TESTING.md update was performed per the repo's standing convention (`/Users/ken/dev/agenthub/CLAUDE.md`), which is a hard project constraint rather than a plan deviation, and is documented here for traceability.

## TDD Gate Compliance

Task 2 (`tdd="true"`) followed the mandatory RED → GREEN sequence:
- **RED** (`c42d238b`, `test(169-01): ...`): new tests added and confirmed to fail — the file did not compile against the 4-arg `checkHealth` signature (`too many arguments in call to checkHealth`), which is the expected fail-fast signal for a seam-adding change (mirrors the precedent set by the earlier AcceptDNS/`prefsFunc` addition).
- **GREEN** (`b244d254`, `feat(169-01): ...`): production code added; full `TestCheckHealth` suite (11 tests including the 2 new ones) passes; `go build ./...`, `go vet ./internal/webserver/`, and the full `go test ./... -count=1` (all 14 packages) are green.
- No REFACTOR commit was needed — GREEN passed cleanly with gofmt-clean output on first implementation.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- FIX-05 (Issue #120) is now addressed with an honest, non-misleading detection mechanism. The consumer contract in `internal/daemon/process.go` (`h.Connected && h.HasCerts && h.IP != ""`) is unaffected — a permission-limited node has `Connected=false` and therefore never triggers a web-server restart, which is safe regardless of `DaemonUp`.
- **Remaining open item:** M-45 (TESTING.md Category W) — live non-admin macsys acceptance test — is `human_judgment: true` and cannot be exercised on this admin dev box or in CI. It requires a real macsys (App Store distribution) Tailscale install and a genuine non-admin macOS Standard user account to reproduce the `sameuserproof` 0640 root:admin permission failure. This is the only outstanding verification gap for FIX-05.
- No blockers for subsequent phase work; the frontend Settings/Tailscale health indicator can now consume the new `permissionLimited` field to render an accurate state distinct from "not connected" (out of scope for this plan — deferred to a follow-up if the frontend consumer needs updating, per 169-CONTEXT.md scope).

---
*Phase: 169-tailscale-detection-fix*
*Completed: 2026-07-05*

## Self-Check: PASSED

All claimed files exist on disk and all claimed commits exist in git history:
- FOUND: internal/webserver/tailscale.go
- FOUND: internal/webserver/tailscale_test.go
- FOUND: .planning/phases/169-tailscale-detection-fix/169-01-SUMMARY.md
- FOUND: a0e87398 (revert), c42d238b (test), b244d254 (feat), f3e5b12b (docs)
