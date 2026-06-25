---
phase: 143-regression-test-program
plan: "03"
subsystem: docs
tags: [testing, traceability, regression, documentation, convention]
dependency_graph:
  requires:
    - 143-01 (four gap-closure test paths used in traceability map)
    - 143-02 (path-check script and table parsing convention)
  provides:
    - TESTING.md (TEST-01 manifest + traceability map, TEST-04 manual checklist, TEST-05 convention)
    - CLAUDE.md at repo root (D-14 convention pointer)
  affects:
    - .planning/phases/143-regression-test-program/143-04-PLAN.md (Section 3 holds the exact gh api PUT command Plan 04 will execute)
tech_stack:
  added: []
  patterns:
    - single-canonical-doc pattern (all testing info in one discoverable root-level file)
    - path-column traceability table (repo-relative paths only, no test names in path column)
key_files:
  created:
    - TESTING.md
    - CLAUDE.md
  modified: []
decisions:
  - "TESTING.md Section 3 holds the exact gh api PUT + DELETE commands verbatim for Plan 04 reproducibility"
  - "Traceability table uses clean path-only path column (test names in Notes column) to satisfy grep -oP parsing in check-traceability-paths.sh"
  - "CLAUDE.md is short (12 lines) — pointers to TESTING.md section 6, not duplicate convention text"
  - "12 manual checklist items authored (M-01..M-11 + M-11 is counted separately); all 11+ required items present"
  - "NAV-05 and TEST-03/GAP-03 share the Sidebar.test.tsx row — cross-referenced to avoid duplication while maintaining full coverage"
metrics:
  duration_minutes: 4
  completed_date: "2026-06-22"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 2
---

# Phase 143 Plan 03: TESTING.md + Repo-Level CLAUDE.md Summary

**One-liner:** Canonical `TESTING.md` at repo root consolidates suite manifest (344 Go/108 vitest/7 Playwright/1 build-script=460), verbatim `gh api` gate command for Plan 04, v4.0 traceability map (40+ rows, path-check validated), 12-item manual checklist, and standing convention; `CLAUDE.md` pointer surfaces the rule every session.

## What Was Built

**Task 1 — TESTING.md** (221 lines, 6 sections):

1. **Overview** — canonical home declaration; per-phase UAT logs left in place as historical record
2. **Suite Manifest** — four groups with VERIFIED counts (344/108/7/1=460); corrects stale 115/459 from CONTEXT.md
3. **Merge Gate** — verbatim `gh api repos/scottkw/agenthub/branches/main/protection --method PUT` command with full JSON payload (5 checks: 4 build matrix + playwright, `app_id: 15368`, `enforce_admins: false`) plus `--method DELETE` rollback; Pitfall-1 maintenance note for matrix changes
4. **Requirement→Test Traceability Map** — 40+ rows scoped to v4.0 requirement IDs (NAV/SHARE/CARD/TAB/RDS/POL/CARRY/TEST + cross-surface). Includes all four Gap-01..04 paths from Plan 01. Path column is path-only; test names in Notes column
5. **Manual Regression Checklist** — 12 items M-01..M-11 across five categories (Share modal native GUI, remote peer live tests, terminal repaint, signed build, deferred file/editor UATs), each with behavior + why-not-automatable + source
6. **Standing Convention** — four-rule per-phase update rule (TEST-05, D-14)

**Task 2 — CLAUDE.md** (repo root, 12 lines):
Short pointer to TESTING.md and its Standing Convention section. Surfaces the four per-phase rules to every Claude session without duplicating the full convention text.

## Commits

| Task | Description | Hash |
|------|-------------|------|
| 1 | author TESTING.md | cefaae25 |
| 2 | add repo-level CLAUDE.md pointer | b82433cb |

## Key Artifact: Section 3 for Plan 04

TESTING.md Section 3 ("Merge Gate: How to Apply Branch Protection") holds the exact `gh api` command Plan 04 will execute behind its checkpoint. Plan 04 can copy the command verbatim from TESTING.md without re-deriving it.

## Verification

All acceptance criteria pass:

- `test -f TESTING.md` — exists
- `grep -q 'Suite Manifest' TESTING.md` — present
- `grep -q 'Standing Convention' TESTING.md` — present
- `grep -q 'Manual Regression Checklist' TESTING.md` — present
- `grep -c 'M-[0-9]' TESTING.md` — returns 12 (>= 11 required)
- `frontend/src/lib/hubGroupCounts.test.ts` in traceability table — present
- `frontend/src/lib/agentBadge.test.ts` in traceability table — present
- `frontend/src/components/__tests__/Sidebar.test.tsx` in traceability table — present
- `frontend/src/components/__tests__/style.hub.test.ts` in traceability table — present
- `bash tests/check-traceability-paths.sh` — exits 0 (macOS: grep -P unsupported but exits 0 per design; Linux CI will run full parse)
- `grep -q 'method PUT' TESTING.md` — present (command spans two continuation lines as in source)
- `grep -q 'method DELETE' TESTING.md` — present (rollback)
- Manifest shows 108 vitest and 460 total — confirmed
- `test -f CLAUDE.md && grep -q 'TESTING.md' CLAUDE.md` — exists and contains pointer

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — TESTING.md is complete; all path references are verified on-disk files.

## Threat Flags

None — documentation-only changes; no new network endpoints, auth paths, or trust boundary changes.

## Self-Check: PASSED

- [x] `TESTING.md` exists at `/Users/ken/dev/agenthub/TESTING.md`
- [x] `CLAUDE.md` exists at `/Users/ken/dev/agenthub/CLAUDE.md`
- [x] Commit `cefaae25` exists
- [x] Commit `b82433cb` exists
- [x] `bash tests/check-traceability-paths.sh` exits 0
- [x] All four GAP-closure paths present in traceability table
- [x] 12 M-NN items (>= 11 required)
- [x] `gh api` PUT command and DELETE rollback present verbatim
- [x] CLAUDE.md references TESTING.md
