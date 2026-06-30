---
phase: 156-install-links-distribution
plan: "03"
subsystem: packaging
tags: [winget, distribute, dry-run, operator-runbook, bash, YAML-validation]
status: complete
completed: "2026-06-27"
duration: "~3 minutes"
tasks_completed: 2
files_changed: 3

dependency_graph:
  requires:
    - 156-01 (WelcomeTab.tsx strings fixed)
    - 156-02 (scripts/install.sh POSIX installer)
  provides:
    - packaging/winget/dry-run-first-submission.sh (end-to-end manifest dry-run)
    - packaging/winget/FIRST-SUBMISSION-RUNBOOK.md (operator procedure, 7 steps)
    - TESTING.md (M-26 manual item, Category O)
  affects:
    - INSTALL-03 (repo-side winget automation proven correct by dry-run)

tech_stack:
  added: []
  patterns:
    - bash dry-run script (set -euo pipefail, mktemp temp-file cleanup with trap)
    - python3 yaml.safe_load as portable YAML validator (no yamllint dependency)
    - grep+sed GitHub API tag_name extraction (no jq dependency)

key_files:
  created:
    - packaging/winget/dry-run-first-submission.sh
    - packaging/winget/FIRST-SUBMISSION-RUNBOOK.md
  modified:
    - TESTING.md

decisions:
  - "python3 yaml.safe_load chosen over yamllint for portability — no extra install required; stdlib since Python 3.x"
  - "grep+sed for GitHub API tag_name extraction (no jq dependency, same pattern as install.sh)"
  - "VERSION stripped of v prefix via ${TAG#v} before passing to populate-manifests.sh (Pitfall 7)"
  - "WINGET_TOKEN scope locked to public_repo only (least privilege, T-156-07)"
  - "continue-on-error removal explicitly documented as post-acceptance step (T-156-08)"
  - "M-26 steps 1-3 are phase gate; steps 4-6 are explicitly documented as non-blockers"
  - "INSTALL-03 has no automated test file — verified by M-26 manual checklist only (per RESEARCH)"

metrics:
  duration: "~3 minutes"
  completed: "2026-06-27"
---

# Phase 156 Plan 03: WinGet First-Submission Dry-Run and Operator Runbook Summary

Reproducible dry-run helper for distribute.yml's winget first-submission path, plus the operator runbook for the live (externally-gated) microsoft/winget-pkgs PR submission.

## One-liner

Bash dry-run helper that resolves the latest GitHub release tag, fetches real checksums, runs populate-manifests.sh, validates all generated YAML via python3 yaml.safe_load, and asserts scottkw.agenthub identity and windows-amd64-installer.exe URL — plus a 7-step operator runbook for the live submission.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Reproducible winget manifest dry-run helper | 9c0b33f0 | packaging/winget/dry-run-first-submission.sh |
| 2 | Operator runbook + TESTING.md M-26 | ec3ad849 | packaging/winget/FIRST-SUBMISSION-RUNBOOK.md, TESTING.md |

## What Was Built

- **packaging/winget/dry-run-first-submission.sh** — bash script (`set -euo pipefail`, executable bit) with 5 steps:
  - Step 1: Resolves latest release tag from GitHub API using `grep '"tag_name"' | sed` (no jq dependency)
  - Step 2: Downloads checksums.txt for the resolved tag to a `mktemp -d` temp dir with `trap ... EXIT` cleanup; asserts a `windows-amd64-installer.exe` entry exists
  - Step 3: Strips `v` prefix with `${TAG#v}` (Pitfall 7) and calls `packaging/winget/populate-manifests.sh "${VERSION}" "${CHECKSUMS_FILE}"`
  - Step 4: Validates every `*.yaml` in `packaging/winget/output/${VERSION}/` using `python3 -c "import yaml, glob; ..."` (no yamllint required)
  - Step 5: Asserts `PackageIdentifier: scottkw.agenthub`, `windows-amd64-installer.exe` URL, and version string are present in `scottkw.agenthub.installer.yaml`
  - Prints `=== PASS ===` on success; exits non-zero on any failure
  - Verified against the real v4.0 release: all 3 manifests parsed, all assertions pass

- **packaging/winget/FIRST-SUBMISSION-RUNBOOK.md** — 7-step operator procedure:
  1. Confirm release has Windows installer asset (checks the exact URL pattern distribute.yml constructs)
  2. Provision `WINGET_TOKEN` (classic PAT, `public_repo` scope only — T-156-07 least-privilege)
  3. Set `WINGET_FIRST_SUBMISSION=true` via `gh variable set`
  4. Trigger `distribute.yml` (tag push or `gh workflow run`)
  5. Monitor the `microsoft/winget-pkgs` PR and address review feedback
  6. Post-acceptance: reset `WINGET_FIRST_SUBMISSION` to false and remove `continue-on-error: true` from distribute.yml (T-156-08 repudiation mitigation)
  7. Verify `winget install scottkw.agenthub` on Windows
  - States explicitly at the top: steps 1–3 of M-26 are the phase gate; steps 4–7 are operator follow-ups, NOT phase-completion blockers

- **TESTING.md**: M-26 added in new Category O (WinGet First Submission Dry-Run, INSTALL-03):
  - Steps 1–3 are the phase-completion gate (dry-run pass, manifest assertion, WINGET_TOKEN provisioned)
  - Steps 4–6 are non-blockers (live submission, post-acceptance reset, Windows verify)
  - Explicit note: INSTALL-03 has no automated test file — verified by M-26 only
  - Suite count unchanged at 503; no Section 4 traceability row added (per INSTALL-03 design)

## Verification Results

- `bash packaging/winget/dry-run-first-submission.sh` — exits 0; v4.0 release resolved; checksums.txt downloaded; all 3 YAML manifests parsed; PackageIdentifier `scottkw.agenthub` and `windows-amd64-installer.exe` URL both present; PASS printed
- `bash tests/check-traceability-paths.sh` — exits 0 (all traceability paths exist)
- `test -f packaging/winget/FIRST-SUBMISSION-RUNBOOK.md` — file exists
- `grep -q 'M-26' TESTING.md` — M-26 present in Section 5
- TESTING.md Total count confirmed at 503 (unchanged)

## Deviations from Plan

None — plan executed exactly as written. The RESEARCH blueprint was followed verbatim for both the dry-run script and the operator runbook structure. All acceptance criteria met.

## Known Stubs

None. `dry-run-first-submission.sh` connects to the live GitHub API and executes `populate-manifests.sh` with real release data. The operator runbook documents the full live submission procedure. No placeholders or stubs remain that block the plan's goal.

## Threat Flags

No new threat surface beyond what is documented in the plan's threat model:
- T-156-07 mitigated: FIRST-SUBMISSION-RUNBOOK.md mandates `public_repo` scope only for WINGET_TOKEN; token referenced exclusively as `secrets.WINGET_TOKEN` (never echoed or printed)
- T-156-08 mitigated: Runbook Step 6b explicitly requires removing `continue-on-error: true` from distribute.yml after acceptance so future submission failures surface
- T-156-09 mitigated: dry-run-first-submission.sh asserts PackageIdentifier `scottkw.agenthub` and `windows-amd64-installer.exe` URL before printing PASS

No new network endpoints, auth paths, file access patterns, or schema changes introduced beyond those in the plan's trust boundary table.

## Self-Check

## Self-Check: PASSED

All files found on disk:
- FOUND: packaging/winget/dry-run-first-submission.sh
- FOUND: packaging/winget/FIRST-SUBMISSION-RUNBOOK.md
- FOUND: TESTING.md (M-26 present)
- FOUND: .planning/phases/156-install-links-distribution/156-03-SUMMARY.md

All commits verified in git log:
- FOUND: 9c0b33f0 (feat: dry-run helper)
- FOUND: ec3ad849 (docs: runbook + M-26)
