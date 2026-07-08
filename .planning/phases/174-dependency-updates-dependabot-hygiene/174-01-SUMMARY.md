---
phase: 174-dependency-updates-dependabot-hygiene
plan: 01
subsystem: infra
tags: [github-actions, dependabot, ci, supply-chain, sha-pinning]

# Dependency graph
requires: []
provides:
  - "4 low-risk CI-action Dependabot bumps applied as SHA-pinned edits on v4.2-funnel-sharing"
  - "Dependabot PRs #114/#113/#103/#85 closed citing Phase 174"
affects: [174-02, 174-03]

# Tech tracking
tech-stack:
  added: []
  patterns: ["SHA-pinned action bumps applied via verbatim-copy replace-all, verified with positive-count grep + yaml.safe_load before commit"]

key-files:
  created: []
  modified:
    - .github/workflows/build.yml
    - .github/workflows/e2e.yml
    - .github/workflows/release.yml

key-decisions:
  - "Applied bumps directly on v4.2-funnel-sharing (the active integration branch) rather than merging the Dependabot PRs into main; each PR closed with a comment citing Phase 174 per the RESEARCH.md merge-strategy recommendation (A1)."

patterns-established:
  - "Multi-file SHA bump: replace-all per file using the exact old-SHA string, then a positive-count grep assertion per new SHA across all touched files (not just a single-file check) to catch partial-occurrence drift (Pitfall 3)."

requirements-completed: [DEP-01]

coverage:
  - id: D1
    description: "actions/setup-go bumped 6.4.0->6.5.0 (SHA 924ae3a1...) in all 3 workflow files, 6 total occurrences"
    requirement: "DEP-01"
    verification:
      - kind: other
        ref: "grep -rho 'setup-go@924ae3a1cded613372ab5595356fb5720e22ba16' .github/workflows/*.yml | wc -l == 6"
        status: pass
    human_judgment: false
  - id: D2
    description: "pnpm/action-setup bumped 6.0.8->6.0.9 (SHA 0ebf4713...) in all 3 workflow files, 6 total occurrences"
    requirement: "DEP-01"
    verification:
      - kind: other
        ref: "grep -rho 'action-setup@0ebf47130e4866e96fce0953f49152a61190b271' .github/workflows/*.yml | wc -l == 6"
        status: pass
    human_judgment: false
  - id: D3
    description: "actions/attest-build-provenance bumped 4.1.0->4.1.1 (SHA 0f67c3f4...) in release.yml, 4 occurrences"
    requirement: "DEP-01"
    verification:
      - kind: other
        ref: "grep -o 'attest-build-provenance@0f67c3f4856b2e3261c31976d6725780e5e4c373' .github/workflows/release.yml | wc -l == 4"
        status: pass
    human_judgment: false
  - id: D4
    description: "softprops/action-gh-release bumped 3.0.0->3.0.1 (SHA 718ea10b...) in release.yml, 1 occurrence"
    requirement: "DEP-01"
    verification:
      - kind: other
        ref: "grep -o 'action-gh-release@718ea10b132b3b2eba29c1007bb80653f286566b' .github/workflows/release.yml | wc -l == 1"
        status: pass
    human_judgment: false
  - id: D5
    description: "All 3 workflow files still parse as valid YAML after the edits"
    requirement: "DEP-01"
    verification:
      - kind: other
        ref: "python3 -c \"import yaml; [yaml.safe_load(open(f)) for f in [...]]\""
        status: pass
    human_judgment: false
  - id: D6
    description: "Dependabot PRs #114, #113, #103, #85 closed citing Phase 174, none merged"
    requirement: "DEP-01"
    verification:
      - kind: other
        ref: "gh pr view <n> --json state -q .state == CLOSED for 114/113/103/85"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-07-08
status: complete
---

# Phase 174 Plan 01: Low-Risk CI-Action Dependabot Bumps Summary

**Four SHA-pinned CI-action bumps (setup-go 6.5.0, pnpm/action-setup 6.0.9, attest-build-provenance 4.1.1, action-gh-release 3.0.1) applied across build.yml/e2e.yml/release.yml on v4.2-funnel-sharing, with the corresponding Dependabot PRs #114/#113/#103/#85 closed citing this phase.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-07-08T16:10:09Z
- **Completed:** 2026-07-08T16:13:12Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- Bumped `actions/setup-go` and `pnpm/action-setup` SHAs across all 3 workflow files (6 occurrences each) via replace-all edits, verified by positive-count grep across all files (not just build.yml).
- Bumped `actions/attest-build-provenance` (4 occurrences) and `softprops/action-gh-release` (1 occurrence) in `release.yml`, preserving SHA-pinning throughout (no floating tags introduced).
- Closed Dependabot PRs #114, #113, #103, #85 with comments citing Phase 174 and confirming the bumps ship with v4.2 rather than merging via the Dependabot PR itself.

## Task Commits

Each task was committed atomically:

1. **Task 1+2: Bump setup-go, pnpm/action-setup, attest-build-provenance, action-gh-release** - `b8940589` (ci) — Tasks 1 and 2 were combined into a single commit per Task 3's explicit instruction (`git status` was checked between edits and verification but no intermediate commit was made, matching the plan's specified commit boundary).
2. **Task 3: Close Dependabot PRs #114/#113/#103/#85** - no code commit (GitHub PR-state changes only, verified via `gh pr view`).

**Plan metadata:** (pending — final docs commit follows this SUMMARY)

## Files Created/Modified
- `.github/workflows/build.yml` - setup-go + pnpm/action-setup SHA bumps (1 occurrence each)
- `.github/workflows/e2e.yml` - setup-go + pnpm/action-setup SHA bumps (1 occurrence each)
- `.github/workflows/release.yml` - attest-build-provenance (x4), setup-go (x4), action-gh-release (x1), pnpm/action-setup (x4) SHA bumps

## Decisions Made
- Applied the bumps directly on `v4.2-funnel-sharing` (active integration branch, 376 commits ahead of main at research time) rather than merging the Dependabot PRs into `main`, per RESEARCH.md's merge-strategy recommendation. Each PR was closed with a comment citing Phase 174 instead of merged, since their `base=main` lacks the v4.2 commits and their CI is stale/flaky.

## Deviations from Plan

None - plan executed exactly as written. Task 1 and Task 2 edits were both completed before the single combined commit specified in Task 3, matching the plan's literal instruction ("Stage and commit the workflow edits from Tasks 1-2 as a single commit").

## Issues Encountered

None. All verification commands from the plan (YAML parse checks, positive-SHA-count greps, PR-state checks) passed on first attempt.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plans 174-02 (Go module bumps: coder/websocket, x/term, nfpm) and 174-03 (dependabot.yml ignore entries + close deferred PRs #104/#88/#102) are independent of this plan's changes and can proceed.
- No blockers identified.

---
*Phase: 174-dependency-updates-dependabot-hygiene*
*Completed: 2026-07-08*

## Self-Check: PASSED

All modified files and commit hashes verified present on disk / in git log.
