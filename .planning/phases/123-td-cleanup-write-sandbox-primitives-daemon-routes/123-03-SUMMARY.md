---
phase: 123-td-cleanup-write-sandbox-primitives-daemon-routes
plan: "03"
subsystem: internal/files
tags: [http-handlers, write-routes, multipart-upload, denylist, atomic-write, daemon-socket, security]
dependency_graph:
  requires:
    - phase: 123-01
      provides: WriteFileAtomic/Rename/Mkdir/MkdirAll/Delete/denylistCheck/ErrProtectedSystemFile + FileWriteResponse/FileOpResponse wire types
  provides:
    - Handler.Write HTTP method (PUT /api/files/write)
    - Handler.Upload HTTP method (POST /api/files/upload) with 50 MiB MaxBytesReader cap
    - Handler.Delete HTTP method (DELETE /api/files/delete)
    - Handler.Rename HTTP method (POST /api/files/rename) with JSON body
    - Handler.Mkdir HTTP method (POST /api/files/mkdir)
    - writeWriteError error mapper (ErrProtectedSystemFile → 403; traversal → 403; other → 500)
    - Five method-prefixed auth-less write routes on daemon Unix socket (FSW-08)
  affects:
    - Phase 124 (capability gating — files.write will gate these routes on the webserver surface)
    - Phase 125 (CodeMirror editor wires through Write/Upload)
    - Phase 126 (TUI write surface calls DaemonClient write methods)
tech_stack:
  added: []
  patterns:
    - HTTP write handlers as sibling file (write.go) to read handlers (handler.go) on the same *Handler type
    - MaxBytesReader(50MiB) BEFORE ParseMultipartForm — enforces upload cap before any bytes hit disk
    - filepath.Base filename sanitization on multipart upload (strips traversal components)
    - writeWriteError error mapper reusing read-side 403/404/400 status convention
    - Five method-prefixed daemon routes (Go 1.22+ mux auto-405 for wrong verb)
    - auth-less loopback-trust pattern (WEB-01 precedent) for all write routes
key_files:
  created:
    - internal/files/write.go
  modified:
    - internal/daemon/api.go
    - internal/files/handler_test.go
    - internal/daemon/api_test.go
key-decisions:
  - "Rename handler decodes {oldRel, newRel} from JSON body (not query params) — safe for arbitrary Unicode filenames with special characters that would require URL-encoding in query params."
  - "Mkdir uses sb.MkdirAll (not sb.Mkdir) so creating nested directories does not require a separate request per parent. isValidationError uses 'files: ' prefix heuristic matching the package-wide error string convention."
  - "isValidationError uses 'files: ' string prefix to detect sandbox validation errors without wrapping them in a new sentinel type — matches the existing package error-string convention."
requirements-completed: [FSW-05, FSW-08, FSW-12]
duration: ~30min
completed: 2026-06-14
---

# Phase 123 Plan 03: HTTP Write Handlers + Daemon Routes Summary

**Five auth-less method-prefixed write routes on the daemon Unix socket (PUT write, POST upload, DELETE delete, POST rename, POST mkdir) with 50 MiB upload cap, denylist 403, and filepath.Base filename sanitization, completing the FSW HTTP surface.**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-06-14T00:00:00Z
- **Completed:** 2026-06-14
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `internal/files/write.go`: Stateless `*Handler` write methods (Write/Upload/Delete/Rename/Mkdir) mirroring the read-side session-resolution and status-mapping convention from handler.go exactly. All five methods open with the `sandboxFor` → 404 block. `writeWriteError` maps `ErrProtectedSystemFile` → 403, validation errors → 403, other → 500.
- `internal/daemon/api.go`: Five method-prefixed write routes registered immediately after the existing read routes (PUT write, POST upload, DELETE delete, POST rename, POST mkdir). Auth-less per WEB-01 loopback-trust precedent. Loopback-trust doc comment extended one line referencing Phase 123-03.
- Full test coverage: 12 tests (9 handler-level in `handler_test.go`, 3 route-level in `api_test.go`) covering the denylist (write and upload layers), traversal rejection, filepath.Base sanitization, 50 MiB over-cap, byte-identical round-trip, wrong-verb 405, and all five route registrations.
- Full `go test -race ./internal/files/... ./internal/daemon/...` green; `gofmt -l` clean; `go vet` clean.

## Task Commits

1. **Task 1 RED: failing write-handler and route tests** - `b21bb0e` (test)
2. **Task 1 GREEN: write handlers + daemon routes** - `762525e` (feat)
3. **Task 2: full race gate + format** - no additional code changes needed; suite was green

## Files Created/Modified

- `internal/files/write.go` (NEW) — Handler.Write/Upload/Delete/Rename/Mkdir + writeWriteError + isValidationError + const maxUploadBytes = 50<<20
- `internal/daemon/api.go` (MODIFIED) — five method-prefixed write routes added after read routes; loopback-trust comment extended
- `internal/files/handler_test.go` (MODIFIED) — 9 write-handler tests added (TestHandlerWrite_RoundTrip, _DenylistForbidden, _Traversal403; TestHandlerUpload_DenylistForbidden, _FilenameSanitized, _OverCap413; TestHandlerRename, TestHandlerMkdir, TestHandlerDelete)
- `internal/daemon/api_test.go` (MODIFIED) — 3 route-level tests added (TestFilesWriteRoutes_WrongVerb405, _Registered, _WriteRoundTrip) + rawPut helper

## Decisions Made

- Rename handler decodes `{oldRel, newRel}` from a JSON request body rather than query parameters. This safely handles arbitrary Unicode filenames with special characters that would require URL-encoding in query params.
- Mkdir uses `sb.MkdirAll` (not `sb.Mkdir`) so creating deeply-nested directory paths does not require a separate request per parent directory.
- `isValidationError` uses a `"files: "` string prefix check to distinguish sandbox validation errors from OS-level I/O errors. This matches the package-wide error string convention established in sandbox.go (every error starts with `"files: "`). No new sentinel type is needed.
- `TestHandlerUpload_DenylistForbidden` uses a home-rooted sandbox via `newHandlerWithHomeSandbox` so the denylist fires. This proves HTTP-upload-layer denylist enforcement (success criterion #3 evidence gap closed).

## Deviations from Plan

None — plan executed exactly as written. The TDD cycle (RED → GREEN) completed cleanly; all tests passed on first GREEN run.

## Known Stubs

None — all five handler methods are fully implemented, tested, and route-registered.

## Threat Surface Scan

All mitigations from the Plan 123-03 STRIDE register are implemented:

| Threat ID | Mitigation | Evidence |
|-----------|-----------|---------|
| T-123-12 | filepath.Base on multipart FileHeader.Filename | write.go:96; TestHandlerUpload_FilenameSanitized |
| T-123-13 | http.MaxBytesReader(50 MiB) BEFORE ParseMultipartForm | write.go:77-78; TestHandlerUpload_OverCap413 |
| T-123-14 | ErrProtectedSystemFile → 403 via writeWriteError | write.go:133-135; TestHandlerWrite_DenylistForbidden + TestHandlerUpload_DenylistForbidden |
| T-123-15 | Method-prefixed mux auto-405 for wrong verb | api.go:149-153; TestFilesWriteRoutes_WrongVerb405 |
| T-123-16 | Daemon socket is Unix socket (loopback boundary); webserver/CSRF gating deferred to Phase 124 | api.go comment; explicitly out of scope |

No new threat surface was introduced beyond what the plan's threat model covers.

## Self-Check: PASSED

- `internal/files/write.go` FOUND
- `internal/daemon/api.go` (with 5 write routes) FOUND
- `internal/files/handler_test.go` (with 9 write tests) FOUND
- `internal/daemon/api_test.go` (with 3 route tests) FOUND
- commit b21bb0e (RED tests) FOUND
- commit 762525e (GREEN implementation) FOUND
- `go test -race ./internal/files/ ./internal/daemon/ -run 'TestHandlerWrite|TestHandlerUpload|TestHandlerRename|TestHandlerMkdir|TestHandlerDelete|TestFilesWriteRoutes'` PASS (12/12)
- `go test -race ./internal/files/... ./internal/daemon/...` PASS (full suite)
- `gofmt -l internal/files/write.go internal/daemon/api.go` clean (no output)
- `go vet ./internal/files/... ./internal/daemon/...` clean
- grep shows all 5 `a.filesHandler.*` write registrations with explicit verb prefixes in api.go
- grep shows `http.MaxBytesReader` at line 77 appears before `ParseMultipartForm` at line 78 in write.go
- Upload uses `filepath.Base(header.Filename)` and routes through `sb.WriteFileAtomic`
- `TestHandlerUpload_DenylistForbidden` asserts HTTP 403 with "Protected system file" (upload-layer denylist evidence)
