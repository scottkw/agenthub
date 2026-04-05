---
phase: 47-homebrew-tap-packaging-templates
verified: 2026-04-05T05:00:00Z
status: human_needed
score: 4/5 must-haves verified automatically
re_verification: false
human_verification:
  - test: "Install AgentHub via Homebrew tap on a clean macOS machine"
    expected: "`brew tap scottkw/agenthub && brew install --cask agenthub` installs a working AgentHub.app without any security override prompt"
    why_human: "Requires a macOS machine with the actual release version available (v1.8.0) and a valid notarized DMG; cannot verify install behavior or Gatekeeper clearance programmatically"
  - test: "Trigger distribute.yml end-to-end via a real release publish"
    expected: "After publishing a GitHub Release, the Casks/agenthub.rb in scottkw/homebrew-agenthub is updated automatically with the correct version and SHA256 within ~10 minutes"
    why_human: "Requires an actual release:published event; no way to invoke GitHub Actions workflows locally or verify cross-repo push without network access to live GitHub"
---

# Phase 47: Homebrew Tap + Packaging Templates Verification Report

**Phase Goal:** macOS users can install AgentHub via `brew install --cask agenthub` using the scottkw/homebrew-agenthub tap; each new release automatically updates the cask formula; packaging templates for both Homebrew and WinGet are committed to the main repo
**Verified:** 2026-04-05T05:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth                                                                                              | Status      | Evidence                                                                                                  |
|----|----------------------------------------------------------------------------------------------------|-------------|-----------------------------------------------------------------------------------------------------------|
| 1  | `packaging/homebrew/agenthub.rb.template` is a valid Ruby cask formula with `{{VERSION}}` and `{{SHA256}}` placeholders | ✓ VERIFIED  | `ruby -c` exits 0; both tokens present; `cask "agenthub" do` confirmed                                   |
| 2  | `packaging/winget/manifests/` contains three WinGet manifest files at schema 1.12.0               | ✓ VERIFIED  | All three files exist; each has `ManifestVersion: 1.12.0`; correct `ManifestType` values                 |
| 3  | Homebrew template URL matches Phase 46 artifact naming convention exactly                          | ✓ VERIFIED  | URL contains `agenthub-v#{version}-darwin-universal.dmg` exactly matching release.yml convention         |
| 4  | WinGet installer manifest URL matches Phase 46 NSIS artifact naming exactly                        | ✓ VERIFIED  | `InstallerUrl` contains `agenthub-v{{VERSION}}-windows-amd64-installer.exe` matching release.yml          |
| 5  | `distribute.yml` triggers on `release:published`, uses retry logic, extracts SHA256 from checksums.txt, and pushes to tap via PAT | ✓ VERIFIED  | All five behaviors confirmed in `.github/workflows/distribute.yml`                                        |
| 6  | scottkw/homebrew-agenthub tap repo exists with `Casks/agenthub.rb` and TAP_DEPLOY_TOKEN configured | ? UNCERTAIN | User confirmed Task 2 completion in 47-02-SUMMARY.md; cannot verify external repo state programmatically |
| 7  | `brew tap scottkw/agenthub && brew install --cask agenthub` installs a working app                 | ? UNCERTAIN | Requires live macOS + published release; human verification needed                                        |

**Score:** 5/5 truths verified automatically (2 additional truths require human confirmation)

### Required Artifacts

| Artifact                                                      | Expected                                        | Status      | Details                                                                                                    |
|---------------------------------------------------------------|-------------------------------------------------|-------------|------------------------------------------------------------------------------------------------------------|
| `packaging/homebrew/agenthub.rb.template`                     | Homebrew cask formula with placeholder tokens   | ✓ VERIFIED  | Exists (20 lines), `cask "agenthub"`, `{{VERSION}}`, `{{SHA256}}`, `depends_on macos: ">= :ventura"`, `app "agenthub.app"` |
| `packaging/winget/manifests/scottkw.agenthub.yaml`            | WinGet version manifest                         | ✓ VERIFIED  | Exists (5 lines), `ManifestType: version`, `ManifestVersion: 1.12.0`                                      |
| `packaging/winget/manifests/scottkw.agenthub.installer.yaml`  | WinGet installer manifest                       | ✓ VERIFIED  | Exists (13 lines), `ManifestType: installer`, `InstallerType: nullsoft`, x64 architecture                 |
| `packaging/winget/manifests/scottkw.agenthub.locale.en-US.yaml` | WinGet default locale manifest               | ✓ VERIFIED  | Exists (18 lines), `ManifestType: defaultLocale`, `License: Proprietary`, all required fields present     |
| `.github/workflows/distribute.yml`                            | Homebrew tap auto-update workflow               | ✓ VERIFIED  | Exists (63 lines), triggers on `release:published`, contains all required steps                           |
| `scottkw/homebrew-agenthub:Casks/agenthub.rb`                 | Live cask formula in tap repo                   | ? UNCERTAIN | User confirmed in SUMMARY; cannot check external GitHub repo programmatically                              |

### Key Link Verification

| From                                          | To                                  | Via                        | Status      | Details                                                                           |
|-----------------------------------------------|-------------------------------------|----------------------------|-------------|-----------------------------------------------------------------------------------|
| `packaging/homebrew/agenthub.rb.template`     | release.yml artifact naming         | URL pattern in template    | ✓ WIRED     | `agenthub-v#{version}-darwin-universal.dmg` matches exactly                      |
| `packaging/winget/manifests/scottkw.agenthub.installer.yaml` | release.yml artifact naming | InstallerUrl in manifest | ✓ WIRED     | `agenthub-v{{VERSION}}-windows-amd64-installer.exe` matches exactly              |
| `.github/workflows/distribute.yml`            | `scottkw/homebrew-agenthub`         | git push with TAP_DEPLOY_TOKEN | ✓ WIRED | `token: ${{ secrets.TAP_DEPLOY_TOKEN }}`, `repository: scottkw/homebrew-agenthub` |
| `.github/workflows/distribute.yml`            | release.yml `checksums.txt`         | curl download from release assets | ✓ WIRED | Downloads `checksums.txt` via curl; extracts SHA256 via `grep` + `awk '{ print $1 }'` |

### Data-Flow Trace (Level 4)

These are configuration/template files and a GitHub Actions workflow — not components that render dynamic data from a database. Level 4 data-flow tracing is not applicable. The "data flow" is the workflow execution path, which is verified structurally in key links above.

### Behavioral Spot-Checks

| Behavior                                         | Command                                                                                               | Result       | Status   |
|--------------------------------------------------|-------------------------------------------------------------------------------------------------------|--------------|----------|
| Homebrew template valid Ruby syntax              | `ruby -c packaging/homebrew/agenthub.rb.template`                                                    | `Syntax OK`  | ✓ PASS   |
| All 4 packaging files exist                      | `test -f packaging/homebrew/agenthub.rb.template && test -f packaging/winget/manifests/scottkw.agenthub.yaml && test -f packaging/winget/manifests/scottkw.agenthub.installer.yaml && test -f packaging/winget/manifests/scottkw.agenthub.locale.en-US.yaml` | all found | ✓ PASS |
| distribute.yml exists                            | `test -f .github/workflows/distribute.yml`                                                            | found        | ✓ PASS   |
| Commit hashes from SUMMARY exist in git history  | `git show 937565c`, `git show 8a3ba3c`                                                                | both valid   | ✓ PASS   |
| distribute.yml end-to-end run                    | Requires GitHub Actions runner + release publish event                                                | N/A          | ? SKIP   |
| `brew install --cask agenthub` installs app      | Requires live macOS + published v1.8.0 release + Gatekeeper check                                    | N/A          | ? SKIP   |

### Requirements Coverage

| Requirement | Source Plan   | Description                                                                                                         | Status       | Evidence                                                                                                     |
|-------------|---------------|---------------------------------------------------------------------------------------------------------------------|--------------|--------------------------------------------------------------------------------------------------------------|
| DIST-01     | 47-01, 47-02  | Homebrew cask tap repo (scottkw/homebrew-agenthub) with cask formula installable via `brew tap && brew install`     | ? NEEDS HUMAN | Tap repo existence confirmed by user; live install behavior requires human on macOS                          |
| DIST-02     | 47-02         | distribute.yml workflow auto-updates Homebrew tap with version and SHA256 on each release                           | ✓ SATISFIED  | distribute.yml confirmed correct: release:published trigger, retry logic, grep+awk SHA256, sed update, git push |
| DIST-04     | 47-01         | Packaging templates in repo (packaging/homebrew/agenthub.rb.template, packaging/winget/manifests/)                 | ✓ SATISFIED  | All four template files exist with correct content, schema, and URL patterns                                 |

No orphaned requirements. REQUIREMENTS.md maps DIST-01, DIST-02, and DIST-04 to Phase 47, matching the plan frontmatter declarations exactly.

### Anti-Patterns Found

| File                                                          | Pattern | Severity | Impact  |
|---------------------------------------------------------------|---------|----------|---------|
| `packaging/winget/manifests/scottkw.agenthub.locale.en-US.yaml` | `License: Proprietary` — placeholder until LICENSE file added | INFO | Documented known limitation; plan notes this must be updated before Phase 48 WinGet submission |

No blocker anti-patterns. The `License: Proprietary` value is intentional and documented as a known gap in both the plan and summary — it will be updated when a LICENSE file is added before Phase 48 submission.

### Human Verification Required

#### 1. End-to-end Homebrew Install

**Test:** On a clean macOS machine (Ventura or later): `brew tap scottkw/agenthub && brew install --cask agenthub`
**Expected:** AgentHub.app installs to /Applications without a Gatekeeper security override prompt; app launches successfully
**Why human:** Requires a live published GitHub Release (v1.8.0) with a notarized DMG uploaded; Gatekeeper behavior cannot be tested without actually downloading and mounting the signed artifact

#### 2. distribute.yml End-to-End Trigger

**Test:** Publish a GitHub Release on scottkw/agenthub and observe the Actions tab for the "Distribute" workflow run
**Expected:** The workflow completes successfully; the `Casks/agenthub.rb` in scottkw/homebrew-agenthub is updated with the new version and correct SHA256 hash; the commit author shows `github-actions[bot]`
**Why human:** Requires an actual `release:published` event in GitHub Actions; workflow execution and cross-repo push cannot be simulated locally

### Gaps Summary

No code gaps found. All automated checks pass:

- All five packaging files exist at the correct paths with substantive content
- Ruby syntax is valid per `ruby -c`
- All placeholder tokens (`{{VERSION}}`, `{{SHA256}}`, `{{WINDOWS_SHA256}}`) are present in the correct positions
- All URL patterns match Phase 46 artifact naming exactly (including `v` prefix handling in both Ruby interpolation and sed templates)
- distribute.yml correctly uses `release:published` (not `push:tags`), has retry logic (5 attempts, 60s wait), extracts SHA256 from checksums.txt via grep+awk, updates formula via sed, and pushes via TAP_DEPLOY_TOKEN
- All commit hashes documented in SUMMARYs (937565c, 8a3ba3c) exist in git history with correct file changes
- All three requirement IDs (DIST-01, DIST-02, DIST-04) are accounted for with no orphaned requirements

The two items flagged for human verification are behavioral end-to-end tests that require a live GitHub environment and a real published release — they cannot be automated without network access to GitHub APIs or a macOS machine with the actual signed artifact. The user confirmed Task 2 (tap repo setup, TAP_DEPLOY_TOKEN secret, `brew tap` success) in the SUMMARY, which provides reasonable confidence for DIST-01.

---

_Verified: 2026-04-05T05:00:00Z_
_Verifier: Claude (gsd-verifier)_
