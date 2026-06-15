# Phase 123: TD Cleanup + Write Sandbox Primitives + Daemon Routes - Pattern Map

**Mapped:** 2026-06-14
**Files analyzed:** 9 (6 modified, 3 new)
**Analogs found:** 9 / 9 (every new/modified file has a verified read-side analog in the live codebase)

> All analogs are v3.4 read-side code in the SAME packages the writes extend. This is a "mirror the existing read primitive on the write surface" phase — there is no greenfield. Match the existing file's conventions exactly (doc-comment density, error-string shape, per-op `os.OpenRoot`, method-prefixed routes). Go conventions per CLAUDE.md: `go fmt`, `golangci-lint`, context-aware client functions (`ctx context.Context`).

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/files/sandbox.go` (EXTEND: `WriteFileAtomic`, `Rename`, `Mkdir`, `MkdirAll`, `Delete`, `denylistCheck`) | model/security primitive | file-I/O (write) | same file — `Sandbox.Open`/`Stat` (sandbox.go:68-95) | exact (same type, same per-op `os.OpenRoot` pattern) |
| `internal/files/write.go` (NEW: `Handler.Write/Upload/Delete/Rename/Mkdir` + `writeWriteError`) | controller (HTTP handler) | request-response + file-I/O + multipart | `handler.go` `Handler.Read`/`List`/`Stat` (handler.go:110-303) | exact (same `Handler`, same `sandboxFor`, same 403/404/400 status convention) |
| `internal/files/types.go` (EXTEND: `FileWriteResponse`, `FileOpResponse`) | model (wire type) | transform | same file — `FileEntry`/`FileListResponse` (types.go:26-46) | exact |
| `internal/files/sandbox_test.go` (EXTEND: `FuzzSandboxWrite`) | test (fuzz harness) | batch/property | same file — `FuzzSandboxPath` (sandbox_test.go:256-339) | exact (reuse the corpus) |
| `internal/files/write_test.go` (NEW: round-trip/atomic/concurrent tests) | test (unit) | file-I/O | `sandbox_test.go` table tests + `FuzzSandboxPath` body | role-match |
| `internal/daemon/api.go` (EXTEND: register 5 write routes) | route registration | request-response | same file — read-route block (api.go:144-147) | exact (method-prefixed mux, loopback-trust comment) |
| `internal/daemon/client.go` (EXTEND: `WriteFile/UploadFile/DeleteFile/RenameFile/MkdirFile`) | client (in-process consumer) | request-response | same file — `ListFiles`/`StatFile`/`ReadFile` (client.go:381-449) + `doJSON` (client.go:488) | exact |
| `internal/daemon/client_remote_files.go` (FIX TD-5: `ExchangeJoinCodeAtURL` 303 parse) | client (HTTP redirect) | request-response (303) | `internal/tui/joincode_prompt.go` `exchangeJoinCodeCmd` (joincode_prompt.go:153-208) | exact (reference impl — port the logic) |
| TD-4 WR-01..05 (webserver/server.go + frontend) | mixed (server route + React/TS) | request-response + UI | self-referential (the items ARE in these files) | n/a — see TD-4 section |

---

## Pattern Assignments

### `internal/files/sandbox.go` — write primitives (FSW-01..04, FSW-06)

**Analog:** `internal/files/sandbox.go:68-112` (`Sandbox.Open`, `Sandbox.Stat`, `validateAndClean`) — the SAME file. Every new write method is the read method's structure with a write syscall substituted.

**Imports already present** (sandbox.go:17-23) — `WriteFileAtomic` adds `crypto/rand` + `encoding/hex` for the temp suffix:
```go
import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    // ADD for WriteFileAtomic:
    // "crypto/rand"
    // "encoding/hex"
)
```

**Core per-op pattern to mirror** (sandbox.go:68-79 — every new method opens this way):
```go
func (s *Sandbox) Open(relPath string) (*os.File, error) {
    cleaned, err := validateAndClean(relPath)   // 1. validate FIRST
    if err != nil {
        return nil, err
    }
    root, err := os.OpenRoot(s.rootPath)         // 2. fresh os.OpenRoot per op — DO NOT cache the handle
    if err != nil {
        return nil, fmt.Errorf("files: open root: %w", err)
    }
    defer root.Close()                           // 3. always close
    return root.Open(cleaned)                    // 4. delegate to the native os.Root method
}
```
Write methods insert the denylist check between steps 1 and 2 (`if err := s.denylistCheck(cleaned); err != nil { return err }`) and substitute the native write method at step 4: `root.OpenFile`+`Rename` (WriteFileAtomic), `root.Rename` (Rename), `root.MkdirAll` (Mkdir/MkdirAll), `root.RemoveAll`/`root.Remove` (Delete). RESEARCH §Pattern 1/2/4 give the exact bodies; the `crypto/rand` temp-suffix idiom matches `internal/daemon/api.go:1034`.

**Validation seam to reuse — do not reimplement** (sandbox.go:100-112): every write method calls `validateAndClean(relPath)`, the SAME function the read side uses. For `Rename`, call it on BOTH `oldRel` AND `newRel` (RESEARCH Pitfall 1 — destination validation is the #1 write traversal risk). The device-name/ADS/UNC/null-byte rejections in `validateRelativePath` (sandbox.go:146-195) are inherited automatically.

**Error sentinel pattern to mirror** (sandbox.go uses package-level `errors.New` vars indirectly; introduce one for the denylist): `var ErrProtectedSystemFile = errors.New("files: protected system file")` — the `"files: ..."` prefix matches every existing error string in the file (e.g. sandbox.go:109, :183).

**Doc-comment convention:** every existing method has a 3-6 line doc comment citing the requirement/pitfall (see sandbox.go:59-67, :81-83). Match this density — cite FSW-01/02/03/04/06 and the relevant RESEARCH pattern.

---

### `internal/files/write.go` — HTTP handlers (FSW-05, FSW-08, FSW-12)

**Analog:** `internal/files/handler.go:110-303` (`List`, `Stat`, `Read`) — same `Handler` type, same package, NEW sibling file.

**Handler is stateless and reused** — do NOT create a new Handler type. Add methods to the existing `*Handler` (handler.go:55-65). The `sandboxFor` (handler.go:70-83) and `relPath` (handler.go:87-93) helpers are already there; reuse them verbatim.

**Session-resolution + status-mapping convention to mirror** (handler.go:110-131 — the load-bearing 404/403/400 contract):
```go
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    sb, _, err := h.sandboxFor(r)
    if err != nil {
        http.Error(w, "session not found", http.StatusNotFound)   // 404 unknown session
        return
    }
    rel := h.relPath(r)
    dir, err := sb.Open(rel)
    if err != nil {
        http.Error(w, "access denied: "+err.Error(), http.StatusForbidden) // 403 validation fail
        return
    }
    // ... 400 "not a directory" for shape errors ...
}
```
Every write handler opens with this exact `sandboxFor` → 404 block. Validation/denylist errors map to 403 via the `writeWriteError` helper (RESEARCH §Pattern 5) — keep the `"access denied: "` prefix for traversal and `"Protected system file"` for the denylist sentinel (success criterion #3).

**JSON response convention** (handler.go:186-187, :239-240):
```go
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(result)
```
Use this for `FileWriteResponse`/`FileOpResponse`. (RESEARCH §Pattern 3 shows a `writeJSON(w, status, v)` helper — either inline this two-liner or add the tiny helper; the inline form matches the existing read handlers exactly.)

**Upload-specific (FSW-05/FSW-12)** — no read-side analog for multipart; follow RESEARCH §Pattern 3 (`http.MaxBytesReader` BEFORE `ParseMultipartForm`, `filepath.Base` on `header.Filename`, route through `sb.WriteFileAtomic`). The 5 MiB read cap idiom in handler.go:104 (`const maxPreviewBytes`) is the template for `const maxUploadBytes = 50 << 20`.

**Imports** — start from handler.go:31-41 (`encoding/json`, `errors`, `io`, `net/http`, `path/filepath`); add `mime/multipart` is NOT needed directly (`r.FormFile` covers it).

---

### `internal/files/types.go` — wire types (FSW-01, FSW-08)

**Analog:** `internal/files/types.go:26-46` (`FileEntry`, `FileListResponse`) — same file.

**Pattern** (struct + JSON tags in declaration order, doc comment explaining field semantics):
```go
type FileListResponse struct {
    Entries   []FileEntry `json:"entries"`
    Truncated bool        `json:"truncated"`
}
```
Add `FileWriteResponse{ Path string `json:"path"`; Size int64 `json:"size"` }` and `FileOpResponse{ Path string `json:"path"`; OK bool `json:"ok"` }` (or per planner's design — RESEARCH §Pattern 3 uses `FileWriteResponse{Path, Size}`). Match the `camelCase` JSON tags (note `isDir`, `isSymlink` at types.go:31-32) and the declaration-order comment at types.go:24-25.

---

### `internal/files/sandbox_test.go` — `FuzzSandboxWrite` (FSW-07)

**Analog:** `internal/files/sandbox_test.go:256-339` (`FuzzSandboxPath`) — same file. The merge gate (`-fuzztime=60s`, 0 crashes) is identical, just `=FuzzSandboxWrite`.

**Corpus reuse:** copy the entire `f.Add(...)` block (sandbox_test.go:257-319 — ~45 seeds covering traversal, encoded, absolute, Windows device names, ADS, null bytes, Unicode tricks, mixed separators). Then append the write-specific seeds from RESEARCH §Code Examples (`../../.ssh/authorized_keys`, `../../.bashrc`, `../../.claude/CLAUDE.md`, temp-name collision probe `foo.txt.agenthub-tmp-deadbeef`).

**Harness scaffold to mirror** (sandbox_test.go:321-339):
```go
root := f.TempDir()
// populate real files so accepted paths can be stat'd to prove in-root
sb, err := files.NewSandbox(root)
if err != nil { f.Fatalf("NewSandbox: %v", err) }
f.Fuzz(func(t *testing.T, rawPath string) {
    // must never panic, never escape root
})
```
The fuzz body exercises each write method (`WriteFileAtomic`, `Rename` source AND destination, `Mkdir`, `Delete`) per RESEARCH §Code Examples, with the in-root assertion mirroring `FuzzSandboxPath`'s "Stat succeeds ⇒ inside root" proxy (sandbox_test.go:338+).

---

### `internal/files/write_test.go` — unit tests (FSW-01..06, FSW-12)

**Analog:** the table-test + fuzz-body conventions in `sandbox_test.go` (same package `files_test`, `files.NewSandbox(f.TempDir())` setup). NEW file. Cover: atomic write (concurrent reader never sees partial — FSW-01), rename both-path validation + cross-dir move (FSW-02), mkdir/delete confinement (FSW-03/04), denylist 403 in a `$HOME`-rooted sandbox (FSW-06), 50 MiB cap (FSW-12). Use the `-race` flag per success criterion #5.

---

### `internal/daemon/api.go` — register 5 write routes (FSW-08)

**Analog:** `internal/daemon/api.go:137-147` (read-route block) — same file, register the new routes immediately after line 147.

**Method-prefixed registration to mirror** (api.go:144-147):
```go
a.mux.HandleFunc("GET /api/files/list", a.filesHandler.List)
a.mux.HandleFunc("GET /api/files/stat", a.filesHandler.Stat)
a.mux.HandleFunc("GET /api/files/read", a.filesHandler.Read)
a.mux.HandleFunc("HEAD /api/files/read", a.filesHandler.Read)
```
Add (RESEARCH §Pattern 5):
```go
a.mux.HandleFunc("PUT /api/files/write", a.filesHandler.Write)
a.mux.HandleFunc("POST /api/files/upload", a.filesHandler.Upload)
a.mux.HandleFunc("DELETE /api/files/delete", a.filesHandler.Delete)
a.mux.HandleFunc("POST /api/files/rename", a.filesHandler.Rename)
a.mux.HandleFunc("POST /api/files/mkdir", a.filesHandler.Mkdir)
```
The `a.filesHandler` field already exists (api.go:54, constructed at api.go:81). The loopback-trust / no-auth / method-prefix-405 doc comment at api.go:137-143 already covers the write routes — extend it one line rather than writing a new block. Add the 405 regression test (`GET /api/files/write` → 405) per RESEARCH Pitfall 6.

---

### `internal/daemon/client.go` — write methods (FSW-09)

**Analog:** `internal/daemon/client.go:381-449` (`ListFiles`, `StatFile`, `ReadFile`) + `filesURL` helper (client.go:366-374) + `doJSON` (client.go:488). Same file.

**Context-aware request + status-as-error pattern to mirror** (client.go:381-399 — note `ctx context.Context` per CLAUDE.md Go convention):
```go
func (c *DaemonClient) ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("list", sessionID, relPath), nil)
    if err != nil { return nil, false, fmt.Errorf("files list: new request: %w", err) }
    resp, err := c.http.Do(req)
    if err != nil { return nil, false, fmt.Errorf("files list: do request: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, false, fmt.Errorf("files list: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
    }
    var out files.FileListResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return nil, false, fmt.Errorf("files list: decode response: %w", err)
    }
    return out.Entries, out.Truncated, nil
}
```
- `WriteFile` (raw bytes, `http.MethodPut`, `Content-Type: application/octet-stream`, body `bytes.NewReader(data)`) — RESEARCH §Code Examples shows the exact body.
- `RenameFile`/`MkdirFile` send a JSON request struct — mirror `doJSON` (client.go:488) which already marshals a body, dials, and maps 4xx→error (used by `RegisterRemoteCap` at client_remote_files.go:144).
- `filesURL` (client.go:366-374) builds the read URLs; extend it or add a sibling for the write ops ("write"/"upload"/"delete"/"rename"/"mkdir").

> **Scope guard (RESEARCH Assumption A2):** Do NOT extend the `FilesClient` interface in `internal/tui/files_client.go` — that is Phase 126 (TUIW-01). Only add concrete `*DaemonClient` methods here.

---

### `internal/daemon/client_remote_files.go` — TD-5 fix (FSW-10)

**Analog (REFERENCE IMPL — port this logic):** `internal/tui/joincode_prompt.go:153-208` (`exchangeJoinCodeCmd`). The TUI already does the 303 parse correctly; the GUI's `ExchangeJoinCodeAtURL` (client_remote_files.go:56-118) is the bug — it JSON-decodes the body on the success path (client_remote_files.go:107-117), but the webserver returns 303 + `Location` with NO body.

**The correct 303 pattern to port** (joincode_prompt.go:155-206):
```go
client := &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{ TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12} },
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse              // DO NOT auto-follow the 303
    },
}
// ... POST form{"code": code} to remoteBaseURL+"/join/exchange" ...
if resp.StatusCode != http.StatusSeeOther {        // expect 303
    return ..., fmt.Errorf("... unexpected status %d", resp.StatusCode)
}
loc := resp.Header.Get("Location")
if strings.Contains(loc, "/join?error=") {         // error-shape first
    kind := strings.TrimPrefix(loc, "/join?error=")
    if i := strings.IndexByte(kind, '&'); i >= 0 { kind = kind[:i] }
    return ..., fmt.Errorf("join exchange: %s", kind)
}
u, err := url.Parse(loc)
capTok := u.Query().Get("cap")                      // extract the cap token
```

**Two critical constraints (RESEARCH Pitfall 5):**
1. **Construct a dedicated `http.Client` inside `ExchangeJoinCodeAtURL`** — do NOT add `CheckRedirect` to the shared package-level `remoteFilesHTTPClient` (client_remote_files.go:37-42), which `RegisterRemoteCap` also uses (client_remote_files.go:124-144). The TUI builds its client inside the closure for exactly this reason.
2. **Preserve the existing error-substring contract** that the modal UI pivots on (documented at client_remote_files.go:48-53): `"expired"`, `"invalid"`, `"not-found"`, `"session-gone"`. The current 4xx-status mapping (client_remote_files.go:89-105) already produces these strings — map the new `/join?error=<kind>` Location onto the same substrings. The TUI's `friendlyJoinCodeError` (joincode_prompt.go:214+) shows the substring-match contract.

---

## Shared Patterns

### Path validation (apply to ALL `internal/files` write methods)
**Source:** `internal/files/sandbox.go:100-112` (`validateAndClean`) + `:146-195` (`validateRelativePath`)
```go
cleaned, err := validateAndClean(relPath)   // device names, ADS, UNC, null bytes, traversal — all inherited
if err != nil { return err }
```
Every write method validates first; `Rename` validates BOTH paths (Pitfall 1). This is the single security authority — `os.Root` is the kernel backstop, not a substitute.

### Per-operation `os.OpenRoot` (apply to ALL `internal/files` write methods)
**Source:** `internal/files/sandbox.go:73-79`
Open a FRESH `os.OpenRoot(s.rootPath)` per call, `defer root.Close()`. Never cache the `*os.Root` handle on the Sandbox (the v3.4 design caches `rootPath`, not the handle — RESEARCH anti-patterns).

### HTTP status convention (apply to ALL `internal/files` handlers)
**Source:** `internal/files/handler.go:111-131`
- 404 `"session not found"` — `sandboxFor` error
- 403 `"access denied: "+err.Error()` — validation/traversal failure
- 403 `"Protected system file"` — denylist sentinel (success criterion #3)
- 400 — shape error ("is a directory" / "not a directory")
- 200 + JSON — success
Mirror the read handlers exactly so the write surface is consistent with v3.4.

### Client request → typed error (apply to ALL `DaemonClient` write methods)
**Source:** `internal/daemon/client.go:381-399`, helper `doJSON` at `:488`
`http.NewRequestWithContext(ctx, ...)` → `c.http.Do` → non-2xx surfaces as `fmt.Errorf("files <op>: %d %s", status, body)`. JSON ops route through `doJSON`.

### Method-prefixed routes (apply to daemon route registration)
**Source:** `internal/daemon/api.go:141-147`
Register with the verb prefix (`"PUT /api/files/write"`); Go 1.22+ mux auto-returns 405 for the wrong verb. Do NOT register a verbless fallback that would mask the 405.

### Loopback trust, no auth (Phase 123 scope guard)
**Source:** `internal/daemon/api.go:137-143` (the doc comment)
Daemon socket write routes are auth-less per WEB-01. Do NOT add `PermFilesWrite`/`requireFilesWrite`/CSRF/Origin checks — those are Phase 124 (RESEARCH §user_constraints, anti-patterns).

---

## No Analog Found

None for the write primitives — every Phase 123 file has a verified v3.4 read-side analog in the same package. Two items have no in-codebase prior art and rely on RESEARCH patterns + stdlib:

| File / Concern | Role | Data Flow | Reason | Use Instead |
|----------------|------|-----------|--------|-------------|
| Upload multipart handling (`write.go` `Upload`) | controller | multipart | No existing multipart endpoint in the codebase | RESEARCH §Pattern 3 (`http.MaxBytesReader` + `r.FormFile` + `filepath.Base`) — pure stdlib |
| Atomic temp+sync+rename (`sandbox.go` `WriteFileAtomic`) | primitive | file-I/O | Read side never writes | RESEARCH §Pattern 1 (verified `os.Root.OpenFile`+`Sync`+`Rename`); temp-suffix idiom mirrors `api.go:1034` `crypto/rand` |

---

## TD-4 (FSW-11) — Source Locations and State Note

> **PLANNER ALERT — RESEARCH file paths are partly stale.** Two TD-4 frontend paths in 123-RESEARCH.md do not match the live tree, and WR-01/WR-02 appear already implemented. Verify state before planning fix work.

| Item | RESEARCH says | Verified live location | State |
|------|---------------|------------------------|-------|
| WR-01 (block `/app/` dir listing) | `webserver/server.go:555-578` | `internal/webserver/server.go:566-588` | **Comment at server.go:583-588 says "Phase 120 WR-01: block directory-index requests" — appears ALREADY DONE.** Confirm, don't re-touch (Chesterton's Fence). |
| WR-02 (cache-control on `/app/`) | `webserver/server.go:555-578` | `internal/webserver/server.go:556-565` | **Comment at server.go:556-565 documents the WR-02 caching policy as implemented (index.html `no-store`, hashed assets via FileServerFS). Appears ALREADY DONE.** Confirm. |
| WR-03 (joinPath name sanitize) | `FileBrowserTab.tsx:58-61` | `frontend/src/components/FileBrowserTab.tsx:399-409` | Partial WR-03 defence already at `navigateInto` (FileBrowserTab.tsx:397-409, comment cites "Phase 120 WR-03"). Confirm coverage. |
| WR-04 (mtime fallback) | `FileRow.tsx:57-65` | `frontend/src/components/FileBrowser/FileRow.tsx` (path differs — `FileBrowser/` subdir) | Verify `formatRowMtime` empty-mtime fallback. |
| WR-05 (comment clarity) | `humanSize.ts:7-22` | `frontend/src/lib/humanSize.ts` (path differs — `lib/` not `components/`) | Comment-only. Tests at `frontend/src/lib/__tests__/humanSize.test.ts`. |

WR-06 (`BreadcrumbBar.tsx`) → NO ACTION (Phase 120). WR-07 (`PreviewPane.tsx`) → already FIXED in 120-04. Do not re-touch (RESEARCH §TD-4 Inventory).

**Recommendation for planner:** Treat WR-01/WR-02 as verify-only (read the cited server.go lines, confirm the policy matches the requirement, record as already-satisfied if so). WR-03/04/05 are small React/TS hardening at the corrected paths above.

---

## Open Questions Forwarded to Planner (from RESEARCH)

1. **`daemonSettings.FilesWrite` + `schemaVersion: 4`** — RESEARCH recommends DEFER to Phase 124 (CAP-08); no FSW requirement gates the auth-less routes on a setting. If pulled forward, the analog is `internal/daemon/engine.go:99/158/182/209/508-517` + `TestSettingsMigration_FilesReadDefaultsTrue`.
2. **Streaming upload variant** (`WriteFileAtomicFrom(relPath string, r io.Reader)`) — RESEARCH A5 prefers `io.Copy` to temp over `io.ReadAll` to avoid buffering 50 MiB. Both satisfy FSW-05. Planner's call.
3. **Windows rename-over-open retry** — RESEARCH recommends a 3-attempt ~50ms bounded retry in `WriteFileAtomic`'s rename step. Low cost, closes a real Windows failure mode.

---

## Metadata

**Analog search scope:** `internal/files/` (sandbox.go, handler.go, types.go, sandbox_test.go), `internal/daemon/` (api.go, client.go, client_remote_files.go), `internal/tui/joincode_prompt.go`, `internal/webserver/server.go`, `frontend/src/components/`, `frontend/src/lib/`
**Files scanned:** 9 analog sources read in full or in targeted ranges; 2 frontend path corrections verified via find
**Pattern extraction date:** 2026-06-14
