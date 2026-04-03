# Stack Research

**Domain:** GitHub Distribution & CI/CD for Wails v2 Desktop App (AgentHub v1.8)
**Researched:** 2026-04-03
**Confidence:** HIGH (versions verified against official GitHub releases, Microsoft docs, and Homebrew docs as of research date)

---

## Scope

This file covers ONLY the new tooling needed for v1.8: release automation, Homebrew tap distribution, and WinGet manifest submission. The existing stack (Go/Wails v2, React, xterm.js, kardianos/service, native macOS cgo tray, etc.) is validated and unchanged.

---

## Recommended Stack

### GitHub Actions — Release Automation

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `googleapis/release-please-action` | `v4` (v4.4.0, Oct 2025) | Auto-generates release PRs from conventional commits; manages CHANGELOG.md; bumps version in manifest | Google-maintained, widely adopted, native `go` release-type, integrates directly with GitHub Releases via PR workflow |
| `softprops/action-gh-release` | `v2` (v2.6.1, Mar 2025) | Attaches build artifacts (DMG, zip, exe) to GitHub Releases on tag push | De-facto standard for release asset uploads; `actions/create-release` is deprecated — this is the replacement |
| `dAppServer/wails-build-action` | `@main` | Builds Wails app for all platforms — handles Go + Node setup and `wails build` invocation | Already used in existing `build.yml`; `@main` is required per repo README; specific version tags are stale |
| `actions/checkout` | `v4` | Repo checkout | Already in use in `build.yml`; current stable |
| `actions/setup-go` | `v5` | Go toolchain setup from `go.mod` | Already in use in `build.yml`; current stable |
| `actions/upload-artifact` | `v4` | Pass build outputs between jobs in a workflow | Already in use in `build.yml`; current stable |

**IMPORTANT — wails-build-action known issue:** As of 2025-02-20, Wails v2.10.0 is broken in the action. If v2.10.x causes issues, pass `wails-version: "v2.9.0"` to the action as a workaround. Monitor `dAppServer/wails-build-action` releases for resolution.

### GitHub Actions — macOS Signing, Notarization, and Packaging

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `create-dmg` (shell script) | v1.2.3 (Nov 2025), installed via `brew install create-dmg` | Wraps hdiutil to produce installer-style DMG with drag-to-Applications layout from notarized `.app` bundle | Standard macOS CI DMG tool; handles `hdiutil` resource-busy retries; integrates cleanly after `xcrun stapler staple` |
| macOS `codesign` + `xcrun notarytool` + `xcrun stapler` | Built into macOS runners | Sign with hardened runtime, submit for notarization, staple ticket | Already fully implemented in `build.yml` — copy steps verbatim into `release.yml` |
| `ditto` | Built into macOS runners | Create notarization submission zip preserving macOS extended attributes | Already used in `build.yml` — `ditto -c -k --keepParent` is required; plain `zip` loses xattrs |

### GitHub Actions — Homebrew Tap Automation

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `peter-evans/repository-dispatch` | `v3` | Trigger tap-update workflow in `scottkw/homebrew-agenthub` from `distribute.yml` | Clean cross-repo triggering via PAT; no GitHub App setup required for an owner-controlled tap |
| Bash + `shasum -a 256` | Built-in (all runners) | Compute SHA256 of downloaded DMG release asset | Zero external dependency; reproducible; correct on macOS and Linux runners |
| `git` CLI in workflow step | Built-in | Commit updated cask formula to tap repo and push | Direct push to `main` of owned tap repo is appropriate; no PR needed for mechanical version bumps |

### GitHub Actions — WinGet Distribution

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `vedantmgoyal9/winget-releaser` | `@main` (v2 tag, Jan 2026) | Submits WinGet manifest PR to `microsoft/winget-pkgs` after each release | Uses Komac under the hood; cross-platform (runs on ubuntu-latest); auto-detects installer URLs from release assets; widely used and trusted by WinGet reviewers for faster approval |

---

## Workflow Architecture

Four workflow files. `build.yml` needs no changes.

```
.github/workflows/
  build.yml            EXISTING — PR/push CI, all platforms, race detector. No changes needed.
  release-please.yml   NEW — Runs on push to main. Opens/updates release PRs via conventional commits.
  release.yml          NEW — Runs on tag push (v*.*.*). Builds all platforms, signs/notarizes/packages macOS, uploads artifacts to GitHub Release.
  distribute.yml       NEW — Runs on release published event. Triggers Homebrew tap update + WinGet submission.
```

### release-please v4 Config Files

Two files must exist at repo root for manifest mode:

**`.release-please-manifest.json`** — tracks current version per path:
```json
{
  ".": "1.7.0"
}
```

**`release-please-config.json`** — package configuration:
```json
{
  "packages": {
    ".": {
      "release-type": "go",
      "changelog-path": "CHANGELOG.md"
    }
  }
}
```

**Conventional commit mapping** (automatic version bumps):
- `fix:` → patch bump (1.7.0 → 1.7.1)
- `feat:` → minor bump (1.7.0 → 1.8.0)
- `feat!:` or `BREAKING CHANGE:` in footer → major bump (1.7.0 → 2.0.0)

### Homebrew Cask Formula Structure

Cask file at `Casks/agenthub.rb` in `scottkw/homebrew-agenthub`:

```ruby
cask "agenthub" do
  version "1.8.0"
  sha256 "{{SHA256_OF_DMG}}"

  url "https://github.com/scottkw/agenthub/releases/download/v#{version}/AgentHub-#{version}.dmg"
  name "AgentHub"
  desc "Desktop app for running AI coding CLIs in tabbed terminal sessions"
  homepage "https://github.com/scottkw/agenthub"

  app "AgentHub.app"

  zap trash: [
    "~/Library/Application Support/AgentHub",
    "~/Library/Preferences/AgentHub.plist",
  ]
end
```

User installs with: `brew tap scottkw/agenthub && brew install --cask agenthub`

### WinGet Manifest Structure (3-file, schema 1.12.0)

Files go under `manifests/s/ScottKW/AgentHub/<version>/` in `microsoft/winget-pkgs` (submitted via `winget-releaser` action as a PR to the fork, then auto-submitted upstream).

**Version file** (`ScottKW.AgentHub.yaml`):
```yaml
PackageIdentifier: "ScottKW.AgentHub"
PackageVersion: "1.8.0"
DefaultLocale: "en-US"
ManifestType: "version"
ManifestVersion: "1.12.0"
```

**Default locale** (`ScottKW.AgentHub.locale.en-US.yaml`):
```yaml
PackageIdentifier: "ScottKW.AgentHub"
PackageVersion: "1.8.0"
PackageLocale: "en-US"
Publisher: "ScottKW"
PackageName: "AgentHub"
License: "MIT"
ShortDescription: "Desktop app for running AI coding CLIs in tabbed terminal sessions"
PackageUrl: "https://github.com/scottkw/agenthub"
ManifestType: "defaultLocale"
ManifestVersion: "1.12.0"
```

**Installer** (`ScottKW.AgentHub.installer.yaml`):
```yaml
PackageIdentifier: "ScottKW.AgentHub"
PackageVersion: "1.8.0"
InstallerType: "nullsoft"
Installers:
  - Architecture: "x64"
    InstallerUrl: "https://github.com/scottkw/agenthub/releases/download/v1.8.0/agenthub-amd64-installer.exe"
    InstallerSha256: "{{SHA256_OF_EXE}}"
ManifestType: "installer"
ManifestVersion: "1.12.0"
```

`InstallerType: "nullsoft"` is correct for NSIS-built executables (Wails builds with `nsis: true` produce NSIS installers). This enables silent install (`/S` flag) automatically.

### Packaging Templates in Source Repo

```
packaging/
  homebrew/
    agenthub.rb.tmpl     — Cask template with {{VERSION}} and {{SHA256}} placeholders for tap-update script
  winget/
    version.yaml         — WinGet version manifest template
    defaultLocale.yaml   — WinGet default locale manifest template
    installer.yaml       — WinGet installer manifest template
```

---

## GitHub Secrets Required

| Secret | Workflow | How to Obtain |
|--------|---------|---------------|
| `MACOS_CERTIFICATE` | `release.yml` | Base64-encoded Developer ID Application `.p12` — migrate from Gitea secrets |
| `MACOS_CERTIFICATE_NAME` | `release.yml` | `"Developer ID Application: Name (TEAMID)"` string |
| `MACOS_CERTIFICATE_PWD` | `release.yml` | `.p12` export password |
| `MACOS_CI_KEYCHAIN_PWD` | `release.yml` | Ephemeral keychain password (any value) |
| `MACOS_NOTARIZATION_APPLE_ID` | `release.yml` | Apple ID email |
| `MACOS_NOTARIZATION_PWD` | `release.yml` | App-specific password from appleid.apple.com |
| `MACOS_NOTARIZATION_TEAM_ID` | `release.yml` | 10-character Apple Team ID |
| `TAP_GITHUB_TOKEN` | `distribute.yml` | Classic PAT with `repo` + `workflow` scope for `scottkw/homebrew-agenthub` |
| `WINGET_TOKEN` | `distribute.yml` | Classic PAT with `public_repo` scope — fine-grained PATs are NOT supported by winget-releaser |
| `RELEASE_PLEASE_TOKEN` | `release-please.yml` | Classic PAT with `contents:write` + `pull-requests:write`, OR use `GITHUB_TOKEN` with `permissions: contents: write, pull-requests: write` in the workflow |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| **GoReleaser** (`goreleaser/goreleaser-action`) | Designed for pure Go CLI binaries — has no awareness of Wails `.app` bundles, macOS signing entitlements, or NSIS packaging. Will produce broken artifacts. | `dAppServer/wails-build-action@main` + custom signing steps already in `build.yml` |
| `google-github-actions/release-please-action` | Archived repo — the old home of the action before it moved to `googleapis/` | `googleapis/release-please-action@v4` |
| `actions/create-release` | Deprecated by GitHub | `softprops/action-gh-release@v2` |
| `macauley/action-homebrew-bump-cask` | Wraps `brew bump-cask-pr` which targets `homebrew-core` — submits PRs to the core tap, not to a private tap | Custom bash tap-update script in own tap workflow, triggered via `repository-dispatch` |
| Fine-grained PAT for `winget-releaser` | `vedantmgoyal9/winget-releaser` explicitly requires a classic PAT — fine-grained PATs are not supported (Komac API limitation) | Classic PAT stored as `WINGET_TOKEN` secret |
| `dAppServer/wails-build-action` pinned to a specific tag (e.g. `@v3`) | Repo README says use `@main`; v2.10.0 Wails compatibility warnings require main branch fixes | `@main` with optional `wails-version: "v2.9.0"` override |
| Submitting WinGet manifests manually for subsequent releases | Manual submission is only required for the **initial** package registration in `winget-pkgs` (first-ever version) | After first version is merged, use `winget-releaser` for all future releases |

---

## Version Compatibility

| Component | Compatible With | Notes |
|-----------|-----------------|-------|
| `release-please-action@v4` (v4.4.0) | `release-please` library v17.1.2 | Bundled together; no separate install needed |
| `softprops/action-gh-release@v2` (v2.6.1) | Any GitHub-hosted runner | No known compatibility constraints |
| `dAppServer/wails-build-action@main` | Wails v2 (NOT v2.10.0 — known broken) | Use `wails-version: "v2.9.0"` as fallback if needed |
| WinGet manifest schema 1.12.0 | `microsoft/winget-pkgs` current | Verified against Microsoft Learn docs updated 2026-03-24 |
| `vedantmgoyal9/winget-releaser@main` | ubuntu-latest runner recommended | Komac-based; cross-platform but Linux preferred |
| `create-dmg` v1.2.3 | macOS 12+ GitHub runners | Install via `brew install create-dmg` in workflow step |

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `googleapis/release-please-action@v4` | Manual `git tag` + `gh release create` | Only if the team cannot maintain conventional commit discipline |
| Custom bash tap-update script | `toolmantim/tap-release` GitHub App | tap-release is simpler but adds an external service dependency; bash script gives full control and no extra auth surface |
| `vedantmgoyal9/winget-releaser` | Manual PR via `wingetcreate` CLI | First-ever package registration MUST be manual — winget-releaser only works after initial version is accepted into winget-pkgs |
| `create-dmg` shell script | Bare `hdiutil` commands | hdiutil works but produces an unformatted DMG without drag-to-install layout; create-dmg adds the expected macOS installer UX |
| Multi-file WinGet manifest (3 YAML) | Singleton manifest (1 YAML) | Singleton is valid only for single-installer, single-locale packages; fine for initial submission, upgrade to multi-file format for richer metadata |

---

## Integration with Existing Build System

The existing `build.yml` works unchanged as the PR/push CI validator. The release pipeline builds on top of it:

1. **Same build action** — `release.yml` uses `dAppServer/wails-build-action@main` with identical platform matrix and flags.
2. **Reuse signing steps verbatim** — `import certificate`, `codesign`, `notarize`, and `cleanup` steps in `build.yml` are copied into `release.yml` without modification.
3. **Add DMG creation after staple** — After `xcrun stapler staple`, run `create-dmg` to wrap the notarized `.app` in a distributable DMG for Homebrew.
4. **NSIS installer already built** — `build.yml` already passes `nsis: true` for Windows builds; `release.yml` picks up `agenthub-amd64-installer.exe` as the WinGet artifact automatically.
5. **`build.sh` unchanged** — Local build script remains for developer use; CI workflows use the action directly and do not call `build.sh`.
6. **Artifact naming convention** — DMG should follow `AgentHub-{version}.dmg` and EXE `agenthub-amd64-installer.exe` for consistent URL patterns in Homebrew cask and WinGet manifest.

---

## Sources

- [googleapis/release-please-action releases](https://github.com/googleapis/release-please-action/releases) — v4.4.0 confirmed (Oct 23, 2025). HIGH confidence.
- [softprops/action-gh-release releases](https://github.com/softprops/action-gh-release/releases) — v2.6.1 confirmed (Mar 16, 2025). HIGH confidence.
- [dAppServer/wails-build-action README](https://github.com/dAppServer/wails-build-action) — `@main` usage confirmed; Wails v2.10.0 broken warning noted. MEDIUM confidence.
- [vedantmgoyal9/winget-releaser](https://github.com/vedantmgoyal9/winget-releaser) — v2 tag (Jan 2026), classic PAT requirement, `@main` usage confirmed. HIGH confidence.
- [create-dmg releases](https://github.com/create-dmg/create-dmg) — v1.2.3 (Nov 18, 2025). HIGH confidence.
- [Homebrew Cask Cookbook](https://docs.brew.sh/Cask-Cookbook) — cask formula stanzas verified (`version`, `sha256`, `url`, `name`, `desc`, `app`, `zap`). HIGH confidence.
- [Microsoft WinGet manifest docs](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest) — schema 1.12.0, three-file format, `nullsoft` InstallerType for NSIS. Updated 2026-03-24. HIGH confidence.
- [release-please-action GitHub Marketplace](https://github.com/marketplace/actions/release-please-action) — v4 config file format, Go release-type. HIGH confidence.
- [Automating Homebrew Tap Updates with GitHub Actions](https://builtfast.dev/blog/automating-homebrew-tap-updates-with-github-actions/) — two-repo pattern, SHA256 computation, sed-based formula update. MEDIUM confidence.

---
*Stack research for: GitHub Distribution & CI/CD — AgentHub v1.8*
*Researched: 2026-04-03*
