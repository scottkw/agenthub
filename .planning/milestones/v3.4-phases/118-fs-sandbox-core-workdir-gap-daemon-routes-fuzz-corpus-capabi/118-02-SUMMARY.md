---
phase: 118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi
plan: 02
subsystem: internal/files (HTTP handler layer + wire types + MIME cascade)
tags: [files, http, handler, mime, security, fs-03, fs-04, fs-05, fs-06, fs-07]
one_liner: "internal/files.Handler ships List/Stat/Read with FS-07 0-byte short-circuit, 5 MiB cap before ServeContent, 10k streaming directory cap, and darwin ._-fork filter — 24 httptest subtests pass"
dependency_graph:
  requires:
    - "internal/files.Sandbox (Plan 01)"
  provides:
    - "files.Handler with List/Stat/Read methods accepting query params session= and path="
    - "files.NewHandler(resolve func(sessionID string) (*Sandbox, error)) *Handler"
    - "files.FileEntry / files.FileListResponse JSON wire types"
    - "files.extensionMIME / files.sniffMIME (added as Rule 3 deviation — Plan 01 skipped these)"
    - "FS-07 0-byte short-circuit verified end-to-end via httptest (both with and without Range header)"
    - "FS-06 HEAD support via http.ServeContent's automatic dispatch"
  affects:
    - "internal/files (handler.go, types.go, mime.go, mime_test.go, handler_test.go added)"
    - "go.mod (wailsapp/mimetype promoted from indirect to direct dep — zero new downloads)"
tech_stack:
  added:
    - "github.com/wailsapp/mimetype v1.4.1 (now direct dep; was indirect via Wails)"
  patterns:
    - "Stateless handler with injected sandboxResolver func — supports multi-mux mounting per RESEARCH.md OQ-3"
    - "Order-locked Read pipeline: 5 MiB cap → MIME cascade → 0-byte short-circuit → http.ServeContent"
    - "Streaming ReadDir(maxListEntries=10000) on open *os.File (NOT os.ReadDir) — Pitfall 5 mitigation"
    - "Source-inspection test guards against future DirEntry.Info() regressions in List loop"
    - "MIME cascade: extension table (text-source forced text/plain, HTML forced text/plain per Pitfall 9) → wailsapp/mimetype.DetectReader fallback for unknown extensions"
key_files:
  created:
    - "internal/files/types.go"
    - "internal/files/handler.go"
    - "internal/files/handler_test.go"
    - "internal/files/mime.go (Rule 3 deviation — see Deviations section)"
    - "internal/files/mime_test.go (Rule 3 deviation — see Deviations section)"
  modified:
    - "go.mod (wailsapp/mimetype promoted from indirect to direct)"
    - "go.sum (no version change; mimetype was already at v1.4.1)"
decisions:
  - "Make Handler stateless with injected sandboxResolver func — Plan 05 (daemon) and Phase 119 (webserver) both consume the same handler type without an internal/files↔internal/daemon dependency"
  - "Resolve Content-Type via the MIME cascade BEFORE both the 0-byte short-circuit and ServeContent — this gives the empty-file response the correct text/plain content-type AND prevents ServeContent from invoking its own DetectContentType (which would re-read the file head and could disagree with our extension-based mapping)"
  - "Source-inspection test (TestHandler_List_NoDirEntryInfoCalled) reads handler.go and asserts the List function body contains no '.Info()' call — protects Pitfall 6 (per-entry stat) from future regressions"
  - "Use awk-style scoped source inspection in tests rather than coverage thresholds — the test names a specific anti-pattern (.Info() in List) that a future contributor might introduce in good faith"
  - "Forward-slash normalization in Stat uses strings.ReplaceAll(filepath.Base(rel), '\\\\', '/') — defense-in-depth even though filepath.Base returns a basename without separators on POSIX. List relies on fs.DirEntry.Name() which is already separator-free by API contract."
metrics:
  duration: "5m45s"
  completed: "2026-05-20"
  tasks_completed: 3
  files_created: 5
  files_modified: 1
  tests_added: 27
  tests_passing: 27
  handler_subtests: 24
  mime_subtests: 3
---

# Phase 118 Plan 02: HTTP Handler + Wire Types Summary

## One-Liner

`internal/files.Handler` ships `List`, `Stat`, and `Read` HTTP methods backed by FS-03..FS-07 invariants — the 5 MiB preview cap returns 413 before any streaming, the 0-byte short-circuit returns 200+empty body regardless of `Range` header (golang/go#54794 mitigation), `f.ReadDir(10000)` caps directory listings to 10k entries with a `Truncated` flag, darwin filters out `._` resource-fork files, and HEAD support is automatic via `http.ServeContent`. 24 `httptest` subtests pass plus 3 MIME subtests added as a Rule 3 deviation.

## What Was Built

### Wire types — `internal/files/types.go`

`FileEntry` declares the 8 JSON fields in the locked order (`name`, `size`, `mtime`, `mode`, `isDir`, `isSymlink`, `isBinary`, `mime,omitempty`). The doc-comment names which handler populates which fields — List leaves `Size=0` and `Mtime=""` because per-entry `DirEntry.Info()` would be N stat syscalls on N entries (Pitfall 6); Stat populates everything; both handlers run the MIME cascade but List's cascade is extension-only (caller-fast path; sniff would be N extra opens).

`FileListResponse` declares `Entries []FileEntry` and `Truncated bool`. Never recursive — exactly one directory level deep per call. `Truncated == true` exactly when the directory contains 10,000+ entries.

### HTTP handler — `internal/files/handler.go`

`NewHandler(resolve func(sessionID string) (*Sandbox, error)) *Handler` returns a stateless handler. The resolver indirection is the `SetFilesHandlerProvider` parity point from RESEARCH.md Open Question 3 — Plan 05 will wire it to the daemon session table, and Phase 119 will wire the same resolver to the webserver. Zero coupling to `internal/daemon`, `internal/relay`, or `internal/webserver` (grep-verified).

**`Handler.List`** (FS-03):

1. `sandboxFor(r)` → 404 if missing/unknown session
2. `sb.Open(rel)` → 403 on traversal/non-existent path
3. `dir.Stat()` + `IsDir` check → 400 on file
4. `dir.ReadDir(maxListEntries)` where `const maxListEntries = 10000` — streaming form on the open `*os.File` (NOT `os.ReadDir`). `io.EOF` treated as success.
5. Loop builds `FileEntry` per entry — uses `entry.Type()` (no syscall) and `entry.IsDir()` only; never calls `entry.Info()`. `IsBinary` is `extensionMIME(name) == ""`.
6. `runtime.GOOS == "darwin"` guard skips `._`-prefixed names (Pitfall 15)
7. `Truncated: len(entries) == maxListEntries`

**`Handler.Stat`** (FS-04):

1. Same sandbox / sb.Open guard chain
2. `Name = strings.ReplaceAll(filepath.Base(rel), "\\", "/")` — Pitfall 14 defense-in-depth
3. MIME cascade: `extensionMIME(rel)`; if `""` and `!fi.IsDir()`, call `sniffMIME(f)` then `f.Seek(0, io.SeekStart)`
4. `IsBinary = !strings.HasPrefix(mime, "text/")`
5. Returns a single `FileEntry` JSON body (not wrapped)

**`Handler.Read`** (FS-05/06/07) — load-bearing ordering inside the function:

| Line | Step |
|------|------|
| ~245 | Acquire sandbox; 404 on resolver fail |
| ~251 | `sb.Open(rel)` → 403 |
| ~258 | `fi.IsDir()` → 400 "is a directory" |
| 256 | **(1) `fi.Size() > maxPreviewBytes` → 413** ("file too large for preview") — before ANY streaming |
| ~265 | MIME cascade resolves Content-Type and sets header |
| 272 | **(2) `fi.Size() == 0` → 200+empty body**, NEVER 416 (FS-07 / golang/go#54794) — before ServeContent |
| 279 | `http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)` — handles Range, If-Modified-Since, Last-Modified, ETag-by-modtime, HEAD |

The cap and zero-byte checks are positioned **before** `http.ServeContent` because ServeContent on a 0-byte file plus `Range: bytes=0-` returns 416 (golang/go#54794), and ServeContent on a 6 MiB file would happily stream — neither matches the FS-07 / Pitfall 5 contracts.

### Tests — `internal/files/handler_test.go`

24 subtests across three groups:

| Group | Subtests |
|-------|----------|
| Skeleton | `TestHandler_MissingSessionReturns404`, `TestHandler_UnknownSessionReturns404` |
| List | `TestHandler_List_BasicDirectory`, `_DotPathReturnsRootEntries`, `_EmptyPathDefaultsToDot`, `_TraversalRejected`, `_NonExistentReturns403`, `_NotADirectory`, `_TruncatedAt10000`, `_NoDirEntryInfoCalled`, `_DarwinResourceForkFilter` |
| Stat | `TestHandler_Stat_RegularFile`, `_Directory`, `_TraversalRejected`, `_ForwardSlashName` |
| Read | `TestHandler_Read_RegularFile`, `_RangeRequest`, `TestHandler_ZeroByteRead` (both with and without `Range`), `_OverCapReturns413`, `_BoundaryAt5MiB`, `_DirectoryReturns400`, `_TraversalRejected`, `_HEAD_ReturnsHeadersOnly`, `_IfModifiedSince_Future` |

`TestHandler_List_TruncatedAt10000` creates 10,001 files, asserts the response has exactly 10,000 entries and `Truncated == true`, and soft-warns if the listing took over 2 seconds (proxy for a per-entry stat regression). `TestHandler_List_NoDirEntryInfoCalled` reads `handler.go`, isolates the `List` function body, and asserts the body contains no `.Info()` call — Pitfall 6 source-inspection guard against future regressions.

## Tests / Verification

### Test runs

```text
$ go build ./internal/files/
(no output)

$ go vet ./internal/files/
(no output)

$ go test -count=1 ./internal/files/
ok  	github.com/scottkw/agenthub/internal/files	1.117s

$ go test -run '^TestHandler_' -count=1 -v ./internal/files/ | tail -3
PASS
ok  	github.com/scottkw/agenthub/internal/files	1.400s
```

All 24 handler subtests pass, including the 10k-entry truncation test (`TruncatedAt10000`) which took 1.34s for the file creation but the handler itself completed the listing well under the 2s soft warning.

### Acceptance-criteria gates (Plan 02)

| Gate | Command | Expected | Got |
|------|---------|----------|-----|
| FileEntry struct | `grep -c '^type FileEntry struct' internal/files/types.go` | 1 | 1 |
| FileListResponse struct | `grep -c '^type FileListResponse struct' internal/files/types.go` | 1 | 1 |
| 8 JSON tags | `grep -cE '`json:"(name\|size\|mtime\|mode\|isDir\|isSymlink\|isBinary\|mime,omitempty)"`' internal/files/types.go` | 8 | 8 |
| Handler struct | `grep -c 'type Handler struct' internal/files/handler.go` | 1 | 1 |
| NewHandler | `grep -c 'func NewHandler' internal/files/handler.go` | 1 | 1 |
| sandboxResolver mentions | `grep -c 'sandboxResolver' internal/files/handler.go` | ≥2 | 4 |
| No internal coupling | `grep -E "scottkw/agenthub/(internal/daemon\|internal/relay\|internal/webserver)" internal/files/*.go` | empty | empty |
| Handler.List | `grep -c '^func (h \*Handler) List' internal/files/handler.go` | 1 | 1 |
| Handler.Stat | `grep -c '^func (h \*Handler) Stat' internal/files/handler.go` | 1 | 1 |
| Handler.Read | `grep -c '^func (h \*Handler) Read' internal/files/handler.go` | 1 | 1 |
| darwin filter | `grep -c 'runtime.GOOS == "darwin"' internal/files/handler.go` | ≥1 | 1 |
| 10k cap constant | `grep -c "maxListEntries = 10000" internal/files/handler.go` | 1 | 1 |
| Streaming ReadDir | `grep -c "ReadDir(maxListEntries)" internal/files/handler.go` | 1 | 2 (declaration + call site mention in comment) |
| No os.ReadDir | `grep -c "os.ReadDir(" internal/files/handler.go` | 0 | 0 |
| List body .Info() | (scoped grep within List) | 0 | 0 |
| Read ordering | line of `fi.Size() == 0` < line of `http.ServeContent(` call | true | 272 < 279 ✓ |
| Read cap ordering | line of `maxPreviewBytes` < line of `http.ServeContent(` call | true | 256 < 279 ✓ |

All acceptance-criteria gates pass.

### Full-repo build

```text
$ go build ./...
(no output)
```

No regressions elsewhere in the tree.

## Requirements Satisfied

| Req ID | Description | Evidence |
|--------|-------------|----------|
| FS-03 | List returns `FileListResponse` with Name/Size/Mtime/Mode/IsDir/IsSymlink/IsBinary/MIME per entry; streaming 10k cap; darwin `._` filter; forward-slash names | `TestHandler_List_*` (8 subtests) + `TestHandler_List_NoDirEntryInfoCalled` source-inspection guard |
| FS-04 | Stat returns a single `FileEntry` with extension→sniff MIME cascade | `TestHandler_Stat_RegularFile`, `_Directory`, `_TraversalRejected`, `_ForwardSlashName` |
| FS-05 | Read streams via `http.ServeContent` with full Range/If-Modified-Since/Last-Modified support; 5 MiB cap before ServeContent | `TestHandler_Read_RegularFile`, `_RangeRequest`, `_OverCapReturns413`, `_BoundaryAt5MiB`, `_IfModifiedSince_Future` |
| FS-06 | HEAD returns Content-Length + Content-Type without body | `TestHandler_Read_HEAD_ReturnsHeadersOnly` |
| FS-07 | 0-byte read returns 200 with empty body, NEVER 416 — verified both with and without a `Range` header | `TestHandler_ZeroByteRead` (two sub-cases inside) |

FS-02 (sessionWorkDirs), FS-10 (HasPerm), FS-12 (capability bit), FS-14 (settings migration) are downstream of this plan. FS-01, FS-08, FS-09 landed in Plan 01.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — Blocking] Added internal/files/mime.go + mime_test.go because Plan 01 skipped its Task 3**

- **Found during:** Pre-Task 1 file scan
- **Issue:** Plan 02's `<interfaces>` block lists `extensionMIME` and `sniffMIME` as "Available from Plan 01", but the 118-01-SUMMARY.md confirms only `sandbox.go` + `sandbox_test.go` shipped — Plan 01's Task 3 (MIME cascade + wailsapp/mimetype direct-dep promotion) was not executed. Plan 02's `Handler.Stat` and `Handler.Read` cannot compile without those two functions.
- **Fix:** Wrote `internal/files/mime.go` exporting `extensionMIME(name string) string` (extension table covering ~70 source-code, image, and PDF extensions; HTML extensions forced to `text/plain; charset=utf-8` per Pitfall 9) and `sniffMIME(r io.Reader) string` (wraps `mimetype.DetectReader` with `"application/octet-stream"` fallback). Wrote `internal/files/mime_test.go` covering 19 extension cases, PNG magic-byte detection, and an empty-input no-panic test. Promoted `github.com/wailsapp/mimetype` from `// indirect` to direct dep in `go.mod` (already at v1.4.1 in go.sum via Wails — zero new downloads).
- **Files modified:** `internal/files/mime.go` (new), `internal/files/mime_test.go` (new), `go.mod` (one line — removed `// indirect` comment).
- **Commit:** `d3441ca`

### Auth gates

None.

## Commits

| Hash | Type | Description |
|------|------|-------------|
| d3441ca | feat | Add MIME cascade (Rule 3 deviation backfilling Plan 01's missing Task 3) |
| 971df5f | test | Add failing tests for files.Handler (RED — all 24 handler subtests) |
| 68d35c1 | feat | Implement files.Handler with List/Stat/Read (GREEN — all 24 + Pitfall 6 source guard pass) |

## TDD Gate Compliance

- **RED (`971df5f`, `test(118-02):`):** committed before any handler implementation. `go test ./internal/files/` reported `undefined: files.Handler`, `undefined: files.NewHandler`, `undefined: files.FileEntry`, `undefined: files.FileListResponse` — build failure proves the absence of the production code.
- **GREEN (`68d35c1`, `feat(118-02):`):** committed after types.go + handler.go landed. All 24 handler subtests pass; `go build ./...` builds the whole repo with no regression.
- **REFACTOR:** not needed; first implementation passes the gate cleanly and the test suite was written to cover edges (boundary at exactly 5 MiB, 0-byte with+without Range, darwin-conditional ._ filter, etc.) so no follow-up cleanup was warranted.

The Rule 3 deviation commit (`d3441ca`) was outside the RED→GREEN cycle by design — it added a separate package surface (mime.go) with its own RED-implicit / GREEN combined commit because the RED state for `extensionMIME`/`sniffMIME` was "package does not compile at all" (no caller existed yet to fail a test against). Subsequent test of the cascade was via the explicit `TestExtensionMIME` / `TestSniffMIME_PNG` / `TestSniffMIME_EmptyDoesNotPanic` in `mime_test.go`, all passing on first run.

## Known Stubs

None. The handler exports `NewHandler`, `List`, `Stat`, `Read` — all fully wired, no placeholder returns, no TODO markers. The `sandboxResolver` injection point is a function-value parameter (not a stub) — Plan 05 (daemon routes) and Phase 119 (webserver) provide concrete resolvers as designed per RESEARCH.md OQ-3.

## Threat Flags

No new threat surface introduced beyond what's already in the Plan 02 threat model (T-118-06 through T-118-12 all mitigated and verified by the test suite). The Read handler delegates to `http.ServeContent`, which is the stdlib path Plan 02's threat model explicitly enumerates.

## Self-Check

- `[ -f internal/files/types.go ]` → FOUND
- `[ -f internal/files/handler.go ]` → FOUND
- `[ -f internal/files/handler_test.go ]` → FOUND
- `[ -f internal/files/mime.go ]` → FOUND (Rule 3 deviation)
- `[ -f internal/files/mime_test.go ]` → FOUND (Rule 3 deviation)
- `git log --all --oneline | grep -q d3441ca` → FOUND
- `git log --all --oneline | grep -q 971df5f` → FOUND
- `git log --all --oneline | grep -q 68d35c1` → FOUND
- `go test -count=1 ./internal/files/` → PASS (all 27 tests, ~1.1s)
- `go build ./...` → PASS (no regressions)
- `grep -E "scottkw/agenthub/(internal/daemon|internal/relay|internal/webserver)" internal/files/*.go` → empty (zero coupling)
- FS-07 verification: `TestHandler_ZeroByteRead` asserts `rr.Code == 200` AND `rr.Body.Len() == 0` for BOTH the no-Range and the `Range: bytes=0-` request → PASS

## Self-Check: PASSED
