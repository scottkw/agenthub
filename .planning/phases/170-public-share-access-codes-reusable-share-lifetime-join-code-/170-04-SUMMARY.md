---
phase: 170-public-share-access-codes-reusable-share-lifetime-join-code-
plan: 04
subsystem: testing
tags: [docs, testing, traceability, manual-uat]

# Dependency graph
requires:
  - phase: 170-01
    provides: "internal/webserver/join_test.go + internal/capability/joincode_test.go FNL-08 coverage"
  - phase: 170-02
    provides: "internal/daemon/api_test.go + internal/daemon/funnel_test.go FNL-08 coverage"
  - phase: 170-03
    provides: "frontend/src/components/__tests__/SessionSharePanel.test.tsx FNL-08 coverage"
provides:
  - "TESTING.md Section 5 Category X — Public Share Access Codes (FNL-08), item M-46 (live off-tailnet reusable-code join + share-lifetime teardown)"
  - "Verified TESTING.md Suite Manifest counts (375 Go / 142 vitest / 9 Playwright / 2 build-script / 528 total) match actual on-disk file counts"
  - "Verified all 5 FNL-08 traceability rows in Section 4 are present and path-valid"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - TESTING.md

key-decisions:
  - "Task 1 (Suite Manifest note + FNL-08 traceability rows) required NO changes — 170-01/02/03's incremental docs commits already reconciled counts, phase notes, and traceability rows correctly; verified by direct file count (`find` for each suite group) and `bash tests/check-traceability-paths.sh`, not just by reading the prose"
  - "Section 4 carries 5 FNL-08 rows, not the 4 the plan's must_haves literally enumerate — joincode_test.go, join_test.go, api_test.go, funnel_test.go, SessionSharePanel.test.tsx — because 170-02's own deviation already added a dedicated api_test.go row alongside funnel_test.go; this is a superset of the plan's required 4, not a gap, so it was left as-is per the plan's own prohibition against inventing or renaming existing rows"
  - "New Category X placed after Category W (the last-lettered category in file order) with M-46 as the next sequential manual-item number, per the plan's explicit instruction"

patterns-established: []

requirements-completed: [FNL-08]

coverage:
  - id: D1
    description: "TESTING.md Suite Manifest (Section 2) counts and per-plan phase notes for 170-01/02/03 are internally consistent with the actual on-disk test-file counts"
    requirement: "FNL-08"
    verification:
      - kind: other
        ref: "find . -name '*_test.go' (excluding node_modules/agenthub-v4.0-redesign) → 375; find frontend/src -name '*.test.ts(x)' → 142; find frontend/e2e -name '*.spec.ts' → 9; find tests -maxdepth 1 -name '*.test.sh' → 2 (528 total, matches TESTING.md Section 2 exactly)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Section 4 has an FNL-08 traceability row for every new/extended test file covering FNL-08, with file-path-only path columns"
    requirement: "FNL-08"
    verification:
      - kind: other
        ref: "grep -n FNL-08 TESTING.md — 5 rows present: internal/capability/joincode_test.go, internal/webserver/join_test.go, internal/daemon/api_test.go, internal/daemon/funnel_test.go, frontend/src/components/__tests__/SessionSharePanel.test.tsx"
        status: pass
    human_judgment: false
  - id: D3
    description: "bash tests/check-traceability-paths.sh exits 0 (every mapped path exists on disk, path column format valid)"
    requirement: "FNL-08"
    verification:
      - kind: other
        ref: "bash tests/check-traceability-paths.sh → 'OK: all traceability paths exist'"
        status: pass
    human_judgment: false
  - id: D4
    description: "A new manual checklist item (M-46) captures the live off-tailnet end-to-end check: recipient types the public URL, enters the reusable code, joins read-only, and a second device reuses the same code; plus share-lifetime teardown after disable"
    requirement: "FNL-08"
    verification:
      - kind: other
        ref: "TESTING.md Section 5, new '### Category X — Public Share Access Codes (FNL-08)' with M-46 item, _Why not automatable:_ and _Source:_ lines present, mirroring the M-34/M-45 format"
        status: pass
    human_judgment: true
    rationale: "M-46 itself is an unautomatable live-UAT checklist item by design (requires a real Funnel-granted tailnet + off-tailnet devices) — its actual execution is deferred to release-time UAT, consistent with the already-deferred M-37..M-45 items in STATE.md's Deferred Items table. This coverage entry only proves the checklist item was correctly authored and registered, not that the live check has been run."

# Metrics
duration: 8min
completed: 2026-07-05
status: complete
---

# Phase 170 Plan 04: TESTING.md Reconciliation Summary

**Final reconciliation pass over TESTING.md for the whole Phase 170 test surface — verified the Suite Manifest counts, phase notes, and all 5 FNL-08 traceability rows left by 170-01/02/03's incremental self-registration were already internally consistent (no drift found), and added the one missing piece: Section 5 Category X / M-46, the live off-tailnet reusable-code-join + share-lifetime-teardown manual checklist item.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-07-06T01:15:00Z (approx, first read)
- **Completed:** 2026-07-06T01:23:00Z (commit 805dbf87)
- **Tasks:** 2 (Task 1 required no file changes — verified only)
- **Files modified:** 1 (TESTING.md)

## Accomplishments
- Verified Task 1's entire scope (Section 2 Suite Manifest counts/phase notes, Section 4 FNL-08 traceability rows) was already correctly reconciled by 170-01/02/03's own incremental docs commits — confirmed by direct on-disk file counts (`find` per suite group: 375 Go / 142 vitest / 9 Playwright / 2 build-script = 528, matching TESTING.md exactly) rather than trusting the prose, and by running `bash tests/check-traceability-paths.sh` (passed both before and after this plan's own edit).
- Added `### Category X — Public Share Access Codes (FNL-08)` with item **M-46** to TESTING.md Section 5: the live off-tailnet reusable-code join + share-lifetime-teardown manual check that cannot be automated — production build, real Funnel-granted tailnet, off-tailnet code entry to read-only join, a second off-tailnet device reusing the same code (proving reusability), and re-entry failing after the owner disables the share (proving lifetime teardown).

## Task Commits

1. **Task 1: Suite Manifest note + FNL-08 traceability rows** - no commit (verification-only; 170-01/02/03's prior docs commits `cc864779`/`835bbaa4`/`2ec5c1c5` already satisfied every acceptance criterion — confirmed via `tests/check-traceability-paths.sh` and direct file-count verification, no drift found).
2. **Task 2: Manual checklist item for the live off-tailnet reusable-code join** - `805dbf87` (docs)

## Files Created/Modified
- `TESTING.md` - added Section 5 `Category X — Public Share Access Codes (FNL-08)` with item M-46

## Decisions Made
- Task 1 required zero file changes: confirmed by re-deriving the Suite Manifest counts from disk (not from re-reading the already-correct prose) and by running the traceability path-check script — both matched exactly what 170-01/02/03 had already written. Documenting a "no-op verified" outcome here rather than making a cosmetic edit to manufacture a commit, per the plan's own instruction that this is a "final reconciliation pass" whose job is catching drift, not guaranteeing busywork.
- Left the existing 5-row FNL-08 traceability set as-is (joincode_test.go, join_test.go, api_test.go, funnel_test.go, SessionSharePanel.test.tsx) even though the plan's must_haves literally name only 4 files — the 5th (api_test.go) was a legitimate, already-committed addition from 170-02's own deviation-tracking and duplicating or removing it would not improve correctness.
- New Category X placed as the next category after the file's last-lettered category (W), with M-46 as the next sequential manual-item number — matching the plan's suggested naming and the existing M-34/M-45 formatting conventions (numbered steps, bold title, `_Why not automatable:_`, `_Source:_`).

## Deviations from Plan

None - plan executed exactly as written. Task 1's "no changes needed" outcome was anticipated by the plan itself (objective explicitly frames this plan as the "final reconciliation pass" over work three prior executors already self-registered incrementally).

## Issues Encountered
None. `bash tests/check-traceability-paths.sh` printed a benign macOS/BSD `grep: invalid option -- P` warning (the script's own internal use of GNU-only `grep -P` on a syntax-check helper path) both before and after this plan's edit — pre-existing script behavior, unrelated to Phase 170's changes, and the script still correctly reported `OK: all traceability paths exist` and exited 0 in both runs. Out of scope per the deviation rules' scope boundary (pre-existing script portability quirk, not caused by this task's changes).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 170 (FNL-08) is now fully closed across all 4 plans: primitive (170-01), daemon wiring (170-02), frontend UI (170-03), and test-suite reconciliation (170-04).
- M-46 is registered as a deferred live-UAT item (release-time, production build + real Funnel-granted tailnet + off-tailnet devices required) — consistent with the existing deferred M-37..M-45 items already tracked in STATE.md's Deferred Items table. No new blocker; this follows the same deferred-UAT class as prior Phase 165-169 manual items.
- Phase 171 (Public Full-Access RW Sharing, FNL-09) is SPEC-FIRST per STATE.md and does not depend on this plan's docs-only work.
- No blockers.

---
*Phase: 170-public-share-access-codes-reusable-share-lifetime-join-code-*
*Completed: 2026-07-05*

## Self-Check: PASSED

`TESTING.md` found on disk; commit `805dbf87` verified present in `git log`.
