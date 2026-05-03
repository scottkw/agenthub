---
phase: "90"
plan: "04"
subsystem: ci-release-pipeline
tags: [ci, hardening, release-yml, job-split, attestation, slsa-l2, sec-09, sec-10, sec-11]
dependency_graph:
  requires: [90-02, 90-03]
  provides: [sign-macos-job, internal-attestation, release-attestation, secret-scoping]
  affects: [.github/workflows/release.yml]
tech_stack:
  added: [actions/attest-build-provenance@v4.1.0, actions/download-artifact@v8.0.1, softprops/action-gh-release@v3.0.0]
  patterns: [slsa-l2-attestation, job-secret-scoping, tar-app-bundle-round-trip, go-list-m-pin, rc-draft-guard]
key_files:
  modified: [.github/workflows/release.yml]
decisions:
  - "All three tasks executed as a single Write to release.yml (atomic); YAML validated before commit"
  - "attest-build-provenance appears 4 times in uses: lines (3 build jobs + publish) — 7 total grep hits because 3 are in inline comments"
  - "sign-macos uses permissions: contents: read (minimal) for offline --bundle verify per RESEARCH Open Q2"
  - "checksums step uses explicit globs (*.dmg *.exe *.tar.gz *.deb) not sha256sum * — excludes attestation bundle .json files that merge-multiple flattens into artifacts/"
metrics:
  duration: "3 minutes"
  completed: "2026-04-24"
  tasks_completed: 3
  tasks_total: 3
  files_modified: 1
---

# Phase 90 Plan 04: release.yml Three-Stage Split Summary

**One-liner:** Split release.yml into secret-isolated build → sign-macos → publish pipeline with SLSA L2 internal + release attestations, SHA-pinned actions, rc-draft guard, and TAP_DEPLOY_TOKEN fix.

## What Was Built

`.github/workflows/release.yml` restructured from a two-stage pipeline (build-macos with signing secrets + publish) into a five-job pipeline:

```
validate → build-macos │
          build-windows │ (parallel, all secret-free)
          build-linux   │
                → sign-macos (environment: release, MACOS_* only)
                        → publish (GITHUB_TOKEN + attestation scopes)
```

### Task 1: Strip secrets from build-* jobs, SHA-pin, tar .app, internal attestation

**build-macos changes:**
- Removed `environment: release` and entire `env:` block (7 MACOS_* secrets) — SEC-11 core: zero secret access during untrusted build
- Added `permissions: id-token: write, attestations: write, contents: read`
- Replaced `wails@latest` install with `go list -m` pattern (SEC-10)
- Added tar step: `tar -czf build/bin/AgentHub.app.tar.gz -C build/bin AgentHub.app` (Pitfall 1 — preserves symlinks and +x bits across upload-artifact boundary)
- Added `attest-build-provenance@a2bbfa25...` step attesting the tarball
- Upload now includes both tarball and attestation bundle as `agenthub-darwin-universal-unsigned`
- Removed all codesign/notarize/DMG/cleanup steps (moved to sign-macos)

**build-windows changes:**
- Added `permissions:` block (id-token, attestations, contents)
- Replaced `wails@latest` with `go list -m` pattern
- Added internal attestation on `.exe` files after rename
- Upload includes attestation bundle

**build-linux changes:**
- Added `permissions:` block
- Replaced `wails@latest` with `go list -m` pattern
- Split nfpm install into its own step using `go list -m` (was `nfpm@latest` inline in deb-create step)
- Added internal attestation on `.tar.gz` + `.deb` files
- Upload includes attestation bundle

**validate:** SHA-pinned all 4 actions.

### Task 2: Add sign-macos job (SEC-11 D-01 + D-04)

New job inserted between `build-linux` and `publish`:
- `environment: release` — ONLY on this job (not build-macos, not publish)
- `env:` contains ONLY the 7 MACOS_* secrets
- `permissions: contents: read` — sufficient for offline `--bundle` verify
- Downloads `agenthub-darwin-universal-unsigned` artifact (tarball + attestation bundle)
- Verifies attestation: `gh attestation verify artifacts/AgentHub.app.tar.gz --bundle "$BUNDLE"` before untar (T-90-18 mitigation)
- Untars: `tar -xzf artifacts/AgentHub.app.tar.gz -C build/bin/` (restores symlinks + +x)
- codesign / notarize / DMG / cleanup steps moved verbatim from old build-macos location
- Uploads `macos-dmg` artifact (signed DMG only)

### Task 3: Restructure publish job (SEC-09 + SEC-11 D-02 + D-05 + D-15)

- `needs:` extended to `[build-macos, build-windows, build-linux, sign-macos]`
- `permissions:` extended with `id-token: write` and `attestations: write`
- checksums step: `sha256sum *.dmg *.exe *.tar.gz *.deb` (explicit globs exclude attestation bundle .json files)
- New step: `Release build-provenance attestation` via `subject-checksums: artifacts/checksums.txt` (D-05 SLSA L2 public proof)
- `softprops/action-gh-release@v2` → SHA-pinned `@b4309332981a82ec1c5618f44dd2e27cc8bfbfda # v3.0.0`
- `draft: ${{ contains(github.ref, '-rc') }}` — hyphen-anchored (D-15; guards against `v3.1.0-archive` false-matches)
- `token: ${{ secrets.GITHUB_TOKEN }}` — was `TAP_DEPLOY_TOKEN` (D-02 fix; TAP_DEPLOY_TOKEN stays in distribute.yml where it belongs)
- `download-artifact@v4` → SHA-pinned `@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1`

## Verification Results

All acceptance criteria passed:

| Check | Result |
|-------|--------|
| YAML valid | PASS |
| `@latest` count = 0 | PASS |
| `attest-build-provenance` uses: count = 4 | PASS |
| `go list -m wails` count >= 3 | PASS (3) |
| `go list -m nfpm` count = 1 | PASS |
| `MACOS_CERTIFICATE:` in build-macos | PASS (0 — only in sign-macos) |
| `environment: release` count = 1 | PASS |
| `gh attestation verify` count = 1 | PASS |
| `TAP_DEPLOY_TOKEN` count = 0 | PASS |
| `contains(github.ref, '-rc')` present | PASS |
| `subject-checksums: artifacts/checksums.txt` present | PASS |
| `softprops@b4309332...` present | PASS |
| All `uses:` are 40-char SHA | PASS |
| grep-gate.sh checks against release.yml | PASS (all 3 checks) |
| codesign --deep appears exactly once | PASS |
| sign-macos job exists | PASS |

## Grep-Gate State After This Plan

- `release.yml`: PASS (all 3 gate checks pass)
- `distribute.yml`: STILL FAILS (Plan 05's territory — winget-releaser@main + other floating refs)
- `bash scripts/grep-gate.sh` (full run): FAILS on distribute.yml — expected until Wave 4

## Deviations from Plan

None — plan executed exactly as written. The three tasks were implemented as a single atomic Write to release.yml (rather than three separate edits) because the interdependencies between job additions and removals required seeing the complete file structure. The YAML was validated before commit.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. The threat mitigations from the plan's STRIDE register are implemented:
- T-90-17: build-macos has empty env: and no environment: declaration
- T-90-18: gh attestation verify --bundle before untar in sign-macos
- T-90-20: Release attestation via subject-checksums in publish
- T-90-22: TAP_DEPLOY_TOKEN removed from publish; GITHUB_TOKEN used

## Handoff to Plan 05

`distribute.yml` is untouched. Plan 05 (Wave 4) can proceed immediately. The signed DMG artifact from sign-macos (`macos-dmg`) and the other platform artifacts flow into publish correctly.

## Self-Check: PASSED

- `.github/workflows/release.yml` exists and is YAML-valid
- Commit `ec69aa6` exists in git log
- sign-macos job at line 288 with `environment: release`
- 4 `uses: actions/attest-build-provenance@...` lines (98, 173, 269, 410)
- `environment: release` appears exactly once (line 288)
