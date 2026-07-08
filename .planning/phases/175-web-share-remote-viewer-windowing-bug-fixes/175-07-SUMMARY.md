---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
plan: 07
subsystem: testing
tags: [documentation, regression-testing, traceability, go, vitest]

# Dependency graph
requires:
  - phase: 175
    provides: "175-02 (new RED test scaffolds), 175-03 (BUG-01 fix), 175-04 (BUG-04 fix), 175-05 (BUG-03 diagnostic instrumentation), 175-06 (BUG-02 fix) — the phase's full set of new/extended test files and the 175-01 live-diagnosis DISPROVED verdict for BUG-03"
provides:
  - "TESTING.md Suite Manifest reconciled to the actual working-tree test-file counts (139 Go / 148 vitest / 9 Playwright / 2 build-script = 298 total), correcting a stale 376/147/9/2/534 that predates this phase"
  - "6 new Section 4 traceability rows for this phase's new/extended test files, mapped to BUG-01..04"
  - "4 new Section 5 manual UAT items (M-48..M-51), one per bug's live-UAT gate"
  - "Category P (M-27/M-28) reconciled to name the post-Phase-159 /app/ React SPA as the reachable web-guest surface, with web/assets/terminal.js labeled dead code"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Suite Manifest correction pattern (170-04/171-04/173-07 precedent): add a NEW dated reconciliation note stating old->new counts, never edit a prior plan's historical note"

key-files:
  created: []
  modified:
    - TESTING.md
    - .planning/phases/175-web-share-remote-viewer-windowing-bug-fixes/deferred-items.md

key-decisions:
  - "Recomputed Go/vitest counts strictly via the plan's specified scoped find commands (find internal -name '*_test.go' + explicit repo-root *_test.go; find frontend/src -name '*.test.ts*'), not a bare find . — this excludes the stale .claude/worktrees/agent-* duplicate checkouts that were almost certainly the source of the prior inflated 376 Go count"
  - "Logged the stale-worktree count-inflation finding to deferred-items.md as an out-of-scope workspace-hygiene follow-up rather than deleting .claude/worktrees/ in this documentation-only plan"
  - "M-50 (BUG-03) is written as a standing regression watch reflecting 175-01's DISPROVED live-diagnosis verdict, not a fresh timed-repro checklist — it directs a future reporter to check for 175-05's diagnostic log lines rather than re-asserting a behavior that already failed to reproduce live"

patterns-established:
  - "M-50's non-reproduction framing: when a live diagnosis DISPROVES a hypothesis and the fix ships as instrumentation rather than a behavior change, the corresponding manual UAT item documents the disproof + what to check on recurrence, instead of prescribing a repro script for a bug that doesn't currently reproduce"

requirements-completed: [BUG-01, BUG-02, BUG-03, BUG-04]

coverage:
  - id: D1
    description: "Suite Manifest Go/vitest counts + Total corrected to match the actual working-tree find output, with a dated old->new reconciliation note"
    requirement: null
    verification:
      - kind: other
        ref: "find internal -name '*_test.go' | wc -l (124) + repo-root *_test.go (15) = 139; find frontend/src -name '*.test.ts*' | wc -l = 148; TESTING.md Suite Manifest table updated to 139/148/9/2/298"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every new test file (app_poll_test.go, session_ended_test.go, scrollback_altscreen_test.go, SessionEndedBanner.test.tsx) plus the two BUG-01 extended files has a Section 4 traceability row with a repo-relative path mapped to its BUG-0x id"
    requirement: null
    verification:
      - kind: other
        ref: "python3 table-row path-existence check over TESTING.md: 241/241 traceability paths resolve on disk (0 missing); bash tests/check-traceability-paths.sh reports OK (known macOS BSD grep -oP limitation means this is a false-OK on this platform, cross-checked with the Python scan per project memory)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Four new M-NN manual checklist items (M-48..M-51) added, one per BUG-0x live gate, with concrete steps + expected outcomes; M-50 correctly reflects the 175-01 DISPROVED verdict rather than prescribing a fresh repro of a non-reproducing bug"
    requirement: null
    verification:
      - kind: other
        ref: "grep -oE 'M-[0-9]+' TESTING.md | sort -u | tail -5 shows M-47..M-51 (highest prior M-47, new M-48/M-49/M-50/M-51 sequential)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Category P (M-27/M-28) no longer calls web/assets/terminal.js 'the web viewer'; it is reconciled to name the /app/ React SPA (TerminalPanel.tsx) as the reachable surface and label terminal.js dead code; M-27/M-28 numbers unchanged"
    requirement: null
    verification:
      - kind: other
        ref: "grep -c 'dead code\\|/app/' TESTING.md -> category P reconciled (>=1 match); manual diff review of the Category P intro paragraph and M-27's Why-not-automatable line"
        status: pass
    human_judgment: false

# Metrics
duration: 7min
completed: 2026-07-08
status: complete
---

# Phase 175 Plan 07: TESTING.md Regression Registration Summary

**Reconciled TESTING.md's Suite Manifest to the actual working-tree test-file counts (correcting a stale, worktree-inflated 376 Go down to the real 139), registered all of Phase 175's new/extended test files in the Traceability map, added four new manual live-UAT items (M-48..M-51) for BUG-01..04 — with M-50 correctly framed as a standing regression watch reflecting BUG-03's DISPROVED live-diagnosis verdict rather than a fresh repro script — and reconciled Category P's stale "the web viewer" wording to the post-Phase-159 `/app/` React SPA architecture.**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-07-08T19:43:45Z
- **Completed:** 2026-07-08T19:51:01Z
- **Tasks:** 2
- **Files modified:** 2 (TESTING.md, deferred-items.md)

## Accomplishments

- **Suite Manifest count correction (376→139 Go, 147→148 vitest, Total 534→298):** while recomputing counts per the plan's exact `find` commands, discovered the previously-documented 376 Go count was drastically inflated relative to the real working-tree state. Root-caused this to three stale `.claude/worktrees/agent-*` leftover parallel-execution checkouts (~348MB, never cleaned up), each containing a full duplicate `internal/` tree with its own `_test.go` files — an unscoped `find . -name '*_test.go'` from the repo root returns 379 (within rounding of the stale 376), strongly suggesting past reconciliation passes were unknowingly counting worktree duplicates rather than the real suite. This plan's counts use only the scoped `find internal ...` / explicit repo-root commands the plan specifies, and add a new dated note (170-04/171-04/173-07 precedent) rather than editing any prior note.
- Added 6 new Section 4 traceability rows: `app_poll_test.go` → BUG-03, `internal/webserver/session_ended_test.go` → BUG-02, `internal/relay/scrollback_altscreen_test.go` → BUG-04, `frontend/src/components/__tests__/SessionEndedBanner.test.tsx` → BUG-02, and `frontend/src/lib/terminalScale.test.ts` + `frontend/src/components/__tests__/TerminalPanel.scale.test.tsx` → BUG-01 (extended in place).
- Added four new Section 5 manual checklist items in a new Category Z: **M-48** (BUG-01 mobile-viewport readability floor), **M-49** (BUG-02 owner-ends-session disconnect banner), **M-50** (BUG-03 — written as a standing regression watch reflecting the 175-01 live-diagnosis DISPROVED verdict, directing a future reporter to check for 175-05's diagnostic instrumentation rather than re-running a timed repro that has already failed to reproduce), **M-51** (BUG-04 live two-client alt-screen reconnect).
- Reconciled Category P's intro paragraph and M-27's "Why not automatable" line: the reachable web-guest surface is now correctly named as the `/app/` React SPA (`WebShareSessionView.tsx` → `TerminalPanel.tsx`, real xterm.js) since the Phase 159 redirect; `web/assets/terminal.js` is explicitly labeled dead code. M-27/M-28 item numbers left unchanged.
- Logged the stale-worktree count-inflation finding to `deferred-items.md` as an out-of-scope workspace-hygiene follow-up (this plan's `files_modified` is `TESTING.md` only — deleting `.claude/worktrees/*` is not a documentation change).

## Task Commits

1. **Task 1: Reconcile Suite Manifest counts + add Section 4 traceability rows** - `a051bb14` (docs)
2. **Task 2: Add Section 5 manual M-NN items + reconcile stale Category P** - `5667327a` (docs)

## Files Created/Modified
- `TESTING.md` - Suite Manifest table counts corrected + new dated reconciliation note; 6 new Section 4 traceability rows; new Category Z with M-48..M-51; Category P intro + M-27 reconciled to the post-Phase-159 `/app/` architecture
- `.planning/phases/175-web-share-remote-viewer-windowing-bug-fixes/deferred-items.md` - new "From 175-07" entry logging the stale `.claude/worktrees/` count-inflation finding as a future workspace-hygiene follow-up

## Decisions Made
- Followed the plan's exact scoped `find` commands (not a bare `find .`) to recompute counts, which is what surfaced and avoided re-propagating the worktree-inflation bug.
- Did not attempt to "fix" or delete the stale `.claude/worktrees/agent-*` directories — out of scope for a documentation-only plan (`files_modified: TESTING.md`); logged instead per the SCOPE BOUNDARY rule.
- Wrote M-50 (BUG-03) as a regression-watch item rather than a repro checklist, since the context note and `175-01-DIAGNOSIS.md` are explicit that the deadline hypothesis was DISPROVED and 175-05 shipped instrumentation, not a behavior fix — prescribing a fresh timed repro would misrepresent the current, already-established state.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Suite Manifest Go count was stale by ~237 files (376 documented vs. 139 actual)**
- **Found during:** Task 1, while recomputing counts per the plan's specified `find` commands.
- **Issue:** The previously-documented Go count (376, carried forward through many prior phase reconciliation notes) was far higher than the actual working-tree count. Root cause: three stale `.claude/worktrees/agent-*` leftover parallel-execution checkout directories, each a full duplicate of `internal/` with its own `_test.go` files, which an unscoped `find .` picks up (confirmed: unscoped count = 379, within rounding of the stale 376).
- **Fix:** Recomputed strictly with the plan's scoped commands (`find internal -name '*_test.go'` + explicit repo-root `*_test.go`, excluding `.claude/worktrees/`), corrected the Suite Manifest table to 139 Go / 148 vitest / 9 Playwright / 2 build-script = 298 total, and added a new dated note explaining the correction and its root cause (not editing any prior note, per the 170-04/171-04/173-07 precedent).
- **Files modified:** `TESTING.md`, `.planning/phases/175-web-share-remote-viewer-windowing-bug-fixes/deferred-items.md` (follow-up logged, not fixed — out of scope)
- **Verification:** `find internal -name '*_test.go' | wc -l` = 124; `find . -maxdepth 1 -name '*_test.go' | wc -l` = 15 (total 139); `find frontend/src -name '*.test.ts*' | wc -l` = 148; `git ls-files` cross-check confirms all counted files are tracked (no untracked drift); Python table-row path-existence scan over TESTING.md's 241 traceability rows finds 0 missing.
- **Committed in:** `a051bb14` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 documentation-accuracy bug, self-caught during the task's own mandatory recompute step)
**Impact on plan:** This IS the task's own stated deliverable ("Recompute the Go and vitest file counts from the actual working tree... add a NEW dated reconciliation note stating the old→new counts") — the magnitude of the correction was larger than the plan's read_first anticipated, but the fix itself is exactly what Task 1 specifies. No scope creep, no runtime code touched.

## Issues Encountered

`bash tests/check-traceability-paths.sh` reports "OK: all traceability paths exist" but its `grep -oP` invocation fails silently on macOS BSD grep ("invalid option -- P"), meaning the official script checks zero paths on this platform and always reports a false-OK (a known, previously-documented project limitation — see project memory and the 173-07 precedent). Cross-verified all 241 Section-4 traceability table-row paths with an equivalent Python regex scan scoped to lines starting with `|`; all 241 resolve to real files on disk (0 missing).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 175 (web-share-remote-viewer-windowing-bug-fixes) is now fully plan-complete: all 7 plans executed, TESTING.md reconciled per the standing Regression Test Convention.
- Remaining open items are all deferred live/manual UATs, tracked as new M-48..M-51 (this plan) plus the pre-existing M-27/M-28 (reconciled wording only, unchanged status): BUG-01 mobile legibility (M-48), BUG-02 disconnect banner (M-49), BUG-03 standing regression watch (M-50, already live-diagnosed DISPROVED — no outstanding repro needed unless it recurs), BUG-04 live alt-screen reconnect (M-51). None of these block phase completion at the code-verification level; they are the phase's manual-UAT surface for `/gsd-verify-work 175`.
- Workspace-hygiene follow-up logged in `deferred-items.md`: stale `.claude/worktrees/agent-*` directories (~348MB) should be cleaned up in a future quick task once confirmed to hold no in-flight work, to prevent future `find`/`grep` sweeps of the repo from re-inflating counts the way the prior Suite Manifest was inflated.

---
*Phase: 175-web-share-remote-viewer-windowing-bug-fixes*
*Completed: 2026-07-08*

## Self-Check: PASSED

Both claimed files confirmed present on disk (`TESTING.md`, `deferred-items.md`).
Both task commit hashes (`a051bb14`, `5667327a`) confirmed present in
`git log --oneline --all`.
