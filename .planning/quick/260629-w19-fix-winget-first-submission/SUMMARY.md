---
task: Fix WinGet first-submission so `winget install scottkw.agenthub` works
slug: 260629-w19-fix-winget-first-submission
date: 2026-06-29
status: complete
commits:
  - a352c352  # distribute.yml: submit prepared manifests instead of `wingetcreate new`
  - 07ae6ebb  # manifests: add yaml-language-server schema header
result: PR opened to microsoft/winget-pkgs — https://github.com/microsoft/winget-pkgs/pull/395007
---

# Summary — WinGet first submission fixed

`winget install scottkw.agenthub` did not work because the package had never
been created in Microsoft's catalog. The release pipeline's `submit-winget`
job had been failing silently (behind `continue-on-error: true`) on every
release.

## What was wrong (three layered defects, found by driving it live)
1. **`WINGET_FIRST_SUBMISSION` never set** → job ran `wingetcreate update` →
   failed: package not in catalog. (Fixed by setting the repo variable `true`.)
2. **`wingetcreate new` with invalid flags** → `new` is an interactive wizard
   that rejects `--urls/--version/--package-identifier/--submit` and can't
   supply Publisher/License non-interactively. The job also had **no checkout**,
   so the prepared manifest templates weren't even present.
3. **Manifest templates missing the schema header** → `wingetcreate submit`
   validation rejected all 3 with "Schema header not found".

## Fix
- `.github/workflows/distribute.yml` `submit-winget`: added `actions/checkout`;
  rewired the first-submission branch to download the release `checksums.txt`,
  run `populate-manifests.sh`, then `wingetcreate submit --prtitle … --no-open`
  (token via `WINGET_CREATE_GITHUB_TOKEN` env). Added `$LASTEXITCODE` guard.
- `packaging/winget/manifests/*.yaml`: added the required
  `# yaml-language-server: $schema=https://aka.ms/winget-manifest.<type>.1.12.0.schema.json`
  header to installer / defaultLocale / version manifests.

## Verified
- Local: YAML lint OK; `dry-run-first-submission.sh` populates + validates
  against live v4.1 (installer SHA256 matches release checksums).
- Live: `distribute.yml -f tag=v4.1` → `submit-winget` SUCCESS →
  `Manifest validation succeeded: True` →
  **PR https://github.com/microsoft/winget-pkgs/pull/395007 (OPEN)**.

## Operator follow-up (gated on Microsoft merging PR #395007 — external, ~1–7 days)
1. Monitor PR #395007 for Microsoft's automated validation; address any feedback.
2. After merge:
   - `gh variable set WINGET_FIRST_SUBMISSION -R scottkw/agenthub --body false`
     (or `gh variable delete`).
   - Remove `continue-on-error: true` from the `submit-winget` job so future
     submission failures surface.
   - Verify on Windows: `winget install scottkw.agenthub`.
3. Steady-state: future releases auto-run the `update` path (already correct).
