# Pitfalls Research

**Domain:** GitHub Distribution & CI/CD — Wails desktop app migrating from Gitea to GitHub with automated release pipelines, Homebrew tap, and WinGet distribution
**Researched:** 2026-04-03
**Confidence:** HIGH — CI/CD pitfalls verified from official docs, GitHub Issues, and community post-mortems; Homebrew docs read directly; WinGet submission process verified from microsoft/winget-pkgs discussions

---

## Critical Pitfalls

### Pitfall 1: Git History Lost When Pushing to GitHub Without `--mirror`

**What goes wrong:**
A simple `git remote add github <url> && git push github main` only pushes the current branch. All other branches, all tags (including v1.0–v1.7 release tags), and reflog data are left behind on Gitea. Release-please, GitHub Releases, and Homebrew cask update workflows all depend on tags being present to identify the "latest" version.

**Why it happens:**
`git push` without flags only pushes the configured branch. Tags are not pushed automatically. Developers testing the migration push one branch, confirm it looks right, and call it done — without realizing tags are missing.

**How to avoid:**
Use a bare clone + mirror push:
```bash
git clone --bare <gitea-url> agenthub-migration.git
cd agenthub-migration.git
git push --mirror <github-url>
```
After the mirror push, verify on GitHub:
- `git ls-remote --tags origin` shows all v1.0–v1.7 tags
- `git log --oneline --decorate` on main shows full history
- GitHub's commit count on the repo page matches Gitea's

**Warning signs:**
- GitHub repo shows fewer commits than Gitea
- `git tag` on a fresh clone of the GitHub remote returns nothing
- Release-please creates a v0.0.0 release PR instead of starting from v1.7.x
- Homebrew tap automation reports "could not find latest tag"

**Phase to address:** Phase: Git Migration — run mirror push as the first step, before any CI workflows reference the repo.

---

### Pitfall 2: Wails macOS Builds Cannot Cross-Compile — Must Use `macos-latest` Runner (Now Arm64)

**What goes wrong:**
Wails v2 macOS builds require native CGO compilation. There is no cross-compile path from Linux to macOS for a Wails app. If the CI matrix tries to build the `.app` bundle on `ubuntu-latest`, it will fail with CGO linker errors because macOS frameworks (CoreFoundation, AppKit, WebKit) are not available. Additionally, as of 2024, `macos-latest` maps to `macos-14` which is ARM64 (Apple Silicon). A universal binary requires explicit `GOARCH=amd64` + `GOARCH=arm64` builds and `lipo` to merge them.

**Why it happens:**
Developers copy a Go cross-compile pattern (which works for pure-Go or simple CGO) and expect it to work for Wails. The existing `build.sh` works locally on a Mac but GitHub Actions uses a specific runner image that changed architecture in mid-2024.

**How to avoid:**
- `macos-14` and `macos-latest` are ARM64. Use `macos-13` for an Intel runner if needed, or build both architectures on `macos-latest` (ARM64) using Rosetta + `GOARCH=amd64`.
- The existing `build.sh` already handles cross-arch via `-arch` flags — replicate that logic in CI.
- Explicitly specify `runs-on: macos-14` (not `macos-latest`) to avoid the mapping changing under you in a future GitHub Actions update.
- Linux builds must use Docker cross-compilation (as `build.sh` already does with `wailsapp/cc` images); Windows builds require `windows-latest` runner.

**Warning signs:**
- CI macOS job fails with: `ld: framework not found CoreFoundation`
- CI produces only an `amd64` binary but `file` shows `Mach-O 64-bit executable arm64`
- `wails build` succeeds locally but produces a different binary size in CI

**Phase to address:** Phase: GitHub Actions CI — define explicit runner versions in the matrix; never use `macos-latest` for production builds.

---

### Pitfall 3: macOS Signing Fails With "Code Has No Resources But Signature Indicates They Must Be Present"

**What goes wrong:**
The `codesign` step succeeds, but notarization or Gatekeeper validation fails with: `code has no resources but signature indicates they must be present`. This is a Wails-specific bug present through at least v2.9.2 where `CFBundleExecutable` in `Info.plist` uses `.Name` (the project name) but the actual binary in the bundle uses `.OutputFilename` (which may have different casing or a custom name).

**Why it happens:**
In Wails v2, the binary name embedded in the `.app` bundle comes from the `wails.json` `name` field, while the actual executable on disk uses `outputfilename`. If these differ (e.g., `name: "AgentHub"` vs `outputfilename: "agenthub"`), the plist and binary name mismatch causes codesign to create a signature that references a file that doesn't exist.

**How to avoid:**
- Ensure `name` and `outputfilename` in `wails.json` are consistent in casing (or explicitly match).
- After `wails build`, verify before signing: `codesign -vvv --strict --verify build/bin/AgentHub.app` — this reports the mismatch immediately.
- Apply the fix from Wails PR #3789: use `OutputFilename` (not `Name`) for `CFBundleExecutable` in the post-build plist, or patch the plist in CI after build and before signing.
- Use `ditto` (not `zip`) when archiving for notarization — `ditto -c -k --keepParent AgentHub.app AgentHub.zip` preserves macOS extended attributes; `zip` destroys them, breaking the signature.

**Warning signs:**
- `codesign` exits 0 but `xcrun notarytool submit` returns "Invalid"
- `codesign -vvv --strict --verify` prints "code has no resources but signature indicates they must be present"
- Notarization log (retrieved with `xcrun notarytool log <uuid>`) shows CFBundleExecutable mismatch

**Phase to address:** Phase: macOS Signing & Notarization — verify plist consistency before signing; add codesign verify step to CI before notarization submission.

---

### Pitfall 4: macOS Notarization Uses Altool in Old Docs — altool Was Decommissioned in 2023

**What goes wrong:**
Any CI script or community blog post from before late 2023 uses `xcrun altool --notarize-app` for Apple notarization. This command now returns: "Notarization of MacOS applications using altool has been decommissioned. Please use notarytool." The entire signing/notarization job fails with no useful error about what to replace it with.

**Why it happens:**
Apple deprecated altool in WWDC 2022 and decommissioned it fully in Fall 2023. Old Wails documentation, the Wails signing guide, and many community workflows still reference altool. Developers copy these examples without checking the current status.

**How to avoid:**
Use exclusively `xcrun notarytool` for all notarization steps:
```bash
xcrun notarytool store-credentials "notarytool-profile" \
  --apple-id "$APPLE_ID" \
  --team-id "$TEAM_ID" \
  --password "$APP_SPECIFIC_PASSWORD"

xcrun notarytool submit AgentHub.zip \
  --keychain-profile "notarytool-profile" \
  --wait

xcrun stapler staple AgentHub.app
```
Store credentials in the runner's keychain at the start of the job, not in the submit command itself.

**Warning signs:**
- Any CI step runs `altool --notarize-app`
- Apple returns "altool has been decommissioned" error mid-workflow
- Notarization step hangs indefinitely (symptom of altool timeout, not notarytool's `--wait`)

**Phase to address:** Phase: macOS Signing & Notarization — write all notarization steps using notarytool from day one; do not copy any CI template that references altool.

---

### Pitfall 5: Hardened Runtime (`--options runtime`) Breaks Apps Requiring Dynamic Code

**What goes wrong:**
The `--options runtime` flag on `codesign` enables Hardened Runtime, which is required for Apple notarization. However, Hardened Runtime disables JIT compilation, disables unsigned dynamic libraries, and restricts certain POSIX capabilities. Wails apps that use CGO with Objective-C (AgentHub's tray implementation) or any dynamic linking at runtime may encounter unexpected crashes or permission denials after notarization.

**Why it happens:**
Notarization requires Hardened Runtime. Developers add `--options runtime` to pass notarization, ship the build, and only discover runtime failures when users report crashes on first launch.

**How to avoid:**
After signing with `--options runtime`, test the signed `.app` bundle before notarizing:
```bash
open -n ./build/bin/AgentHub.app
```
Test all critical flows: tray creation, daemon connection, session creation, Tailscale health check. If a specific capability is required (e.g., `com.apple.security.cs.allow-jit`), add an entitlements plist:
```bash
codesign --force --deep --sign "$CERT_NAME" \
  --options runtime \
  --entitlements entitlements.plist \
  AgentHub.app
```

**Warning signs:**
- App crashes immediately on launch after notarization (not before)
- macOS Console.app shows `AMFI: code signature validation failed`
- App works when launched from Xcode/unsigned context but not from Finder/Gatekeeper

**Phase to address:** Phase: macOS Signing & Notarization — test signed app locally before submitting for notarization; add entitlements plist if CGO dynamic features are required.

---

### Pitfall 6: `GITHUB_TOKEN` Cannot Push to External Repos — Homebrew Tap Update Fails Silently

**What goes wrong:**
The distribution workflow that auto-updates the Homebrew tap (`scottkw/homebrew-agenthub`) after a release cannot use the built-in `GITHUB_TOKEN`. `GITHUB_TOKEN` is scoped to the repository where the workflow is running (`scottkw/agenthub`). Any attempt to push to `scottkw/homebrew-agenthub` fails with a 403 Forbidden error — and if the workflow doesn't check the push exit code, it silently completes "successfully" while the tap is not updated.

**Why it happens:**
GitHub Actions auto-generates `GITHUB_TOKEN` per workflow run, scoped to one repo. This is intentional for security. Developers assume that being the same GitHub account/org owner means the token has broader access.

**How to avoid:**
Create a dedicated Personal Access Token (classic PAT, not fine-grained) with `public_repo` scope. Store it as a secret in the main repo (e.g., `HOMEBREW_TAP_PAT`). Use it explicitly in the workflow when pushing to the tap repo:
```yaml
- uses: actions/checkout@v4
  with:
    repository: scottkw/homebrew-agenthub
    token: ${{ secrets.HOMEBREW_TAP_PAT }}
```
Verify the push succeeds by checking the exit code and failing the workflow loudly if it does not.

**Warning signs:**
- Distribution workflow shows green checkmark but tap repo has no new commit
- GitHub API returns 403 or "Resource not accessible by integration" in the workflow logs
- Tap formula still shows the old version after a release

**Phase to address:** Phase: Homebrew Tap — create and store PAT before writing the workflow; test the cross-repo push in isolation.

---

### Pitfall 7: WinGet Submission Requires a Pre-Existing Manual Entry for the First Version

**What goes wrong:**
Automated WinGet submission via `winget-releaser` or `wingetcreate update` only works for packages already present in the winget-pkgs community repository. The action exits with "Package not found" on the first submission for a new package identifier. There is no automated path for the initial submission — it must be done manually via a hand-crafted PR to `microsoft/winget-pkgs`.

**Why it happens:**
Developers see `vedantmgoyal9/winget-releaser` and assume it handles everything including the initial submission. The README states "At least one version of your package should already be present in the Windows Package Manager Community Repository" — a prerequisite that is easy to miss.

**How to avoid:**
Phase the WinGet work into two parts:
1. **Manual initial submission:** Create the v1.8.0 manifest by hand (3 YAML files: version, installer, defaultLocale). Submit a PR to `microsoft/winget-pkgs`. Wait for review and merge (typically a few hours to 1 day).
2. **Automated subsequent updates:** After the initial merge, use `wingetcreate update` in CI for all subsequent versions.

For the initial submission, use `wingetcreate new <installer-url>` to generate the manifest skeleton, then review and correct fields before submitting.

**Warning signs:**
- `wingetcreate update` fails with "Could not find package" or 404 on package lookup
- Automated distribution workflow triggers on release but WinGet job fails immediately
- WinGet CI runs but no PR appears in microsoft/winget-pkgs

**Phase to address:** Phase: WinGet Distribution — document the two-phase approach; do not automate the initial submission.

---

### Pitfall 8: WinGet PAT Must Be Classic (Not Fine-Grained) and `winget-releaser` Requires `public_repo` Scope

**What goes wrong:**
The `vedantmgoyal9/winget-releaser` action requires a Personal Access Token to submit PRs to `microsoft/winget-pkgs`. Fine-grained PATs are not supported by the action as of 2024. Additionally, the PAT must have `public_repo` scope at minimum. Submitters who use fine-grained PATs or insufficient scopes get a `403 Forbidden: Must have admin rights to Repository` error from the Octokit client inside the action.

**Why it happens:**
GitHub has been transitioning from classic to fine-grained PATs, and many developers default to fine-grained. The underlying issue is that `microsoft/winget-pkgs` requires the action to fork the repo, which requires specific permissions that fine-grained PATs handle differently.

**How to avoid:**
- Generate a **classic PAT** (not fine-grained) at https://github.com/settings/tokens
- Grant `public_repo` scope
- Store as `WINGET_TOKEN` secret in the main repo
- Pass it to the releaser action via `token: ${{ secrets.WINGET_TOKEN }}`

**Warning signs:**
- Error: "Must have admin rights to Repository" despite having `public_repo` on a fine-grained PAT
- PR creation step fails but `wingetcreate` manifest generation succeeds
- Using a fine-grained PAT scoped to `microsoft/winget-pkgs` specifically (this does not work)

**Phase to address:** Phase: WinGet Distribution — generate and test the classic PAT before writing the workflow.

---

### Pitfall 9: release-please `release-type: go` Ignores `version-file` — Version Not Updated Automatically

**What goes wrong:**
Setting `release-type: go` with a `version-file` pointing to a custom Go file (e.g., `internal/version/version.go`) results in release-please doing nothing to update the version in that file. The release PR is created correctly, but the Go source file's version constant is not bumped. Shipping this means the `version` shown in the app's UI (`WelcomeTab.tsx` and the CLI) stays hardcoded at the old version.

**Why it happens:**
There was a documented bug in release-please where `version-file` was silently ignored for Go projects (Issue #2541, fixed in PR #2542). However, depending on which version of the action is pinned, the fix may not be present. Additionally, `release-type: go` only knows how to update `go.mod` module lines — it does not understand arbitrary Go constant files.

**How to avoid:**
Use `release-type: simple` combined with `extra-files` to update any non-standard version location:
```json
{
  "release-type": "simple",
  "extra-files": [
    "internal/version/version.go"
  ]
}
```
In `internal/version/version.go`, annotate the version constant with the release-please marker:
```go
// x-release-please-version
const Version = "1.8.0"
```
Also update `WelcomeTab.tsx` via `extra-files` if it hardcodes `VERSION`. Verify the release PR actually modifies both files before merging.

**Warning signs:**
- Release-please PR shows changelog and tag bump but no diff in Go source files
- App shows old version string after a release
- `git log internal/version/version.go` shows no commits from release-please

**Phase to address:** Phase: release-please Setup — configure `release-type: simple` with `extra-files` annotations on all version-bearing files before the first release.

---

### Pitfall 10: release-please Analyzes Only Commits Since the Last Release Tag — Non-Conventional Commits Are Invisible

**What goes wrong:**
AgentHub's commit history from Gitea uses a mix of commit styles (from PROJECT.md: `chore:`, `docs:`, `fix:` prefixes are present but not consistently). When release-please scans the history from the last tag, any commit without a recognized conventional commit prefix (`feat:`, `fix:`, `perf:`, `chore:`, etc.) is ignored. If none of the commits since the last tag are recognized, release-please creates no release PR at all — it silently does nothing, waiting for the next push.

**Why it happens:**
release-please is strict about conventional commits. It does not attempt to infer intent from commit message content. Existing AgentHub commits like "port icon build to build.sh" (no prefix) or merge commits are skipped entirely.

**How to avoid:**
- Audit the commit history from the last Gitea tag to HEAD for conventional commit compliance before enabling release-please.
- Do not rely on release-please to retroactively create a release for non-conforming history. Instead, manually create a v1.8.0 GitHub Release tag to establish a baseline, then enforce conventional commits in all future work.
- Add a commit-message linter (commitlint) to the CI PR workflow to enforce the format before merge.
- Document the commit format in CONTRIBUTING.md or the project README.

**Warning signs:**
- release-please action runs on every push to main but never creates or updates a release PR
- `release-please bootstrap` output shows "no commits to release" despite recent work
- GitHub Release is created manually but release-please then immediately wants to create another release PR for the same commits

**Phase to address:** Phase: release-please Setup — audit commit history before enabling; create a baseline tag manually; add commitlint to CI.

---

### Pitfall 11: Homebrew Cask Requires `.app` Bundle — Wails Outputs Different Formats Per Platform

**What goes wrong:**
Homebrew casks for macOS desktop apps require a `.app` bundle distributed as a DMG, PKG, or ZIP. Wails `wails build` for macOS produces an `.app` directory (not a DMG by default). The `url` field in the cask must point to a downloadable archive — not a GitHub Release asset that is just the raw `.app` directory. If the release job uploads the `.app` directly (without wrapping it), users get a download failure or a broken installation.

**Why it happens:**
Wails does not automatically create a DMG. The existing `build.sh` uses `ditto` to create a ZIP for notarization but does not produce a DMG installer. Homebrew casks work with ZIP archives of `.app` bundles but most professional distributions use DMGs (with drag-to-Applications UX).

**How to avoid:**
Decide on distribution format early and be consistent:
- **ZIP of .app** (simpler): Archive with `ditto -c -k --keepParent AgentHub.app AgentHub-macos.zip`. The cask `url` points to this ZIP; the `app "AgentHub.app"` stanza extracts and moves it.
- **DMG** (better UX): Use `create-dmg` or `hdiutil` in the CI release job to create a DMG with an Applications symlink. The cask uses `pkg "AgentHub.dmg"` or specifies the app inside the volume.

For AgentHub, ZIP is the simpler choice given the existing `ditto` pipeline. Document the format choice in the packaging template so the cask `url` and the CI upload name stay in sync.

**Warning signs:**
- Homebrew reports "No cask artifact" during `brew install`
- The release asset is an `.app` directory uploaded as-is (GitHub releases don't support directory uploads — it would be silently omitted or broken)
- Cask `sha256` check fails because the locally brewed archive differs from what the CI uploaded

**Phase to address:** Phase: Homebrew Tap — define the release artifact format (ZIP) before writing the cask; test `brew install` from the tap before publicizing.

---

### Pitfall 12: Homebrew Cask for an App With Both GUI and CLI Binary — `binary` Stanza May Crash the App

**What goes wrong:**
AgentHub ships a single binary that dispatches to GUI (no args), CLI (subcommands), and daemon modes. A naive cask might install the `.app` bundle AND add a `binary` stanza pointing to the executable inside the bundle (e.g., `binary "#{appdir}/AgentHub.app/Contents/MacOS/agenthub"`). This is technically correct for a pure CLI binary inside an app, but for apps with Wails/Electron-like WebView backends, launching the binary directly (outside the `.app` bundle context) often fails because the bundle's resources are not accessible via the binary's relative paths.

**Why it happens:**
Community examples show adding a `binary` stanza to expose CLI functionality. Developers assume a Go binary is relocatable. Wails apps embed resources relative to the bundle structure and expect to find `Resources/` and `Frameworks/` adjacent to the executable.

**How to avoid:**
- Test the `binary` stanza path by running the binary directly outside the bundle before including it in the cask.
- If `agenthub` CLI mode works from `Contents/MacOS/agenthub` (it likely does, since CLI mode does not initialize Wails), include the `binary` stanza.
- If CLI mode fails when invoked outside the bundle, create a thin wrapper script at a PATH location instead, or document that users should run via `/Applications/AgentHub.app/Contents/MacOS/agenthub`.
- Add a `zap` stanza to clean up the daemon socket and preferences on uninstall.

**Warning signs:**
- `agenthub` CLI commands work in development but fail after Homebrew install
- Error references missing `Resources/` path when invoked from the symlinked `binary` location
- App crashes with "no such file or directory" for embedded assets

**Phase to address:** Phase: Homebrew Tap — test CLI invocation via the `binary` stanza path before publishing the cask.

---

### Pitfall 13: Artifact Naming Collisions in Multi-Platform Release Matrix

**What goes wrong:**
A GitHub Actions matrix that runs macOS, Linux, and Windows jobs in parallel all uploading to `actions/upload-artifact` with the same artifact name (e.g., `agenthub-release`) will silently overwrite each other. Only the last job's artifact survives. When the release job later downloads and attaches assets to the GitHub Release, only one platform's binary is present.

**Why it happens:**
Matrix jobs run concurrently. `actions/upload-artifact` with `if-no-files-found: warn` (the default) does not error on collisions. Developers test with a single platform and don't notice the problem until all three platforms run simultaneously.

**How to avoid:**
Include the platform and architecture in the artifact name:
```yaml
- uses: actions/upload-artifact@v4
  with:
    name: agenthub-${{ matrix.os }}-${{ matrix.arch }}
    path: dist/
```
In the release job, download all artifacts with a wildcard:
```yaml
- uses: actions/download-artifact@v4
  with:
    pattern: agenthub-*
    merge-multiple: true
```
Use a consistent file naming scheme in the uploaded files too: `agenthub-darwin-arm64`, `agenthub-linux-amd64`, `agenthub-windows-amd64.exe`.

**Warning signs:**
- GitHub Release has only one platform's binary after a multi-platform CI run
- CI logs show "artifact already exists" or earlier uploads being overwritten
- `actions/download-artifact` downloads only 1 file despite 3 platform jobs completing

**Phase to address:** Phase: GitHub Actions CI — use platform-specific artifact names from the first workflow iteration.

---

### Pitfall 14: WinGet Windows Installer Must Be an `.exe` or `.msix` — Raw Go Binary Is Rejected

**What goes wrong:**
The WinGet manifest requires `InstallerType` to be one of: `exe`, `msi`, `msix`, `appx`, `zip`, `inno`, `nullsoft`, `wix`, `burn`, `portable`. Uploading the raw Wails-produced `.exe` binary (not wrapped in an installer) requires `InstallerType: portable` and `PortableCommandAlias` to be set. If this is omitted, the manifest schema validation fails with "Invalid InstallerType" during the automated winget-pkgs PR checks.

**Why it happens:**
A raw Wails `.exe` is a self-contained binary, not an installer. Developers assume `exe` means any executable file. `InstallerType: exe` specifically means a self-extracting/setup executable that handles installation, not a portable binary.

**How to avoid:**
For a portable binary distribution:
```yaml
InstallerType: portable
PortableCommandAlias: agenthub
```
Or create a proper NSIS/WiX installer in the CI pipeline. For v1.8, `portable` is the correct choice — it tells winget to place the binary in the user's WinGet portable directory and add it to PATH.

Validate the manifest locally before submitting:
```bash
winget validate --manifest manifests/s/scottkw/AgentHub/1.8.0/
```

**Warning signs:**
- `winget validate` reports "Invalid installer type" on the manifest
- The winget-pkgs automated PR check bot reports schema validation failure
- Users report `winget install` errors about unexpected installer type

**Phase to address:** Phase: WinGet Distribution — use `InstallerType: portable` from the start; validate manifests locally before submission.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hardcoded version string in `WelcomeTab.tsx` | No tooling setup needed | Version display is wrong after every release unless manually updated | Never — annotate with `x-release-please-version` marker and add to `extra-files` |
| Single PAT shared for Homebrew tap + WinGet | Fewer secrets to manage | One PAT rotation breaks both distribution channels | Acceptable for sole maintainer if PAT expiry is tracked |
| Manual initial WinGet submission only, no automation | Avoids complexity | Must remember to run automation on next release | Never acceptable — automate v2 submission immediately after manual v1 merge |
| Using `macos-latest` instead of pinned `macos-14` | Always uses newest runner | Runner architecture changes break CI unexpectedly | Never for release builds — pin runner versions |
| ZIP of `.app` instead of DMG | Simpler CI pipeline | Less polished install UX (no drag-to-Applications) | Acceptable for initial distribution; DMG can be added later |
| Classic PAT for cross-repo operations | Works with all actions | Must be manually rotated; no auto-expiry enforcement | Acceptable — calendar reminder to rotate PAT annually |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `actions/upload-artifact` with matrix | Same artifact name across matrix legs overwrites | Use `${{ matrix.os }}-${{ matrix.arch }}` in artifact name |
| macOS `codesign` | Use `zip` to archive before notarization | Use `ditto -c -k --keepParent` — `zip` strips extended attributes |
| macOS notarization | Use `altool` (decommissioned 2023) | Use `xcrun notarytool submit --wait` exclusively |
| Apple signing in CI | Pass secrets as direct env vars to codesign | Import p12 certificate into a temporary keychain; never write cert file to disk unprotected |
| `GITHUB_TOKEN` for tap push | Assume `GITHUB_TOKEN` works for cross-repo push | Create a classic PAT with `public_repo` scope stored as a separate secret |
| release-please `release-type: go` | Expect `version-file` to be updated | Use `release-type: simple` + `extra-files` with `x-release-please-version` markers |
| WinGet `winget-releaser` action | Use fine-grained PAT | Use classic PAT with `public_repo` scope |
| Homebrew tap SHA256 | Use filename-prefixed output from `shasum` | Use bare hex: `shasum -a 256 file.zip | awk '{ print $1 }'` |
| WinGet `InstallerType` | Use `exe` for a raw portable binary | Use `InstallerType: portable` for a self-contained executable |
| Wails build in CI | Use `go build` with `-tags wailsassets` | Use `wails build` command — it handles frontend bundling, embed tags, and plist generation |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Building all 3 platforms sequentially | Release CI takes 45+ minutes | Use matrix strategy for parallel platform builds | Every release if not parallelized |
| Re-downloading Go dependencies on every CI run | Slow builds, potential rate limits | Cache `~/.cache/go-build` and `~/go/pkg/mod` with `actions/cache@v4` using go.sum hash as key | High-frequency CI runs |
| macOS notarization `--wait` polling without timeout | Job hangs indefinitely if Apple's servers have issues | Set a CI job timeout; Apple notarization normally takes <2 minutes; 10 minutes is a safe upper bound | Apple service incidents |
| WinGet submission waits for merge in CI | CI job stays open for 2+ hours waiting for human review | Fire-and-forget: submit the PR and let it be merged asynchronously; do not block release on WinGet merge | Every release with WinGet automation |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Writing the p12 certificate to disk in CI | Certificate leaked in CI logs or artifact uploads | Import directly into keychain: `security import cert.p12 -k ~/Library/Keychains/signing.keychain-db` — never write the decoded cert to a file |
| Storing Apple notarization credentials as CI secrets without app-specific password | Main Apple ID password exposure | Always use an app-specific password from https://appleid.apple.com — not the main Apple ID password |
| Using a PAT with `repo` (full) scope instead of `public_repo` | Broader than necessary access | Scope PATs to `public_repo` only for public repository operations |
| Embedding signing credentials in workflow YAML | Credentials in version control | All signing credentials must be GitHub Actions secrets, never in workflow files or committed config |
| Release artifacts not verified with SHA256 before distribution | Supply chain tampering | Generate `SHA256SUMS` file in the release job and upload it as a release asset; Homebrew cask must verify it |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| WinGet first-time install shows no progress | Users think install is frozen | `InstallerType: portable` shows minimal UI — document expected behavior in release notes |
| Homebrew `brew upgrade agenthub` removes running daemon | Running sessions terminated mid-work | Add a `caveats` stanza in the cask warning users to quit the app before upgrading |
| GitHub Release notes are auto-generated by release-please (commit-style) | Technical jargon, not user-facing changelog | Supplement release-please CHANGELOG.md with a human-written "What's new" section in the GitHub Release body |
| Notarization stapling not done — users see "damaged app" on first launch | App appears broken on download | Always run `xcrun stapler staple` after notarization and verify with `spctl --assess --type exec AgentHub.app` |

---

## "Looks Done But Isn't" Checklist

- [ ] **Git migration:** `git ls-remote --tags https://github.com/scottkw/agenthub` shows all v1.0–v1.7 tags — not just the latest.
- [ ] **CI macOS build:** The release artifact is a universal binary (`file AgentHub.app/Contents/MacOS/agenthub` shows `Mach-O universal binary with 2 architectures`), not arm64-only.
- [ ] **Signing:** `codesign -vvv --strict --verify AgentHub.app` exits 0 before notarization submission.
- [ ] **Notarization:** `spctl --assess --verbose --type exec AgentHub.app` outputs "accepted" after stapling — not before.
- [ ] **Homebrew tap:** `brew install scottkw/agenthub/agenthub` on a clean machine succeeds and the app launches.
- [ ] **Homebrew tap:** `agenthub --help` works after install (CLI binary accessible via `binary` stanza or wrapper).
- [ ] **Homebrew tap:** After a new release, `brew upgrade agenthub` fetches the new version (tap formula SHA256 and URL updated by CI).
- [ ] **WinGet:** `winget install scottkw.AgentHub` works on a clean Windows machine after the initial PR merges.
- [ ] **WinGet validation:** `winget validate --manifest manifests/s/scottkw/AgentHub/1.8.0/` exits 0 locally before PR submission.
- [ ] **release-please:** The release PR diff shows version bumps in ALL version-bearing files (Go source, WelcomeTab.tsx, CHANGELOG.md) — not just go.mod.
- [ ] **Artifact naming:** GitHub Release page for v1.8.0 shows exactly 3 platform binaries (macOS, Linux, Windows) — not 1 or 2.
- [ ] **SHA256SUMS:** SHA256SUMS file is present in the GitHub Release and values match the uploaded binaries.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Git history not migrated (tags missing) | LOW | Re-run `git push --mirror` from a bare clone; force-push tags to GitHub |
| Signing fails (plist/binary name mismatch) | LOW | Patch `Info.plist` `CFBundleExecutable` in post-build CI step; re-sign and re-notarize |
| altool used for notarization | LOW | Replace all `altool` commands with `notarytool` equivalents; no code changes required |
| Homebrew tap not updated after release | LOW | Manually update the cask formula with new SHA256 and URL; commit to tap repo |
| WinGet manifest schema rejection | MEDIUM | Fix the manifest YAML, resubmit PR; no release rollback needed; fix is same day |
| Artifact collision (only 1 platform uploaded) | LOW | Re-run the failed release job with corrected artifact names; re-attach assets to the GitHub Release |
| release-please not bumping version files | LOW | Add `x-release-please-version` markers to files; update `extra-files` config; next release will pick them up |
| Notarization staple missing | MEDIUM | Re-download the app, re-staple: `xcrun stapler staple AgentHub.app`; re-upload release asset |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Git history lost (no --mirror) | Git Migration | `git ls-remote --tags` shows v1.0–v1.7 tags on GitHub |
| Wails macOS cross-compile (wrong runner) | GitHub Actions CI | CI macOS job uses pinned `macos-14`; builds universal binary |
| Wails plist/binary name mismatch for signing | macOS Signing & Notarization | `codesign -vvv --strict --verify` exits 0 in CI |
| altool decommissioned (use notarytool) | macOS Signing & Notarization | No `altool` reference in any workflow YAML |
| Hardened Runtime breaking CGO features | macOS Signing & Notarization | Signed app tested manually before notarization submission |
| GITHUB_TOKEN cross-repo failure | Homebrew Tap | Distribution workflow actually commits to tap repo after release |
| WinGet first submission is manual | WinGet Distribution | v1.8.0 manifest merged to winget-pkgs before automation is enabled |
| WinGet PAT must be classic | WinGet Distribution | Classic PAT with `public_repo` stored as secret; workflow uses it |
| release-please version-file ignored | release-please Setup | Release PR diff shows version change in Go source and TSX files |
| Non-conventional commit history | release-please Setup | Baseline tag set manually; commitlint in CI from first PR |
| Homebrew cask needs ZIP not raw .app | Homebrew Tap | `brew install` succeeds on a clean machine from the tap |
| Homebrew binary stanza crashes app | Homebrew Tap | `agenthub --help` works from symlinked binary path |
| Artifact naming collision | GitHub Actions CI | Release has all 3 platform binaries attached |
| WinGet InstallerType: portable required | WinGet Distribution | `winget validate` passes locally; no schema errors in PR check |

---

## Sources

- Wails cross-platform build guide: https://wails.io/docs/guides/crossplatform-build/
- Wails v2.9.2 binary signing bug (CFBundleExecutable mismatch): https://github.com/wailsapp/wails/issues/3868
- macOS code signing and notarization in GitHub Actions: https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/
- Apple notarytool (altool decommissioned Fall 2023): https://developer.apple.com/news/releases/
- GitHub-hosted runner architecture (macos-14 = arm64): https://github.com/actions/runner-images/issues/9741
- release-please Go version-file bug (Issue #2541, fixed in PR #2542): https://github.com/googleapis/release-please/issues/2541
- release-please customizing with extra-files: https://github.com/googleapis/release-please/blob/main/docs/customizing.md
- GITHUB_TOKEN cross-repo access limitation: https://docs.github.com/actions/security-guides/automatic-token-authentication
- WinGet submission process and timeline: https://github.com/microsoft/winget-pkgs/discussions/19502
- WinGet releaser action (requires classic PAT): https://github.com/vedantmgoyal9/winget-releaser
- WinGet PAT permission error (classic required): https://github.com/microsoft/winget-create/issues/130
- WinGet manifest schema docs: https://learn.microsoft.com/en-us/windows/package-manager/package/manifest
- Homebrew Cask Cookbook (required fields, binary stanza risks): https://docs.brew.sh/Cask-Cookbook
- Homebrew Electron binary stanza crash issue: https://github.com/Homebrew/homebrew-cask/issues/252423
- ditto vs zip for notarization (extended attributes): https://www.kencochrane.com/2020/08/01/build-and-sign-golang-binaries-for-macos-with-github-actions/
- Automating Homebrew tap updates with GitHub Actions: https://builtfast.dev/blog/automating-homebrew-tap-updates-with-github-actions/
- Git mirror push (preserving history): https://gist.github.com/niksumeiko/8972566

---
*Pitfalls research for: AgentHub v1.8 — GitHub Distribution & CI/CD*
*Researched: 2026-04-03*
