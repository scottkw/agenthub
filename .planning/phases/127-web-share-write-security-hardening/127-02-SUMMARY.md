---
phase: 127-web-share-write-security-hardening
plan: "02"
subsystem: internal/files
tags: [security, sandbox, symlink-escape, fuzz, write-path, SEC-01, SEC-03, SEC-06]
dependency_graph:
  requires: [127-01]
  provides: [TestSandbox_WritePathSymlinkEscapeBlocked, FuzzSandboxWrite-merge-gate-confirmed, SEC-03-upload-confirmed]
  affects: [internal/files/sandbox_test.go]
tech_stack:
  added: []
  patterns: [os.Root write-path TOCTOU boundary, WR-03 positive-control guard, fuzz merge gate]
key_files:
  created: []
  modified:
    - internal/files/sandbox_test.go
decisions:
  - "No production code modified — os.Root boundary already enforces write-path symlink rejection; test makes SC1 explicit"
  - "Positive control writes a probe file to outside dir (not just reads) before exercising escape, ensuring outside is writable"
  - "SEC-03 multipart filepath.Base sanitization confirmed in handler_test.go (TestHandlerUpload_FilenameSanitized); over-cap 413 confirmed in TestHandlerUpload_OverCap413 / TestHandlerWrite_OverCap413"
metrics:
  duration: "~70 minutes (including 60s fuzz gate)"
  completed: "2026-06-15"
  tasks_completed: 2
  files_changed: 1
---

# Phase 127 Plan 02: Write-Path Symlink-Escape Test (SEC-01) + FuzzSandboxWrite Merge Gate (SEC-06) + SEC-03 Upload-Abuse Confirmation Summary

**One-liner:** Write-path symlink-escape test (SEC-01) added via os.Root TOCTOU boundary assertion; FuzzSandboxWrite 60s merge gate confirmed zero crashes; SEC-03 upload-abuse coverage (filepath.Base sanitization + MaxBytesReader over-cap rejection) confirmed green.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | TestSandbox_WritePathSymlinkEscapeBlocked | a092894 | internal/files/sandbox_test.go |
| 2 | Confirm FuzzSandboxWrite merge gate + SEC-03 upload-abuse coverage | (run-only, no code change) | — |

## What Was Built

### Task 1: TestSandbox_WritePathSymlinkEscapeBlocked

Added `TestSandbox_WritePathSymlinkEscapeBlocked` to `internal/files/sandbox_test.go` (after line 248, before FuzzSandboxPath). The test mirrors `TestSandbox_SymlinkEscapeBlocked` for the write path:

- Creates `outside := t.TempDir()` with `outside/secret` = "leaked"
- **Positive control (WR-03):** reads `outside/secret` (must == "leaked") and writes a probe file to confirm `outside` is genuinely writable — prevents ENOENT masking
- Plants `root/escape -> outside/` symlink; skips on Windows and unsupported platforms
- Asserts each write method through the symlink returns a non-nil error:
  - `sb.WriteFileAtomic("escape/pwned", ...)` — want error
  - `sb.Rename("a.txt", "escape/pwned")` — want error (destination traversal)
  - `sb.Mkdir("escape/sub")` — want error
- After each, `os.Stat` the outside path and asserts it was NOT created
- Final re-read of `outside/secret` asserts it equals "leaked" (sentinel unmodified)

**Protection mechanism:** `os.OpenRoot(s.rootPath)` + native `root.OpenFile` / `root.Rename` / `root.Mkdir` atomically reject escaping symlinks (T-127-05). This test makes SC1 verifiable; it does not add protection.

### Task 2: FuzzSandboxWrite merge gate (SEC-06) + SEC-03 confirmation

Run-only task (no code changes):

- `go test -race ./internal/files/... -count=1` — PASS
- `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` — **0 crashes**, PASS (72,377 executions, 190 interesting corpus entries)
- `go test ./internal/files/ -run 'TestUpload|TestMaxUploadBytes' -count=1` — PASS (5 tests)
- SEC-03 multipart upload-filename sanitization confirmed: `TestHandlerUpload_FilenameSanitized` (filepath.Base strips `../../.bashrc` to `.bashrc`), `TestHandlerUpload_EmptyFilename400`, `TestHandlerUpload_DotFilename400`, `TestHandlerUpload_DotDotFilename400` — all green
- SEC-03 over-cap rejection confirmed: `TestHandlerUpload_OverCap413` (413, file not on disk), `TestHandlerWrite_OverCap413` (413, file not on disk) — both green
- No crash corpus files created under `internal/files/testdata/fuzz`

## Deviations from Plan

None — plan executed exactly as written.

## Threat Model Coverage

| Threat ID | Disposition | Confirmed By |
|-----------|-------------|--------------|
| T-127-05 | mitigate (verify) | TestSandbox_WritePathSymlinkEscapeBlocked — all 3 write methods reject escaping symlink; nothing created outside root |
| T-127-06 | mitigate | FuzzSandboxWrite 60s gate — 0 crashes; corpus includes case-variation + multipart vectors from 127-01 |
| T-127-12 | mitigate (verify) | TestHandlerUpload_FilenameSanitized (filepath.Base strip confirmed); TestHandlerUpload_OverCap413 / TestHandlerWrite_OverCap413 (MaxBytesReader rejects before ParseMultipartForm, file not created on disk) |

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes introduced. Test-only plan.

## Known Stubs

None — test-only plan; no UI or data-flow stubs.

## Self-Check: PASSED

- [x] `internal/files/sandbox_test.go` modified and present
- [x] Commit `a092894` exists: `git log --oneline | grep a092894`
- [x] `TestSandbox_WritePathSymlinkEscapeBlocked` passes: `go test ./internal/files/ -run TestSandbox_WritePathSymlinkEscapeBlocked -count=1` exits 0
- [x] FuzzSandboxWrite 60s: 0 crashes (PASS)
- [x] `go test -race ./internal/files/... -count=1`: PASS
- [x] No production source (sandbox.go, write.go) modified
- [x] No STATE.md or ROADMAP.md changes
