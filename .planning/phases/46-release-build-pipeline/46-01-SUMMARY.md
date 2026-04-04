---
phase: 46-release-build-pipeline
plan: "01"
subsystem: ci-cd
tags: [github-actions, release, codesign, notarization, dmg, nsis, deb, nfpm]
dependency_graph:
  requires: [45-release-please-ci-signing-removal]
  provides: [release.yml, multi-platform-artifacts]
  affects: [.github/workflows/release.yml]
tech_stack:
  added: [create-dmg, nfpm, softprops/action-gh-release@v2]
  patterns: [tag-triggered workflow, parallel build jobs, artifact fan-in publish]
key_files:
  created: [.github/workflows/release.yml]
  modified: []
decisions:
  - "Manual signing after wails-build-action: wails-build-action uses different secret naming (sign-macos-app-cert pattern); manual post-build signing gives full control over DMG creation order"
  - "ubuntu-latest only for Linux release: ubuntu-22.04 build in build.yml is for validation; release ships webkit2gtk-4.1 (ubuntu-latest) artifacts as primary target"
  - "softprops/action-gh-release with GITHUB_TOKEN+permissions:contents:write: no PAT needed for uploading to existing release; PAT needed only for triggering workflows"
metrics:
  duration: 88s
  completed: "2026-04-04"
  tasks_completed: 2
  tasks_total: 2
  files_created: 1
  files_modified: 0
---

# Phase 46 Plan 01: Release Build Pipeline Summary

**One-liner:** Tag-triggered multi-platform release workflow producing signed/notarized macOS DMG, Windows NSIS installer + bare EXE, Linux tar.gz + deb, and SHA256 checksums via 4-job parallel pipeline.

## What Was Built

`.github/workflows/release.yml` — 203-line GitHub Actions workflow triggered on `v*` tag pushes (from release-please). Four parallel-then-fan-in jobs:

1. **build-macos** (`macos-latest`, `environment: release`): Uses `wails-build-action@main` to build `.app`, imports the .p12 certificate to a temp keychain, signs with hardened runtime + timestamp + entitlements, notarizes via `xcrun notarytool` with `--wait`, staples the ticket, creates a DMG with `create-dmg --codesign`. Cleanup step runs `if: always()`.

2. **build-windows** (`windows-latest`): `wails-build-action@main` with `nsis: true` and `build-webview2: embed` produces `agenthub-amd64-installer.exe` and `agenthub.exe`. A bash rename step prefixes both with `${GITHUB_REF_NAME}` following the `agenthub-{VERSION}-{platform}-{arch}.{ext}` convention.

3. **build-linux** (`ubuntu-latest`): Installs `libwebkit2gtk-4.1-dev`, builds with `build-flags: -tags webkit2_41`, creates `agenthub-{VERSION}-linux-amd64.tar.gz` via `tar`, creates `.deb` via `nfpm` with `VERSION_BARE="${GITHUB_REF_NAME#v}"` (strips leading `v` — Pitfall 4 from research).

4. **publish** (`ubuntu-latest`, `needs: [build-macos, build-windows, build-linux]`): Downloads all 3 artifact sets with `merge-multiple: true`, generates `checksums.txt` via `sha256sum *`, uploads everything to the GitHub Release via `softprops/action-gh-release@v2` with `fail_on_unmatched_files: true`.

## Artifact Naming Convention

| Artifact | Pattern |
|----------|---------|
| macOS DMG | `agenthub-v1.8.0-darwin-universal.dmg` |
| Windows installer | `agenthub-v1.8.0-windows-amd64-installer.exe` |
| Windows EXE | `agenthub-v1.8.0-windows-amd64.exe` |
| Linux tar.gz | `agenthub-v1.8.0-linux-amd64.tar.gz` |
| Linux deb | `agenthub-v1.8.0-linux-amd64.deb` |
| Checksums | `checksums.txt` |

## Requirements Satisfied

- **REL-02:** Multi-platform artifacts produced on tag push (macOS DMG, Windows EXE + NSIS, Linux tar.gz + deb)
- **REL-04:** SHA256 checksums file attached to each GitHub Release

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Manual signing post-wails-build-action | wails-build-action sign inputs use different secret naming; manual control needed for correct sign-notarize-staple-DMG order |
| ubuntu-latest only for Linux release | ubuntu-22.04 in build.yml is for CI validation; webkit2gtk-4.1 is the current standard |
| permissions:contents:write on publish job | GITHUB_TOKEN with explicit write permission suffices for uploading to existing release; no PAT needed |

## Pitfalls Avoided (from Research)

1. `environment: release` declared — all 7 MACOS_* secrets accessible
2. DMG created AFTER notarize+staple — no unstapled app inside DMG
3. `ditto -c -k --keepParent` used (not `zip -r`) for notarization zip
4. `${GITHUB_REF_NAME#v}` strips leading `v` for nfpm version field
5. `--timestamp` flag on codesign — Trusted Timestamp Authority record present
6. `security set-key-partition-list` present — no "User interaction not allowed" on CI
7. `build-flags: -tags webkit2_41` on Linux — matches ubuntu-latest webkit API

## Deviations from Plan

None — plan executed exactly as written. Research skeleton implemented faithfully.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | 2082e03 | feat(46-01): create release.yml multi-platform release workflow |
| 2 | (no file change) | Validation-only task; release.yml passed all YAML and anti-pattern checks |

## Self-Check

- [x] `.github/workflows/release.yml` exists (203 lines, >= 120)
- [x] YAML parses without errors (PyYAML; `on:` key is boolean True — known YAML quirk, valid for GitHub Actions)
- [x] 4 jobs present: build-macos, build-windows, build-linux, publish
- [x] build-macos has `environment: release`
- [x] publish needs all 3 build jobs
- [x] All 7 MACOS_* secrets referenced
- [x] Correct signing order: import cert → sign → notarize → staple → DMG
- [x] `ditto` used (not `zip -r`)
- [x] `--timestamp` on codesign
- [x] `security set-key-partition-list` present
- [x] `webkit2_41` tag on Linux
- [x] nfpm `${GITHUB_REF_NAME#v}` strips v-prefix
- [x] `softprops/action-gh-release@v2` with `fail_on_unmatched_files: true`
- [x] `sha256sum * > checksums.txt` in publish job
- [x] `merge-multiple: true` on download-artifact

## Self-Check: PASSED
