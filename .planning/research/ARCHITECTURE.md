# Architecture Research

**Domain:** GitHub Distribution & CI/CD — Wails v2 Desktop App (AgentHub v1.8)
**Researched:** 2026-04-03
**Confidence:** HIGH — existing build.yml and build.sh inspected directly; CI/CD patterns verified via official docs and working external references

---

## System Overview

### End-to-End Distribution Pipeline

```
Developer commits (feat:/fix:/chore:) → main branch
    │
    ▼
release-please.yml listens on: push → main
    │  maintains Release PR
    │  when merged: bumps versions, creates git tag, creates GitHub Release
    ▼
git tag push (v1.8.0)
    │
    ├── triggers release.yml (on: push tags: ['v*.*.*'])
    │       │
    │       ├── macos-latest: wails build darwin/universal + sign + notarize
    │       │     → uploads agenthub-darwin-universal.zip to Release
    │       │
    │       ├── ubuntu-latest: wails build linux/amd64
    │       │     → uploads agenthub-linux-amd64.tar.gz to Release
    │       │
    │       └── windows-latest: wails build windows/amd64 + NSIS
    │             → uploads agenthub-windows-amd64-installer.exe + .exe to Release
    │
    └── GitHub Release published
            │
            triggers distribute.yml (on: release: types: [published])
                    │
                    ├── Homebrew job
                    │     triggers scottkw/homebrew-agenthub update workflow
                    │     (downloads .zip, sha256, updates cask .rb, commits)
                    │
                    └── WinGet job
                          winget-releaser reads .exe from release assets
                          opens PR at microsoft/winget-pkgs via scottkw fork
```

### Workflow File Responsibilities

| Workflow File | Trigger | Responsibility |
|---------------|---------|----------------|
| `build.yml` (existing, modified) | push, pull_request | PR validation: tests + race detector + build all platforms. **No signing. No release uploads.** |
| `release-please.yml` (new) | push to main | Parse conventional commits, maintain Release PR, create tag + GitHub Release on merge |
| `release.yml` (new) | `push: tags: ['v*.*.*']` | Multi-platform builds with macOS signing/notarization; upload artifacts to GitHub Release |
| `distribute.yml` (new) | `release: types: [published]` | Homebrew tap update + WinGet manifest submission |

---

## Recommended Project Structure

```
.github/
├── workflows/
│   ├── build.yml                 # MODIFY: strip signing steps (move to release.yml)
│   ├── release-please.yml        # NEW: conventional commit versioning
│   ├── release.yml               # NEW: tag-triggered release builds
│   └── distribute.yml            # NEW: package manager distribution
│
release-please-config.json        # NEW: release-please configuration
.release-please-manifest.json     # NEW: version tracking state (initially {"." : "1.7.0"})
CHANGELOG.md                      # NEW: generated and maintained by release-please
│
packaging/
├── homebrew/
│   └── agenthub.rb.tmpl          # NEW: cask formula template with placeholders
└── winget/
    └── manifests/                # NEW: initial WinGet manifest files (manual bootstrap)
        ├── AgentHub.AgentHub.installer.yaml
        ├── AgentHub.AgentHub.locale.en-US.yaml
        └── AgentHub.AgentHub.yaml
```

### Version Source Locations (Files That Must Stay in Sync)

The project currently has version values in three places. release-please updates them all via `extra-files` annotations:

| File | Current Value | Field | Annotation Needed |
|------|--------------|-------|-------------------|
| `wails.json` | `"productVersion": "1.0.0"` | `info.productVersion` | `// x-release-please-version` on that line |
| `frontend/src/components/WelcomeTab.tsx` | `const VERSION = '1.0.0'` | string literal | `// x-release-please-version` on that line |
| `CHANGELOG.md` | (new file) | entire file | release-please primary output |

The Go binary itself has no version constant — add a `Version` package-level var in `main.go` and inject via `-ldflags "-X main.Version=v1.8.0"` in `release.yml` at build time.

---

## Architectural Patterns

### Pattern 1: Separate build.yml from release.yml (Critical Separation)

**What:** `build.yml` handles PR validation (tests + builds, no signing). `release.yml` handles release builds (signing + asset upload), triggered only by tag push.

**Why this matters for AgentHub specifically:** The existing `build.yml` already runs macOS signing/notarization steps on every push when `MACOS_CERTIFICATE` is set. This means every PR merge calls Apple's notarization API — consuming rate limit budget, adding 10-15 minutes to every macOS run, and creating unnecessary noise in Apple's logs.

**What to change in build.yml:** Remove the following steps entirely (move equivalent logic to release.yml):
- "Import macOS certificate" step (lines 76-83)
- "Sign macOS app with hardened runtime" step (lines 85-91)
- "Notarize macOS app" step (lines 93-102)
- "Cleanup macOS keychain and certificate" step (lines 104-108)

The build matrix, wails-build-action invocations, Go test runs, and artifact uploads for PR inspection can remain in build.yml.

### Pattern 2: release-please for Automatic Versioning

**What:** `googleapis/release-please-action@v4` maintains a "Release PR" that accumulates unreleased changes. When the PR is merged, it bumps all annotated version strings, commits CHANGELOG.md, and creates a git tag + GitHub Release.

**Configuration for AgentHub (Go project with extra version files):**

```json
// release-please-config.json
{
  "release-type": "go",
  "packages": {
    ".": {
      "changelog-path": "CHANGELOG.md",
      "extra-files": [
        {
          "type": "generic",
          "path": "wails.json"
        },
        {
          "type": "generic",
          "path": "frontend/src/components/WelcomeTab.tsx"
        }
      ]
    }
  }
}
```

```json
// .release-please-manifest.json
{
  ".": "1.7.0"
}
```

Files referenced in `extra-files` must contain annotation markers so release-please knows which line to update:

```typescript
// WelcomeTab.tsx — add comment on the same line
const VERSION = '1.7.0' // x-release-please-version
```

```json
// wails.json — add comment after the value (JSON doesn't support comments natively,
// but release-please's generic strategy uses regex and will match the line)
// Alternative: use a VERSION file approach if wails.json comment syntax causes issues
"productVersion": "1.7.0", // x-release-please-version
```

**Known limitation:** release-please's `version-file` option for Go release-type has a reported bug (googleapis/release-please issue #2541 — version-file ignored for Go type). Use `extra-files` with `generic` type and annotation markers instead, which is the confirmed working approach.

**Required permissions in release-please.yml:**

```yaml
permissions:
  contents: write
  pull-requests: write
```

**Token note:** GitHub's `GITHUB_TOKEN` works for release-please's PR creation and tag push. The tag push fired by release-please correctly triggers downstream `push: tags:` workflows — this is a common misconception. The GITHUB_TOKEN restriction only prevents triggering *workflow_dispatch* events from within a workflow; a tag `push` event is a native Git event and fires regardless of token type.

### Pattern 3: Tag-Triggered Multi-Platform Release Builds

**What:** `release.yml` triggers on `push: tags: ['v*.*.*']`, runs the same 4-leg matrix as `build.yml` (darwin/universal, linux/amd64 on two Ubuntu versions, windows/amd64), but adds signing/notarization on macOS and uploads artifacts to the GitHub Release.

**Key difference from build.yml:** wails-build-action with `package: true` (default) automatically uploads artifacts to the GitHub Release when the workflow is triggered by a tag push.

**Version injection via ldflags:** The release tag is available as `github.ref_name` in the workflow. Pass it to the Go binary at build time:

```yaml
# In release.yml, for the wails-build-action step:
go-ldflags: "-X main.Version=${{ github.ref_name }}"
```

This requires adding `var Version string` to `main.go`. The CLI `--version` flag (or `agenthub version` command) should print this value.

**macOS signing flow (reuses existing secrets from build.yml):**

The signing steps are already proven in `build.yml`. They move to `release.yml` unchanged:
1. Decode `MACOS_CERTIFICATE` (base64) → `.p12`
2. Create isolated `build.keychain`, import certificate, set partition list
3. `wails-build-action` builds the `.app`
4. `codesign --deep --force --options runtime --timestamp --entitlements build/entitlements.plist`
5. `ditto -c -k --keepParent` → `.zip` for notarytool
6. `xcrun notarytool submit --wait` (blocks until Apple completes review, typically 2-5 min)
7. `xcrun stapler staple` → attaches ticket to `.app`
8. `security delete-keychain build.keychain` cleanup
9. Re-zip with `ditto` for release upload (the notarization `.zip` gets deleted in step 8's cleanup)

**Release asset naming convention** (important: Homebrew and WinGet depend on predictable names):

| Platform | Asset Name |
|----------|-----------|
| macOS | `agenthub-darwin-universal.zip` (signed .app inside) |
| Linux (ubuntu-latest) | `agenthub-linux-amd64.tar.gz` |
| Windows | `agenthub-windows-amd64-installer.exe` (NSIS) |
| Windows (portable) | `agenthub-windows-amd64.exe` |

### Pattern 4: Homebrew Tap via Cross-Repo Workflow Dispatch

**What:** After `distribute.yml` detects a published release, it uses the GitHub CLI to trigger a `workflow_dispatch` event in the `scottkw/homebrew-agenthub` repository. The tap repo's update workflow downloads the macOS release asset, computes SHA256, updates the cask `.rb` file, and commits.

**Tap repository structure (`scottkw/homebrew-agenthub`):**

```
homebrew-agenthub/
├── Casks/
│   └── agenthub.rb              # The live cask formula
└── .github/
    └── workflows/
        └── update-cask.yml      # Triggered by workflow_dispatch with version + sha256
```

**Naming:** The tap repository MUST be named `homebrew-agenthub` (the `homebrew-` prefix enables the shorthand `brew tap scottkw/agenthub`).

**User install commands:**
```bash
brew tap scottkw/agenthub
brew install --cask agenthub
# Or in one command:
brew install --cask scottkw/agenthub/agenthub
```

**Cask formula structure** (`Casks/agenthub.rb`):
```ruby
cask "agenthub" do
  version "1.8.0"
  sha256 "abc123..."

  url "https://github.com/scottkw/agenthub/releases/download/v#{version}/agenthub-darwin-universal.zip"
  name "AgentHub"
  desc "AI coding session manager"
  homepage "https://github.com/scottkw/agenthub"

  app "agenthub.app"

  zap trash: [
    "~/Library/Application Support/agenthub",
    "~/Library/Logs/agenthub",
  ]
end
```

**distribute.yml Homebrew job:**
```yaml
homebrew:
  runs-on: ubuntu-latest
  steps:
    - name: Compute macOS asset SHA256
      run: |
        URL="https://github.com/${{ github.repository }}/releases/download/${{ github.event.release.tag_name }}/agenthub-darwin-universal.zip"
        SHA256=$(curl -sL "$URL" | sha256sum | cut -d' ' -f1)
        echo "SHA256=$SHA256" >> "$GITHUB_ENV"

    - name: Trigger tap update
      run: |
        gh workflow run update-cask.yml \
          --repo scottkw/homebrew-agenthub \
          --field version="${{ github.event.release.tag_name }}" \
          --field sha256="${{ env.SHA256 }}"
      env:
        GH_TOKEN: ${{ secrets.TAP_DEPLOY_TOKEN }}
```

**Required secret:** `TAP_DEPLOY_TOKEN` — classic GitHub PAT with `repo` + `workflow` scopes (fine-grained tokens do not support cross-repo workflow_dispatch reliably).

### Pattern 5: WinGet Submission via winget-releaser Action

**What:** `vedantmgoyal9/winget-releaser@v2` (backed by Komac) reads the Windows `.exe` installer from the published release assets, generates the three required WinGet manifests, and opens a PR against `microsoft/winget-pkgs` via the `scottkw` fork.

**Prerequisites:**
1. Fork `microsoft/winget-pkgs` under the `scottkw` account (one-time)
2. Create a classic GitHub PAT with `public_repo` scope on the `scottkw` account
3. Store as `WINGET_TOKEN` repository secret
4. Submit the first package version manually to establish the package identifier in `winget-pkgs`

**Package identifier:** `AgentHub.AgentHub` (convention: `Publisher.AppName`, globally unique in the WinGet catalog)

**distribute.yml WinGet job:**
```yaml
winget:
  runs-on: ubuntu-latest
  steps:
    - uses: vedantmgoyal9/winget-releaser@v2
      with:
        identifier: AgentHub.AgentHub
        token: ${{ secrets.WINGET_TOKEN }}
        installers-regex: 'agenthub-windows-amd64-installer\.exe$'
```

**WinGet PR lifecycle:** After `winget-releaser` opens the PR, Microsoft's automated validation bot reviews it (typically 1-3 days for approved publishers, longer for new packages). First submission requires manual review of the package identity; subsequent releases are faster.

---

## Data Flow

### Full Release Data Flow (Tag to Package Manager)

```
1. Developer merges release-please Release PR
       │
       ▼
2. release-please commits:
   - CHANGELOG.md (new version section)
   - wails.json productVersion: "1.8.0"
   - WelcomeTab.tsx VERSION = '1.8.0'
   creates: git tag v1.8.0
   creates: GitHub Release (published, no assets yet)
       │
       ├── 3a. release.yml fires (push: tags: v1.8.0)
       │         macos-latest:
       │           wails build darwin/universal
       │           ldflags: -X main.Version=v1.8.0
       │           codesign + notarytool + stapler
       │           upload: agenthub-darwin-universal.zip → Release
       │
       │         ubuntu-latest:
       │           wails build linux/amd64
       │           tar.gz binary
       │           upload: agenthub-linux-amd64.tar.gz → Release
       │
       │         windows-latest:
       │           wails build windows/amd64 + NSIS
       │           upload: agenthub-windows-amd64-installer.exe → Release
       │           upload: agenthub-windows-amd64.exe → Release
       │
       └── 3b. distribute.yml fires (release: published)
                 Homebrew job:
                   download agenthub-darwin-universal.zip
                   sha256sum → SHA256
                   gh workflow run update-cask.yml \
                     --repo scottkw/homebrew-agenthub \
                     --field version=v1.8.0 \
                     --field sha256=SHA256
                   → tap repo commits updated agenthub.rb
                   → users: brew update && brew upgrade --cask agenthub

                 WinGet job:
                   winget-releaser reads agenthub-windows-amd64-installer.exe
                   generates AgentHub.AgentHub manifest files
                   opens PR against microsoft/winget-pkgs via scottkw fork
                   → after Microsoft review: winget upgrade AgentHub.AgentHub
```

**Timing note:** Step 3b (`distribute.yml`) fires at the moment the release is published. At that point, `release.yml` (3a) may still be running. The macOS build + notarization takes 15-25 minutes. Homebrew and WinGet both download release assets — if `distribute.yml` starts before the macOS asset is uploaded, the SHA256 download will fail.

**Mitigation options:**
- Make the Homebrew job in `distribute.yml` retry with exponential backoff
- Use `workflow_run` trigger to chain `distribute.yml` after `release.yml` completes (but this loses the `release.published` event context)
- Add a `wait-for-asset` step with `gh release view --json assets` polling

The recommended approach is retry with backoff in the Homebrew job, since `release.published` is the cleanest semantic trigger.

### Version String Flow

```
git tag v1.8.0
    │
    ├── release-please already updated in previous commit:
    │     wails.json: "productVersion": "1.8.0"
    │       → embedded in macOS .app Info.plist by Wails
    │       → embedded in Windows .exe version info by Wails
    │     WelcomeTab.tsx: VERSION = '1.8.0'
    │       → displayed in splash screen in-app
    │
    └── release.yml ldflags injection:
          go build -ldflags "-X main.Version=v1.8.0"
          → agenthub --version prints "v1.8.0"
          → agenthub version command output
```

---

## Integration Points

### Existing Components: Modify vs. Create New

| Component | Action | What Changes |
|-----------|--------|-------------|
| `.github/workflows/build.yml` | **MODIFY** | Remove the 4 macOS signing/notarization steps (lines 75-108). Keep everything else. |
| `build.sh` | **NO CHANGE** | release.yml uses wails-build-action directly. build.sh remains for local developer use. |
| `wails.json` | **MODIFY** | Add `// x-release-please-version` annotation so release-please can bump `productVersion` |
| `frontend/src/components/WelcomeTab.tsx` | **MODIFY** | Add `// x-release-please-version` comment on VERSION line. Also fix hardcoded `'1.0.0'` → `'1.7.0'` |
| `main.go` | **MODIFY** | Add `var Version string` package-level variable; wire to CLI `--version` flag |
| `go.mod` | **NO CHANGE** | Module path stays; release-please Go type reads it but does not modify go.mod for versioning |

### New Files Required

| File | Purpose |
|------|---------|
| `.github/workflows/release-please.yml` | Conventional commit parsing, release PR maintenance |
| `.github/workflows/release.yml` | Tag-triggered multi-platform builds with signing |
| `.github/workflows/distribute.yml` | Post-release Homebrew + WinGet distribution |
| `release-please-config.json` | release-please configuration for Go + extra-files |
| `.release-please-manifest.json` | Version state file (initial: `{"." : "1.7.0"}`) |
| `CHANGELOG.md` | Generated by release-please on first release PR |
| `packaging/homebrew/agenthub.rb.tmpl` | Cask formula template for reference/bootstrapping |
| `packaging/winget/manifests/*.yaml` | Initial WinGet manifest files (manually submitted) |

### New Repository Required

| Repo | Purpose | Required Before |
|------|---------|----------------|
| `scottkw/homebrew-agenthub` | Homebrew tap hosting cask formula | distribute.yml Homebrew job |

### Required Secrets (Repository Settings)

| Secret Name | Used By | Description |
|-------------|---------|-------------|
| `MACOS_CERTIFICATE` | `release.yml` | Base64-encoded .p12 developer certificate (already in build.yml) |
| `MACOS_CERTIFICATE_NAME` | `release.yml` | Certificate CN, e.g. "Developer ID Application: Name (TEAMID)" |
| `MACOS_CERTIFICATE_PWD` | `release.yml` | Certificate export password |
| `MACOS_CI_KEYCHAIN_PWD` | `release.yml` | Temporary keychain password for CI |
| `MACOS_NOTARIZATION_APPLE_ID` | `release.yml` | Apple ID email |
| `MACOS_NOTARIZATION_PWD` | `release.yml` | App-specific password from appleid.apple.com |
| `MACOS_NOTARIZATION_TEAM_ID` | `release.yml` | Apple Developer Team ID |
| `TAP_DEPLOY_TOKEN` | `distribute.yml` | Classic PAT (scottkw account), scopes: `repo` + `workflow` |
| `WINGET_TOKEN` | `distribute.yml` | Classic PAT (scottkw account), scope: `public_repo` |

All 7 macOS secrets already exist in the repository (confirmed in build.yml env block). Only `TAP_DEPLOY_TOKEN` and `WINGET_TOKEN` are new.

---

## Build Order for Phases

Dependencies between v1.8 components:

```
Phase A: release-please.yml + config files
  (no external deps; unblocks all downstream)
      │
      ▼
Phase B: release.yml (multi-platform release builds)
  (needs: Phase A for tag creation; existing macOS secrets)
  (also: modify build.yml to remove signing)
      │
      ▼
Phase C: packaging/ templates
  (can run parallel to Phase B once release asset names are confirmed)
      │
      ▼
Phase D: scottkw/homebrew-agenthub tap repo + distribute.yml Homebrew leg
  (needs: Phase B for confirmed asset names; Phase C for cask template)
  (needs: TAP_DEPLOY_TOKEN secret added)
      │
      ▼
Phase E: WinGet initial manifest + distribute.yml WinGet leg
  (needs: Phase B for Windows installer in release)
  (needs: manual first submission to winget-pkgs to establish package identity)
  (needs: WINGET_TOKEN secret added)
```

**Phase A is the only true blocker.** Phases C and D can be developed in parallel with Phase B once the asset naming is finalized. Phase E can proceed independently from Phase D.

---

## Anti-Patterns

### Anti-Pattern 1: Signing in build.yml (Current State — Must Fix)

**What the current code does:** `build.yml` runs macOS signing/notarization on every push when `MACOS_CERTIFICATE` is set. This means every PR merge triggers a full Apple notarization API call.

**Why it's wrong:** Apple's notarization has rate limits. Each submission takes 2-15 minutes. Every PR build on macOS becomes significantly slower. Notarization submissions for development commits create noise in Apple's developer portal.

**Fix:** Remove the 4 signing steps from `build.yml`. Add them to `release.yml` only. The unsigned `.app` artifact from `build.yml` is sufficient for PR validation.

### Anti-Pattern 2: Triggering distribute.yml on Tag Push Instead of Release Published

**What people do:** Use `push: tags: ['v*']` to trigger distribution workflows alongside `release.yml`.

**Why it's wrong:** Release assets are uploaded during `release.yml`, which takes 15-25 minutes (macOS notarization is the bottleneck). If `distribute.yml` starts on the tag push, the macOS `.zip` asset doesn't exist yet. Homebrew's SHA256 download fails.

**Do this instead:** Trigger `distribute.yml` on `release: types: [published]`. The release-please-created release is published immediately (not a draft), but asset upload happens during `release.yml`. Add retry logic to the Homebrew job to wait for the asset.

### Anti-Pattern 3: Using GITHUB_TOKEN for Cross-Repo Workflow Dispatch

**What people do:** Pass `${{ secrets.GITHUB_TOKEN }}` to `gh workflow run` targeting `scottkw/homebrew-agenthub`.

**Why it's wrong:** `GITHUB_TOKEN` is scoped to the current repository. It cannot trigger workflow_dispatch or push commits in other repositories.

**Do this instead:** Use a classic PAT (`TAP_DEPLOY_TOKEN`) with `repo` + `workflow` scopes stored as a repository secret. Fine-grained tokens can work for repo access but have inconsistent support for cross-repo workflow_dispatch.

### Anti-Pattern 4: Omitting `--wait` from notarytool Submit

**What people do:** Call `xcrun notarytool submit "$ZIP" --keychain-profile "..."` without `--wait`.

**Why it's wrong:** Without `--wait`, the command exits immediately with code 0 after queuing the submission. The GitHub Actions step passes, but the binary is never actually notarized. `xcrun stapler staple` then silently fails or attaches nothing. Users get Gatekeeper warnings on download.

**Do this instead:** Always use `--wait`. The existing `build.sh` already does this correctly (line 186). The same pattern must carry forward to `release.yml`.

### Anti-Pattern 5: Single-Version SOURCE in release-please (Go-type version-file)

**What people do:** Configure `"release-type": "go"` with `"version-file": "internal/version/version.go"` expecting release-please to bump it.

**Why it's wrong:** Known bug in release-please (issue #2541): the `version-file` configuration is ignored for Go release-type projects. The version file never gets updated.

**Do this instead:** Use `"extra-files"` with `"type": "generic"` and add `// x-release-please-version` annotation comments to each file. This is the confirmed working approach for files that aren't the primary `CHANGELOG.md`.

### Anti-Pattern 6: Hardcoded Package Identifier in WinGet Before First Manual Submission

**What people do:** Configure `winget-releaser` with an identifier like `AgentHub.AgentHub` before the package exists in the WinGet catalog.

**Why it's wrong:** The first submission to `microsoft/winget-pkgs` must be done manually (create the initial manifest files, submit a PR manually). After that PR is approved and the package identity is established, automated submissions via `winget-releaser` work correctly. Automating before the first approval means every release PR gets rejected as "unknown package."

**Do this instead:** Bootstrap `packaging/winget/manifests/` manually, submit the first PR to `winget-pkgs` by hand, wait for approval, then enable the `distribute.yml` WinGet job.

---

## Scaling Considerations

Not applicable for CI/CD infrastructure — this is a pipeline, not a service. The relevant capacity concern is GitHub Actions runner minutes:

| Runner | Minutes per release | Billed multiplier | Cost per release |
|--------|--------------------|--------------------|-----------------|
| macOS (release.yml) | ~20-25 min | 10x | 200-250 billed min |
| Linux (release.yml) | ~5-8 min | 1x | 5-8 billed min |
| Windows (release.yml) | ~8-12 min | 2x | 16-24 billed min |
| Linux (build.yml, PR) | ~5-8 min per platform | 1x | ~15-25 billed min per PR |

GitHub free tier provides 2,000 billed minutes/month. With 2-3 releases per month and ~20 PRs, the macOS signing cost is the dominant factor. Moving signing to `release.yml` only (away from every PR) is essential to stay within free tier.

---

## Sources

- `build.yml` direct inspection — confirmed existing signing steps, wails-build-action usage, matrix structure, secret names (HIGH confidence)
- `build.sh` direct inspection — confirmed `sign_and_notarize` function: ditto archive, notarytool --wait, stapler, cleanup (HIGH confidence)
- `wails.json` direct inspection — confirmed `productVersion: "1.0.0"` as version location (HIGH confidence)
- `WelcomeTab.tsx` direct inspection — confirmed hardcoded `VERSION = '1.0.0'` tech debt (HIGH confidence)
- [release-please GitHub repository](https://github.com/googleapis/release-please) — extra-files configuration, annotation marker syntax (HIGH confidence)
- [release-please issue #2541](https://github.com/googleapis/release-please/issues/2541) — version-file ignored for Go type; use extra-files instead (MEDIUM confidence — issue is open, workaround confirmed)
- [Automating Homebrew Tap Updates with GitHub Actions](https://builtfast.dev/blog/automating-homebrew-tap-updates-with-github-actions/) — cross-repo workflow_dispatch pattern, TAP_DEPLOY_TOKEN scope requirements (MEDIUM confidence)
- [WinGet Releaser Action](https://github.com/marketplace/actions/winget-releaser) — PAT scope (`public_repo`), fork requirement, Komac-backed implementation (HIGH confidence)
- [How to Create and Maintain a Tap — Homebrew Documentation](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap) — `homebrew-` naming convention, `Casks/` directory structure, install command syntax (HIGH confidence)
- [Automatic Code-signing and Notarization for macOS apps using GitHub Actions](https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/) — 7-secret pattern confirmed matches build.yml secrets (MEDIUM confidence)

---

*Architecture research for: AgentHub v1.8 — GitHub Distribution & CI/CD*
*Researched: 2026-04-03*
