# Architecture Research — v3.4 File Browser (Read-Only)

**Domain:** Sandboxed read-only filesystem service integrated with daemon-centric capability-gated architecture
**Researched:** 2026-05-20
**Confidence:** HIGH (all claims verified against actual source files)

---

## 1. Existing Architecture Snapshot (load-bearing facts)

### 1.1 Daemon API layer (`internal/daemon/api.go`)

The daemon owns a single `http.ServeMux` bound to a Unix socket (named pipe on Windows). Routes are registered in `registerRoutes()`. All settings RPCs follow `GET /settings/<name>` + `PATCH /settings/<name>`. Session-lifecycle RPCs follow `POST /sessions`, `GET /sessions/{id}`, `DELETE /sessions/{id}`. No existing route namespace exists for files. The API is consumed exclusively over the loopback socket by Wails bindings, the CLI, and the TUI — it is NOT the web-facing server.

### 1.2 WebServer layer (`internal/webserver/server.go`)

A separate `WebServer` instance bound to the Tailscale IP (or LAN IP in local mode). Routes are registered in `setupRoutes()`. Every session-bound endpoint is wrapped in `ws.requireCapability(...)` which reads `?cap=<token>` from the query string, verifies the HMAC-SHA256 token, checks the grant list, and attaches `capability.Claims` to the request context. The capability middleware pattern is implemented in `internal/webserver/capability_mw.go`.

### 1.3 Capability token structure (`internal/capability/capability.go`)

```go
type Claims struct {
    SID     string `json:"sid"`
    Perms   string `json:"perms"`    // "read" or "read,write"
    IAT     int64  `json:"iat"`
    GrantID string `json:"grant_id"`
    V       int    `json:"v"`
}
```

Tokens are issued in `internal/daemon/api.go:issueCapabilitiesForSession()`. Two tokens per session: `Perms: "read"` and `Perms: "read,write"`. The `Perms` field is currently a free-form comma-separated string. The `requireCapability` middleware exposes claims to the handler via `capability.WithClaims` / `capability.FromContext`.

### 1.4 Relay frame protocol (`internal/relay/protocol.go`)

```
0x01 MsgOutput  — PTY stdout → client
0x02 MsgResize  — resize from server
0x10 MsgInput   — client keyboard → PTY stdin
0x11 MsgResize2 — alt resize (client → server)
0x12 MsgPing    — keepalive
0x20 MsgMeta    — JSON metadata push (viewerCount, etc.)
```

Range `0x20–0x2F` is documented as "reserved for future server-push frame types". This is the only extensibility hook in the relay protocol. The relay is a PTY fan-out protocol — it has no request-response semantics. Adding file ops as relay frames would be architecturally wrong (they are not PTY output).

### 1.5 WorkDir is NOT stored after session creation

`CreateSession` passes `workDir` to `pty.NativePTYBackend.Create()` which sets `cmd.Dir`. The `Session` struct (in `internal/pty/session.go`) does NOT have a `WorkDir` field. The `SessionEngine` does NOT track a `sessionWorkDirs` map. After a session is spawned, there is no way to recover its launch directory through existing engine methods.

**Consequence:** `SessionEngine` needs a new `sessionWorkDirs map[string]string` field populated at `CreateSession` time, so the file service can look up the sandbox root for a given session ID.

### 1.6 Tab management patterns (frontend + TUI)

**Frontend (App.tsx):** Singleton tabs (Welcome, Sessions, Remote, Settings) follow the find-or-add pattern: `tabs.find(t => t.type === '<type>')` — if found, focus; otherwise append + focus. Tab descriptors carry a `type` string field. Session tabs carry `sessionId`; non-session tabs use `sessionId: ''`. The file browser needs the `type: 'file-browser'` and carries a `sessionId` so it is per-session, not singleton — one per active session (same as terminal tabs). The `__file_browser__<sessionId>` ID pattern avoids collision with session UUIDs.

**TUI (internal/tui/model.go):** Tabs are `tabID iota` constants. The `sidebarTabs` array in `update.go` drives sidebar→tab navigation. The `openTab(id)` method implements find-or-activate. A new `tabFiles tabID` constant fits this pattern exactly. Because the TUI does not have per-session sub-tabs (it has a unified session list), `tabFiles` is a content tab that shows a file browser for the currently selected session in the sidebar. This is simpler than per-session tab instances.

---

## 2. Architecture Decision Record

### Decision 1: Where does the FS service live?

**Recommendation: `internal/files/` — a new sub-package.**

Rationale:
- The sandbox logic (`filepath.Clean → symlink-resolve → prefix-check`) is pure, testable without daemon or webserver dependencies. It needs its own test surface.
- The HTTP handlers are stateless given a sandbox root — they do not need to be methods on `SessionEngine` or `WebServer`.
- The webserver mounts the handlers via the same `SetXxxProvider` injection pattern already used for `SetPluginSettingsProvider` and `SetSessionResolver`. This keeps `internal/files` importable by both `internal/webserver` (for web-facing REST) and `internal/daemon` (for daemon-socket REST) without circular imports.
- Module boundary: `internal/files` imports nothing from `internal/daemon`, `internal/relay`, or `internal/webserver`. It takes a root path string and an `http.ResponseWriter`/`http.Request` pair. Zero coupling.

```
internal/files/
├── sandbox.go         — Sandbox struct, Resolve(root, relPath) → absPath, error
├── handler.go         — HTTP handlers: List, Stat, Read (uses http.ServeContent)
├── sandbox_test.go    — path traversal fuzz + unit tests (required before merge)
└── handler_test.go    — round-trip HTTP tests
```

### Decision 2: REST endpoint shape and which mux owns it

**Both muxes expose file endpoints, with different authentication:**

| Endpoint | Daemon socket mux | WebServer mux |
|----------|------------------|---------------|
| `GET /sessions/{id}/files/list?path=<rel>` | No auth (loopback only) | `requireCapability` + `files.read` perm check |
| `GET /sessions/{id}/files/stat?path=<rel>` | No auth (loopback only) | `requireCapability` + `files.read` perm check |
| `GET /sessions/{id}/files/read?path=<rel>` | No auth (loopback only, Range-capable) | `requireCapability` + `files.read` perm check |

**Rationale:**

The daemon socket API (`internal/daemon/api.go`) is the local transport for the GUI, CLI, and TUI. Adding file routes here gives the Wails GUI a clean path to serve file data without going through the web port (which may not be running). The Wails GUI calls `DaemonClient` methods, same as all other GUI-originated session operations.

The WebServer mux (`internal/webserver/server.go`) gates these same routes behind `requireCapability`. The `files.read` check (see Decision 4 below) lives here.

A single `FileHandler` from `internal/files` is constructed per session by the daemon and injected into both muxes. The daemon registers routes on both muxes; the WebServer mount goes through `SetFilesHandlerProvider` (same injection pattern as `SetPluginSettingsProvider`).

### Decision 3: Local vs remote session symmetry

**Use HTTPS over Tailscale for remote session file access — NOT a new relay frame type.**

Full call-path trace:

**Local session, GUI client:**
```
FileBrowserTab.tsx
  → fetch("http://127.0.0.1:<relay-port>/sessions/<id>/files/list?path=...")
    (daemon socket exposed via DaemonClient HTTP, no auth needed)
  → internal/daemon/api.go handler
    → internal/files.Handler.List(sandbox root from sessionWorkDirs[id])
    → JSON response
```

**Local session, web-share viewer:**
```
web terminal.html / web FileBrowser page
  → fetch("https://<tailscale-fqdn>:<port>/api/files/list?session=<id>&path=...&cap=<token>")
  → internal/webserver/server.go  (requireCapability checks files.read perm)
    → internal/files.Handler.List(sandbox root from sessionResolver)
    → JSON response
```

**Remote (tailnet) session, local GUI client:**
```
FileBrowserTab.tsx (knows remote session URL base)
  → fetch("https://<remote-fqdn>:<port>/api/files/list?session=<id>&path=...&cap=<token>")
    (HTTPS over Tailscale — same channel the GUI already uses for tailnet peer discovery)
  → remote daemon's WebServer mux
    → remote internal/files.Handler.List
    → JSON response
```

**Why NOT relay frame type for files:**

The relay is a binary fan-out protocol for PTY I/O — no request-response semantics. Implementing file ops over relay would require adding a request-correlation layer (request ID, response framing, timeout handling) that duplicates what HTTP already provides. HTTP gives Range, Content-Type, 404/403/200, caching headers, streaming chunked responses — all for free. The existing precedent for remote-session metadata is HTTPS (tailnet peer discovery is HTTPS, remote attach goes via WSS relay only for the PTY stream). File ops are metadata, not PTY stream, so they follow the HTTPS path.

The `FileBrowserTab` already has access to the remote session's web URL (same `url` field used in `RemoteSession.URL` for tailnet peers). The tab uses that base URL for its API calls.

### Decision 4: Capability bit plumbing

**Extend `Claims.Perms` string with `files.read` as a new token in the comma-separated list.**

Current format: `"read"` or `"read,write"`.
New format: `"read"` / `"read,write"` / `"read,files.read"` / `"read,write,files.read"`.

**Default-ON for session owner, default-OFF for web-share viewer:**

In `issueCapabilitiesForSession()` in `internal/daemon/api.go`, the read-write token gets `Perms: "read,write,files.read"` (session owner). The read-only token gets `Perms: "read"` (web-share viewer does NOT get `files.read` by default).

The webserver's `requireCapability` middleware checks the `files.read` bit in a new `requireFilesRead` wrapper:

```go
// internal/webserver/capability_mw.go addition
func (ws *WebServer) requireFilesRead(next http.HandlerFunc) http.HandlerFunc {
    return ws.requireCapability(func(w http.ResponseWriter, r *http.Request) {
        claims := capability.FromContext(r.Context())
        if !strings.Contains(claims.Perms, "files.read") {
            http.Error(w, "files.read capability required", http.StatusForbidden)
            return
        }
        next(w, r)
    })
}
```

The daemon socket routes (accessed by GUI/CLI/TUI over loopback) do NOT check the capability at all — the loopback boundary is the auth.

**Claims schema version:** The `V` field in `Claims` is currently always `1`. Adding a new Perms token does not change the JSON structure, so `V` stays at `1` — old tokens simply lack `files.read` and get 403 on file endpoints, which is correct.

### Decision 5: Wails binding layer for GUI file access

**Use daemon HTTP API (via `DaemonClient`) for file operations, NOT new Wails bindings.**

Existing pattern decision: Wails bindings are used for GUI-specific operations (open file dialog, open browser URL, show notification, save file dialog). Session data (list, kill, rename, status) all go through `DaemonClient` HTTP calls. File browsing is session data, not GUI-shell behavior.

The `FileBrowserTab.tsx` makes `fetch()` calls directly to the daemon socket relay port (same as how the web terminal connects). This is consistent with how `TerminalPanel.tsx` connects to the relay WSS endpoint — the daemon port is discovered via `GetRelayPort()` Wails binding at startup and stored in App.tsx state.

Concretely: `FileBrowserTab.tsx` calls `fetch(\`http://127.0.0.1:${relayPort}/sessions/${sessionId}/files/list?path=${encodeURIComponent(path)}\`)`.

For remote sessions, it uses the pre-constructed `url` base from the remote session descriptor (already available in the GUI's remote session state), appending `/api/files/list?session=<id>&cap=<token>`.

No new `App.go` Wails bindings needed for file I/O operations.

### Decision 6: Frontend tab integration

**Pattern: per-session, non-singleton tab. Opened from DaemonManagerPanel + SessionSharePanel.**

New tab type: `'file-browser'`. Tab ID: `'__files__' + sessionId`. This makes it per-session (one file browser per session) while still being findable via the `type + sessionId` composite key.

```typescript
// In App.tsx
const handleOpenFileBrowser = useCallback((sessionId: string, sessionName: string) => {
  const tabId = '__files__' + sessionId
  const existing = tabs.find(t => t.id === tabId)
  if (existing) {
    setActiveId(existing.id)
    return
  }
  const newTab: Tab = {
    id: tabId,
    name: sessionName + ' — Files',
    sessionId,
    cli: '',
    type: 'file-browser',
  }
  setTabs(prev => [...prev, newTab])
  setActiveId(tabId)
}, [tabs])
```

**Trigger points:**
1. DaemonManagerPanel: new "Browse Files" button/link per session row (alongside Kill + web toggle)
2. TabBar: right-click context menu on a session tab → "Browse Files" item (same menu that has "Rename" and "Save Terminal As…")

Both routes call `handleOpenFileBrowser(sessionId, sessionName)` from App.tsx.

### Decision 7: TUI integration

**New `tabFiles tabID` constant. Files view is a content tab, not a sidebar item. Scoped to the currently selected session in the sidebar.**

Rationale: the TUI has exactly 4 content tabs today (Home, Sessions, Remote, Settings). Adding Files as a 5th tab is consistent. The TUI does not have per-session sub-tabs — the unified session list in the Sessions tab drives selection context. The Files tab shows the file browser for `m.sessions[m.selected]`.

Navigation: pressing `f` on a selected session in the Sessions tab opens `tabFiles`. Alternatively, a sidebar entry at position `sidebarFocus=4` (after Settings) provides direct access. Sidebar-only navigation is simpler and avoids key collision with `f` potentially used for find operations.

**New files in TUI package:**
- `internal/tui/files.go` — `FilesModel` struct with directory state, cursor position, type-ahead buffer, preview content; `Update(msg)` and `View()` methods following Bubble Tea patterns from `modal.go`
- `internal/tui/files_fetch.go` — `fetchDirMsg`, `fetchPreviewMsg`, Tea commands calling `DaemonClient.ListFiles` / `ReadFile`

**State in Model:**
```go
// internal/tui/model.go additions
tabFiles tabID = iota  // add to const block

// In Model struct:
filesModel *FilesModel  // nil until first tabFiles open; non-nil after
```

The `FilesModel` is created lazily when `tabFiles` is first opened, seeded with the currently selected session. Re-opening the tab for a different session recreates the `FilesModel`.

### Decision 8: Build order

**Phase dependency graph:**

```
Phase A: internal/files package + daemon API routes
    ↓ unblocks
Phase B: WebServer routes + capability bit     Phase C: DaemonClient methods
    ↓                                               ↓
Phase D: FileBrowserTab.tsx (frontend)         Phase E: TUI files view
    ↓
Phase F: Web-share viewer test + Playwright e2e
```

**Recommended phase sequencing:**

**Phase 1 — Sandbox core + daemon routes (no frontend)**
- `internal/files/sandbox.go` — `Sandbox.Resolve()` with traversal prevention
- `internal/files/handler.go` — `List`, `Stat`, `Read` HTTP handlers
- Path traversal fuzz tests in `internal/files/sandbox_test.go` (required before merge)
- `SessionEngine` gains `sessionWorkDirs map[string]string` (populated at CreateSession)
- `SessionEngine.GetSessionWorkDir(id string) string` method
- Daemon API routes: `GET /sessions/{id}/files/list`, `/stat`, `/read` in `api.go`
- `DaemonClient.ListFiles`, `StatFile`, `ReadFile` in `client.go`
- `daemon/types.go` additions: `FileListResponse`, `FileEntry`, `FileStatResponse`
- No frontend dependency — full integration test via `curl` against daemon socket

**Phase 2 — WebServer routes + capability bit**
- `Claims.Perms` string extended to carry `files.read`
- `issueCapabilitiesForSession` adds `files.read` to write-token Perms
- `requireFilesRead` middleware wrapper in `capability_mw.go`
- WebServer routes in `setupRoutes()`: `GET /api/files/list`, `/stat`, `/read` with `requireFilesRead`
- `webserver.SetFilesHandlerProvider(fn func(sessionID string) http.Handler)` injection point
- Daemon wires the provider at startup (same pattern as `SetPluginSettingsProvider`)
- Web-share viewer 403 test (read-only token + file endpoint = 403)
- Zero new CSP violations (file API is JSON, no new asset types)

Phases 1 and 2 can be combined in one phase or split — Phase 2 depends on Phase 1's `internal/files` package.

**Phase 3 — FileBrowserTab.tsx (desktop + web)**
- New `frontend/src/components/FileBrowserTab.tsx`
- Tree/list layout, breadcrumb bar, type-ahead filter, preview pane
- Wired to daemon local API (fetch via relay port) for local sessions
- Wired to remote HTTPS base URL for remote sessions
- `handleOpenFileBrowser` in App.tsx + tab type `'file-browser'`
- DaemonManagerPanel "Browse Files" trigger
- TabBar right-click context menu "Browse Files" item
- Playwright e2e: local session file browse, remote session file browse, viewer 403

**Phase 4 — TUI files view**
- `internal/tui/files.go` + `files_fetch.go`
- `tabFiles` added to model.go + update.go
- DaemonClient `ListFiles` / `ReadFile` called from TUI Tea commands
- Keyboard navigation per PROJECT.md TUI requirements
- Help overlay updated with file browser keys

**Why this order:**
1. The `internal/files` package is dependency-free and its correctness (especially traversal prevention) must be proven before any surface depends on it. Ship and fuzz-test it in isolation.
2. WebServer routes and capability bit in Phase 2 can be verified by a web-only integration test (curl with a token) before the frontend exists.
3. Frontend tab in Phase 3 can use the already-frozen daemon API without risk of API shape changes.
4. TUI in Phase 4 reuses the same `DaemonClient` methods added in Phase 1 — no API work needed.

### Decision 9: Streaming and Range support

**Use `http.ServeContent` for the `/read` endpoint. No relay frame involvement.**

`http.ServeContent(w, r, filename, modtime, content)` handles:
- `Range: bytes=N-M` headers → `206 Partial Content`
- `If-Modified-Since` / `ETag` conditional GETs
- Content-Type inference from filename
- Correct `Content-Length`

For the `internal/files/handler.go` implementation:

```go
func (h *Handler) ServeRead(w http.ResponseWriter, r *http.Request) {
    root := h.rootFn() // injected function returning sandbox root
    relPath := r.URL.Query().Get("path")
    abs, err := h.sandbox.Resolve(root, relPath)
    if err != nil {
        http.Error(w, "access denied", http.StatusForbidden)
        return
    }
    f, err := os.Open(abs)
    if err != nil { ... }
    defer f.Close()
    fi, _ := f.Stat()
    // 5 MB cap: if fi.Size() > 5<<20, serve first 5MB only (text preview cap)
    http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}
```

The relay frame protocol is not involved. File reads are independent REST calls, not part of the PTY stream. For remote sessions, the call goes directly to the remote daemon's WebServer over HTTPS (Tailscale) — the relay is not in the path at all.

---

## 3. System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        agenthub binary                                   │
│                                                                           │
│  ┌──────────┐  ┌───────────┐  ┌──────────┐  ┌───────────────────────┐  │
│  │  Wails   │  │    CLI    │  │   TUI    │  │  Daemon (background)  │  │
│  │  (GUI)   │  │ subcommand│  │  BubbleTea│  │                      │  │
│  └────┬─────┘  └─────┬─────┘  └────┬─────┘  │  SessionEngine       │  │
│       │              │              │        │    sessionWorkDirs    │  │
│       └──────────────┴──────────────┘        │  FileHandler mount   │  │
│                      │ Unix socket           │                      │  │
│                      ▼ HTTP/JSON             │  ┌────────────────┐  │  │
│              ┌───────────────┐               │  │internal/files/ │  │  │
│              │  DaemonClient  │──────────────▶│  │  Sandbox.Resolve│  │  │
│              │  .ListFiles()  │               │  │  Handler.List  │  │  │
│              │  .StatFile()   │               │  │  Handler.Stat  │  │  │
│              │  .ReadFile()   │               │  │  Handler.Read  │  │  │
│              └───────────────┘               │  └────────────────┘  │  │
│                                              │                      │  │
│                                              │  WebServer           │  │
│                                              │   /api/files/*       │  │
│                                              │   requireFilesRead   │  │
│                                              └───────────────────────┘  │
│                                                         │               │
│                                                         ▼ HTTPS/TLS     │
│                                                  Tailscale network       │
│                                               (remote clients + peers)   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Flow Diagrams

### 4.1 Local session — GUI file list

```
User clicks "Browse Files" on session row in DaemonManagerPanel
    │
    ▼
App.tsx handleOpenFileBrowser(sessionId, sessionName)
    │  find-or-add tab of type 'file-browser' with id '__files__'+sessionId
    ▼
FileBrowserTab.tsx mounts, initial path = "."
    │
    ▼
fetch(`http://127.0.0.1:${relayPort}/sessions/${id}/files/list?path=.`)
    │  relayPort from GetRelayPort() Wails binding (read on app startup)
    ▼
internal/daemon/api.go  GET /sessions/{id}/files/list
    │  no auth check (loopback socket)
    │  engine.GetSessionWorkDir(id) → "/Users/ken/proj"
    ▼
internal/files.Handler.List(root="/Users/ken/proj", relPath=".")
    │  sandbox.Resolve → "/Users/ken/proj" (clean, no escape)
    ▼
os.ReadDir("/Users/ken/proj")
    │
    ▼
JSON response: [{name: "src", isDir: true, size: 0, mtime: "..."}, ...]
    │
    ▼
FileBrowserTab.tsx renders directory listing
```

### 4.2 Remote (tailnet) session — GUI file list

```
Remote session attached; FileBrowserTab receives:
  - sessionId: "abc123"
  - baseURL: "https://remotebox.ts.net:7443"  (from RemoteSession.url)
  - capToken: "<read,write,files.read token>"  (obtained via capability exchange)
    │
    ▼
fetch(`https://remotebox.ts.net:7443/api/files/list?session=abc123&path=.&cap=<token>`)
    │  HTTPS over Tailscale — same channel used for tailnet peer discovery
    ▼
remote daemon's internal/webserver/server.go
    │  requireFilesRead middleware:
    │    1. verify HMAC token
    │    2. check claims.SID == "abc123"
    │    3. check grant active
    │    4. check session web-enabled
    │    5. check "files.read" in claims.Perms
    ▼
remote internal/files.Handler.List(root from remote sessionWorkDirs["abc123"])
    │
    ▼
JSON response over HTTPS/Tailscale
    │
    ▼
FileBrowserTab.tsx renders directory listing (same component, different data source)
```

---

## 5. Component Map: New vs Modified Files

### 5.1 New files

| File | Purpose |
|------|---------|
| `internal/files/sandbox.go` | `Sandbox` struct + `Resolve(root, relPath) (abs string, err error)` — traversal prevention: `filepath.Clean` → symlink-resolve via `filepath.EvalSymlinks` → `strings.HasPrefix` guard; rejects absolute paths, null bytes, Windows drive letters/UNC |
| `internal/files/handler.go` | `Handler` struct with `List`, `Stat`, `Read` HTTP handlers; `Read` delegates to `http.ServeContent` for Range support; 5 MB preview cap |
| `internal/files/sandbox_test.go` | Path traversal fuzz tests (Go fuzzing, `go test -fuzz`); unit tests for all rejection cases |
| `internal/files/handler_test.go` | Round-trip HTTP tests via `httptest.NewRecorder` |
| `frontend/src/components/FileBrowserTab.tsx` | Tree/list layout, breadcrumb bar bounded at session cwd, sort controls, type-ahead filter, preview pane; uses `fetch()` to daemon or remote HTTPS depending on `isRemote` prop |
| `frontend/src/components/__tests__/FileBrowserTab.test.tsx` | Source-inspection tests (same `fs.readFileSync` pattern) |
| `internal/tui/files.go` | `FilesModel` struct (directory listing, cursor, type-ahead, preview); `Update(msg) (FilesModel, tea.Cmd)` + `View(width, height int) string`; lipgloss-bordered in TokyoNight palette |
| `internal/tui/files_fetch.go` | `fetchDirMsg`, `fetchPreviewMsg` types; `fetchDirCmd`, `fetchPreviewCmd` Tea command functions calling DaemonClient |

### 5.2 Modified files

| File | Change |
|------|--------|
| `internal/daemon/engine.go` | Add `sessionWorkDirs map[string]string` field to `SessionEngine`; populate in `CreateSession`; add `GetSessionWorkDir(id string) string` method; initialize map in `NewSessionEngine()` |
| `internal/daemon/types.go` | Add `FileEntry`, `FileListResponse`, `FileStatResponse` wire types |
| `internal/daemon/api.go` | Register `GET /sessions/{id}/files/list`, `/sessions/{id}/files/stat`, `/sessions/{id}/files/read` routes in `registerRoutes()`; implement handlers that call `internal/files.Handler` |
| `internal/daemon/client.go` | Add `ListFiles(sessionID, path string) ([]FileEntry, error)`, `StatFile(sessionID, path string) (FileStatResponse, error)`, `ReadFile(sessionID, path string) ([]byte, error)` methods |
| `internal/webserver/server.go` | Add `filesHandlerProvider func(sessionID string) http.Handler` field; `SetFilesHandlerProvider` setter; register `GET /api/files/list`, `/api/files/stat`, `/api/files/read` routes in `setupRoutes()` wrapped with `requireFilesRead` |
| `internal/webserver/capability_mw.go` | Add `requireFilesRead` wrapper function (chains `requireCapability` + `files.read` Perms check) |
| `internal/capability/capability.go` | No structural change — `Perms` is already a free-form string. Add a `const PermFilesRead = "files.read"` constant for use at issuance and check sites |
| `internal/daemon/api.go` (capability section) | `issueCapabilitiesForSession`: add `files.read` to the write-token Perms: `"read,write,files.read"` |
| `frontend/src/App.tsx` | Add `handleOpenFileBrowser(sessionId, sessionName)` callback; add `'file-browser'` tab type to `Tab` union; render `<FileBrowserTab>` when `activeId` matches a file-browser tab |
| `frontend/src/components/DaemonManagerPanel.tsx` | Add "Browse Files" button per session row; accept `onOpenFileBrowser: (id: string, name: string) => void` prop |
| `frontend/src/components/TabBar.tsx` | Add "Browse Files" item to right-click context menu on session tabs; call `onOpenFileBrowser` prop |
| `internal/tui/model.go` | Add `tabFiles tabID` constant; add `filesModel *FilesModel` field to `Model` |
| `internal/tui/update.go` | Handle `tabFiles` in tab rendering switch; keyboard shortcut `f` on Sessions tab opens file browser for selected session; wire `filesModel.Update` into the main `Update` switch |
| `internal/tui/view.go` | Render `filesModel.View()` when `activeTabID() == tabFiles` |

**Files that do NOT change:**
- `internal/relay/` — no relay protocol change
- `internal/capability/capability.go` — struct is unchanged; only a constant is added
- `App.go` — no new Wails bindings (file ops go through DaemonClient HTTP, not Wails)
- `wailsjs/go/main/App.{d.ts,js}` — no new stubs needed
- `web/embed.go` — file browser is served via REST from the daemon, not an embedded HTML page
- `internal/tailnet/` — unchanged

---

## 6. Anti-Patterns to Avoid

### Anti-Pattern 1: Relay frame type for file ops

**What people might do:** Add a new `0x21 MsgFileRequest` / `0x22 MsgFileResponse` relay frame pair, routing file requests through the existing WSS connection.

**Why wrong:** The relay is a PTY fan-out protocol. It has no request-response correlation, no timeout handling, no error propagation, and no per-request headers. Implementing file ops over it would require building a mini-HTTP over binary frames. HTTP already exists and is already proven. The relay's `0x20–0x2F` reserved range is for server-push metadata frames (one-directional), not for request-response protocols.

**Do instead:** REST over HTTPS. For remote sessions, Tailscale provides the transport.

### Anti-Pattern 2: Storing WorkDir in the Session PTY struct

**What people might do:** Add `WorkDir string` to `internal/pty/session.go` and populate it in `NativePTYBackend.Create`.

**Why wrong:** `internal/pty` is a low-level PTY package with no session-metadata concern. Adding application-level metadata (WorkDir) to it couples the PTY abstraction to the session concept. The `SessionEngine` already maintains parallel maps for `tabNames` and `sessionCLIs` — `sessionWorkDirs` follows the same established pattern.

**Do instead:** `SessionEngine.sessionWorkDirs map[string]string`, populated at `CreateSession`, never exposed outside the engine except via `GetSessionWorkDir`.

### Anti-Pattern 3: New Wails binding for file list/read

**What people might do:** Add `func (a *App) ListFiles(sessionID, path string) ([]FileEntry, error)` to `App.go`.

**Why wrong:** Wails bindings are for GUI-shell operations (file dialogs, browser open, notifications, tray). Session data goes through `DaemonClient` HTTP. The file browser needs to work for remote sessions (where a Wails binding would fail — it can only reach the local daemon). Using `fetch()` to the relay port (for local) or to the remote HTTPS URL (for remote) is the correct and already-proven pattern.

**Do instead:** `DaemonClient.ListFiles` for local sessions; direct `fetch()` to remote HTTPS base URL for remote sessions.

### Anti-Pattern 4: Singleton file browser tab

**What people might do:** Make the file browser a singleton tab (one global file browser that switches sessions via a session picker).

**Why wrong:** The existing pattern for session-bound content is per-session tabs (TerminalPanel is per-session). Users expect to compare files from two sessions simultaneously by having two file browser tabs open. The `__files__<sessionId>` tab ID pattern is a direct extension of the existing non-singleton session tab model.

**Do instead:** Per-session tab, one per session, with find-or-activate semantics (re-open focuses existing tab for that session).

### Anti-Pattern 5: Symlink-only traversal check

**What people might do:** Check only for `..` segments in the path string without resolving symlinks.

**Why wrong:** A symlink in the session working directory pointing to `/etc/passwd` is a valid traversal attack. The full sequence must be: `filepath.Clean` (normalize) → `filepath.EvalSymlinks` (resolve all symlinks in the absolute path) → `strings.HasPrefix(resolved, root)` (verify resolved path stays under root). The `EvalSymlinks` call must happen on the absolute path constructed from root + rel, not the rel path alone.

**Do instead:** Implement in `internal/files/sandbox.go` as the single source of truth, tested with fuzz tests before any consumer merges.

---

## 7. Open Questions for Phase Research

1. **Windows path sandboxing:** `filepath.EvalSymlinks` on Windows also handles junctions and reparse points. Verify that `strings.HasPrefix` on Windows paths is case-insensitive (Windows filesystem is case-insensitive; Go's `strings.HasPrefix` is case-sensitive). May need `strings.EqualFold` for the prefix check or normalization to lowercase.

2. **Session WorkDir for shell sessions:** Shell sessions use `$HOME` as WorkDir when none is specified (v3.3 behavior). The file browser sandbox root should be the actual resolved WorkDir (after `$HOME` substitution), not the empty string passed by the user. Verify the engine stores the resolved WorkDir in `sessionWorkDirs`, not the raw request value.

3. **Per-session file browser cap token for GUI:** The GUI's local file calls go to the daemon socket (no token needed). But the GUI also needs tokens for remote session file calls. How does the GUI obtain a token for a remote session's files endpoint? Current remote session attach uses the `url` field from `RemoteSession` which contains `?cap=<token>`. Verify that the token's `Perms` will include `files.read` once Phase 2 is deployed. This requires both ends to be running v3.4+. For v3.4, remote file access only works when the remote peer is also v3.4+.

4. **CSP audit for file browser:** The file browser serves JSON from the same origin (daemon API or webserver). No new asset types. The preview pane renders text (via `<pre>`) and markdown (via a markdown renderer — choice of library needs to be made). Markdown renderers that use `innerHTML` may need CSP relaxation. Recommend using a renderer that produces only safe HTML (no `dangerouslySetInnerHTML` with untrusted content, or a sanitization layer). This is a Phase 3 concern.

5. **Binary file preview in TUI:** PROJECT.md says "binaries show 'Use desktop or web to preview'". The TUI needs a heuristic for binary detection — check for null bytes in the first 512 bytes (same heuristic used by `git` and `file`). Implement in `internal/files/handler.go` as a helper and expose it in the `FileEntry.IsBinary bool` field.

---

## 8. Sources

- `internal/capability/capability.go` — verified Claims struct, Perms format, Sign/Verify interface
- `internal/webserver/capability_mw.go` — verified requireCapability pattern (all 7 enforcement steps)
- `internal/webserver/server.go` — verified setupRoutes(), injection pattern (SetPluginSettingsProvider, SetSessionResolver), mux structure
- `internal/daemon/api.go` — verified registerRoutes(), issueCapabilitiesForSession(), Perms string values
- `internal/daemon/engine.go` — verified SessionEngine fields, sessionWorkDirs gap, tabNames/sessionCLIs map pattern
- `internal/daemon/types.go` — verified wire type patterns
- `internal/daemon/client.go` — verified DaemonClient method patterns
- `internal/relay/protocol.go` — verified frame types, 0x20-0x2F reserved range comment
- `internal/relay/server.go` — verified relay is PTY-only, no request-response semantics
- `App.go` — verified no file-related Wails bindings; DaemonClient is the session data channel
- `frontend/src/App.tsx` — verified Tab type structure, singleton vs per-session tab patterns, find-or-add pattern
- `frontend/src/components/DaemonManagerPanel.tsx` — verified prop interface, session row structure
- `internal/tui/model.go` — verified tabID const pattern, openTab find-or-activate, Model struct fields
- `internal/tui/update.go` — verified sidebarTabs array, tab rendering switch
- `.planning/PROJECT.md` — v3.4 milestone requirements, scope discipline
- `.planning/milestones/v3.3.1-research/ARCHITECTURE.md` — plugin settings injection pattern (SetPluginSettingsProvider), capabilities issuance pattern
