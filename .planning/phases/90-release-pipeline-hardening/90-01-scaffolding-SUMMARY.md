---
phase: "90"
plan: "01"
subsystem: ci-hardening
tags: [ci, hardening, scaffolding, wave-0, sec-09, sec-10]
dependency_graph:
  requires: []
  provides:
    - scripts/grep-gate.sh (SEC-09 + SEC-10 grep gate)
    - tests/build-script.test.sh Section 12 (red assertions for Plan 03)
    - 90-TAP-BRANCH-SETUP.md (D-16 manual prerequisite runbook)
  affects:
    - "Plan 02: tools.go — no overlap, can proceed in parallel"
    - "Plan 03: build.sh — must satisfy Section 12 red tests"
    - "Plan 06: E2E rc verification — requires tap branch doc"
tech_stack:
  added: []
  patterns:
    - bash grep gate with set -euo pipefail
    - TAP-style assertion framework (existing pass/fail helpers)
key_files:
  created:
    - scripts/grep-gate.sh
    - .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md
  modified:
    - tests/build-script.test.sh (+Section 12, 22 lines)
decisions:
  - "grep-gate.sh placed in scripts/ (not .github/workflows/) per D-09 Claude's Discretion — auditor must not live inside the thing it audits"
  - "Section 12 intentionally red at Wave 0 — three failing assertions are the acceptance contract for Plan 03"
  - "TAP-BRANCH-SETUP.md documents human prerequisite for Plan 06 E2E; no automation possible (requires push access to scottkw/homebrew-agenthub)"
metrics:
  duration: "~8 minutes"
  completed: "2026-04-24T12:56:54Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 1
  lines_added: 111
---

# Phase 90 Plan 01: Scaffolding Summary

**One-liner:** SHA-pin regression guard script and red-test scaffold for SEC-09/SEC-10, plus D-16 tap branch runbook.

## What Was Built

Wave 0 of Phase 90 delivers the acceptance infrastructure that subsequent waves must satisfy — zero production code changes, three new artifacts:

1. **`scripts/grep-gate.sh`** — three independent grep checks that together implement the SEC-09 + SEC-10 regression guard:
   - Floating-ref check: rejects `uses:` lines ending with `@main`, `@master`, `@vN`, or `@word`
   - `@latest` check: rejects any `@latest` in workflows, `build.sh`, or `tests/`
   - Non-SHA check: rejects any `uses: owner/repo@X` where X is not a 40-char hex SHA

2. **`tests/build-script.test.sh` Section 12** — three red assertions for SEC-10 compliance:
   - `@latest` must be absent from `build.sh` (currently FAILS — `build.sh:65` has `@latest`)
   - `go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2` pattern must be present (FAILS)
   - `WAILS_PINNED_VER` gate must be present (FAILS)
   All three will turn green when Plan 03 Task 3 lands.

3. **`.planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md`** — runbook documenting the one-time manual step for creating `release-90-test` on `scottkw/homebrew-agenthub` before pushing the `v3.1.0-rc1` tag.

## Current Gate State

- `bash scripts/grep-gate.sh` — exits 1 (FAIL: 32 unpinned action refs detected across 4 workflow files)
- `bash tests/build-script.test.sh` — exits 1 (35 pass, 3 fail — Section 12 red)

Both failures are the intended Wave 0 state. The grep gate will pass after Waves 1-4 (Plans 02-05) land. Section 12 will pass after Plan 03 Task 3.

## Commits

| Task | Commit | Message |
|------|--------|---------|
| Task 1: grep-gate.sh | 4b32dc5 | ci(90): add scripts/grep-gate.sh — SEC-09/SEC-10 regression guard (Phase 90 Wave 0) |
| Task 2: Section 12 + tap-branch doc | 11928da | test(90): scaffold SEC-10 assertions + D-16 tap branch runbook (Phase 90 Wave 0) |

## Deviations from Plan

None — plan executed exactly as written.

## Handoff to Next Plans

- **Plan 02** (tools.go): no file overlap, can proceed immediately
- **Plan 03** (build.sh + CI hardening): must satisfy Section 12's three red assertions and must make grep-gate.sh pass for `build.sh` + `tests/` scope
- **Plan 06** (E2E rc verification): must complete the manual step documented in 90-TAP-BRANCH-SETUP.md before pushing `v3.1.0-rc1`

## Known Stubs

None — this is test infrastructure only; no data flows to UI.

## Self-Check: PASSED

| Item | Status |
|------|--------|
| scripts/grep-gate.sh exists | FOUND |
| 90-TAP-BRANCH-SETUP.md exists | FOUND |
| 90-01-scaffolding-SUMMARY.md exists | FOUND |
| Commit 4b32dc5 exists | FOUND |
| Commit 11928da exists | FOUND |
