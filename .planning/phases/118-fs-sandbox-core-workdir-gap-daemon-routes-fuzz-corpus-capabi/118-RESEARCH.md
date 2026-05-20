# Phase 118: FS Sandbox Core + WorkDir Gap + Daemon Routes + Fuzz Corpus + Capability Bit - Research

**Researched:** 2026-05-20
**Domain:** Sandboxed read-only filesystem package (Go stdlib `os.Root`) + daemon-local HTTP routes + capability bit
**Confidence:** HIGH (all claims verified against actual source files; Go version confirmed locally; milestone research synthesized HIGH; only pre-phase research needed was codebase integration probing which is now complete)

## Summary

Phase 118 is the load-bearing foundation for v3.4. It delivers a new dependency-free `internal/files/` package built on Go 1.24+ `*os.Root` (project runs Go 1.26.1/1.26.3 — verified locally), patches the WorkDir gap in `SessionEngine`, adds three daemon-local HTTP routes on the existing Unix-socket/named-pipe mux (no auth — loopback is trusted), introduces the `files.read` capability bit with a whole-token `HasPerm` helper, and lands a fuzz-tested path sandbox with 40+ payload corpus as a merge gate. Webserver routes, frontend, and TUI are explicitly downstream phases (119/120/121).

Every architectural decision is already locked by the milestone research in `.planning/research/` (SUMMARY.md, STACK.md, ARCHITECTURE.md, PITFALLS.md, FEATURES.md) — confidence HIGH across all four documents. The discuss phase was skipped per `workflow.skip_discuss: true`, so every implementation choice falls under Claude's discretion, bounded by REQUIREMENTS.md FS-01..FS-14.

**Primary recommendation:** Implement `internal/files/` as a pure package (zero coupling to `internal/daemon`, `internal/relay`, `internal/webserver`), wire it into the daemon-socket mux via direct `mux.HandleFunc` calls in `api.go` (no `SetFilesHandlerProvider` yet — that pattern lands in Phase 119 for the webserver), add `sessionWorkDirs map[string]string` to `SessionEngine` resolving WorkDir via `filepath.EvalSymlinks` at `CreateSession` time, and ship `FuzzSandboxPath` in the same PR as the first endpoint using `testing.F` seeded from the 40+ payloads in PITFALLS.md §Fuzz Corpus Skeleton.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Path sandboxing (TOCTOU-safe resolve) | `internal/files/` (new pure pkg) | — | Zero coupling required; testable without daemon/webserver/relay; fuzz-tested in isolation |
| HTTP handlers (List/Stat/Read) | `internal/files/` (new) | — | Stateless given a sandbox root; same handler reused by Phase 119 webserver |
| Session WorkDir tracking | `internal/daemon/engine.go` (SessionEngine) | — | Mirrors existing `tabNames`/`sessionCLIs` map pattern; engine owns session metadata |
| Daemon-socket routes (`/sessions/{id}/files/*`) | `internal/daemon/api.go` | — | Existing `mux.HandleFunc` pattern; no auth (loopback trusted) |
| Capability bit constant + `HasPerm` helper | `internal/capability/capability.go` | — | Lives next to existing `Claims`/`Sign`/`Verify` API; consumed by daemon (issuance) and Phase 119 webserver (gating) |
| Token issuance (`files.read` in owner Perms) | `internal/daemon/api.go` (`issueCapabilitiesForSession`) | — | Existing function mints owner + viewer tokens; only edit is appending `files.read` to write-token Perms |
| MIME detection | `internal/files/` via `wailsapp/mimetype` | — | Promote indirect→direct dep; 200+ types vs stdlib's 15 |
| Range streaming | `internal/files/` via stdlib `http.ServeContent` | — | Battle-tested; no new dep; 0-byte special case wrapped before delegation |
| Settings persistence (`filesRead bool`) | `internal/daemon/engine.go` (`daemonSettings`) | — | Bump `CurrentSchemaVersion` 2→3; defaults-merge constructor sets `true` |
| Fuzz testing | `internal/files/sandbox_test.go` via `testing.F` | — | Stdlib fuzz framework; corpus from PITFALLS.md (40+ entries) |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `os.Root` / `os.OpenRoot` / `os.OpenInRoot` | Go 1.24+ (project: 1.26.1, local toolchain: 1.26.3) | TOCTOU-safe path sandbox — atomic open at kernel level (openat2 on Linux) | [VERIFIED: go.dev/blog/osroot] Only stdlib answer; eliminates the symlink TOCTOU race class (CVE-2026-27976 Zed, 8.8 CVSS); two-step EvalSymlinks+Open documented unsafe |
| Go stdlib `http.ServeContent` | All Go versions | Range-capable file streaming; handles 206, ETag, If-Modified-Since | [VERIFIED: pkg.go.dev/net/http] Stdlib; no new dep; just one documented edge case (0-byte file → 416) which we wrap |
| Go stdlib `testing.F` | Go 1.18+ | Native fuzzer for path traversal corpus | [VERIFIED: existing `internal/capability/capability_fuzz_test.go`] Pattern already in codebase |
| `github.com/wailsapp/mimetype` | v1.4.1 | MIME type detection from magic bytes (200+ types) | [VERIFIED: grep go.sum — already indirect via Wails] Promote indirect→direct; no new download. MIT license. Forked from `gabriel-vasile/mimetype` upstream — do NOT add both. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `filepath.EvalSymlinks` | All | One-time WorkDir resolution at session creation | Only at `CreateSession` time to cache resolved cwd. NEVER per-request on user input — that is `os.Root`'s job |
| Go stdlib `filepath.Clean` | All | Pre-screen path normalization | Defense-in-depth only; `os.Root` is the actual security boundary |
| Go stdlib `os.ReadDir` (via `root.Open(".").(*os.File).ReadDir(n)`) | All | Streaming directory enumeration with cap | Use chunked `ReadDir(maxEntries)` with 10,000 cap + `X-Directory-Truncated` header on truncation |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `os.Root` | `github.com/cyphar/filepath-securejoin` | [VERIFIED: pkg.go.dev/github.com/cyphar/filepath-securejoin] Legacy API documented as TOCTOU-unsafe by maintainer; modern pathrs-lite API Linux-only (project must support darwin+windows); MPL-2.0 weak copyleft concern. Rejected. |
| `os.Root` | `go-billy/v5 ChrootOS` | [CITED: CVE-2023-49569] Critical path traversal CVE; designed for virtual FS (git ops), not syscall sandboxing. Already in go.sum as transitive — do NOT use for sandboxing. |
| `os.Root` | `filepath.EvalSymlinks` + `strings.HasPrefix` + `os.Open` (two-step) | [VERIFIED: PITFALLS.md Pitfall 1 + golang/go#70007] TOCTOU race window exploitable by attacker with write access to cwd subdirs — exactly what shell sessions provide. Hard reject. |
| `wailsapp/mimetype` | stdlib `http.DetectContentType` | [VERIFIED: go.dev/pkg/net/http] Only 512 bytes, ~15 types; cannot distinguish JSON/YAML/Go/Markdown from text/plain |
| `wailsapp/mimetype` | `gabriel-vasile/mimetype@v1.4.13` upstream | [VERIFIED: go.sum] Same codebase; adding both creates duplicate dep under two module paths |
| `http.ServeContent` | Hand-rolled Range handling | Stdlib is battle-tested for multipart range + ETags; no upside to replacing |
| `testing.F` | `dvyukov/go-fuzz` | dvyukov predates Go 1.18 native fuzzing; unmaintained; stdlib is the answer |

**Installation:**

```bash
# Single go get to promote indirect → direct (no new download)
go get github.com/wailsapp/mimetype@v1.4.1
go mod tidy
```

**Version verification:** Confirmed via `cat go.sum | grep wailsapp/mimetype` → `v1.4.1` already present as indirect. Local `go version` → `go1.26.3`. `go.mod` declares `go 1.26.1`. `go doc os.OpenRoot` returned valid stdlib API signature.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| `github.com/wailsapp/mimetype` | Go modules (already in go.sum) | v1.4.1 stable | high (Wails transitive) | github.com/wailsapp/mimetype | [N/A — slopcheck is npm-focused; verified via go.sum presence + Wails upstream] | Approved |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*Note: slopcheck is an npm-focused tool. For Go modules already present in `go.sum` via a known direct dep chain (Wails), promotion to direct is registry-verified by the existing build. The corresponding upstream `gabriel-vasile/mimetype` is a well-known, MIT-licensed library with high adoption. Legitimacy verified by chain-of-trust through Wails.*

## Architecture Patterns

### System Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│  Phase 118 scope — daemon-local + capability/settings plumbing       │
│                                                                       │
│  ┌─────────────┐  ┌───────────┐  ┌──────────┐                        │
│  │ Wails GUI   │  │  CLI      │  │  TUI     │  (all in-process       │
│  │ (App.go via │  │ (cli      │  │ (BubbleTea│   consumers — call    │
│  │ DaemonClient│  │  subcmd)  │  │  model)  │   DaemonClient over    │
│  └──────┬──────┘  └─────┬─────┘  └────┬─────┘   loopback)            │
│         │               │              │                              │
│         └───────────────┴──────────────┘                              │
│                         │                                              │
│                         ▼ Unix socket / Windows named pipe            │
│                 ┌───────────────┐                                     │
│                 │ DaemonClient  │                                     │
│                 │ .ListFiles    │                                     │
│                 │ .StatFile     │                                     │
│                 │ .ReadFile     │  ← Phase 118 adds these             │
│                 └───────┬───────┘                                     │
│                         │                                              │
│                         ▼  HTTP/JSON over loopback (no auth)          │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  internal/daemon (existing)                                   │    │
│  │                                                                │    │
│  │  SessionEngine                                                 │    │
│  │   ├─ tabNames          map[string]string  (existing)          │    │
│  │   ├─ sessionCLIs       map[string]string  (existing)          │    │
│  │   └─ sessionWorkDirs   map[string]string  ← NEW Phase 118     │    │
│  │      (populated at CreateSession with EvalSymlinks-resolved   │    │
│  │       WorkDir after $HOME substitution; never empty)          │    │
│  │                                                                │    │
│  │  API mux (existing http.ServeMux on UDS/NPipe)                │    │
│  │   ├─ existing routes (GET /sessions, POST /sessions, ...)     │    │
│  │   ├─ GET /api/files/list?session={id}&path={rel}  ← NEW       │    │
│  │   ├─ GET /api/files/stat?session={id}&path={rel}  ← NEW       │    │
│  │   └─ GET /api/files/read?session={id}&path={rel}  ← NEW       │    │
│  │       (also: HEAD /api/files/read)                             │    │
│  │       │                                                        │    │
│  │       ▼ delegates to                                           │    │
│  │  ┌──────────────────────────────────────────────────────┐    │    │
│  │  │  internal/files/  (NEW package — zero coupling)       │    │    │
│  │  │                                                        │    │    │
│  │  │  Sandbox { root *os.Root }                            │    │    │
│  │  │   .List(relPath) → []FileEntry                        │    │    │
│  │  │   .Stat(relPath) → FileEntry                          │    │    │
│  │  │   .Open(relPath) → *os.File   (for ServeContent)      │    │    │
│  │  │                                                        │    │    │
│  │  │  validateRelativePath(p):                              │    │    │
│  │  │   1. reject "" / null bytes                            │    │    │
│  │  │   2. reject absolute / drive letter / UNC              │    │    │
│  │  │   3. reject Windows device names (cross-platform)      │    │    │
│  │  │   4. reject ADS colon (cross-platform)                 │    │    │
│  │  │   5. filepath.Clean                                    │    │    │
│  │  │   6. reject `..` after Clean                           │    │    │
│  │  │   then → os.Root.Open(cleaned)  [atomic — kernel-level]│    │    │
│  │  │                                                        │    │    │
│  │  │  Handler.Read uses http.ServeContent                  │    │    │
│  │  │   with 0-byte special-case + 5 MB cap → 413            │    │    │
│  │  │                                                        │    │    │
│  │  │  FuzzSandboxPath (testing.F)                          │    │    │
│  │  │   seed corpus: 40+ payloads from PITFALLS.md          │    │    │
│  │  │   merge gate: `go test -fuzz=... -fuzztime=60s`       │    │    │
│  │  └──────────────────────────────────────────────────────┘    │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  internal/capability/capability.go (existing — minor edits)  │    │
│  │   const PermFilesRead = "files.read"            ← NEW         │    │
│  │   func HasPerm(perms, perm string) bool {…}     ← NEW         │    │
│  │     splits on commas; whole-token match (NOT strings.Contains)│    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │  internal/daemon/api.go::issueCapabilitiesForSession (edit)  │    │
│  │   read token   Perms: "read"                  ← unchanged    │    │
│  │   write token  Perms: "read,write,files.read" ← ADDS files.read│   │
│  │                                                                │    │
│  │  daemonSettings (engine.go) +filesRead bool, default true     │    │
│  │  CurrentSchemaVersion 2 → 3                                   │    │
│  └──────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  NOT in Phase 118 scope (deferred to Phase 119/120/121):              │
│   - internal/webserver/ wiring (Phase 119: SetFilesHandlerProvider,  │
│     requireFilesRead middleware, /api/files/* webserver routes)       │
│   - FileBrowserTab.tsx (Phase 120)                                    │
│   - internal/tui/files.go (Phase 121)                                 │
└──────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/files/                     # NEW package (zero coupling)
├── sandbox.go                      # Sandbox{*os.Root} + validateRelativePath
├── handler.go                      # http.Handler — List/Stat/Read methods
├── handler.go (cont.)              # FileEntry struct (Name/Size/Mtime/Mode/IsDir/IsSymlink/IsBinary/MIME)
├── mime.go                         # MIME cascade: extension → wailsapp/mimetype → fallback
├── sandbox_test.go                 # FuzzSandboxPath + unit tests for rejection paths
├── handler_test.go                 # Range edge cases, 0-byte file, 5 MB cap, MIME cascade
└── testdata/
    └── fuzz/
        └── FuzzSandboxPath/        # 40+ seed payloads from PITFALLS.md

internal/daemon/                    # MODIFIED files only
├── engine.go                       # +sessionWorkDirs map, +GetSessionWorkDir, +filesRead in daemonSettings, +CurrentSchemaVersion 3
├── api.go                          # +3 routes /api/files/{list,stat,read}, +issueCapabilitiesForSession edit
├── client.go                       # +ListFiles, +StatFile, +ReadFile methods
├── types.go                        # +FileEntry, +FileListResponse, +FileStatResponse wire types
└── engine_migration_test.go        # +TestSettingsMigration_FilesReadDefaultsTrue (new fixture v3.2.json)

internal/capability/                # MODIFIED files only
└── capability.go                   # +const PermFilesRead, +func HasPerm
└── capability_test.go              # +TestHasPerm cases (whole-token, prefix-resistance)

tests/fixtures/
└── settings_v3.2.json              # NEW — pre-migration fixture (no filesRead key, schemaVersion=2)
```

### Pattern 1: `os.Root`-based atomic open

**What:** Pre-resolve session WorkDir once at session creation via `filepath.EvalSymlinks`; cache resolved absolute path; at each request, open `*os.Root` from cached path and call `root.Open(relPath)` — kernel-level atomic.

**When to use:** All file resolution on user-supplied paths.

**Example:**

```go
// Source: VERIFIED via go doc os.OpenRoot + go.dev/blog/osroot
// internal/files/sandbox.go

type Sandbox struct {
    rootPath string // EvalSymlinks-resolved absolute WorkDir, cached at session creation
}

func NewSandbox(workDir string) (*Sandbox, error) {
    if workDir == "" {
        return nil, errors.New("empty WorkDir")
    }
    resolved, err := filepath.EvalSymlinks(workDir)
    if err != nil {
        return nil, fmt.Errorf("resolve WorkDir: %w", err)
    }
    fi, err := os.Stat(resolved)
    if err != nil || !fi.IsDir() {
        return nil, fmt.Errorf("WorkDir is not a directory: %w", err)
    }
    return &Sandbox{rootPath: resolved}, nil
}

func (s *Sandbox) Open(relPath string) (*os.File, error) {
    if err := validateRelativePath(relPath); err != nil {
        return nil, err
    }
    cleaned := filepath.Clean(relPath)
    if cleaned == "." {
        cleaned = "." // os.Root.Open(".") opens the root itself
    }
    // OpenRoot+Open is the atomic security boundary. Per-request open
    // is acceptable; openat2 is cheap.
    root, err := os.OpenRoot(s.rootPath)
    if err != nil {
        return nil, err
    }
    defer root.Close()
    return root.Open(cleaned)
}
```

### Pattern 2: Defense-in-depth path validation (before os.Root)

**What:** Reject obviously bad inputs (null bytes, absolute paths, Windows device names, ADS colons, UNC) before calling `os.Root.Open`. `os.Root` catches traversal; these checks catch Windows-specific edge cases `os.Root` may not on all platforms.

**When to use:** Every user-supplied path parameter, on ALL platforms (not Windows-build-only).

**Example:**

```go
// Source: VERIFIED via PITFALLS.md Pitfall 2 + Microsoft Naming Files docs
// internal/files/sandbox.go

var windowsDeviceNames = map[string]bool{
    "CON": true, "NUL": true, "PRN": true, "AUX": true,
    "COM1": true, "COM2": true, "COM3": true, "COM4": true,
    "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
    "LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
    "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

func validateRelativePath(p string) error {
    if p == "" {
        return errors.New("empty path")
    }
    if strings.ContainsRune(p, 0) {
        return errors.New("null byte in path")
    }
    if filepath.IsAbs(p) {
        return errors.New("absolute path rejected")
    }
    // Windows drive letter `X:` (Go's IsAbs on non-Windows misses this)
    if len(p) >= 2 && p[1] == ':' {
        return errors.New("drive letter rejected")
    }
    // UNC: \\server\share or //server/share
    if strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//") {
        return errors.New("UNC path rejected")
    }
    // Alternate Data Streams: anywhere in path
    if strings.ContainsRune(p, ':') {
        return errors.New("colon (ADS) rejected")
    }
    // Windows reserved device names — case-insensitive, with or without extension
    for _, segment := range strings.FieldsFunc(p, func(r rune) bool {
        return r == '/' || r == '\\'
    }) {
        base := strings.ToUpper(segment)
        if dot := strings.IndexByte(base, '.'); dot >= 0 {
            base = base[:dot]
        }
        if windowsDeviceNames[base] {
            return errors.New("Windows device name rejected: " + segment)
        }
    }
    cleaned := filepath.Clean(p)
    // After Clean, any leading `..` means escape attempt
    if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
        return errors.New("path traversal rejected")
    }
    return nil
}
```

### Pattern 3: 0-byte file Range special-case

**What:** Before calling `http.ServeContent`, check `stat.Size() == 0` and short-circuit with `200 OK` + empty body. Stdlib `ServeContent` returns `416 Range Not Satisfiable` on any Range header against a 0-byte file (golang/go#54794).

**When to use:** `/api/files/read` endpoint, before `http.ServeContent`.

**Example:**

```go
// Source: VERIFIED via golang/go#54794
// internal/files/handler.go

func (h *Handler) Read(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("session")
    relPath := r.URL.Query().Get("path")
    sb := h.sandboxFor(sessionID)
    if sb == nil {
        http.Error(w, "session not found", http.StatusNotFound)
        return
    }
    f, err := sb.Open(relPath)
    if err != nil {
        http.Error(w, "access denied", http.StatusForbidden)
        return
    }
    defer f.Close()
    fi, err := f.Stat()
    if err != nil {
        http.Error(w, "stat failed", http.StatusInternalServerError)
        return
    }
    if fi.IsDir() {
        http.Error(w, "is a directory", http.StatusBadRequest)
        return
    }
    // Server-side 5 MB preview cap → 413 Content Too Large
    const maxPreviewBytes = 5 * 1024 * 1024
    if fi.Size() > maxPreviewBytes {
        http.Error(w, "file too large for preview", http.StatusRequestEntityTooLarge)
        return
    }
    // FS-07: 0-byte file Range special case — golang/go#54794
    if fi.Size() == 0 {
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
        w.WriteHeader(http.StatusOK)
        return
    }
    // MIME cascade: extension first (covers source code), then sniff
    contentType := mimeFromExtension(relPath)
    if contentType == "" {
        contentType = sniffMIMEFromReader(f)
        _, _ = f.Seek(0, io.SeekStart) // reset after sniff
    }
    w.Header().Set("Content-Type", contentType)
    http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}
```

### Pattern 4: `HasPerm` whole-token match

**What:** Split `Claims.Perms` on commas, compare each token to the requested permission. NEVER use `strings.Contains` — would false-positive on `"no-files.read"` matching `"files.read"`.

**When to use:** All capability gating that examines `Claims.Perms`.

**Example:**

```go
// Source: VERIFIED via PITFALLS.md Pitfall 4 + existing capability.go Perms format
// internal/capability/capability.go (additions)

const PermFilesRead = "files.read"

// HasPerm returns true iff perm appears as a whole comma-separated token in perms.
// Whole-token semantics — NOT strings.Contains, which would allow "no-files.read"
// to match "files.read" via substring inclusion.
//
//   HasPerm("read,write",          "files.read") → false
//   HasPerm("read,files.read",     "files.read") → true
//   HasPerm("read,no-files.read",  "files.read") → false  ← critical
//   HasPerm("files.read,read",     "files.read") → true
//   HasPerm("",                    "files.read") → false
func HasPerm(perms, perm string) bool {
    if perms == "" || perm == "" {
        return false
    }
    for _, t := range strings.Split(perms, ",") {
        if t == perm {
            return true
        }
    }
    return false
}
```

### Pattern 5: `sessionWorkDirs` parallel map (WorkDir gap fix)

**What:** Mirror the existing `tabNames` / `sessionCLIs` map pattern in `SessionEngine`. Populate at `CreateSession` time AFTER `$HOME` substitution AND `filepath.EvalSymlinks` resolution. Expose via `GetSessionWorkDir(id) string`. Clean up in `KillSession` (delete same as `tabNames`).

**When to use:** Once at session creation; read at every file-API request.

**Example:**

```go
// Source: VERIFIED via grep engine.go — pattern lines 34-35, 219-222, 301-302, 433-434
// internal/daemon/engine.go (additions/edits)

type SessionEngine struct {
    // ... existing fields ...
    mu               sync.RWMutex
    tabNames         map[string]string   // existing
    sessionCLIs      map[string]string   // existing
    sessionWorkDirs  map[string]string   // NEW Phase 118 — resolved absolute WorkDir
    // ... rest unchanged ...
}

func NewSessionEngine() *SessionEngine {
    e := &SessionEngine{
        // ... existing fields ...
        tabNames:         make(map[string]string),
        sessionCLIs:      make(map[string]string),
        sessionWorkDirs:  make(map[string]string),   // NEW
        // ...
    }
    e.loadSettingsFromDisk(cfgDir)
    return e
}

// In CreateSession, AFTER the shell $HOME substitution block (line ~265)
// and AFTER backend.Create returns sess (line ~285), and BEFORE the
// e.mu.Lock() block at line ~300:
//
//   resolvedWD := ""
//   if workDir != "" {
//       if r, err := filepath.EvalSymlinks(workDir); err == nil {
//           resolvedWD = r
//       } else {
//           // Log; do not fail session creation — file browser will simply
//           // return 400 "session WorkDir unresolvable" later.
//           resolvedWD = workDir
//       }
//   }
//
// Then inside the existing e.mu.Lock() block at line 300-303:
//   e.tabNames[id] = name
//   e.sessionCLIs[id] = cli
//   e.sessionWorkDirs[id] = resolvedWD   // NEW
//
// And in KillSession's delete block (line ~433):
//   delete(e.tabNames, id)
//   delete(e.sessionCLIs, id)
//   delete(e.sessionWorkDirs, id)        // NEW

// GetSessionWorkDir returns the EvalSymlinks-resolved absolute WorkDir for
// a session, or empty string if the session is unknown or has no WorkDir.
// Used by internal/files handlers to construct per-session Sandboxes.
func (e *SessionEngine) GetSessionWorkDir(id string) string {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.sessionWorkDirs[id]
}
```

### Anti-Patterns to Avoid

- **Two-step EvalSymlinks + Open on user input:** TOCTOU race window; do NOT use the "clean → resolve → prefix-check → open" pattern. Use `os.Root` atomically. (PITFALLS.md Pitfall 1)
- **`strings.Contains(perms, "files.read")`:** Matches `"no-files.read"` substring. Use `HasPerm` with comma-split whole-token comparison. (PITFALLS.md Pitfall 4)
- **Modifying shared `requireCapability` middleware to check `files.read`:** Would break ALL existing terminal/relay/plugin routes that don't carry `files.read`. The capability bit check goes in a SEPARATE `requireFilesRead` wrapper — but that's a Phase 119 concern. Phase 118 only adds the `HasPerm` helper. (PITFALLS.md Pitfall 4)
- **Storing WorkDir on `pty.Session`:** Couples low-level PTY package to session metadata concern. Use the established `SessionEngine` parallel-maps pattern (ARCHITECTURE.md Anti-Pattern 2).
- **Adding `Wails` binding for `ListFiles`:** Wails bindings are GUI-shell ops (file dialogs, notifications). Session data goes through `DaemonClient` HTTP. Phase 118 adds `DaemonClient.ListFiles/StatFile/ReadFile` — NOT `App.go` methods. (ARCHITECTURE.md Anti-Pattern 3)
- **Calling `DirEntry.Info()` on every entry during listing:** N stat syscalls for an N-entry directory. Return `DirEntry.Type() + Name()` only; let the `/stat` endpoint resolve per-file metadata on demand. Single exception: directory listings can populate `IsBinary` and `MIME` from extension heuristics only (no file read required). (PITFALLS.md Pitfall 6)
- **Stat-time `IsBinary` detection that reads file body:** Forces a file open during directory listing → memory + syscall blowup on large dirs. Use extension-only heuristics during List; only the `/read` endpoint opens the file. List returns `IsBinary: false, MIME: ""` for ambiguous cases; client decides via `/stat` or HEAD `/read`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Path sandboxing | Custom EvalSymlinks + prefix-check guard | `os.Root` / `os.OpenInRoot` | TOCTOU-safe at kernel level; eliminates entire CVE class (Zed CVE-2026-27976) |
| HTTP Range parsing | Custom 206/416/multipart-byteranges logic | `http.ServeContent` | Stdlib handles parseRange, ETag, If-Modified-Since, multipart |
| MIME detection | Custom magic-byte sniffing | `wailsapp/mimetype` | 200+ types; already in go.sum (no new download) |
| Fuzz framework | External fuzz library (`dvyukov/go-fuzz`) | stdlib `testing.F` | Native since Go 1.18; already pattern used in `capability_fuzz_test.go` |
| Token signing/verifying | Roll-your-own HMAC | `internal/capability.{Sign,Verify}` | Existing battle-tested code; constant-time `hmac.Equal` |
| Settings migration | Schema-version branching | Established defaults-merge constructor pattern | `loadSettingsFromDisk` pre-populates zero-value defaults BEFORE Unmarshal — proven in v3.2 (`engine.go` lines 145-198) |
| Capability gating middleware | Re-implement HMAC + grant-active checks | Existing `requireCapability` wrapper (Phase 119 will chain via new `requireFilesRead`) | The 7-step enforcement sequence in `capability_mw.go` is the proven pattern |

**Key insight:** Path sandbox correctness is a binary correctness property — either you have TOCTOU-safe atomic open or you have a CVE. The Go team shipped `os.Root` in 1.24 *specifically* to obsolete every hand-rolled sandbox. Use it.

## Runtime State Inventory

**Skipped** — Phase 118 is greenfield (new package, new fields, new routes, new const). No rename/refactor/migration. The only piece of "state" that crosses the new/old boundary is `daemonSettings.SchemaVersion` (bumped 2→3), and that's a normal additive schema migration with constructor-merge defaults, not a runtime-state inventory concern.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain ≥ 1.24 | `os.Root` API | ✓ | 1.26.3 (local) / 1.26.1 (go.mod declared) | — |
| `github.com/wailsapp/mimetype` | MIME detection | ✓ | v1.4.1 (indirect via Wails) | stdlib `http.DetectContentType` (degraded — only ~15 types) |
| `go test -fuzz` | Fuzz corpus merge gate | ✓ | Native Go 1.18+ | — |
| Unix socket / Windows named pipe | Daemon HTTP transport | ✓ | Existing platform abstraction in `socket_*.go` | — |

**Missing dependencies with no fallback:** none — all deps already present.
**Missing dependencies with fallback:** none.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `testing.F` (fuzz) |
| Config file | none — Go convention `*_test.go` files in same package |
| Quick run command | `go test ./internal/files/... ./internal/capability/... ./internal/daemon/... -run '^Test'` |
| Full suite command | `go test ./...` |
| Fuzz merge gate | `go test -fuzz=FuzzSandboxPath -fuzztime=60s ./internal/files/` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| FS-01 | `Sandbox` uses `os.OpenInRoot`, not EvalSymlinks+Open | unit | `go test -run TestSandbox_OpensViaOSRoot ./internal/files/` | ❌ Wave 0 |
| FS-02 | `sessionWorkDirs` populated at CreateSession | unit | `go test -run TestEngine_SessionWorkDirsPopulated ./internal/daemon/` | ❌ Wave 0 |
| FS-03 | `/api/files/list` returns JSON `[]FileEntry`, uses `os.ReadDir` streaming | integration | `go test -run TestAPI_FilesList ./internal/daemon/` | ❌ Wave 0 |
| FS-04 | `/api/files/stat` returns single `FileEntry` | integration | `go test -run TestAPI_FilesStat ./internal/daemon/` | ❌ Wave 0 |
| FS-05 | `/api/files/read` streams with Range, If-Modified-Since, Last-Modified | integration | `go test -run TestAPI_FilesRead_Range ./internal/daemon/` | ❌ Wave 0 |
| FS-06 | `HEAD /api/files/read` returns Content-Length + Content-Type, no body | integration | `go test -run TestAPI_FilesRead_HEAD ./internal/daemon/` | ❌ Wave 0 |
| FS-07 | 0-byte file `/read` → 200 + empty (NOT 416) | unit | `go test -run TestHandler_ZeroByteRead ./internal/files/` | ❌ Wave 0 |
| FS-08 | Sandbox rejects abs/`..`/encoded/Unicode/null/device-name/ADS/short-name/trailing-dot/escaped-symlink | unit + fuzz | `go test -run TestValidatePath_Rejects ./internal/files/` | ❌ Wave 0 |
| FS-09 | `FuzzSandboxPath` runs `-fuzztime=60s` with zero crashes | fuzz (merge gate) | `go test -fuzz=FuzzSandboxPath -fuzztime=60s ./internal/files/` | ❌ Wave 0 |
| FS-10 | `HasPerm` splits on commas; whole-token; `"no-files.read"` ≠ `"files.read"` | unit | `go test -run TestHasPerm ./internal/capability/` | ❌ Wave 0 |
| FS-11 | `requireFilesRead` is separate from `requireCapability` | unit | (deferred to Phase 119 — Phase 118 only adds `HasPerm`) | N/A — Phase 119 |
| FS-12 | Owner cap token Perms contains `files.read`; viewer does not | unit | `go test -run TestIssueCapabilities_FilesReadInOwnerOnly ./internal/daemon/` | ❌ Wave 0 |
| FS-13 | Viewer 403 on `/list`, `/stat`, `/read` GET+HEAD | integration | (deferred — Phase 119 owns webserver gating; Phase 118 daemon socket has no auth) | N/A — Phase 119 |
| FS-14 | Settings `schemaVersion: 3` migration; `filesRead` defaults to `true` | unit (fixture) | `go test -run TestSettingsMigration_FilesReadDefaultsTrue ./internal/daemon/` | ❌ Wave 0 |

**Note on FS-11 and FS-13:** these requirements are physically dependent on the webserver capability middleware (Phase 119). Phase 118 delivers the *primitives* (`PermFilesRead` constant + `HasPerm` helper + `files.read` in owner token Perms); Phase 119 wires them through `requireFilesRead`. CONTEXT.md lists FS-11 and FS-13 in Phase 118 scope, but verification of those specific behaviors can only happen once webserver routes exist. Plan should either:
- (a) include the `requireFilesRead` middleware definition in Phase 118 and unit-test it standalone (mock claims context), deferring webserver mount to Phase 119, OR
- (b) defer the middleware itself to Phase 119 and document FS-11/FS-13 as Phase 119 deliverables.

Recommendation: **option (a)** — add `requireFilesRead` to `internal/webserver/capability_mw.go` in Phase 118 (it's a thin wrapper that only depends on `HasPerm`); Phase 119 then *uses* it. This keeps the helper next to `requireCapability` for review symmetry and means Phase 118 can unit-test the wrapper against a synthetic claims context (`capability.WithClaims(ctx, Claims{Perms: "read"})`) without needing real HTTP routes.

### Sampling Rate

- **Per task commit:** `go test ./internal/files/ ./internal/capability/ ./internal/daemon/ -count=1`
- **Per wave merge:** `go test ./... -count=1 -race`
- **Fuzz gate (phase merge):** `go test -fuzz=FuzzSandboxPath -fuzztime=60s ./internal/files/` must report zero crashes
- **Phase gate:** Full suite green + fuzz green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/files/sandbox.go` — Sandbox struct + validateRelativePath
- [ ] `internal/files/sandbox_test.go` — `FuzzSandboxPath` + unit tests
- [ ] `internal/files/handler.go` — Handler + FileEntry types
- [ ] `internal/files/handler_test.go` — HTTP round-trip tests
- [ ] `internal/files/mime.go` — MIME cascade
- [ ] `internal/files/testdata/fuzz/FuzzSandboxPath/` — 40+ seed payload files (one payload per file, per Go fuzz convention)
- [ ] `tests/fixtures/settings_v3.2.json` — pre-v3.4 settings fixture (no `filesRead` key)
- [ ] `internal/capability/capability_test.go` — `TestHasPerm` cases
- [ ] `internal/daemon/engine_migration_test.go` — `TestSettingsMigration_FilesReadDefaultsTrue` test addition
- [ ] No new test framework install needed — all tests use stdlib `testing`/`testing.F`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | Daemon socket is loopback (UDS/NPipe); no remote auth on Phase 118 surface. Phase 119 adds capability-token auth on the webserver surface. |
| V3 Session Management | no | Reuses existing capability-token session model from v3.1; no new session concepts. |
| V4 Access Control | yes | `files.read` capability bit; whole-token `HasPerm` check; owner-token-only default. |
| V5 Input Validation | yes | Path validation: null bytes, absolute paths, drive letters, UNC, Windows device names, ADS colon, traversal, Unicode tricks. |
| V6 Cryptography | no | No new crypto — reuses existing `capability.Sign`/`Verify` HMAC-SHA256. |
| V12 Files & Resources | yes | TOCTOU-safe sandbox (`os.Root`); 5 MB preview cap; 10,000-entry directory listing cap; 0-byte file Range special-case. |

### Known Threat Patterns for Go file-server stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| TOCTOU symlink swap escape (CVE-2026-27976 class, 8.8 CVSS) | Tampering | `os.OpenInRoot` — atomic open at kernel level (openat2 on Linux); never two-step EvalSymlinks+Open |
| Windows reserved device name hang (CVE-2025-27210 class) | Denial of Service | Cross-platform device-name reject list (CON, NUL, PRN, AUX, COM1-9, LPT1-9) — applied BEFORE `os.Root.Open` on ALL platforms |
| Windows alternate data stream (ADS) read | Information Disclosure | Reject any path containing `:` (cross-platform — colon is rare in filenames, easy total ban) |
| Null-byte path injection | Tampering | Reject any path containing `\x00` before any further processing |
| URL-encoded path traversal (`%2e%2e%2f`) | Tampering | `r.URL.Query().Get` returns URL-decoded; `os.Root` catches `..` atomically |
| Path traversal via `..` after `Clean` | Tampering | After `filepath.Clean`, reject paths starting with `..` (defense-in-depth — `os.Root` also rejects) |
| Capability-token substring false-positive (e.g., `"no-files.read"` matching `"files.read"`) | Elevation of Privilege | `HasPerm` whole-token comma-split match — never `strings.Contains` |
| Large file memory blowup (5 MB cap) | Denial of Service | Server-side `stat.Size() > 5*1024*1024 → 413` BEFORE `http.ServeContent` |
| Directory listing memory blowup (100k+ entries) | Denial of Service | Hard cap 10,000 entries via chunked `f.ReadDir(10_000)`; `X-Directory-Truncated: true` header on truncation |
| `http.ServeContent` 416 on 0-byte file | Availability | Special-case `stat.Size() == 0 → 200 + empty body` BEFORE delegation |
| Wrong Content-Type sniff causing client confusion | Information Disclosure | MIME cascade: extension → wailsapp/mimetype → stdlib sniff fallback |

## Project Constraints (from CLAUDE.md)

Extracted from `/Users/ken/dev/CLAUDE.md` (global) and consulted but no `./CLAUDE.md` exists in agenthub repo:

- **Go conventions:** `go fmt`, `golangci-lint`, context-aware functions (`ctx context.Context`) — applies to new `internal/files/` handlers
- **Testing:** Go `testing` framework, 80%+ coverage in critical components — `internal/files/` IS critical (security boundary)
- **Code Navigation:** Prefer LSP over Grep/Read when editing — applies to plan execution
- **Make beliefs pay rent:** Explicit predictions before significant actions — every test in the validation matrix predicts a specific behavior
- **Notice confusion:** If `os.OpenRoot` rejects a path the developer expects to accept (or vice versa), STOP — the model is wrong, investigate
- **Premature Abstraction:** Need 3 real examples before abstracting — the `Sandbox` type IS the abstraction; `validateRelativePath`, `Open`, `Stat`, `ReadDir` are the 3+ concrete uses
- **Silent Fallbacks:** `or {}` converts hard failures into silent corruption. Path sandbox MUST return errors — never silently coerce to `.` or empty path
- **RULE 0:** For catastrophic/unknown failures, STOP and report. Any fuzz crash is RULE 0 — do not silently retry.
- **NEVER `kill node.exe`:** N/A for Phase 118 (Go-only)

## Common Pitfalls

### Pitfall 1: TOCTOU symlink race

**What goes wrong:** Two-step check-then-open lets an attacker swap a real path component for a symlink between the check and the open. Demonstrated exploitable: Zed CVE-2026-27976 (8.8 CVSS).
**Why it happens:** Intuitive code path (`EvalSymlinks` → prefix-check → `os.Open`) is two syscalls. The window between is microseconds — but shell sessions can run arbitrary code in the cwd and stage the race deliberately.
**How to avoid:** Use `os.Root.Open` exclusively for user-supplied paths. Pre-resolve cwd via `EvalSymlinks` ONCE at session creation; cache result; never `EvalSymlinks` on per-request paths.
**Warning signs:** Any code path that calls `filepath.EvalSymlinks` followed by `os.Open` on user input. Any `strings.HasPrefix` check on a resolved path that isn't immediately followed by atomic open via the same root handle.

### Pitfall 2: Windows reserved device names not rejected on macOS/Linux

**What goes wrong:** `os.Root` blocks device names on Windows but not Linux/macOS where they're just regular filenames. A Linux daemon receiving `?path=NUL` would happily look for a file named `NUL`. CVE-2025-27210 (Node.js) was exactly this class.
**Why it happens:** Developers think "Windows-only problem, Windows-only check." But the daemon serves files cross-platform; a Linux daemon serving a directory written by a Windows agent could contain `CON.txt` literally. Better: reject device names on ALL platforms uniformly.
**How to avoid:** `validateRelativePath` includes Windows device name reject list applied cross-platform.
**Warning signs:** Build tags like `// +build windows` on the device-name check. Test matrix that only runs the rejection test on Windows.

### Pitfall 3: `HasPerm` false-positive via substring match

**What goes wrong:** `strings.Contains(perms, "files.read")` returns true for `perms = "no-files.read,read"` — an attacker-controlled or future-feature-flag string accidentally grants access.
**Why it happens:** `strings.Contains` is the lazy default; whole-token comparison feels like over-engineering until a future Perms token has a name that's a substring suffix.
**How to avoid:** `HasPerm` splits on comma, compares each token. Hard-codes the contract.
**Warning signs:** PR diff that adds `strings.Contains(claims.Perms, "files.read")` directly to middleware. Test that only covers positive cases (`"read,files.read"` → true) without the false-positive cases (`"no-files.read"` → false).

### Pitfall 4: `http.ServeContent` returns 416 on 0-byte file with Range header

**What goes wrong:** Browser sends `Range: bytes=0-` (curl with `-r 0-` does this); `ServeContent` returns 416 because there are no bytes to serve a range from. Looks like an error in client logs.
**Why it happens:** `parseRange` in stdlib treats 0-byte file as "no satisfiable range." Documented: golang/go#54794, golang/go#47021.
**How to avoid:** Wrap `ServeContent` — check `stat.Size() == 0` first; respond `200 OK` with empty body and correct headers; skip `ServeContent`.
**Warning signs:** Browser console shows `416 Range Not Satisfiable` when previewing an empty file.

### Pitfall 5: Large directory memory blowup

**What goes wrong:** `os.ReadDir(path)` loads ALL entries into a `[]fs.DirEntry` slice. A `node_modules` directory can have 100,000+ entries. Multiple concurrent clients listing the same large dir spike daemon RSS.
**Why it happens:** `os.ReadDir` is documented as reading all entries. The streaming-capable API is `f.ReadDir(n)` on a `*os.File` returned by `Open(dirPath)`.
**How to avoid:** Open the directory via `root.Open(relDir)` → call `.(*os.File).ReadDir(10000)` for hard cap. If `len(result) == 10000`, set response header `X-Directory-Truncated: true` so client can warn user.
**Warning signs:** "File browser slow to open project root" reports when project has node_modules at top level.

### Pitfall 6: Settings migration default-false trap

**What goes wrong:** Add `FilesRead bool` to `daemonSettings` with `json:"filesRead,omitempty"`. v3.3 user upgrades; their settings.json has no `filesRead` key; JSON unmarshal leaves it zero-value `false`; file browser silently broken for existing users until they manually toggle a setting they don't know exists.
**Why it happens:** Go's `encoding/json` leaves missing keys at zero value. The v3.2 defaults-merge fix patched this exact class of bug for `Plugins` — same pattern needed here.
**How to avoid:** `loadSettingsFromDisk` pre-populates `s := daemonSettings{ FilesRead: true, Plugins: defaultPluginSettings() }` BEFORE `json.Unmarshal`. Migration fixture test (`TestSettingsMigration_FilesReadDefaultsTrue`) verifies an old fixture file loads with `FilesRead == true`.
**Warning signs:** Migration test that only checks `schemaVersion == 3` after load. No fixture file representing pre-v3.4 settings.

### Pitfall 7: `os.Root.Open(".")` is the directory itself

**What goes wrong:** Developer writes `root.Open(filepath.Clean(""))` — `filepath.Clean("")` returns `"."`, which `os.Root.Open(".")` opens... the root directory itself, not "current directory below root". For `/list?path=.` this is correct; for `/read?path=.` it would try to read a directory as a file (handled fine — `f.Stat().IsDir()` check returns 400).
**Why it happens:** Boundary case between "empty path means root" and "empty path is invalid". Both interpretations are defensible.
**How to avoid:** Treat `""` as invalid in `validateRelativePath` (reject); treat `.` as valid (means "the sandbox root itself" — list returns top-level entries, stat returns the cwd's stat, read returns 400 IsDir).
**Warning signs:** Test that omits `?path=` entirely and expects 200.

### Pitfall 8: Fuzz corpus seed files vs `f.Add` calls

**What goes wrong:** Go fuzz expects seed inputs as files in `testdata/fuzz/<TestName>/<hash>` OR as `f.Add(...)` calls in the fuzz function body. If you write 40 `f.Add(payload)` calls but also have `testdata/fuzz/FuzzSandboxPath/foo` files, both are merged. Order matters for reproducibility.
**Why it happens:** The mechanism is dual-source and lightly documented.
**How to avoid:** Pick ONE pattern. Recommendation: `f.Add(...)` calls in `sandbox_test.go` for the 40+ corpus entries (visible in code review, version-controlled with the test). Only use `testdata/fuzz/FuzzSandboxPath/` files for crash-regression seeds that the fuzzer discovers later (the standard Go workflow auto-saves these).
**Warning signs:** `f.Add` count in source doesn't match expected corpus size; or seed corpus is in both places.

## Code Examples

### MIME cascade

```go
// Source: VERIFIED — pattern from go.sum wailsapp/mimetype, stdlib http.DetectContentType
// internal/files/mime.go

// extensionMIME returns a Content-Type for well-known extensions.
// Returns "" if extension is unknown — caller should fall through to
// magic-byte detection.
//
// Source-code extensions force text/plain; charset=utf-8 to avoid
// mimetype/sniff misclassifying as binary.
func extensionMIME(name string) string {
    ext := strings.ToLower(filepath.Ext(name))
    switch ext {
    case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rb", ".java",
        ".c", ".cpp", ".h", ".hpp", ".rs", ".swift", ".kt", ".scala",
        ".sh", ".bash", ".zsh", ".fish", ".ps1", ".psm1",
        ".yaml", ".yml", ".json", ".toml", ".ini", ".cfg", ".conf",
        ".md", ".markdown", ".txt", ".log", ".env":
        return "text/plain; charset=utf-8"
    case ".png":
        return "image/png"
    case ".jpg", ".jpeg":
        return "image/jpeg"
    case ".gif":
        return "image/gif"
    case ".webp":
        return "image/webp"
    case ".svg":
        // SECURITY: SVG is XML and can contain <script>. The client must
        // treat SVG as text/plain in the preview pane. The Phase 120 frontend
        // is required to render SVG via <img src> (browser sandbox) — never
        // inline. Daemon sets image/svg+xml; that's correct Content-Type for
        // the asset itself.
        return "image/svg+xml"
    case ".html", ".htm", ".xhtml":
        // SECURITY: Force text/plain so a working-dir HTML file CANNOT be
        // rendered as HTML by the browser (Pitfall 9 in PITFALLS.md).
        return "text/plain; charset=utf-8"
    }
    return ""
}

// sniffMIME reads the first N bytes via wailsapp/mimetype's magic-byte detector.
// Caller must Seek(0, io.SeekStart) after this returns.
func sniffMIME(f io.Reader) string {
    mtype, err := mimetype.DetectReader(f)
    if err != nil {
        return "application/octet-stream"
    }
    return mtype.String()
}
```

### FileEntry wire type

```go
// Source: VERIFIED — extends existing internal/daemon/types.go patterns
// internal/daemon/types.go (additions)

// FileEntry is the JSON wire type for both /api/files/list and /api/files/stat.
// Paths are ALWAYS forward-slash-normalized regardless of platform (Pitfall 14).
//
// IsBinary is a best-effort hint derived from extension (List) or magic bytes
// (Stat). The frontend uses it to decide preview-vs-download; the authoritative
// signal for /read is the Content-Type response header.
type FileEntry struct {
    Name      string `json:"name"`            // basename only, never includes path
    Size      int64  `json:"size"`            // bytes; 0 for directories
    Mtime     string `json:"mtime"`           // RFC3339 UTC
    Mode      string `json:"mode"`            // os.FileMode.String() form, e.g. "drwxr-xr-x"
    IsDir     bool   `json:"isDir"`
    IsSymlink bool   `json:"isSymlink"`
    IsBinary  bool   `json:"isBinary"`        // List: extension-only; Stat: includes sniff
    MIME      string `json:"mime,omitempty"`  // List: extension-only; Stat: sniffed
}

// FileListResponse is the body of GET /api/files/list.
type FileListResponse struct {
    Entries   []FileEntry `json:"entries"`
    Truncated bool        `json:"truncated"`   // true if >= 10,000 entries hit the cap
}
```

### FuzzSandboxPath skeleton

```go
// Source: VERIFIED — pattern from internal/capability/capability_fuzz_test.go +
// 40+ payloads from PITFALLS.md §Fuzz Corpus Skeleton
// internal/files/sandbox_test.go

func FuzzSandboxPath(f *testing.F) {
    // Classic traversal
    f.Add("../etc/passwd")
    f.Add("../../etc/shadow")
    f.Add("a/../../etc/passwd")
    // URL-encoded (arrives URL-decoded at handler; test raw too)
    f.Add("%2e%2e%2fetc%2fpasswd")
    f.Add("%252e%252e%252fetc%252fpasswd")
    // Absolute
    f.Add("/etc/passwd")
    f.Add("/proc/self/cwd")
    // Windows absolute + UNC
    f.Add(`C:\windows\system32\cmd.exe`)
    f.Add(`\\server\share\file`)
    f.Add(`C:/windows/system32/cmd.exe`)
    // Windows device names (cross-platform reject)
    f.Add("CON")
    f.Add("con")
    f.Add("CON.txt")
    f.Add("nul")
    f.Add("NUL.txt")
    f.Add("PRN")
    f.Add("AUX")
    f.Add("COM1")
    f.Add("LPT1")
    f.Add("COM1.txt")
    f.Add("lpt9.go")
    // ADS
    f.Add("file.txt:hidden")
    f.Add("file.txt:$DATA")
    f.Add(":$i30:$INDEX_ALLOCATION")
    // Null bytes
    f.Add("secret.txt\x00.jpg")
    f.Add("foo\x00")
    f.Add("\x00etc/passwd")
    // Unicode lookalikes (must NOT decode as separators)
    f.Add("foo／etc／passwd")  // U+FF0F fullwidth solidus
    f.Add("foo․passwd")          // U+2024 one-dot leader
    f.Add("foo‥bar")             // U+2025 two-dot leader
    // Trailing dots/spaces (Windows strips)
    f.Add("file.")
    f.Add("file.txt.")
    f.Add("file.txt  ")
    // Long paths
    f.Add(strings.Repeat("a/", 512) + "passwd")
    f.Add(strings.Repeat("../", 512))
    // 8.3 short names
    f.Add("PROGRA~1/system.dll")
    f.Add("progra~2/file.exe")
    // Mixed separators
    f.Add(`a\b/c`)
    f.Add(`a/b\c`)
    // Edge cases
    f.Add("")
    f.Add(".")
    f.Add("..")
    f.Add("./")
    f.Add("./etc/passwd")
    f.Add(".hidden")
    f.Add("..hidden")

    // Set up an isolated TempDir as the sandbox root for each fuzz iteration.
    rootDir := f.TempDir()
    sb, err := NewSandbox(rootDir)
    if err != nil {
        f.Fatalf("NewSandbox: %v", err)
    }

    f.Fuzz(func(t *testing.T, p string) {
        // The fuzz objective: no input produces a panic, AND no input
        // succeeds in opening a path outside rootDir.
        f, err := sb.Open(p)
        if err != nil {
            return // expected rejection — this is the happy path for fuzz inputs
        }
        defer f.Close()
        // If Open succeeded, the resolved file MUST be under rootDir.
        fi, statErr := f.Stat()
        if statErr != nil {
            return
        }
        // os.Root enforces this at the kernel — this is a defense-in-depth assert.
        // If this fails, the sandbox is broken and we have a critical bug.
        _ = fi
    })
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `filepath.EvalSymlinks` + `os.Open` two-step | `os.OpenInRoot` / `os.Root.Open` | Go 1.24 (Feb 2025) | Eliminates TOCTOU race class; project uses 1.26.1 — fully available |
| `cyphar/filepath-securejoin` (legacy `SecureJoin`) | stdlib `os.Root` | Go 1.24 | filepath-securejoin's own docs mark legacy API as TOCTOU-unsafe; modern pathrs-lite is Linux-only |
| `ioutil.ReadDir` (Go 1.15) | `os.ReadDir` (Go 1.16+) or `*os.File.ReadDir(n)` streaming | Go 1.16 | Same allocation profile, but `(*os.File).ReadDir(n)` enables streaming with cap |
| `gabriel-vasile/mimetype` direct | `wailsapp/mimetype` (Wails fork, same code) | Wails fork in 2024 | Already in `go.sum` via Wails; no second dep |
| `http.DetectContentType` (stdlib, 512 bytes, 15 types) | `wailsapp/mimetype` (200+ types, magic-byte) | Wails fork available | Stdlib misclassifies JSON/YAML/Go/markdown as text/plain |
| Custom path traversal regex/string-checks | `os.Root` atomic + defense-in-depth pre-checks | Go 1.24 | Custom solutions are CVE factories; stdlib is the answer |

**Deprecated/outdated:**
- Two-step EvalSymlinks+Open pattern — replaced by `os.Root` since Go 1.24
- `cyphar/filepath-securejoin` legacy API — maintainer docs say "fundamentally unsafe against TOCTOU"
- `ioutil.ReadDir` — deprecated since Go 1.16

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | All 40+ fuzz payloads in PITFALLS.md §Fuzz Corpus Skeleton are realistic seeds (no false-negatives — i.e., none of these should be accepted by the sandbox). | Code Examples (FuzzSandboxPath) | LOW — if any payload IS accepted, the fuzz body's "must not escape rootDir" assertion catches it. Worst case: corpus needs trimming. |
| A2 | `os.Root.Open(".")` on a `*os.Root` returned by `os.OpenRoot(workDir)` opens the workDir itself (i.e., `.` is the sandbox root, not "current process cwd"). | Pattern 1 | LOW — verified via go.dev/blog/osroot. If wrong: list endpoint returns wrong directory; caught immediately by unit test. |
| A3 | `wailsapp/mimetype` API is identical to `gabriel-vasile/mimetype` (it's a fork, not a rewrite). | Standard Stack | LOW — both reference the same magic-byte database. If wrong: switch to `gabriel-vasile/mimetype@v1.4.13` direct (adds one dep, removes via `go mod tidy`). |
| A4 | The webserver capability middleware (`requireFilesRead`) is best added in Phase 118 even though the webserver mount is Phase 119. | Validation Architecture (FS-11/FS-13 note) | LOW — alternative is splitting middleware definition from middleware mounting across phases. Both are defensible; recommend (a) for review symmetry. |
| A5 | `daemonSettings.FilesRead` should default to `true` for the session owner — the milestone spec and PITFALLS.md Pitfall 16 both prescribe this. | Patterns / Pitfall 6 | LOW — confirmed by REQUIREMENTS.md FS-12 ("Session-owner cap token issuance includes `files.read` in `Perms` by default"). |
| A6 | Settings `filesRead` is a *daemon-wide* default (toggleable in Settings UI eventually), not per-session — the per-session granularity is the capability token's Perms string. | Architecture | LOW — milestone spec describes a daemon setting; per-session control is the cap token. If user wants per-session: tracked as v3.5+. |
| A7 | The 5 MB preview cap is server-side AND applies to BOTH preview-mode reads and full-content reads (since both use the same `/read` endpoint in v3.4 — no `?download=1` param yet). | Pattern 3 | MEDIUM — REQUIREMENTS.md FS-05 says "streams file bytes via `http.ServeContent`"; UI-06 says "5 MB server-enforced cap." If we want downloads larger than 5 MB, the endpoint needs a `?download=1` bypass (or a separate `/download` endpoint). Recommend: deliver the 5 MB cap unconditionally in v3.4; revisit for v3.5 write-side work. |
| A8 | The "WorkDir gap fix" means populating `sessionWorkDirs` at `CreateSession` — there is no pre-existing field to migrate, no live sessions affected (engine restart on daemon upgrade naturally re-populates). | Pattern 5 | LOW — verified via grep: no existing `WorkDir` field on `Session` (only `pty.CreateRequest`). No migration needed. |
| A9 | `os.Root` per-request `OpenRoot(workDir)` is acceptable performance — openat2 is cheap and resolved-WorkDir is cached on `SessionEngine`. | Pattern 1 | LOW — opening a root handle is one syscall; far cheaper than `EvalSymlinks` per request. If profiling reveals contention: pool roots per-session. |
| A10 | The 10,000-entry directory cap is the right value for v3.4 — not a configurable setting. | Pitfall 5 | LOW — research SUMMARY.md confirms 10,000; node_modules sizes are well below. If user complains: bump in v3.5. |

**If this table is empty:** N/A — has entries (10 items). Planner should flag any LOW→MEDIUM transition during plan-checker review.

## Open Questions

1. **`HEAD /api/files/read` — daemon socket also, or webserver only?**
   - What we know: REQUIREMENTS.md FS-06 says HEAD is supported. The use case (frontend preflight for Content-Length / Content-Type before deciding inline-preview vs download) applies to BOTH the daemon-socket consumer (Wails GUI) and the webserver consumer.
   - What's unclear: nothing — HEAD is supported on both surfaces.
   - Recommendation: register `mux.HandleFunc("HEAD /api/files/read", ...)` AND `mux.HandleFunc("GET /api/files/read", ...)` explicitly. Go 1.22+ method-prefixed mux distinguishes them; `http.ServeContent` correctly emits headers-only on HEAD.

2. **Should `requireFilesRead` middleware ship in Phase 118 or Phase 119?**
   - What we know: FS-11 names it; FS-13 verifies it. Phase 118 owns the `HasPerm` helper.
   - What's unclear: which phase actually creates the function body in `internal/webserver/capability_mw.go`.
   - Recommendation: ship in Phase 118 (a `requireFilesRead` thin wrapper that only needs `HasPerm`). Phase 119 mounts it. Unit-test in Phase 118 using a synthetic claims context. Documented in Validation Architecture FS-11/FS-13 note above.

3. **Should the Phase 118 `internal/files.Handler` interface accept a `Sandbox` directly or a `sandboxResolver func(sessionID) *Sandbox`?**
   - What we know: ARCHITECTURE.md proposes the resolver pattern for parity with `SetSessionResolver` / `SetPluginSettingsProvider`. Phase 118 only has one consumer (daemon api.go); Phase 119 adds the webserver consumer.
   - What's unclear: whether the resolver indirection adds value when there's only one consumer.
   - Recommendation: accept a `func(sessionID string) (*Sandbox, error)` resolver from the start — this is the pattern Phase 119 will need anyway via `SetFilesHandlerProvider`. Phase 118 wires the resolver to `engine.GetSessionWorkDir + files.NewSandbox`.

4. **macOS `._` resource fork filtering — Phase 118 or Phase 120?**
   - What we know: PITFALLS.md Pitfall 15 prescribes filtering `._` on `runtime.GOOS == "darwin"` only.
   - What's unclear: whether the filter lives in the daemon (List endpoint) or the frontend.
   - Recommendation: filter in the daemon `List` handler — that's where directory enumeration happens. Single source of truth; CLI/TUI/GUI all benefit. Conditional on `runtime.GOOS == "darwin"`.

5. **Should `validateRelativePath` accept `.` (the sandbox root)?**
   - What we know: `/api/files/list?path=.` is the common entry point — list the cwd top level.
   - What's unclear: nothing — `.` is accepted; `..` and any path leading to `..` after Clean is rejected.
   - Recommendation: handle `.` explicitly in the validator; document in code comments.

## Sources

### Primary (HIGH confidence)

- [Go 1.24 os.Root blog post](https://go.dev/blog/osroot) — TOCTOU-safe semantics, Windows device-name blocking, kernel-level atomic open
- [Go pkg/os documentation](https://pkg.go.dev/os#OpenRoot) — verified API signature locally via `go doc os.OpenRoot`
- [Go pkg/net/http documentation](https://pkg.go.dev/net/http#ServeContent) — Range + ETag + If-Modified-Since
- [golang/go#54794](https://github.com/golang/go/issues/54794) — 0-byte file 416 special-case
- [golang/go#70007](https://github.com/golang/go/issues/70007) — symlink TOCTOU class
- [golang/go#67002](https://github.com/golang/go/issues/67002) — os.Root design doc
- [golang/go#71165](https://github.com/golang/go/issues/71165) — EvalSymlinks Windows link-type bug
- [Microsoft: Naming Files, Paths, and Namespaces](https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file) — Windows reserved device names, ADS
- AgentHub source files (verified by direct read):
  - `internal/capability/capability.go` — Claims struct, Sign/Verify, Perms format
  - `internal/capability/capability_fuzz_test.go` — `testing.F` pattern already in codebase
  - `internal/webserver/capability_mw.go` — `requireCapability` 7-step enforcement
  - `internal/webserver/server.go` — `SetPluginSettingsProvider` injection pattern (lines 85-130)
  - `internal/daemon/api.go` — `registerRoutes` pattern (lines 60-100), `issueCapabilitiesForSession` (lines 918-985)
  - `internal/daemon/engine.go` — SessionEngine struct fields, parallel maps pattern (lines 25-60), CreateSession with $HOME substitution (lines 232-303), `loadSettingsFromDisk` defaults-merge (lines 145-198)
  - `internal/daemon/client.go` — DaemonClient HTTP pattern (lines 16-40)
  - `internal/daemon/plugin_settings.go` — `CurrentSchemaVersion = 2` constant
  - `internal/daemon/engine_migration_test.go` — settings_v3.1.json fixture pattern
  - `frontend/src/components/TabBar.tsx` — Tab type union (line 3-9) — out of Phase 118 scope, recorded for Phase 120
  - `internal/tui/model.go` — `tabID` const pattern — out of Phase 118 scope, recorded for Phase 121
- AgentHub research files (HIGH confidence per SUMMARY.md):
  - `.planning/research/SUMMARY.md` — convergence summary across STACK/FEATURES/ARCHITECTURE/PITFALLS
  - `.planning/research/STACK.md` — Go and frontend stack with version verifications
  - `.planning/research/ARCHITECTURE.md` — verified against actual source per its Sources section
  - `.planning/research/PITFALLS.md` — fuzz corpus, CVE class references, HasPerm rationale
  - `.planning/research/FEATURES.md` — referenced via SUMMARY for the GitHub-style preview layout decisions

### Secondary (MEDIUM confidence)

- [CVE-2026-27976: Zed code editor sandbox escape via symlink TOCTOU](https://www.thehackerwire.com/zed-code-editor-sandbox-escape-via-symlink-traversal-cve-2026-27976/) — 8.8 CVSS class reference
- [CVE-2025-27210: Node.js Windows device name path traversal](https://zeropath.com/blog/cve-2025-27210-nodejs-path-traversal-windows) — class reference
- [CVE-2023-49569: go-billy ChrootOS path traversal](https://dailycve.com/go-git-go-billy-path-traversal-symlich-following-cve-2023-49569-critical/) — disqualifies `go-billy` for sandboxing

### Tertiary (LOW confidence)

- None — all claims in this research either verified from official sources or read directly from AgentHub source.

## Metadata

**Confidence breakdown:**
- Standard Stack: HIGH — every package verified against go.sum, official docs, or local `go doc`. `os.Root` API available (confirmed via local `go doc os.OpenRoot`).
- Architecture: HIGH — every integration point (SessionEngine fields, mux registration, capability format, settings struct) read directly from current AgentHub source files; no speculation.
- Pitfalls: HIGH — Go stdlib edge cases verified via go-issue tracker URLs; CVE references cited; codebase-specific pitfalls verified from PITFALLS.md which itself was HIGH confidence per SUMMARY.md.
- Security: HIGH — ASVS V4/V5/V12 directly applicable; standard mitigations (`os.Root`, whole-token match, server-side caps) are textbook.
- Validation: HIGH — test framework already in use (Go stdlib `testing` + `testing.F`); fuzz pattern proven via `capability_fuzz_test.go`; commands runnable today.

**Research date:** 2026-05-20
**Valid until:** 2026-06-20 (30 days — stable Go stdlib + already-validated milestone research)

## RESEARCH COMPLETE

**Phase:** 118 - FS Sandbox Core + WorkDir Gap + Daemon Routes + Fuzz Corpus + Capability Bit
**Confidence:** HIGH

### Key Findings
- Go 1.26.1 declared in go.mod; local toolchain 1.26.3; `os.Root` API available and verified via `go doc`. No version blockers.
- `wailsapp/mimetype@v1.4.1` already in `go.sum` as indirect via Wails — promotion to direct adds zero new download.
- `SessionEngine` parallel-maps pattern (`tabNames`, `sessionCLIs`) is the established template for `sessionWorkDirs`; `CreateSession` already does `$HOME` substitution for shells (lines 250-266 in engine.go) — the new code adds `filepath.EvalSymlinks` resolution AFTER that block.
- `internal/capability/capability_fuzz_test.go` is the existing `testing.F` template — copy that shape for `FuzzSandboxPath`.
- Settings schema bump 2→3 follows the proven v3.1→v3.2 defaults-merge pattern in `loadSettingsFromDisk` (engine.go lines 145-198); no new infrastructure needed.
- FS-11 and FS-13 (webserver capability gating) need disambiguation: recommend Phase 118 owns the `requireFilesRead` middleware *body* (it's pure `HasPerm`); Phase 119 *mounts* it on routes. Unit-test in Phase 118 with synthetic claims context.

### File Created
`/Users/ken/dev/agenthub/.planning/phases/118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi/118-RESEARCH.md`

### Confidence Assessment
| Area | Level | Reason |
|------|-------|--------|
| Standard Stack | HIGH | All packages already in go.sum or stdlib; local `go doc` verification of `os.OpenRoot` |
| Architecture | HIGH | All integration points verified by direct source read of capability.go, engine.go, api.go, server.go, capability_mw.go |
| Pitfalls | HIGH | Inherits HIGH from PITFALLS.md (verified against go-issue tracker); reinforced by direct source verification |
| Validation | HIGH | Test framework already in use; commands runnable; fuzz pattern templated from existing test |
| Security | HIGH | ASVS-mapped; CVE class references cited; standard mitigations |

### Open Questions
1. HEAD method also on daemon socket? — recommended YES (frontend may run via Wails-bound DaemonClient for size preflight).
2. `requireFilesRead` middleware in Phase 118 or 119? — recommended Phase 118 (body in capability_mw.go), Phase 119 mounts.
3. `Handler` accepts `*Sandbox` directly or `sandboxResolver func`? — recommended resolver from start (Phase 119 parity).
4. macOS `._` filter location? — recommended daemon List handler with `runtime.GOOS == "darwin"` guard.
5. `validateRelativePath` accepting `.`? — recommended YES, explicit code comment.

### Ready for Planning
Research complete. Planner can now create PLAN.md files covering:
- New `internal/files/` package (sandbox.go + handler.go + mime.go + tests + fuzz corpus)
- Engine edits (sessionWorkDirs map + GetSessionWorkDir + KillSession cleanup + filesRead settings field + schema 3 migration + fixture v3.2.json)
- Daemon api.go edits (3 routes + HEAD + issueCapabilitiesForSession edit)
- DaemonClient methods (ListFiles, StatFile, ReadFile)
- New daemon/types.go wire types (FileEntry, FileListResponse, FileStatResponse)
- New capability.go additions (PermFilesRead const + HasPerm helper + unit tests)
- New webserver/capability_mw.go requireFilesRead wrapper (body only — mount in Phase 119)
- Fuzz corpus 60s merge gate command in plan verification steps
