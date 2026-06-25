---
phase: 145-windows-files-test-fixes
plan: "02"
subsystem: internal/files
tags: [test-fix, windows-ci, concurrent-read, FILE_SHARE_DELETE, build-tags]
dependency_graph:
  requires: [145-01]
  provides: [windows-concurrent-read-fix]
  affects: [internal/files]
tech_stack:
  added: []
  patterns: [build-tagged-platform-helper, syscall.CreateFile-FILE_SHARE_DELETE]
key_files:
  created:
    - internal/files/concurrent_read_windows_test.go
    - internal/files/concurrent_read_unix_test.go
  modified:
    - internal/files/write_test.go
    - TESTING.md
decisions:
  - "Use build-tagged helper files rather than a single file with runtime GOOS detection — the syscall.CreateFile symbol is undefined on non-Windows and would fail vet"
  - "Use syscall.CreateFile directly (not os.OpenFile) to control share flags — os.OpenFile on Windows hardcodes FILE_SHARE_READ|FILE_SHARE_WRITE without FILE_SHARE_DELETE"
  - "Leave writeAtomicRename retry loop in sandbox.go untouched — the retry is still valuable for production (antivirus scanners, etc); the test fix is the correct scope"
  - "os import retained in write_test.go — still used by many other tests in the file; removing it would break compilation"
metrics:
  duration: "~8 minutes"
  completed: "2026-06-22T06:05:00Z"
  tasks_completed: 3
  files_modified: 4
---

# Phase 145 Plan 02: Windows Files Test Fixes (Concurrent-Read) Summary

Build-tagged `readFilePlatformSafe` helpers for Windows (syscall.CreateFile with FILE_SHARE_DELETE) and non-Windows (os.ReadFile delegate), wired into the concurrent-read test goroutine so WriteFileAtomic's POSIX-semantics rename can succeed while a reader holds the destination open.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Create build-tagged readFilePlatformSafe helpers | cfa64145 | internal/files/concurrent_read_windows_test.go, internal/files/concurrent_read_unix_test.go |
| 2 | Wire reader goroutine to readFilePlatformSafe | cf325d6d | internal/files/write_test.go |
| 3 | Full-suite no-regression + Windows cross-compile gate + TESTING.md update | f04f2c98 | TESTING.md |

## What Was Built

**concurrent_read_windows_test.go — Windows build-tagged helper (Task 1)**

Created with `//go:build windows` tag, `package files_test` package clause. Defines `readFilePlatformSafe(path string) ([]byte, error)` that: converts path via `syscall.UTF16PtrFromString`, calls `syscall.CreateFile` with `GENERIC_READ` and `FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE`, wraps the handle in `os.NewFile`, defers `f.Close()`, and returns `io.ReadAll(f)`. Imports `io`, `os`, `syscall`. Propagates errors from both `UTF16PtrFromString` and `CreateFile`.

The FILE_SHARE_DELETE flag is the key: `windows.Renameat` uses `FILE_RENAME_POSIX_SEMANTICS` which requires all open destination handles to permit delete. Without this flag, `NtSetInformationFile` fails, `writeAtomicRename` exhausts its retry loop, and `WriteFileAtomic` returns an error that fires `t.Errorf`.

**concurrent_read_unix_test.go — Non-Windows build-tagged helper (Task 1)**

Created with `//go:build !windows` tag, `package files_test` package clause. Defines the same `readFilePlatformSafe` signature as a one-line delegate to `os.ReadFile`. On POSIX, `rename(2)` is atomic regardless of open read handles — no share flags needed.

**write_test.go — Reader goroutine wired to readFilePlatformSafe (Task 2)**

Single-line change in `TestWriteFileAtomic_ConcurrentReadNeverPartial` reader goroutine: replaced `os.ReadFile(targetPath)` with `readFilePlatformSafe(targetPath)`. The three atomicity assertions (never empty, never partial length, never mixed content) are unchanged and still execute. The writer loop is unchanged. `sandbox.go` production retry loop is not modified (`git diff internal/files/sandbox.go` is empty). The `os` import is retained — used by many other tests in the file.

**TESTING.md (Task 3)**

- Go test file count updated: 344 → 346 (added 2 new `*_test.go` files)
- Traceability map: added 3 FIX-02 rows mapping `write_test.go`, `concurrent_read_windows_test.go`, and `concurrent_read_unix_test.go` to the FIX-02 requirement

**Task 3 — verification results**

- `go test -race -short ./internal/files/`: PASS
- `go test -race -count=3 -run TestWriteFileAtomic_ConcurrentReadNeverPartial ./internal/files/`: PASS
- `GOOS=windows GOARCH=amd64 go vet ./internal/files/`: PASS
- `GOOS=windows GOARCH=amd64 go test -c ./internal/files/ -o /dev/null`: PASS
- `git diff internal/files/sandbox.go`: empty (production retry loop untouched)
- No production `.go` files outside `*_test.go` modified

## Deviations from Plan

### Auto-added (CLAUDE.md Standing Rule)

**1. [CLAUDE.md - Regression Test Convention] TESTING.md updated**
- **Found during:** Task 3 (per CLAUDE.md standing rule: every phase adding test files must update TESTING.md)
- **Action:** Updated Go test count 344 → 346; added 3 FIX-02 traceability rows; ran `bash tests/check-traceability-paths.sh` — exits 0
- **Files modified:** TESTING.md
- **Commit:** f04f2c98

## Known Stubs

None.

## Threat Flags

None. Both new files are `_test.go` files (test-only build; not compiled into production binary). The FILE_SHARE_DELETE change affects only the test reader goroutine — production `writeAtomicRename` in `sandbox.go` is unmodified.

## Self-Check: PASSED

- `internal/files/concurrent_read_windows_test.go` exists: confirmed
- `internal/files/concurrent_read_unix_test.go` exists: confirmed
- `internal/files/write_test.go` modified (readFilePlatformSafe wired): confirmed
- `TESTING.md` updated (count + traceability): confirmed
- Commit cfa64145 exists: confirmed (git log shows it)
- Commit cf325d6d exists: confirmed (git log shows it)
- Commit f04f2c98 exists: confirmed (git log shows it)
- `go test -race -short ./internal/files/` exits 0: confirmed
- `go test -race -count=3 -run TestWriteFileAtomic_ConcurrentReadNeverPartial ./internal/files/` exits 0: confirmed
- `GOOS=windows GOARCH=amd64 go vet ./internal/files/` exits 0: confirmed
- `GOOS=windows GOARCH=amd64 go test -c ./internal/files/ -o /dev/null` exits 0: confirmed
- `git diff internal/files/sandbox.go` empty: confirmed
- `bash tests/check-traceability-paths.sh` exits 0: confirmed
