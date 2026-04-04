# Phase 45: release-please + CI Signing Removal - Research

**Researched:** 2026-04-04
**Domain:** GitHub Actions release automation, release-please, CI workflow modification
**Confidence:** HIGH

## Summary

Phase 45 adds two new GitHub Actions workflows and modifies one existing workflow. The first new workflow (`release-please.yml`) calls `googleapis/release-please-action@v4` on every push to `main` to detect conventional commits and open/update a Release PR. The second workflow is deferred to Phase 46 (`release.yml`). The existing `build.yml` is stripped of its macOS signing and notarization steps — those steps move exclusively to `release.yml` in Phase 46 and stay scoped to the `release` environment where the 7 macOS secrets live.

The project's tags are `v1.0` through `v1.7` (simple short form, not `v1.0.0`). release-please requires bootstrapping: manually creating `release-please-config.json` and `.release-please-manifest.json` with the current version set to `1.7.0` so the next Release PR produces `v1.8.0`. The only version file that needs updating is `wails.json` (field `info.productVersion`), which supports JSON-path targeting in release-please extra-files config.

A critical environment secret placement issue was discovered: all 7 macOS signing secrets were stored in the `release` GitHub environment (not repo-level). The current `build.yml` references `secrets.MACOS_*` without declaring `environment: release`, meaning the env-vars resolve to empty strings (not secrets) — the signing steps are conditionally guarded by `env.MACOS_CERTIFICATE != ''` which happens to prevent failures, but it also means builds have never actually signed. Phase 45's signing removal from `build.yml` is correct regardless: remove the steps + env declarations entirely.

**Primary recommendation:** Use `release-type: simple` (not `go`) since the Go release-type's version-file handling was buggy until June 2025 (issue #2541) and simple provides identical behavior. Use extra-files with JSON-path to update `wails.json`. Token must be a PAT (not GITHUB_TOKEN) so the Release PR triggers `build.yml` CI checks.

<user_constraints>
## User Constraints (from STATE.md Accumulated Context)

### Locked Decisions
- Do NOT rely on `version-file` config in release-please — use `extra-files` with `// x-release-please-version` annotation due to Go type bug (issue #2541)
- macOS signing moves from `build.yml` to `release.yml` only — saves notarization quota on every PR

### Claude's Discretion
- release-please-action version pin (latest stable is v4.4.0)
- Whether to use `release-type: go` or `release-type: simple`
- bootstrap-sha value for changelog history start
- PAT token secret name in GitHub Actions

### Deferred Ideas (OUT OF SCOPE)
- Phase 46: release.yml multi-platform build pipeline
- Homebrew tap, WinGet distribution (Phases 47-48)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REL-01 | release-please.yml workflow creates Release PRs with auto-versioned CHANGELOG.md from conventional commits | Verified: googleapis/release-please-action@v4.4.0 supports this pattern; config + manifest bootstrap approach confirmed |
| REL-03 | Existing build.yml modified to remove macOS signing (moved to release-only), retaining tests and race detector | Verified: all 7 signing secrets are in `release` environment (not repo-level); build.yml signing steps + env block can be cleanly excised |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| googleapis/release-please-action | v4.4.0 | Creates Release PRs on conventional-commit pushes to main | Official maintained action from Google; v4 is current major version (released Oct 2025) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| actions/checkout | v4 | Not needed in release-please.yml (stateless) | Used in build.yml already |
| release-type: simple | built-in | Strategy that updates CHANGELOG.md and any extra-files | Use when no language-native version file exists |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| release-type: simple | release-type: go | go type had version-file bug (issue #2541, fixed June 2025); simple is functionally identical for this project — CHANGELOG.md + extra-files |
| PAT token | GITHUB_TOKEN | GITHUB_TOKEN cannot trigger downstream workflows; build.yml CI won't run on Release PRs |

**Installation:** No npm install. Action is referenced directly in workflow YAML.

**Version verification:** `googleapis/release-please-action@v4.4.0` confirmed as latest stable (released Oct 23, 2025).

## Architecture Patterns

### Files to Create
```
.github/workflows/
├── release-please.yml       # New: triggers on push to main
└── build.yml                # Modified: signing steps removed
release-please-config.json   # New: root config
.release-please-manifest.json # New: version tracking
```

### Pattern 1: release-please.yml Workflow

**What:** Minimal workflow that runs release-please-action on every push to main.
**When to use:** Always — this is the triggering mechanism for Release PRs.

```yaml
# Source: https://github.com/googleapis/release-please-action
name: release-please

on:
  push:
    branches:
      - main

permissions:
  contents: write
  pull-requests: write

jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          token: ${{ secrets.RELEASE_PLEASE_TOKEN }}
          config-file: release-please-config.json
          manifest-file: .release-please-manifest.json
```

**Why PAT not GITHUB_TOKEN:** GitHub blocks downstream workflow runs triggered by GITHUB_TOKEN. The Release PR needs `build.yml` to run CI checks before merging. A classic PAT with `repo` scope stored as repo secret `RELEASE_PLEASE_TOKEN` is required.

### Pattern 2: release-please-config.json for Simple Release Type

**What:** Configures the root package using `simple` release type with wails.json version tracking via JSON-path.
**When to use:** Single-package repo at root path.

```json
{
  "$schema": "https://raw.githubusercontent.com/googleapis/release-please/main/schemas/config.json",
  "release-type": "simple",
  "packages": {
    ".": {
      "extra-files": [
        {
          "type": "json",
          "path": "wails.json",
          "jsonpath": "$.info.productVersion"
        }
      ]
    }
  },
  "bootstrap-sha": "<sha of v1.7 tag commit>"
}
```

**Note on `bootstrap-sha`:** Set this to the commit SHA of the `v1.7` tag to prevent release-please from parsing the entire history back to v1.0. Only commits AFTER that SHA are included in the first Release PR's CHANGELOG.

### Pattern 3: .release-please-manifest.json Bootstrap

**What:** Seeds release-please with current version so next Release PR targets v1.8.0.

```json
{
  ".": "1.7.0"
}
```

**Critical:** Without this file, release-please doesn't know the current version and will produce incorrect SemVer. The `"."` key maps to the root package defined in config.

### Pattern 4: build.yml Signing Removal

**What:** Remove all macOS signing/notarization steps and the env block that references MACOS_* secrets.

**Remove entirely from build.yml:**
- The top-level `env:` block (7 MACOS_* env vars)
- `Import macOS certificate` step
- `Sign macOS app with hardened runtime` step
- `Notarize macOS app` step
- `Cleanup macOS keychain and certificate` step

**Keep unchanged:**
- All matrix entries (darwin/universal, linux/amd64, ubuntu-22.04, windows/amd64)
- `Run Go tests (all platforms, race detector)` step
- `Run build script tests` step
- `Build with Wails` step
- All `Upload *artifact` steps

**Note:** No `environment: release` declaration is needed in build.yml after removing signing — the secrets are environment-scoped to `release` and were never accessible to build.yml anyway (all env vars resolved to empty strings, which is why the `env.MACOS_CERTIFICATE != ''` condition silently skipped signing on every PR build).

### Anti-Patterns to Avoid
- **Using GITHUB_TOKEN for release-please token:** CI won't run on Release PRs.
- **Setting `environment: release` in release-please.yml:** release-please doesn't sign; environment gates are for release.yml only.
- **Using `release-type: go` without verifying the fix is available:** The fix in PR #2542 was merged June 2025 and release-please-action@v4.3.0 (Aug 2025) bundles release-please 17.1.2 which includes the fix. Safe to use if desired, but `simple` is equally valid and more predictable.
- **Omitting `bootstrap-sha`:** release-please will scan entire history and generate a massive CHANGELOG.
- **Using short tag format in manifest:** The manifest stores bare semver (e.g., `"1.7.0"` not `"v1.7.0"`). The action prepends `v` when creating tags.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parsing conventional commits | Custom git log parsing | release-please-action | Commit type mapping (feat→minor, fix→patch, BREAKING CHANGE→major) is non-trivial; footers/trailers require careful parsing |
| Generating CHANGELOG.md | sed/awk on git log | release-please-action | Grouping, deduplication, PR links, issue links |
| Version bumping wails.json | Custom jq script in workflow | release-please extra-files json type | jsonpath targeting handles nested fields cleanly |
| Creating GitHub Release | gh CLI in a script | release-please-action output + Phase 46 release.yml | release-please handles the PR → tag → release lifecycle |

**Key insight:** release-please is a state machine that tracks version across PRs. Hand-rolling any part breaks the state machine (e.g., a manual version bump bypasses manifest, causing the next Release PR to compute wrong semver).

## Common Pitfalls

### Pitfall 1: GITHUB_TOKEN Blocks CI on Release PRs
**What goes wrong:** The Release PR created by release-please never shows green CI — build.yml doesn't run on it.
**Why it happens:** GitHub's security model prevents GITHUB_TOKEN from triggering workflow runs to avoid recursive loops.
**How to avoid:** Use a classic PAT with `repo` scope as `RELEASE_PLEASE_TOKEN` repo secret.
**Warning signs:** Release PRs show no CI status checks.

### Pitfall 2: Missing .release-please-manifest.json Causes Wrong SemVer
**What goes wrong:** First Release PR proposes v1.0.0 or an unexpected version.
**Why it happens:** release-please starts from version 0.0.0 if it cannot find a manifest entry.
**How to avoid:** Commit `.release-please-manifest.json` with `{"." : "1.7.0"}` before the workflow runs for the first time.
**Warning signs:** Release PR title says `chore(main): release 1.0.0` or `1.0.1`.

### Pitfall 3: macOS Secrets Are Environment-Scoped, Not Repo-Scoped
**What goes wrong:** Any workflow that references `secrets.MACOS_*` without declaring `environment: release` receives empty strings.
**Why it happens:** Secrets were configured in the `release` GitHub environment in Phase 44.
**How to avoid:** build.yml must NOT include `environment: release` (that's for release.yml only). Remove the env block entirely from build.yml.
**Warning signs:** MACOS_CERTIFICATE env var is empty in build logs — this is actually already the case and is the desired end state.

### Pitfall 4: Commit SHA for bootstrap-sha Must Be the v1.7 Tag Commit
**What goes wrong:** Massive CHANGELOG generated from all historical commits, or changelog starts from wrong point.
**Why it happens:** bootstrap-sha is inclusive-exclusive (exclusive start boundary).
**How to avoid:** Set bootstrap-sha to the SHA of the v1.7 tag's target commit (`git rev-list -n 1 v1.7`).
**Warning signs:** First Release PR's CHANGELOG includes items from v1.0 era.

### Pitfall 5: "Allow GitHub Actions to create and approve pull requests" Must Be Enabled
**What goes wrong:** release-please-action fails with 403 error when attempting to create/update the Release PR.
**Why it happens:** GitHub repository settings may have this option disabled by default.
**How to avoid:** Enable in GitHub repo Settings → Actions → General → "Allow GitHub Actions to create and approve pull requests".
**Warning signs:** Workflow run fails with error about pull request creation.

## Code Examples

### Complete release-please.yml
```yaml
# Source: https://github.com/googleapis/release-please-action
name: release-please

on:
  push:
    branches:
      - main

permissions:
  contents: write
  pull-requests: write

jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        id: release
        with:
          token: ${{ secrets.RELEASE_PLEASE_TOKEN }}
          config-file: release-please-config.json
          manifest-file: .release-please-manifest.json
```

### release-please-config.json (simple release type with wails.json tracking)
```json
{
  "$schema": "https://raw.githubusercontent.com/googleapis/release-please/main/schemas/config.json",
  "release-type": "simple",
  "packages": {
    ".": {
      "extra-files": [
        {
          "type": "json",
          "path": "wails.json",
          "jsonpath": "$.info.productVersion"
        }
      ]
    }
  },
  "bootstrap-sha": "PLACEHOLDER_SHA"
}
```

### .release-please-manifest.json (bootstrap to current version)
```json
{
  ".": "1.7.0"
}
```

### Getting the bootstrap-sha
```bash
git rev-list -n 1 v1.7
```

### build.yml — Lines to Remove

**Remove the entire top-level `env:` block (lines 33-39):**
```yaml
    env:
      MACOS_CERTIFICATE: ${{ secrets.MACOS_CERTIFICATE }}
      MACOS_CERTIFICATE_NAME: ${{ secrets.MACOS_CERTIFICATE_NAME }}
      MACOS_CERTIFICATE_PWD: ${{ secrets.MACOS_CERTIFICATE_PWD }}
      MACOS_CI_KEYCHAIN_PWD: ${{ secrets.MACOS_CI_KEYCHAIN_PWD }}
      MACOS_NOTARIZATION_APPLE_ID: ${{ secrets.MACOS_NOTARIZATION_APPLE_ID }}
      MACOS_NOTARIZATION_PWD: ${{ secrets.MACOS_NOTARIZATION_PWD }}
      MACOS_NOTARIZATION_TEAM_ID: ${{ secrets.MACOS_NOTARIZATION_TEAM_ID }}
```

**Remove these 4 steps (lines 75-108):**
- `Import macOS certificate` (if: runner.os == 'macOS' && env.MACOS_CERTIFICATE != '')
- `Sign macOS app with hardened runtime` (if: runner.os == 'macOS' && env.MACOS_CERTIFICATE != '')
- `Notarize macOS app` (if: runner.os == 'macOS' && env.MACOS_CERTIFICATE != '')
- `Cleanup macOS keychain and certificate` (if: runner.os == 'macOS' && env.MACOS_CERTIFICATE != '' && always())

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual CHANGELOG.md editing | release-please auto-generation from conventional commits | Ongoing | Zero manual release prep |
| Triggering downstream workflows from GITHUB_TOKEN | PAT-based token for release-please | GitHub security model | Must explicitly provide PAT |
| release-type: go with version-file | Works correctly as of release-please 17.x (merged June 2025) | June 2025 | Can use go type now, but simple is equally valid |
| release-please-action v3 (archived) | release-please-action v4 (googleapis repo) | 2024 | Old google-github-actions/release-please-action is archived; use googleapis/release-please-action@v4 |

**Deprecated/outdated:**
- `google-github-actions/release-please-action`: Archived, do not use. Use `googleapis/release-please-action@v4`.
- `release-type: node` for non-Node projects: Only relevant if package.json version tracking is needed.

## Open Questions

1. **Does the RELEASE_PLEASE_TOKEN PAT already exist as a GitHub secret?**
   - What we know: The `gh api repos/scottkw/agenthub/actions/secrets` call returned 0 secrets (empty list). Environment secrets are in `release` environment.
   - What's unclear: Whether a PAT for release-please was created in Phase 44 or is expected to be created in Phase 45.
   - Recommendation: Phase 45 plan must include a step to create a classic PAT (repo scope) and add it as a repo-level secret `RELEASE_PLEASE_TOKEN`. This is a human action (PAT creation requires GitHub web UI).

2. **Does GitHub repo have "Allow GitHub Actions to create and approve pull requests" enabled?**
   - What we know: `gh api repos/scottkw/agenthub/actions/permissions` shows `"enabled": true, "allowed_actions": "all"` but this is about which actions can run, not PR creation permission.
   - What's unclear: The separate "Allow GitHub Actions to create and approve pull requests" setting is not exposed via the Actions permissions API in the same call.
   - Recommendation: Plan must include verifying/enabling this setting in GitHub repo Settings → Actions → General.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| gh CLI | Creating RELEASE_PLEASE_TOKEN secret | ✓ | 2.89.0 | — |
| GitHub repo scottkw/agenthub | All CI | ✓ | — | — |
| release environment on GitHub | Scoped to Phase 46 (signing) | ✓ | — | — |
| googleapis/release-please-action | release-please.yml | ✓ (public action) | v4.4.0 | — |
| Classic PAT (RELEASE_PLEASE_TOKEN) | release-please.yml token | ✗ (not yet created) | — | Cannot use GITHUB_TOKEN — must create PAT |

**Missing dependencies with no fallback:**
- `RELEASE_PLEASE_TOKEN` repo secret: Must be created as a classic PAT with `repo` scope. Requires human action in GitHub web UI. Plan must include this step.

**Missing dependencies with fallback:**
- None additional.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | bash (tests/build-script.test.sh) + go test |
| Config file | none — invoked directly |
| Quick run command | `bash tests/build-script.test.sh` |
| Full suite command | `go test -race ./... && bash tests/build-script.test.sh` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REL-01 | release-please.yml exists and is valid YAML | smoke | `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release-please.yml'))"` | ❌ Wave 0 |
| REL-01 | release-please-config.json is valid JSON with required fields | smoke | `python3 -m json.tool release-please-config.json` | ❌ Wave 0 |
| REL-01 | .release-please-manifest.json is valid JSON with version 1.7.0 | smoke | `python3 -c "import json; d=json.load(open('.release-please-manifest.json')); assert d['.']=='1.7.0'"` | ❌ Wave 0 |
| REL-03 | build.yml contains no MACOS_* env references | smoke | `grep -c "MACOS_" .github/workflows/build.yml; test $? -eq 1` | ✅ (existing file, verify post-edit) |
| REL-03 | build.yml contains no codesign or notarytool calls | smoke | `grep -c "codesign\|notarytool" .github/workflows/build.yml; test $? -eq 1` | ✅ (existing file, verify post-edit) |
| REL-03 | Go race detector test still passes | unit | `go test -race ./...` | ✅ |
| REL-03 | Build script tests still pass | unit | `bash tests/build-script.test.sh` | ✅ |

### Sampling Rate
- **Per task commit:** `python3 -m json.tool release-please-config.json && python3 -m json.tool .release-please-manifest.json`
- **Per wave merge:** `go test -race ./... && bash tests/build-script.test.sh`
- **Phase gate:** Full suite green + Release PR appears on GitHub after a conventional-commit push to main

### Wave 0 Gaps
- [ ] No new test files needed — validation is structural (JSON/YAML lint) and runtime (CI trigger test)
- [ ] Smoke command for YAML validity requires PyYAML: `pip install pyyaml` or use `python3 -c "import json" && ruby -e "require 'yaml'"` as fallback
- [ ] End-to-end validation (does a Release PR actually appear?) requires a real push to main with a conventional commit — this is the Phase gate test, not automatable pre-push

*(Runtime test: push a `fix: test release-please` commit to main and verify Release PR appears within ~60 seconds.)*

## Sources

### Primary (HIGH confidence)
- [googleapis/release-please-action](https://github.com/googleapis/release-please-action) — action inputs, outputs, workflow permissions, GITHUB_TOKEN vs PAT behavior
- [googleapis/release-please-action Releases](https://github.com/googleapis/release-please-action/releases) — v4.4.0 confirmed latest (Oct 23, 2025)
- [Phase 44-02-SUMMARY.md](.planning/phases/44-git-migration-to-github/44-02-SUMMARY.md) — secrets are in `release` environment (not repo-level); confirmed via `gh api`

### Secondary (MEDIUM confidence)
- [release-please customizing.md](https://github.com/googleapis/release-please/blob/main/docs/customizing.md) — extra-files JSON type with jsonpath, annotation syntax
- [release-please manifest-releaser.md](https://github.com/googleapis/release-please/blob/main/docs/manifest-releaser.md) — bootstrap-sha, manifest format, initial version seeding

### Tertiary (LOW confidence)
- [issue #2541: version-file ignored for go release-type](https://github.com/googleapis/release-please/issues/2541) — referenced in STATE.md decisions; fixed in PR #2542 (June 2025). Using `simple` release-type sidesteps this entirely.
- [issue #1000: triggering subsequent actions without PAT](https://github.com/googleapis/release-please-action/issues/1000) — community discussion confirming PAT requirement for downstream workflow triggers

## Metadata

**Confidence breakdown:**
- Standard stack (release-please-action v4.4.0): HIGH — verified via official releases page
- Architecture (config/manifest structure): HIGH — verified via official docs + manifest-releaser.md
- Pitfalls (environment-scoped secrets discovery): HIGH — confirmed via `gh api` live inspection
- PAT requirement for downstream triggers: HIGH — documented in official action README
- wails.json JSON-path updater: MEDIUM — json type with jsonpath documented in customizing.md; not tested against wails.json specifically

**Research date:** 2026-04-04
**Valid until:** 2026-05-04 (release-please-action is stable; v4.x unlikely to break within 30 days)
