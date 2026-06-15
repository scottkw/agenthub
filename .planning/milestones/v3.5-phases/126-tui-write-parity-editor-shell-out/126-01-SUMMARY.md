---
phase: 126-tui-write-parity-editor-shell-out
plan: 01
subsystem: tui
tags: [go, interface, tui, bubbletea, http-client, tls, files]

# Dependency graph
requires:
  - phase: 123-td-cleanup-write-sandbox-primitives-daemon-routes
    provides: "DaemonClient WriteFile/DeleteFile/RenameFile/MkdirFile with FileWriteResponse/FileOpResponse return types"
  - phase: 122-remote-tailnet-file-browse
    provides: "RemoteFilesClient 4-read-method base + filesURL helper + NewRemoteFilesClientForTest + redactCapFromURL"
provides:
  - "FilesClient interface extended from 4 to 8 methods (4 read + 4 write)"
  - "Compile-time guard: *daemon.DaemonClient satisfies FilesClient"
  - "Compile-time guard: *RemoteFilesClient satisfies FilesClient"
  - "RemoteFilesClient.WriteFile/DeleteFile/RenameFile/MkdirFile with httptest.TLSServer round-trip tests"
  - "CAP-LEAK invariant (T-126-01) enforced + asserted in test"
affects: [126-02, 126-03, 126-04, tui-write-parity-editor-shell-out]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "FilesClient duck-typing: interface matches live DaemonClient signatures verbatim (response structs, not error-only)"
    - "Compile-time interface assertion: var _ FilesClient = (*T)(nil) in the file that owns T"
    - "CAP-LEAK invariant: RemoteFilesClient errors interpolate only (statusCode, body), never the URL with cap="
    - "TDD RED/GREEN flow: failing compile tests committed before implementation"

key-files:
  created: []
  modified:
    - internal/tui/files_client.go
    - internal/tui/remote_files_client.go
    - internal/tui/remote_files_client_test.go

key-decisions:
  - "FilesClient write-method return types are files.FileWriteResponse/FileOpResponse (matching DaemonClient), NOT error-only (ARCHITECTURE.md §4.1 sketch was stale)"
  - "UploadFile excluded from FilesClient interface per TUIW-06 descope"
  - "Daemon compile-time guard lives in files_client.go; RemoteFilesClient guard stays in remote_files_client.go"
  - "RenameFile uses a local remoteRenameRequest struct in remote_files_client.go to avoid cross-package visibility of daemon.renameRequest"
  - "httptest.TLSServer + NewRemoteFilesClientForTest used for all write round-trip tests (mirrors existing read tests)"

patterns-established:
  - "Write method error format: 'remote files <op>: %d %s' (status, body) — never URL"
  - "remoteRenameRequest local struct for JSON body (no daemon import in tui package for this)"

requirements-completed: [TUIW-01]

# Metrics
duration: 15min
completed: 2026-06-14
---

# Phase 126 Plan 01: FilesClient 8-Method Interface + RemoteFilesClient Write Methods Summary

**FilesClient extended 4→8 methods with response-struct returns matching DaemonClient; both *daemon.DaemonClient and *RemoteFilesClient satisfy the interface via compile-time guards; 5 httptest.TLSServer round-trip tests pass including a CAP-LEAK invariant assertion**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-06-14T00:00:00Z
- **Completed:** 2026-06-14T00:15:00Z
- **Tasks:** 2 (Task 1: interface + daemon guard; Task 2: RemoteFilesClient write methods + tests)
- **Files modified:** 3

## Accomplishments

- Extended `FilesClient` interface from 4 read methods to 8 (4 read + 4 write), with write return types matching the live DaemonClient signatures exactly (FileWriteResponse/FileOpResponse, NOT error-only)
- Added `var _ FilesClient = (*daemon.DaemonClient)(nil)` compile-time guard in files_client.go — build fails immediately if any signature diverges
- Implemented `WriteFile`, `DeleteFile`, `RenameFile`, `MkdirFile` on `RemoteFilesClient` with CAP-LEAK invariant (T-126-01): error strings never contain the cap token
- Added 5 httptest.TLSServer round-trip tests: Write/Delete/Rename/Mkdir happy paths + WriteCapLeak assertion; full TDD RED→GREEN flow

## Task Commits

Each task was committed atomically:

1. **Task 1 + Task 2 RED: FilesClient 8-method interface + daemon guard + failing write tests** - `8b9333a` (test)
2. **Task 2 GREEN: RemoteFilesClient write methods implementation** - `5da0a5f` (feat)

_Note: Task 1 (interface changes) was committed in the same RED commit because the interface file changes and failing tests are co-dependent for the compile-time guard._

## Files Created/Modified

- `internal/tui/files_client.go` — Extended FilesClient from 4 to 8 methods; added daemon import; added `var _ FilesClient = (*daemon.DaemonClient)(nil)` compile-time guard
- `internal/tui/remote_files_client.go` — Added `bytes` import; added WriteFile/DeleteFile/RenameFile/MkdirFile methods with CAP-LEAK-safe error strings; added `remoteRenameRequest` local struct
- `internal/tui/remote_files_client_test.go` — Added `bytes`/`io` imports; added TestRemoteFilesClient_Write/Delete/Rename/Mkdir/WriteCapLeak

## Decisions Made

- **Response types must be structs, not error-only**: The milestone ARCHITECTURE.md §4.1 interface sketch used `error`-only returns for write methods. The actual Phase 123 DaemonClient methods return `files.FileWriteResponse`/`files.FileOpResponse`. The interface must match the existing implementer — error-only would have caused a compile-time failure. The daemon compile-time guard catches this instantly.
- **UploadFile excluded**: UploadFile is descoped for TUI (TUIW-06). Interface has exactly 8 methods (4 read + 4 write), no UploadFile.
- **remoteRenameRequest as local struct**: The daemon package has a private `renameRequest` struct. Rather than importing daemon just for this type, a local `remoteRenameRequest` struct is defined in remote_files_client.go with identical JSON tags.
- **Daemon guard placement**: The daemon compile-time guard lives in `files_client.go` (where the interface is defined), not in the daemon package (which would create a cycle). This is the natural location.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

**Worktree was behind main**: The worktree branch had not yet received Phase 123/124/125 commits (including the DaemonClient write methods). The objective specified merging main if behind; a `git merge main` fast-forward was performed before starting. No conflicts.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. The new write methods route through the same HTTPS+cap transport already established in Phase 122. The CAP-LEAK invariant (T-126-01) is enforced identically to the existing read methods.

## Known Stubs

None. All 8 FilesClient methods are fully implemented on both `*daemon.DaemonClient` and `*RemoteFilesClient`.

## Next Phase Readiness

- FilesClient is the contract foundation for all remaining Phase 126 plans (126-02 through 126-04)
- Both implementers satisfy the 8-method interface — the write dispatch pipeline (`files_cmds.go`, `files.go`, `update.go`) can now reference `client.WriteFile/DeleteFile/RenameFile/MkdirFile` directly through the interface
- Phase 128 (remote write parity UAT) depends on RemoteFilesClient write methods existing; those are now implemented and unit-tested

## Self-Check

Files exist:
- internal/tui/files_client.go: FOUND
- internal/tui/remote_files_client.go: FOUND
- internal/tui/remote_files_client_test.go: FOUND

Commits exist:
- 8b9333a: FOUND
- 5da0a5f: FOUND

## Self-Check: PASSED

---
*Phase: 126-tui-write-parity-editor-shell-out*
*Completed: 2026-06-14*
