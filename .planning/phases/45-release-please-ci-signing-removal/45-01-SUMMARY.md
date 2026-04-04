---
phase: 45-release-please-ci-signing-removal
plan: 01
subsystem: ci-cd
tags: [release-please, github-actions, ci, signing, versioning]
dependency_graph:
  requires: []
  provides: [release-please-workflow, release-please-config, version-manifest, clean-build-ci]
  affects: [.github/workflows/build.yml, .github/workflows/release-please.yml, release-please-config.json, .release-please-manifest.json]
tech_stack:
  added: [googleapis/release-please-action@v4]
  patterns: [conventional-commits, semantic-versioning, json-path-version-update]
key_files:
  created:
    - .github/workflows/release-please.yml
    - release-please-config.json
    - .release-please-manifest.json
  modified:
    - .github/workflows/build.yml
decisions:
  - "Use release-type: simple (not go) to sidestep Go version-file bug (issue #2541)"
  - "Use PAT token RELEASE_PLEASE_TOKEN so Release PRs trigger build.yml CI checks"
  - "bootstrap-sha set to v1.7 tag commit SHA to limit CHANGELOG scope"
  - "wails.json productVersion updated via JSON-path extra-file (not version-file config)"
  - "macOS signing removed from build.yml entirely — moves to release.yml in Phase 46"
metrics:
  duration: "~10 minutes"
  completed: "2026-04-04T18:02:00Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 3
  files_modified: 1
---

# Phase 45 Plan 01: Release-Please Workflow and CI Signing Removal Summary

## One-liner

release-please.yml with googleapis/release-please-action@v4, simple release-type, wails.json JSON-path versioning, bootstrapped at v1.7.0; build.yml stripped of all 4 macOS signing/notarization steps and 7-var MACOS_* env block.

## What Was Built

### Task 1: Create release-please workflow and configuration files

Created three files to wire up automated versioning via release-please:

1. **`.github/workflows/release-please.yml`** — Triggers on push to `main`, runs `googleapis/release-please-action@v4` with PAT-based token (`RELEASE_PLEASE_TOKEN`) and custom config/manifest file paths.

2. **`release-please-config.json`** — Configures `release-type: simple` for the root package, with `extra-files` pointing at `wails.json` via `$.info.productVersion` JSON-path. `bootstrap-sha` set to `7b186ce29ea24014fe4b635cb57d2563e4d67f67` (v1.7 tag commit).

3. **`.release-please-manifest.json`** — Seeds the root package version at `1.7.0` so the first Release PR targets `v1.8.0`.

Commit: `85b9909`

### Task 2: Remove macOS signing steps from build.yml

Removed from `.github/workflows/build.yml`:
- The 9-line `env:` block with 7 `MACOS_*` secret references
- `Import macOS certificate` step (6 lines)
- `Sign macOS app with hardened runtime` step (5 lines)
- `Notarize macOS app` step (8 lines)
- `Cleanup macOS keychain and certificate` step (4 lines)

Total: 44 lines deleted. Zero `MACOS_`, `codesign`, `notarytool`, `keychain`, or `certificate` references remain.

Retained unchanged: full 4-entry matrix (darwin/universal, linux/amd64 x2, windows/amd64), Go race detector tests, build script tests, Wails build step, and all 4 `upload-artifact@v4` steps.

Commit: `e9ce25c`

## Verification Results

All checks passed:
- `release-please-config.json` valid JSON with `release-type: simple` and correct `bootstrap-sha`
- `.release-please-manifest.json` valid JSON with `".": "1.7.0"`
- `.github/workflows/release-please.yml` valid YAML, `googleapis/release-please-action@v4` present
- `.github/workflows/build.yml` valid YAML, 0 MACOS_ references, 0 codesign/notarytool references
- 4 `upload-artifact@v4` steps retained
- All 4 matrix entries retained

## Deviations from Plan

None — plan executed exactly as written. The bootstrap-sha in the plan (`7b186ce29ea24014fe4b635cb57d2563e4d67f67`) matched the live `git rev-list -n 1 v1.7` output, confirming it was pre-verified.

## Known Stubs

None — no data stubs, placeholder text, or incomplete wiring.

## User Setup Required (Post-Phase)

Before release-please can create Release PRs, the following must be done in GitHub:

1. **Create `RELEASE_PLEASE_TOKEN` secret:** GitHub Settings -> Developer settings -> Personal access tokens -> Tokens (classic) -> Generate new token with `repo` scope. Add as repo-level secret at GitHub repo Settings -> Secrets and variables -> Actions -> New repository secret.

2. **Enable PR creation permissions:** GitHub repo Settings -> Actions -> General -> "Allow GitHub Actions to create and approve pull requests" -> Enable.

These are human actions requiring GitHub web UI; they cannot be automated by the executor.

## Self-Check: PASSED

Files verified:
- FOUND: /Users/ken/dev/agenthub/.github/workflows/release-please.yml
- FOUND: /Users/ken/dev/agenthub/release-please-config.json
- FOUND: /Users/ken/dev/agenthub/.release-please-manifest.json
- FOUND: /Users/ken/dev/agenthub/.github/workflows/build.yml (modified)

Commits verified:
- FOUND: 85b9909 (feat(45-01): add release-please workflow and configuration files)
- FOUND: e9ce25c (feat(45-01): remove macOS signing and notarization from build.yml)
