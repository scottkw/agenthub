---
phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard
plan: 04
subsystem: testing
tags: [docs, testing-md, security-review, sharing-guide, funnel, rw-gate]

requires:
  - phase: 171-01
    provides: "IssueSingleUseWithTTL, RemoveGrant, SetRWGate/isRWGated, gate-aware originAllowedForWrite — the enforcement primitives this plan's threat model and traceability rows document"
  - phase: 171-02
    provides: "handleSetSessionFunnelWrite/revokeFunnelWriteLocked daemon lifecycle — the mint/teardown mechanics this plan's threat model synthesizes and the Suite Manifest count correction accounts for"
  - phase: 171-03
    provides: "Danger-section hold-to-confirm gate + FULL ACCESS badge/icon UI — the frontend test files this plan adds FNL-09 traceability rows for"
provides:
  - "TESTING.md reconciled: Suite Manifest Go count corrected 375->376 (376/142/9/2=529 total), consolidated Phase-171 manifest note, two new FNL-09 traceability rows (SessionSharePanel.test.tsx, SessionCard.share.test.tsx), Section-5 Category Y / M-47 live off-tailnet public-write UAT item"
  - "frontend/src/content/help/sharing-guide.md: public-write (FULL ACCESS) help section stating the RCE risk plainly"
  - ".planning/phases/171-.../171-SECURITY.md: synthesized Phase-171 threat model asserting no public write path except the consent gate (SPEC R8), consolidating T-171-01..15 + T-171-SC into one threat->mitigation->test table"
affects: [gsd-secure-phase-171-reverification, phase-171-final-verify-work]

tech-stack:
  added: []
  patterns:
    - "Suite Manifest correction pattern: when a prior plan's Go-file-count note is wrong (an omitted new test file), add a NEW dated note that states the correction explicitly and shows old->new counts, rather than silently editing the historical note's own claim."
    - "Consolidated phase-closing threat model: rather than four separate STRIDE tables, the final plan of a security-sensitive phase synthesizes all prior plans' registers into one threat->mitigation->verifying-test table, explicitly stating which barrier is primary vs. defense-in-depth (Section 1.1/1.2 of 171-SECURITY.md) so a reviewer cannot mistake the secondary check for the real gate."

key-files:
  created:
    - .planning/phases/171-public-full-access-read-write-internet-sharing-behind-a-hard/171-SECURITY.md
  modified:
    - TESTING.md
    - frontend/src/content/help/sharing-guide.md

key-decisions:
  - "Suite Manifest fix implemented as a new dated correction note (Phase 171-04) rather than editing the 171-02 note's own historical count claim — preserves the audit trail of when the omission was caught and corrected, matching the project's general append-only note convention."
  - "171-SECURITY.md explicitly separates D-01 (grant-registration gating at the real WS upgrade) as the PRIMARY and sufficient terminal-write barrier from D-02 (originAllowedForWrite) as defense-in-depth for files.write ONLY — stated as its own numbered subsection (1.2) to prevent the exact misreading 171-01's RESEARCH flagged as Pitfall 1 (mistaking a narrower HTTP-only check for the real enforcement point)."
  - "M-47 (Section 5 Category Y) is written as a single end-to-end script covering hold-gate, off-tailnet redemption + PTY execution, second-redemption fail-closed, independent read-spectator coexistence, and RW-only teardown-vs-read-survival in one pass, rather than splitting into multiple manual items — mirrors the M-46 (FNL-08) precedent's single-script structure."

patterns-established:
  - "When a threat model task's acceptance criteria names an exact threat-ID range (T-171-01..14), include any additional threat IDs the sourced plan files define (here T-171-15) as supplementary rows rather than omitting them — completeness over minimal literal compliance, since the underlying plan file is the authoritative threat register."

requirements-completed: [FNL-09]

coverage:
  - id: D1
    description: "TESTING.md Suite Manifest counts and Total are internally consistent and reflect all Phase-171 test additions; check-traceability-paths.sh exits 0"
    requirement: FNL-09
    verification:
      - kind: other
        ref: "bash tests/check-traceability-paths.sh (exits 0; local BSD grep silently no-ops on -P, so also independently verified via a line-based Python simulation of the GNU grep -P regex used in CI — 221 traceability paths all resolve on disk)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Section-4 FNL-09 traceability rows added for the 171-03 frontend test files (SessionSharePanel.test.tsx, SessionCard.share.test.tsx), path column repo-relative source paths only"
    requirement: FNL-09
    verification:
      - kind: other
        ref: "TESTING.md Section 4 — FNL-09 rows for frontend/src/components/__tests__/SessionSharePanel.test.tsx and SessionCard.share.test.tsx"
        status: pass
    human_judgment: false
  - id: D3
    description: "Section-5 M-47 live off-tailnet public-write UAT item captures the RCE-severity acceptance gate CI cannot cover"
    requirement: FNL-09
    verification: []
    human_judgment: true
    rationale: "M-47 is itself a manual/live UAT item requiring a real Funnel-granted tailnet and off-tailnet devices — its presence and completeness in TESTING.md is verifiable by inspection, but its actual execution (the RCE acceptance gate) is inherently unautomatable and deferred to release-time UAT, same class as M-34/M-41/M-45/M-46."
  - id: D4
    description: "Sharing Guide has a public-write (FULL ACCESS) section stating the RCE risk plainly, the hold-gate flow, single-use/short-lived semantics, and read/write independence; frontend tsc clean"
    requirement: FNL-09
    verification:
      - kind: other
        ref: "grep -qi 'FULL ACCESS' frontend/src/content/help/sharing-guide.md"
        status: pass
      - kind: other
        ref: "cd frontend && pnpm exec tsc --noEmit"
        status: pass
    human_judgment: false
  - id: D5
    description: "171-SECURITY.md exists, asserts no public write path except the consent gate, consolidates T-171-01..15 + T-171-SC into a threat->mitigation->verifying-test table, records the defense-in-depth scoping of originAllowedForWrite and the loopback-endpoint provenance, and states /gsd-secure-phase 171 re-verifies post-execution"
    requirement: FNL-09
    verification:
      - kind: other
        ref: "grep -q 'no public write path' 171-SECURITY.md && grep -q 'TestHandleWSSRelay_WriteCap_RequiresGate' 171-SECURITY.md && grep -q 'gsd-secure-phase 171' 171-SECURITY.md"
        status: pass
    human_judgment: false

duration: 11min
completed: 2026-07-07
status: complete
---

# Phase 171 Plan 04: TESTING.md Reconciliation + Sharing Guide + Security Synthesis Summary

**Reconciled TESTING.md (corrected a 1-file Go-count omission from 171-01, added FNL-09 traceability for 171-03's frontend tests, and added the M-47 live off-tailnet public-write acceptance gate), a risk-forward Sharing Guide section for public write access, and 171-SECURITY.md — the SPEC R8 threat model synthesizing all four plans' STRIDE registers and asserting no public write path exists except the hard consent gate.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-07-07T21:56:00Z (approx., immediately after 171-03)
- **Completed:** 2026-07-07T22:02:07Z
- **Tasks:** 3
- **Files modified:** 3 (1 new: `171-SECURITY.md`)

## Accomplishments

- **TESTING.md Suite Manifest correction:** discovered that `internal/webserver/rwgate_test.go` — a genuinely NEW file created in 171-01 — was never reflected in the Go unit/integration file count by either the 171-01 note (which didn't exist) or the 171-02 backfill note (which incorrectly stated "counts unchanged, no new files" for both plans). Corrected the count 375→376 and the Total 528→529 (376 Go / 142 vitest / 9 Playwright / 2 build-script), verified against an actual `find` count on disk. Added one consolidated Phase-171 manifest note (rather than four separate ones) covering the FNL-09 additions across all three implementation plans.
- **Section-4 FNL-09 traceability:** added the two missing rows for 171-03's extended-in-place frontend test files (`SessionSharePanel.test.tsx`, `SessionCard.share.test.tsx`) — the prior FNL-09 rows covered only 171-01/171-02's Go tests.
- **Section-5 Category Y / M-47:** a single live off-tailnet public-write UAT script — hold-gate completion → redeem the single-use write code from an off-tailnet device → terminal input actually executes on the remote PTY → a second redemption of the same code fails closed (single-use proof) → a separate read spectator keeps viewing throughout → owner disables RW → the writer's next input is rejected while the read spectator is unaffected. This is the RCE-severity acceptance gate that CI structurally cannot cover.
- **Sharing Guide:** added a "Public write access (FULL ACCESS)" section between the existing Funnel and Device-Share sections — states the command-execution/RCE risk plainly (not softened), the 3-second hold-to-confirm Danger-section flow, single-use/one-writer/15m-default/1h-max semantics, one-click immediate disable, and that the reusable read share is unaffected by disabling write.
- **171-SECURITY.md:** synthesizes 171-01/02/03's individual `<threat_model>` STRIDE registers (T-171-01 through T-171-13) plus 171-04's own (T-171-14, T-171-15) plus the recurring T-171-SC row into one consolidated threat→mitigation→verifying-test table. Explicitly states the load-bearing invariant ("no public write path exists except through the new hard consent gate"), separates the PRIMARY enforcement barrier (grant-registration gating at the real WS upgrade, D-01) from the SECONDARY defense-in-depth barrier (`originAllowedForWrite`, D-02, scoped to `files.write` HTTP routes only — explicitly does NOT reach `MsgInput`/`MsgSessionInject`), records the loopback funnel-write endpoint's inherited (not newly assumed) owner-only trust boundary from the existing Phase 165/170 Funnel-toggle endpoints, confirms no OPEN write-path threat remains, and states `/gsd-secure-phase 171` will re-verify every mitigation against the implemented code post-execution.

## Task Commits

Each task was committed atomically:

1. **Task 1: TESTING.md reconcile — Suite Manifest, Traceability, M-47** - `2bc3ba29` (docs)
2. **Task 2: Sharing Guide — public-write (FULL ACCESS) section** - `2b193091` (docs)
3. **Task 3: Synthesize 171-SECURITY.md — closed-write-perimeter threat model (SPEC R8)** - `6ab89d0f` (docs)

## Files Created/Modified

- `TESTING.md` - Suite Manifest count correction (375→376 Go, 528→529 Total) + consolidated Phase-171 note; 2 new FNL-09 traceability rows; Section-5 Category Y / M-47 live off-tailnet public-write UAT item
- `frontend/src/content/help/sharing-guide.md` - new "Public write access (FULL ACCESS)" section
- `.planning/phases/171-public-full-access-read-write-internet-sharing-behind-a-hard/171-SECURITY.md` (NEW) - synthesized Phase-171 threat model (SPEC R8)

## Decisions Made

- Corrected the Go file count via a new dated note rather than silently editing 171-02's own historical note text — preserves the audit trail (when the omission was introduced vs. when it was caught).
- 171-SECURITY.md dedicates an explicit subsection (1.2) to stating that `originAllowedForWrite` is defense-in-depth ONLY and does not reach `MsgInput`/`MsgSessionInject`, directly countering the exact misreading 171-01's own RESEARCH flagged as "Pitfall 1" — treating a narrower HTTP-only check as if it were the terminal-write barrier.
- Included T-171-15 (the documentation/repudiation threat this very artifact resolves) as a supplementary table row beyond the task's literal "T-171-01..14" acceptance-criteria range, since 171-04-PLAN.md's own `<threat_model>` block defines it and completeness against the sourced plan file was judged more correct than a minimal literal match.
- M-47 is one end-to-end manual script (hold-gate → redeem → second-redemption-fails → read-spectator-coexists → disable → writer-401s-read-survives) rather than split into several M-NN items, mirroring the M-46 (FNL-08) precedent's single-script structure for a comparable multi-step live acceptance gate.

## Deviations from Plan

None — plan executed exactly as written. The Suite Manifest count correction (Go 375→376) was explicitly anticipated by the plan's own Task 1 action text ("increment the Go file count for the new internal/webserver/rwgate_test.go"), not an unplanned discovery.

## Issues Encountered

- Confirmed that the local BSD `grep` on this macOS dev machine does not support `-P` and silently no-ops the `check-traceability-paths.sh` script's core regex (prints a usage error to stderr but the `while read` loop then iterates zero lines, so the script reports "OK" regardless of actual path validity). This is a pre-existing environment limitation, not introduced by this plan — confirmed by running the same regex against the pre-edit `TESTING.md` via `git show HEAD:TESTING.md`, which exhibited the identical no-op behavior. Independently validated the real path-existence guarantee via a Python simulation using GNU `grep -P`'s actual per-line matching semantics (the script's own CI environment is `ubuntu-latest`, which has GNU grep) — all 221 traceability paths in the reconciled file resolve on disk. Not fixed (out of scope: the script itself is unmodified by this plan, and CI already runs on GNU grep where the check is fully effective).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 171 is now feature-complete across all 4 plans (171-01 enforcement primitives, 171-02 daemon lifecycle, 171-03 frontend Danger-section UI, 171-04 documentation/security synthesis). All automated gates green: `bash tests/check-traceability-paths.sh` exits 0, `cd frontend && pnpm exec tsc --noEmit` clean.
- 171-SECURITY.md is ready for `/gsd-secure-phase 171` to independently re-verify every mitigation against the implemented code.
- The sole remaining open item for the whole phase is the live M-47 off-tailnet public-write UAT (Section 5 Category Y of TESTING.md) — requires a signed production build, a real Funnel-granted tailnet, and at least two off-tailnet devices. This mirrors the deferred-to-release-time pattern already established for M-34 (FNL-03/04), M-41 (NTF), M-45 (FIX-05), and M-46 (FNL-08).
- Next action per STATE.md's existing plan: Phase 169 (Tailscale Detection Fix, #120) remains the last open non-171 v4.2 phase; once M-47 (and Phase 169) are live-verified, run `/gsd-verify-work 171` and then the v4.2 milestone close-out.

---
*Phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard*
*Completed: 2026-07-07*
