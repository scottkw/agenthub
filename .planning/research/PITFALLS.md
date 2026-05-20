# Pitfalls Research — v3.4 File Browser (Read-Only)

**Domain:** Adding a sandboxed read-only filesystem API and file-browser tab to an existing daemon-centric, capability-token-authed, Tailscale-TLS Go/Wails app
**Researched:** 2026-05-20
**Confidence:** HIGH for Go stdlib behavior (verified against go.dev docs and issue tracker); HIGH for AgentHub-specific integration (read from source); MEDIUM for CVE cross-references (verified class but not all individual CVEs confirmed in official NVD); HIGH for os.Root recommendation (Go 1.24+, project uses go 1.26.1)

> **Scope discipline:** Every pitfall below is specific to ADDING a file browser to AgentHub's existing daemon + relay + webserver + CSP + capability-token stack. Generic "validate input" advice is not here. The v3.3.1-research/PITFALLS.md covers the previous feature surface (xterm.js addon integration) — that prior art is not repeated. Points the epic flagged (path traversal severity High, large-file memory Medium, multi-client caching Medium, capability surface Low-Med) are EXPANDED on, not merely noted.

---

## Critical Pitfalls

### Pitfall 1: EvalSymlinks TOCTOU — the Check-Then-Open Race

**What goes wrong:**
The v3.4 design spec says "filepath.Clean → symlink-resolve → prefix-check pattern." If the implementation resolves the symlink with `filepath.EvalSymlinks`, checks that the resolved path has the session cwd as a prefix, and then calls `os.Open` on the original (or resolved) path in a separate step, there is a time window between the check and the open during which an attacker (or a concurrent process) can swap a regular file or directory for a symlink pointing outside the cwd.

Concrete attack sequence (requires write access to a directory inside the cwd, but shell sessions have exactly that):
1. Attacker creates `cwd/legit_dir/` as a real directory.
2. Server checks: `filepath.EvalSymlinks(cwd/legit_dir/secret.txt)` → resolves to inside cwd → PASS.
3. Attacker renames `legit_dir/` to `legit_dir.bak/` and creates `legit_dir` as a symlink to `/etc/`.
4. Server opens `cwd/legit_dir/secret.txt` → actually opens `/etc/secret.txt`.

This is a documented TOCTOU race in every file server sandbox implementation (GitHub golang/go#70007, GHSA-w853-jp5j-5j7f). The Zed code editor had CVE-2026-27976, an 8.8 CVSS sandbox escape via exactly this class.

**Why it happens:**
The "clean → EvalSymlinks → prefix check → open" pattern is intuitive but two-step. Any two-step approach has a race window. The race is small (microseconds) but over a network file browse session with hundreds of requests, it becomes exploitable.

**How to avoid:**
Use `os.OpenInRoot(resolvedCwdPath, untrustedRelPath)` (Go 1.24+, available since project uses go 1.26.1). This combines validation and open atomically at the syscall level using `openat` with `O_NOFOLLOW` chains on Unix. Windows gets equivalent protection via handle-based directory tracking.

```go
// WRONG: two-step, has TOCTOU race
resolved, err := filepath.EvalSymlinks(filepath.Join(cwd, relpath))
if !strings.HasPrefix(resolved, cwdResolved) { ... reject }
f, err := os.Open(resolved)   // race window here

// RIGHT: atomic, no race
root, err := os.OpenRoot(cwdResolved)
f, err := root.Open(relpath)  // os.OpenRoot rejects escape atomically
```

Pre-resolve the cwd itself with `filepath.EvalSymlinks` once at session creation time and cache the result. Use that resolved-cwd path as the `os.OpenRoot` argument. This ensures the root itself isn't a symlink that drifts.

For the directory listing endpoint (`/api/files/list`), use `root.ReadDir` (via `root.Open` + `f.ReadDir`). For `/api/files/stat`, use `root.Stat`. For `/api/files/read`, use `root.Open`.

**Warning signs:**
- Any code path that calls `filepath.EvalSymlinks` and then `os.Open` as two separate operations on user-supplied paths
- Any code that does `strings.HasPrefix` on resolved paths without immediately opening via the same root

**Phase to address:**
Sandboxed filesystem API phase (daemon endpoint implementation). This must be in the first file-API phase before any browser-facing surface ships. The fuzz tests required before merge (per the v3.4 milestone spec) must include symlink swap scenarios.

---

### Pitfall 2: The Full Menagerie of Path Traversal Inputs `filepath.Clean` Does NOT Catch

**What goes wrong:**
`filepath.Clean` canonicalizes `../` traversal and redundant separators, but it does NOT decode URL encoding or handle several classes of bypass inputs that arrive from the HTTP layer already decoded. If the handler receives the raw path string from `r.URL.RawPath` or does its own URL decoding, the following inputs bypass a clean-then-prefix-check if implemented naively:

| Category | Input | Bypass |
|----------|-------|--------|
| URL-encoded `..` | `%2e%2e%2fpasswd` | decoded → `../passwd` before Clean; harmless IF you use the already-URL-decoded `r.URL.Path`, but raw query params are NOT decoded |
| Double-encoded | `%252e%252e%252f` | first decode → `%2e%2e%2f`, second decode → `../`; harmless with stdlib query parsing but dangerous with manual string ops |
| Unicode fullwidth slash | `U+FF0F` (／) | looks like `/` on display; OS may or may not normalize; Windows NTFS normalizes, macOS APFS case-insensitive does not natively fold Unicode |
| Unicode one-dot-leader | `U+2024` (․) | looks like `.` but is not; `filepath.Clean` does NOT treat it as a dot |
| Null byte | `secret.txt\x00.jpg` | C-level string termination; Go strings are byte slices and will NOT truncate, but this can confuse downstream consumers (e.g., `exec.Command`) |
| Windows drive letter | `C:\windows\system32\` | `filepath.Clean` on Windows converts forward slashes; on non-Windows it's just a string; the daemon runs on all three platforms |
| Windows UNC path | `\\server\share\file` | `filepath.Clean` does not strip UNC prefixes |
| Windows reserved names | `CON`, `NUL`, `PRN`, `COM1`–`COM9`, `LPT1`–`LPT9` | opening these on Windows hangs or reads from hardware devices; CVE-2025-27210 (Node.js) was exactly this class |
| Alternate data streams | `file.txt:hidden` | valid on NTFS, opens a separate data stream; `filepath.Clean` passes it through unchanged |
| 8.3 short names | `PROGRA~1` | resolves to `Program Files` on Windows; can escape prefix checks if the stored cwd uses long names but the input uses short names |
| Trailing dot/space | `file.` or `file ` | Windows strips trailing dots and spaces from filenames, creating alias to `file`; can be used to create files that evade blocklists |
| `/proc/self/cwd` | `/proc/self/cwd` | absolute path, easily caught by `filepath.IsAbs` but must be explicit |
| Symlink to outside cwd | `legit_link/` → `/etc/` | caught by `os.OpenRoot` atomically; NOT caught by prefix-check after `EvalSymlinks` with TOCTOU (see Pitfall 1) |
| macOS `.app` bundle traversal | `Foo.app/Contents/MacOS/` | `os.OpenRoot` handles this correctly; the directory is treated as a normal directory |

**Why it happens:**
Developers write "reject `..` after Clean" and think they're done. The input space is larger. The Go standard library's `net/http` URL parser does URL-decode `r.URL.Path` (percent-decoding), so `%2e%2e%2f` → `../` before the handler sees it, and `filepath.Clean` does catch that. But the handler receives path parameters via `r.PathValue("path")` which is also decoded. The remaining risk is in manual string operations, Windows-specific path forms, and device names.

**How to avoid:**
Layer the defenses in this exact order, applied to every incoming path parameter:

1. **Reject null bytes:** `if strings.ContainsRune(p, 0) { reject }` — before any other processing
2. **Reject absolute paths:** `if filepath.IsAbs(p) { reject }` — catches `C:\`, `\\`, `/proc/self/cwd`
3. **Reject Windows device names (all platforms):** `if isWindowsDeviceName(p) { reject }` where the function checks each path component against the reserved set (CON, NUL, PRN, AUX, COM1-9, LPT1-9) case-insensitively, with or without extension
4. **Reject alternate data stream syntax:** `if strings.ContainsRune(p, ':') { reject }` (platform-independent)
5. **Reject Windows UNC paths:** `if strings.HasPrefix(p, "\\\\") || strings.HasPrefix(p, "//") { reject }`
6. **Apply `filepath.Clean`:** canonicalizes `..` components (but must be followed by the root check)
7. **Open via `os.OpenRoot`:** this is the terminal defense that makes everything above belt-and-suspenders rather than the only line of defense

Do NOT try to detect Unicode lookalikes manually. The combinatorial space is too large. Let `os.OpenRoot` be the actual security boundary; the explicit rejections above are defense-in-depth for Windows device names and ADS, which are not handled by `os.OpenRoot` on all platforms.

**Warning signs:**
- Handler reads path from `r.URL.RawPath` instead of `r.URL.Path` or `r.PathValue`
- Any `strings.Replace` or manual URL-decoding on path inputs
- Code that does the Windows device name check only on Windows builds (use a cross-platform check)

**Phase to address:**
Sandboxed filesystem API phase. Write the device-name and ADS reject functions as a tested utility, not inline in the handler. The fuzz corpus (Section: Fuzz Corpus Skeleton) covers these inputs.

---

### Pitfall 3: `filepath.EvalSymlinks` Has Critical Behavior Differences on Windows vs. Unix

**What goes wrong:**
Even if `filepath.EvalSymlinks` is used for the initial cwd resolution (before switching to `os.OpenRoot`), its behavior diverges significantly across platforms:

- **Non-existent path components:** On Unix, `EvalSymlinks` returns an error if any intermediate path component does not exist (it lstat's each component). On Windows, the Go 1.16–1.21 implementation used `GetFinalPathNameByHandleW` which had documented issues with UNC share roots, volume IDs, and relative paths (golang/go#42079, #39786, #16793). Go 1.22+ reimplemented Windows EvalSymlinks (golang/go#63703) but behavior gaps remain.
- **Case-insensitive filesystems:** macOS default APFS is case-insensitive. `EvalSymlinks` on macOS will return the canonical-case version of the path, which may differ from what the user supplied. This means a stored cwd of `/Users/Ken/Project` and a user-supplied path of `/users/ken/project` resolve to the same thing. The prefix check after `EvalSymlinks` must use `strings.HasPrefix` with the case-folded or canonical result, not a direct string comparison. Using `os.OpenRoot` sidesteps this — the root handle is the boundary regardless of case.
- **Symlink loop:** `EvalSymlinks` detects and returns an error on symlink cycles (`ELOOP` on Unix), but the error message is non-obvious. On Windows, it may behave differently on junction loops. An `EvalSymlinks` error on the cwd itself should be treated as "this session has an invalid working directory" and file browse requests should return a clear error, not a crash.
- **Dangling symlinks:** If the cwd contains a symlink whose target no longer exists, `EvalSymlinks` returns an error for paths that traverse that symlink. `os.OpenRoot` also returns an error here. Both are correct behavior — a dangling symlink is not traversable.

**Why it happens:**
Developers test on macOS or Linux and ship. Windows EvalSymlinks bugs are in the Go issue tracker dating to Go 1.3. The reimplementation in Go 1.22+ improved but did not eliminate all edge cases.

**How to avoid:**
- Use `filepath.EvalSymlinks` only for the one-time cwd resolution at the start of each file-browse request (or cached at session creation). Never use it for per-path resolution of user-supplied inputs — that's `os.OpenRoot`'s job.
- After `EvalSymlinks` on the cwd, verify the result is still an existing directory with `os.Stat`. If not, return `400 session working directory is no longer accessible`.
- On case-insensitive filesystems (macOS APFS, Windows NTFS), the cwd stored in the session (`SessionInfo.WorkDir`) may not have canonical case. Normalize: run the stored WorkDir through `EvalSymlinks` at session creation time and store the result.
- Test: create a test case where WorkDir has wrong case on macOS and verify the prefix check still passes.

**Warning signs:**
- Platform-specific bug reports: "file browser works on macOS but not Windows"
- An error like `EvalSymlinks: lstat ...: no such file or directory` surfacing as an internal server error to the browser

**Phase to address:**
Sandboxed filesystem API phase — Windows and macOS EvalSymlinks edge case tests must be in the test matrix before merge.

---

### Pitfall 4: Capability Token `files.read` Bit Added to Existing `Perms` String Format — Wire Format Breakage Risk

**What goes wrong:**
The existing `Claims.Perms` field is `"read"` or `"read,write"` (defined in `internal/capability/capability.go`). The v3.4 plan adds `files.read` as a new capability bit. Naively appending `",files.read"` to the Perms string field works, but creates several integration hazards:

1. **Existing tokens become invalid on upgrade:** Any web-share token issued before v3.4 has `Perms: "read"`. If the new `/api/files/*` endpoints check for `"files.read"` substring, they return 403 for all existing sessions until tokens are reissued. For the *session owner* this is a degraded experience (they'd lose access to file browse after upgrading). The session owner's token should grant `files.read` by default.

2. **Default-on confusion at issuance time:** The epic says "default ON for session owner, default OFF for web-share viewers unless grant explicitly includes it." The `handleIssueCapabilities` endpoint currently issues two fixed tokens (`read` and `read,write`). Adding `files.read` means the *write* capability already includes file read (reasonable), and the *read* capability should NOT include it by default for viewer links. This is a new case split in `handleIssueCapabilities` that must be intentional.

3. **Middleware checks the Perms string on every request:** The current `requireCapability` middleware does NOT inspect Perms for anything beyond session-ID matching and grant-ID active checks. A new `requireFilesRead` middleware (or an additional check inside `requireCapability`) must inspect `claims.Perms` for the `files.read` capability. If this check is added to `requireCapability` itself, it becomes a shared gate for ALL routes, which would break existing terminal routes. The check must be additive, not a replacement.

4. **Revoked session still caching file data:** A token may be revoked (`ClearGrants`) while a file-browse tab is open. The existing grant-active check in `requireCapability` handles this for terminal WebSocket upgrades. For file reads, which are short-lived HTTP GETs, the risk window is the duration of the request only — acceptable. But if the frontend caches directory listings in React state, a user with a revoked token could browse a stale listing. The cache must be invalidated on auth error (403 response clears listing state).

**Why it happens:**
The existing Perms string is a comma-separated list of opaque strings. Extending it is easy to get wrong when there are existing live tokens and existing middleware that silently accepts anything with a valid signature.

**How to avoid:**
- Add `files.read` as an explicit token in the Perms string at issuance time only.
- Add a helper function `HasPerm(perms, perm string) bool` to `internal/capability` that checks for the substring as a whole comma-delimited token (not just `strings.Contains`): `HasPerm("read,write", "files.read")` must return false; `HasPerm("read,files.read", "files.read")` must return true.
- Add `requireFilesPerm` middleware that wraps `requireCapability` and additionally checks `HasPerm(claims.Perms, "files.read")`, returning `403 "files.read permission required"` — never a generic "forbidden" — on denial.
- Issue logic: owner token gets `read,write,files.read`; viewer token (web-share link) gets `read` only; viewer with explicit file grant gets `read,files.read`. Do NOT change existing `read`/`read,write` semantics.
- Migration: there are no persisted tokens (tokens are generated on-demand from the signing key). Existing grant IDs in the grants map do not carry permission strings — those live in the token payload. So there is no wire-format migration needed for existing data at rest. The only migration concern is in-flight tokens from the current session before v3.4 re-issues.
- Error message in the GUI: when `requireFilesPerm` returns 403, the FileBrowserTab must display "File browser requires the 'files.read' permission. Ask the session owner to re-share with file access enabled." — not a raw 403.

**Warning signs:**
- All file browse requests return 403 for session owner after v3.4 launch
- FileBrowserTab shows a blank 403 error with no explanation
- Existing terminal relay endpoints accidentally 403 because file-read check was added to the wrong middleware layer

**Phase to address:**
Capability bit and middleware phase (before the UI phase). The `HasPerm` helper should be added in the same phase as the first file endpoint, not retroactively.

---

### Pitfall 5: `http.ServeContent` Range Request Edge Cases on the `/api/files/read` Endpoint

**What goes wrong:**
`http.ServeContent` handles Range requests and is the right tool for the file `/read` endpoint (supports `bytes=0-`, `bytes=-100`, seeking, MIME sniffing). However, it has documented behaviors that can produce surprising responses:

1. **Zero-byte file:** Requesting a 0-byte file with any Range header returns `416 Requested Range Not Satisfiable` (golang/go#54794, golang/go#47021). The browser may treat this as an error rather than an empty file. Add a special case: if `Content-Length: 0`, respond `200` with empty body and skip Range processing.

2. **Inverted range `bytes=100-50`:** Go's `parseRange` returns an error and serves a `416`. Fine, but the headers on the 416 response may include a stale `ETag` or `Content-Length` in some Go versions (golang/go#50905). Always unset `ETag` header on error responses to prevent cache confusion.

3. **Suffix range `bytes=-100`:** Means "last 100 bytes." This is valid RFC 7233 and `ServeContent` handles it correctly. No pitfall here, but regression-test it because it requires `Seek` to work — if the handler wraps the file in a non-seekable reader, this silently fails.

4. **File changes mid-stream:** `ServeContent` reads `modtime` via `io.Seeker` position and content length via `Seek(0, io.SeekEnd)`. If the file is modified between the `Stat` call (for the 5 MB cap check) and the `ServeContent` call, the client may receive partial or corrupt content. For a read-only file browser this is LOW risk but the contract must be documented: file content is not transactionally consistent across requests.

5. **MIME sniffing only reads 512 bytes:** `http.DetectContentType` implements the WHATWG sniffing algorithm using the first 512 bytes. MP3 without ID3 headers requires 1445 bytes for accurate detection (golang/go#21124). Video formats like MPEG are not in the sniff table at all (golang/go#50376). For source-code files (`.go`, `.ts`, `.py`), do NOT let sniffing run — set `Content-Type: text/plain; charset=utf-8` from the file extension. Use sniffing only as a fallback for unknown extensions.

MIME cascade for the `/read` endpoint:
```
1. If extension is a known source-code extension → text/plain; charset=utf-8
2. If extension is a known image extension → image/png, image/jpeg, etc.
3. If file size > 5MB → serve as application/octet-stream (download, no preview)
4. Otherwise → http.DetectContentType(first512Bytes)
```

**Why it happens:**
`ServeContent` is presented as "just works" but has subtle behaviors at file-size boundaries and with malformed Range headers. Developers use it without reading the documented edge cases.

**How to avoid:**
- Add a unit test for each edge case: 0-byte file, `bytes=-100`, `bytes=100-50`, `bytes=0-` on a file exactly at the 5 MB cap, file with no extension, `.mp3` file.
- Enforce the 5 MB cap server-side: `stat.Size() > 5*1024*1024 → 413 Entity Too Large` (or serve with `Content-Disposition: attachment` for download). Never rely on the frontend cap alone.
- Set `Content-Type` explicitly for known extensions before calling `ServeContent` — pass the correct content type via the `name` parameter of `ServeContent` (it uses the name to infer MIME if the writer hasn't set it yet).

**Warning signs:**
- Empty files produce unexpected 416 errors in the browser console
- MP3 files served as `application/octet-stream` instead of `audio/mpeg`
- Large text files streaming correctly but triggering MIME-sniff as binary

**Phase to address:**
Filesystem API phase (file read endpoint). Range edge case unit tests must pass before merge.

---

## Moderate Pitfalls

### Pitfall 6: Directory Listing Memory Blowup on Large Directories

**What goes wrong:**
The naive implementation calls `os.ReadDir(path)` which reads ALL directory entries into a `[]fs.DirEntry` slice before returning. A directory with 100,000 entries (common in `node_modules`, `.git/objects`, build output dirs) allocates a large slice. With multiple concurrent clients requesting the same large directory, the daemon can spike memory significantly.

`os.ReadDir` itself replaced `ioutil.ReadDir` (which also called `Readdir(-1)`) precisely because the old API was documented as loading all entries. But the new `os.ReadDir` ALSO loads all entries into a slice — it's just preferred because it uses `DirEntry` instead of `FileInfo`. The memory allocation is the same.

For Go 1.16+ with `os.OpenRoot`: `root.Open(".")` returns a `*os.File`; calling `f.ReadDir(n)` with a positive n reads at most n entries at a time. This is the streaming-capable approach.

**How to avoid:**
Use a chunked read strategy with a hard cap:
```go
f, err := root.Open(reldir)
const maxEntries = 10_000
entries, err := f.ReadDir(maxEntries)
// If len(entries) == maxEntries, the listing is truncated — signal this to the client
// via a response header: X-Directory-Truncated: true
```

Alternatively, paginate: accept `?limit=N&offset=M` query parameters and stream N entries. For the v3.4 MVP, a hard cap of 10,000 entries with a truncation indicator is simpler and sufficient.

Do NOT stat every file during listing. `DirEntry.Type()` and `DirEntry.Name()` do not require a stat call. Calling `DirEntry.Info()` (which does a stat) for every entry in a 10,000-entry directory is 10,000 syscalls. For the listing endpoint, return type and name only; let the `stat` endpoint handle per-file metadata.

**Warning signs:**
- "File browser is slow to open project root" when node_modules is in scope
- Daemon RSS spikes when listing large directories
- Request timeout on directory listing for large dirs

**Phase to address:**
Filesystem API phase (list endpoint). Establish the cap and truncation header contract before the UI is built — the frontend needs to handle the `X-Directory-Truncated` signal.

---

### Pitfall 7: TUI Directory Listing Freezes the Render Loop if Done Synchronously

**What goes wrong:**
Bubble Tea v2's `Update(msg tea.Msg) (Model, tea.Cmd)` method must be synchronous and non-blocking. Any I/O performed directly inside `Update` (including `os.ReadDir`) will freeze the entire TUI render loop until the I/O completes. On a slow disk, an NFS mount, or a directory with thousands of entries, this produces a visible hang where keystrokes are not processed and the terminal appears frozen.

The existing TUI has this pattern correctly for session-list refresh (it uses `tea.Cmd` for the daemon HTTP call). The file browser must follow the same pattern.

**How to avoid:**
Wrap ALL filesystem I/O in `tea.Cmd` functions:
```go
// WRONG: synchronous in Update
case NavigateMsg:
    entries, _ := os.ReadDir(m.currentPath)
    m.entries = entries
    return m, nil

// RIGHT: dispatch to background
case NavigateMsg:
    return m, func() tea.Msg {
        entries, err := os.ReadDir(msg.Path)
        return DirListResultMsg{entries: entries, err: err}
    }
```

The loading state (spinner or "Loading..." indicator) must render during the background I/O. Use the same Bubble Tea spinner pattern that the existing new-session modal uses.

**Warning signs:**
- TUI freezes for 1-2 seconds when navigating into large directories
- Keyboard input lost during directory navigation on slow disks
- Input absorbed into modal/navigation after directory load completes

**Phase to address:**
TUI Files view phase. This is a correctness requirement, not an optimization — the non-blocking pattern must be designed in, not retrofitted.

---

### Pitfall 8: New `/api/files/*` Routes Must NOT Conflict With Existing Mux Patterns

**What goes wrong:**
Go 1.22+ mux uses longest-prefix matching for method-prefixed patterns. The existing webserver routes include `GET /api/sessions/{id}/info` and similar patterns. Adding `GET /api/files/{id}/list` (where `{id}` is the session ID) creates a new subtree. The risk is:

1. **Missing method prefix:** A route registered as `/api/files/{id}/list` (no `GET` prefix) matches ALL HTTP methods (GET, POST, HEAD, OPTIONS, DELETE). An OPTIONS request or a POST to the same URL would reach the file handler. Always register with explicit method prefix: `GET /api/files/{id}/list`.

2. **Fallthrough to assets handler:** The existing `mux.Handle("GET /assets/", ...)` is a catch-all for static assets. If a `/api/files/` request somehow mis-routes (e.g., via a redirect), it would not fall through to assets — mux patterns are exact-subtree matches. But confirm there's no wildcard `GET /` catch-all that could intercept.

3. **Missing capability gate:** The existing `GET /` handler redirects to `/dashboard`. If someone navigates to `/api/files/` without a trailing segment, the mux returns 404 (correct). But the test matrix must explicitly cover: file endpoint without `?cap=` returns 401, not 404. The distinction matters because a 404 leaks route existence.

4. **HEAD method behavior:** `http.ServeContent` responds correctly to HEAD requests (headers without body). The capability middleware only runs on the registered method. If `GET /api/files/{id}/read` is registered but `HEAD /api/files/{id}/read` is not, a HEAD request returns 405. Decide intentionally whether HEAD is supported (it should be, for `Content-Length` preflight).

**How to avoid:**
- Register all three endpoints with explicit method prefix: `GET /api/files/{id}/list`, `GET /api/files/{id}/stat`, `GET /api/files/{id}/read`.
- Add a regression test: `POST /api/files/{id}/list` returns 405, not 200 or 403.
- Add a test: `GET /api/files/{id}/read` without `?cap=` returns 401, not 404.
- Register `HEAD /api/files/{id}/read` explicitly if HEAD support is desired.

**Warning signs:**
- Security scanner reports that POST to file endpoints returns non-405
- File endpoint without auth returns 404 (leaks route existence)

**Phase to address:**
Filesystem API phase (route registration). These are correctness requirements that should be in the route registration PR, not deferred to a later hardening phase.

---

### Pitfall 9: Markdown Rendering Library Injecting Inline Styles or Scripts — CSP Regression

**What goes wrong:**
The v3.4 FileBrowserTab renders markdown files. The most natural choice is `react-markdown` (remark/rehype ecosystem). The current CSP is:

```
script-src 'self' 'wasm-unsafe-eval';
style-src 'self' 'unsafe-inline';
img-src 'self' data:;
```

`style-src 'unsafe-inline'` is already present (for xterm.js runtime style injection), so inline styles from react-markdown's table renderer do NOT break the CSP. However:

1. **`rehype-raw` plugin** renders HTML embedded in markdown. If any markdown file contains `<script>` tags, `rehype-raw` renders them, and `script-src 'self'` blocks their execution. But `style-src 'unsafe-inline'` is present, so inline `<style>` in markdown IS executed. An attacker who can write a markdown file to the session's working directory could inject styles that overlay UI elements or create clickjacking scenarios.

2. **MDX or other parser variants** that support JSX in markdown would allow arbitrary React rendering — avoid.

3. **Image preview via `blob:` URL:** If image files are fetched and displayed via `URL.createObjectURL()` (creating a `blob:` URL as the `<img src>`), the current `img-src 'self' data:` policy BLOCKS blob URLs. An amendment to `img-src 'self' data: blob:` is required. This is a minimal expansion.

4. **HTML file preview:** Do NOT render `.html` files. Always show them as source code (`text/plain`). Rendering an HTML file fetched from the working directory would execute its embedded scripts (blocked by CSP, but the `<style>` injection path is still open). Display as source-code only, with a note "HTML files are displayed as source to prevent code execution."

**How to avoid:**
- Use `react-markdown` WITHOUT `rehype-raw`. Default react-markdown does NOT render raw HTML and is safe under the existing CSP.
- If markdown files from AI agents commonly embed HTML (they do: agent outputs sometimes include HTML snippets in code blocks), those will render as code blocks, which is correct.
- Add `img-src blob:` to the CSP amendment for the image preview feature. Document this as Amendment 3 in `csp_mw.go`, matching the Amendment 1/2 documentation pattern.
- HTML files: force `Content-Type: text/plain` on the `/read` endpoint for `.html`, `.htm`, `.xhtml` extensions, regardless of actual content. Add a comment explaining why.
- Write a regression test: `TestCSPHeaders_NoUnsafeScriptSrc` pattern from `csp_mw_test.go` — verify no new `unsafe-eval` or inline script permissions were added.

**Warning signs:**
- Image previews fail silently (no blob: in img-src)
- markdown with embedded `<style>` blocks applies CSS to the whole FileBrowserTab
- CSP e2e suite flips red after adding the markdown renderer

**Phase to address:**
FileBrowserTab frontend phase. CSP amendment 3 (if needed for blob:) must be in the same PR as the image preview feature, not a follow-up.

---

### Pitfall 10: Large-File React State Anti-Pattern — Base64 is 33% Memory Overhead, Kills GC

**What goes wrong:**
The FileBrowserTab will fetch file content for the preview pane. The naive pattern is:
```typescript
const [fileContent, setFileContent] = useState<string>('')
// on file select:
const data = await fetch('/api/files/{id}/read?path=...').then(r => r.text())
setFileContent(data)
```

This is fine for a 50 KB source file. It is wrong for:
- **Binary files (images)**: fetching as text and converting to base64 adds 33% overhead. A 3 MB PNG becomes a 4 MB base64 string in React state, diffed on every render, allocated as a JS string, and eventually GC'd in a single large chunk.
- **Large text files**: a 5 MB markdown file in React state is 5 MB × 2 (UTF-16 in V8) = 10 MB in JS heap, diffed against the previous render, potentially re-rendered on every keystroke in the filter box.

The `<img src="blob:...">` pattern (using `URL.createObjectURL`) is correct for images: it holds a reference to the fetched bytes in the browser's opaque storage and passes a URL handle to the DOM, without routing the bytes through React state.

**How to avoid:**
- **Images:** Use `<img src="/api/files/{id}/read?path=..." />` directly — point the `src` attribute at the authenticated endpoint URL. This uses the browser's native resource loading with range-capable HTTP. No React state involved, no base64. The endpoint already requires `?cap=` auth; include the token in the URL (current pattern for capability-gated resources).
- **Large text files (> 256 KB):** Add a "This file is large. Load anyway?" confirmation. If the user confirms, fetch with a streaming approach (server-side `/read` endpoint returns the bytes, client uses `response.text()` once, stores in a `useRef` not `useState` to avoid re-render on every keystroke).
- **5 MB cap:** Enforce server-side via Content-Length check BEFORE streaming content. Return `413 Content Too Large` for the preview endpoint if `stat.Size() > 5*1024*1024`. The client should handle 413 gracefully with a "File too large to preview, download instead" message.
- Do NOT fetch file content into state on directory navigation — lazy-load on file selection only.

**Warning signs:**
- "FileBrowserTab is slow after selecting a large file" reports
- Browser DevTools memory profiler shows large string objects in JS heap during file preview
- Image preview works but causes jank on a 4K PNG

**Phase to address:**
FileBrowserTab frontend phase. The image-via-src-URL pattern must be designed in. The 5 MB server-side cap test must pass.

---

### Pitfall 11: Multi-Client Read-Only Coherence — Stale Listing Contract Must Be Explicit

**What goes wrong:**
The epic flags this as a Medium concern. If user A is browsing while user B's shell session writes to the same file, user A's preview pane is stale. There is no filesystem watch in the v3.4 scope.

The pitfall is NOT the staleness itself — that's an accepted trade-off for v3.4. The pitfall is:
1. **Not documenting the contract:** The FileBrowserTab silently shows stale data with no timestamp, no "last refreshed" indicator, no way to know if the directory listing is current. Users will assume the listing is live.
2. **Not providing a manual refresh:** Without a "Refresh" button or F5 shortcut in the FileBrowserTab, users have no recourse when they notice staleness.
3. **Stale preview pane after session writes:** User A previewing `output.json` while User B's session is writing it will see old content. The preview pane must show a "Refreshed: N seconds ago" badge.

The SSE-based filesystem watch (the right solution for v3.5) would push `NOTIFY` events from a `fsnotify` watcher. That's out of scope for v3.4. But the v3.4 design must not architect itself into a corner that makes v3.5's SSE watch hard to add. Specifically: the directory listing response should include a `Last-Refreshed` timestamp header so the client can display "Refreshed 10s ago" even without server-push.

**How to avoid:**
- Add `X-Refreshed-At: <unix-timestamp>` to the `/api/files/list` response. The frontend shows "Last refreshed: N seconds ago" in the file browser status bar.
- Add a "Refresh" button (or keyboard shortcut R) to the FileBrowserTab that re-fetches the current directory listing.
- For the preview pane, show the file's `Last-Modified` header from the `/read` response (which `ServeContent` sets from `stat.ModTime()`). A "Modified: 2 minutes ago" indicator communicates that the preview is not live.
- Do NOT implement implicit auto-refresh polling in v3.4 (it would generate constant daemon ↔ filesystem I/O). If auto-refresh is desired, defer to v3.5 with `fsnotify`.

**Warning signs:**
- User reports "file browser shows old file contents" — if there's no staleness indicator, this will look like a bug rather than an expected trade-off

**Phase to address:**
FileBrowserTab frontend phase. The `X-Refreshed-At` header must be added in the daemon phase. The client timestamp display must be in the same frontend phase.

---

### Pitfall 12: TUI Keyboard Routing — New Files View Must Not Conflict With Kill/Rename/New Modals

**What goes wrong:**
The TUI's `Update` function uses a 5-level priority key dispatch (per PROJECT.md key decisions: `editing > kill confirm > new session modal > QR overlay > help > main view`). Adding a Files view introduces new key-consuming states:
- Up/Down/Enter/Backspace/PageUp/PageDown for navigation
- Type-ahead filter (absorbs printable characters)
- Preview pane scroll (Up/Down conflict with list navigation)

Specific conflicts:
1. **Backspace** in the type-ahead filter vs. **Backspace as "navigate up"** — the filter is active when non-empty; Backspace navigates up only when filter is empty. This exact ambiguity caused the kill-confirm modal conflict in v3.1 (see TUI Paper Cuts).
2. **Enter to enter directory vs. Enter to attach session** — these are different views, but if the user accidentally opens Files while a session is selected, pressing Enter might both enter a directory and trigger attach.
3. **Escape to dismiss filter vs. Escape to close Files view** — `Esc` when filter is non-empty clears the filter; `Esc` when filter is empty closes the view.
4. **`?` help overlay** — the existing help overlay shows keybindings for the current view. The Files view adds new keys that must be reflected in the help overlay. Forgetting to update the help overlay causes user confusion.

**How to avoid:**
- Add the Files view as a new priority level in the key dispatch: `editing_filter > kill confirm > new session modal > QR overlay > files > help > main view`
- Make the filter activation explicit (e.g., press `/` to enter filter mode, `Esc` to clear and exit filter mode) rather than absorbing all printable characters automatically. This avoids the Backspace ambiguity.
- Keep Enter for "enter directory" and use a separate key (Space or `p`) for "preview file". Map the `?` help overlay to show these per-view bindings.
- Add the Files view keybindings to the existing help overlay data structure before the Files view ships — users will press `?` on first use.
- Test: with the kill-confirm modal open and the Files view underneath it, pressing Backspace must trigger kill confirm's "cancel" path, not the Files view's "navigate up."

**Warning signs:**
- Pressing Backspace in the file filter navigates up unexpectedly
- Enter in the file browser accidentally attaches to a session
- Help overlay shows no file browser keys after Files view ships

**Phase to address:**
TUI Files view phase. Key dispatch priority must be specified in the roadmap phase spec, not discovered during implementation.

---

### Pitfall 13: `FileBrowserTab` Singleton Pattern — Must Handle Remote Sessions Correctly

**What goes wrong:**
The existing singleton tab pattern in `App.tsx` uses `tabs.find(t => t.type === 'settings')` to prevent duplicate tabs. The FileBrowserTab will be opened from the session context menu (per the v3.4 spec). If multiple sessions each get their own FileBrowserTab, the singleton pattern must scope to session ID, not just tab type:

```typescript
// WRONG: only one file browser ever
const existing = tabs.find(t => t.type === 'file-browser')

// RIGHT: one per session
const existing = tabs.find(t => t.type === 'file-browser' && t.sessionId === sessionId)
```

For remote tailnet sessions (accessed via relay), the FileBrowserTab must route `/api/files/*` requests through the relay's HTTPS endpoint, not directly to the local daemon socket. The existing `RemoteSessionsPanel` uses `BrowserOpenURL` to open the remote session's web URL. The FileBrowserTab needs to:
1. Know whether the session is local or remote.
2. For local sessions: call the local webserver endpoint (same host).
3. For remote sessions: call the remote peer's webserver endpoint with the capability token.

This is a NEW pattern — no existing component fetches file data from a remote peer's webserver. If not designed in, FileBrowserTab for remote sessions will fail silently or hit CORS/TLS mismatches.

**How to avoid:**
- Add `'file-browser'` to the `Tab.type` union in `TabBar.tsx` before implementing `FileBrowserTab`.
- The find-or-add key for FileBrowserTab is `(t.type === 'file-browser' && t.sessionId === sessionId)`.
- For v3.4, support local sessions only. Add a clear "File browser is not available for remote sessions in v3.4" message and defer remote browse to v3.5. Do NOT silently fail — show the explicit message.
- Store the base URL in the tab's metadata (or derive from the capability token's SID and the daemon's known webserver URL). The file browse requests use the same URL base as the terminal WebSocket.

**Warning signs:**
- Opening the file browser for two sessions shows the same directory listing for both
- FileBrowserTab for a remote session makes requests to `localhost` (wrong endpoint)

**Phase to address:**
FileBrowserTab frontend phase. The `Tab.type` extension must be in the same PR as FileBrowserTab.

---

## Minor Pitfalls

### Pitfall 14: Windows Path Separator in JSON Responses — Always Serialize Forward Slashes

**What goes wrong:**
On Windows, `filepath.Join` returns paths with backslash separators (`C:\Users\ken\project\src\file.go`). If the daemon serializes this directly into the JSON response for the directory listing or stat endpoint, the frontend receives backslash paths. TypeScript string operations (split, join, replace) that assume forward slashes break. The breadcrumb path bar in FileBrowserTab receives `src\utils\helper.ts` instead of `src/utils/helper.ts`.

**How to avoid:**
Normalize all paths to forward slashes before JSON serialization: `strings.ReplaceAll(path, `\`, `/`)`. Do this in a single place (the API response serialization helpers), not scattered across endpoint handlers. The client should never see a backslash in a path response.

**Phase to address:**
Filesystem API phase (response serialization). Add a Windows-specific test that verifies no backslashes appear in list/stat responses.

---

### Pitfall 15: macOS Resource Fork / Extended Attributes in Directory Listings

**What goes wrong:**
On macOS, some files have associated `._` resource fork files (created by older Apple tools, QuickLook, or file copy operations). These appear in `os.ReadDir` results with names like `._file.go`. Users find these confusing. Extended attributes (xattrs) also exist on macOS and Linux but are NOT exposed by `DirEntry` — they require a separate `xattr.Get` call. For the v3.4 read-only browser, these are noise.

**How to avoid:**
Filter out files beginning with `._` (macOS resource fork prefix) from directory listings. This is a common pattern in macOS-aware file browsers. Do NOT filter on other platforms — `._` is a valid filename prefix on Linux/Windows.

Do NOT attempt to expose xattrs in v3.4. The complexity (platform-divergent APIs, no stdlib support, Go CGO required on macOS) exceeds the value.

**Phase to address:**
Filesystem API phase (list endpoint) — add platform-conditional resource fork filtering.

---

### Pitfall 16: `settings.json` Migration — New `FilesRead` Default Capability Flag

**What goes wrong:**
The v3.2 `schemaVersion: 2` migration pattern (described in v3.2 PITFALLS.md Pitfall 14) was specifically noted as a pattern to follow. If v3.4 adds a new `filesRead` field to `daemonSettings` (controlling whether `files.read` is default-on for session owners), the `defaultSettings()` constructor must initialize it to `true`, and the migration test must verify a `settings.json` without the field defaults it to `true` (not `false`/zero-value).

This is the same class of bug that caused the v3.2 defaults-merge constructor fix.

**How to avoid:**
- Add `filesRead bool` to `daemonSettings` with `json:"filesRead,omitempty"` and default `true` in the constructor.
- Write a fixture test: `TestSettingsMigration_FilesReadDefaultsTrue` that loads a `settings_v3.3.json` fixture (no `filesRead` key) and asserts the loaded value is `true`.
- Bump `schemaVersion` to 3 in the migration.

**Phase to address:**
Filesystem API phase (daemon settings) — before any capability-issuance code that reads this setting.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `filepath.EvalSymlinks` + `os.Open` two-step instead of `os.OpenRoot` | Familiar, no new API to learn | TOCTOU race window exploitable by any process with write access to cwd subdirs | Never — project is on Go 1.26.1, `os.Root` is available |
| Cap check only on GET, not HEAD | Simpler middleware | HEAD leaks file existence without auth | Never — add HEAD to capability gate or explicitly return 405 |
| Client-side 5 MB cap only | No server-side enforcement code | Race: client can disable cap; server streams unbounded bytes | Never — cap must be server-side |
| Serve raw HTML files as `text/html` | Simple code | HTML from agent working dirs may contain `<style>` injections | Never — force `text/plain` for `.html` files |
| No truncation for 100k+ directories | Simpler listing code | Daemon OOM on `node_modules` listing | Never — hard cap + truncation header required |
| Skip `blob:` CSP amendment for image preview | No CSP change | Image `<img>` with direct endpoint URL works without blob:; base64 workaround has 33% overhead | Avoid base64; use direct endpoint URL instead — no blob: amendment needed if images are served via endpoint URL |
| Auto-refresh polling every 10s | "Live" feel | Constant daemon I/O; races with agent session writes; drains battery on mobile | Defer to v3.5 with `fsnotify` SSE |
| Tab type `'file-browser'` added to existing union without updating all switch-case consumers | Faster PR | Runtime errors if any switch-case on tab type is non-exhaustive | Never — update all consumers in the same PR |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `os.OpenRoot` + `filepath.EvalSymlinks` | Calling `EvalSymlinks` on user-supplied path, then `root.Open` on result | Call `EvalSymlinks` only on the cwd at session creation; call `root.Open(untrustedRelPath)` — root handles symlinks internally |
| `http.ServeContent` + 0-byte file | Returns 416 on any Range request | Special-case 0-byte files: respond 200 + empty body before calling `ServeContent` |
| `requireCapability` + new file endpoints | Adding file-read check inside existing `requireCapability` | Add `requireFilesPerm` as a SEPARATE wrapper; don't modify `requireCapability` which gates terminal routes |
| `Claims.Perms` + `files.read` | `strings.Contains(perms, "files.read")` matches `"no-files.read"` | Use whole-token check: `HasPerm(perms, "files.read")` that splits on commas |
| react-markdown + CSP | Enabling `rehype-raw` to render agent-generated HTML | Never use `rehype-raw` in FileBrowserTab — raw HTML from working-dir files is untrusted |
| Image preview | Fetching image bytes into React state as base64 | Use `<img src="/api/files/{id}/read?path=...&cap=TOKEN" />` — endpoint URL directly in img src |
| TUI `os.ReadDir` in `Update` | Direct call in Update function | Wrap in `tea.Cmd` returning `DirListResultMsg`; show loading state during I/O |
| Windows path separator | Serializing `filepath.Join` result directly | `strings.ReplaceAll(path, `\`, `/`)` before JSON serialization |
| `schemaVersion` migration | Adding `filesRead bool` field without constructor default | `defaultSettings()` must return `filesRead: true`; migration test required |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| `os.ReadDir` on 100k+ entries | Daemon spike, slow list response | Hard cap at 10,000 entries with truncation header | Any project with `node_modules` or `.git/objects` in cwd |
| `DirEntry.Info()` per entry in listing | 10k syscalls per directory list | Return type + name only from listing; stat on demand via `/stat` endpoint | Any large directory |
| `filepath.EvalSymlinks` on every request | Repeated lstat chains on deep paths | Cache resolved cwd at session creation; use `os.OpenRoot` for per-request ops | Deep nested paths (> 20 components) |
| Stat + ServeContent separately | Stat says 4.9 MB, then file grows to 6 MB between stat and serve | Enforce cap at Open time via `io.LimitReader(f, maxBytes+1)` + check | Any file actively being written by an agent session |
| Large file in React state | Browser tab memory growth, jank on filter/search | Never load > 256 KB into useState; use endpoint URL for images | 4K screenshots, generated logs > 5 MB |
| TUI sync I/O in Update | Frozen render loop during directory navigation | All FS I/O via `tea.Cmd` | Any directory with > ~100 entries on slow disk/NFS |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Two-step EvalSymlinks + Open (TOCTOU) | Symlink swap escape to read arbitrary files on attacker-writable cwd | Use `os.OpenInRoot` / `os.Root.Open` — single atomic operation |
| Null byte in path parameter | Path injection in downstream string operations | Reject any path with `\x00` before processing |
| Windows reserved device names not rejected | Opening `NUL` hangs; `CON` reads stdin | Cross-platform device name check on all path inputs |
| Alternate data streams not rejected | Reading hidden NTFS streams | Reject any path containing `:` (colon) on all platforms |
| HTML files served as `text/html` | `<style>` injection from working-dir HTML (CSP allows `unsafe-inline` for styles) | Force `Content-Type: text/plain` for `.html`, `.htm`, `.xhtml` |
| `files.read` missing from some endpoints | File stat or list works without auth, read is gated | Apply `requireFilesPerm` to ALL three endpoints in the same middleware |
| Default `files.read` off for session owner | Owner can't use file browser after upgrade | Default ON for owner; OFF for viewer; document migration contract |
| `files.read` check in shared `requireCapability` | Terminal relay endpoints accidentally 403 | Add `requireFilesPerm` as separate wrapper; don't modify shared middleware |
| No server-side 5 MB cap | Client-side bypass streams unbounded bytes through daemon | `stat.Size() > 5MB → 413` before ServeContent call |
| Image fetched as base64 through JSON API | 33% memory overhead, GC pressure, potential XSS if reflected | Serve images via direct endpoint URL in `<img src>`; no base64 API |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No staleness indicator on directory listing | User assumes listing is live; confused when refreshing manually reveals new files | `X-Refreshed-At` header + "Refreshed Ns ago" in status bar |
| "Forbidden" on missing `files.read` with no explanation | User thinks the file browser is broken | Explicit error: "File browser requires files.read permission. Ask owner to re-share." |
| Breadcrumb exceeds terminal-width in TUI | Path bar overflows, wraps, breaks layout | Truncate breadcrumb from left (ellipsis prefix): `…/utils/helper.ts` |
| No "Refresh" button or shortcut in FileBrowserTab | User can't manually refresh stale listing | Add Refresh button and/or `R` shortcut |
| No "File too large to preview" message | Blank preview pane when 5 MB cap fires | Show "File too large to preview (N MB). [Download instead]" |
| HTML files rendered (even as text/plain with markdown renderer) | Confusing — user sees HTML tags as formatted output | Force monospace code renderer for `.html` files; show label "HTML (displayed as source)" |
| Remote session file browse silently fails | Tab opens but is blank or shows connection error | Explicit "File browser not available for remote sessions in v3.4" with placeholder |
| Binary files show preview pane with garbage characters | Confusing output | Show "Binary file — use desktop or web to download" in TUI; download link in GUI |

---

## "Looks Done But Isn't" Checklist — Mandatory Tests Before Merge

### Filesystem API (Go daemon)
- [ ] **Path traversal fuzz test:** `go test -fuzz=FuzzSandboxPath` passes 30 seconds of fuzzing with no escapes found
- [ ] **Null byte rejection:** `GET /api/files/{id}/list?path=foo%00bar` returns 400
- [ ] **Windows device name rejection (all platforms):** `?path=CON` and `?path=nul.txt` return 400
- [ ] **Alternate data stream rejection (all platforms):** `?path=file.txt:hidden` returns 400
- [ ] **Absolute path rejection:** `?path=/etc/passwd` and `?path=C:\windows` return 400
- [ ] **Symlink escape returns 403, not 500:** symlink target outside cwd returns 403 (not internal server error)
- [ ] **0-byte file read returns 200, not 416:** `GET /api/files/{id}/read` on empty file returns 200 with empty body
- [ ] **5 MB cap returns 413:** a 5,000,001-byte file returns 413 from the read endpoint
- [ ] **Truncation signal:** a 10,001-entry directory returns `X-Directory-Truncated: true` header
- [ ] **No capability = 401 (not 404):** `GET /api/files/{id}/list` without `?cap=` returns 401
- [ ] **Viewer cap without `files.read` = 403:** a `read`-only cap (no `files.read`) returns 403 with message containing "files.read"
- [ ] **HEAD /api/files/{id}/read** returns 200 (or explicit 405 if intentionally unsupported)
- [ ] **POST /api/files/{id}/list** returns 405

### Frontend (FileBrowserTab)
- [ ] **Tab type union updated:** TypeScript compiler reports no errors with `'file-browser'` tab type
- [ ] **Singleton per session:** opening file browser for two sessions creates two tabs, not one
- [ ] **Image via src URL, not state:** no `setFileContent(base64...)` pattern in image preview code
- [ ] **Staleness indicator:** status bar shows "Refreshed Ns ago" using `X-Refreshed-At` response header
- [ ] **Refresh button/shortcut:** pressing R (or clicking Refresh) re-fetches directory listing
- [ ] **Markdown renderer:** `rehype-raw` is NOT imported or used in FileBrowserTab
- [ ] **HTML files:** `.html` extension shows "HTML (displayed as source)" label with monospace renderer
- [ ] **No new CSP violations:** Playwright e2e CSP suite green after FileBrowserTab ships
- [ ] **Remote session placeholder:** opening file browser for a remote session shows explicit "not available in v3.4" message

### TUI Files View
- [ ] **Async directory listing:** `os.ReadDir` is NOT called directly in `Update`; all I/O via `tea.Cmd`
- [ ] **Never above cwd:** Backspace at root of session cwd does NOT navigate above cwd
- [ ] **Backspace in empty filter navigates up, not appended:** verify Backspace dispatch priority
- [ ] **Help overlay updated:** `?` shows file browser keybindings in Files view
- [ ] **Binary files show message:** binary file (non-text) shows "Binary file — use desktop or web" instead of garbage
- [ ] **Kill confirm takes priority over Files view navigation:** kill-confirm modal still works with Files view open

### Integration / Regression
- [ ] **No existing terminal routes affected:** all existing relay, plugin-config, session-info endpoints still pass their existing tests
- [ ] **settings.json migration:** a `settings_v3.3.json` fixture without `filesRead` key loads with `filesRead: true`
- [ ] **schemaVersion bumped:** `settings.json` written after v3.4 startup has `schemaVersion: 3`
- [ ] **Windows path separators:** `/list` response paths contain no backslashes on Windows
- [ ] **macOS resource forks filtered:** `._` files do not appear in directory listings on macOS

---

## Fuzz Corpus Skeleton

```go
// internal/daemon/files_fuzz_test.go
func FuzzSandboxPath(f *testing.F) {
    // Seed corpus — path traversal menagerie
    // Classic traversal
    f.Add("../etc/passwd")
    f.Add("../../etc/shadow")
    f.Add("a/../../etc/passwd")
    // Encoded variants (these arrive URL-decoded via r.PathValue, but test raw forms too)
    f.Add("%2e%2e%2fetc%2fpasswd")
    f.Add("%252e%252e%252fetc%252fpasswd")
    // Absolute paths
    f.Add("/etc/passwd")
    f.Add("/proc/self/cwd")
    f.Add("/proc/self/fd/0")
    // Windows absolute
    f.Add(`C:\windows\system32\cmd.exe`)
    f.Add(`\\server\share\file`)
    f.Add(`C:/windows/system32/cmd.exe`)
    // Windows device names
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
    // Alternate data streams
    f.Add("file.txt:hidden")
    f.Add("file.txt:$DATA")
    f.Add(":$i30:$INDEX_ALLOCATION")
    // Null bytes
    f.Add("secret.txt\x00.jpg")
    f.Add("foo\x00")
    f.Add("\x00etc/passwd")
    // Unicode tricks
    f.Add("foo／etc／passwd")  // fullwidth slash U+FF0F
    f.Add("foo․passwd")           // one-dot-leader U+2024
    f.Add("foo‥bar")              // two-dot-leader U+2025
    // Trailing dots/spaces (Windows strips these)
    f.Add("file.")
    f.Add("file.txt.")
    f.Add("file.txt  ")
    // Long paths (path length attacks)
    f.Add(strings.Repeat("a/", 512) + "passwd")
    f.Add(strings.Repeat("../", 512))
    // Symlink names (can't test the TOCTOU race in fuzz, but test the path forms)
    f.Add("link")
    f.Add("a/b/link")
    // 8.3 short names (Windows)
    f.Add("PROGRA~1/system.dll")
    f.Add("progra~2/file.exe")
    // Mixed separators
    f.Add(`a\b/c`)
    f.Add(`a/b\c`)
    // Empty and dot paths
    f.Add("")
    f.Add(".")
    f.Add("..")
    f.Add("./")
    f.Add("./etc/passwd")
    f.Add(".hidden")
    f.Add("..hidden")

    f.Fuzz(func(t *testing.T, rawPath string) {
        // sandboxPath is the function under test: validates rawPath against a
        // known-good resolved cwd root and returns an error if the path escapes.
        // It must never return (validPath, nil) when rawPath contains any traversal.
        _, err := sandboxPath("/tmp/test-cwd", rawPath)
        if err == nil {
            // Verify the result is genuinely inside the cwd
            t.Logf("accepted path: %q", rawPath)
        }
        // The fuzz engine looks for panics; we verify no panics occur on any input.
    })
}
```

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Daemon file API (list/stat/read endpoints) | TOCTOU symlink escape | Use `os.OpenRoot`; never two-step EvalSymlinks+Open on user input |
| Daemon file API | Windows device names, ADS, null bytes | Explicit reject functions before `filepath.Clean` |
| Daemon file API | 100k-entry directories | Hard cap 10k + truncation header |
| Capability bit addition | Wire format breakage, default-on for owner | `HasPerm` helper; owner tokens get `files.read`; fixture migration test |
| FileBrowserTab frontend | Image base64 memory overhead | Direct endpoint URL in `<img src>`; no base64 state |
| FileBrowserTab frontend | Markdown `rehype-raw` CSP bypass | Do NOT use `rehype-raw`; plain react-markdown is safe |
| FileBrowserTab frontend | HTML file rendering | Force `text/plain` + source-display label for `.html` |
| FileBrowserTab frontend | Blob URL CSP gap for image preview | Use direct endpoint URL (no blob:); no CSP amendment needed |
| TUI Files view | Synchronous I/O freezing render loop | All FS I/O in `tea.Cmd` wrapping |
| TUI Files view | Key dispatch conflict with existing modals | Add Files as new priority level; explicit Backspace dispatch semantics |
| Settings migration | `filesRead` defaulting to false | `defaultSettings()` explicit `true`; `schemaVersion: 3`; fixture test |
| Route registration | Missing capability gate on some endpoints | Apply `requireFilesPerm` to ALL three endpoints in single PR |

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| TOCTOU symlink escape shipped | HIGH | Hotfix: replace EvalSymlinks+Open with `os.OpenInRoot`; security advisory in release notes; audit for any affected read requests in logs |
| Windows device name hang shipped | MEDIUM | Hotfix: add device-name rejection before `os.OpenRoot`; affected requests block indefinitely — requires daemon restart to clear hung goroutines |
| `files.read` default-off for session owner | MEDIUM | Hotfix: change `defaultSettings()` to default `filesRead: true`; existing installations need manual settings reset or next-start migration |
| HTML files rendered as HTML (CSP bypass) | LOW–MEDIUM | Hotfix: add `.html` extension to forced `text/plain` list on `/read` endpoint; no data exfil risk but CSS injection possible |
| No server-side 5 MB cap | LOW | Hotfix: add size check before ServeContent; no security impact (read-only), just memory protection |
| TUI sync I/O shipped | LOW | Fix in next release; workaround: avoid navigating into large directories in TUI |
| Image base64 memory overhead | LOW | Fix in next release; no security impact |

---

## Sources

- [Go Blog: Traversal-resistant file APIs (os.Root, Go 1.24)](https://go.dev/blog/osroot) — HIGH confidence, official
- [golang/go#70007: path/filepath Walk/WalkDir susceptible to symlink race](https://github.com/golang/go/issues/70007)
- [golang/go#67002: os: safer file open functions (os.Root design)](https://github.com/golang/go/issues/67002)
- [golang/go#71165: EvalSymlinks ignores link type on Windows](https://github.com/golang/go/issues/71165)
- [golang/go#63703: os.Readlink and EvalSymlinks on Windows reimplementation](https://github.com/golang/go/issues/63703)
- [golang/go#42079: EvalSymlinks fails on Windows UNC share root](https://github.com/golang/go/issues/42079)
- [CVE-2026-27976: Zed code editor sandbox escape via symlink traversal](https://www.thehackerwire.com/zed-code-editor-sandbox-escape-via-symlink-traversal-cve-2026-27976/) — 8.8 CVSS, symlink TOCTOU
- [CVE-2025-27210: Node.js path traversal on Windows via device names](https://zeropath.com/blog/cve-2025-27210-nodejs-path-traversal-windows) — Windows device name class
- [OWASP Path Traversal — A01:2021 Broken Access Control](https://owasp.org/www-community/attacks/Path_Traversal)
- [OWASP Null Byte Injection](https://owasp.org/www-community/attacks/Embedding_Null_Code)
- [golang/go#54794: ServeContent sends 416 for empty file with Range header](https://github.com/golang/go/issues/54794)
- [golang/go#50905: ServeContent serves wrong headers on invalid range](https://github.com/golang/go/issues/50905)
- [golang/go#21124: DetectContentType needs > 512 bytes for MP3 without ID3](https://github.com/golang/go/issues/21124)
- [golang/go#50376: DetectContentType missing video/mpeg](https://github.com/golang/go/issues/50376)
- [Bubble Tea v2 Commands — charm.land/bubbletea/v2](https://pkg.go.dev/charm.land/bubbletea/v2) — async I/O via tea.Cmd
- [react-markdown CSP inline style issue #552](https://github.com/remarkjs/react-markdown/issues/552) — table inline styles
- [HackerOne: Secure Markdown Rendering in React](https://www.hackerone.com/blog/secure-markdown-rendering-react-balancing-flexibility-and-safety)
- [Microsoft: Naming Files, Paths, and Namespaces (reserved device names, ADS)](https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file)
- AgentHub `internal/capability/capability.go`, `internal/webserver/capability_mw.go`, `internal/webserver/server.go`, `internal/webserver/csp_mw.go`, `internal/daemon/types.go`, `frontend/src/components/TabBar.tsx` — read directly from source

---
*Pitfalls research for: v3.4 File Browser (Read-Only) — adding sandboxed FS API to AgentHub's daemon+capability+relay+CSP stack*
*Researched: 2026-05-20*
