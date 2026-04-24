---
phase: 90
plan: "03"
subsystem: ci-hardening
tags: [ci, sha-pin, sec-09, sec-10, build-yml, release-please, build-sh, wave-2]
dependency_graph:
  requires: [90-02]
  provides: [sha-pinned-build-yml, sha-pinned-release-please, sec10-compliant-buildsh]
  affects: [90-04, 90-05]
tech_stack:
  added: []
  patterns:
    - go-list-m-wails-version-derivation
    - sha-pinned-github-actions
    - pitfall-5-empty-string-gate
key_files:
  created: []
  modified:
    - .github/workflows/build.yml
    - .github/workflows/release-please.yml
    - build.sh
decisions:
  - "v4→v7 for upload-artifact accepted per A1: plain file uploads are backward-compatible; build.yml is not the release pipeline"
  - "release-please-action bumped from @v4 to v5.0.0 SHA simultaneously with SHA-pin — avoids immediate follow-up PR; v5 preserves config-file + manifest-file input names"
  - "WAILS_PINNED_VER gate warns (not errors) on version mismatch — wails version output format varies; the gate correctly aborts on missing go.mod entry only"
metrics:
  duration: "~5min"
  completed: "2026-04-24T13:12:10Z"
  tasks_completed: 3
  files_modified: 3
  commits: 3
---

# Phase 90 Plan 03: SHA-pin build.yml + release-please.yml + build.sh SUMMARY

**One-liner:** SHA-pinned all 8 `uses:` lines in build.yml and 1 in release-please.yml to immutable 40-char SHAs, replaced `wails@latest` with `go list -m`-derived version + Pitfall-5 gate in both build.yml and build.sh (SEC-09 + SEC-10).

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | SHA-pin build.yml + wails install | cf8413e | .github/workflows/build.yml |
| 2 | SHA-pin release-please.yml | 8543633 | .github/workflows/release-please.yml |
| 3 | build.sh @latest → go-list + WAILS_PINNED_VER gate | e0ce067 | build.sh |

## Changes Made

### .github/workflows/build.yml (8 SHA pins + 1 install swap)

| Action | Before | After |
|--------|--------|-------|
| actions/checkout | @v4 | @de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2 |
| actions/setup-go | @v5 | @4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0 |
| pnpm/action-setup | @v4 | @903f9c1a6ebcba6cf41d87230be49611ac97822e # v6.0.3 |
| actions/setup-node | @v4 | @48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6.4.0 |
| actions/upload-artifact (x4) | @v4 | @043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1 |
| Install Wails CLI | go install wails@latest | go list -m derived + Pitfall-5 gate |

### .github/workflows/release-please.yml (1 SHA pin + minor version bump)

| Action | Before | After |
|--------|--------|-------|
| googleapis/release-please-action | @v4 (floating) | @45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0 |

### build.sh (lines 61-67 replaced with 16-line block)

Old block: `WAILS` path check with `@latest` install hint.

New block:
1. `WAILS_PINNED_VER` derived via `go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2`
2. Empty-string gate: aborts with explanatory error if wails not in go.mod (Pitfall-5 defense)
3. Binary existence check with pinned-version install hint (no @latest)
4. Best-effort version sanity check (WARN only — not blocking)

## Test Results

### Section 12 (Plan 01 red tests → now green)

```
=== Section 12: SEC-10 compliance — no @latest refs ===
  PASS: build.sh contains no @latest references (SEC-10)
  PASS: build.sh contains go list -m pin pattern
  PASS: build.sh gates on WAILS_PINNED_VER

Results: 38 passed, 0 failed
```

All three previously-red Section 12 assertions are now green.

### Grep-gate state after this plan

- build.yml: PASSES (zero floating refs)
- release-please.yml: PASSES (zero floating refs)
- release.yml: FAILS (owned by Plan 04)
- distribute.yml: FAILS (owned by Plan 05)

Grep-gate overall still exits non-zero — expected partial progress. Will pass after Plans 04 + 05 complete.

## Deviations from Plan

None — plan executed exactly as written.

## Handoff to Plan 04

Plan 04 can start immediately. No file overlap:
- Plan 03 files: build.yml, release-please.yml, build.sh
- Plan 04 files: release.yml (large multi-job workflow)

## Self-Check

| Check | Result |
|-------|--------|
| .github/workflows/build.yml exists | FOUND |
| .github/workflows/release-please.yml exists | FOUND |
| build.sh exists | FOUND |
| commit cf8413e exists | FOUND |
| commit 8543633 exists | FOUND |
| commit e0ce067 exists | FOUND |
| grep-gate fails only on release.yml + distribute.yml | CONFIRMED |
| build-script.test.sh: 38 passed, 0 failed | CONFIRMED |

## Self-Check: PASSED
