---
phase: 124-files-write-capability-webserver-write-routes-web-share-opt-in
plan: 01
subsystem: webserver capability middleware
tags: [capability, middleware, csrf, go, security, access-control]
requirements: [CAP-01, CAP-02, CAP-03, CAP-07, CAP-09]

dependency_graph:
  requires:
    - internal/files/write.go (Phase 123 — Handler.Write/Upload/Delete/Rename/Mkdir)
    - internal/capability/capability.go (HasPerm, Claims, PermFilesRead)
    - internal/webserver/capability_mw.go (requireCapability, requireFilesRead)
    - internal/webserver/origin_mw.go (requireAllowedOrigin pattern, then inverted)
  provides:
    - PermFilesWrite constant in capability package
    - requireFilesWrite middleware (third separate wrapper: requireCapability → HasPerm → Origin)
    - originAllowedForWrite helper (CSRF: absent Origin passes, present must match BaseURL)
    - Five webserver write routes behind requireFilesWrite
  affects:
    - internal/webserver/server.go (five new route mounts)
    - internal/webserver/capability_test.go (TestRequireFilesWrite, TestHasPerm_NoStringsContains_Write)

tech_stack:
  added: []
  patterns:
    - Third separate capability wrapper (mirrors requireFilesRead structure, adds Origin check)
    - CSRF Origin inversion (absent-Origin passes for desktop Wails; present must match FQDN)
    - Static-grep gate (TestHasPerm_NoStringsContains_Write mirrors TestRequireCapability_UnchangedByPhase118)

key_files:
  created: []
  modified:
    - internal/capability/capability.go
    - internal/webserver/capability_mw.go
    - internal/webserver/server.go
    - internal/webserver/capability_test.go

decisions:
  - "requireFilesWrite is a third separate wrapper; files.write check was NOT added to requireCapability (would break non-file routes, CAP-02)"
  - "originAllowedForWrite inverts requireAllowedOrigin: absent Origin passes vacuously for desktop Wails; present Origin requires strict BaseURL match; fail closed when BaseURL empty with a present Origin"
  - "Five write routes reuse the existing filesDispatch closure without duplication"
  - "Worktree was behind main (missing Phase 123 write.go); merged main before implementing to get Handler.Write/Upload/Delete/Rename/Mkdir"

metrics:
  duration: ~35 minutes
  completed: 2026-06-14
  tasks_completed: 3
  files_changed: 4
---

# Phase 124 Plan 01: files.write Capability — Webserver Write Routes + Middleware Summary

**One-liner:** HMAC-gated `files.write` capability middleware with CSRF-safe vacuous-absent-Origin check mounts five write routes on the webserver tier.

## Tasks Completed

| # | Name | Commit | Status |
|---|------|--------|--------|
| 0 | Write failing tests (RED) | `04b5506` | DONE |
| 1 | PermFilesWrite const + requireFilesWrite + originAllowedForWrite | `de9db3e` | DONE |
| 2 | Mount five write routes behind requireFilesWrite | `e8650f8` | DONE |

## What Was Built

### `internal/capability/capability.go`
Added `PermFilesWrite = "files.write"` directly beneath `PermFilesRead` with a mirrored doc comment noting whole-token HasPerm semantics (never substring).

### `internal/webserver/capability_mw.go`
Added two new functions:
- `requireFilesWrite(next http.HandlerFunc) http.HandlerFunc` — third separate wrapper composing `requireCapability` (401 on bad/absent cap) → `HasPerm(PermFilesWrite)` (403 on missing perm) → `originAllowedForWrite` (403 on CSRF) → `next`.
- `originAllowedForWrite(r *http.Request) bool` — CSRF Origin check with Critical Inversion 1: absent `Origin` header returns `true` (desktop Wails fetch sends none); present `Origin` must byte-match `ws.BaseURL()`; fail closed when `BaseURL()` is empty with a present Origin.

### `internal/webserver/server.go`
Mounted five write routes after the existing read mounts, reusing the same `filesDispatch` closure:
```
PUT    /api/files/write    → requireFilesWrite → filesDispatch → h.Write
POST   /api/files/upload   → requireFilesWrite → filesDispatch → h.Upload
DELETE /api/files/delete   → requireFilesWrite → filesDispatch → h.Delete
POST   /api/files/rename   → requireFilesWrite → filesDispatch → h.Rename
POST   /api/files/mkdir    → requireFilesWrite → filesDispatch → h.Mkdir
```
Verbs match `daemon/api.go:149-153` exactly for Go 1.22+ auto-405 on wrong verbs.

### `internal/webserver/capability_test.go`
Added two tests:
- `TestRequireFilesWrite` — 403/2xx matrix across all 5 routes × correct verb + Origin sub-cases (mismatched→403, absent→pass, matching→pass) + 401-priority assertion.
- `TestHasPerm_NoStringsContains_Write` — static-grep gate: fails any write-path file containing `strings.Contains(` paired with `"files.write"`.

## Verification

All success criteria met:

- **SC#1:** cap without `files.write` → 403; with `files.write` → 2xx on all 5 routes (`TestRequireFilesWrite` GREEN)
- **SC#2:** present mismatched Origin → 403; absent Origin → pass (`TestRequireFilesWrite` Origin sub-cases GREEN)
- **SC#3:** no write-path `strings.Contains` for `files.write` (`TestHasPerm_NoStringsContains_Write` GREEN)
- `go test -race ./internal/webserver/ ./internal/capability/ -count=1` GREEN
- `go vet ./internal/webserver/ ./internal/capability/` clean
- `gofmt -l` reports no diffs on touched files

## Deviations from Plan

### Pre-execution: Worktree behind main — merged main before implementing

**Found during:** Pre-task setup
**Issue:** The worktree was initialized at `d725107` (v3.4.2 tag), which predates all Phase 123 commits. `internal/files/write.go` (with `Handler.Write/Upload/Delete/Rename/Mkdir`) was absent — Task 2 would have failed with undefined method references.
**Fix:** Ran `git merge --no-edit main` in the worktree to fast-forward to `5a94c92`, bringing in the Phase 123 write engine and all planning files.
**Files affected:** 51 files added/modified by the merge (all Phase 123 + Phase 124 planning docs)
**Impact:** No plan-specified files were modified by the merge; the merge only added Phase 123 implementation that was a prerequisite.

## Known Stubs

None — no stub values, placeholder text, or unwired data sources introduced in this plan.

## Threat Flags

No new network endpoints, auth paths, or file access patterns beyond what the plan's threat model accounts for. The five new webserver write routes are explicitly in the threat model (T-124-01 through T-124-05) and all mitigations are applied:
- T-124-01: `capability.HasPerm` used (never `strings.Contains`)
- T-124-02: Static-grep gate (`TestHasPerm_NoStringsContains_Write`) passes
- T-124-03: `originAllowedForWrite` rejects present-mismatched Origin
- T-124-04: `requireCapability` body unchanged (`TestRequireCapability_UnchangedByPhase118` GREEN)
- T-124-05: Middleware ordering preserves 401-before-403 distinction

## Self-Check: PASSED

- FOUND: internal/capability/capability.go
- FOUND: internal/webserver/capability_mw.go
- FOUND: internal/webserver/server.go
- FOUND: internal/webserver/capability_test.go
- FOUND: commit 04b5506 (RED tests)
- FOUND: commit de9db3e (capability + middleware)
- FOUND: commit e8650f8 (route mounts)
