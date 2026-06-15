---
phase: 127-web-share-write-security-hardening
plan: "01"
subsystem: internal/files
tags: [security, denylist, sandbox, hardening, case-fold, config-dir]
dependency_graph:
  requires: []
  provides: [hardened-denylistCheck, TestDenylist_DaemonConfigDir, TestDenylist_CaseVariation, FuzzSandboxWrite-seeds]
  affects: [internal/files/sandbox.go, internal/files/write_test.go, internal/files/sandbox_test.go]
tech_stack:
  added: []
  patterns: [strings.ToLower case-fold, os.UserConfigDir derivation, EvalSymlinks-rebase homeRaw/home pattern]
key_files:
  created: []
  modified:
    - internal/files/sandbox.go
    - internal/files/write_test.go
    - internal/files/sandbox_test.go
decisions:
  - "ASCII case-fold (strings.ToLower) chosen over NFC normalization — all protected names are ASCII; NFC documented as LOW residual"
  - "EvalSymlinks-rebase pattern: compute cfgBase relative to homeRaw (unresolved), then rebase onto resolved home — handles macOS /var/... vs /private/var/... divergence"
  - "homeRaw retained alongside EvalSymlinks-resolved home so os.UserConfigDir() path can be rebased correctly"
  - "Static .config/agenthub/ prefix kept as belt-and-suspenders for Linux/cross-platform copied trees"
  - "No new imports to internal/files — cycle-free constraint maintained"
metrics:
  duration: ~30min
  completed: 2026-06-14
  tasks_completed: 2
  files_modified: 3
---

# Phase 127 Plan 01: Denylist Hardening (case-fold + daemon config dir) Summary

**One-liner:** ASCII case-fold + os.UserConfigDir-derived macOS config dir added to denylistCheck, closing .BASHRC bypass and ~/Library/Application Support/agenthub overwrite hole.

## What Was Built

### Task 1: denylistCheck hardening (internal/files/sandbox.go)

Two net-new production security fixes landed inside `denylistCheck` only — WriteFileAtomic/Rename/Mkdir/Delete bodies untouched:

**GAP 2 fix (case-fold):** `strings.ToLower(filepath.Base(canonAbs))` before the base-name switch and `strings.ToLower(filepath.ToSlash(rel))` before the dir-prefix loop. `.BASHRC`, `.Bashrc`, `.BasHrc` and case-varied dir prefixes (`.SSH/`, `.Claude/`) now match on macOS/Windows case-insensitive volumes.

**GAP 1 fix (macOS daemon config dir):** `os.UserConfigDir()` called inside `denylistCheck` to derive the platform-correct config dir prefix. On macOS this returns `~/Library/Application Support` (not `~/.config`), so the daemon's `settings.json` at `~/Library/Application Support/agenthub/settings.json` is now protected.

**EvalSymlinks rebase pattern:** On macOS `t.TempDir()` returns `/var/folders/...` but `filepath.EvalSymlinks(home)` resolves to `/private/var/folders/...`. Since `os.UserConfigDir()` is derived from the `HOME` env var (unresolved), cfgBase has the `/var/` prefix while home has `/private/var/`. Fix: compute `cfgBase` relative to `homeRaw` (before EvalSymlinks), then rebase onto the resolved `home`. This makes `filepath.Rel` produce a clean relative path in both production and test contexts.

### Task 2: Tests + fuzz seeds

**TestDenylist_CaseVariation** (write_test.go): asserts `.BASHRC`, `.Bashrc`, `.SSH/authorized_keys`, `.Claude/CLAUDE.md` all return `ErrProtectedSystemFile` in a home-rooted sandbox.

**TestDenylist_DaemonConfigDir** (write_test.go): asserts `os.UserConfigDir()/agenthub/settings.json` returns `ErrProtectedSystemFile` for WriteFileAtomic/Delete/Rename-into. Uses `os.UserConfigDir()` derivation (no hardcoded `Library/Application Support`). On macOS sets HOME to tmpdir and rebases the derived cfgBase onto it. Skips when cfgDir cannot be faked under fakeHome (Windows, some CI configurations).

**TestDenylist_NonHomeRootedUnaffected** preserved — passes unchanged.

**FuzzSandboxWrite seeds** (sandbox_test.go): `../../.BASHRC`, `../../.Bashrc`, `.SSH/authorized_keys`, `../../../etc/passwd` added after existing write-specific seeds. Fuzz gate `go test -fuzz=FuzzSandboxWrite -fuzztime=60s` confirmed 0 crashes on host (105,964 executions, 183 interesting corpus entries).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] EvalSymlinks divergence between home and cfgBase on macOS**
- **Found during:** Task 2 test execution — TestDenylist_DaemonConfigDir failed with `WriteFileAtomic returned nil` when cfgDir was under fakeHome
- **Issue:** `filepath.EvalSymlinks(home)` resolves `/var/folders/...` to `/private/var/folders/...`. But `os.UserConfigDir()` is derived from the raw HOME env var, so cfgBase retains `/var/...` prefix. `filepath.Rel("/private/var/.../fakehome", "/var/.../fakehome/Library/...")` fails (different prefixes), `strings.HasPrefix(cfgRel, "..")` is true, and the derived prefix is silently not added to protectedDirs.
- **Fix:** Introduced `homeRaw` (before EvalSymlinks). When cfgBase is under homeRaw, rebase it onto the resolved home: `cfgBase = filepath.Join(home, rel-from-homeRaw)`. This also handles the real-world case correctly: on macOS in production, `os.UserConfigDir()` = `/Users/ken/Library/Application Support`, `homeRaw` = `/Users/ken`, `home` = `/Users/ken` (EvalSymlinks is a no-op on real home), so the rebase is a no-op and everything works.
- **Files modified:** `internal/files/sandbox.go`
- **Commit:** `46b682b`

## Acceptance Criteria Verification

- `go build ./internal/files/` exits 0: PASS
- `gofmt -l internal/files/sandbox.go` prints nothing: PASS
- `grep -c "os.UserConfigDir" internal/files/sandbox.go` >= 1: PASS (2 occurrences)
- `grep -c '"internal/daemon"' internal/files/sandbox.go` == 0: PASS (0 — only a comment reference, no import)
- WriteFileAtomic/Rename/Mkdir/Delete bodies unchanged (diff touches only denylistCheck): PASS (all @@ hunks in sandbox.go within denylistCheck)
- `go test ./internal/files/ -run TestDenylist -count=1` exits 0: PASS (all subtests green)
- `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` 0 crashes: PASS
- `grep -c "TestDenylist_DaemonConfigDir" internal/files/write_test.go` >= 1: PASS (2)
- `grep -c "\.BASHRC" internal/files/sandbox_test.go` >= 1: PASS (1)
- No hardcoded "Library/Application Support" literal in test assertions: PASS (only in comment)
- `go test -race ./internal/files/...` green: PASS

## Known Stubs

None.

## Threat Flags

None. All changes are within the existing `denylistCheck` security boundary — no new network endpoints, auth paths, or schema changes introduced.

## Self-Check: PASSED

Files confirmed present:
- internal/files/sandbox.go — modified in place
- internal/files/write_test.go — modified in place
- internal/files/sandbox_test.go — modified in place

Commits confirmed:
- 8906fb4 fix(127-01): harden denylistCheck — ASCII case-fold + os.UserConfigDir config dir
- 46b682b fix(127-01): canonicalize cfgBase against resolved home to handle macOS symlinks
- fe9d0dd test(127-01): add TestDenylist_DaemonConfigDir, case-variation tests, fuzz seeds
