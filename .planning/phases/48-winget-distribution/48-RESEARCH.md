# Phase 48: WinGet Distribution - Research

**Researched:** 2026-04-04
**Domain:** Windows Package Manager (WinGet) — manual first submission + automated distribute.yml job
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DIST-03 | WinGet manifest submitted to microsoft/winget-pkgs (manual first submission, then automated via distribute.yml) | Manual submission process documented; winget-releaser action configuration verified; package identifier format confirmed; PAT requirements confirmed |

</phase_requirements>

---

## Summary

Phase 48 has two distinct workstreams that must happen in sequence: (1) a one-time manual submission to establish `scottkw.agenthub` as a known package in the microsoft/winget-pkgs community repository, and (2) adding a `winget-releaser` job to the existing `distribute.yml` so that future releases submit manifests automatically.

The manual submission requires Windows — `winget validate` and `SandboxTest.ps1` are Windows-only. The three-file manifest set (`scottkw.agenthub.yaml`, `scottkw.agenthub.installer.yaml`, `scottkw.agenthub.locale.en-US.yaml`) was created in Phase 47 and lives in `packaging/winget/manifests/`. It needs only two pre-submission edits: replace `{{VERSION}}` and `{{WINDOWS_SHA256}}` with the actual values from a published release, then run `winget validate`.

The automated leg uses `vedantmgoyal9/winget-releaser@main` — a thin wrapper around Komac that opens a PR to microsoft/winget-pkgs on every `release:published` event. The action requires a classic PAT (`public_repo` scope) stored as `WINGET_TOKEN`, a fork of microsoft/winget-pkgs under `scottkw`, and the package to already exist in the repository (hence the manual-first constraint).

**Primary recommendation:** Do the manual submission first (human action in Wave 1), then add the `winget-releaser` job to `distribute.yml` (automated code change in Wave 2), then verify with the next real release.

---

## Project Constraints (from CLAUDE.md)

| Directive | Impact on Phase |
|-----------|----------------|
| pnpm preferred for Node; go mod for Go | Not relevant — this phase is YAML + GitHub Actions only |
| GitHub Actions for CI/CD | distribute.yml additions must use Actions |
| Never install packages globally | Not relevant — no local installs in this phase |
| Classic PAT (not fine-grained) for winget-releaser | Confirmed by research: fine-grained PATs are unsupported |

---

## Standard Stack

### Core

| Tool / Action | Version | Purpose | Why Standard |
|---------------|---------|---------|--------------|
| `vedantmgoyal9/winget-releaser` | `@main` | Automates manifest PRs to microsoft/winget-pkgs | De-facto standard for GitHub-hosted apps; used by PowerShell/Win32-OpenSSH, MicaForEveryone, HidHide, many others |
| WinGet manifest schema | 1.12.0 | Three-file YAML format for package metadata | Already baked into Phase 47 manifests — no schema upgrade needed |
| `winget validate` | Built into winget CLI | Local manifest validation before PR | Required step before any manual submission |

### Supporting

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `winget-pkgs` fork (scottkw) | Target for winget-releaser PRs | Must be created before automated job can function |
| `SandboxTest.ps1` (in winget-pkgs repo) | Install test in Windows Sandbox | Optional but strongly recommended for first submission |
| `wingetcreate` | Alternative to manual YAML editing for first submission | Only needed if manifests need regeneration; Phase 47 templates are ready |

**Installation for automated job:**
No package installation — `winget-releaser` is a GitHub Action consumed directly in YAML.

---

## Architecture Patterns

### Pattern 1: Two-Phase Distribution (Manual-First, Then Automated)

**What:** WinGet requires a package identity to exist in microsoft/winget-pkgs before any automation can work. The first version must be submitted as a PR by a human.

**When to use:** Always — this is a hard constraint of the winget-pkgs ecosystem, not a design choice.

**Sequence:**
1. Fork `microsoft/winget-pkgs` under `scottkw` account
2. Populate manifests from Phase 47 templates with real version + SHA256
3. Run `winget validate` locally (Windows required)
4. Place files at `manifests/s/scottkw/agenthub/<version>/` in the fork
5. Open PR to microsoft/winget-pkgs master branch
6. Wait for Azure Pipeline automated validation (~30-40 min) + manual moderator review (hours to ~1 day)
7. After merge: add `winget-releaser` job to `distribute.yml`

### Pattern 2: winget-releaser Automated Submission

**What:** On every `release:published` event, `winget-releaser` generates updated manifests (using Komac internally) and opens a PR to microsoft/winget-pkgs from the `scottkw` fork.

**When to use:** All releases after the first manual submission.

**Example workflow addition to distribute.yml:**
```yaml
  submit-winget:
    runs-on: ubuntu-latest
    steps:
      - uses: vedantmgoyal9/winget-releaser@main
        with:
          identifier: scottkw.agenthub
          installers-regex: 'agenthub-v[\d.]+-windows-amd64-installer\.exe$'
          token: ${{ secrets.WINGET_TOKEN }}
```

**Key notes on the example above:**
- `identifier` must match the `PackageIdentifier` in the manifests exactly: `scottkw.agenthub`
- `installers-regex` must be restrictive enough to exclude the bare EXE (`agenthub-v*-windows-amd64.exe`) and match only the NSIS installer (`agenthub-v*-windows-amd64-installer.exe`). The default regex matches all `.exe` files, which would incorrectly include both files.
- `token` references `WINGET_TOKEN` (already decided as the secret name in STATE.md)
- `fork-user` defaults to the repository owner (`scottkw`) — correct for this project, no override needed
- The action strips the leading `v` from the release tag automatically (e.g., `v1.8.0` → `1.8.0`)
- The action runs on `ubuntu-latest` — cross-platform support is built into winget-releaser via Komac

### Manifest Folder Structure in winget-pkgs

```
manifests/
└── s/
    └── scottkw/
        └── agenthub/
            └── 1.8.0/
                ├── scottkw.agenthub.yaml
                ├── scottkw.agenthub.installer.yaml
                └── scottkw.agenthub.locale.en-US.yaml
```

The letter folder (`s`) is the lowercase first letter of the publisher name (`scottkw`). This path must match the `PackageIdentifier` and `PackageVersion` values in the YAML files.

### Pre-Submission Manifest Edits Required

The Phase 47 templates have two `{{PLACEHOLDER}}` tokens that must be replaced before submission:

| Token | Replace with | Source |
|-------|-------------|--------|
| `{{VERSION}}` (3 files) | `1.8.0` (bare version, no `v` prefix) | GitHub Release tag minus `v` |
| `{{WINDOWS_SHA256}}` (installer manifest) | actual SHA256 of NSIS installer | `checksums.txt` on the GitHub Release |

Also: the locale manifest has `License: Proprietary` which was noted in STATE.md as needing an update if a LICENSE file is added. Since no LICENSE file exists, `Proprietary` is acceptable for submission — WinGet accepts this value.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Manifest generation for future releases | Custom sed/template scripts | `winget-releaser` action | Komac handles architecture detection, URL matching, SHA256 computation, PR formatting, and API calls to fork |
| Manifest validation | Manual YAML inspection | `winget validate <path>` CLI | Catches schema errors, field format violations, and missing required fields before PR |
| Fork management | Manual fork sync | winget-releaser handles fork creation/sync | The action creates the fork PR automatically from the existing fork |

**Key insight:** The only hand-rolled work is the one-time manual PR to establish package identity. Everything after that is automated.

---

## Common Pitfalls

### Pitfall 1: Default `installers-regex` Matches Both EXE Artifacts

**What goes wrong:** The default regex `.(exe|msi|msix|appx)(bundle){0,1}$` matches any `.exe` file in the release. Phase 46 attaches two EXE files: `agenthub-v*-windows-amd64-installer.exe` (NSIS) and `agenthub-v*-windows-amd64.exe` (bare binary). winget-releaser would try to submit both as installers.

**Why it happens:** The default is designed for projects with a single installer binary.

**How to avoid:** Set `installers-regex: 'agenthub-v[\d.]+-windows-amd64-installer\.exe$'` to match only the NSIS installer. The `\.` escapes the dot; `$` anchors the end to prevent partial matches.

**Warning signs:** winget-releaser job fails with "multiple installers found" or the manifest PR has two installer entries.

### Pitfall 2: Forking microsoft/winget-pkgs Before First Submission

**What goes wrong:** winget-releaser needs `scottkw/winget-pkgs` to exist as a fork of `microsoft/winget-pkgs` before it can open PRs. If the fork doesn't exist, the action fails.

**Why it happens:** The action doesn't create forks — it only pushes branches to existing forks.

**How to avoid:** Fork `microsoft/winget-pkgs` under the `scottkw` account as a prerequisite step, before the automated job runs for the first time.

**Warning signs:** winget-releaser job fails with a 404 on `scottkw/winget-pkgs`.

### Pitfall 3: Version in Manifests Has `v` Prefix

**What goes wrong:** WinGet `PackageVersion` must not have a `v` prefix. The Phase 47 templates use `{{VERSION}}` as a placeholder. If filled in as `v1.8.0` instead of `1.8.0`, winget validate fails with a schema error.

**Why it happens:** GitHub release tags have the `v` prefix; WinGet versions don't.

**How to avoid:** Strip the `v` prefix when populating the template: use `VERSION_BARE="${GITHUB_REF_NAME#v}"` (same pattern used in release.yml).

**Warning signs:** `winget validate` error: `PackageVersion` does not match the expected pattern.

### Pitfall 4: Manifest Path in Fork Doesn't Match PackageIdentifier

**What goes wrong:** microsoft/winget-pkgs bot rejects PRs where the file path doesn't match the `PackageIdentifier` and `PackageVersion` in the YAML. The path must be exactly `manifests/s/scottkw/agenthub/1.8.0/`.

**Why it happens:** Bot enforces strict path-content consistency.

**How to avoid:** Verify the path letter is the lowercase first character of the publisher name in `PackageIdentifier`. `scottkw` → letter `s`.

**Warning signs:** PR gets `Manifest-Path-Error` label from the WinGet bot.

### Pitfall 5: Fine-Grained PAT Fails

**What goes wrong:** winget-releaser doesn't support fine-grained GitHub PATs. Using a fine-grained PAT as `WINGET_TOKEN` causes silent or cryptic auth failures.

**Why it happens:** The action requires classic PAT with `public_repo` scope.

**How to avoid:** Create a classic PAT (Settings → Developer settings → Personal access tokens → Tokens (classic)) with only `public_repo` scope.

**Warning signs:** winget-releaser job fails with permission denied or 403 when pushing to fork.

### Pitfall 6: Automated Job Runs Before Package Exists in Registry

**What goes wrong:** If a release is published before the manual first-submission PR is merged, the winget-releaser job will run successfully (it opens a PR to winget-pkgs) but the PR will be rejected because the package identity doesn't exist.

**Why it happens:** winget-releaser doesn't check for prior package existence before submitting.

**How to avoid:** Add the `winget-releaser` job to `distribute.yml` only after the manual first-submission PR is merged. Or gate it with a `workflow_dispatch` for the first run. Safest: add the job as a separate commit after confirming manual PR merge.

**Warning signs:** winget PR labeled `Blocking-Issue` with note that package doesn't exist.

---

## Code Examples

### Complete winget-releaser Job for distribute.yml

```yaml
# Source: vedantmgoyal9/winget-releaser README (github.com/marketplace/actions/winget-releaser)
  submit-winget:
    runs-on: ubuntu-latest
    steps:
      - uses: vedantmgoyal9/winget-releaser@main
        with:
          identifier: scottkw.agenthub
          installers-regex: 'agenthub-v[\d.]+-windows-amd64-installer\.exe$'
          token: ${{ secrets.WINGET_TOKEN }}
```

This job is added as a second job in the existing `distribute.yml` alongside `update-homebrew-tap`. It triggers on the same `release:published` event.

### Manifest Template Population (Manual Submission)

```bash
# Strip v prefix from release tag
VERSION_BARE="1.8.0"  # from GITHUB_REF_NAME=v1.8.0

# Extract Windows SHA256 from checksums.txt
WINDOWS_SHA256=$(grep "agenthub-v${VERSION_BARE}-windows-amd64-installer.exe" checksums.txt | awk '{print $1}')

# Populate templates (run from repo root)
for f in packaging/winget/manifests/*.yaml; do
  sed -e "s/{{VERSION}}/${VERSION_BARE}/g" \
      -e "s/{{WINDOWS_SHA256}}/${WINDOWS_SHA256}/g" \
      "$f" > "$(basename $f)"
done
```

### winget validate Command

```cmd
# Run on Windows with WinGet installed, from the directory containing the populated manifests
winget validate manifests\s\scottkw\agenthub\1.8.0
```

### Sparse Clone of microsoft/winget-pkgs Fork (Windows / Manual Submission)

```powershell
# Source: learn.microsoft.com/en-us/windows/package-manager/package/repository
git clone --filter=blob:none --no-checkout https://github.com/scottkw/winget-pkgs
cd winget-pkgs
git sparse-checkout set manifests\s\scottkw
git checkout
git checkout -b scottkw-agenthub-1.8.0
# Now copy populated manifests to manifests\s\scottkw\agenthub\1.8.0\
git add .
git commit -m "Add scottkw.agenthub version 1.8.0"
git push
# Then open PR to microsoft/winget-pkgs master
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| WinGet manifest v1 (single file) | Three-file split (version + installer + locale) | Schema 1.1+ | Phase 47 already uses schema 1.12.0 — no change needed |
| `winget-releaser@v2` pinned | `winget-releaser@main` for auto-updates | Ongoing | Use `@main` as recommended; Dependabot can pin to SHA if needed |

**No deprecated patterns apply:** The Phase 47 manifests are already at schema 1.12.0 which is current.

---

## Open Questions

1. **Does `scottkw.agenthub` conflict with an existing WinGet package identifier?**
   - What we know: No search results found any existing `scottkw.agenthub` entry; the `scottkw` publisher folder likely does not exist yet in microsoft/winget-pkgs
   - What's unclear: Cannot verify without searching microsoft/winget-pkgs directly on Windows or via the GitHub web UI
   - Recommendation: Verify with `winget search scottkw` (Windows) or browse `https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/scottkw` before submitting — if the path doesn't exist, the submission is safe

2. **Should `submit-winget` job run in parallel with `update-homebrew-tap` or sequentially?**
   - What we know: They are completely independent — different package managers, different targets
   - What's unclear: No ordering constraint exists
   - Recommendation: Run in parallel (separate jobs under the same `release:published` trigger) to minimize total workflow time

3. **License field: will `Proprietary` cause a WinGet policy rejection?**
   - What we know: STATE.md notes `License: Proprietary` as the current value; WinGet policies require accurate metadata; no LICENSE file exists in the repo
   - What's unclear: Whether WinGet moderators accept `Proprietary` or require an SPDX identifier; Microsoft's policy is focused on accurate description rather than requiring open-source
   - Recommendation: Submit with `Proprietary` as-is; this is an accurate description. If a moderator requests a change, it's a one-line edit.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `winget` CLI | `winget validate`, `winget install` (success criterion) | ✗ (macOS dev machine) | — | Windows-only; human action required on Windows machine |
| `wingetcreate` | Optional manifest generation helper | ✗ | — | Not needed — Phase 47 templates are ready |
| GitHub Actions (ubuntu-latest) | `winget-releaser` job | ✓ | Managed by GitHub | — |
| `scottkw/winget-pkgs` fork | winget-releaser automated PRs | Unknown — must be created | — | Fork must be created manually; no fallback |
| `WINGET_TOKEN` secret | winget-releaser job | Unknown — must be configured | — | Secret must be created; no fallback |

**Missing dependencies with no fallback:**
- `winget` CLI — Windows-only; `winget validate` and success criterion (SC1) require a Windows machine. This is a human-action checkpoint, not a blocker for the code changes.
- `scottkw/winget-pkgs` fork — must be created by user in GitHub UI before the automated job can run
- `WINGET_TOKEN` classic PAT — must be created by user and stored as repo secret

**Missing dependencies with fallback:**
- None

---

## Validation Architecture

The `nyquist_validation` key is absent from `.planning/config.json`, which means it is treated as enabled.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | None (YAML + GitHub Actions — no unit test framework applicable) |
| Config file | none |
| Quick run command | `winget validate packaging/winget/manifests/` (Windows only) |
| Full suite command | `powershell .\Tools\SandboxTest.ps1 manifests\s\scottkw\agenthub\<version>` (Windows Sandbox) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | Notes |
|--------|----------|-----------|-------------------|-------|
| DIST-03 | `winget install scottkw.agenthub` installs AgentHub | manual | `winget install scottkw.agenthub` | Windows-only; post-merge validation |
| DIST-03 | `winget validate` passes on manifests | manual | `winget validate <path>` | Windows-only; pre-submission gate |
| DIST-03 | winget-releaser job runs without error | smoke | Review GitHub Actions log after next release | Post-deployment verification |
| DIST-03 | `WINGET_TOKEN` secret configured correctly | smoke | winget-releaser job exit code | Fails fast if PAT missing or wrong scope |

### Sampling Rate

- **Per task commit:** n/a — no automated test runner; validate YAML syntax manually
- **Per wave merge:** `winget validate` (Windows) before opening manual PR
- **Phase gate:** SC1 (`winget install scottkw.agenthub` succeeds) — asynchronous; depends on Microsoft merging the PR

### Wave 0 Gaps

None — no test files to create. Validation is human-gated (winget CLI on Windows) and post-deployment (winget-releaser job log).

---

## Sources

### Primary (HIGH confidence)
- `vedantmgoyal9/winget-releaser` GitHub Marketplace page — all inputs, fork requirement, PAT requirement, version info
- `learn.microsoft.com/en-us/windows/package-manager/package/repository` — official manual submission process, folder structure, PR labels, validation pipeline detail
- Phase 47 SUMMARY files — established decisions: InstallerType nullsoft, License Proprietary, schema 1.12.0, template tokens, artifact URL pattern
- Phase 47 manifest files in `packaging/winget/manifests/` — confirmed token locations and current values

### Secondary (MEDIUM confidence)
- WebSearch: winget-pkgs PR review timeline (~30-40 min automated + hours to ~1 day for human review) — from community discussion threads

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — winget-releaser is the ecosystem standard; verified from official marketplace page
- Manual submission process: HIGH — verified from official Microsoft Learn docs
- Architecture patterns: HIGH — directly derived from Phase 47 deliverables and confirmed API docs
- Pitfalls: HIGH — all derived from documented constraints (regex default, fork requirement, PAT type) not speculation

**Research date:** 2026-04-04
**Valid until:** 2026-07-04 (stable ecosystem; WinGet schema and winget-releaser inputs change infrequently)
