---
phase: 106-distribution-pipeline-followups
type: phase-summary
status: code-complete-secrets-pending
requirements: [DIST-01, DIST-02, DIST-03]
---

# Phase 106 Summary — Distribution Pipeline Followups

## What this phase delivered

All 3 deferred Phase 91 follow-ups land as workflow file changes. Code is complete; **user must configure two GitHub secrets/vars before the next release** to actually exercise the fixes.

| Req | Fix | File |
|-----|-----|------|
| DIST-01 | Switch `release.yml` upload step to use a PAT (`RELEASE_PUBLISH_TOKEN`) so `release.published` triggers `distribute.yml` automatically. Falls back to `GITHUB_TOKEN` if PAT not set. | `.github/workflows/release.yml:438-455` |
| DIST-02 | `distribute.yml` submit-winget step now reads `$env:RELEASE_TAG` (defined at workflow env-block, available for both `release.published` and `workflow_dispatch`). Empty-tag guard added. Old broken `${{ github.event.release.tag_name }}` interpolation removed. | `.github/workflows/distribute.yml:93-130` |
| DIST-03 | Submit-winget step now branches on `WINGET_FIRST_SUBMISSION` variable — `wingetcreate new` on first submission to microsoft/winget-pkgs, `wingetcreate update` for steady state. | `.github/workflows/distribute.yml:113-128` |

## User action items (REQUIRED before next release)

### 1. Create the RELEASE_PUBLISH_TOKEN secret (for DIST-01)

Open https://github.com/settings/personal-access-tokens?type=fine-grained → "Generate new token":

- **Token name**: `agenthub-release-publish`
- **Resource owner**: scottkw
- **Repository access**: Selected repositories → `scottkw/agenthub`
- **Permissions** (Repository):
  - `Contents`: Read and write
  - `Metadata`: Read-only (auto)
  - `Pull requests`: Read and write (optional — only if release-please needs it)
- **Expiration**: 1 year (set a calendar reminder to rotate)

Copy the generated token, then add to the repo:

```
gh secret set RELEASE_PUBLISH_TOKEN --body "<paste-token-here>" --repo scottkw/agenthub
```

OR via the web UI: repo Settings → Secrets and variables → Actions → New repository secret.

### 2. Set WINGET_FIRST_SUBMISSION=true (for DIST-03, one-time only)

For the **first** v3.3 release (or whichever release first submits to microsoft/winget-pkgs):

```
gh variable set WINGET_FIRST_SUBMISSION --body "true" --repo scottkw/agenthub
```

After microsoft/winget-pkgs accepts the package submission (typically within 24-48h after the bot review), UNSET this variable:

```
gh variable delete WINGET_FIRST_SUBMISSION --repo scottkw/agenthub
```

Subsequent releases will then use the `update` path automatically (steady state).

### 3. Verify on next release

After the next tagged release (`v3.3.0` or similar):

1. Confirm `release.yml` ran and published — should see the release on the GitHub releases page.
2. Confirm `distribute.yml` triggered automatically (NOT via manual `gh workflow run`).
3. Check the WinGet job's log — should show the correct tag, version, and installer URL (no double-dashes, no empty `$version`).
4. If first submission: confirm `wingetcreate new` ran; verify the PR landed in microsoft/winget-pkgs.
5. If steady-state: confirm `wingetcreate update` ran without errors.

## Deferred / out of scope

- GitHub App token migration (Phase 91 Option 2). Defer — PAT works fine. Revisit if multiple maintainers.
- Homebrew tap automation verification. Already shipped in v3.1, unchanged.
- Linux APT/RPM repo automation. Out of scope per milestone Anti-Goals.

## Absorbs deferred directory

`.planning/deferred/91-distribution-pipeline-followups/` can now be archived (deferred → done). The CONTEXT.md there gave the rationale for these 3 fixes; this phase implements them. Archive on milestone close.

## Status

**Phase status:** code-complete-secrets-pending
**Phase fully complete when:** user creates RELEASE_PUBLISH_TOKEN secret (one-time) and sets WINGET_FIRST_SUBMISSION=true (one-time, then removes after acceptance). Workflow files already deployed via this phase.
