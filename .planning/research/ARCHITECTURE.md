# Architecture Research — v3.5 Write-Side File Browser + In-App Editor

**Domain:** Write operations and in-app code editor integrating with an existing read-only sandboxed file browser
**Researched:** 2026-06-14
**Confidence:** HIGH (all claims verified against actual source files in the repository)

---

## 1. Existing Architecture Snapshot (load-bearing facts for v3.5)

### 1.1 What v3.4 actually built (verified from source)

**`internal/files/` package** — stateless `Handler` with `List`, `Stat`, `Read` methods. The `Sandbox` struct wraps `os.OpenRoot` (Go 1.24+) for kernel-level TOCTOU-safe path resolution. `validateRelativePath` applies layered defenses before `os.Root` ever sees a path. The fuzz corpus covers 40+ payloads and is a merge gate.

**Capability model** — `internal/capability/capability.go` has `Claims.Perms` as a comma-separated string. `HasPerm` is whole-token, never `strings.Contains`. Only one bit exists today: `PermFilesRead = "files.read"`. The `requireFilesRead` middleware in `internal/webserver/capability_mw.go` is a SEPARATE wrapper from `requireCapability` — the separation is load-bearing (adding `files.read` to `requireCapability` would break all relay routes that don't carry the bit).

**Daemon routes** — `GET /api/files/list`, `/api/files/stat`, `/api/files/read`, `HEAD /api/files/read` registered on the daemon socket mux in `api.go`. Auth-less by design (loopback trust boundary). Mirrored on the webserver mux behind `requireFilesRead`. Both surfaces share the same `*files.Handler` instance set via `SetFilesHandler`.

**Remote proxy** — `internal/daemon/remote_files.go` implements `/api/files/remote/{sessionID}/{list,stat,read}`. Looks up `(baseURL, capToken)` from `RemoteCapStore`, proxies outbound request, strips/forces `?session` and `?cap` params, forwards selected response headers. The desktop GUI uses this proxy (CORS-safe, cap material stays in daemon process). The TUI dials remote peers directly via `RemoteFilesClient`.

**`FilesClient` interface** — `internal/tui/files_client.go` declares:
```go
type FilesClient interface {
    ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error)
    StatFile(ctx context.Context, sessionID, relPath string) (files.FileEntry, error)
    ReadFile(ctx context.Context, sessionID, relPath string) (data []byte, mime string, err error)
    HeadFile(ctx context.Context, sessionID, relPath string) (size int64, mime string, mtime time.Time, err error)
}
```
`*daemon.DaemonClient` satisfies this via duck typing; so does `*tui.RemoteFilesClient`. Both are driven by the same `handleFilesKey` + `applyFilesListMsg` TUI pipeline.

**TD-5 concrete shape** — `DaemonClient.ExchangeJoinCodeAtURL` in `internal/daemon/client_remote_files.go` expects JSON `{cap:"..."}` from `POST /join/exchange`. The actual webserver `handleJoinExchange` responds `303 + Location: /sessions/{id}?cap=<token>`. Error paths (4xx/5xx) work; the success path fails JSON-decode on the redirect body. The TUI path (`joinCodePromptModel.exchangeJoinCodeCmd`) already correctly parses the 303 Location header — only the desktop GUI helper is broken.

**TD-4 concrete items** (WR-01..05 from Phase 120):
- WR-01: `/app/` directory listing exposed (webserver `server.go:555-578`)
- WR-02: `/app/` bundle missing cache-control headers
- WR-03: `FileBrowserTab.tsx` `joinPath` sanitization gap (names returned from server should have leading slash stripped)
- WR-04: `FileRow.tsx` `formatRowMtime` fallback behavior when mtime is empty string
- WR-05: `humanSize.ts` comment clarity

---

## 2. Write Endpoint Design

### 2.1 REST verb and operation mapping

Add write operations to the `internal/files` package. All write ops stay under the `/api/files/` prefix, following the same `?session=<id>&path=<rel>` query convention as read routes:

| Route | Verb | Operation | Notes |
|-------|------|-----------|-------|
| `/api/files/write` | `PUT` | Atomic file write (create or overwrite) | Body = file bytes; write-temp-rename pattern |
| `/api/files/upload` | `POST` | Multipart file upload | `multipart/form-data`; supports binary files |
| `/api/files/delete` | `DELETE` | Delete file or empty dir | Requires `?path=<rel>`; no recursive delete |
| `/api/files/rename` | `POST` | Rename/move within sandbox | Body = `{"from": "<rel>", "to": "<rel>"}` |
| `/api/files/mkdir` | `POST` | Create directory | Body = `{"path": "<rel>"}` or `?path=<rel>` |

**Why PUT for write, not POST?** PUT is idempotent and semantically correct for "write this exact content to this path" — consistent with how `http.ServeContent` conceptually mirrors GET. POST is correct for upload (multipart creates a new resource, not necessarily idempotent) and for operations with a body schema (rename, mkdir).

**Atomic write pattern** — `PUT /api/files/write` must:
1. Validate `relPath` via `validateRelativePath` (same gate as reads)
2. Open a temp file via `root.Create("<relPath>.tmp.<random>")` — temp must be inside the sandbox root, not `os.TempFile` to a system temp dir (would escape the sandbox)
3. Write body to temp file
4. `root.Rename(tmpName, relPath)` — atomic on POSIX; near-atomic on Windows (NTFS transactional rename)
5. On any error after temp creation: best-effort `root.Remove(tmpName)`

This pattern is the v3.5 equivalent of the v3.4 `http.ServeContent` delegation — standard Go filesystem operations, all within the `os.Root` boundary.

**Body size cap for write** — mirror the 5 MiB read cap: reject uploads > 50 MiB at the handler level (or a user-configurable cap in `daemonSettings`). Start with 50 MiB hardcoded; add a setting later if field demand exists.

### 2.2 Request/response shape consistency with read side

Read side uses JSON for metadata (`List`, `Stat`) and raw bytes for content (`Read`). Write side follows the same split:

```
PUT /api/files/write?session=<id>&path=<rel>
  Content-Type: application/octet-stream
  Body: <raw file bytes>
  Response 200: {"path": "<rel>", "size": <n>}
  Response 400: plain text (validation error)
  Response 403: plain text ("access denied: ...")
  Response 413: plain text ("file too large")

DELETE /api/files/delete?session=<id>&path=<rel>
  Response 200: {"path": "<rel>"}
  Response 400: plain text ("is a directory" if non-empty, "path required")
  Response 403: plain text ("access denied: ...")

POST /api/files/rename?session=<id>
  Content-Type: application/json
  Body: {"from": "<rel>", "to": "<rel>"}
  Response 200: {"from": "<rel>", "to": "<rel>"}
  Response 400: plain text (validation or conflict)
  Response 403: plain text ("access denied: ...")

POST /api/files/mkdir?session=<id>
  Content-Type: application/json
  Body: {"path": "<rel>"}
  Response 200: {"path": "<rel>"}
  Response 400: plain text (already exists)
  Response 403: plain text ("access denied: ...")
```

New response types for `internal/files/types.go`:
```go
type FileWriteResponse struct {
    Path string `json:"path"`
    Size int64  `json:"size"`
}

type FileOpResponse struct {
    Path string `json:"path,omitempty"`
    From string `json:"from,omitempty"`
    To   string `json:"to,omitempty"`
}
```

---

## 3. Capability Model: `files.write`

### 3.1 Adding the `PermFilesWrite` constant

In `internal/capability/capability.go`:

```go
const PermFilesWrite = "files.write"
```

**Should `files.write` imply `files.read`?** No — follow the established `HasPerm` whole-token semantics. A token with `files.write` but not `files.read` can write but cannot list or read. In practice, `issueCapabilitiesForSession` will always include both when granting write access — but the middleware does NOT infer one from the other. This avoids a foot-gun where a stripped token accidentally grants read.

### 3.2 Default token issuance

Existing token format from `issueCapabilitiesForSession` (simplified):

| Token | Current `Perms` | v3.5 `Perms` |
|-------|----------------|--------------|
| Session owner (read-write) | `"read,write,files.read"` | `"read,write,files.read,files.write"` |
| Web-share viewer (read-only) | `"read"` | `"read"` (unchanged — write is opt-in) |

The operator-controlled `daemonSettings.FilesRead` boolean already gates whether `files.read` is included in the owner token. A parallel `daemonSettings.FilesWrite` boolean (also defaulting to `true` if `FilesRead` is true) gates `files.write`. These follow `schemaVersion: 3` to `schemaVersion: 4` migration using the established defaults-merge constructor pattern.

### 3.3 `requireFilesWrite` middleware

Exact same pattern as `requireFilesRead` in `internal/webserver/capability_mw.go`:

```go
func (ws *WebServer) requireFilesWrite(next http.HandlerFunc) http.HandlerFunc {
    return ws.requireCapability(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := capability.ClaimsFromContext(r.Context())
        if !ok {
            http.Error(w, "files.write capability required", http.StatusForbidden)
            return
        }
        if !capability.HasPerm(claims.Perms, capability.PermFilesWrite) {
            http.Error(w, "files.write capability required", http.StatusForbidden)
            return
        }
        next(w, r)
    })
}
```

Webserver routes:
```
PUT    /api/files/write   → ws.requireFilesWrite(filesDispatch(h.Write))
POST   /api/files/upload  → ws.requireFilesWrite(filesDispatch(h.Upload))
DELETE /api/files/delete  → ws.requireFilesWrite(filesDispatch(h.Delete))
POST   /api/files/rename  → ws.requireFilesWrite(filesDispatch(h.Rename))
POST   /api/files/mkdir   → ws.requireFilesWrite(filesDispatch(h.Mkdir))
```

Daemon socket routes: registered without middleware (loopback trust, same as reads).

### 3.4 Web-share opt-in grant flow

The share flow today mints a viewer token with `Perms: "read"`. To grant write access to a web-share viewer:

1. `issueCapabilitiesForSession` gains a third token variant: `Perms: "read,files.read,files.write"` for an explicit write-enabled viewer grant.
2. The GUI share panel grows an opt-in toggle: "Allow file editing" (off by default). When toggled on, the session URL includes the write-enabled token instead of the read-only one.
3. The `JoinCodeManager` path (`POST /join/exchange`) currently returns the read-only viewer cap. To grant write access via a join code, the join code itself must be issued for a write-enabled grant — meaning the "Share" UI that generates the join code must offer the "Allow file editing" checkbox before generating the code.

This is security-critical: the server that generates the join code decides the Perms; the client consuming it cannot upgrade the Perms from the client side.

### 3.5 Remote write via daemon proxy + `RemoteCapStore`

The desktop GUI write path follows the exact pattern established for reads in Phase 122:

```
FileBrowserTab.tsx (write op, e.g. PUT /api/files/remote/<sid>/write)
    → daemon socket
    → proxyRemoteFiles(w, r, "write")
    → remoteCaps.Get(sessionID) → (baseURL, capToken)
    → PUT https://<remote>/api/files/write?session=<sid>&path=<rel>&cap=<capToken>
    → remote webserver's requireFilesWrite middleware
    → remote files.Handler.Write(sandbox)
```

The `proxyRemoteFiles` function in `internal/daemon/remote_files.go` dispatches by operation name. It needs to forward the request body for write operations — the current implementation uses `http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, nil)` with a nil body. This must change to `r.Body` for write methods:

```go
var body io.Reader
if r.Method == http.MethodPut || r.Method == http.MethodPost {
    body = r.Body
}
req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, body)
```

The `PUT /api/files/remote/{sessionID}/write`, `POST /api/files/remote/{sessionID}/rename`, etc. routes must be registered in `api.go`'s `registerRoutes`. The `RemoteCapStore` token must include `files.write` — which means the join code exchange must have granted that bit.

---

## 4. `FilesClient` Interface Extension

### 4.1 Write methods to add

```go
type FilesClient interface {
    // Read-side methods (existing — DO NOT CHANGE)
    ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error)
    StatFile(ctx context.Context, sessionID, relPath string) (files.FileEntry, error)
    ReadFile(ctx context.Context, sessionID, relPath string) (data []byte, mime string, err error)
    HeadFile(ctx context.Context, sessionID, relPath string) (size int64, mime string, mtime time.Time, err error)

    // Write-side methods (v3.5 additions)
    WriteFile(ctx context.Context, sessionID, relPath string, data []byte) (files.FileWriteResponse, error)
    DeleteFile(ctx context.Context, sessionID, relPath string) error
    RenameFile(ctx context.Context, sessionID, from, to string) error
    MkdirFile(ctx context.Context, sessionID, relPath string) error
}
```

**`*daemon.DaemonClient` satisfies the full interface** — add `WriteFile`, `DeleteFile`, `RenameFile`, `MkdirFile` methods to `internal/daemon/client.go` following the exact same pattern as `ListFiles`/`StatFile`/`ReadFile`/`HeadFile`.

**`*tui.RemoteFilesClient` satisfies the full interface** — add the same four methods, constructing HTTPS requests with the `?cap=<token>` bearer, same as the read methods.

### 4.2 Interface parity proof pattern

v3.4 proved read parity by 3 independent observers (daemon-proxy Go, `RemoteFilesClient` Go, Playwright HTTPS browser). v3.5 must prove write parity the same way:

- Local write: daemon socket → `files.Handler.Write` (sandbox)
- Remote write via GUI: daemon proxy → remote webserver → remote `files.Handler.Write`
- Remote write via TUI: `RemoteFilesClient.WriteFile` → remote webserver → remote `files.Handler.Write`

Integration tests for write parity: write a file via `DaemonClient.WriteFile`, verify it via `DaemonClient.ReadFile`; repeat via `RemoteFilesClient` against an `httptest.TLSServer`; add one Playwright cell for a browser `PUT /api/files/write` call via the web-share surface.

---

## 5. In-App Editor Integration

### 5.1 React `FileBrowserTab` — read-only to editable

**Where the editor mounts:** The existing `PreviewPane` in `FileBrowserTab.tsx` renders text files in a read-only view (`<pre>` for plain text, `react-markdown` for `.md`). In v3.5, for text files when `files.write` capability is present, the preview pane switches to an editor component.

Pattern: add a `mode` prop (`'preview' | 'edit'`) to `PreviewPane`. The "Edit" action appears in the file row context menu and as a button in the preview pane header. Clicking edit:
1. Loads the file content (already in preview state — reuse)
2. Replaces the read-only renderer with the editor component
3. Editor becomes the content of `PreviewPane` with the same pane dimensions

**Editor component structure:**
```typescript
// frontend/src/components/Editor.tsx
interface EditorProps {
  sessionId: string
  relPath: string
  initialContent: string
  mime: string
  onSave: (content: string) => Promise<void>
  onDiscard: () => void
  readOnly?: boolean
}
```

**Save flow:**
1. User edits content in the editor
2. "Save" button (or Cmd+S) calls `onSave(editorContent)`
3. `onSave` calls `PUT /api/files/write?session=<id>&path=<rel>` via `fetch()` to the daemon port (same pattern as all other file calls)
4. On 200: refresh the `FileBrowserTab` listing (re-fetch the current directory to update mtime/size)
5. On error: surface inline error in editor status bar (no full-pane takeover — user needs to see the content to copy it out)

**Capability guard:** The editor's edit toggle only appears when `useFilesCapability` includes `files.write`. The `useFilesCapability` hook (already exists for `files.read`) needs a `canWrite` return value:
```typescript
const { canRead, canWrite, permissionDenied } = useFilesCapability(sessionId)
```

### 5.2 CodeMirror 6 vs Monaco (decision deferred to plan time per PROJECT.md)

Both are viable. Architectural difference: CodeMirror 6 is a collection of composable packages that can be tree-shaken; Monaco is a single large bundle. In a Wails context where assets are embedded in the binary:

- CodeMirror 6 core + language packs + themes: ~200-400 KB minified+gzipped (varies by languages)
- Monaco: ~2-4 MB minified+gzipped (full IDE feature set)

For v3.5's scope (edit + save with syntax highlighting), CodeMirror 6 is a better fit: smaller binary, composable language packs per file extension, aligns with the vendoring discipline (`vendor_drift_test.go` gate already enforces same-version lockstep for all vendored assets).

**If CodeMirror 6:** install `@codemirror/view`, `@codemirror/state`, `@codemirror/commands`, `@codemirror/language`, plus per-language packages (e.g., `@codemirror/lang-javascript`, `@codemirror/lang-python`, `@codemirror/lang-go`). Vendor all of them under `web/vendor/codemirror/` or include in the Vite bundle (Vite handles tree-shaking; no CDN).

**If Monaco:** `monaco-editor` package is large but has out-of-the-box language support. Requires a web worker for the language service — needs CSP carve-out (`worker-src 'self' blob:`) since Monaco spawns workers from blob URLs. This is a non-trivial CSP change requiring a dedicated security audit step.

**Recommendation for roadmapper:** CodeMirror 6 is the lower-risk choice for the existing CSP model. Monaco requires a dedicated security audit sub-phase before committing; treat it as a risk item in the editor phase if selected.

### 5.3 TUI `$EDITOR` shell-out

The TUI has no embedded renderer — it uses `tea.Exec` (Bubble Tea v2's subprocess pattern) to hand control to an external editor, the same way `attach` hands control to the PTY.

**Flow:**
1. User presses `e` on a file in the Files view (`tabFiles`)
2. `fetchPreviewCmd` reads the file content via `client.ReadFile` (already available from preview)
3. Content is written to a temp file using system `os.CreateTemp` — the temp file is outside the sandbox and is local-machine-only (editor subprocess runs locally regardless of session remoteness)
4. TUI calls `tea.Exec(editor subprocess, onComplete)` — TUI suspends, editor has full terminal
5. On editor exit, `onComplete` fires as a `tea.Msg`
6. The `onComplete` handler reads the temp file and returns a `writeBackCmd` tea command calling `client.WriteFile(ctx, sessionID, relPath, content)`
7. On `WriteFile` success: refresh the directory listing; show success status
8. Temp file is removed regardless of success/fail

**Key constraint — all filesystem I/O via `tea.Cmd`:** The static-grep gate `TestFiles_NoSyncFSCalls` enforces no synchronous `os.*` calls in TUI update code. The write-back (`client.WriteFile`) must be a `tea.Cmd` returning a `tea.Msg`. Writing the temp file for the editor is a one-time setup step that happens in the `tea.Exec` subprocess wrapper (not in the TUI update loop), so it does not trigger the gate — but `DaemonClient.WriteFile` after editor exit must be a `tea.Cmd`, not a direct call in `Update`.

**`$EDITOR` resolution order:**
1. `$VISUAL` env var
2. `$EDITOR` env var
3. Platform fallback: `vi` (POSIX), `notepad` (Windows)
4. If no editor found: show error toast, do not attempt exec

**Remote sessions via TUI:** When `fm.client` is a `*RemoteFilesClient`, the same shell-out path works — `ReadFile` fetches the bytes over HTTPS, writes to local temp file, spawns editor locally, then `WriteFile` sends the edited bytes back over HTTPS. The temp file is always on the local machine regardless of session locality.

---

## 6. System Overview with Write Layer

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                              agenthub binary                                   │
│                                                                                 │
│  ┌───────────────┐  ┌──────────┐  ┌────────────────────────────────────────┐  │
│  │  Wails (GUI)  │  │   TUI    │  │  Daemon (background)                   │  │
│  │               │  │ BubbleTea│  │                                        │  │
│  │FileBrowserTab │  │FilesModel│  │  internal/files/Handler                │  │
│  │  + Editor     │  │  + $EDIT │  │    List/Stat/Read/HEAD  (v3.4)         │  │
│  └──────┬────────┘  └────┬─────┘  │    Write/Delete/Rename/Mkdir (v3.5)   │  │
│         │                │        │                                        │  │
│         │ Unix socket    │ Unix   │  RemoteCapStore                        │  │
│         │ HTTP/JSON      │ socket │  (sessionID → baseURL + capToken)      │  │
│         ▼                ▼        │                                        │  │
│  ┌────────────────────────────┐   │  WebServer (Tailscale HTTPS)           │  │
│  │      DaemonClient          │   │    requireFilesRead  (v3.4)            │  │
│  │  List/Stat/Read/Head       │──▶│    requireFilesWrite (v3.5)            │  │
│  │  Write/Delete/Rename/Mkdir │   │    /api/files/remote/* proxy           │  │
│  └────────────────────────────┘   │                                        │  │
│           TUI also uses:           └───────────────────┬────────────────────┘  │
│  ┌────────────────────────────┐                        │                       │
│  │    RemoteFilesClient       │                        │ HTTPS/Tailscale       │
│  │  (direct HTTPS + cap)      │──────────────────────▶ Remote Tailnet Peer    │
│  │  List/Stat/Read/Head       │                       /api/files/* (r+w)       │
│  │  Write/Delete/Rename/Mkdir │                                                │
│  └────────────────────────────┘                                                │
└───────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Data Flow Diagrams

### 7.1 Local write — GUI editor save

```
User edits file in Editor component
    │
    ▼
"Save" (Cmd+S) → onSave(content)
    │
    ▼
fetch(`http://127.0.0.1:${relayPort}/api/files/write?session=<id>&path=<rel>`, {
  method: 'PUT', body: content, headers: {'Content-Type': 'application/octet-stream'}
})
    │ daemon socket — no auth (loopback trust boundary)
    ▼
internal/daemon/api.go  PUT /api/files/write
    │  engine.GetSessionWorkDir(id) → "/path/to/workdir"
    │  files.NewSandbox("/path/to/workdir")
    ▼
internal/files/Handler.Write(sandbox)
    │  validateRelativePath(relPath)
    │  root.Create("<rel>.tmp.<rand>") inside sandbox
    │  io.Copy(tmpFile, r.Body) with 50 MiB cap
    │  root.Rename(tmpName, relPath)  — atomic
    ▼
200 {"path": "<rel>", "size": <n>}
    │
    ▼
FileBrowserTab refreshes directory listing (re-fetch /list)
```

### 7.2 Remote write — GUI proxy path

```
User edits file in FileBrowserTab for a remote tailnet session
    │
    ▼
PUT /api/files/remote/<sid>/write?path=<rel>
  (fetch to local daemon port — same as local, different URL prefix)
    │
    ▼
daemon's proxyRemoteFiles(w, r, "write")
    │  remoteCaps.Get(sid) → (baseURL="https://remotebox.ts.net:9443", capToken)
    │  Forwards r.Body to upstream request (new in v3.5)
    │  Builds: PUT https://remotebox.ts.net:9443/api/files/write?session=<sid>&path=<rel>&cap=<token>
    ▼
remote daemon's webserver — requireFilesWrite
    │  HasPerm(claims.Perms, "files.write") — true if write-enabled token
    ▼
remote internal/files/Handler.Write(sandbox at remote workdir)
    │
    ▼
200 {"path": "<rel>", "size": <n>} forwarded back through proxy
    │
    ▼
FileBrowserTab refreshes listing
```

### 7.3 Remote write — TUI `$EDITOR` shell-out with `RemoteFilesClient`

```
User presses 'e' on file in TUI Files view (remote session)
    │
    ▼
fetchPreviewCmd (tea.Cmd) → RemoteFilesClient.ReadFile(ctx, sid, path)
    │  HTTPS to remote /api/files/read?session=<sid>&path=<rel>&cap=<token>
    ▼
bytes written to local /tmp/agenthub-edit-<rand>.<ext>
    │
    ▼
tea.Exec(exec.Command(editorBin, tmpPath), onEditorExit)
    │  TUI suspended, $EDITOR has full terminal
    ▼
editor exits → onEditorExit tea.Msg fires
    │
    ▼
writeBackCmd (tea.Cmd) → RemoteFilesClient.WriteFile(ctx, sid, path, content)
    │  PUT https://remote/api/files/write?session=<sid>&path=<rel>&cap=<token>
    ▼
200 → filesModel receives writeBackSuccessMsg → refreshes listing
temp file removed
```

---

## 8. New vs Modified Components

### 8.1 New files

| File | Purpose |
|------|---------|
| `internal/files/write.go` | `Handler.Write`, `Handler.Delete`, `Handler.Rename`, `Handler.Mkdir` HTTP handlers; atomic write-temp-rename pattern; 50 MiB body cap; sandbox-internal temp file creation |
| `internal/files/write_test.go` | Round-trip tests for all four write ops; atomic-write verification; concurrent-write safety; fuzz extension for write paths |
| `frontend/src/components/Editor.tsx` | CodeMirror 6 (or Monaco) editor wrapper; `EditorProps` interface; Cmd+S save; discard confirmation; syntax detection by MIME/extension |
| `frontend/src/components/__tests__/Editor.test.tsx` | Source-inspection tests (same `?raw` pattern as `TerminalPanel.test.tsx`) |
| `frontend/src/hooks/useFilesWrite.ts` | `useFilesWrite(sessionId, relPath)` hook: `{ save, delete: rm, rename, mkdir, isSaving, saveError }`; wraps fetch calls to write endpoints |

### 8.2 Modified files

| File | Change |
|------|--------|
| `internal/files/handler.go` | No change to existing methods — write handlers land in `write.go` (same package) |
| `internal/files/types.go` | Add `FileWriteResponse`, `FileOpResponse` wire types |
| `internal/capability/capability.go` | Add `const PermFilesWrite = "files.write"` |
| `internal/webserver/capability_mw.go` | Add `requireFilesWrite` wrapper (mirrors `requireFilesRead` exactly) |
| `internal/webserver/server.go` | Register five write routes under `requireFilesWrite` in `setupRoutes()` |
| `internal/daemon/api.go` | Register write routes on daemon socket mux (auth-less); register remote-write proxy routes; update `issueCapabilitiesForSession` to include `files.write` in owner token |
| `internal/daemon/remote_files.go` | `proxyRemoteFiles` must forward `r.Body` for PUT/POST methods; register write op names in the proxy dispatch |
| `internal/daemon/client.go` | Add `WriteFile`, `DeleteFile`, `RenameFile`, `MkdirFile` methods to `DaemonClient` |
| `internal/tui/files_client.go` | Add `WriteFile`, `DeleteFile`, `RenameFile`, `MkdirFile` to `FilesClient` interface |
| `internal/tui/remote_files_client.go` | Implement write methods on `RemoteFilesClient` |
| `internal/tui/files.go` | Add `editMode` state; `$EDITOR` shell-out via `tea.Exec`; write-back `tea.Msg` dispatch |
| `internal/tui/files_cmds.go` | Add `writeBackCmd`, `deleteCmd`, `renameCmd` tea commands |
| `frontend/src/components/FileBrowserTab.tsx` | Add edit/delete/rename/mkdir triggers; switch preview pane to `<Editor>` in edit mode; connect `useFilesWrite` hook; `canWrite` from updated `useFilesCapability` |
| `frontend/src/hooks/useFilesCapability.ts` | Add `canWrite: boolean` return value (checks `files.write` in cap token Perms) |
| `internal/daemon/client_remote_files.go` | Fix TD-5: `ExchangeJoinCodeAtURL` success path must parse 303 Location header for `?cap=<token>` instead of attempting JSON decode on redirect body |
| `internal/daemon/engine.go` | `daemonSettings.FilesWrite bool` field + `filesWriteEnabled()` method; `schemaVersion: 4` migration |
| `internal/daemon/types.go` | `daemonSettings` struct gains `FilesWrite bool` |

### 8.3 Files that do NOT change

- `internal/relay/` — no PTY protocol involvement; file writes are independent REST calls
- `internal/files/sandbox.go` — the sandbox boundary logic is read/write agnostic; `os.OpenRoot` handles writes too (same kernel-level containment)
- `App.go` — no new Wails bindings needed; file writes go through `DaemonClient` HTTP, same as reads
- `wailsjs/go/main/App.{d.ts,js}` — no new stubs needed
- `internal/tailnet/` — unchanged

---

## 9. TD-4 and TD-5 Placement in Build Order

**TD-5 (`ExchangeJoinCodeAtURL` JSON-vs-303 mismatch) must be fixed in the first write phase**, before any remote write testing. The broken join-code exchange means the desktop GUI cannot acquire a cap for a remote session at all today — it fails silently on the success path. Remote write testing is impossible without a working cap exchange.

Concrete fix: in `internal/daemon/client_remote_files.go`, change the success path in `ExchangeJoinCodeAtURL` to:
1. Set `http.Client.CheckRedirect` to return `http.ErrUseLastResponse` (disables auto-follow)
2. Detect `resp.StatusCode == 303`
3. Parse `Location` header URL
4. Extract `?cap=<token>` query parameter from the Location URL
5. Return the token

The TUI's `exchangeJoinCodeCmd` in `joincode_prompt.go` already does this correctly — use it as the reference implementation for the desktop fix.

**TD-4 (WR-01..05 file-browser hardening) can be bundled into the first phase** as a cleanup sweep. None of the 5 items block any v3.5 feature work — they are hardening fixes on shipped code. Folding them into the sandbox write primitive phase keeps the "while we're touching the file layer" cleanup tight.

---

## 10. Suggested Build Order (Phase 123+)

Phase dependency rationale mirrors the v3.4 build order but adds write-specific concerns.

### Phase 123: TD Cleanup + Write Sandbox Primitives + `PermFilesWrite` + Daemon Routes

**Dependencies:** None (extends existing `internal/files/` and `internal/daemon/` packages)
**Contents:**
- TD-4 fixes: WR-01..05
- TD-5 fix: `ExchangeJoinCodeAtURL` JSON-vs-303 in `client_remote_files.go`
- `internal/files/write.go`: `Handler.Write`, `Handler.Delete`, `Handler.Rename`, `Handler.Mkdir`
- `internal/files/write_test.go`: round-trip + fuzz extension
- New `FileWriteResponse`, `FileOpResponse` types in `types.go`
- `PermFilesWrite` constant in `capability.go`
- Daemon socket write routes in `api.go` (auth-less)
- `DaemonClient.WriteFile`, `DeleteFile`, `RenameFile`, `MkdirFile` in `client.go`
- `daemonSettings.FilesWrite` field + `schemaVersion: 4` migration

**Gate:** `go test ./internal/files/... ./internal/daemon/...` green; fuzz-write reports zero crashes; TD-5 fixed and tested

### Phase 124: `files.write` Capability + Webserver Write Routes

**Dependencies:** Phase 123 (sandbox write primitives frozen, `PermFilesWrite` constant available)
**Contents:**
- `requireFilesWrite` middleware in `capability_mw.go`
- Webserver write routes in `server.go` (five routes under `requireFilesWrite`)
- `issueCapabilitiesForSession` updated: owner token includes `files.write`
- `daemonSettings.FilesWrite` gates `files.write` inclusion (mirrors `FilesRead` pattern)
- Share UI opt-in for write capability (toggle in share panel, defaults off)
- Integration tests: viewer without `files.write` → 403 on write endpoints; owner token → 200
- Remote proxy write routes in `api.go` + body forwarding in `remote_files.go`

**Gate:** 403/200 integration tests; `TestHasPerm_NoStringsContains` gate extended for `files.write`; zero new CSP violations

### Phase 125: React Editor (Desktop + Web)

**Dependencies:** Phases 123 + 124 (write API frozen, capability model live)
**Contents:**
- Editor library decision finalized (CodeMirror 6 recommended) + install + vendor gate update
- `frontend/src/components/Editor.tsx` with syntax highlighting
- `frontend/src/hooks/useFilesWrite.ts`
- `useFilesCapability` extended with `canWrite`
- `FileBrowserTab.tsx` edit/delete/rename/mkdir triggers
- `PreviewPane` switches to editor in edit mode
- Playwright cross-browser e2e: local write, web-share write with `files.write` cap, 403 without `files.write`

**Gate:** Cross-browser Playwright e2e; `vendor_drift_test.go` updated for editor vendor bundle if vendored

### Phase 126: TUI `$EDITOR` Shell-Out

**Dependencies:** Phase 123 (`FilesClient` interface extended, `DaemonClient` write methods available)
**Contents:**
- `FilesClient` interface write methods added in `files_client.go`
- `RemoteFilesClient` write methods in `remote_files_client.go`
- `files.go` edit mode + `tea.Exec` shell-out + `writeBackCmd` wiring
- `files_cmds.go` write/delete/rename commands
- `TestFiles_NoSyncFSCalls` gate extended to cover write-path files
- Integration test: `$EDITOR` override via env var; write via `RemoteFilesClient` against `httptest.TLSServer`

**Gate:** All TUI tests green; `TestFiles_NoSyncFSCalls` passes; `$EDITOR` shell-out test passes

### Phase 127: Web-Share Write Security Hardening

**Dependencies:** Phases 124 + 125 (write routes live, capability model complete)
**Contents:**
- Write security audit: body size cap enforcement, atomic rename failure paths, concurrent write race tests
- Capability escalation audit: verify no path allows a read-only token to reach write endpoints
- Web-share write capability opt-in UX finalization
- Playwright e2e for web-share write scenarios (guest with `files.write` cap)

**Gate:** Security audit documented; Playwright web-share write e2e

### Phase 128: Remote Write Parity + Cross-Surface Integration

**Dependencies:** Phases 123-127 (full stack live locally)
**Contents:**
- Remote write parity proof: 3-observer pattern (daemon-proxy Go, `RemoteFilesClient` Go, Playwright HTTPS browser) — mirrors Phase 122's read parity proof
- TUI remote write via `RemoteFilesClient` end-to-end test
- GUI remote write via daemon proxy end-to-end test
- Cross-surface parity: same file content via all three paths produces byte-identical results at the remote sandbox

**Gate:** 3-observer write parity test passes; no regression on Phase 122 remote read tests; write parity ready for two-machine UAT

---

## 11. Integration Points Summary

| Integration Point | v3.4 State | v3.5 Change |
|-------------------|-----------|-------------|
| `internal/files/sandbox.go` | Read-only `Open()`/`Stat()` via `os.OpenRoot` | No change — `os.OpenRoot` handles writes natively; write ops in `write.go` |
| `internal/files/handler.go` | `List`/`Stat`/`Read` | No change — write handlers in sibling `write.go` |
| `internal/capability/capability.go` | `PermFilesRead` + `HasPerm` | Add `PermFilesWrite` constant only |
| `internal/webserver/capability_mw.go` | `requireFilesRead` | Add parallel `requireFilesWrite` |
| `internal/webserver/server.go` | 4 read routes | Add 5 write routes under `requireFilesWrite` |
| `internal/daemon/api.go` | 4 read + 4 remote proxy read routes | Add 5 write + 5 remote proxy write routes |
| `internal/daemon/remote_files.go` | `proxyRemoteFiles` handles GET/HEAD (nil body) | Extend to forward `r.Body` for PUT/POST |
| `internal/daemon/client.go` | 4 read methods | Add 4 write methods |
| `internal/daemon/client_remote_files.go` | Broken `ExchangeJoinCodeAtURL` (JSON-vs-303) | Fix: parse 303 Location header for cap token (TD-5) |
| `internal/tui/files_client.go` | 4-method `FilesClient` interface | Extend to 8 methods |
| `internal/tui/remote_files_client.go` | 4 read methods | Add 4 write methods |
| `internal/tui/files.go` | Browse + preview | Add edit mode + `tea.Exec` + write-back Cmd |
| `frontend/src/components/FileBrowserTab.tsx` | Browse + preview | Add editor, write action triggers |

---

## 12. Anti-Patterns to Avoid

### Anti-Pattern 1: Writing temp files outside the sandbox

**What people might do:** `os.CreateTemp("", "edit-*.go")` to create a temp file, then `os.Rename` into the sandbox.

**Why wrong:** `os.Rename` across filesystem boundaries is not atomic on all platforms. More critically, the temp file is in the system temp dir — a race window for symlink attacks (`/tmp/agenthub-edit-xyz` could be a symlink placed before `os.Rename` fires). The `os.Root` sandbox boundary must contain the entire write path.

**Do instead:** `root.Create("<rel>.tmp.<rand>")` where `root` is the `os.Root` handle. The temp file is inside the sandbox. `root.Rename(tmp, rel)` is atomic within the same directory tree.

### Anti-Pattern 2: `files.write` implying `files.read` in middleware

**What people might do:** `requireFilesWrite` internally calls `requireFilesRead` so a write-capable token automatically gets read access.

**Why wrong:** It couples two independent capabilities and breaks the whole-token semantics that `HasPerm` is designed to enforce. Middleware composing capabilities silently creates an implicit dependency invisible at issuance time.

**Do instead:** Middleware is orthogonal. Issue tokens that include both when both are needed. Check each bit independently at each endpoint.

### Anti-Pattern 3: Adding `files.write` check to `requireCapability` switch

**What people might do:** Extend the existing `requireCapability` middleware to also enforce `files.write` on write routes.

**Why wrong:** `requireCapability` is used by relay routes and plugin config routes that must never be affected by file capability bits. The v3.4 decision to use a SEPARATE `requireFilesRead` wrapper was explicitly load-bearing. The same separation is required for `requireFilesWrite`.

**Do instead:** New `requireFilesWrite` function, parallel structure to `requireFilesRead`, not touching `requireCapability`.

### Anti-Pattern 4: Monaco without CSP audit phase

**What people might do:** Add Monaco as the editor without addressing its `blob:` URL web worker requirement.

**Why wrong:** Monaco spawns its language service worker via `new Worker(blob:...)`. The current CSP has `worker-src 'self'` (implicit from `default-src 'self'`). A `blob:` worker URL requires `worker-src blob:` — a non-trivial CSP expansion requiring Playwright cross-browser verification.

**Do instead:** If Monaco is chosen, add a CSP audit sub-task to Phase 125. CodeMirror 6 has no worker requirement and fits the existing CSP without changes.

### Anti-Pattern 5: Synchronous write in TUI update loop

**What people might do:** Call `client.WriteFile` directly in the TUI `Update` function after the editor subprocess exits.

**Why wrong:** The `TestFiles_NoSyncFSCalls` static-grep gate enforces that all filesystem I/O in TUI update code is via `tea.Cmd`. A synchronous call in `Update` blocks the Bubble Tea render loop. The gate was built specifically for the Files view and covers `files_cmds.go` and `files.go`.

**Do instead:** Return a `writeBackCmd(client, sid, path, content)` tea command from `Update` and handle the result via `filesWriteSuccessMsg` / `filesWriteErrorMsg`.

---

## 13. Sources

- `internal/files/handler.go` — verified `Handler` struct, `Read`/`List`/`Stat` method signatures; `sandboxResolver` injection pattern
- `internal/files/sandbox.go` — verified `os.OpenRoot` usage, `validateRelativePath` layered defenses, `validateAndClean` pattern
- `internal/files/types.go` — verified `FileEntry`, `FileListResponse` wire shapes
- `internal/capability/capability.go` — verified `PermFilesRead`, `HasPerm` whole-token semantics, `Claims.Perms` format
- `internal/webserver/capability_mw.go` — verified `requireFilesRead` separation from `requireCapability`; confirmed pattern for `requireFilesWrite`
- `internal/webserver/server.go` — verified `SetFilesHandler`, write route registration model, `setupRoutes` mount points
- `internal/daemon/api.go` — verified read routes, remote proxy routes, `issueCapabilitiesForSession`, `SetFilesHandler` call sites
- `internal/daemon/remote_files.go` — verified `proxyRemoteFiles` uses nil body for GET/HEAD; confirmed body forwarding gap for write ops
- `internal/daemon/client.go` — verified `ListFiles`, `StatFile`, `ReadFile`, `HeadFile` signatures and HTTP client patterns
- `internal/daemon/client_remote_files.go` — verified TD-5 bug: JSON decode on 303 redirect body at lines 107-117; confirmed 4xx/5xx paths work
- `internal/tui/files_client.go` — verified `FilesClient` interface (4 methods); confirmed duck-typing relationship with `DaemonClient`
- `internal/tui/remote_files_client.go` — verified `RemoteFilesClient` implements `FilesClient` via duck typing
- `.planning/milestones/v3.4-phases/122-remote-session-file-browse-wiring/122-01-RECOVERY-SUMMARY.md` — TD-5 concrete shape: "expects JSON `{cap:"..."}`, actual webserver returns 303 + Location header"
- `.planning/milestones/v3.4-phases/122-remote-session-file-browse-wiring/122-04-PLAN.md` — TUI join-code exchange parses 303 Location header correctly (reference implementation for TD-5 fix)
- `.planning/milestones/v3.4-phases/120-filebrowsertab-tsx-desktop-web/120-VERIFICATION.md` — TD-4 items WR-01..05 concrete locations
- `.planning/milestones/v3.4-ROADMAP.md` — v3.4 phase structure; deferred items TD-4, TD-5
- `.planning/PROJECT.md` — v3.5 milestone scope decisions (2026-06-14); Key Decisions table

---
*Architecture research for: v3.5 write-side file browser + in-app code editor*
*Researched: 2026-06-14*
