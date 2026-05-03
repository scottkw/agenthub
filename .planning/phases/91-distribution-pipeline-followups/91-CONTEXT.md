---
phase: 91-distribution-pipeline-followups
status: open
created: 2026-05-03
parent_milestone: v3.1.1 (or v3.2 — milestone TBD; see 91-PLAN-OUTLINE.md)
discovered_during: v3.1.0 ship (2026-05-03)
---

# Phase 91 Context: Distribution Pipeline Follow-ups

Three issues surfaced during the v3.1.0 ship that didn't block release but
need clean-up so the next ship is fully autonomous and propagation to all
distribution channels is reliable.

## Issue 91-A: release.yml uses GITHUB_TOKEN, distribute.yml doesn't auto-fire

**Symptom on v3.1.0:** When `release.yml` published the public GitHub release
(via `softprops/action-gh-release@v3.0.0`), the `release.published` event
should have triggered `distribute.yml`. It did not. We had to manually
`gh workflow run distribute.yml --field tag=v3.1.0` to publish the Homebrew
tap and (attempt) the WinGet submission.

**Root cause:** GitHub Actions safety policy. Per the docs:

> "When you use the repository's GITHUB_TOKEN to perform tasks, events
> triggered by the GITHUB_TOKEN will not create a new workflow run."

`release.yml` line 429 passes `token: ${{ secrets.GITHUB_TOKEN }}` to the
release-creation action, so the resulting `release.published` event is
muted from triggering downstream workflows.

**Why it worked for v2.2.x and v3.0:** Unknown — needs git archaeology.
Either an older mechanism, an older workflow file, or a momentary GitHub
behaviour that has since changed. The current `release.yml` (post Plan 04)
definitely uses `GITHUB_TOKEN`.

**Fix paths (pick one):**

1. **Switch to a PAT (Personal Access Token).** Create a fine-grained PAT
   with `contents: write` on `scottkw/agenthub`, store as
   `RELEASE_PUBLISH_TOKEN`, swap into `release.yml`'s `token:` line.
   Downstream `release.published` then fires `distribute.yml` automatically.
   - Pro: zero manual steps on future ships.
   - Con: PAT is a long-lived credential; needs rotation hygiene.

2. **Use a GitHub App token.** Generate a short-lived installation token
   from a GitHub App, swap into `release.yml`. Same trigger semantics as
   the PAT, no long-lived credential.
   - Pro: more secure than a PAT.
   - Con: more setup; need a GitHub App registered to the org.

3. **Document the manual dispatch step in the release runbook.** Don't
   change tokens. Just always run `gh workflow run distribute.yml` after
   each tag.
   - Pro: zero new infrastructure.
   - Con: every ship requires the operator to remember the manual step;
     easy to forget on a maintainer hand-off.

**Recommendation:** Option 1 (PAT) for now; revisit App-token if the
project gets contributors who shouldn't share the PAT.

## Issue 91-B: distribute.yml's submit-winget step uses event-only templating

**Symptom on v3.1.0:** Even though `distribute.yml` was dispatched correctly
(via workflow_dispatch with `tag=v3.1.0`), the `submit-winget` step
substituted empty strings:

```
$version = "".Trim('v')
$installerUrl = ".../agenthub--windows-amd64-installer.exe"  # double-dash, missing tag
```

**Root cause:** Lines 96-97 of `distribute.yml` use
`${{ github.event.release.tag_name }}` directly, which is only populated for
the `release: published` event. On `workflow_dispatch`, that field is empty.
The `update-homebrew-tap` job (lines 19-24) handles both events correctly
by reading the `RELEASE_TAG` env var (which combines `inputs.tag` with
`github.ref_name`).

**Fix:** Use the same `RELEASE_TAG` env var pattern in submit-winget:

```yaml
- name: Submit to WinGet
  shell: pwsh
  run: |
    $version = "${env:RELEASE_TAG}".TrimStart('v')
    $installerUrl = "https://github.com/scottkw/agenthub/releases/download/${env:RELEASE_TAG}/agenthub-${env:RELEASE_TAG}-windows-amd64-installer.exe"
    .\wingetcreate.exe ...
```

(In PowerShell, `${env:VAR}` is the proper syntax. Bash-style `$RELEASE_TAG`
won't expand.)

## Issue 91-C: WinGet first submission requires `wingetcreate new`, not `update`

**Symptom on v3.1.0:** Even after fixing 91-B locally, the underlying
`wingetcreate update scottkw.agenthub` would still fail with:

```
ERROR: repos/microsoft/winget-pkgs/contents/manifests/s/scottkw/agenthub
       was not found.
```

**Root cause:** AgentHub has never been submitted to `microsoft/winget-pkgs`.
`wingetcreate update` requires the package to already exist in the registry;
the first submission needs `wingetcreate new` (or a manual PR using the
template manifests in `packaging/winget/`).

**Known going in:** STATE.md flagged this as carried-from-v3.0:
> "WinGet first submission to microsoft/winget-pkgs deferred until first
> release is published."

**Fix path:** One-time manual submission, then the existing workflow's
`update` flow takes over for subsequent versions.

Steps:
1. Run `wingetcreate new --version 3.1.0 --urls <installer-url>` locally
   on a Windows machine (or via VM).
2. Review the generated manifest, ensure metadata matches
   `packaging/winget/manifests/s/scottkw/agenthub/3.1.0/`.
3. Submit the PR to `microsoft/winget-pkgs`.
4. Wait for Microsoft moderation (typically 1-3 days).
5. After acceptance, the `update` flow in `distribute.yml` works for v3.1.1
   and beyond.

Alternative: open the PR manually using the existing template manifests in
`packaging/winget/manifests/s/scottkw/agenthub/` — they were prepared in
v1.8 Phase 47 for exactly this moment.

## Out of scope (do NOT bundle here)

- Releasing v3.1.1 or v3.2 — separate milestone decision.
- Any v3.1 runtime regressions — those would be hotfixes, not Phase 91.
