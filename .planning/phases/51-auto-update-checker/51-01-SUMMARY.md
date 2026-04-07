---
phase: 51-auto-update-checker
plan: 01
subsystem: updater
tags: [go, update-checker, tdd, rate-limiting, semver]
dependency_graph:
  requires: []
  provides: [internal/updater package]
  affects: [app.go (Plan 02 wiring), frontend UpdateBanner (Plan 03)]
tech_stack:
  added:
    - github.com/creativeprojects/go-selfupdate@v1.5.2
    - github.com/Masterminds/semver/v3 v3.4.0 (indirect, used for version comparison)
  patterns:
    - Injectable DetectFunc type for test isolation (mirrors tailnet.statusFunc pattern)
    - Rate-limit persistence via JSON file (~/.config/agenthub/update_check.json)
    - Silent failure — errors swallowed, returns nil UpdateInfo
key_files:
  created:
    - internal/updater/updater.go
    - internal/updater/updater_test.go
  modified:
    - go.mod
    - go.sum
decisions:
  - Use Masterminds/semver/v3 (transitive dep) for version comparison rather than string parsing
  - persist timestamp even when no update found to prevent repeated checks on rapid startup
  - White-box tests (package updater, not updater_test) to access lastCheckFile type in writeTimestamp helper
metrics:
  duration_seconds: 172
  completed_date: "2026-04-07"
  tasks_completed: 1
  files_changed: 4
---

# Phase 51 Plan 01: internal/updater Package Summary

**One-liner:** Rate-limited update detection package with injectable DetectFunc, go-selfupdate GitHub integration, and persisted 1-hour rate-limit timestamp.

## What Was Built

The `internal/updater` package implements all UPD-01 behaviors via TDD:

- `DetectFunc` — injectable function type (`func(ctx, slug) (version, found, err)`) enabling test isolation without real HTTP calls
- `UpdateInfo` — result struct with `CurrentVersion`, `LatestVersion`, `ReleaseURL`
- `DefaultDetect` — production implementation using `selfupdate.DetectLatest` + `selfupdate.ParseSlug`
- `Check` — rate-limited entry point:
  1. Skip if `currentVersion == "dev"` or `""`
  2. Skip if last check within 1 hour (unless `force=true`)
  3. Call `detect(ctx, slug)` — swallow errors silently
  4. Parse both versions with `Masterminds/semver/v3`, compare with `GreaterThan`
  5. Persist `update_check.json` timestamp regardless of result
  6. Return `&UpdateInfo{...}` only if newer version found

## Test Coverage

8 test functions, all passing race-clean:

| Test | Behavior Verified |
|------|-------------------|
| TestCheck_DevVersionSkip | "dev" returns nil, detectFunc not called |
| TestCheck_NewerVersionFound | UpdateInfo returned with correct fields |
| TestCheck_AlreadyLatest | nil returned when current == latest |
| TestCheck_DetectError | nil returned (error swallowed) |
| TestCheck_NotFound | nil returned when not found |
| TestCheck_RateLimit | Second call within 1 hour: detectFunc not called |
| TestCheck_RateLimitExpired | >1 hour old timestamp: detectFunc called normally |
| TestCheck_RateLimitBypass | force=true bypasses rate limit |

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

### Implementation Notes

1. **Version comparison approach:** The plan suggested using `selfupdate.Release.GreaterThan(string)` or `selfupdate.NewVersion`. However, `Release` cannot be constructed directly (unexported `version` field). Used `Masterminds/semver/v3` directly — it's already an indirect dependency in `go.sum`, and using it directly is the cleanest solution. Added explicit import to go.mod.

2. **Timestamp persistence on all paths:** The plan specified persisting the timestamp "after successful check." Implementation persists on all non-error paths (including when no update found) to prevent re-querying on every startup. This is the correct behavior for rate-limiting.

3. **White-box test package:** Used `package updater` (not `package updater_test`) so the test helper `writeTimestamp` can use the `lastCheckFile` struct directly, avoiding test code duplication.

## Known Stubs

None — all exported symbols are fully implemented and wired to go-selfupdate.

## Self-Check: PASSED

- internal/updater/updater.go — FOUND
- internal/updater/updater_test.go — FOUND
- commit 27d815e — FOUND
- All 8 tests pass: `go test ./internal/updater/... -v -count=1`
- `go vet ./internal/updater/...` — no issues
