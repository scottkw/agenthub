---
phase: 168-bug-fix-settings-polish
plan: 07
subsystem: testing
tags: [documentation, traceability, regression-suite, testing-md]

# Dependency graph
requires:
  - phase: 168-01
    provides: "Hub.RemoteViewerCount() FIX-04 test coverage (hub_test.go, engine_test.go, api_test.go — extended in place, no new files)"
  - phase: 168-02
    provides: "FIX-01 web-guest plugin-config self-fetch/SSE test coverage (WebShareSessionView.plugin-config.test.tsx)"
  - phase: 168-03
    provides: "FIX-03 in-app tab open behavior (App.open-remote.test.tsx extended)"
  - phase: 168-04
    provides: "UX-01 stay-on-hub test coverage"
  - phase: 168-05
    provides: "UX-02 footer Share modal test coverage"
  - phase: 168-06
    provides: "FIX-02 disconnect-viewers test coverage"
provides:
  - "TESTING.md §4 Traceability map has rows for all six Phase 168 requirements (FIX-01, FIX-02, FIX-03, FIX-04, UX-01, UX-02)"
  - "TESTING.md §5 M-13 reworded to describe FIX-03's in-app-tab behavior (was the retired external-browser flow)"
  - "TESTING.md §5 new M-42 (FIX-02 live two-viewer smoke test) and M-43 (FIX-01 live browser-console CSP/hot-swap check)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Discovered mid-milestone convention drift: plans 168-01..06 each self-registered their own TESTING.md changes inline (docs commits per-plan) rather than deferring to this isolated plan as originally designed — this plan's remaining work was the two gaps those self-registrations missed (FIX-04 traceability rows, M-13 reword, M-42/M-43 manual items), not a from-scratch registration pass."

key-files:
  created: []
  modified:
    - TESTING.md

key-decisions:
  - "Fixed a pre-existing off-by-one in the §2 Suite Manifest Total (524 -> 525) discovered while cross-checking counts against actual files on disk (374 Go + 140 vitest + 9 Playwright + 2 build-script = 525, not 524) — Rule 1 bug fix, directly in the table this plan is responsible for maintaining."
  - "web-plugin-hot-swap.spec.ts registered as a documented adaptation/reference row under FIX-01, explicitly NOT a passing gate (still test.describe.skip pending a separate addon-load-mechanism investigation) — per the plan's explicit prohibition against blind-un-skipping it."
  - "M-13 reworded in place (not a new M-NN item) per the plan's explicit prohibition against adding a separate FIX-03 manual item."

requirements-completed: [FIX-01, FIX-02, FIX-03, FIX-04, UX-01, UX-02]

coverage:
  - id: D1
    description: "Every new automated test file added across plans 01-06 is registered in TESTING.md §2 Suite Manifest and §4 Traceability map with a repo-relative path only; check-traceability-paths.sh passes"
    requirement: "FIX-01"
    verification:
      - kind: other
        ref: "bash tests/check-traceability-paths.sh"
        status: pass
    human_judgment: false
  - id: D2
    description: "TESTING.md §4 has traceability rows for FIX-04 (Phase 168-01), previously missing entirely — internal/relay/hub_test.go, internal/daemon/engine_test.go, internal/daemon/api_test.go"
    requirement: "FIX-04"
    verification:
      - kind: other
        ref: "bash tests/check-traceability-paths.sh"
        status: pass
    human_judgment: false
  - id: D3
    description: "Category G / M-13 is reworded to describe FIX-03's new in-app-tab behavior (not the retired external-browser flow); no duplicate FIX-03 manual item added"
    requirement: "FIX-03"
    verification:
      - kind: other
        ref: "grep -q 'in-app' TESTING.md"
        status: pass
    human_judgment: false
  - id: D4
    description: "New manual M-NN items exist for the FIX-02 live two-viewer smoke test (M-42) and the FIX-01 live browser-console CSP/hot-swap check (M-43)"
    requirement: "FIX-02"
    verification:
      - kind: other
        ref: "grep -Eq \"two-(browser|viewer)|two browser|Disconnect all viewers\" TESTING.md"
        status: pass
    human_judgment: false
  - id: D5
    description: "Full phase verification gate passes: go test -race -short ./..., pnpm vitest run (frontend), pnpm exec tsc --noEmit"
    verification:
      - kind: other
        ref: "go test -race -short ./..."
        status: pass
      - kind: other
        ref: "cd frontend && pnpm vitest run"
        status: pass
      - kind: other
        ref: "cd frontend && pnpm exec tsc --noEmit"
        status: pass
    human_judgment: false

# Metrics
duration: 15min
completed: 2026-07-01
status: complete
---

# Phase 168 Plan 07: Wire tests + manual checklist into TESTING.md Summary

**TESTING.md is now the complete, accurate canonical regression record for Phase 168: all six requirements (FIX-01..04, UX-01, UX-02) have traceability rows, M-13 reflects the shipped in-app-tab behavior, two new live-only manual checks (M-42, M-43) cover what code alone can't prove, and a stale Suite Manifest total was corrected.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-07-01T16:52:00-05:00 (approx, first Read)
- **Completed:** 2026-07-01T16:58:31-05:00 (last commit)
- **Tasks:** 2 completed
- **Files modified:** 1 (TESTING.md)

## Accomplishments

- **Discovered and closed the actual gap:** plans 168-01 through 168-06 had each already self-registered their own TESTING.md changes inline (as documented in git history: `docs(168-02)`, `docs(168-04)`, `docs(168-06)`, etc. commits), contrary to this plan's original premise that TESTING.md would be untouched until this isolated final plan. The real remaining work was narrower than the plan's `<read_first>` implied — the specific gaps were: FIX-04 had zero traceability rows anywhere in the file, `web-plugin-hot-swap.spec.ts` had no documented-adaptation row, M-13 still described the retired external-browser flow, and no manual items existed for the FIX-02 two-viewer or FIX-01 hot-swap/CSP live checks.
- Added 4 new §4 Traceability rows: 3 for FIX-04 (`internal/relay/hub_test.go`, `internal/daemon/engine_test.go`, `internal/daemon/api_test.go` — all extended in place, no new files) and 1 for FIX-01's `frontend/e2e/web-plugin-hot-swap.spec.ts` as an explicit documented-adaptation/reference row (still skipped, not a passing gate).
- Added a Suite Manifest `>` note for Phase 168-01 (the only Phase-168 plan lacking one), matching the established per-plan documentation pattern.
- Reworded M-13 (Category G) in place to describe FIX-03's shipped in-app-tab behavior, added a third sub-scenario (two-session isolation) reflecting 168-03's actual D3 coverage, and updated the `_Source:_` line — without adding a new M-NN item (per the plan's explicit prohibition).
- Added Category V with two new manual checklist items: **M-42** (FIX-02, #117 — live two-real-browser-viewer smoke test, neither viewer kicked, "Disconnect all viewers" drops both without revoking the cap) and **M-43** (FIX-01, #112 — live browser-console check that plugin-config hot-swaps on the `/app/` share URL without reload, notes the documented no-CSP-header caveat).
- Fixed a pre-existing arithmetic bug in the §2 Suite Manifest Total: the four group counts (374 Go + 140 vitest + 9 Playwright + 2 build-script) sum to 525, but the table's printed `**Total**` said 524. Verified against actual `find` counts on disk before fixing.

## Task Commits

1. **Task 1: Register new test files in Suite Manifest + Traceability map** - `07894fb5` (docs)
2. **Task 2: Reword M-13 and add manual M-NN items** - `a9d776e3` (docs)

**Plan metadata:** (this commit, docs: complete plan)

## Files Created/Modified

- `TESTING.md` - §2 Suite Manifest Total fixed (524→525) + new Phase 168-01 note; §4 Traceability map gained 4 rows (FIX-04 ×3, FIX-01 web-plugin-hot-swap.spec.ts documented-adaptation row); §5 M-13 reworded for the in-app-tab behavior + new Category V (M-42, M-43)

## Decisions Made

- Split the two tasks into separate commits by applying targeted diff hunks (`git apply` on hand-built patch files) rather than editing-then-splitting via `git add -p`, since the two tasks' edits landed in disjoint regions of the same file (Suite Manifest/Traceability near the top, Manual Checklist near the bottom) — this kept each commit's diff scoped exactly to its task.
- Fixed the Suite Manifest Total off-by-one as a Rule 1 bug fix (directly in the table this plan is responsible for maintaining), verified against `find` counts on disk rather than assumed.
- Registered `web-plugin-hot-swap.spec.ts` as an explicit non-passing documented-adaptation row rather than leaving it unmentioned, satisfying the plan's "record it as a documented adaptation/reference item" instruction literally.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Suite Manifest Total arithmetic error (524 → 525)**
- **Found during:** Task 1 (cross-checking Suite Manifest counts against `find` while preparing to add FIX-04 rows)
- **Issue:** §2 Suite Manifest's per-group counts (374 Go / 140 vitest / 9 Playwright / 2 build-script) sum to 525, but the printed `**Total**` row said 524 — a stale/miscalculated total left over from a prior plan's commit.
- **Fix:** Corrected `**Total**` to 525, matching both the row sum and live `find -name "*_test.go"` / `*.test.ts(x)` / `*.spec.ts` counts on disk.
- **Files modified:** TESTING.md
- **Verification:** Manual `find` recount confirms 374/140/9/2; `bash tests/check-traceability-paths.sh` still passes.
- **Committed in:** 07894fb5 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix, Rule 1)
**Impact on plan:** Directly in scope — the total is part of the exact table Task 1's acceptance criteria references ("All new test files appear in the Suite Manifest"). No scope creep beyond the table this plan owns.

## Issues Encountered

`bash tests/check-traceability-paths.sh` fails to run its intended `grep -oP` check on this macOS dev machine (BSD grep has no `-P` flag) — the script's own comment documents this as a known Linux-CI-only limitation, and it degrades to a silent no-op "OK" rather than erroring. Compensated by manually verifying all 101 unique traceability paths resolve on disk via a `perl`-based equivalent regex before and after each edit. The real CI check runs on `ubuntu-latest` with GNU grep and will catch anything this local run could have missed; none was found.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- TESTING.md is now a complete and internally-consistent canonical regression record for all of Phase 168 (FIX-01..04, UX-01, UX-02) — no further TESTING.md work is expected for this phase.
- Two new manual checklist items (M-42, M-43) and the reworded M-13 are ready to run as part of the phase's live UAT pass; they require a live daemon + real browser clients (M-42 needs two independent browser connections, M-43 needs DevTools Console access) and cannot be exercised through headless automation.
- The full phase-gate verification (`go test -race -short ./...`, `pnpm vitest run`, `pnpm exec tsc --noEmit`) all pass clean as of this plan's completion, confirming plans 01-06's implementations remain green together.
- Standing-convention observation for future milestones: since 168-01..06 each self-registered TESTING.md changes as they landed (contrary to this plan's "touch TESTING.md exactly once" premise), a future milestone could either (a) keep this final isolated-plan pattern as a safety-net gap-closure pass (as it effectively functioned here), or (b) drop the dedicated final plan and rely on each phase's own self-registration, updating the standing convention in TESTING.md §6 / CLAUDE.md accordingly. Not actioned here — out of scope for this plan.

---
*Phase: 168-bug-fix-settings-polish*
*Completed: 2026-07-01*
