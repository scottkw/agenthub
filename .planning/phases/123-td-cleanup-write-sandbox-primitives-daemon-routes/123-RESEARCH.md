# Phase 123: TD Cleanup + Write Sandbox Primitives + Daemon Routes - Research

**Researched:** 2026-06-14
**Domain:** Go filesystem write sandbox (`os.Root`), atomic durable writes, daemon Unix-socket HTTP routes, Go fuzz harness, TD cleanup (303 redirect parsing + file-browser hardening)
**Confidence:** HIGH (all load-bearing claims verified against the live codebase and the installed Go 1.26.4 stdlib; the one milestone-research error — `os.Root.Rename` "does not exist" — was empirically disproven and corrected below)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
None — discuss phase was skipped (`workflow.skip_discuss`). The phase boundary, requirements (FSW-01..FSW-12), and success criteria from the ROADMAP are the binding contract.

### Claude's Discretion
All implementation choices are at Claude's discretion. Use the ROADMAP phase goal, success criteria, and codebase conventions to guide decisions. (CONTEXT.md §Implementation Decisions)

### Deferred Ideas (OUT OF SCOPE)
None declared in CONTEXT. From the milestone scope, the following are explicitly NOT in Phase 123 (they land in Phases 124-128): `requireFilesWrite` middleware, the `PermFilesWrite` capability bit wiring into the webserver, CSRF/Origin checks, the React CodeMirror editor, TUI `$EDITOR` shell-out, web-share opt-in UI, and remote-write proxy body forwarding. Phase 123 is the **sandbox + daemon-socket + TD-cleanup** layer only.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FSW-01 | `Sandbox.WriteFileAtomic(relPath, content)` — sibling temp file inside root, `f.Sync()`, atomic rename; no `O_TRUNC` in-place; temp stays in sandbox | §Standard Stack (os.Root write API), §Pattern 1 (atomic write), §Pitfall 5 |
| FSW-02 | `Sandbox.Rename(oldRel, newRel)` — validate BOTH paths via `validateAndClean`; supports same-dir rename AND cross-dir move | §Pattern 2, §Pitfall 2; **CORRECTION: `os.Root.Rename` DOES exist in Go 1.26.4** — see §State of the Art |
| FSW-03 | `Sandbox.Mkdir` / `Sandbox.MkdirAll` within sandbox; traversal rejected | §Standard Stack (os.Root.Mkdir/MkdirAll exist natively) |
| FSW-04 | `Sandbox.Delete(relPath)` — file or recursive subtree, guaranteed within root | §Standard Stack (os.Root.Remove/RemoveAll exist natively) |
| FSW-05 | Upload write path streams multipart part via `WriteFileAtomic`; filename sanitized `filepath.Base` then `validateAndClean` | §Pattern 3 (upload), §Pitfall 6 |
| FSW-06 | Shell-RC denylist enforced inside ALL write methods; `403 Protected system file` | §Pattern 4 (denylist), §Pitfall 4 |
| FSW-07 | `FuzzSandboxWrite` extends FuzzSandboxPath corpus; 60s, 0 crashes = merge gate | §Code Examples (fuzz harness), §Validation Architecture |
| FSW-08 | Daemon socket write routes (PUT write, POST upload, DELETE delete, POST rename, POST mkdir), auth-less (WEB-01 loopback trust) | §Pattern 5 (daemon routes), §Architecture |
| FSW-09 | `DaemonClient` gains `WriteFile/UploadFile/DeleteFile/RenameFile/MkdirFile` | §Code Examples (client methods) |
| FSW-10 | **TD-5:** fix `ExchangeJoinCodeAtURL` 303-shim — parse `Location ?cap=<token>` | §Pattern 6 (TD-5 fix), reference impl verified in `joincode_prompt.go` |
| FSW-11 | **TD-4:** Phase 120 WR-01..05 hardening | §TD-4 Inventory (exact line refs verified) |
| FSW-12 | 50 MiB upload cap via `http.MaxBytesReader` BEFORE `ParseMultipartForm` | §Pattern 3, §Pitfall 6 |
</phase_requirements>

## Summary

Phase 123 extends the v3.4 read-only `internal/files.Sandbox` (a stateless wrapper that opens a fresh `os.OpenRoot` per operation after running `validateAndClean`) with five write primitives, enforces a shell-RC denylist inside every write method, registers five auth-less write routes on the daemon Unix socket, adds five `DaemonClient` write methods, extends the fuzz corpus, and closes two carried tech-debts (TD-4 file-browser hardening, TD-5 join-code 303 parsing). It is the load-bearing security foundation for the entire v3.5 write epic — no capability model changes, no webserver/CSRF/editor work (those are Phases 124-128).

The most important finding overrides the milestone research: **`*os.Root` in the installed Go 1.26.4 has native, relative-to-root, TOCTOU-safe `Rename`, `MkdirAll`, `WriteFile`, `Remove`, and `RemoveAll` methods.** The milestone `PITFALLS.md` (and FSW-02's reference to golang/go#69462) assert that `os.Root.Rename` does not exist and must be hand-rolled with `os.Rename` on constructed absolute paths — this is **stale and wrong for this Go version** (I enumerated the actual `*os.Root` method set and ran `go doc os.Root.Rename`). The phase should use the native `root.Rename` / `root.MkdirAll` / `root.RemoveAll` methods directly, which is *strictly safer* than the hand-rolled `os.Rename`-on-absolute-paths workaround the pitfalls doc recommends (the workaround re-introduces a TOCTOU window the native method closes). `validateAndClean` on both paths is still required as defense-in-depth and for clear error messages, but the terminal security boundary is `root.Rename` itself.

**Primary recommendation:** Mirror the existing v3.4 `Sandbox` per-operation `os.OpenRoot` pattern exactly. Add write methods that (1) `validateAndClean` the path(s), (2) run the denylist check, (3) open a fresh `os.OpenRoot(s.rootPath)`, (4) call the native `root.*` write method. Add write handlers in a new `internal/files/write.go` sibling to `handler.go`. Register daemon routes with explicit method prefixes (Go 1.22+ mux returns 405 automatically for wrong verbs). Use native `root.Rename` for atomic temp→target. Fix TD-5 by copying the already-correct `joincode_prompt.go` 303-parsing logic into `ExchangeJoinCodeAtURL`. Address TD-4 WR-01..05 at the verified Phase 120 source locations.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Path validation / sandbox confinement | `internal/files` (Sandbox) | Go stdlib `os.Root` (kernel TOCTOU boundary) | The sandbox is the single security authority; `os.Root` is the syscall-level enforcement |
| Atomic durable write | `internal/files` (Sandbox.WriteFileAtomic) | — | Temp+sync+rename is a filesystem concern, owned by the sandbox |
| Shell-RC denylist | `internal/files` (Sandbox write methods) | — | Server-side, defense-in-depth; must be in the sandbox layer not the handler (so all callers get it) |
| HTTP verb → operation mapping | `internal/files` (Handler in write.go) | — | Stateless HTTP surface, mirrors v3.4 read handler |
| Route registration (loopback trust) | `internal/daemon` (api.go mux) | — | Daemon owns the Unix-socket mux; auth-less per WEB-01 |
| Multipart parse + size cap | `internal/files` (Handler.Upload) | Go stdlib `net/http`, `mime/multipart` | Upload abuse mitigations live at the HTTP boundary |
| Client write methods | `internal/daemon` (DaemonClient) | — | In-process GUI/TUI consumers call the daemon socket |
| Join-code 303 → cap parsing (TD-5) | `internal/daemon` (client_remote_files.go) | — | Client-side HTTP redirect handling; reference impl already in `internal/tui` |
| File-browser hardening (TD-4) | `internal/webserver` + `frontend/src` | — | WR-01/02 are server `/app/` routes; WR-03/04/05 are React/TS |

## Standard Stack

### Core (stdlib only — no new modules for Phase 123)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os` (`os.Root`) | Go 1.26.4 | TOCTOU-safe write primitives: `Create`, `OpenFile`, `Mkdir`, `MkdirAll`, `Remove`, `RemoveAll`, `Rename`, `WriteFile` | Kernel-level sandbox; already the v3.4 read boundary `[VERIFIED: go doc os.Root.* on installed toolchain]` |
| `net/http` | Go 1.26.4 | Daemon socket mux (method-prefixed routes), `MaxBytesReader` | Already the daemon transport `[VERIFIED: internal/daemon/api.go]` |
| `mime/multipart` | Go 1.26.4 | Upload multipart parse (`r.FormFile`, `FileHeader.Filename`) | GO-2024-2599 fixed in Go 1.22+; current 1.26.4 safe `[CITED: STACK.md]` |
| `crypto/rand` + `encoding/hex` | Go 1.26.4 | Temp-file random suffix (`relPath + ".agenthub-tmp-" + randomHex()`) | Already used for grant IDs in `api.go` `[VERIFIED: internal/daemon/api.go:1034]` |

### Supporting (existing internal packages — extend, do not replace)
| Package | Purpose | When to Use |
|---------|---------|-------------|
| `internal/files` | Add `write.go` (handlers) + extend `sandbox.go` (Sandbox methods) + extend `types.go` (`FileWriteResponse`, `FileOpResponse`) | All write logic |
| `internal/daemon` | Add 5 routes in `api.go`; add 5 methods in `client.go`; fix TD-5 in `client_remote_files.go` | Daemon socket surface + client |
| `internal/capability` | NO change in Phase 123 — `PermFilesWrite` is a Phase 124 (CAP-01) concern, not FSW | Do not add the constant here unless a plan explicitly scopes it forward |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| native `root.Rename` | hand-rolled `os.Rename(absOld, absNew)` (what PITFALLS.md recommends) | The hand-rolled path re-opens a TOCTOU window and requires manual prefix-checking; native `root.Rename` is safer and simpler. **Use native.** |
| native `root.MkdirAll` | iterative `root.Mkdir` per component (what PITFALLS.md recommends) | Iterative is unnecessary in Go 1.26 — `root.MkdirAll` exists and is TOCTOU-safe. **Use native.** |
| `root.RemoveAll` for recursive delete | manual sandbox-confined walk | `root.RemoveAll` is confined to the root by construction. **Use native.** |

**Installation:** No new dependencies. `go.mod` unchanged. No `frontend/package.json` changes in Phase 123 (CodeMirror is Phase 125).

**Version verification:** `go version` → `go1.26.4 darwin/arm64`. `go.mod` declares `go 1.26.3`. `go doc os.Root.Rename` confirms the signature `func (r *Root) Rename(oldname, newname string) error` with "Both paths are relative to the root." `[VERIFIED: installed toolchain]`

## Package Legitimacy Audit

> Not applicable to Phase 123 — **zero external packages installed.** All write primitives use the Go standard library (`os`, `net/http`, `mime/multipart`, `crypto/rand`). The CodeMirror packages from STACK.md are Phase 125 (EDIT-01), not Phase 123. No slopcheck run required.

## Architecture Patterns

### System Architecture Diagram

```
                  ┌─────────────────────────────────────────────────────────┐
  In-process      │                    agenthub binary                       │
  consumers       │                                                          │
  (Phase 123      │   ┌──────────────┐         ┌────────────────────────┐    │
   targets the    │   │ DaemonClient │  Unix   │  Daemon API mux        │    │
   daemon socket) │   │ WriteFile    │ socket  │  (api.go)              │    │
                  │   │ UploadFile   ├────────▶│  PUT  /api/files/write │    │
   Wails GUI ────▶│   │ DeleteFile   │ no auth │  POST /api/files/upload│    │
   TUI       ────▶│   │ RenameFile   │(WEB-01) │  DEL  /api/files/delete│    │
                  │   │ MkdirFile    │         │  POST /api/files/rename│    │
                  │   └──────────────┘         │  POST /api/files/mkdir │    │
                  │                            └───────────┬────────────┘    │
                  │                                        │ filesHandler     │
                  │                            ┌───────────▼────────────┐    │
                  │                            │ files.Handler (write.go)│    │
                  │                            │  Write/Upload/Delete/   │    │
                  │                            │  Rename/Mkdir            │    │
                  │                            │  - 50 MiB MaxBytesReader │    │
                  │                            │  - filepath.Base + clean │    │
                  │                            └───────────┬────────────┘    │
                  │                                        │ sandboxResolver  │
                  │                            ┌───────────▼────────────┐    │
                  │                            │ files.Sandbox           │    │
                  │                            │  validateAndClean(path) │    │
                  │                            │  denylist check (FSW-06)│    │
                  │                            │  os.OpenRoot(rootPath)  │    │
                  │                            │  root.Rename/MkdirAll/  │    │
                  │                            │  RemoveAll/Create        │    │
                  │                            └───────────┬────────────┘    │
                  │                                        │ syscall          │
                  │                            ┌───────────▼────────────┐    │
                  │                            │ kernel os.Root boundary │    │
                  │                            │ (TOCTOU-safe; rejects   │    │
                  │                            │  symlink/.. escape)     │    │
                  │                            └─────────────────────────┘    │
                  └─────────────────────────────────────────────────────────┘

  TD-5 (separate, client-side): DaemonClient.ExchangeJoinCodeAtURL
    POST {remoteBaseURL}/join/exchange  --CheckRedirect: ErrUseLastResponse-->
    303 See Other + Location: /sessions/{id}?cap=<token>  --parse cap--> return token
```

### Recommended Project Structure (additions only)
```
internal/files/
├── sandbox.go        # EXTEND: add WriteFileAtomic, Rename, Mkdir, MkdirAll, Delete, denylist
├── handler.go        # UNCHANGED (read handlers stay)
├── write.go          # NEW: Handler.Write/Upload/Delete/Rename/Mkdir HTTP methods
├── types.go          # EXTEND: FileWriteResponse, FileOpResponse
├── sandbox_test.go   # EXTEND: FuzzSandboxWrite, denylist tests, rename-dest traversal
├── write_test.go     # NEW: round-trip + atomic + concurrent-read tests
internal/daemon/
├── api.go            # EXTEND: register 5 write routes (method-prefixed, auth-less)
├── client.go         # EXTEND: WriteFile/UploadFile/DeleteFile/RenameFile/MkdirFile
├── client_remote_files.go  # FIX TD-5: ExchangeJoinCodeAtURL 303 parsing
```

### Pattern 1: Atomic Durable Write (FSW-01)
**What:** Write to a sibling temp file inside the sandbox root, `f.Sync()`, then `root.Rename` to the target. The temp file MUST be a sibling (same directory) so the rename is intra-filesystem and atomic.
**When to use:** Every file write — `Write` handler and `Upload` handler both route through `WriteFileAtomic`.
**Example:**
```go
// Source: pattern derived from PITFALLS.md §Pitfall 5 + verified os.Root API.
// Mirrors the existing Sandbox per-op os.OpenRoot pattern (sandbox.go:68-79).
func (s *Sandbox) WriteFileAtomic(relPath string, content []byte) error {
    cleaned, err := validateAndClean(relPath)
    if err != nil {
        return err
    }
    if err := s.denylistCheck(cleaned); err != nil { // FSW-06
        return err
    }
    root, err := os.OpenRoot(s.rootPath)
    if err != nil {
        return fmt.Errorf("files: open root: %w", err)
    }
    defer root.Close()

    var rnd [8]byte
    _, _ = rand.Read(rnd[:])
    tmp := cleaned + ".agenthub-tmp-" + hex.EncodeToString(rnd[:])

    f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
    if err != nil {
        return err
    }
    if _, err := f.Write(content); err != nil {
        f.Close()
        _ = root.Remove(tmp)
        return err
    }
    if err := f.Sync(); err != nil { // fdatasync before rename — crash durability
        f.Close()
        _ = root.Remove(tmp)
        return err
    }
    if err := f.Close(); err != nil {
        _ = root.Remove(tmp)
        return err
    }
    if err := root.Rename(tmp, cleaned); err != nil { // atomic within same root
        _ = root.Remove(tmp)
        return err
    }
    return nil
}
```
**Notes:** `O_EXCL` on the temp prevents two concurrent writers racing on the same temp name. The `crypto/rand` suffix makes collision practically impossible. On Windows `root.Rename` to an existing destination can fail if the target is open by another process — a short retry loop is a reasonable cross-platform hardening (flag for the planner; the v3.4 code does not yet have one).

### Pattern 2: Rename / Move with Both-Path Validation (FSW-02)
**What:** Validate BOTH source and destination through `validateAndClean`, run the denylist on the destination (and source for delete-class moves), then call native `root.Rename`.
**When to use:** Same-directory rename and cross-directory move (a move is a rename to a different parent — `root.Rename` handles both within the root).
**Example:**
```go
// Source: verified os.Root.Rename (relative-to-root, TOCTOU-safe) + PITFALLS §Pitfall 2.
func (s *Sandbox) Rename(oldRel, newRel string) error {
    oldClean, err := validateAndClean(oldRel)
    if err != nil {
        return fmt.Errorf("files: rename source: %w", err)
    }
    newClean, err := validateAndClean(newRel) // THE #1 write-side traversal risk if skipped
    if err != nil {
        return fmt.Errorf("files: rename destination: %w", err)
    }
    if err := s.denylistCheck(newClean); err != nil { // cannot rename INTO a protected path
        return err
    }
    if err := s.denylistCheck(oldClean); err != nil { // nor move a protected path out
        return err
    }
    root, err := os.OpenRoot(s.rootPath)
    if err != nil {
        return fmt.Errorf("files: open root: %w", err)
    }
    defer root.Close()
    return root.Rename(oldClean, newClean) // native; both relative to root
}
```

### Pattern 3: Upload — MaxBytesReader Before Parse, Filename Sanitization (FSW-05, FSW-12)
**What:** Wrap `r.Body` in `http.MaxBytesReader` with the 50 MiB cap BEFORE `ParseMultipartForm`. Never trust `FileHeader.Filename` — run it through `filepath.Base` then `validateAndClean`, then route bytes through `WriteFileAtomic`.
**When to use:** The `POST /api/files/upload` handler.
**Example:**
```go
// Source: STACK.md §Decision 3 + PITFALLS §Pitfall 6. 50 MiB = milestone-locked cap.
const maxUploadBytes = 50 << 20

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
    sb, _, err := h.sandboxFor(r)
    if err != nil {
        http.Error(w, "session not found", http.StatusNotFound)
        return
    }
    r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes) // FIRST — before parse
    if err := r.ParseMultipartForm(8 << 20); err != nil {
        http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
        return
    }
    dir := r.FormValue("dir") // relative dir within the sandbox
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    safeName := filepath.Base(header.Filename) // strip any "../" path components
    target := filepath.Join(dir, safeName)      // validateAndClean runs inside WriteFileAtomic
    data, err := io.ReadAll(file)               // bounded by MaxBytesReader above
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if err := sb.WriteFileAtomic(target, data); err != nil {
        writeWriteError(w, err) // maps 403/400/500 — see Pattern 5
        return
    }
    writeJSON(w, http.StatusOK, FileWriteResponse{Path: target, Size: int64(len(data))})
}
```
**Note:** Streaming `io.Copy(tmpFile, file)` is more memory-efficient than `io.ReadAll` for large files; the `WriteFileAtomic` signature takes `[]byte` per FSW-01, so a streaming variant (`WriteFileAtomicFrom(relPath string, r io.Reader)`) may be worth adding to avoid buffering 50 MiB in memory. Flag for the planner — both satisfy FSW-05; the streaming form is the better engineering choice and still routes through temp+sync+rename.

### Pattern 4: Shell-RC Denylist (FSW-06)
**What:** A server-side check inside every Sandbox write method that rejects operations targeting sensitive files, returning a sentinel error mapped to `403 Protected system file`. The check operates on the resolved absolute path (`filepath.Join(s.rootPath, cleaned)`), so it fires only when the session's working directory IS at/above `$HOME`.
**Canonical denylist (from PITFALLS §Pitfall 8 + success criterion #3):**
- Shell RC: `.bashrc`, `.zshrc`, `.profile`, `.bash_profile`, `.zprofile`, `.zshenv`, `.bash_login`
- SSH: anything under `.ssh/` (`authorized_keys`, `config`, `known_hosts`, private keys)
- Agent config: anything under `.claude/` (`CLAUDE.md`, `settings.json`)
- Daemon's own config dir: `.config/agenthub/` (and the platform equivalent)
**Example:**
```go
// Source: PITFALLS §Pitfall 8, success criterion #3. Check the absolute resolved
// path so the denylist only bites home-rooted sandboxes (the dangerous case).
var ErrProtectedSystemFile = errors.New("files: protected system file")

func (s *Sandbox) denylistCheck(cleaned string) error {
    abs := filepath.Join(s.rootPath, cleaned)
    home, _ := os.UserHomeDir()
    if home == "" {
        return nil
    }
    rel, err := filepath.Rel(home, abs)
    if err != nil || strings.HasPrefix(rel, "..") {
        return nil // target is not under $HOME — denylist does not apply
    }
    base := filepath.Base(abs)
    switch base {
    case ".bashrc", ".zshrc", ".profile", ".bash_profile",
        ".zprofile", ".zshenv", ".bash_login":
        return ErrProtectedSystemFile
    }
    // directory-prefix protections (forward-slash normalized)
    relSlash := filepath.ToSlash(rel)
    for _, dir := range []string{".ssh/", ".claude/", ".config/agenthub/"} {
        if relSlash == strings.TrimSuffix(dir, "/") || strings.HasPrefix(relSlash, dir) {
            return ErrProtectedSystemFile
        }
    }
    return nil
}
```
**Anti-pattern:** Do NOT put the denylist only in the HTTP handler — it must be in the Sandbox method so ALL callers (daemon route, future webserver route, future TUI client) inherit it. The success criterion requires enforcement "on all five write methods."

### Pattern 5: Daemon Route Registration + Error Mapping (FSW-08)
**What:** Register write routes with explicit method prefixes on the existing daemon mux, auth-less (loopback trust). Map Sandbox sentinel errors to HTTP status.
**When to use:** `api.go` `registerRoutes` (add after the existing read routes at line 147).
**Example:**
```go
// Source: mirrors internal/daemon/api.go:144-147 (existing read routes).
// Method-prefixed per Go 1.22+ mux: wrong verb auto-returns 405 (Pitfall 18).
a.mux.HandleFunc("PUT /api/files/write", a.filesHandler.Write)
a.mux.HandleFunc("POST /api/files/upload", a.filesHandler.Upload)
a.mux.HandleFunc("DELETE /api/files/delete", a.filesHandler.Delete)
a.mux.HandleFunc("POST /api/files/rename", a.filesHandler.Rename)
a.mux.HandleFunc("POST /api/files/mkdir", a.filesHandler.Mkdir)

// Error mapping (write.go helper):
func writeWriteError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, ErrProtectedSystemFile):
        http.Error(w, "Protected system file", http.StatusForbidden)
    case isValidationError(err): // path traversal / device name / etc.
        http.Error(w, "access denied: "+err.Error(), http.StatusForbidden)
    default:
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```
**Note on read-route convention:** The existing read routes return `403` for validation failures ("access denied: ..."), `404` for unknown session, `400` for "is a directory". Mirror these exactly so the write surface is consistent with v3.4. Success criterion #3 specifically requires `403 Protected system file` for denylisted writes.

### Pattern 6: TD-5 Fix — ExchangeJoinCodeAtURL 303 Parsing (FSW-10)
**What:** The current `ExchangeJoinCodeAtURL` (`client_remote_files.go:107-117`) JSON-decodes the success-path response body — but the webserver returns `303 See Other` with `Location: /sessions/{id}?cap=<token>` and NO body, so the decode silently fails and the GUI can never acquire a remote cap. The TUI's `exchangeJoinCodeCmd` (`joincode_prompt.go:153-208`) already does this correctly — it is the reference implementation.
**The fix (port the TUI logic):**
1. Set `http.Client.CheckRedirect` to return `http.ErrUseLastResponse` (disable auto-follow — otherwise Go chases the 303 into `/sessions/<sid>` and loses the cap).
2. Detect `resp.StatusCode == http.StatusSeeOther` (303).
3. Read the `Location` header.
4. Handle the error-shape `Location: /join?error=<kind>` first (map to the existing modal error substrings: expired/invalid/not-found/session-gone).
5. `url.Parse(loc)`, then `u.Query().Get("cap")` for the token.
**Critical:** The current `remoteFilesHTTPClient` (the shared package-level client) has NO `CheckRedirect` — adding `ErrUseLastResponse` to that shared client could change behavior for `RegisterRemoteCap` and any future caller. Construct a dedicated client inside `ExchangeJoinCodeAtURL` (exactly as the TUI does inside its closure) rather than mutating the shared one.
**Existing error-substring contract to preserve** (the modal UI pivots on these): `"expired"`, `"invalid"`, `"not-found"`, `"session-gone"`. Map the `/join?error=<kind>` Location and the existing 4xx status codes onto these.

### Anti-Patterns to Avoid
- **Hand-rolling rename with `os.Rename(absOld, absNew)`** — the milestone PITFALLS.md recommends this, but it re-opens a TOCTOU window. Go 1.26.4 has native `root.Rename`. Use it.
- **Caching a single `os.Root` handle on the Sandbox** — the v3.4 design deliberately opens a fresh `os.OpenRoot` per operation (after a one-time `EvalSymlinks` at construction). Match this; do not "optimize" by caching the root handle (the rootPath is the cached invariant, not the handle).
- **`O_TRUNC` in-place writes** — a concurrent AI-agent reader must never see an empty/partial file. Always temp+sync+rename.
- **Temp file in `os.TempDir()`** — escapes the sandbox and loses cross-filesystem rename atomicity. Temp must be a sibling inside the root.
- **Denylist only in the handler** — must be in the Sandbox method (all callers inherit).
- **Mutating the shared `remoteFilesHTTPClient` for the TD-5 fix** — construct a dedicated client with `CheckRedirect` in the function.
- **Adding `PermFilesWrite` / `requireFilesWrite` / Origin checks in Phase 123** — those are Phase 124 (CAP). Keep Phase 123 scoped to FSW.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Sandbox-confined rename | `os.Rename` on `filepath.Join`-constructed absolute paths | `root.Rename(oldClean, newClean)` | Native method is TOCTOU-safe and relative-to-root; hand-rolled reopens the race |
| Recursive directory create | iterative `root.Mkdir` loop | `root.MkdirAll(name, perm)` | Native, confined, single call |
| Recursive subtree delete | manual sandbox-confined walk | `root.RemoveAll(name)` | Native, confined by construction |
| Multipart parsing | any third-party upload lib | stdlib `mime/multipart` via `r.FormFile` | Streams to disk, RFC-correct, zero deps |
| Upload size guard | post-read length check | `http.MaxBytesReader` before `ParseMultipartForm` | Rejects before buffering — DoS-safe |
| 303 redirect parsing (TD-5) | new redirect logic | copy `joincode_prompt.go:153-208` | Reference impl already correct and tested |

**Key insight:** The entire write surface is achievable with the Go standard library because `os.Root` (Go 1.24+, fully fleshed out by 1.26) provides every write primitive relative-to-root and TOCTOU-safe. The milestone research's "os.Root API gaps" section is obsolete for this toolchain — verify against `go doc`, not the pitfalls doc.

## Runtime State Inventory

> Phase 123 is a code-and-routes phase, not a rename/refactor/migration. Most categories are N/A. The TD-4 sweep and the daemon settings field deserve explicit answers.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — Phase 123 introduces write primitives but the sandbox writes into the session's existing working directory; no new datastore, no renamed keys. Verified: no DB/collection/key changes in FSW-01..12. | None |
| Live service config | None — no n8n/Datadog/Tailscale config touches. The daemon mux gains 5 routes (code, in git). | None |
| OS-registered state | None — no Task Scheduler / launchd / systemd / pm2 changes. | None |
| Secrets/env vars | None added by FSW. (Phase 126 will read `$EDITOR`/`$VISUAL` — out of scope here.) | None |
| Build artifacts | None — no `go.mod` change, no new vendored assets, no `pyproject`/egg-info. `frontend/package.json` is untouched in Phase 123 (CodeMirror is Phase 125). | None |
| **daemonSettings.FilesWrite** | The milestone ARCHITECTURE.md §8.2 lists a `daemonSettings.FilesWrite bool` + `schemaVersion: 4` migration as a Phase 123 item. **HOWEVER:** the settings/capability gating is a Phase 124 (CAP-08) requirement; FSW-01..12 contain NO settings-migration requirement. **Recommendation:** do NOT bump `schemaVersion` or add `FilesWrite` in Phase 123 unless a plan explicitly pulls CAP-08 forward — the FSW success criteria do not require it, and the daemon routes are auth-less (no setting gates them). Flag as an Open Question. | Defer to Phase 124 unless planner decides otherwise |

**Nothing found requiring data migration.** Verified by reading FSW-01..12 and the actual `internal/files`/`internal/daemon` source.

## Common Pitfalls

### Pitfall 1: Rename destination not validated (the #1 write-side traversal risk)
**What goes wrong:** Validating `from` but treating `to` as a trusted string → `to=../../.ssh/authorized_keys` escapes.
**Why it happens:** Read-side mental model ("can I open this?") doesn't transfer — the destination doesn't exist yet, so it must be validated by string analysis.
**How to avoid:** `validateAndClean` BOTH paths (Pattern 2). Native `root.Rename` also rejects the escape at the syscall layer, but validate first for clear errors and defense-in-depth.
**Warning signs:** Rename handler that reads `to` without `validateAndClean`; happy-path-only rename test.

### Pitfall 2: Non-atomic write corrupting files for a concurrent AI-agent reader
**What goes wrong:** `O_TRUNC` open zeros the file before bytes are written; a crash or a concurrent read sees empty/partial content. Catastrophic for live `go build` / `npm install`.
**How to avoid:** `WriteFileAtomic` (Pattern 1) — temp + `f.Sync()` + `root.Rename`.
**Warning signs:** any `OpenFile(..., O_TRUNC, ...)` in a write path; missing `f.Sync()` before rename.

### Pitfall 3: Multipart filename injection / unbounded upload
**What goes wrong:** `FileHeader.Filename` can be `../../.bashrc`; no size cap fills the disk.
**How to avoid:** `filepath.Base` + `validateAndClean`; `http.MaxBytesReader(50 MiB)` BEFORE `ParseMultipartForm` (Pattern 3).
**Warning signs:** `header.Filename` used directly as a path; `ParseMultipartForm` called before wrapping the body.

### Pitfall 4: Shell-RC overwrite when session cwd is `$HOME`
**What goes wrong:** Claude Code's default cwd is `$HOME`; a write to `.bashrc`/`.ssh/authorized_keys`/`.claude/CLAUDE.md` is "correct" sandbox behavior but grants persistent code execution.
**How to avoid:** Denylist in every Sandbox write method (Pattern 4); `403 Protected system file`.
**Warning signs:** denylist only client-side; no test writing `.bashrc` in a home-rooted sandbox.

### Pitfall 5: TD-5 — mutating the shared HTTP client / auto-following the 303
**What goes wrong:** Letting Go auto-follow the 303 chases `/sessions/<sid>` and loses the cap; mutating the shared `remoteFilesHTTPClient` changes `RegisterRemoteCap` behavior.
**How to avoid:** Dedicated client with `CheckRedirect: ErrUseLastResponse` inside `ExchangeJoinCodeAtURL`; detect 303, parse `Location` (Pattern 6).
**Warning signs:** `CheckRedirect` added to the package-level client; JSON decode on the success path.

### Pitfall 6: Wrong-verb routes returning 403/200 instead of 405
**What goes wrong:** A route registered without a method prefix matches all verbs; a `files.read`-only future cap should get 405 on `DELETE`, not silent success.
**How to avoid:** Method-prefixed registration (Pattern 5). Add regression tests: `GET /api/files/write` → 405; `POST /api/files/read` → 405.
**Warning signs:** `a.mux.HandleFunc("/api/files/write", ...)` without the `PUT ` prefix.

## Code Examples

### DaemonClient write methods (FSW-09)
```go
// Source: mirrors internal/daemon/client.go:381-449 (read methods) + filesURL helper (366).
func (c *DaemonClient) WriteFile(ctx context.Context, sessionID, relPath string, data []byte) (files.FileWriteResponse, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodPut,
        c.filesURL("write", sessionID, relPath), bytes.NewReader(data))
    if err != nil {
        return files.FileWriteResponse{}, fmt.Errorf("files write: new request: %w", err)
    }
    req.Header.Set("Content-Type", "application/octet-stream")
    resp, err := c.http.Do(req)
    if err != nil {
        return files.FileWriteResponse{}, fmt.Errorf("files write: do request: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return files.FileWriteResponse{}, fmt.Errorf("files write: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
    }
    var out files.FileWriteResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return files.FileWriteResponse{}, fmt.Errorf("files write: decode: %w", err)
    }
    return out, nil
}
// DeleteFile/RenameFile/MkdirFile follow the same shape; Rename/Mkdir send a JSON
// body via a small request struct, paralleling doJSON (client.go:488).
```
Note: the `FilesClient` *interface* (`internal/tui/files_client.go`) extension to 8 methods is a **Phase 126 (TUIW-01)** concern. Phase 123 only needs the concrete `DaemonClient` methods (FSW-09). Adding the interface methods now would force `RemoteFilesClient` to implement them prematurely — keep the interface at 4 until Phase 126.

### FuzzSandboxWrite harness (FSW-07)
```go
// Source: extends internal/files/sandbox_test.go:256 FuzzSandboxPath.
// Reuse the SAME corpus (all f.Add payloads from FuzzSandboxPath), exercised
// against the WRITE surface, plus write-specific seeds.
func FuzzSandboxWrite(f *testing.F) {
    // Reuse every FuzzSandboxPath seed (copy the f.Add block), then add:
    f.Add("../../.ssh/authorized_keys")  // rename-dest + write traversal
    f.Add("../../.bashrc")
    f.Add("../../.claude/CLAUDE.md")
    f.Add("../../../etc/cron.d/pwn")
    f.Add("foo.txt.agenthub-tmp-deadbeef") // temp-name collision probe
    f.Add("..%2f..%2f.bashrc")

    root := f.TempDir()
    sb, err := files.NewSandbox(root)
    if err != nil { f.Fatalf("NewSandbox: %v", err) }

    f.Fuzz(func(t *testing.T, rawPath string) {
        // Must never panic and never escape the root. Exercise each write method.
        _ = sb.WriteFileAtomic(rawPath, []byte("x"))
        _ = sb.Rename(rawPath, "safe-target.txt")
        _ = sb.Rename("a.txt", rawPath)   // rename-DESTINATION traversal (Pitfall 1)
        _ = sb.Mkdir(rawPath)
        _ = sb.Delete(rawPath)
        // Assertion: no file was created/modified OUTSIDE root. After each op,
        // verify nothing exists above root (walk the parent, or stat known
        // sentinel paths) — mirror the FuzzSandboxPath in-root assertion.
    })
}
```
The merge gate: `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/...` reporting 0 crashes (success criterion #1).

## State of the Art

| Old Approach (milestone research) | Current Reality (Go 1.26.4) | When Changed | Impact |
|-----------------------------------|------------------------------|--------------|--------|
| "`os.Root.Rename` does NOT exist (golang/go#69462)" — hand-roll with `os.Rename` on absolute paths | `func (r *Root) Rename(oldname, newname string) error`, relative-to-root, TOCTOU-safe | golang/go#69462 shipped before Go 1.26 | **Use native `root.Rename`** — simpler AND safer than the recommended workaround |
| "`os.Root.MkdirAll` does NOT exist — iterate `root.Mkdir`" | `func (r *Root) MkdirAll(name string, perm FileMode) error` exists | Shipped before 1.26 | Use native `root.MkdirAll` |
| "`os.Root.WriteFile` does NOT exist" | `func (r *Root) WriteFile(name string, data []byte, perm FileMode) error` exists | Shipped before 1.26 | Could use directly, but `WriteFileAtomic` (temp+rename) is still required for durability — `root.WriteFile` is `O_TRUNC`-equivalent and non-atomic |

**Verification:** Enumerated all exported `*os.Root` methods on the installed toolchain: `Name, Close, Open, Create, OpenFile, OpenRoot, Chmod, Mkdir, MkdirAll, Chown, Lchown, Chtimes, Remove, RemoveAll, Stat, Lstat, Readlink, Rename, Link, Symlink, ReadFile, WriteFile, FS`. `[VERIFIED: AST scan of $GOROOT/src/os + go doc]`

**Deprecated/outdated guidance to ignore in the milestone research:**
- PITFALLS.md §Pitfall 1 ("`os.Root` Write API Has Gaps — Rename Is NOT Available") — false for Go 1.26.4.
- ARCHITECTURE.md §8.1 implies a hand-rolled rename; use native instead.
- Note STACK.md §"New Go Capabilities" table is CORRECT (it lists `os.Root.Rename` as existing) — STACK.md and PITFALLS.md contradict each other; STACK.md is right.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `daemonSettings.FilesWrite` + `schemaVersion: 4` should NOT be added in Phase 123 (it's a CAP-08 / Phase 124 concern; no FSW requirement gates routes on a setting) | §Runtime State Inventory, §Open Questions | If the planner intends to gate the auth-less daemon routes (they aren't gated today), this would need adding; low risk since daemon socket is loopback-trusted regardless |
| A2 | The `FilesClient` interface extension stays at 4 methods in Phase 123; only concrete `DaemonClient` methods are added (FSW-09) | §Code Examples | If a Phase 123 plan tries to extend the interface, `RemoteFilesClient` must implement write methods early (a Phase 126 concern) — would expand scope |
| A3 | The 50 MiB upload cap and 8 MiB `ParseMultipartForm` in-memory threshold are appropriate defaults | §Pattern 3 | 50 MiB is milestone-locked (FSW-12); the 8 MiB parse threshold is a tuning choice, low risk |
| A4 | Denylist scopes to `$HOME`-rooted sandboxes via `filepath.Rel(home, abs)`; non-home sessions are unaffected | §Pattern 4 | If a sensitive file lives outside `$HOME` (e.g., system-wide), it wouldn't be caught — but the threat model (PITFALLS §8) is specifically home-dir sessions |
| A5 | A streaming upload variant (`io.Copy` to temp) is preferable to `io.ReadAll` for 50 MiB files | §Pattern 3 note | Both satisfy FSW-05; `io.ReadAll` buffers up to 50 MiB in memory per concurrent upload — a real but bounded cost |

## Open Questions

1. **Should Phase 123 add `daemonSettings.FilesWrite` + `schemaVersion: 4`?**
   - What we know: ARCHITECTURE.md §8.2/§10 lists it under Phase 123; CAP-08 (Phase 124) also owns settings migration; FSW-01..12 contain no settings requirement; the daemon socket routes are auth-less regardless of any setting.
   - What's unclear: whether the planner wants the settings field landed early (alongside the read-side `FilesRead` pattern) or deferred to Phase 124 with the rest of the capability gating.
   - Recommendation: **Defer to Phase 124.** Phase 123's success criteria do not require it, and adding a schema migration here splits the CAP-08 work awkwardly. If the planner disagrees, the pattern to mirror is `engine.go:99/158/182/209/508-517` and `TestSettingsMigration_FilesReadDefaultsTrue`.

2. **Owner `files.write` default — opt-in vs default-on (documentation conflict, NOT a Phase 123 concern but flag it).**
   - What we know: STATE.md §Key Decisions says "Default-ON for session owner (mirrors `files.read`)". REQUIREMENTS.md (CAP-04), the milestone scope table, and the most recent git commit (`808767f docs: make files.write opt-in for all tokens (no owner default-on)`) all say **opt-in, never default-on**.
   - What's unclear: STATE.md is stale relative to the latest decision.
   - Recommendation: The git-committed REQUIREMENTS.md is authoritative — `files.write` is opt-in. This is a **Phase 124 (CAP-04)** decision and does not affect Phase 123 (which has no capability bit), but the planner should note STATE.md needs correcting and not propagate the "default-ON" wording.

3. **Windows rename-over-open-file retry loop.**
   - What we know: `root.Rename` to an existing destination can fail on Windows if the target is open by another process (PITFALLS §Pitfall 5). The v3.4 code has no retry.
   - What's unclear: whether v3.5 needs the retry now or can defer (cross-platform is a stated target).
   - Recommendation: Add a short bounded retry (3 attempts, ~50ms) in `WriteFileAtomic`'s rename step. Low cost, closes a real Windows failure mode.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All Phase 123 code + fuzz | ✓ | go1.26.4 darwin/arm64 | — |
| `os.Root` write API | FSW-01..06 | ✓ | Go 1.26.4 (Rename/MkdirAll/RemoveAll/WriteFile all present) | — |
| `go test -fuzz` | FSW-07 merge gate | ✓ | built into toolchain | — |
| race detector (`-race`) | success criterion #5 | ✓ | built into toolchain | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None. Phase 123 is pure stdlib + existing internal packages.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) + `testing.F` fuzzing |
| Config file | none — standard `go test` |
| Quick run command | `go test ./internal/files/... ./internal/daemon/...` |
| Full suite command | `go test -race ./internal/files/... ./internal/daemon/...` |
| Fuzz gate | `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FSW-01 | atomic write; concurrent reader never sees partial | unit | `go test ./internal/files/ -run TestWriteFileAtomic` | ❌ Wave 0 (`write_test.go`) |
| FSW-02 | rename validates both paths; cross-dir move works | unit | `go test ./internal/files/ -run TestRename` | ❌ Wave 0 |
| FSW-03 | mkdir/mkdirall confined; traversal rejected | unit | `go test ./internal/files/ -run TestMkdir` | ❌ Wave 0 |
| FSW-04 | delete file + recursive subtree within root | unit | `go test ./internal/files/ -run TestDelete` | ❌ Wave 0 |
| FSW-05 | upload filename sanitized; routes through atomic | unit | `go test ./internal/files/ -run TestUpload` | ❌ Wave 0 |
| FSW-06 | denylist 403 on .bashrc/.ssh/.claude in home-rooted sandbox | unit | `go test ./internal/files/ -run TestDenylist` | ❌ Wave 0 |
| FSW-07 | fuzz write surface, 0 crashes | fuzz | `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/...` | ❌ Wave 0 (extend `sandbox_test.go`) |
| FSW-08 | 5 daemon routes; PUT write round-trips over socket; wrong verb → 405 | integration | `go test ./internal/daemon/ -run TestFilesWriteRoutes` | ❌ Wave 0 (extend `api_test.go`) |
| FSW-09 | DaemonClient write methods round-trip | integration | `go test ./internal/daemon/ -run TestDaemonClientWrite` | ❌ Wave 0 (extend `client_test.go`) |
| FSW-10 | TD-5: ExchangeJoinCodeAtURL parses 303 cap | unit | `go test ./internal/daemon/ -run TestExchangeJoinCode` | ❌ Wave 0 (extend test for 303 path) |
| FSW-11 | TD-4 WR-01..05 hardening | unit + manual | per-item (see TD-4 inventory) | partial |
| FSW-12 | 50 MiB cap; over-cap → clear error not truncated file | integration | `go test ./internal/daemon/ -run TestUploadSizeCap` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/files/... ./internal/daemon/...`
- **Per wave merge:** `go test -race ./internal/files/... ./internal/daemon/...` + 60s fuzz
- **Phase gate:** Full suite green with `-race`; `FuzzSandboxWrite` 0 crashes; success criteria #1-5 all pass before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/files/write_test.go` — covers FSW-01..06 (atomic, rename, mkdir, delete, upload, denylist)
- [ ] `internal/files/sandbox_test.go` extension — `FuzzSandboxWrite` (FSW-07)
- [ ] `internal/daemon/api_test.go` extension — 5 write-route tests incl. 405 wrong-verb (FSW-08, FSW-12)
- [ ] `internal/daemon/client_test.go` extension — DaemonClient write-method round-trips (FSW-09)
- [ ] `internal/daemon/client_remote_files` test — 303 success path + error-shape Location (FSW-10)
- Framework install: none — `go test` is built in.

## Security Domain

> `security_enforcement` config key not located in `.planning/config.json` scope read; treating as enabled (absent = enabled). This phase is the v3.5 security foundation, so the domain is squarely in scope.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Daemon socket is auth-less by design (WEB-01 loopback trust); capability auth is Phase 124 |
| V3 Session Management | no | No session tokens at the FSW layer (Phase 124+) |
| V4 Access Control | yes | Shell-RC denylist (FSW-06) — server-side, in every write method; `os.Root` sandbox confinement |
| V5 Input Validation | yes | `validateAndClean` on every path incl. rename destination; `filepath.Base` on upload filename; method-prefixed routes |
| V6 Cryptography | yes (light) | `crypto/rand` for temp-file suffix; TD-5 cap token redaction in error messages (`redactCapTokenFromError`) — never hand-roll |
| V12 File & Resources | yes | `http.MaxBytesReader` (50 MiB); no archive extraction (no zip-slip surface); temp-file cleanup on failure |

### Known Threat Patterns for Go file-write sandbox over loopback socket
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via rename destination | Tampering | `validateAndClean` BOTH paths + native `root.Rename` syscall boundary |
| Symlink-escape write (TOCTOU) | Tampering | `os.Root` kernel confinement on every op (fresh `os.OpenRoot` per call) |
| Shell-RC / SSH-key overwrite (home-dir session) | Elevation of Privilege | Server-side denylist → `403 Protected system file` |
| Multipart filename injection | Tampering | `filepath.Base` + `validateAndClean` |
| Upload disk-fill DoS | Denial of Service | `http.MaxBytesReader` before `ParseMultipartForm`; 413 before any bytes hit disk |
| Partial/corrupt file on crash | Tampering / DoS | Atomic temp + `f.Sync()` + rename; temp cleanup on error |
| Cap-token leak in TD-5 error path | Information Disclosure | `redactCapTokenFromError` (already exists); dedicated client, no token logging |

## Sources

### Primary (HIGH confidence — verified against live code / installed toolchain)
- `go doc os.Root.Rename` / `.MkdirAll` / `.WriteFile` + AST enumeration of `$GOROOT/src/os` — confirms native write API on Go 1.26.4
- `internal/files/sandbox.go` — per-op `os.OpenRoot` pattern, `validateAndClean`, `validateRelativePath`, device-name/ADS/UNC defenses
- `internal/files/handler.go` — stateless Handler, `sandboxResolver`, error-status conventions (403/404/400), `maxPreviewBytes`
- `internal/files/types.go` — `FileEntry`, `FileListResponse` (wire-type conventions for new `FileWriteResponse`/`FileOpResponse`)
- `internal/files/sandbox_test.go:256` — `FuzzSandboxPath` corpus + harness (to extend for `FuzzSandboxWrite`)
- `internal/daemon/api.go:144-158` — read-route registration, loopback-trust comment, method-prefix convention; `:1015-1084` `issueCapabilitiesForSession` (`files.read` token wiring)
- `internal/daemon/client.go:366-484` — `filesURL` helper + read-method patterns (to mirror for write methods)
- `internal/daemon/client_remote_files.go:107-117` — the exact TD-5 bug (JSON decode on 303 body)
- `internal/daemon/remote_files.go:169` — `proxyRemoteFiles` nil-body (Phase 124 CAP-10 concern, not Phase 123)
- `internal/tui/joincode_prompt.go:153-208` — the correct 303-parsing reference impl for the TD-5 fix
- `internal/capability/capability.go` — `HasPerm` whole-token semantics, `PermFilesRead` (Phase 124 reference)
- `.planning/milestones/v3.4-phases/120-.../120-VERIFICATION.md:141-146` — TD-4 WR-01..05 exact OPEN status + source locations

### Secondary (MEDIUM-HIGH — milestone research, cross-checked against code)
- `.planning/research/ARCHITECTURE.md`, `STACK.md` — write design, daemon flow, stdlib sufficiency (STACK.md's os.Root table is correct)
- `.planning/research/PITFALLS.md` — threat menagerie (CORRECTED: its os.Root "API gaps" claims are stale for Go 1.26.4)
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md` — requirement text + phase plan (STATE.md "default-ON" wording is stale — see Open Question 2)

## TD-4 Inventory (FSW-11) — verified OPEN at these exact locations

| Item | Location | Fix |
|------|----------|-----|
| WR-01 | `internal/webserver/server.go:555-578` | `/app/` directory listings exposed — disable directory listing for the bundle route |
| WR-02 | `internal/webserver/server.go:555-578` | `/app/` bundle missing cache-control headers — add appropriate `Cache-Control` |
| WR-03 | `frontend/src/components/FileBrowserTab.tsx:58-61` | `joinPath` name sanitization — strip leading slash from server-returned names (UI defense-in-depth; partial code already at line 399) |
| WR-04 | `frontend/src/components/FileRow.tsx:57-65` | `formatRowMtime` fallback when mtime is empty string |
| WR-05 | `frontend/src/components/humanSize.ts:7-22` | comment clarity |

WR-06 (`BreadcrumbBar.tsx:49-58`) was analyzed and downgraded to NO ACTION in Phase 120; WR-07 (`PreviewPane.tsx:144`) was already FIXED in Plan 120-04. Do not re-touch.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib, verified against the installed toolchain via `go doc` and AST enumeration
- Architecture: HIGH — read directly from live `internal/files` and `internal/daemon` source; the v3.4 patterns are unambiguous
- Pitfalls: HIGH — corrected the one milestone-research error (os.Root API gaps) empirically; remaining pitfalls cross-checked against code
- TD-4/TD-5: HIGH — exact source locations and the reference 303 impl verified in code

**Research date:** 2026-06-14
**Valid until:** 2026-07-14 (stable — stdlib-only; the only volatile element was the os.Root API surface, now pinned to the installed 1.26.4)
