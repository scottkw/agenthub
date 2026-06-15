---
phase: 123-td-cleanup-write-sandbox-primitives-daemon-routes
plan: "04"
subsystem: internal/daemon
tags: [client-methods, write-routes, multipart-upload, context-aware, typed-errors, tdd]
dependency_graph:
  requires:
    - phase: 123-01
      provides: FileWriteResponse/FileOpResponse wire types + Sandbox write primitives
    - phase: 123-03
      provides: five daemon write routes (PUT write, POST upload, DELETE delete, POST rename, POST mkdir)
  provides:
    - DaemonClient.WriteFile(ctx, sessionID, relPath, data) → FileWriteResponse
    - DaemonClient.UploadFile(ctx, sessionID, dir, filename, data) → FileWriteResponse
    - DaemonClient.DeleteFile(ctx, sessionID, relPath) → FileOpResponse
    - DaemonClient.RenameFile(ctx, sessionID, oldRel, newRel) → FileOpResponse
    - DaemonClient.MkdirFile(ctx, sessionID, relPath) → FileOpResponse
  affects:
    - Phase 124 (capability gating — files.write will gate these routes on the webserver surface)
    - Phase 125 (CodeMirror editor wires through WriteFile/UploadFile)
    - Phase 126 (TUI write surface calls these DaemonClient write methods)
tech_stack:
  added:
    - mime/multipart (stdlib — multipart form body builder for UploadFile)
  patterns:
    - context-aware request: http.NewRequestWithContext(ctx, method, filesURL(...), body)
    - status-as-typed-error: non-2xx → fmt.Errorf("files <op>: %d %s", status, body)
    - filesURL helper: existing helper extended for write op names (write/upload/delete/rename/mkdir)
    - JSON request struct: renameRequest{OldRel, NewRel} for RenameFile JSON body
    - multipart writer: mime/multipart.NewWriter + CreateFormFile for UploadFile
    - FilesClient interface scope guard: NOT extended (Phase 126 / TUIW-01 only)
key_files:
  modified:
    - internal/daemon/client.go
    - internal/daemon/client_test.go
key-decisions:
  - "UploadFile builds a multipart/form-data body client-side (mime/multipart.NewWriter + CreateFormFile) matching the server-side handler.go form field contract (dir + file). This is stdlib-only — no new dependency."
  - "RenameFile sends {oldRel, newRel} as a JSON body (not query params) to handle Unicode filenames with special characters safely. Mirrors the server-side renameRequest struct exactly."
  - "MkdirFile passes the target path via filesURL (query param) not a request body — the server reads it via h.relPath(r) just like read handlers. No JSON wrapper needed."
  - "FilesClient interface in internal/tui/files_client.go is NOT extended (T-123-19 scope guard). This is Phase 126 (TUIW-01) scope only. Acceptance criterion asserts file is unmodified."
  - "Merge conflict in client_test.go resolved by keeping both ExchangeJoinCode tests (from 123-02) and Plan-04 write round-trip tests. No duplication."
requirements-completed: [FSW-09]
duration: ~6min
completed: 2026-06-14
---

# Phase 123 Plan 04: DaemonClient Write Methods Summary

**Five context-aware DaemonClient write methods (WriteFile, UploadFile, DeleteFile, RenameFile, MkdirFile) that consume the Plan-03 daemon routes, mirroring the existing read-method patterns, with 7 round-trip tests all passing under -race.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-06-14T15:40:44Z
- **Completed:** 2026-06-14T15:46:50Z
- **Tasks:** 2
- **Files modified:** 2 (internal/daemon/client.go, internal/daemon/client_test.go)

## Accomplishments

- `internal/daemon/client.go`: Five `*DaemonClient` write methods added following the Phase 118/Plan 05 ListFiles/StatFile/ReadFile pattern exactly:
  - `WriteFile` — `http.MethodPut`, `Content-Type: application/octet-stream`, body `bytes.NewReader(data)`, decodes `FileWriteResponse`
  - `UploadFile` — builds `multipart/form-data` body with `mime/multipart.NewWriter`, sends dir+file fields, decodes `FileWriteResponse`
  - `DeleteFile` — `http.MethodDelete`, decodes `FileOpResponse`
  - `RenameFile` — JSON marshals `renameRequest{OldRel, NewRel}`, POST with `Content-Type: application/json`, decodes `FileOpResponse`
  - `MkdirFile` — POST via `filesURL("mkdir", ...)`, no body, decodes `FileOpResponse`
  - Each: `http.NewRequestWithContext(ctx, ...)`, non-2xx → `fmt.Errorf("files <op>: %d %s", status, body)`
  - `mime/multipart` import added (only new stdlib import)
  - `renameRequest` struct defined in client.go to avoid import of server-side handler type
- `internal/daemon/client_test.go`: 7 write method tests added (plan RED commit `11d6142`):
  - `TestDaemonClientWrite_RoundTrip`: write → read-back byte equality
  - `TestDaemonClientUpload_RoundTrip`: multipart upload → read-back byte equality
  - `TestDaemonClientDelete_RoundTrip`: delete → stat returns error
  - `TestDaemonClientRename_RoundTrip`: rename → new readable, old gone
  - `TestDaemonClientMkdir_RoundTrip`: mkdir → appears in directory listing
  - `TestDaemonClientWrite_NonOKError`: traversal path → 403 typed error
  - `TestDaemonClientWrite_ContextCancel`: cancelled ctx → error (no hang)
- `FilesClient` interface in `internal/tui/files_client.go` is unmodified — still exactly 4 methods (scope guard T-123-19 honored)
- Full `go test -race ./internal/files/... ./internal/daemon/...` green
- `gofmt -l internal/daemon/client.go` clean (no output)

## Task Commits

1. **Task 1 RED: failing tests for write methods** — `11d6142` (test)
2. **Task 1 GREEN: implement write methods** — `e307702` (feat)
3. **Merge resolution: bring prerequisite Phase 123 files** — `f9339ed` (chore)

Note: Task 2 (full race gate + format) required no additional code commit — the suite was already green and gofmt clean from Task 1's implementation.

## Files Created/Modified

- `internal/daemon/client.go` (MODIFIED) — WriteFile, UploadFile, DeleteFile, RenameFile, MkdirFile methods added; mime/multipart import; renameRequest struct
- `internal/daemon/client_test.go` (MODIFIED) — 7 write round-trip + error + cancel tests added; ExchangeJoinCode tests from 123-02 preserved in merge resolution

## Decisions Made

- UploadFile builds a multipart/form-data body client-side (mime/multipart.NewWriter + CreateFormFile) matching the server-side handler.go form field contract (dir + file). This is stdlib-only — no new dependency.
- RenameFile sends {oldRel, newRel} as a JSON body (not query params) to handle Unicode filenames with special characters safely. Mirrors the server-side renameRequest struct exactly.
- MkdirFile passes the target path via filesURL (query param) not a request body — the server reads it via h.relPath(r) just like read handlers. No JSON wrapper needed.
- FilesClient interface in internal/tui/files_client.go is NOT extended (T-123-19 scope guard). Phase 126 (TUIW-01) scope only.
- Merge conflict in client_test.go resolved by keeping both ExchangeJoinCode tests (from 123-02) and Plan-04 write round-trip tests. No duplication.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Worktree missing Phase 123 prerequisite files**
- **Found during:** Task 1 GREEN (build failed: `files.FileWriteResponse undefined`)
- **Issue:** The worktree branch was created before Phase 123 (from `d725107`). Plans 123-01 through 123-03 added `FileWriteResponse`/`FileOpResponse` wire types, `internal/files/write.go`, and the five daemon write routes — all of which the client methods depend on. These files existed on `main` but not in the worktree's working tree.
- **Fix:** Merged `main` into the worktree branch. Conflict in `client_test.go` was resolved additively — both sets of tests (ExchangeJoinCode from 123-02 and write round-trip from 123-04) were kept with no duplication.
- **Files affected:** Merge added all Phase 123 source files to worktree: `internal/files/write.go`, `internal/files/types.go` (write types), `internal/daemon/api.go` (5 routes), etc.
- **Commit:** `f9339ed`

## Known Stubs

None — all five DaemonClient write methods are fully implemented and round-trip tested.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes beyond what the plan's threat model covers. The client methods operate over the existing daemon Unix socket with no new trust boundaries.

| Threat ID | Mitigation | Evidence |
|-----------|-----------|---------|
| T-123-17 | Information Disclosure: server body echoed in error — accepted (loopback socket, no secrets) | matches existing read-method error convention |
| T-123-18 | DoS: context cancellation aborts request (no hang) | TestDaemonClientWrite_ContextCancel passes |
| T-123-19 | Tampering: FilesClient interface NOT extended | git diff shows internal/tui/files_client.go unmodified |

## Self-Check: PASSED

- `internal/daemon/client.go` with 5 write methods FOUND
- `internal/daemon/client_test.go` with 7 write tests FOUND
- commit `11d6142` (RED tests) FOUND
- commit `e307702` (GREEN implementation) FOUND
- commit `f9339ed` (merge resolution) FOUND
- `go test -race ./internal/daemon/ -run 'TestDaemonClientWrite|TestDaemonClientUpload|TestDaemonClientDelete|TestDaemonClientRename|TestDaemonClientMkdir'` — 7/7 PASS
- `go test -race ./internal/files/... ./internal/daemon/...` — PASS (full suite)
- `gofmt -l internal/daemon/client.go` — clean (no output)
- grep: all 5 `func (c *DaemonClient) WriteFile/UploadFile/DeleteFile/RenameFile/MkdirFile` found in client.go, each with `ctx context.Context` first param
- `git diff HEAD -- internal/tui/files_client.go` — no output (file unmodified, still 4 methods)
