---
phase: 118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi
plan: 03
subsystem: capability + webserver middleware
tags: [capability, security, middleware, files.read, FS-10, FS-11, FS-13]
dependency_graph:
  requires:
    - "internal/capability package (Sign/Verify, Claims struct, WithClaims/ClaimsFromContext)"
    - "internal/webserver requireCapability middleware (existing 7-step gate)"
  provides:
    - "capability.PermFilesRead constant ('files.read')"
    - "capability.HasPerm(perms, perm) helper with whole-token comma-split semantics"
    - "(*WebServer).requireFilesRead wrapper for Phase 119 to mount on /api/files/*"
  affects:
    - "Phase 119 will import requireFilesRead and PermFilesRead to gate webserver file routes"
    - "Phase 118 Plan 05 will set Perms: 'read,write,files.read' on owner cap tokens"
tech_stack:
  added: []
  patterns:
    - "Whole-token comma-split capability check (NEVER strings.Contains)"
    - "Wrapper-of-wrapper middleware composition (requireFilesRead delegates to requireCapability)"
    - "Source-inspection invariant guards (TestRequireCapability_UnchangedByPhase118)"
key_files:
  created: []
  modified:
    - internal/capability/capability.go
    - internal/capability/capability_test.go
    - internal/webserver/capability_mw.go
    - internal/webserver/capability_test.go
decisions:
  - "Used capability.ClaimsFromContext (the actual API) instead of capability.FromContext (the name written in the plan) — Rule 3 deviation"
  - "Added defense-in-depth ok-check on ClaimsFromContext inside requireFilesRead — Rule 2 hardening"
  - "Bounded TestRequireCapability_UnchangedByPhase118's body extraction at the closing brace (\\n}\\n) rather than the next \\nfunc — the next-func bound erroneously captured the docstring of requireFilesRead"
metrics:
  duration: "~5 min"
  tasks_completed: 2
  files_modified: 4
  commits: 4
  completed_date: "2026-05-20"
requirements: [FS-10, FS-11, FS-13]
---

# Phase 118 Plan 03: Capability Bit + HasPerm + requireFilesRead Wrapper Summary

Add the `files.read` capability primitive — `PermFilesRead` constant, whole-token `HasPerm` helper (Pitfall 4 mitigation), and a `requireFilesRead` middleware wrapper composing `requireCapability` + `HasPerm` — wired and unit-tested but not yet mounted on any route (Phase 119 mounts).

## What was done

| Task | Description | Commits |
|------|-------------|---------|
| 1 | `PermFilesRead = "files.read"` constant + `HasPerm(perms, perm) bool` helper with whole-token comma-split semantics in `internal/capability/capability.go`; 10 TestHasPerm subtests including the explicit `"no-files.read"` false-positive guard | `dc7237f` (RED test), `82c4dca` (GREEN feat) |
| 2 | `(*WebServer).requireFilesRead` wrapper in `internal/webserver/capability_mw.go` composing the existing `requireCapability` with the new `HasPerm` check; four `TestRequireFilesRead` subtests + `TestRequireCapability_UnchangedByPhase118` source-inspection invariant guard | `78b2608` (RED test), `0bacc35` (GREEN feat) |

## Files Touched

| File | Change Type | Purpose |
|------|-------------|---------|
| `internal/capability/capability.go` | added const + func | `PermFilesRead` constant and `HasPerm` helper |
| `internal/capability/capability_test.go` | added tests | `TestHasPerm` (10 subtests), `TestPermFilesRead_Constant`, `TestHasPerm_NoStringsContains` source guard |
| `internal/webserver/capability_mw.go` | added method | `(*WebServer).requireFilesRead` wrapper — `requireCapability` body unchanged |
| `internal/webserver/capability_test.go` | added tests + imports | `TestRequireFilesRead` (4 subtests), `TestRequireCapability_UnchangedByPhase118` source guard; added `net/http/httptest` and `os` imports |

## Key Decisions / Architectural Notes

- **Whole-token semantics, never `strings.Contains`.** `HasPerm` splits on comma and compares each token for exact equality. The classic Pitfall 4 false-positive case `HasPerm("no-files.read", "files.read")` returns `false` — a substring-based check would return `true` and silently grant access. The `TestHasPerm_NoStringsContains` source-inspection test enforces this at the implementation level (not just the observed behaviour) by extracting the `HasPerm` function body and asserting `strings.Split` is present and `strings.Contains` is absent.
- **Separate wrapper, never modify shared `requireCapability`.** Adding the `files.read` check to the existing `requireCapability` would break every existing terminal/relay/plugin route that doesn't carry the bit (T-118-14). `requireFilesRead` is a thin wrapper around `requireCapability`; the existing function body is byte-for-byte unchanged. `TestRequireCapability_UnchangedByPhase118` source-inspects `capability_mw.go` and asserts the literal string `"files.read"` does not appear inside `requireCapability`'s function body.
- **403 body must contain literal `"files.read"`.** This is the load-bearing FS-13 contract assertion so the Phase 120 frontend can surface a meaningful permission-denied UX instead of a generic "Forbidden". The wrapper emits `"files.read capability required"` on both miss paths.
- **Claims struct field order is untouched.** Field declaration order is load-bearing for HMAC verification per the package doc-comment. Phase 118 is purely additive — new const and new function. T-118-16 mitigation verified by grep.

## Deviations from Plan

### Rule 3 (auto-fix blocking) — plan referenced wrong API name

**Found during:** Task 2 implementation.

**Issue:** The plan's action block referenced `capability.FromContext(r.Context())`, but the actual capability package API is `ClaimsFromContext(ctx context.Context) (Claims, bool)` defined in `internal/capability/context.go`.

**Fix:** Used `capability.ClaimsFromContext(r.Context())` and handled the returned `(Claims, bool)` tuple.

**Files modified:** `internal/webserver/capability_mw.go`

**Commit:** `0bacc35`

### Rule 2 (auto-add missing critical functionality) — defense-in-depth ClaimsFromContext ok-check

**Found during:** Task 2 implementation.

**Issue:** `ClaimsFromContext` returns `(Claims, bool)`; the bool signals whether claims were attached. The plan's pseudo-code ignored the bool. If the wrapper chain were ever mis-composed and `requireCapability` did not attach claims, the inner handler would receive a zero-value `Claims{}` and `HasPerm("", "files.read")` would return `false` — correct on the happy path, but the failure mode would be silent rather than explicit.

**Fix:** Added an explicit `if !ok { 403 }` guard so a mis-composed wrapper chain returns 403 with the same `"files.read capability required"` body as the perms-missing path, never silently trusting an unverified caller.

**Files modified:** `internal/webserver/capability_mw.go`

**Commit:** `0bacc35`

### Rule 1 (auto-fix bug) — TestRequireCapability_UnchangedByPhase118 false-positive

**Found during:** Task 2 GREEN gate run.

**Issue:** First implementation of the source-inspection guard bounded `requireCapability`'s body extent at the next `\nfunc ` line. That captured the doc-comment of the immediately-following `requireFilesRead` method (which legitimately mentions `"files.read"`), producing a false positive.

**Fix:** Re-bounded at `\n}\n` (the canonical end of a top-level Go function) so the extracted body terminates at `requireCapability`'s closing brace before any subsequent docstring begins.

**Files modified:** `internal/webserver/capability_test.go`

**Commit:** `0bacc35`

## Threat Coverage

The plan's `<threat_model>` listed four mitigations; all are in place:

| Threat | Mitigation Status |
|--------|-------------------|
| T-118-13 (HasPerm substring false-positive) | `HasPerm` uses `strings.Split` + exact equality; `TestHasPerm` covers 10 subtests including the explicit `"no-files.read"` guard; `TestHasPerm_NoStringsContains` source-inspects the body |
| T-118-14 (adding files.read check to shared requireCapability) | `requireFilesRead` is a separate wrapper; `TestRequireCapability_UnchangedByPhase118` source-inspects `requireCapability`'s body for "files.read" and asserts count == 0 |
| T-118-15 (403 message UX) | 403 body is `"files.read capability required"` — contains the literal substring `"files.read"`; verified by `TestRequireFilesRead/403_when_files.read_missing` and `TestRequireFilesRead/viewer_token_(read_only)_gets_403` |
| T-118-16 (Claims field order change) | Plan does not modify `Claims` struct; verified by grep on `type Claims struct` block |

## Verification Output

All commands from the plan's `<verification>` block exit 0:

```text
go build ./internal/capability/ ./internal/webserver/   → 0
go vet ./internal/capability/ ./internal/webserver/     → 0 (no diagnostics)
go test -run '^TestHasPerm$' ./internal/capability/     → PASS (10 subtests)
go test -run '^TestRequireFilesRead$' ./internal/webserver/ → PASS (4 subtests)
go test ./internal/capability/ ./internal/webserver/    → PASS (no regressions)
```

## TDD Gate Compliance

Both tasks followed the RED → GREEN cycle and committed each gate separately:

- Task 1: `test(118-03): add failing TestHasPerm + PermFilesRead constant tests` (`dc7237f`) → `feat(118-03): add PermFilesRead constant and HasPerm helper to capability pkg` (`82c4dca`)
- Task 2: `test(118-03): add failing TestRequireFilesRead + UnchangedByPhase118 guard` (`78b2608`) → `feat(118-03): add requireFilesRead wrapper composing requireCapability + HasPerm` (`0bacc35`)

No REFACTOR commit was needed — the implementations were small enough that the GREEN-phase code was already clean.

## Known Stubs

None. All code paths are wired through to working assertions.

## Self-Check

- `internal/capability/capability.go` — FOUND (PermFilesRead + HasPerm present)
- `internal/capability/capability_test.go` — FOUND (TestHasPerm, TestPermFilesRead_Constant, TestHasPerm_NoStringsContains)
- `internal/webserver/capability_mw.go` — FOUND (requireFilesRead defined; requireCapability unchanged)
- `internal/webserver/capability_test.go` — FOUND (TestRequireFilesRead + TestRequireCapability_UnchangedByPhase118)
- Commit dc7237f — FOUND in `git log`
- Commit 82c4dca — FOUND in `git log`
- Commit 78b2608 — FOUND in `git log`
- Commit 0bacc35 — FOUND in `git log`

## Self-Check: PASSED
