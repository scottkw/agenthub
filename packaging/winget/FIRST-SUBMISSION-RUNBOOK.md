# WinGet First Submission Runbook

> **Scope:** These steps are **operator follow-ups — not phase-completion blockers.**
> Phase 156 ships when the dry-run (TESTING.md M-26 steps 1–3) is verified.
> Steps 4–7 below are post-dry-run actions gated on Microsoft's external PR review.
>
> **Run the local pre-flight first:**
> ```bash
> bash packaging/winget/dry-run-first-submission.sh
> ```
> See TESTING.md M-26 for the full phase gate.

This runbook documents the live submission of `scottkw.agenthub` to the
[microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) catalog.

The repo-side automation is complete in `.github/workflows/distribute.yml`
(the `submit-winget` job, lines 74–131). This runbook covers:

- One-time operator setup (WINGET_TOKEN, WINGET_FIRST_SUBMISSION flag)
- Triggering the live submission
- Shepherding the Microsoft PR
- Post-acceptance reset (required to keep steady-state releases working)

---

## Step 1: Confirm the release has the Windows installer asset

Before triggering the submission, verify that the target release has the
Windows installer present in GitHub Releases:

1. Go to: https://github.com/scottkw/agenthub/releases
2. Find the target release. It **must** be a real (non-rc) tag — the
   `submit-winget` job skips rc tags:
   `if: ${{ !contains(github.ref, '-rc') }}` (`distribute.yml` line 77).
3. Confirm the asset `agenthub-v<VERSION>-windows-amd64-installer.exe` exists.

`distribute.yml` constructs the installer URL as:
```
https://github.com/scottkw/agenthub/releases/download/$tag/agenthub-$tag-windows-amd64-installer.exe
```
If this asset is absent, the `wingetcreate new` command will fail.
Verify the asset is uploaded **before** triggering the workflow.

---

## Step 2: Provision `WINGET_TOKEN` (if not already done)

The `submit-winget` job uses `${{ secrets.WINGET_TOKEN }}` (line 79 of
`distribute.yml`) to authenticate `wingetcreate` to open a fork PR on behalf
of an account that can fork `microsoft/winget-pkgs`.

**Token requirements (least privilege — T-156-07 mitigation):**

| Property | Value |
|----------|-------|
| Token type | Classic PAT (Personal Access Token) |
| Scope | **`public_repo` only** — no additional scopes |
| Account | Any GitHub account that can fork `microsoft/winget-pkgs` |

**Why `public_repo` only?** `wingetcreate` uses the token to fork
`microsoft/winget-pkgs` and open a PR from the fork. No private repo write
or admin scopes are required. Least-privilege limits blast radius if the
secret is exposed.

**Provision the secret:**
```bash
# Create a classic PAT at: https://github.com/settings/tokens
# Select scope: public_repo (only)
# Then store it as a GitHub Actions secret:
gh secret set WINGET_TOKEN --body "<YOUR_PAT>"
```

Verify it is stored:
```bash
gh secret list | grep WINGET_TOKEN
```

> **Security note:** The token is consumed exclusively as `${{ secrets.WINGET_TOKEN }}`
> (an environment variable for `wingetcreate`) — it is never echoed, printed,
> or interpolated into log lines. GitHub Actions automatically masks it in logs.

---

## Step 3: Set `WINGET_FIRST_SUBMISSION=true`

`distribute.yml` reads the `WINGET_FIRST_SUBMISSION` repo variable (line 108)
to choose between `wingetcreate new` (first submission) and `wingetcreate update`
(steady-state updates). Set this variable before the first live submission:

```bash
gh variable set WINGET_FIRST_SUBMISSION --body "true"
```

Verify it is set:
```bash
gh variable list | grep WINGET_FIRST_SUBMISSION
```

> **Important:** If this variable is absent or not `"true"`, the workflow defaults
> to `wingetcreate update scottkw.agenthub`, which **will fail** because the
> package does not exist in the catalog yet.

---

## Step 4: Trigger `distribute.yml`

Trigger the workflow using a real (non-rc) release tag. Two options:

**Option A — Push a real release tag** (normal release flow):
```bash
git tag v<VERSION>
git push origin v<VERSION>
```
The `release.published` event triggers `distribute.yml` automatically via
the `RELEASE_PUBLISH_TOKEN` secret. The `submit-winget` job reads the tag
from `github.ref_name`.

**Option B — Manual workflow dispatch** (if the release is already published):
```bash
gh workflow run distribute.yml -f tag=v<VERSION>
```

Either way, the `submit-winget` job will:
1. Download and SHA256-verify `wingetcreate.exe` (pinned to v1.12.8.0, line 80–82).
2. Run `wingetcreate new --urls <installer_url> --version <VERSION> --package-identifier scottkw.agenthub --submit`.
3. Open a PR to `microsoft/winget-pkgs` from the fork account.
4. The PR URL appears in the Actions job log.

---

## Step 5: Monitor the PR

1. Open the `submit-winget` Actions job log — the PR URL is printed there.
2. Go to the `microsoft/winget-pkgs` PR and monitor Microsoft's automated
   validation pipeline (usually completes within minutes to hours).
3. Microsoft bots run `winget validate` and verify the installer URL is reachable.
4. Address any manifest feedback requested by the reviewers.
5. Wait for the PR to be merged (typically 1–7 days, depending on queue length).

> **Note:** `continue-on-error: true` is currently set on the `submit-winget`
> job (`distribute.yml` line 76). This prevents the first-submission attempt
> from blocking future releases while the Microsoft review is pending.
> **It must be removed after the PR is merged (Step 6).**

---

## Step 6: After PR merge — reset to steady state

Once Microsoft merges the `microsoft/winget-pkgs` PR, immediately perform
these cleanup steps to restore normal release behavior:

### 6a. Reset `WINGET_FIRST_SUBMISSION`

```bash
# Set to false so steady-state releases use the update path:
gh variable set WINGET_FIRST_SUBMISSION --body "false"

# OR delete the variable entirely (workflow defaults to 'false' when absent):
gh variable delete WINGET_FIRST_SUBMISSION
```

### 6b. Remove `continue-on-error: true` from `distribute.yml`

Edit `.github/workflows/distribute.yml` — remove line 76 (`continue-on-error: true`
and its comment). This ensures future submission failures surface instead of
silently passing (T-156-08 repudiation mitigation):

**Before (current state):**
```yaml
  submit-winget:
    runs-on: windows-latest           # D-08: wingetcreate is .NET+Windows only
    continue-on-error: true           # WinGet first submission pending — remove after accepted
    if: ${{ !contains(github.ref, '-rc') }}   # D-17: skip rc tags
```

**After (post-acceptance):**
```yaml
  submit-winget:
    runs-on: windows-latest           # D-08: wingetcreate is .NET+Windows only
    if: ${{ !contains(github.ref, '-rc') }}   # D-17: skip rc tags
```

Commit the change:
```bash
git add .github/workflows/distribute.yml
git commit -m "chore: remove continue-on-error from submit-winget after first submission accepted"
git push origin main
```

---

## Step 7: Verify on Windows

After the catalog update propagates (usually within a few minutes of PR merge):

```powershell
winget install scottkw.agenthub
```

Confirm the installer downloads and completes without error. The `agenthub`
binary should be available after installation.

---

## Reference

| Resource | Path |
|----------|------|
| Local pre-flight script | `packaging/winget/dry-run-first-submission.sh` |
| Manifest templates | `packaging/winget/manifests/` |
| Workflow file | `.github/workflows/distribute.yml` (submit-winget, lines 74–131) |
| Phase gate | `TESTING.md` M-26 |
| Operator follow-ups | `.planning/STATE.md` — Operator Next Steps section |
