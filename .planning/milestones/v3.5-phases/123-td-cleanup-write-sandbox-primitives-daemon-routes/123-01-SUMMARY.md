---
phase: 123-td-cleanup-write-sandbox-primitives-daemon-routes
plan: "01"
subsystem: internal/files
tags: [sandbox, write-primitives, denylist, fuzz, atomic-write, security]
dependency_graph:
  requires: []
  provides:
    - Sandbox.WriteFileAtomic (FSW-01)
    - Sandbox.Rename (FSW-02)
    - Sandbox.Mkdir / Sandbox.MkdirAll (FSW-03)
    - Sandbox.Delete (FSW-04)
    - Sandbox.denylistCheck + ErrProtectedSystemFile (FSW-06)
    - FileWriteResponse / FileOpResponse wire types (FSW-01, FSW-05)
    - FuzzSandboxWrite merge gate (FSW-07)
  affects:
    - All downstream write surfaces (daemon routes, webserver, TUI, remote)
tech_stack:
  added: []
  patterns:
    - per-op os.OpenRoot (fresh handle each call, never cached)
    - atomic write via sibling temp + f.Sync() + root.Rename (no O_TRUNC)
    - crypto/rand 8-byte suffix + O_EXCL for race-free temp naming
    - EvalSymlinks on $HOME in denylistCheck (macOS /var vs /private/var canonicalization)
    - Windows-only bounded 3-attempt rename retry (50ms intervals)
key_files:
  modified:
    - internal/files/sandbox.go
    - internal/files/types.go
    - internal/files/sandbox_test.go
  created:
    - internal/files/write_test.go
decisions:
  - "EvalSymlinks($HOME) required in denylistCheck: NewSandbox resolves workDir via EvalSymlinks, storing /private/var/... as rootPath on macOS. os.UserHomeDir() returns the HOME env var as-is (/var/...). filepath.Rel sees them as unrelated trees and returns a .. prefix, bypassing the denylist. Fix: EvalSymlinks the home dir too inside denylistCheck before computing Rel."
  - "Comment-level O_TRUNC and os.TempDir references removed from WriteFileAtomic doc so grep -c invariant counts are 0 0 (acceptance criteria)."
  - "writeAtomicRename helper extracted to isolate the Windows retry logic and keep the single-attempt non-Windows path obvious."
metrics:
  duration: "~45 minutes"
  completed: "2026-06-14"
  tasks_completed: 4
  files_created: 1
  files_modified: 3
---

# Phase 123 Plan 01: Write Sandbox Primitives Summary

**One-liner:** Atomic sandbox write primitives (temp+Sync+Rename) with crypto/rand temp suffix, os.Root-native Rename/Mkdir/Delete, shell-RC denylist via EvalSymlinks-normalized $HOME comparison, and FuzzSandboxWrite 60s merge gate (0 crashes).

## What Was Built

Extended the v3.4 read-only `internal/files.Sandbox` with five write primitives and the shell-RC denylist, completing the security core for the v3.5 write epic.

### `internal/files/sandbox.go` (extended)

**New methods (all follow the read-side per-op `os.OpenRoot` pattern):**

- `WriteFileAtomic(relPath string, content []byte) error` — writes via sibling temp (`.agenthub-tmp-<8rand_hex>`) with `O_EXCL|O_CREATE`, `f.Sync()` for crash durability, then `root.Rename` atomically. Windows-only bounded retry (3 attempts, 50ms). Never O_TRUNC, never os.TempDir (FSW-01).
- `Rename(oldRel, newRel string) error` — validates BOTH paths via `validateAndClean`, denylists both, then `root.Rename` (native, TOCTOU-safe). Destination path validation is the #1 write-side traversal risk; explicitly mitigated (FSW-02, T-123-01).
- `Mkdir(relPath string) error` — validate + denylist + `root.Mkdir` (FSW-03).
- `MkdirAll(relPath string) error` — validate + denylist + `root.MkdirAll` (native, not iterative) (FSW-03).
- `Delete(relPath string) error` — validate + denylist + `root.RemoveAll` (FSW-04).

**New sentinel and helper:**
- `var ErrProtectedSystemFile = errors.New("files: protected system file")` — "files: " prefix matches package convention (FSW-06).
- `denylistCheck(cleaned string) error` — checks shell-RC files, `.ssh/`, `.claude/`, `.config/agenthub/` under `$HOME` using EvalSymlinks-normalized comparison (FSW-06, T-123-03).

### `internal/files/types.go` (extended)

- `FileWriteResponse{Path string json:"path", Size int64 json:"size"}` — success body for Write and Upload handlers.
- `FileOpResponse{Path string json:"path", OK bool json:"ok"}` — success body for Rename, Mkdir, Delete handlers.

### `internal/files/write_test.go` (new)

22 unit tests covering FSW-01..06:
- `TestWriteFileAtomic` (round-trip, no leftover temp, overwrite, subdir, traversal rejection)
- `TestWriteFileAtomic_ConcurrentReadNeverPartial` (atomic invariant under 200 concurrent writes)
- `TestRename_SameDir`, `TestRename_CrossDirMove`, `TestRename_DestinationTraversalRejected`, `TestRename_SourceTraversalRejected`
- `TestMkdir`, `TestMkdirAll`, `TestMkdir_TraversalRejected`
- `TestDelete_File`, `TestDelete_RecursiveSubtree`, `TestDelete_TraversalRejected`
- `TestDenylist_HomeRooted` (all 9 protected targets x 4 write methods)
- `TestDenylist_NonHomeRootedUnaffected`

### `internal/files/sandbox_test.go` (extended)

Added `FuzzSandboxWrite` immediately after `FuzzSandboxPath`, reusing its full 45-seed corpus plus 6 write-specific seeds. Exercises all 5 write methods per fuzzer iteration with in-root assertion. 60s run: 0 crashes, ~190k executions.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] denylistCheck EvalSymlinks mismatch on macOS**
- **Found during:** Task 3 (TestDenylist_HomeRooted failures)
- **Issue:** `NewSandbox` calls `filepath.EvalSymlinks(workDir)`, storing `/private/var/folders/...` as `rootPath`. `denylistCheck` called `os.UserHomeDir()` which returns the `HOME` env var as-is (`/var/folders/...` on macOS). `filepath.Rel(home, abs)` returned a `..` prefix, causing all home-rooted denylist checks to silently pass.
- **Fix:** Added `filepath.EvalSymlinks(home)` inside `denylistCheck` before computing `filepath.Rel` so both paths are on the same canonical form.
- **Files modified:** `internal/files/sandbox.go`
- **Commit:** 0c1fb7b

## Known Stubs

None — all write methods are fully implemented and tested.

## Threat Surface Scan

No new threat surface beyond what the plan's threat model covers. All mitigations from the STRIDE register (T-123-01 through T-123-06) are implemented:
- T-123-01 (rename dest traversal): validateAndClean on BOTH paths + root.Rename syscall backstop
- T-123-02 (symlink TOCTOU): fresh os.OpenRoot per op
- T-123-03 (shell-RC overwrite): denylistCheck in ALL 5 methods
- T-123-04 (torn file): WriteFileAtomic temp+Sync+rename
- T-123-05 (temp name collision): crypto/rand suffix + O_EXCL
- T-123-06 (Windows rename failure): bounded 3-attempt retry

## Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Write-primitive unit + fuzz tests (RED) | 54b9b10 | `write_test.go` (new), `sandbox_test.go` (FuzzSandboxWrite added) |
| 2 | Wire types + shell-RC denylist | 50af53b | `types.go` (FileWriteResponse/FileOpResponse), `sandbox.go` (denylistCheck) |
| 3 | Write primitives GREEN | 0c1fb7b | `sandbox.go` (WriteFileAtomic/Rename/Mkdir/MkdirAll/Delete) |
| 4 | Fuzz merge gate green + invariant cleanup | 6d66cd0 | `sandbox.go` (comment cleanup), `sandbox_test.go` (gofmt) |

## Self-Check: PASSED

- `internal/files/sandbox.go` FOUND
- `internal/files/types.go` FOUND
- `internal/files/write_test.go` FOUND
- `internal/files/sandbox_test.go` (with FuzzSandboxWrite) FOUND
- commit 54b9b10 FOUND
- commit 50af53b FOUND
- commit 0c1fb7b FOUND
- commit 6d66cd0 FOUND
- `go test -race ./internal/files/ -count=1` PASS
- `FuzzSandboxWrite -fuzztime=60s` 0 crashes PASS
- `gofmt -l internal/files/` clean
- `go vet ./internal/files/...` clean
- `grep -c O_TRUNC internal/files/sandbox.go` = 0
- `grep -c os.TempDir internal/files/sandbox.go` = 0
