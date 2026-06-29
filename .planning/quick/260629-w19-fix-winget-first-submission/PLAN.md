---
task: Fix WinGet first-submission so `winget install scottkw.agenthub` works
slug: 260629-w19-fix-winget-first-submission
date: 2026-06-29
status: in-progress
---

# Fix WinGet first submission

## Problem
`winget install scottkw.agenthub` does not work — the package was never
created in the `microsoft/winget-pkgs` catalog. The v4.0 and v4.1
`distribute.yml → submit-winget` jobs both failed silently (masked by
`continue-on-error: true`).

## Root cause (two defects)
1. **Wrong gate state** — the `WINGET_FIRST_SUBMISSION` repo variable was
   never set, so the job ran `wingetcreate update` (steady-state) instead of
   the first-submission path. `update` failed: the package does not exist in
   the catalog yet (`repos/.../manifests/s/scottkw/agenthub was not found`).
2. **Invalid first-submission command** — even on the `true` path, the job ran
   `wingetcreate new --urls --version --package-identifier --submit`. Per
   Microsoft's winget-create docs, **`new` is an interactive wizard and does
   not support any of those flags**. It can't supply the required
   Publisher/License/Description non-interactively. Confirmed live: every flag
   rejected as "unknown". The job also had **no `actions/checkout`**, so it
   could not see the repo's prepared manifest templates.

## Fix (matches the repo's own design — prepared templates + `submit`)
In `.github/workflows/distribute.yml` `submit-winget` job:
- Add `actions/checkout` (pinned v6.0.3) so `packaging/winget/` is available.
- First-submission branch: download the release `checksums.txt`, run
  `populate-manifests.sh <version> checksums.txt` to fill the templates, then
  `wingetcreate submit --prtitle "..." --no-open <output-dir>`. Token comes
  from the `WINGET_CREATE_GITHUB_TOKEN` job env (no `--token`, avoids logging).
- Steady-state `update` branch unchanged.
- Add explicit `$LASTEXITCODE` failure check.

## Verification
- `python3 yaml.safe_load` on distribute.yml → OK.
- `bash packaging/winget/dry-run-first-submission.sh` against live v4.1 →
  3 manifests populate + validate; installer SHA256 matches release
  checksums.txt (`c8ccb97…272ea1`).
- Re-trigger `distribute.yml -f tag=v4.1` → `submit-winget` opens a real PR to
  `microsoft/winget-pkgs` (operator then shepherds Microsoft review).

## Operator follow-up after Microsoft merges the PR
- `gh variable set WINGET_FIRST_SUBMISSION --body false` (or delete it).
- Remove `continue-on-error: true` from the `submit-winget` job.
- `winget install scottkw.agenthub` to confirm.
