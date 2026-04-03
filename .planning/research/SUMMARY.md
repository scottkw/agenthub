# Project Research Summary

**Project:** AgentHub v1.8 — GitHub Distribution & CI/CD
**Domain:** Release automation, Homebrew tap distribution, and WinGet package manager submission for a Wails v2 cross-platform desktop app
**Researched:** 2026-04-03
**Confidence:** HIGH

## Executive Summary

AgentHub v1.8 is a pure CI/CD infrastructure milestone — no new app features, only distribution automation. The project migrates from Gitea to GitHub and establishes a fully automated release pipeline: conventional commits drive release-please versioning, tag-triggered multi-platform builds with macOS signing/notarization populate GitHub Releases, and a post-release distribution workflow updates both a Homebrew cask tap and a WinGet package manifest. The pattern is well-understood, with high-quality reference implementations available (including the existing `scottkw/storcat` repo and the current `build.yml` signing pipeline).

The recommended approach is four GitHub Actions workflow files: the existing `build.yml` (modified to remove signing), plus three new files — `release-please.yml`, `release.yml`, and `distribute.yml`. Critically, macOS signing/notarization moves out of `build.yml` into `release.yml` only, saving significant GitHub Actions runner minutes and Apple notarization API quota on every PR. The Homebrew tap uses a cross-repo `workflow_dispatch` pattern with a classic PAT. WinGet requires a manual first submission to establish package identity before automation takes over.

The most significant risks are process risks rather than technical ones: git history loss during Gitea migration, the release-please PAT misconfiguration that prevents tags from triggering downstream workflows, and the WinGet requirement for a pre-existing manual submission. All three are well-documented and avoidable with explicit phase ordering. The macOS CGO cross-compilation constraint and the `--wait` requirement on `notarytool submit` are the critical technical pitfalls from the existing pipeline.

## Key Findings

### Recommended Stack

The v1.8 stack adds no new application technologies — only GitHub Actions tooling. The core toolset is: `googleapis/release-please-action@v4` for automated versioning from conventional commits, `dAppServer/wails-build-action@main` (already in use) for multi-platform builds, `softprops/action-gh-release@v2` for artifact uploads, `create-dmg` v1.2.3 for macOS DMG packaging, `gh workflow run` for cross-repo Homebrew tap triggering, and `vedantmgoyal9/winget-releaser@main` for WinGet manifest submission.

**Core technologies:**
- `googleapis/release-please-action@v4`: Automated versioning, CHANGELOG.md generation, and git tag creation from conventional commits — eliminates manual release tagging
- `dAppServer/wails-build-action@main`: Multi-platform Wails builds including macOS universal, Linux, and Windows NSIS — already proven in `build.yml`; pin to `@main`, not version tags; use `wails-version: "v2.9.0"` override if v2.10.x issues arise
- `softprops/action-gh-release@v2`: Attaches built artifacts to GitHub Releases — replaces deprecated `actions/create-release`
- `create-dmg` v1.2.3: Wraps `hdiutil` to produce installer-style DMG with drag-to-Applications layout for Homebrew distribution
- `vedantmgoyal9/winget-releaser@main`: Komac-backed WinGet manifest generation and PR submission to `microsoft/winget-pkgs` via the `scottkw` fork — requires classic PAT with `public_repo` scope

Two new secrets are required beyond the 7 existing macOS signing secrets: `TAP_DEPLOY_TOKEN` (classic PAT, `repo` + `workflow` scopes) and `WINGET_TOKEN` (classic PAT, `public_repo` scope). Fine-grained PATs are explicitly incompatible with both cross-repo workflow dispatch and `winget-releaser`.

### Expected Features

The v1.8 feature set is tightly scoped: a professional release pipeline. Every item on the must-have list is P1 with clear implementation paths from verified sources.

**Must have (table stakes):**
- GitHub Releases with multi-platform binary artifacts — users cannot download without this
- Versioned SemVer release tags with CHANGELOG.md — release-please automates both
- macOS codesigning and notarization on release builds — Gatekeeper blocks unsigned binaries on 10.15+
- SHA256 checksums file attached to each release — required by Homebrew and security-conscious users
- Consistent artifact naming convention — package managers and automation depend on predictable names

**Should have (differentiators):**
- Homebrew cask tap (`brew install --cask agenthub`) — macOS power users expect this; eliminates download friction and enables `brew upgrade`
- WinGet submission (`winget install AgentHub.AgentHub`) — Windows legitimacy signal and zero-friction install for CLI-first users
- Automated SHA256 injection in `distribute.yml` — eliminates the most error-prone manual step in every release
- `packaging/homebrew/` and `packaging/winget/` templates in repo — makes distribution logic inspectable and testable locally
- release-please Release PR workflow — provides changelog and version review gate before each release ships

**Defer (v2+):**
- Linux `.deb` / AppImage packaging — only if user demand emerges
- Scoop manifest — low ROI vs WinGet for Windows users
- Submission to `homebrew/homebrew-cask` core — requires established user base; personal tap has identical install UX
- Windows EV code signing — $300-500+/year with hardware token; SmartScreen does not block execution like Gatekeeper
- VERSION injection into WelcomeTab.tsx — existing tech debt; separate concern from release automation

### Architecture Approach

The pipeline follows a strict sequential trigger chain: `push to main` fires `release-please.yml` which maintains a Release PR; merging that PR creates a git tag which fires `release.yml` for multi-platform builds; completing the builds populates the GitHub Release which fires `distribute.yml` for Homebrew and WinGet. The single most important structural change is removing macOS signing from `build.yml` — keeping it there burns Apple notarization quota and 20+ GitHub Actions minutes on every PR build. See `ARCHITECTURE.md` for the complete data flow diagram and explicit file modification/creation inventory.

**Major components:**
1. `release-please.yml` — reads conventional commits on `main`, maintains Release PR, creates git tag and empty GitHub Release on merge; uses `release-type: go` with `extra-files` annotations for version file updates
2. `release.yml` — tag-triggered multi-platform builds (macOS universal, Linux amd64 x2, Windows amd64); signs/notarizes macOS using existing 7-secret pipeline; uploads artifacts with consistent naming to GitHub Release
3. `distribute.yml` — post-release event handler: downloads macOS ZIP, computes SHA256, triggers tap repo update via PAT; runs `winget-releaser` to submit WinGet manifest PR; includes retry logic for asset availability race condition
4. `scottkw/homebrew-agenthub` (separate repo) — tap repo with `Casks/agenthub.rb`; updated via cross-repo `workflow_dispatch` with `version` and `sha256` inputs; must be named `homebrew-agenthub` to enable `brew tap scottkw/agenthub` shorthand
5. `packaging/` templates — `homebrew/agenthub.rb.tmpl` and `winget/manifests/*.yaml` for reference and bootstrapping

**Files to modify in existing repo:**
- `.github/workflows/build.yml` — remove signing steps (lines 75-108); leave build matrix intact
- `wails.json` — add `// x-release-please-version` annotation on `productVersion` line
- `frontend/src/components/WelcomeTab.tsx` — add annotation; fix hardcoded `'1.0.0'` to `'1.7.0'`
- `main.go` — add `var Version string`; wire to `--version` flag

### Critical Pitfalls

1. **Git history lost during Gitea migration** — Use `git clone --bare` + `git push --mirror` for the initial push; verify all v1.0-v1.7 tags are present on GitHub before enabling any workflows; a missing tag baseline causes release-please to create a v0.0.0 release PR

2. **Signing left in `build.yml` (current state)** — Every PR merge currently triggers Apple notarization API (10-15 min, burns quota); remove the 4 signing steps from `build.yml` immediately; move them to `release.yml` only

3. **`distribute.yml` asset availability race condition** — `release: types: [published]` fires when the release is published, but macOS notarization in `release.yml` takes 15-25 more minutes; Homebrew SHA256 download will fail if it runs before the asset is uploaded; implement retry with exponential backoff in the Homebrew job

4. **WinGet first submission must be manual** — `winget-releaser` only works after the package identity exists in `microsoft/winget-pkgs`; manually submit the v1.8.0 manifests first; only enable the automated job after the first PR is merged

5. **`GITHUB_TOKEN` cannot push to other repos** — Both the Homebrew tap update and winget-releaser require classic PATs (not fine-grained, not `GITHUB_TOKEN`); store `TAP_DEPLOY_TOKEN` and `WINGET_TOKEN` before writing the distribute workflow

6. **release-please `version-file` is ignored for Go release-type (issue #2541)** — Use `extra-files` with `"type": "generic"` and `// x-release-please-version` annotation comments; do not rely on `version-file` config

7. **`notarytool submit` without `--wait` silently fails** — Without `--wait`, the command exits 0 immediately but notarization never completes; `xcrun stapler staple` then silently fails; users get Gatekeeper warnings; always use `--wait` (already correct in `build.sh`)

## Implications for Roadmap

Based on the dependency graph in `ARCHITECTURE.md`, the phase structure is clear and ordered by hard dependencies.

### Phase 1: Git Migration to GitHub
**Rationale:** All subsequent phases depend on the GitHub repository existing with complete history. This is the true Day 0 blocker.
**Delivers:** GitHub repo with full Gitea history, all v1.0-v1.7 tags, and confirmed CI secrets migrated from Gitea
**Addresses:** Table stakes — users cannot access releases without GitHub hosting
**Avoids:** Pitfall 1 (history loss) — requires mirror push, not default push

### Phase 2: release-please Versioning Setup
**Rationale:** release-please creates the git tags that trigger all downstream workflows; nothing else can ship without it. Low complexity, no external deps.
**Delivers:** Automated Release PR workflow, CHANGELOG.md generation, version bump in `wails.json` and `WelcomeTab.tsx`, baseline tag set to `1.7.0`
**Uses:** `googleapis/release-please-action@v4`, `release-please-config.json`, `.release-please-manifest.json`
**Avoids:** Pitfall 9 (version-file bug) — use `extra-files` with annotation markers; Pitfall 10 (non-conventional commits) — audit history and create baseline tag before enabling

### Phase 3: Release Build Pipeline
**Rationale:** Depends on Phase 2 for tag creation. Establishes the artifact naming convention that Homebrew and WinGet depend on — naming must be locked before building distribution automation.
**Delivers:** `release.yml` with multi-platform matrix, macOS signing/notarization, SHA256 checksums, artifacts uploaded to GitHub Release; `build.yml` modified to remove signing
**Uses:** `dAppServer/wails-build-action@main`, `softprops/action-gh-release@v2`, existing 7 macOS secrets, `create-dmg` v1.2.3
**Avoids:** Pitfall 2 (macOS cross-compile) — pin `macos-14`, not `macos-latest`; Pitfall 3 (signing resources error) — verify plist consistency before signing; Pitfall 4 (altool decommissioned) — use `notarytool` exclusively; Pitfall 13 (artifact naming collisions) — platform-specific names in matrix

### Phase 4: Packaging Templates
**Rationale:** Can run parallel to Phase 3 once artifact names are confirmed. Templates must exist before distribute.yml can render them.
**Delivers:** `packaging/homebrew/agenthub.rb.tmpl` and `packaging/winget/manifests/*.yaml` reference templates in the main repo
**Uses:** Homebrew Cask Cookbook stanzas, WinGet schema 1.12.0 three-file format
**Avoids:** Pitfall 11 (Homebrew requires .app in ZIP/DMG, not bare directory) — use `ditto` or `create-dmg` packaging in release.yml

### Phase 5: Homebrew Tap Distribution
**Rationale:** Depends on Phase 3 (confirmed artifact names, macOS asset available) and Phase 4 (cask template). Requires `scottkw/homebrew-agenthub` repo to be created first.
**Delivers:** `scottkw/homebrew-agenthub` tap repo with `Casks/agenthub.rb` and `update-cask.yml`; `distribute.yml` Homebrew job with retry logic; users can `brew install --cask agenthub`
**Uses:** `TAP_DEPLOY_TOKEN` classic PAT, `gh workflow run` cross-repo dispatch, `sha256sum`
**Avoids:** Pitfall 6 (GITHUB_TOKEN cross-repo failure) — classic PAT required; Pitfall 12 (binary stanza crash) — test CLI invocation outside bundle before adding `binary` stanza

### Phase 6: WinGet Distribution
**Rationale:** Independent from Homebrew (Phase 5) but requires Phase 3 for the Windows installer artifact. Two-phase process: manual first submission, then automated subsequent releases.
**Delivers:** Initial `packaging/winget/` manifests manually submitted to `microsoft/winget-pkgs`; `distribute.yml` WinGet job via `winget-releaser` for subsequent versions; `winget install AgentHub.AgentHub` works
**Uses:** `vedantmgoyal9/winget-releaser@main`, `WINGET_TOKEN` classic PAT, fork of `microsoft/winget-pkgs` under `scottkw`
**Avoids:** Pitfall 7 (manual first submission required) — bootstrap manually before enabling automation; Pitfall 8 (fine-grained PAT rejected) — classic PAT with `public_repo` only; Pitfall 14 (raw binary rejected) — NSIS installer from existing `nsis: true` satisfies `InstallerType: nullsoft`

### Phase Ordering Rationale

- Phase 1 before everything: GitHub repo must exist with all tags for release-please baseline
- Phase 2 before Phase 3: Tags cannot be created until release-please is configured; without tags, release.yml never fires
- Phase 3 before Phases 5-6: Artifact names and format must be finalized before Homebrew cask URL and WinGet installer URL can be written
- Phases 4-5-6 can be developed in parallel after Phase 3 artifact naming is confirmed
- Phase 6 WinGet has a multi-day external dependency (Microsoft PR review) — start early; submission is async and does not block milestone completion

### Research Flags

Phases with standard patterns (skip research-phase):
- **Phase 2 (release-please):** Official docs are complete; existing storcat reference implementation available; no additional research needed
- **Phase 3 (release builds):** Existing `build.yml` is the source of truth; steps are copy-edit, not net-new; no additional research needed
- **Phase 4 (packaging templates):** Template content fully specified in STACK.md and FEATURES.md; straightforward file authoring

Phases that may need validation during execution:
- **Phase 1 (Git migration):** Verify mirror push preserves all tags before proceeding; one-way step if done wrong
- **Phase 5 (Homebrew tap):** Test `brew install --cask` end-to-end after initial cask commit; cask formula edge cases (binary stanza, livecheck) may need iteration
- **Phase 6 (WinGet):** First manual submission has an external timeline dependency on Microsoft review; test `winget validate` locally before submitting

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Versions verified against official releases as of 2026-04-03; action names and PAT scopes confirmed from source repos |
| Features | HIGH | Scoped tightly to distribution infrastructure; well-established patterns with multiple reference implementations |
| Architecture | HIGH | Existing `build.yml` and `build.sh` inspected directly; CI/CD patterns verified from official docs and working cross-repo examples |
| Pitfalls | HIGH | 14 pitfalls documented from official GitHub Issues, Apple deprecation notices, and direct code inspection of existing files |

**Overall confidence:** HIGH

### Gaps to Address

- **Asset availability race condition in distribute.yml:** The retry-with-backoff approach for Homebrew SHA256 download is recommended but the exact polling implementation (interval, max attempts) needs to be written against observed release.yml run times (typically 20-25 min for macOS notarization). Use `gh release view --json assets` polling.

- **`wails.json` annotation syntax:** JSON does not natively support comments. The `// x-release-please-version` annotation works with release-please's regex-based `generic` type, but this should be verified on the first Release PR. A standalone `VERSION` file is the fallback if the annotation approach fails.

- **Homebrew binary stanza:** Whether `agenthub` CLI mode works when invoked directly outside the `.app` bundle context (via a Homebrew `binary` stanza symlink) is untested. Requires a quick local test post-signing before the cask is published.

- **WinGet NSIS vs portable:** STACK.md specifies `InstallerType: nullsoft` (NSIS) for the NSIS installer produced by `wails-build-action` with `nsis: true`. Verify the NSIS installer (not the bare `.exe`) is the artifact being uploaded to the release — `winget-releaser` auto-detects from `installers-regex`.

## Sources

### Primary (HIGH confidence)
- `.github/workflows/build.yml` direct inspection — confirmed existing signing steps, matrix structure, 7 secret names
- `build.sh` direct inspection — confirmed `sign_and_notarize` using `notarytool --wait` and `ditto`
- `wails.json` direct inspection — confirmed `productVersion: "1.0.0"` and JSON structure
- [googleapis/release-please-action releases](https://github.com/googleapis/release-please-action/releases) — v4.4.0 confirmed Oct 2025
- [softprops/action-gh-release releases](https://github.com/softprops/action-gh-release/releases) — v2.6.1 confirmed Mar 2025
- [Homebrew Cask Cookbook](https://docs.brew.sh/Cask-Cookbook) — cask stanza verification
- [How to Create and Maintain a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap) — `homebrew-` naming, `Casks/` structure
- [Microsoft WinGet manifest docs](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest) — schema 1.12.0, updated 2026-03-24
- [vedantmgoyal9/winget-releaser](https://github.com/vedantmgoyal9/winget-releaser) — v2 tag Jan 2026, classic PAT requirement confirmed
- [create-dmg releases](https://github.com/create-dmg/create-dmg) — v1.2.3 Nov 2025

### Secondary (MEDIUM confidence)
- [release-please issue #2541](https://github.com/googleapis/release-please/issues/2541) — `version-file` ignored for Go type; `extra-files` workaround confirmed
- [dAppServer/wails-build-action README](https://github.com/dAppServer/wails-build-action) — `@main` usage required; Wails v2.10.0 broken warning
- [Automating Homebrew Tap Updates with GitHub Actions](https://builtfast.dev/blog/automating-homebrew-tap-updates-with-github-actions/) — cross-repo pattern and SHA256 computation
- [Automatic Code-Signing and Notarization for macOS apps using GitHub Actions](https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/) — 7-secret pattern confirmed matches existing build.yml
- [WinGet PR review timeline](https://github.com/microsoft/winget-pkgs/discussions/19502) — hours to ~1 day for new packages

### Tertiary (LOW confidence)
- [Naming Your Binary Executable Releases](https://blog.urth.org/2023/04/16/naming-your-binary-executable-releases/) — artifact naming convention informed by community practice; no authoritative standard exists

---
*Research completed: 2026-04-03*
*Ready for roadmap: yes*
