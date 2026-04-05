---
phase: 48-winget-distribution
plan: "01"
subsystem: infra
tags: [winget, github-actions, distribute, windows, packaging]

requires:
  - phase: 47-homebrew-tap-packaging-templates
    provides: packaging/winget/manifests/ template files with {{VERSION}} and {{WINDOWS_SHA256}} tokens
  - phase: 46-release-build-pipeline
    provides: release.yml producing agenthub-v*-windows-amd64-installer.exe NSIS artifact and checksums.txt

provides:
  - submit-winget job in distribute.yml using vedantmgoyal9/winget-releaser@main
  - populate-manifests.sh helper script for one-time manual WinGet first submission

affects:
  - 48-winget-distribution plan 02 (manual first submission depends on this tooling)
  - future releases (distribute.yml submit-winget job triggers on every release:published)

tech-stack:
  added:
    - vedantmgoyal9/winget-releaser@main (GitHub Action wrapping Komac for WinGet PR automation)
  patterns:
    - Parallel distribution jobs in distribute.yml (update-homebrew-tap + submit-winget run independently)
    - Restrictive installer regex to exclude bare EXE from NSIS installer matching

key-files:
  created:
    - packaging/winget/populate-manifests.sh
    - packaging/winget/.gitignore
  modified:
    - .github/workflows/distribute.yml

key-decisions:
  - "installers-regex uses 'agenthub-v[\\d.]+-windows-amd64-installer\\.exe$' to exclude the bare EXE artifact — default regex matches all .exe files"
  - "submit-winget job has no needs: dependency — runs in parallel with update-homebrew-tap"
  - "populate-manifests.sh guards against v-prefix in VERSION argument (Pitfall 3: WinGet PackageVersion must not have v prefix)"

patterns-established:
  - "Pattern: Restrictive installer regex in winget-releaser prevents matching both EXE artifacts"
  - "Pattern: VERSION validation (no v prefix) guards WinGet manifest schema compliance"

requirements-completed: [DIST-03]

duration: 2min
completed: 2026-04-05
---

# Phase 48 Plan 01: WinGet Distribution — Automated Job and Manual Submission Tooling Summary

**winget-releaser job added to distribute.yml with restrictive installer regex, plus populate-manifests.sh helper for one-time manual WinGet first submission**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-05T21:45:07Z
- **Completed:** 2026-04-05T21:46:41Z
- **Tasks:** 2
- **Files modified:** 3 (distribute.yml modified; populate-manifests.sh, .gitignore created)

## Accomplishments

- Added `submit-winget` job to distribute.yml using `vedantmgoyal9/winget-releaser@main` with restrictive regex that matches only the NSIS installer (not bare EXE)
- Created `populate-manifests.sh` helper script that validates VERSION format, extracts SHA256 from checksums.txt, and populates all three WinGet manifest templates for manual first submission
- Added `.gitignore` in packaging/winget/ to prevent populated manifests from being committed

## Task Commits

Each task was committed atomically:

1. **Task 1: Add submit-winget job to distribute.yml** - `c89dca1` (feat)
2. **Task 2: Create manifest population helper script** - `d5efa33` (feat)

## Files Created/Modified

- `.github/workflows/distribute.yml` - Added `submit-winget` job alongside existing `update-homebrew-tap`
- `packaging/winget/populate-manifests.sh` - Helper script for manual first submission; validates inputs, extracts SHA256, populates templates
- `packaging/winget/.gitignore` - Excludes `output/` directory from git

## Decisions Made

- Used `installers-regex: 'agenthub-v[\d.]+-windows-amd64-installer\.exe$'` (not the default) to prevent matching the bare `agenthub-v*-windows-amd64.exe` artifact alongside the NSIS installer
- `submit-winget` job has no `needs:` key — runs in parallel with `update-homebrew-tap` since they are independent (different package managers, different targets)
- `populate-manifests.sh` guards against v-prefix in VERSION argument with explicit validation and error message (Pitfall 3 from research)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None — both tasks executed cleanly. The worktree did not have distribute.yml or packaging/winget/ (phases 46-47 files), so those prerequisite files were brought forward from the main branch before adding the new changes.

## User Setup Required

Before the automated `submit-winget` job can function:
1. **Fork microsoft/winget-pkgs** under the `scottkw` GitHub account (required by winget-releaser for PR submission)
2. **Create `WINGET_TOKEN` secret** in the scottkw/agenthub repo — must be a classic PAT (not fine-grained) with `public_repo` scope
3. **Complete manual first submission** (Phase 48 Plan 02) — winget-releaser only works after `scottkw.agenthub` exists in microsoft/winget-pkgs

Use `packaging/winget/populate-manifests.sh <VERSION> <checksums.txt>` to generate submission-ready manifests.

## Next Phase Readiness

- distribute.yml `submit-winget` job is ready to trigger on `release:published` events
- populate-manifests.sh is ready for use in Phase 48 Plan 02 manual first submission
- All prerequisite tooling is in place; manual steps (fork, PAT, first PR) remain for Phase 48 Plan 02

---
*Phase: 48-winget-distribution*
*Completed: 2026-04-05*
