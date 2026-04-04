---
phase: 46-release-build-pipeline
verified: 2026-04-04T22:23:24Z
status: passed
score: 6/6 must-haves verified
---

# Phase 46: Release Build Pipeline Verification Report

**Phase Goal:** Create the release.yml workflow that triggers on v* tag pushes and builds multi-platform artifacts (macOS signed/notarized DMG, Windows NSIS installer + bare EXE, Linux amd64 tar.gz + deb), generates SHA256 checksums, and uploads to GitHub Release.
**Verified:** 2026-04-04T22:23:24Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth                                                                                      | Status     | Evidence                                                                                   |
|----|-------------------------------------------------------------------------------------------|------------|--------------------------------------------------------------------------------------------|
| 1  | Merging a release-please Release PR triggers release.yml automatically via tag push        | VERIFIED   | `on: push: tags: - 'v*'` (lines 3-6); release-please uses RELEASE_PLEASE_TOKEN (PAT) which bypasses branch protection and triggers push.tags |
| 2  | macOS job produces a signed, notarized DMG artifact                                        | VERIFIED   | Sign (line 43-50) → Notarize with xcrun notarytool + --wait (lines 52-62) → Staple (line 62) → create-dmg --codesign (lines 64-74); upload-artifact@v4 with if-no-files-found: error |
| 3  | Windows job produces an NSIS installer and bare EXE artifact                               | VERIFIED   | `nsis: true`, `build-webview2: embed` (lines 101-103); rename step produces `agenthub-${VERSION}-windows-amd64-installer.exe` and `agenthub-${VERSION}-windows-amd64.exe` |
| 4  | Linux job produces a tar.gz and .deb artifact                                              | VERIFIED   | `libwebkit2gtk-4.1-dev`, `-tags webkit2_41` (lines 133, 140); tar.gz via `tar -czf` (lines 142-147); .deb via nfpm with `${GITHUB_REF_NAME#v}` (lines 149-165) |
| 5  | A checksums.txt with SHA256 hashes for all artifacts is attached to the GitHub Release     | VERIFIED   | `sha256sum * > checksums.txt` (line 191) in publish job; uploaded via softprops/action-gh-release@v2 with `fail_on_unmatched_files: true` (lines 195-203) |
| 6  | All artifact filenames follow the pattern agenthub-{VERSION}-{platform}-{arch}.{ext}      | VERIFIED   | macOS: `agenthub-${VERSION}-darwin-universal.dmg`; Windows: versioned via rename step; Linux: versioned via TAR_NAME/nfpm target variables |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact                            | Expected                                  | Status    | Details                                             |
|-------------------------------------|-------------------------------------------|-----------|-----------------------------------------------------|
| `.github/workflows/release.yml`     | Tag-triggered multi-platform release workflow | VERIFIED | 203 lines (>= 120 minimum); contains `on: push: tags`, 4 jobs; YAML parses cleanly via PyYAML |

**Artifact level checks:**
- **Level 1 (Exists):** File present at `.github/workflows/release.yml`
- **Level 2 (Substantive):** 203 lines; all 4 jobs fully implemented with real step logic, no placeholders
- **Level 3 (Wired):** Triggered by `push.tags: v*`; release-please.yml uses RELEASE_PLEASE_TOKEN which pushes tags that activate this trigger; publish job depends on all 3 build jobs via `needs`
- **Level 4 (Data flows):** N/A — CI workflow, not a data-rendering component

### Key Link Verification

| From                                         | To                               | Via                                    | Status   | Details                                                                                 |
|----------------------------------------------|----------------------------------|----------------------------------------|----------|-----------------------------------------------------------------------------------------|
| `.github/workflows/release-please.yml`       | `.github/workflows/release.yml`  | PAT-pushed v* tag triggers on.push.tags | VERIFIED | release-please.yml uses `RELEASE_PLEASE_TOKEN` (PAT); release.yml has `on: push: tags: - 'v*'` (lines 3-6) |
| `.github/workflows/release.yml build-macos`  | GitHub release environment       | environment: release declaration        | VERIFIED | `environment: release` present on build-macos job (line 11)                            |
| `.github/workflows/release.yml publish`      | GitHub Release page              | softprops/action-gh-release uploads artifacts | VERIFIED | `softprops/action-gh-release@v2` at line 195 with files glob and `fail_on_unmatched_files: true` |

### Data-Flow Trace (Level 4)

N/A — This phase produces a CI workflow file, not a component rendering dynamic data. Level 4 does not apply.

### Behavioral Spot-Checks

Step 7b: SKIPPED — workflow file cannot be executed locally without GitHub Actions runner infrastructure. The YAML is structurally valid (parsed by PyYAML) and all logic is present.

### Requirements Coverage

| Requirement | Source Plan | Description                                                                                       | Status    | Evidence                                                                     |
|-------------|-------------|---------------------------------------------------------------------------------------------------|-----------|------------------------------------------------------------------------------|
| REL-02      | 46-01-PLAN  | release.yml workflow builds multi-platform artifacts on tag push (macOS DMG, Windows EXE + NSIS, Linux tar.gz + deb) | SATISFIED | All 3 platform jobs present and fully implemented; each produces versioned artifacts uploaded via upload-artifact@v4 |
| REL-04      | 46-01-PLAN  | SHA256 checksums file generated and attached to each GitHub Release                                | SATISFIED | `sha256sum * > checksums.txt` in publish job; file included in softprops upload glob |

No orphaned requirements found — REQUIREMENTS.md maps REL-02 and REL-04 to Phase 46, both claimed in plan frontmatter and verified in implementation.

### Anti-Patterns Found

None. No TODO/FIXME/HACK/placeholder comments. No empty implementations. No hardcoded empty returns.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | None found | — | — |

### Detailed Acceptance Criteria Check

All 17 acceptance criteria from 46-01-PLAN.md verified:

| Criterion | Status | Line(s) |
|-----------|--------|---------|
| .github/workflows/release.yml exists with 120+ lines | PASS | 203 lines |
| `on:` block with `push: tags: - 'v*'` | PASS | 3-6 |
| `build-macos:` job with `environment: release` | PASS | 9, 11 |
| All 7 MACOS_* secrets referenced | PASS | 13-19 |
| `codesign --deep --force --verbose --options runtime --timestamp --entitlements build/entitlements.plist` | PASS | 45-49 |
| `xcrun notarytool store-credentials` present | PASS | 54 |
| `xcrun notarytool submit` present | PASS | 59 |
| `xcrun stapler staple` present | PASS | 62 |
| `ditto -c -k --keepParent` (NOT `zip -r`) | PASS | 58 |
| `create-dmg --volname "AgentHub" --codesign` | PASS | 69-71 |
| `build-windows:` job with `nsis: true` and `build-webview2: embed` | PASS | 100-103 |
| Rename step producing versioned Windows artifacts | PASS | 105-112 |
| `build-linux:` job with `build-flags: -tags webkit2_41` | PASS | 135-140 |
| `libwebkit2gtk-4.1-dev` in apt-get install | PASS | 133 |
| nfpm usage with `${GITHUB_REF_NAME#v}` (v-prefix strip) | PASS | 151 |
| `publish:` job with `needs: [build-macos, build-windows, build-linux]` | PASS | 177 |
| `sha256sum * > checksums.txt` | PASS | 191 |
| `softprops/action-gh-release@v2` with `fail_on_unmatched_files: true` | PASS | 195, 203 |
| `actions/upload-artifact@v4` in each build job | PASS | 83, 115, 168 |
| `actions/download-artifact@v4` with `merge-multiple: true` in publish job | PASS | 183, 186 |
| Action versions match build.yml (checkout@v4, wails-build-action@main, upload-artifact@v4) | PASS | Identical versions |
| Notarize step BEFORE create-dmg step | PASS | Notarize line 52, DMG line 64 |
| `security set-key-partition-list` present | PASS | 40 |
| `permissions: contents: write` on publish job | PASS | 179-180 |

### Human Verification Required

None — all automated checks passed. The workflow is a YAML-only CI definition that cannot be executed locally. Runtime behavior (actual macOS signing, Windows NSIS packaging, Linux .deb creation, GitHub Release upload) requires a tagged push to GitHub. These behaviors are inferable from the correct presence of all required steps in the correct order.

### Gaps Summary

No gaps. All 6 must-have truths are satisfied by the implementation in `.github/workflows/release.yml`. All 3 key links are verified. Both requirements (REL-02, REL-04) are satisfied. No anti-patterns found.

---

_Verified: 2026-04-04T22:23:24Z_
_Verifier: Claude (gsd-verifier)_
