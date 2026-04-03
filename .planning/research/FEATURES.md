# Feature Research

**Domain:** GitHub-based release automation, package manager distribution, and CI/CD for a Wails/Go cross-platform desktop app
**Researched:** 2026-04-03 (v1.8 GitHub Distribution & CI/CD)
**Confidence:** MEDIUM-HIGH (GitHub Actions patterns verified via official docs and community examples; WinGet review timeline from community discussions; Homebrew tap structure from official docs)

---

## v1.8 Milestone: GitHub Distribution & CI/CD

### Scope

This section covers only what is NEW in v1.8. The existing app ships: multi-platform Wails builds
(macOS/Linux/Windows), build.yml CI with race detection and macOS signing/notarization, build.sh for
local cross-compilation. Research focus: GitHub release automation (release-please, release.yml,
distribute.yml), Homebrew cask tap, WinGet submission, artifact naming conventions, changelog
generation, and packaging templates.

Existing signing infrastructure in build.yml (7 secrets: MACOS_CERTIFICATE, MACOS_CERTIFICATE_NAME,
MACOS_CERTIFICATE_PWD, MACOS_CI_KEYCHAIN_PWD, MACOS_NOTARIZATION_APPLE_ID, MACOS_NOTARIZATION_PWD,
MACOS_NOTARIZATION_TEAM_ID) is already implemented and does not need to be researched. The
reference implementation is scottkw/storcat with 4 workflows: build.yml, release-please.yml,
release.yml, distribute.yml.

Prior milestone research (v1.7) preserved below.

---

## Table Stakes (Users Expect These)

Features users assume exist in a professionally distributed cross-platform desktop app. Missing
these = product feels like an alpha or developer-only tool.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| GitHub Releases with binary artifacts | Every open-source desktop tool publishes versioned binaries to GitHub Releases; users expect to download directly without building from source | MEDIUM | release.yml triggered on tag push. Multi-platform matrix (macos-latest, ubuntu-latest, ubuntu-22.04, windows-latest) mirrors existing build.yml. Each platform produces its own artifact. macOS: .app inside .zip (ditto-preserved) or .dmg. Linux: .tar.gz. Windows: .exe installer + bare .exe. |
| Versioned release tags (SemVer) | Users expect v1.8.0, v1.8.1 etc. in GitHub Releases — random commit hashes are unusable for pinning | LOW | release-please automates this. Conventional commits (feat:, fix:, feat!:) drive SemVer bumps. Maintains CHANGELOG.md. Creates a release PR that must be merged to cut a release. |
| CHANGELOG.md auto-generated | Users and contributors expect a changelog; manual CHANGELOG maintenance is error-prone and typically skipped | LOW | release-please generates CHANGELOG.md from conventional commit messages. Each entry categorized as Features, Bug Fixes, etc. No manual maintenance required after setup. |
| macOS binary is codesigned and notarized | macOS 10.15+ Gatekeeper blocks unsigned .app from unverified developers; users see "can't be opened because Apple cannot check it for malicious software" | MEDIUM | Already implemented in build.yml. release.yml must replicate the same 7-secret signing+notarization pipeline. The existing pipeline uses xcrun notarytool with --wait for synchronous notarization. |
| Artifact SHA256 checksums published | Security-conscious users and package managers (Homebrew) require checksums to verify downloads haven't been tampered with | LOW | GitHub Actions: `shasum -a 256` or `sha256sum` per artifact. Upload checksums file to GitHub Release alongside binaries. Homebrew cask formula requires exact sha256 per architecture. |
| Consistent artifact naming | Package managers and automation scripts must parse artifact filenames; inconsistent naming breaks automation | LOW | Convention: `agenthub-{version}-{os}-{arch}.{ext}`. Examples: `agenthub-1.8.0-darwin-universal.zip`, `agenthub-1.8.0-linux-amd64.tar.gz`, `agenthub-1.8.0-windows-amd64-installer.exe`. Version without "v" prefix in filename is common (Homebrew uses bare version in URL). |

---

## Differentiators (Competitive Advantage)

Features that make distribution significantly better than the minimum viable "upload binaries to
GitHub Releases." Aligned with Core Value: reducing setup friction for users.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Homebrew cask tap (brew install --cask) | macOS power users expect `brew install --cask agenthub` — no download page, no drag-to-Applications; auto-update via `brew upgrade` | MEDIUM | Separate repo: scottkw/homebrew-agenthub. Cask file at Casks/scottkw-agenthub.rb (username prefix for uniqueness). distribute.yml in main repo triggers after release: fetches new SHA256, updates version + url in cask file, commits + pushes to tap repo. Requires PAT with repo scope for cross-repo write. brew tap scottkw/agenthub then brew install --cask scottkw/agenthub/scottkw-agenthub. |
| WinGet submission (winget install) | Windows power users use winget for CLI-first app management; presence in WinGet signals legitimacy | HIGH | vedantmgoyal9/winget-releaser action automates manifest creation and PR submission to microsoft/winget-pkgs. Requires forking microsoft/winget-pkgs under scottkw org (one-time setup) and classic PAT (fine-grained PATs not yet supported by the action). PR goes through automated validation (30-40min) + manual moderator review (hours to ~1 day). This is an async process — distribute.yml submits the PR; merge timing is Microsoft's. |
| Automated SHA256 computation in distribute.yml | Homebrew casks require exact sha256 per download URL; computing and committing this automatically eliminates the most error-prone step | MEDIUM | After release artifacts are uploaded, distribute.yml downloads each artifact and runs sha256sum/shasum. Injects computed hashes into cask template via sed or a Go/Python script. Commits updated cask to tap repo. This is the core value of distribute.yml. |
| Packaging templates in repo (packaging/) | packaging/homebrew/ and packaging/winget/ directories contain the templates used by distribute.yml — developers can inspect and modify distribution logic locally | LOW | packaging/homebrew/agenthub.rb.tmpl: cask template with VERSION and SHA256_DARWIN_UNIVERSAL placeholders. packaging/winget/: YAML manifest templates for version, installer, locale files. Templates live in source repo; distribute.yml renders them at release time. |
| Release PR workflow (release-please) | release-please creates a "Release PR" that shows exactly what will be in the next release before it ships; team can review changelog and version bump before cutting the release | LOW | googleapis/release-please-action@v4. Maintains a pending "Release: v1.8.0" PR. Each new conventional commit on main updates the PR. Merging the PR: updates CHANGELOG.md, bumps version, creates git tag, creates GitHub Release. No manual tagging required. |

---

## Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| GoReleaser for Wails builds | GoReleaser is the standard Go release tool and handles multi-platform builds natively | Wails apps require the `wails build` command with specific flags (`-tags wailsassets`, platform matrix, WebKit dependencies on Linux) — GoReleaser's standard Go build pipeline does not support this. Using GoReleaser would mean duplicating the Wails build configuration that already works in CI. | Keep the existing wails-build-action matrix. Use release-please for versioning + changelog only. Attach artifacts to GitHub Release manually in release.yml via `gh release upload`. |
| Homebrew core submission | Submit to homebrew/homebrew-cask for broader discoverability | Homebrew core requires apps to be "notable" with verifiable user base; new open-source apps are typically rejected during triage. The review process is slow and the bar is high. | Personal tap (scottkw/homebrew-agenthub) has no gatekeeping, deploys immediately, and provides identical install UX for users who know the tap name. Can submit to core after establishing user base. |
| NSIS installer for Windows in Homebrew/Scoop | Windows users expect installers via winget, Scoop, or Chocolatey | Scoop and Chocolatey are separate ecosystems, each requiring their own manifest format, submission PRs, and ongoing maintenance. Splitting effort across 3 Windows package managers dilutes maintenance. | WinGet is the Microsoft-native package manager with Windows 11 built in; prioritize WinGet for v1.8. Scoop/Chocolatey are future if there's user demand. |
| Automatic version injection into frontend | "Version number should appear in the app UI" | If version is injected at build time via ldflags, the WelcomeTab.tsx currently has a hardcoded version string (known tech debt in PROJECT.md). This is a separate concern from release automation. | Accept hardcoded version as existing tech debt. Address VERSION injection in a separate phase if prioritized. Release automation does not need to solve this. |
| Signing Windows binaries in release.yml | Windows code signing removes SmartScreen warnings | Windows EV code signing certificates cost $300-500+/year, require physical hardware token (not compatible with CI without workarounds), and the process is significantly more complex than macOS notarization. SmartScreen warnings do not block execution the same way Gatekeeper does. | Ship unsigned Windows binaries for v1.8. Users can right-click > "Run Anyway". Address Windows signing as a future milestone if user complaints emerge. |
| Automatic GitHub Pages changelog site | "Publish CHANGELOG.md as a website" | CHANGELOG.md generated by release-please is already readable on GitHub. A separate GitHub Pages site is additional infrastructure with minimal user value at this stage. | Link to GitHub Releases page in README. release-please GitHub Release notes are well-formatted and serve as the changelog display. |

---

## Feature Dependencies

```
[release-please.yml — version bump + tag + CHANGELOG]
    └──triggers──> [release.yml on tag push (v*)]
                       └──produces──> [GitHub Release with artifacts]
                                          └──triggers──> [distribute.yml on release published]
                                                             ├──updates──> [Homebrew tap repo]
                                                             └──submits──> [WinGet PR]

[Existing build.yml signing pipeline (7 secrets)]
    └──replicated in──> [release.yml for production releases]

[packaging/homebrew/agenthub.rb.tmpl]
    └──rendered by──> [distribute.yml to update tap repo]

[packaging/winget/*.yaml templates]
    └──rendered by──> [vedantmgoyal9/winget-releaser in distribute.yml]

[scottkw/homebrew-agenthub tap repo (separate GitHub repo)]
    └──written to by──> [distribute.yml via PAT]
    └──read from by──> [brew install --cask]

[microsoft/winget-pkgs fork under scottkw]
    └──required by──> [vedantmgoyal9/winget-releaser]
    └──PR submitted to microsoft/winget-pkgs]
```

### Dependency Notes

- **release-please must fire before release.yml:** release-please creates the git tag; release.yml triggers `on: push: tags: ["v*"]`. The PAT GITHUB_TOKEN limitation applies: release-please-created tags do NOT trigger downstream workflows unless a PAT (not GITHUB_TOKEN) is used in the release-please step. Requires a separate PAT secret stored as `RELEASE_PLEASE_TOKEN` or equivalent. This is the most common gotcha with release-please pipelines.
- **Signing secrets are already configured:** The 7 macOS signing secrets in build.yml already exist in the GitHub repo settings. release.yml can reference the same secrets with the same names.
- **distribute.yml requires additional secrets:** Cross-repo write access to homebrew-agenthub tap requires `HOMEBREW_TAP_TOKEN` (classic PAT, repo scope). WinGet submission requires `WINGET_TOKEN` (classic PAT, public_repo scope, fork of microsoft/winget-pkgs required).
- **WinGet submission is async:** distribute.yml submits a PR to microsoft/winget-pkgs. This PR undergoes automated validation (30-40min) and manual moderator review (hours to ~1 day). The workflow cannot wait for merge — it fires and forgets.
- **Homebrew sha256 is architecture-specific:** The cask formula requires separate sha256 values for each download URL. For darwin/universal there is one sha256. If Linux and Windows are added to the cask in future, each needs its own. For v1.8, cask covers macOS only.
- **NSIS installer already built by wails-build-action:** The Windows matrix in build.yml already produces `agenthub-amd64-installer.exe` via `nsis: true`. release.yml should upload both the installer and the bare `agenthub.exe` so users can choose.

---

## MVP Definition for v1.8

### Launch With (v1.8 — what makes distribution real)

Minimum viable to consider GitHub distribution complete.

- [ ] release-please.yml — automated versioning, CHANGELOG.md, and git tags from conventional commits
- [ ] release.yml — multi-platform builds on tag push with macOS signing/notarization, artifacts uploaded to GitHub Release, SHA256 checksums
- [ ] scottkw/homebrew-agenthub tap repo — with initial cask formula
- [ ] distribute.yml — auto-updates Homebrew tap on release
- [ ] packaging/homebrew/ and packaging/winget/ template directories in main repo
- [ ] WinGet manifest generation and PR submission in distribute.yml

### Add After Core Works (v1.8.x)

- [ ] Linux .deb or AppImage packaging — if user demand emerges from Linux community
- [ ] Scoop manifest — if Windows users request it
- [ ] VERSION injection into WelcomeTab.tsx — address tech debt if annoying in practice

### Future Consideration (v2+)

- [ ] Submission to homebrew/homebrew-cask (core) — after establishing user base
- [ ] Windows code signing — if SmartScreen friction becomes a blocker for user adoption
- [ ] Chocolatey package — niche Windows power user market, low ROI vs WinGet

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| GitHub Releases with binaries | HIGH | MEDIUM | P1 |
| release-please versioning + CHANGELOG | HIGH | LOW | P1 |
| macOS signing/notarization in release.yml | HIGH | LOW (reuse existing) | P1 |
| SHA256 checksums in release | MEDIUM | LOW | P1 |
| Artifact naming convention | HIGH | LOW | P1 |
| Homebrew cask tap + formula | HIGH | MEDIUM | P1 |
| distribute.yml auto-update tap | HIGH | MEDIUM | P1 |
| WinGet manifest + submission | MEDIUM | HIGH | P2 |
| packaging/ template directory | MEDIUM | LOW | P1 |
| PAT for release-please tag trigger | HIGH | LOW | P1 (blocker) |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

---

## Workflow Architecture

Four workflows that chain together:

### build.yml (existing — do not change)
- Trigger: `push` and `pull_request` on any branch
- Purpose: PR validation, race detection, smoke build
- Does NOT create releases or upload artifacts

### release-please.yml (new)
- Trigger: `push` to `main`
- Action: googleapis/release-please-action@v4
- Outputs: maintains "Release: v{version}" PR on main
- On PR merge: creates git tag `v{version}`, creates GitHub Release (empty), updates CHANGELOG.md
- Requirement: must use PAT token (not GITHUB_TOKEN) so created tags trigger release.yml

### release.yml (new)
- Trigger: `push: tags: ["v*"]`
- Matrix: same 4 legs as build.yml (macos-latest, ubuntu-latest, ubuntu-22.04, windows-latest)
- Steps: build, sign (macOS), compute SHA256, upload artifact to GitHub Release
- Artifact naming: `agenthub-{version}-{platform}-{arch}.{ext}`
- Uploads: artifacts + SHA256 checksums file

### distribute.yml (new)
- Trigger: `release: types: [published]`
- Steps:
  1. Download release artifacts to compute/verify SHA256
  2. Update Homebrew cask in scottkw/homebrew-agenthub tap (via PAT + git push)
  3. Submit WinGet manifests via vedantmgoyal9/winget-releaser
- Required new secrets: `HOMEBREW_TAP_TOKEN` (PAT, repo scope), `WINGET_TOKEN` (PAT, public_repo scope)

---

## Artifact Naming Convention

| Platform | Artifact | Format |
|----------|----------|--------|
| macOS universal | `agenthub-{version}-darwin-universal.zip` | .app bundle zipped with `ditto` (preserves xattrs for signing) |
| Linux amd64 (Ubuntu 24) | `agenthub-{version}-linux-amd64.tar.gz` | bare binary in tar.gz |
| Linux amd64 (Ubuntu 22) | `agenthub-{version}-linux-amd64-ubuntu22.tar.gz` | bare binary, older glibc compat |
| Windows amd64 installer | `agenthub-{version}-windows-amd64-installer.exe` | NSIS installer |
| Windows amd64 bare | `agenthub-{version}-windows-amd64.exe` | bare binary for users who prefer no installer |
| Checksums | `agenthub-{version}-checksums.txt` | sha256 one-line per artifact |

**Version format in filenames:** bare version without `v` prefix (e.g., `1.8.0` not `v1.8.0`). The git tag is `v1.8.0`; the filename strips the `v`. This is conventional and matches Homebrew's URL template interpolation.

---

## Homebrew Cask Template

Minimum viable cask for `packaging/homebrew/agenthub.rb.tmpl`:

```ruby
cask "scottkw-agenthub" do
  version "VERSION_PLACEHOLDER"

  on_arm do
    sha256 "SHA256_DARWIN_UNIVERSAL_PLACEHOLDER"
  end
  on_intel do
    sha256 "SHA256_DARWIN_UNIVERSAL_PLACEHOLDER"
  end

  url "https://github.com/scottkw/agenthub/releases/download/v#{version}/agenthub-#{version}-darwin-universal.zip",
      verified: "github.com/scottkw/agenthub/"

  name "AgentHub"
  desc "Desktop app for running AI coding CLIs in tabbed terminal sessions"
  homepage "https://github.com/scottkw/agenthub"

  livecheck do
    url :url
    strategy :github_latest
  end

  app "agenthub.app"
end
```

Notes:
- Universal binary means same sha256 for both `on_arm` and `on_intel` blocks
- `livecheck` block enables `brew livecheck` and auto-update infrastructure
- `verified:` is required when URL host differs from homepage host (GitHub releases URL vs homepage)
- Users install with: `brew tap scottkw/agenthub && brew install --cask scottkw-agenthub`

---

## WinGet Manifest Structure

Three YAML files required per version (schema 1.6.0 as of 2025):
- `scottkw.agenthub.installer.yaml` — installer URLs, SHA256, architecture
- `scottkw.agenthub.locale.en-US.yaml` — description, publisher, homepage, tags
- `scottkw.agenthub.yaml` — version file linking the above

The `vedantmgoyal9/winget-releaser` action generates these from the GitHub Release automatically given the `packageId` (`scottkw.agenthub`) and `token`. This eliminates the need to handcraft YAML templates for v1.8. The `packaging/winget/` directory can hold manually-crafted baseline manifests for reference/testing.

Prerequisite one-time setup:
1. Fork `microsoft/winget-pkgs` under the `scottkw` GitHub account
2. Create classic PAT with `public_repo` scope (fine-grained PATs not yet supported)
3. Store PAT as `WINGET_TOKEN` secret in scottkw/agenthub

---

## CI/CD Secrets Inventory

| Secret Name | Purpose | Already Exists | New for v1.8 |
|-------------|---------|----------------|--------------|
| MACOS_CERTIFICATE | Base64 encoded Developer ID .p12 | Yes (build.yml) | No |
| MACOS_CERTIFICATE_NAME | Certificate CN string | Yes | No |
| MACOS_CERTIFICATE_PWD | .p12 export password | Yes | No |
| MACOS_CI_KEYCHAIN_PWD | Temp keychain password | Yes | No |
| MACOS_NOTARIZATION_APPLE_ID | Apple ID email | Yes | No |
| MACOS_NOTARIZATION_PWD | App-specific password | Yes | No |
| MACOS_NOTARIZATION_TEAM_ID | Apple Developer Team ID | Yes | No |
| RELEASE_PLEASE_TOKEN | PAT for release-please tag creation | No | Yes — needed so tags trigger release.yml |
| HOMEBREW_TAP_TOKEN | PAT with repo scope for tap repo write | No | Yes |
| WINGET_TOKEN | Classic PAT with public_repo scope for winget-pkgs fork | No | Yes |

---

## Phase-Specific Notes

### Phase: release-please.yml

Low-risk, low-complexity. The googleapis/release-please-action@v4 is well-documented and used by
thousands of projects. The critical nuance is the PAT token requirement: GitHub's security model
prevents GITHUB_TOKEN from triggering downstream workflow runs, so release-please must use a PAT
to create tags that will fire release.yml.

Configuration file `release-please-config.json` and `.release-please-manifest.json` are required
at repo root. Package type is `simple` (not `go` — the go type targets Go module version files,
not the app version).

### Phase: release.yml

Medium complexity due to macOS signing pipeline. The signing steps are already proven in build.yml
— they just need to be replicated with one important difference: the build output must be packaged
as a downloadable artifact (zip/tar.gz) before upload. For macOS, `ditto -c -k --keepParent` is
the correct archiving method (already used for notarization — can reuse the same zip). For Linux,
`tar czf`. For Windows, upload NSIS installer directly (already .exe).

The `gh release upload` CLI command or `actions/upload-release-asset` uploads artifacts to the
GitHub Release created by release-please.

### Phase: Homebrew Tap

Two-repo setup. scottkw/homebrew-agenthub is a simple repo with a `Casks/` directory. The
`distribute.yml` in the main repo uses `sed` or a Python one-liner to replace VERSION and SHA256
placeholders in the template, then commits and pushes via git in the CI runner (using the PAT for
auth). No separate CI on the tap repo required — it's a data repo, not a code repo.

### Phase: WinGet

The async nature of WinGet submission is the most important characteristic. `distribute.yml`
submits the PR using `vedantmgoyal9/winget-releaser@v2`. The job succeeds when the PR is
submitted, not when it is merged. WinGet moderation is typically hours to <1 day. The package will
not be installable via `winget install scottkw.agenthub` until Microsoft merges the PR. This is
not a blocker — it is just expected timeline.

---

## Sources

- release-please-action GitHub: https://github.com/googleapis/release-please-action (MEDIUM confidence — official project)
- release-please GITHUB_TOKEN limitation: https://github.com/googleapis/release-please-action/issues/1000
- Homebrew cask tap structure: https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap (HIGH confidence — official docs)
- Homebrew cask cookbook: https://docs.brew.sh/Cask-Cookbook (HIGH confidence — official docs)
- macauley/action-homebrew-bump-cask: https://github.com/macauley/action-homebrew-bump-cask
- Automating Homebrew tap updates: https://builtfast.dev/blog/automating-homebrew-tap-updates-with-github-actions/ (MEDIUM confidence)
- vedantmgoyal9/winget-releaser: https://github.com/vedantmgoyal9/winget-releaser (MEDIUM confidence — widely used action)
- WinGet PR review timeline: https://github.com/microsoft/winget-pkgs/discussions/19502 (MEDIUM confidence — community discussion)
- WinGet manifest structure: https://learn.microsoft.com/en-us/windows/package-manager/package/manifest (HIGH confidence — official docs)
- Wails cross-platform build with GitHub Actions: https://wails.io/docs/next/guides/crossplatform-build/ (HIGH confidence — official docs)
- Artifact naming blog: https://blog.urth.org/2023/04/16/naming-your-binary-executable-releases/ (LOW confidence — individual blog)
- macOS signing in GitHub Actions: https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/ (MEDIUM confidence — established blog, consistent with Wails docs)
- Existing build.yml in repo: /Users/ken/dev/agenthub/.github/workflows/build.yml (HIGH confidence — source of truth)

---

## Prior Milestone Research (v1.7 — Daemon UX & Branding)

The v1.7 research (system tray, remote session indicators, app icons, splash screen) has been
preserved below in a condensed form. Full content available in git history at the commit prior to
this v1.8 update.

Key decisions from v1.7 relevant to v1.8:
- Native macOS cgo NSStatusBar tray (no fyne.io/systray) — affects build complexity for CI
- `ditto` for notarization archive — confirmed correct, reused in release.yml packaging
- Post-build cp of pre-built ICNS into bundle — release.yml must replicate this step after wails build

See git history for full v1.7 FEATURES.md content.

---

*Feature research for: v1.8 GitHub Distribution & CI/CD — release automation, Homebrew tap, WinGet submission*
*Researched: 2026-04-03*
