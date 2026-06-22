---
phase: 143-regression-test-program
plan: "04"
subsystem: ci
tags: [testing, branch-protection, merge-gate, deferred, github]
status: deferred
dependency_graph:
  requires:
    - 143-03 (TESTING.md Section 3 holds the exact gh api PUT/DELETE commands)
  provides:
    - "Decision record: branch protection deferred until v4.0 CI is green"
  affects:
    - .planning/STATE.md (operator follow-up: apply branch protection once CI green)
tech_stack:
  added: []
  patterns: []
key_files:
  created: []
  modified: []
decisions:
  - "Checkpoint Task 1 (apply|defer) resolved to DEFER — user decision 2026-06-22"
  - "Rationale: applying/verifying a merge gate against red CI is meaningless. v4.0 CI is currently red due to pre-existing bugs (#100 daemon styled-tail data race, #101 internal/files Windows tests), unrelated to Phase 143"
  - "The exact gh api PUT command + DELETE rollback are preserved verbatim in TESTING.md Section 3 — applying later is copy-paste"
  - "TEST-02 (live merge gate) remains PENDING; tracked as an operator follow-up in STATE.md, gated on #100/#101 being fixed and CI going green"
metrics:
  completed_date: "2026-06-22"
  tasks_completed: 0
  tasks_total: 3
  files_changed: 0
---

# Phase 143 Plan 04: Branch Protection Merge Gate — DEFERRED

**One-liner:** The branch-protection merge gate (TEST-02) was deferred at its confirmation checkpoint because pushing v4.0 to remote `main` revealed v4.0 CI is red from two pre-existing bugs (#100, #101); applying a merge gate against red CI would be hollow. The exact `gh api` command is preserved in TESTING.md §3 and the application is tracked as an operator follow-up.

## What Happened

Plan 04 is `autonomous: false` — its Task 1 is a `checkpoint:decision` (apply | defer) that runs **before** any live mutation of GitHub repo settings. At that checkpoint:

1. Inspection found remote `main` was 214 commits behind local v4.0 (the entire milestone was unpushed) and its last CI run was red.
2. The user authorized pushing v4.0 to remote `main` (push ≠ release; releases are tag-gated behind the `release` environment approval). v4.0 was pushed (`974850cc..e7334706`).
3. CI then ran against real v4.0 code for the first time and came back **red**:
   - **Build (all 4 platforms)** failed at "Run Go tests (all platforms, race detector)" — a **data race** in `internal/daemon` styled-tail (`GetSessionStyledTailLines`). Filed as **#100**.
   - **e2e (playwright)** failed; **`internal/files`** Windows-only test failures (pre-existing since v3.6). Filed as **#101**.
4. Both failures pre-date Phase 143 and are out of its scope. Phase 143's purpose — *build the regression test program* — succeeded: it surfaced a real data race that local `go test` (no `-race`) never caught.

## Decision: DEFER

The merge gate is deferred until v4.0 CI is green. Applying branch protection now (against red CI) would:
- make the Task-3 smoke test meaningless, and
- pin required checks the remote cannot currently satisfy.

## Tasks (not executed)

| Task | Type | Status |
|------|------|--------|
| 1. Confirm branch-protection mutation | checkpoint:decision | Resolved → **defer** |
| 2. Apply branch protection via gh api | auto | Not run (gated on CI green) |
| 3. Smoke-test the gate (failing PR blocked) | checkpoint:human-verify | Not run (gated on Task 2) |

## TEST-02 Status

**PENDING.** The live merge gate is not yet applied. Tracked as an operator follow-up in STATE.md, gated on #100 and #101 being fixed and CI going green. Apply via the verbatim command in TESTING.md §3, then run the Task-3 smoke test.

## Self-Check: DEFERRED (by user decision)

- [x] Checkpoint Task 1 reached and resolved to "defer"
- [x] Bugs surfaced by the push filed: #100 (daemon race), #101 (files Windows)
- [x] `gh api` command preserved in TESTING.md §3 for later application
- [x] TEST-02 deferral recorded in STATE.md operator follow-ups
- [ ] Branch protection applied (deferred — gated on CI green)
- [ ] Gate smoke-tested (deferred — gated on application)
