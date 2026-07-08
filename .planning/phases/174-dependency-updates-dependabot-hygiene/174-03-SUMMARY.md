---
phase: 174-dependency-updates-dependabot-hygiene
plan: 03
subsystem: infra
tags: [dependabot, ci, go-modules, github-actions, wails, tailscale]

# Dependency graph
requires:
  - phase: 174-01
    provides: 4 low-risk CI-action Dependabot bumps applied + PRs closed
  - phase: 174-02
    provides: coder/websocket + x/term + nfpm Go module bumps applied + verified
provides:
  - .github/dependabot.yml gomod ignore entries for wails/v2 (>=2.11.0) and tailscale.com (>=1.100.0)
  - .github/dependabot.yml github-actions ignore entry for actions/checkout (semver-major)
  - Dependabot PRs #104, #88, #102 closed citing Phase 174 rationale
affects: [175-web-share-remote-viewer-windowing-bug-fixes, 176-platform-hardening-bug-fixes]

# Tech tracking
tech-stack:
  added: []
  patterns: ["surgical dependabot ignore (versions/update-types) preserves future security patches while deferring a specific risky bump"]

key-files:
  created: []
  modified: [".github/dependabot.yml"]

key-decisions:
  - "Used the versions form (Option A from RESEARCH.md) for wails/v2 and tailscale.com rather than update-types, matching the plan's exact acceptance criteria (versions==['>=2.11.0'] / ['>=1.100.0'])."
  - "Committed both Task 1 (gomod entries) and Task 2 (github-actions entry) as a single commit per Task 3's explicit instruction, since both edit the same file and the plan's commit message describes all three deferrals together."
  - "All three gh pr close calls succeeded without runtime permission denial (unlike 174-02's #89/#106/#105 blocker) — no fallback/orchestrator handoff needed for this plan."

patterns-established:
  - "Surgical ignore entries (versions: ['>=X.Y.Z'] or update-types: ['version-update:semver-major']) are the standing pattern for deferring a specific risky dependency bump without freezing the whole dependency line's security patches."

requirements-completed: [DEP-02]

coverage:
  - id: D1
    description: "dependabot.yml gomod block ignores wails/v2 >=2.11.0 without freezing 1.x patches; guarded by go-webview2 v1.0.19 pin (unchanged)"
    requirement: "DEP-02"
    verification:
      - kind: other
        ref: "python3 yaml assertion: gomod ignore contains github.com/wailsapp/wails/v2 with versions==['>=2.11.0'], go-webview2 entry unchanged"
        status: pass
    human_judgment: false
  - id: D2
    description: "dependabot.yml gomod block ignores tailscale.com >=1.100.0 without freezing 1.98.x/1.99.x patches"
    requirement: "DEP-02"
    verification:
      - kind: other
        ref: "python3 yaml assertion: gomod ignore contains tailscale.com with versions==['>=1.100.0']"
        status: pass
    human_judgment: false
  - id: D3
    description: "dependabot.yml github-actions block ignores actions/checkout major bumps (v7+) while allowing v6 patch/minor updates"
    requirement: "DEP-02"
    verification:
      - kind: other
        ref: "python3 yaml assertion: github-actions ignore contains actions/checkout with update-types==['version-update:semver-major']"
        status: pass
    human_judgment: false
  - id: D4
    description: "Each ignore entry carries an inline comment stating defer rationale + revisit condition"
    requirement: "DEP-02"
    verification:
      - kind: other
        ref: "git diff .github/dependabot.yml — visible inline comments above each of the 3 new entries"
        status: pass
    human_judgment: false
  - id: D5
    description: "Dependabot PRs #104, #88, #102 closed citing this phase's ignore rationale (not merged)"
    requirement: "DEP-02"
    verification:
      - kind: other
        ref: "gh pr view 104/88/102 --json state -q .state == CLOSED (all 3), confirmed via plan's verify command"
        status: pass
    human_judgment: false

duration: 1min
completed: 2026-07-08
status: complete
---

# Phase 174 Plan 03: Dependabot High-Risk Deferrals Summary

**Added 3 surgical `.github/dependabot.yml` ignore entries (wails/v2 >=2.11.0, tailscale.com >=1.100.0, actions/checkout semver-major) and closed Dependabot PRs #104/#88/#102 citing Phase 174 rationale — all three closures succeeded with no permission-classifier block.**

## Performance

- **Duration:** 1 min
- **Started:** 2026-07-08T16:29:48Z
- **Completed:** 2026-07-08T16:31:38Z
- **Tasks:** 3
- **Files modified:** 1

## Accomplishments
- Extended the existing gomod `ignore:` list with `github.com/wailsapp/wails/v2` (`versions: [">=2.11.0"]`) and `tailscale.com` (`versions: [">=1.100.0"]`), each with an inline defer-rationale comment; the pre-existing `go-webview2` entry was left untouched.
- Added a new `ignore:` key to the github-actions ecosystem block (previously had none) for `actions/checkout` scoped to `update-types: ["version-update:semver-major"]`, with an inline rationale comment.
- Committed the config change on `v4.2-funnel-sharing` and closed Dependabot PRs #104, #88, #102 with the exact rationale text specified in the plan — all three closed cleanly.

## Task Commits

Each task was committed atomically:

1. **Task 1 + Task 2: Add gomod ignore entries (wails/v2, tailscale.com) + github-actions ignore entry (actions/checkout)** - `8e8f8263` (ci) — both tasks edit the same file (`.github/dependabot.yml`); committed together per Task 3's instruction to "commit the config" as one step before closing the PRs.
3. **Task 3: Close PRs #104/#88/#102** - no additional file commit (PR-close is a GitHub API action, not a git commit); verified via `gh pr view --json state`.

_Note: Tasks 1 and 2 are file edits on the same target file with no intervening commit boundary specified by the plan; Task 3's "commit the config" instruction covers both._

## Files Created/Modified
- `.github/dependabot.yml` - added 3 surgical ignore entries (gomod: wails/v2, tailscale.com; github-actions: actions/checkout) each with rationale comments

## Decisions Made
- Used the `versions` form (RESEARCH.md Option A) for the gomod entries exactly as the plan's automated verification expects (`versions==['>=2.11.0']` / `['>=1.100.0']`), not the coarser `update-types` alternative.
- Committed Tasks 1 and 2 together since both are edits to the same file and the plan's Task 3 commit message ("defer wails/tailscale/checkout via surgical dependabot ignore entries") names all three deferrals as one unit.

## Deviations from Plan

None - plan executed exactly as written. All three `gh pr close` calls succeeded (no runtime permission-classifier denial this time, unlike the 174-02 precedent for #89/#106/#105); no orchestrator handoff or STATE.md blocker entry needed for this plan's PR closures.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. Note per plan's "Effective-branch note": these ignore entries take effect for Dependabot once `v4.2-funnel-sharing` merges to `main` (Dependabot reads config from the default branch); a transient PR re-open before that merge is expected, not a defect.

## Next Phase Readiness

Phase 174 (dependency-updates-dependabot-hygiene) is now fully executed: 174-01 (4 low-risk CI-action bumps), 174-02 (3 Go module bumps), 174-03 (3 high-risk deferrals) all complete. All 10 originally-enumerated Dependabot PRs have been actioned: 7 merged/applied-and-closed (174-01/174-02), 3 deferred-and-closed (174-03). Remaining open item: STATE.md's existing Blockers section still lists #89/#106/#105 as a carry-forward from 174-02 pending user confirmation those closures landed — this plan's #104/#88/#102 closures are separate and confirmed CLOSED live via `gh pr view`.

---
*Phase: 174-dependency-updates-dependabot-hygiene*
*Completed: 2026-07-08*

## Self-Check: PASSED
