---
phase: 176-platform-hardening-bug-fixes
plan: 04
subsystem: docs
tags: [testing, docs, regression-suite, traceability, v4.2]

# Dependency graph
requires:
  - phase: 176-01
    provides: BUG-05 (#124) main.go darwin-guard + DMABUF env guard fix
  - phase: 176-02
    provides: BUG-06 (#123) /app/ CSP header fix + TestCSPHeaderStrict_App
  - phase: 176-03
    provides: BUG-07 (#127) live-repro evidence, DOES-NOT-REPRODUCE (BRANCH B, no code change)
provides:
  - "TESTING.md Section 5 Category AA (M-52 Linux/Wayland GUI, M-53 /app/ CSP console sweep)"
  - "TESTING.md Section 4 BUG-06 traceability row (internal/webserver/csp_integration_test.go)"
  - "TESTING.md Section 2 Suite Manifest dated Phase 176 reconciliation note (counts unchanged)"
affects: []

tech-stack:
  added: []
  patterns:
    - "Suite Manifest correction note pattern reused verbatim (171-04/173-07/175-07 precedent): newest note prepended above older notes, never edited in place"

key-files:
  created: []
  modified:
    - TESTING.md

key-decisions:
  - "No BUG-07 Section 4 traceability row added — 176-03 took BRANCH B (DOES-NOT-REPRODUCE, zero code/test change), so per the plan's IFF clause the closure justification lives in 176-03-SUMMARY.md's evidence only, not a fabricated test row"
  - "Suite Manifest note states net +0 new test files (not a bump) — BUG-05 added no test, BUG-06 extended an existing file in place, BUG-07 touched nothing"

patterns-established: []

requirements-completed: [BUG-05, BUG-06, BUG-07]

coverage:
  - id: D1
    description: "TESTING.md carries M-52 (BUG-05 Linux/Wayland GUI launch + menu bar + no-freeze) and M-53 (BUG-06 production-build /app/ CSP console sweep) in a new Section 5 Category AA, following the established M-NN format (numbered steps + why-not-automatable + Source lines)"
    requirement: "BUG-05, BUG-06"
    verification:
      - kind: unit
        ref: "grep -q 'M-52' TESTING.md && grep -q 'M-53' TESTING.md"
        status: pass
    human_judgment: false
  - id: D2
    description: "Section 4 has a BUG-06 traceability row pointing to internal/webserver/csp_integration_test.go with TestCSPHeaderStrict_App in the Notes column; no BUG-07 row since 176-03 took BRANCH B"
    requirement: "BUG-06, BUG-07"
    verification:
      - kind: unit
        ref: "grep -q 'TestCSPHeaderStrict_App' TESTING.md && grep -q 'csp_integration_test.go | Go' TESTING.md"
        status: pass
    human_judgment: false
  - id: D3
    description: "Suite Manifest carries a dated Phase 176 note confirming file counts are unchanged (139 Go / 148 vitest / 9 Playwright / 2 build-script, total 298); check-traceability-paths.sh exits 0"
    requirement: "BUG-05, BUG-06, BUG-07"
    verification:
      - kind: unit
        ref: "bash tests/check-traceability-paths.sh (exit 0, false-OK on macOS BSD grep per 173-07 precedent — manually validated via Python regex sweep)"
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-07-09
status: complete
---

# Phase 176 Plan 04: TESTING.md Reconciliation Summary

**Reconciled TESTING.md for Phase 176's three bug fixes — added manual items M-52 (BUG-05 Linux/Wayland GUI) and M-53 (BUG-06 /app/ CSP console sweep), a Section 4 traceability row for BUG-06, and a dated Suite Manifest note confirming zero net-new test files, as the single serialized editor of TESTING.md for the phase.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-09T13:34:43Z
- **Completed:** 2026-07-09T13:43:10Z
- **Tasks:** 2 completed
- **Files modified:** 1 (TESTING.md)

## Accomplishments

- Added Section 4 traceability row for BUG-06 (`internal/webserver/csp_integration_test.go`, `TestCSPHeaderStrict_App`), and a note explaining why BUG-07 intentionally has NO traceability row (176-03's BRANCH B/DOES-NOT-REPRODUCE verdict left zero code or test diff — the closure evidence lives entirely in 176-03-SUMMARY.md's screenshot + computed-style readout).
- Added a new Section 5 "Category AA — Platform & Hardening Bug Fixes (BUG-05..07)" with:
  - **M-52** (BUG-05, #124): live Linux/Wayland GUI launch check (no segfault, working menu bar, no DMABUF freeze), marked OPPORTUNISTIC per D-11 (reporter's own from-source verification already accepted as sufficient to ship).
  - **M-53** (BUG-06, #123, D-06): production-build `/app/` CSP console sweep, marked OPPORTUNISTIC/release-time (requires `wails build -tags "webkit2_41,wailsassets"` since `/app/` 503s under `wails dev`/plain `go build`).
- Reconciled the Section 2 Suite Manifest with a dated Phase 176 note confirming file counts are **unchanged** (139 Go / 148 vitest / 9 Playwright / 2 build-script, total 298) — verified directly via `find internal -name '*_test.go' | wc -l` (124) + repo-root `*_test.go` (15) = 139, and `find frontend/src -name '*.test.ts*' | wc -l` = 148, both matching the pre-phase counts exactly.
- Ran `bash tests/check-traceability-paths.sh` — exits 0, but per the established 173-07 precedent this is a **false-OK on macOS**: BSD grep does not support `-P`, so the script's `grep -oP` errors silently and the while-loop never actually iterates any paths (confirmed by reproducing the `grep: invalid option -- P` stderr directly). Manually validated path correctness with an equivalent Python regex sweep (`re.findall(r'\| ([^\|]+\.(?:go|ts|tsx|sh)) \|', ...)` + `os.path.exists`) — the new BUG-06 row's path resolves, and the only "missing" match from the sweep is a pre-existing false positive in Category N's manual install-script code block (unrelated `.sh` substring inside a multi-line shell snippet, not a real table row — pre-dates this plan, out of scope per the scope-boundary rule).

## Task Commits

Each task was committed atomically:

1. **Task 1: Add manual M-NN items (BUG-05 Linux GUI, BUG-06 /app/ CSP sweep) and Section 4 traceability rows** - `3680ab98` (docs)
2. **Task 2: Reconcile the Suite Manifest counts and run check-traceability-paths.sh** - `450b0fa3` (docs)

_No TDD tasks in this plan — documentation-only plan per the plan's own `<threat_model>` (no code, no runtime surface, no input parsing)._

## Files Created/Modified

- `TESTING.md` — Section 4: new BUG-06 traceability row + BUG-07 no-row explanatory note. Section 5: new "Category AA — Platform & Hardening Bug Fixes (BUG-05..07)" with M-52 and M-53. Section 2: new dated Phase 176 Suite Manifest reconciliation note (prepended above the existing Phase 175-07 note, per the established newest-on-top convention).

## Decisions Made

- No BUG-07 Section 4 row — the plan's own must_haves make this conditional ("IFF plan 176-03 landed a fix"); 176-03 explicitly took BRANCH B (DOES-NOT-REPRODUCE, zero code change), so adding a fabricated row would misrepresent the codebase. The closure justification for GitHub #127 is recorded as prose in Section 4 pointing to 176-03-SUMMARY.md's live evidence instead.
- Suite Manifest note states counts are unchanged (net +0) rather than inventing any bump, per the plan's explicit "Do NOT invent a file-count bump" instruction — verified independently with the same scoped `find` commands used by the 175-07 precedent.
- Followed the established correction-note pattern exactly (171-04/173-07/175-07 precedent): added a NEW dated note at the top of the note stack rather than editing any prior phase's note.

## Deviations from Plan

None - plan executed exactly as written. Both tasks matched their `<done>` criteria.

## Issues Encountered

- `bash tests/check-traceability-paths.sh` is unreliable on this macOS dev box (BSD `grep` lacks `-P`; the script prints `grep: invalid option -- P` to stderr then silently reports "OK: all traceability paths exist" with exit 0 having checked nothing) — this is the exact limitation already documented in Phase 173-07's Suite Manifest note and in [[reference: 173-07 precedent]]. Not a new bug; resolved by manually validating path correctness with an equivalent Python regex sweep as the plan's action explicitly anticipates ("if the macOS BSD-grep unreliability noted in Phase 173-07 applies, validate the new rows' paths manually"). The one false-positive "missing" path the sweep surfaced (a `.sh` substring inside Category N's manual install-script code block) pre-dates this plan and is out of scope per the deviation rules' scope boundary (only fix issues directly caused by the current task's changes).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- TESTING.md now truthfully reflects all three Phase 176 bug fixes: BUG-05 (#124, manual M-52), BUG-06 (#123, automated traceability row + manual M-53 opportunistic sweep), and BUG-07 (#127, closure-by-evidence with no fabricated test row).
- Phase 176's four plans (176-01 through 176-04) are all complete. Per 176-01/176-02/176-03's shared convention, GitHub issues #124/#123/#127 remain OPEN — closure is deferred to phase-level verification/ship (e.g. `/gsd-verify-work 176`) so all three are closed together with the accumulated evidence, consistent with prior-phase precedent (170, 171).
- No blockers.

---
*Phase: 176-platform-hardening-bug-fixes*
*Completed: 2026-07-09*

## Self-Check: PASSED

- FOUND: TESTING.md (M-52, M-53, BUG-06 row all present via grep)
- FOUND: commit 3680ab98 (Task 1)
- FOUND: commit 450b0fa3 (Task 2)
- `bash tests/check-traceability-paths.sh` exits 0 (with documented macOS BSD-grep caveat, manually cross-checked)
