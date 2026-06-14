# Pitfalls Research — v3.5 Write-Side File Browser + In-App Code Editor

**Domain:** Adding write operations (create/save/upload/delete/rename/mkdir) and an in-app code editor to an existing sandboxed read-only file browser in a Go/Wails daemon + React + Bubble Tea v2 desktop app with Tailscale web-share capability surface
**Researched:** 2026-06-14
**Confidence:** HIGH for Go stdlib write API behavior (verified against source + docs); HIGH for AgentHub-specific integration (read from live source); HIGH for multipart/upload abuse patterns (well-documented class); HIGH for `os.Root` write API gaps (Go 1.24+ official docs + issue tracker); MEDIUM for CodeMirror/Monaco WebView quirks (verified class, not exhaustively source-confirmed for this exact Wails version); MEDIUM for CSRF patterns over capability-token auth (well-documented class, confirmed applicable)

> **Scope discipline:** Every pitfall below is specific to ADDING write operations to AgentHub's existing v3.4 `os.OpenRoot`-sandboxed read-only filesystem API and FileBrowserTab/TUI Files view. Generic "validate input" advice from v3.4 research is not repeated. The v3.4 PITFALLS.md covered the read surface in depth; this document covers what changes and what is NEW when writes are introduced. Read-side pitfalls that are equally applicable to writes (e.g., path traversal, device names) are referenced, not rewritten.

---

## Critical Pitfalls

### Pitfall 1: `os.Root` Write API Has Gaps — `Rename` Is NOT Available on `*os.Root`

**What goes wrong:**
The v3.4 `Sandbox` uses `os.OpenRoot(rootPath)` which returns `*os.Root`. The v3.4 read surface uses `root.Open(relPath)`, `root.Stat(relPath)`, and `f.ReadDir(n)` — all available on `*os.Root`. For writes, the natural assumption is that `*os.Root` exposes a full filesystem API. It does not. As of Go 1.24/1.26, `*os.Root` exposes:

- `root.Open(relPath)` — open for reading
- `root.OpenFile(relPath, flag, perm)` — open with flags including `os.O_WRONLY|os.O_CREATE`
- `root.Create(relPath)` — shorthand for `OpenFile(name, O_RDWR|O_CREATE|O_TRUNC, 0666)`
- `root.Mkdir(relPath, perm)` — create one directory level
- `root.Stat(relPath)` and `root.Lstat(relPath)`
- `root.Remove(relPath)` — remove file or empty directory
- `root.RemoveAll(relPath)` — remove tree (Go 1.24+)

**Missing from `*os.Root`:**
- `root.Rename(oldRelPath, newRelPath)` — does NOT exist. Rename requires `os.Rename(src, dst)` with full absolute paths, which bypasses the sandbox. This is a critical gap for the rename operation.
- `root.MkdirAll(relPath, perm)` — does NOT exist. Only `root.Mkdir` (single level).
- `root.WriteFile(relPath, data, perm)` — does NOT exist; must use `root.OpenFile` + Write.

**Why it happens:**
Developers assume `*os.Root` mirrors the `os` package API. The Go team intentionally restricted the initial API surface; `Rename` across roots is semantically complex. See golang/go#67002 and golang/go#69462 for the ongoing design discussion.

**Consequences of naive workaround:**
- Implementing rename as `os.Rename(absOld, absNew)` is correct ONLY if BOTH paths are derived by appending the validated relative path to the sandbox root's absolute path. If either path is assembled from user input without going through `validateAndClean` first, the rename target can escape the sandbox.
- The rename target (destination path) is a NEW path that must be independently validated — not just the source. Developers often validate the source (the existing file) and forget to validate the destination (the new name/location).

**How to avoid:**
- For rename: validate BOTH source and destination relative paths through `validateAndClean` (the existing v3.4 function in `sandbox.go`), THEN construct absolute paths as `filepath.Join(s.rootPath, cleanedSrc)` / `filepath.Join(s.rootPath, cleanedDst)`. Verify the joined path still starts with `s.rootPath` as a defense-in-depth check before calling `os.Rename`.
- Add `Rename(oldRelPath, newRelPath string) error` to `Sandbox` — do NOT expose raw `os.Rename` anywhere in the handler. The `Sandbox` struct is the correct abstraction boundary.
- For mkdir: `root.Mkdir` creates one level; to mimic `os.MkdirAll` across multiple new levels, decompose the relative path and call `root.Mkdir` for each missing component. Do NOT call `os.MkdirAll` on a constructed absolute path (the `root.Mkdir` chain is TOCTOU-safe; `os.MkdirAll` with constructed paths re-introduces the race).
- Add `MkdirAll(relPath string, perm os.FileMode) error` to `Sandbox` as a helper that calls `root.Mkdir` iteratively.

**Warning signs:**
- Any `os.Rename` call that takes a path derived from user input without sandbox validation
- Any `os.MkdirAll` in a write handler that constructs an absolute path from user input
- Rename succeeds for a path that contains `../` in the destination

**Phase to address:**
Write-side sandbox extension (the first write-operation phase). `Sandbox.Rename` and `Sandbox.MkdirAll` must be implemented in `internal/files/sandbox.go` in the same phase as the first write endpoint. The fuzz corpus must include rename-target traversal payloads.

---

### Pitfall 2: Rename Target Path Escaping the Sandbox — The Destination Is a Second Attack Vector

**What goes wrong:**
Path traversal research for v3.4 focused on the source file being read. Write operations introduce a second attack vector: the destination of a rename or the path of a newly created file. An attacker (or a misconfigured client) can supply:

```
POST /api/files/rename?session=X&from=foo.txt&to=../../.ssh/authorized_keys
```

The rename source (`foo.txt`) validates cleanly. The rename target (`../../.ssh/authorized_keys`) traverses out of the sandbox. Even with `os.Rename` taking absolute paths, if the destination absolute path is constructed from `filepath.Join(rootPath, "../../.ssh/authorized_keys")`, `filepath.Join` cleans the path and resolves it to `/home/user/.ssh/authorized_keys`, which is outside the sandbox.

This is distinct from read-side traversal because:
1. **The consequence is arbitrary file creation/modification**, not just reading. Overwriting `~/.ssh/authorized_keys` or `~/.bashrc` or `~/.claude/settings.json` grants persistent code execution.
2. **Rename is atomic** — there is no partial state if the destination is wrong. One successful request writes the payload.
3. **The rename destination cannot be validated by opening it** (it doesn't exist yet). A destination path must be validated by string analysis before the operation.

**Why it happens:**
Read-side validation often only validates "can I open this path?". For writes, developers add write flags to the open call and consider themselves done. They do not separately validate the rename destination or the mkdir path.

**How to avoid:**
- Every user-supplied path parameter for write operations must pass through `validateAndClean` before any filesystem operation. This includes: `to` in rename, `path` in mkdir, `path` in create/write/upload, `path` in delete.
- After `validateAndClean`, additionally verify the joined absolute path starts with `s.rootPath + string(os.PathSeparator)` (or equals `s.rootPath`). This is the defense-in-depth catch for any gap in `validateAndClean` that a future code change might introduce.
- For rename specifically: the destination parent directory must exist within the sandbox. Validate that `filepath.Dir(absDestination)` is within the sandbox before calling `os.Rename`. Do NOT silently create parent directories on rename — require the parent to already exist.
- Extend the fuzz corpus with rename destination traversal payloads:
  - `../../.ssh/authorized_keys`
  - `../../.bashrc`
  - `../../../etc/cron.d/pwn`
  - `../../.claude/settings.json`

**Warning signs:**
- Rename endpoint that validates `from` path but takes `to` as an unvalidated string
- Any `filepath.Join(rootPath, userInput)` where `userInput` comes directly from the HTTP request without going through `validateAndClean`
- Integration test for rename that only tests happy path, not destination traversal

**Phase to address:**
Write-side sandbox extension (same phase as rename endpoint). Rename destination traversal must be in the fuzz corpus before merge.

---

### Pitfall 3: Symlink-Following on Write — Write TOCTOU Is More Dangerous Than Read TOCTOU

**What goes wrong:**
The v3.4 read surface uses `root.Open(relPath)` which follows symlinks within the sandbox but rejects symlinks that escape the root. For writes, `root.OpenFile(relPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)` behavior on a symlink path:

- If `relPath` is a symlink to a file WITHIN the sandbox: `os.Root` follows the symlink and writes to the target. Likely correct (editing a symlinked file edits the target).
- If `relPath` is a symlink to a file OUTSIDE the sandbox: `os.Root` rejects this. Correct.
- If `relPath` does not exist but its PARENT is a symlink to a directory outside the sandbox: this is the TOCTOU scenario from v3.4 Pitfall 1, now with write consequences.

The write TOCTOU is more dangerous than the read TOCTOU: on read, an attacker reads a file. On write, an attacker can create or overwrite any file the daemon process has write access to — all user files, shell rc files, agent config files.

**Consequence of write escape:**
- Overwrite `~/.bashrc` → code executes on next shell session
- Overwrite `~/.ssh/authorized_keys` → SSH backdoor
- Overwrite `~/.claude/CLAUDE.md` → inject instructions into future Claude Code sessions
- Create arbitrary files at any writable path on the system

**How to avoid:**
`os.Root`'s write operations (OpenFile, Create, Mkdir, Remove) provide the same TOCTOU-safe sandbox guarantee as reads. The mitigation is:
- Rely on `os.Root` as the terminal write security boundary exactly as for reads.
- All `Sandbox` write methods (`Create`, `WriteFile`, `Rename`, `Remove`, `Mkdir`) must use `root.OpenFile`/`root.Create`/`root.Remove`/`root.Mkdir` — never `os.OpenFile`/`os.Create` with constructed absolute paths.
- Add a dedicated write-path symlink escape test: create a symlink inside the sandbox pointing outside, attempt a write via the symlink path, verify `403` is returned.

**Warning signs:**
- Any write handler that calls `validateAndClean` and then `os.OpenFile` with a constructed absolute path (bypassing `root.OpenFile`)
- Two-step "check then write" pattern: `sb.Stat(relPath)` followed by `os.Create(filepath.Join(rootPath, relPath))`

**Phase to address:**
Write-side sandbox extension. All `Sandbox` write methods must use `root.*` operations. The write-path symlink escape test is a mandatory merge gate.

---

### Pitfall 4: CSRF on State-Changing Endpoints — POST/PUT/DELETE Over Web-Share Without Origin Check

**What goes wrong:**
The v3.4 read surface is all GET requests. GET is CSRF-safe by convention (idempotent). v3.5 introduces POST/PUT/PATCH/DELETE endpoints for write operations. Any POST/DELETE that mutates filesystem state can be issued from a malicious third-party website if the browser is connected to the same Tailscale network.

The existing WebSocket Origin check (Phase 88, v3.1) is byte-for-byte strict and blocks cross-origin WS upgrades. But that check applies only to the WS relay — not to the HTTP file API endpoints. The v3.4 file routes (`/api/files/*`) are HTTP routes with cap-token authentication but NO Origin check.

**Current partial mitigation:**
Cap tokens are in query parameters, not cookies. A cross-origin POST without the cap token returns `401`. Passive CSRF (form submission) is therefore blocked because forms cannot include the cap token unless they know it. However, if the frontend stores the cap token accessibly (e.g., URL bar is visible, URL is shared), active use by an informed attacker is possible.

**The real risk with `files.write` in the cap token:**
A web-share URL with `files.write` in the capability allows the recipient to delete or modify any file within the session's working directory. If this URL is intercepted or forwarded, the interceptor gets write access. The mitigation is explicit opt-in for `files.write` with clear UI warnings.

**How to avoid:**
- Add an `Origin` check to ALL state-changing (`POST`, `PUT`, `PATCH`, `DELETE`) write endpoints. Requests with an `Origin` header that does not match the server's FQDN must return `403 Forbidden`. This parallels the WebSocket Origin check from Phase 88.
- For same-origin desktop GUI requests (no `Origin` header from Wails fetch): the check passes vacuously.
- For web-share requests: `Origin` is the same host as the server, which passes.
- Specifically: `requireFilesWrite` middleware must check `Origin` header presence and validity.
- The `files.write` cap must be off-by-default for web-share issuance. The UI for re-sharing must have a labeled "Allow writes" checkbox that is unchecked by default.

**Warning signs:**
- POST /api/files/write endpoint without an Origin check
- Default session re-share URL that includes `files.write` in the generated cap
- No UI warning when the user grants `files.write` in a shared URL

**Phase to address:**
`files.write` capability and write endpoint security phase. The Origin check is a hard requirement in `requireFilesWrite` middleware.

---

### Pitfall 5: Non-Atomic Writes Corrupting Files on Process Crash or Concurrent Read

**What goes wrong:**
Naive file save:
```go
f, _ := root.OpenFile(relPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
f.Write(content)
f.Close()
```
creates a window where a reader sees a partially written file. `O_TRUNC` zeros the file immediately on open — before any bytes are written. A crash between open and `f.Close()` leaves the file empty or partial.

For AI agent working directories this is especially dangerous:
- Truncating a Go source file and crashing mid-write corrupts it; `go build` fails.
- Truncating `package.json` and crashing produces an empty file that breaks `npm install`.
- AI agents may be reading their own output files concurrently.

**How to avoid:**
Write-then-rename (write to a temp file in the same directory, then rename to final path):
```go
// In sandbox.go Sandbox.WriteFileAtomic:
tmpName := relPath + ".agenthub-tmp-" + randomHex()
f := root.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644) // O_EXCL: fail if exists
f.Write(content)
f.Sync()  // fdatasync — flushes to storage before rename
f.Close()
sb.Rename(tmpName, relPath)  // atomic on POSIX; near-atomic on Windows
```

This approach:
- On crash after `Write` but before `Rename`: original file untouched; temp file is an orphan.
- On crash after `Rename`: new content is fully written.
- Is the standard pattern used by vim, emacs, VS Code, and every serious editor.

**Caveats:**
- `Sandbox.Rename` (implemented per Pitfall 1) must be used, not raw `os.Rename`.
- The temp file must be in the SAME directory as the target for `os.Rename` to be atomic (cross-filesystem rename falls back to copy+delete).
- `O_EXCL` on the temp file prevents two concurrent writes from racing.
- On Windows, `os.Rename` to an existing destination may fail if the destination is open by another process. Implement a short retry loop.

**Warning signs:**
- File save handler that opens with `O_TRUNC` instead of writing to a temp + rename
- No `f.Sync()` before close (data may be in kernel buffer at crash time)
- Temp file written to `/tmp` instead of the same directory as the target

**Phase to address:**
Write-side sandbox extension (implement `Sandbox.WriteFileAtomic`) AND editor save endpoint. Every file-save code path must go through `WriteFileAtomic`. This is a correctness requirement.

---

### Pitfall 6: Multipart Upload Abuse — Unbounded Size, Path Injection via Filename, Partial Upload Corruption

**What goes wrong:**
File upload is the highest-risk write operation because the client controls:
1. **File size:** A large upload can fill the daemon host's disk or exhaust memory during buffering.
2. **Filename in the multipart `Content-Disposition` header:** `Content-Disposition: form-data; name="file"; filename="../../.bashrc"` — `mime/multipart`'s `FileHeader.Filename` contains exactly this string. If used directly as the upload target path, it traverses the sandbox.
3. **Partial uploads:** If the upload stream is interrupted after the file is opened for writing but before all bytes are written, the server may have a truncated/empty file at the target path.

**How to avoid:**

**Filename injection:**
- Extract `FileHeader.Filename` from the multipart header.
- Run it through `filepath.Base` to strip any directory components.
- Then run the result through `validateAndClean`.
- Alternatively: require the upload destination path to be a separate form field or query parameter (not derived from the filename).

**Size limit:**
- Before accepting any bytes, check `r.ContentLength` against the configured max upload size (recommended: 100 MB).
- If `ContentLength` is unknown (-1), enforce via `http.MaxBytesReader(w, r.Body, maxUploadBytes)` which causes `ParseMultipartForm` to fail with `multipart: message too large`.
- Return `413 Request Entity Too Large` before writing anything to disk.
- Do NOT enforce the limit only after reading the entire body into memory.

**Partial upload:**
- Write to a temp file (same atomic write pattern as Pitfall 5).
- Only rename to final destination after entire upload is written and `f.Sync()` completes.
- On any error mid-upload, remove the temp file.
- In the request handler, watch `r.Context().Done()` for client disconnect and clean up the partial temp file.

**Zip bomb:**
- Do NOT implement automatic extraction of any archive format in v3.5. Accept raw file bytes only. A 1 KB zip that expands to 1 GB is a disk-filling DoS.

**Warning signs:**
- Upload handler that reads `FileHeader.Filename` and uses it directly as a path parameter
- No `http.MaxBytesReader` wrapping before `r.ParseMultipartForm`
- Upload that writes to the final destination file directly (not via temp+rename)
- Test that only exercises happy path without oversized uploads or malformed `Content-Disposition`

**Phase to address:**
Upload endpoint implementation phase. All four mitigations must be implemented together. The upload endpoint is the highest single-endpoint risk in v3.5.

---

### Pitfall 7: `files.write` Capability Scope Creep — Granting Write Access Beyond Intended Scope

**What goes wrong:**
The v3.4 `files.read` capability is session-scoped. For `files.write`, the same session-scoping is required. Additional scope-creep vectors:

**Vector 1: `strings.Contains` instead of `HasPerm`**
Adding a new check as:
```go
strings.Contains(claims.Perms, "files.write")
```
would false-positive on `"no-files.write"` — exactly the bug `HasPerm` was designed to prevent.

**Vector 2: `files.write` in the default session-owner token**
Unlike `files.read` (defaulted on for the session owner in v3.4), write access is a deliberate opt-in. The owner must explicitly enable writes. A default `files.write` token means every re-shared URL accidentally grants write access.

**Vector 3: Write cap does not imply read for other sessions**
A `files.write` token for session A must not grant write access to session B. The `requireCapability` SID check already handles this — but the new `requireFilesWrite` must be layered on top of `requireCapability`, not as a replacement.

**Vector 4: Symlink write escape (verified independently)**
`os.Root`'s write operations reject symlink escapes, but a dedicated test for the WRITE path (not just the read path) is required. The v3.4 fuzz tests targeted read-path escapes; the write path must have its own escape tests.

**How to avoid:**
- Add `PermFilesWrite = "files.write"` to `internal/capability/capability.go` alongside `PermFilesRead`.
- Add `requireFilesWrite` middleware parallel to `requireFilesRead`. Use `HasPerm`, not `strings.Contains`.
- Add static grep gate `TestHasPerm_NoStringsContains_Write` that source-inspects `requireFilesWrite` for any use of `strings.Contains`.
- Default: `files.write` must NOT be in the session-owner token by default. Opt-in only.
- Dedicated write-path symlink escape test.

**Warning signs:**
- New `requireFilesWrite` middleware that uses `strings.Contains` instead of `HasPerm`
- Session-owner token that includes `files.write` by default
- No test for write via symlink-to-outside-sandbox

**Phase to address:**
`files.write` capability phase. `PermFilesWrite`, `requireFilesWrite`, and the static grep gate in the same phase as the first write endpoint.

---

### Pitfall 8: Overwriting Shell RC Files and Agent Config Grants Persistent Code Execution

**What goes wrong:**
The sandbox root is the session's working directory — typically a project directory nested inside `$HOME`. The sandbox correctly confines writes to that directory. BUT, when a session's working directory IS `$HOME` itself (Claude Code's default when started without a specific directory), the sandbox root is `/home/user/` or `/Users/ken/`. A `files.write` token for this session can overwrite:
- `~/.bashrc`, `~/.zshrc`, `~/.profile` → code executes on next shell session
- `~/.ssh/authorized_keys` → SSH backdoor
- `~/.claude/CLAUDE.md` → inject instructions into future Claude Code sessions
- `~/.config/agenthub/settings.json` → daemon settings manipulation

This is NOT a sandbox escape — the sandbox correctly confines to the working directory. It is correct-but-dangerous behavior that must be blocked via a denylist.

**Why this matters for v3.5 specifically:**
Remote web-share viewers who receive a `files.write` cap can overwrite these files from any tailnet device.

**How to avoid:**
- Implement a server-side denylist of paths that must never be writable via the file API, regardless of sandbox root. At minimum:
  - `~/.ssh/authorized_keys`, `~/.ssh/config`, `~/.ssh/known_hosts`
  - Shell RC files: `.bashrc`, `.zshrc`, `.profile`, `.bash_profile`, `.zprofile`, `.zshenv`, `.bash_login`
  - `.claude/CLAUDE.md`, `.claude/settings.json`, `.config/agenthub/settings.json`
  - The daemon's own config file
- Implement as an absolute-path check AFTER sandbox validation: if the resolved absolute write path matches any denylist entry, return `403 Protected system file`.
- Present a warning in the GUI/TUI when the session's working directory IS `$HOME` and `files.write` is being enabled.

**Warning signs:**
- No test that verifies `.bashrc` within a home-directory sandbox is write-protected
- The denylist is checked only client-side (frontend) and not server-side
- No warning shown when a session's working directory is `$HOME`

**Phase to address:**
`files.write` capability and security hardening phase. The denylist must be in `internal/files/sandbox.go` as a method-level guard on `WriteFile`, `Remove`, and `Rename` operations. Client-side warnings are UX, not security.

---

## Moderate Pitfalls

### Pitfall 9: Lost-Update Race Between Concurrent Web-Share Editors

**What goes wrong:**
v3.5 introduces in-app editing. A web-share viewer with `files.write` can edit files. A second viewer (or the desktop owner) can edit the same file simultaneously. There is no locking and no conflict detection. The last writer wins and silently overwrites the other's changes.

**How to avoid:**
- Implement an ETag/mtime precondition on the save endpoint:
  - When the editor loads a file, the response includes an `ETag` or `Last-Modified` header.
  - On save, the client sends `If-Match: <etag>` (or `If-Unmodified-Since: <mtime>`).
  - The server checks: if the file's current mtime/ETag does not match, return `412 Precondition Failed`.
  - The client must re-fetch, show the conflict, and allow manual merge or force-overwrite.
- For v3.5 MVP, a simple mtime-check precondition catches the concurrent-write case. Full ETag-based merging can follow.
- Do NOT silently overwrite without any concurrency check.

**Warning signs:**
- Save endpoint that does not accept any precondition header
- No client-side ETag or mtime tracking in the editor component

**Phase to address:**
Editor save endpoint phase. Mtime-check precondition is the minimum viable concurrency protection.

---

### Pitfall 10: Editor Library Vendoring Without Breaking the `vendor_drift_test.go` CI Gate

**What goes wrong:**
The project enforces zero-CDN, vendor-everything discipline via `vendor_drift_test.go` (Phase 93, v3.2). Adding CodeMirror 6 or Monaco via npm must not introduce CDN loading. Specifically:

- **CodeMirror 6:** Installed as npm packages, bundled by Vite into the application JS at build time. No CDN tags in the output. The `vendor_drift_test.go` gate checks `web/vendor/xterm/` (the explicitly vendored tree), NOT Vite build output. CodeMirror 6 added as an npm dependency does NOT violate the existing gate — but verify with CI before merging.

- **Monaco Editor:** Requires a Web Worker served from the same origin. In a Wails app with embedded assets, the worker JS must be embedded in the binary via `embed.FS`. The `-tags wailsassets` build requirement from project memory applies. If the Monaco worker is not embedded, production builds fail with a blank editor or MIME type error. This is the documented wailsassets pitfall from project memory.

**How to avoid:**
- Prefer CodeMirror 6: bundles cleanly as a Vite dependency without a separate worker file.
- After installing editor npm packages: run `vendor_drift_test.go` in CI to confirm zero violations before merging the editor phase.
- Add a Playwright test that verifies zero CSP violations in the browser console after the editor ships.
- If Monaco is chosen, explicitly test the worker embedding path in both `wails dev` and production (`-tags wailsassets`) modes before committing to the decision.

**Warning signs:**
- `vendor_drift_test.go` failure after adding the editor npm dependency
- `Refused to load script` CSP errors in Wails WebView console after shipping the editor
- Editor works in `wails dev` but shows blank in production build (Monaco worker not embedded)

**Phase to address:**
Editor library selection and initial integration phase. Run the full CI gate as part of editor phase verification.

---

### Pitfall 11: Large-File Editor Performance — Loading Files Larger Than 1 MB Into CodeMirror/Monaco

**What goes wrong:**
Both CodeMirror 6 and Monaco are designed for source code editing (typically small files). Attempting to open a 2 MB generated log file or large JSON dataset causes:
- **CodeMirror 6:** Tree-sitter syntax highlighting can take 2-5 seconds on 2 MB files. UI remains responsive but highlighting is absent during parsing.
- **Monaco:** Fully synchronous during initial load; a 2 MB file freezes the Wails WebView during initial render.
- **Both:** In-memory representation of a 2 MB text file exceeds raw byte size.

**How to avoid:**
- Two-tier threshold:
  - Files up to 256 KB: open directly in the editor.
  - Files 256 KB to 1 MB: show "This file is large and may load slowly. Open in editor?" confirmation.
  - Files over 1 MB: decline to open; show "File too large to edit. Use the desktop editor or download the file."
- The 5 MB server-side read cap already prevents the extreme case; the 1 MB threshold is a UX protection.
- TUI `$EDITOR` shell-out has no threshold needed — `$EDITOR` handles large files natively.

**Warning signs:**
- Reports of "editor freezes when opening minified JS"
- No warning shown before loading a 500 KB file into the editor

**Phase to address:**
Editor integration phase. File size thresholds must be designed in at the editor load path, not added post-shipping.

---

### Pitfall 12: CodeMirror in Wails WebView — Tab Key, Paste Conflict, and Mobile Touch

**What goes wrong:**
CodeMirror 6 captures Tab key presses to insert indentation. In the Wails WebView environment several interaction conflicts arise:

**Tab conflict:** The Wails WebView may or may not honor CodeMirror's Tab key capture depending on the platform's tab-focus handling. Verify Tab inserts indentation and does not navigate browser focus.

**Paste (Cmd-V) conflict:** The existing Wails `app.go` intercepts Cmd-V for the macOS clipboard workaround (Phase 49, v1.9). CodeMirror also handles Cmd-V. The two handlers may conflict, causing double-paste or no-paste.

**Mobile/iPad touch:** The v3.3.1 `touchScrollHandler.ts` translates single-finger swipe to terminal scroll and is registered on `.terminal-session-container`. The CodeMirror editor will be in `FileBrowserTab`, a different container. Verify single-finger scroll within the editor pane works correctly on iPad.

**IME (Input Method Editor):** CodeMirror 6 has IME support, but the Wails WebView's composition events may conflict with it for Japanese/Chinese/Korean typing. This is the first component in AgentHub to handle composition events.

**How to avoid:**
- Verify Tab behavior in the Wails WebView on all three platforms during editor phase UAT.
- Test Cmd-V paste into CodeMirror on macOS in the Wails WebView. If double-paste occurs, conditionally disable the existing Cmd-V handler when a CodeMirror editor has focus.
- Add iPad touch testing to the editor UAT checklist.
- Verify IME composition works by testing with a Japanese input method.

**Warning signs:**
- Tab key navigates focus out of the editor instead of inserting indentation
- Paste in CodeMirror inserts content twice on macOS
- Single-finger swipe in the editor on iPad scrolls the file list instead of the editor content

**Phase to address:**
Editor integration phase. Platform-specific interaction testing must be in the verification checklist.

---

### Pitfall 13: TUI `$EDITOR` Shell-Out — Terminal State Corruption, Unset `$EDITOR`, Stale Listing

**What goes wrong:**
The TUI's existing attach pattern uses `tea.Exec` for PTY attach (Bubble Tea v2's suspend/resume pattern). The `$EDITOR` shell-out must use the same pattern. Failure modes:

**`$EDITOR` unset:**
On systems where `$EDITOR` is not set (macOS default, minimal Linux installs), `os.Getenv("EDITOR")` returns `""`. Launching an empty command produces a cryptic error. Code must check `$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi` via `exec.LookPath`.

**Editor crash:**
If the editor exits with a non-zero status (user Ctrl-C'd out, editor crashed), the `tea.Exec` completion handler receives a non-nil error. The TUI must NOT mark the file as "successfully saved" on error. Check the exit code explicitly.

**Terminal state corruption on resume:**
Some editors (particularly `vim` on certain terminals) leave the terminal in an unexpected state after exiting (alternate screen not properly restored, cursor position wrong). Mitigation: after `tea.Exec` returns, issue a `tea.ClearScreen` command to force a full redraw. This is the established pattern for post-attach TUI resume (already used in the TUI attach flow).

**Stale listing after editor save:**
After the user saves the file in `$EDITOR` and returns to the TUI, the directory listing shows the OLD mtime. The TUI must unconditionally refresh the directory listing after `tea.Exec` returns, regardless of exit code.

**How to avoid:**
- `resolveEditor()` helper: `$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi` via `exec.LookPath`. Return actionable error "No editor found. Set $EDITOR in your shell profile." if none found.
- Editor crash handling: log a toast "Editor exited with error (code N)" but still refresh listing.
- Post-editor terminal restoration: always issue `tea.ClearScreen` in the `tea.Exec` completion handler.
- Listing refresh: unconditionally dispatch `loadDirCmd` after `tea.Exec` returns.

**Warning signs:**
- TUI shows garbage after returning from `vi` on some terminals
- Directory listing not refreshed after editor save
- No toast shown when editor exits non-zero
- Crash or panic when `$EDITOR` is unset

**Phase to address:**
TUI `$EDITOR` shell-out phase. The `resolveEditor` helper and post-exec listing refresh must be in the initial implementation.

---

### Pitfall 14: Delete Confirmation Inconsistency Across Surfaces

**What goes wrong:**
The TUI has a kill-confirm modal for session deletion. The GUI has a quit-confirmation modal. Both establish the expectation that destructive operations require confirmation. If delete does NOT require confirmation on one surface but DOES on another, users will accidentally delete files.

Specific risk: web-share viewers with `files.write` may not expect a confirmation dialog. If the desktop GUI requires confirmation but the web-share surface does not, the behavior violates the cross-surface parity rule and creates surprise data loss.

**How to avoid:**
- Define the confirmation contract upfront: every surface must require a confirmation step before delete and rename.
- Web surface: delete button opens a modal dialog "Delete `foo.txt`? This cannot be undone." with Cancel and Delete buttons.
- TUI: `d` enters delete-confirm state; `?` help overlay shows the delete keybinding; pressing `d` again or `y` confirms; Escape cancels.
- Remote surface: same behavior as local (remote delete must not bypass confirmation).
- Cross-surface parity test: verify each surface's delete flow requires at least one confirmation step.

**Warning signs:**
- Delete button in the web UI that triggers the operation without a modal
- TUI delete that fires on a single `d` keypress without a confirmation step
- Inconsistent behavior: one surface confirms, another does not

**Phase to address:**
Cross-surface write parity phase. Confirmation semantics defined before implementation.

---

### Pitfall 15: Remote Write Failure Modes — Cap Expiry Mid-Edit, Network Loss During Upload

**What goes wrong:**
v3.5 introduces remote writes via the daemon proxy and TUI `RemoteFilesClient`. Two failure modes not present in the read surface:

**Cap expiry mid-edit:**
A user opens a remote file for editing. While editing, the remote session's web-share cap expires. When the user attempts to save, the daemon proxy returns `401`. If the editor clears the buffer on `401`, the user loses their unsaved work.

**Network loss during upload:**
A large upload over Tailscale is interrupted mid-transfer. The remote peer's upload handler has a partial temp file. If the context-aware cleanup does not run (or if the handler ignores `r.Context().Done()`), orphaned partial temp files accumulate on the remote host.

**How to avoid:**
- Cap expiry: the editor must detect `401`/`403` on save attempts and show "Your write access has expired. Copy your changes to the clipboard and re-share the session." Do NOT clear the editor buffer on a 401 save attempt.
- Extend the v3.4 `EnableWebSharingTakeover` pattern for write failures.
- Partial temp files: implement a `context.Context`-aware upload handler that watches for request cancellation. On `context.Done`, remove the partial temp file via `root.Remove(tmpPath)`.
- Startup-time orphan scan: on daemon start, scan session working directories for `*.agenthub-tmp-*` files older than 24 hours and remove them.

**Warning signs:**
- Editor buffer cleared when save returns 401
- Remote working directory accumulates `.agenthub-tmp-*` files after repeated interrupted uploads
- No visible message to the user when a write fails due to cap expiry

**Phase to address:**
Remote write parity phase. The cap-expiry editor behavior must be in the requirements for the remote write phase. Orphaned temp file cleanup in the upload endpoint implementation phase.

---

## Minor Pitfalls

### Pitfall 16: Encoding and Line-Ending Corruption on Binary Files and Windows/Unix Line Endings

**What goes wrong:**
- **Line endings:** CodeMirror 6 normalizes all line endings to LF by default. Saving a Windows file (CRLF) from a browser editor silently converts CRLF to LF, which may break Windows-specific tools.
- **Encoding:** If the file on disk is in a non-UTF-8 encoding (ISO-8859-1, Shift-JIS), fetching it as text and saving back silently corrupts the encoding.
- **Binary files:** If a binary file is accidentally opened in the editor, the round-trip through `response.text()` in JavaScript may corrupt null bytes or high bytes.

**How to avoid:**
- On the save path: store content as raw bytes from the request body. Do NOT re-encode.
- Do NOT normalize CRLF to LF on save. Preserve the original line ending style.
- Refuse to open binary files in the in-app editor (enforce via `IsBinary: true` check in FileBrowserTab before showing an "Edit" button).
- Verify CRLF preservation with a test that writes a CRLF file, edits it in the editor, saves, and verifies line endings are preserved.

**Phase to address:**
Editor save endpoint phase. Binary file exclusion enforced in the frontend. Line-ending preservation verified by test.

---

### Pitfall 17: `settings.json` Migration — `FilesWrite` Default False and `schemaVersion: 4`

**What goes wrong:**
v3.4 added `FilesRead bool` to `daemonSettings` and bumped `schemaVersion` to `3`. v3.5 adds `FilesWrite bool`. The defaults-merge constructor pattern must be followed exactly. `FilesWrite` defaulting to `false` is the correct default (write access is opt-in). The migration test must verify this explicitly.

**How to avoid:**
- Add `FilesWrite bool` with `json:"filesWrite,omitempty"` to `daemonSettings`. Default `false` in the constructor.
- Bump `schemaVersion` to `4` in `defaultSettings()`.
- Write `TestSettingsMigration_FilesWriteDefaultsFalse` that loads a `settings_v3.4.json` fixture and asserts `filesWrite == false`.
- Verify the existing `FilesRead` default (`true`) is preserved after the migration (regression for v3.4 behavior).

**Phase to address:**
`files.write` capability phase. Settings migration and capability addition are tightly coupled — don't split them.

---

### Pitfall 18: Write Endpoint Method Routing — POST/PUT/DELETE Must Have Explicit Method Prefixes

**What goes wrong:**
v3.4 registered all file routes as `GET /api/files/{list,stat,read}`. v3.5 adds write endpoints that must use non-GET methods. If any are registered without an explicit method prefix in the Go mux, they match all HTTP methods. Additionally, `requireFilesRead` and `requireFilesWrite` middleware must not overlap — a `files.read`-only cap must get `405 Method Not Allowed` on `DELETE /api/files/delete`, not a `403`.

**How to avoid:**
- Register write endpoints with explicit method prefixes: `POST /api/files/write`, `DELETE /api/files/delete`, `PATCH /api/files/rename`, `PUT /api/files/mkdir`, `POST /api/files/upload`.
- `requireFilesWrite` checks `HasPerm(claims.Perms, PermFilesWrite)`.
- The read-load path for the editor uses `GET /api/files/read` with `requireFilesRead`. A cap with only `files.write` should also be able to read for the edit-load workflow — either implicitly grant `files.read` when `files.write` is issued, or make the editor pre-load use `requireFilesWrite`.
- Add regression tests: `GET /api/files/write` → `405`; `POST /api/files/read` → `405`; `DELETE /api/files/list` → `405`.

**Phase to address:**
Write endpoint route registration phase (same as first write endpoint).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `os.Rename(absPath, absPath2)` directly in handler instead of `Sandbox.Rename` | Simpler code | Both paths must be separately validated; easy to miss destination traversal | Never — `Sandbox.Rename` is the authority |
| `O_TRUNC` in-place save instead of temp+rename | One fewer file operation | Silent corruption on crash or concurrent read | Never — always use atomic temp+rename |
| No `If-Match` precondition on save | Simpler API | Last writer silently wins in any multi-viewer scenario | Acceptable for v3.5 MVP only if a v3.6 issue is filed |
| `strings.Contains` for `files.write` perm check | Familiar, brief | False-positive on `"no-files.write"` — the exact bug `HasPerm` was designed to prevent | Never — use `HasPerm` |
| `files.write` default-on for session owner | Less click-through | Owner accidentally grants write on re-share; one errant click gives remote viewer FS write | Never — `files.write` is opt-in only, never in default share URL |
| No size limit on upload endpoint | Simpler code | Any client can fill the daemon host's disk | Never — `http.MaxBytesReader` before `ParseMultipartForm` |
| Monaco instead of CodeMirror 6 | Richer editor feature set | 3-5 MB bundle; Web Worker must be embedded in Wails binary; more complex vendoring | Acceptable only if Monaco is ratified at plan time with Worker embedding verified on all three platforms |
| No hardened path denylist for shell RC files | Sandbox handles it | Correct-but-dangerous: home-directory sessions can overwrite `.bashrc` or `.ssh/authorized_keys` via write API | Never — denylist is load-bearing when session cwd is `$HOME` |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| `*os.Root` + rename | Attempting `root.Rename` (method does not exist) | Add `Sandbox.Rename(src, dst string)` that validates both paths via `validateAndClean`, constructs absolute paths, calls `os.Rename` |
| `*os.Root` + MkdirAll | Calling `os.MkdirAll` with constructed absolute path | Add `Sandbox.MkdirAll` that calls `root.Mkdir` iteratively per path component |
| Multipart filename | Using `FileHeader.Filename` as target path | `filepath.Base(header.Filename)` + `validateAndClean` — never use `Filename` directly |
| Rename destination | Validating only `from`, not `to` | Both paths through `validateAndClean`; verify joined absolute paths start with `s.rootPath` |
| Atomic write | `O_TRUNC` in-place write | Temp file in same directory + `Sync` + `Rename` to final path |
| `files.write` perm check | `strings.Contains(claims.Perms, "files.write")` | `HasPerm(claims.Perms, capability.PermFilesWrite)` |
| Remote write cap expiry | Clearing editor buffer on 401 | Preserve buffer; show "write access expired" with copy-to-clipboard fallback |
| Partial upload cleanup | Orphaned temp file on client disconnect | `context.Done` watcher in upload handler; startup-time orphan scan |
| TUI post-editor listing | Not refreshing after `tea.Exec` returns | Always dispatch `loadDirCmd` in the `tea.Exec` completion handler |
| TUI `$EDITOR` unset | Empty command or hardcoded fallback | `resolveEditor()`: `$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi` via `exec.LookPath` |
| Origin check on writes | Adding `requireFilesWrite` without Origin check | Mirror the Phase 88 WS Origin check in `requireFilesWrite` for POST/PUT/DELETE |
| `schemaVersion` migration | Adding `FilesWrite bool` without bumping to `4` | `defaultSettings()` bumps to `4`; fixture test for `settings_v3.4.json` loading as `filesWrite: false` |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Large file into CodeMirror/Monaco state | Browser tab OOM; editor freeze on open | 256 KB soft threshold; 1 MB hard limit with refusal; no editor for >1 MB | Any file >256 KB (generated files, lock files, large JSON) |
| Unbounded multipart upload | Daemon memory exhaustion; host disk full | `http.MaxBytesReader` before `ParseMultipartForm`; stream to disk, not memory | Any upload >configured limit |
| `O_TRUNC` on active file | Other readers see empty/partial file mid-write window | Atomic temp+rename on all saves | Any concurrent reader during a write |
| No `f.Sync()` before rename | Data in kernel buffer, not on disk; lost on crash | `f.Sync()` (fdatasync) before temp file rename | Any system crash after write but before OS flush |
| Sync I/O in TUI `$EDITOR` return handler | TUI render loop blocked while reading post-edit directory | `loadDirCmd` in completion handler follows the existing tea.Cmd discipline | Any large directory with slow disk |
| Syntax highlighting for huge files | Tree-sitter parser runs for seconds on 2 MB files | File-size threshold before editor load; disable highlighting for files >512 KB | Any minified JS, generated code, or large JSON |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Rename destination not validated through sandbox | Arbitrary file creation/overwrite anywhere on host | Both `from` and `to` through `validateAndClean`; verify joined paths stay within `s.rootPath` |
| `files.write` granted by default in session-owner token | Every re-share URL implicitly includes write access | `files.write` NEVER in the default-issued token; explicit opt-in only |
| No Origin check on write endpoints | CSRF-class attack from any tailnet browser | `requireFilesWrite` checks `Origin` header (parallel to Phase 88 WS check) |
| `strings.Contains` for `files.write` check | `"no-files.write"` false-positives as authorized | `HasPerm(perms, PermFilesWrite)` — same fix as v3.4's `PermFilesRead` |
| No denylist for shell RC files | Modifies shell startup files when session cwd is `$HOME` | Absolute-path denylist in `Sandbox.Write*` for shell RC, SSH keys, agent config |
| Multipart `Content-Disposition` filename used directly as upload path | Path traversal to any writable path | `filepath.Base` + `validateAndClean` on `FileHeader.Filename` |
| No size limit on upload body | Disk exhaustion, memory OOM | `http.MaxBytesReader(w, r.Body, maxUploadBytes)` before `r.ParseMultipartForm` |
| `O_TRUNC` in-place file save | Partial file visible to concurrent readers; data loss on crash | Atomic temp+rename on all saves |
| Auto-extracting uploaded zip archives | Zip bomb fills host disk; directory traversal in zip paths | Do NOT implement zip extraction in v3.5; accept raw bytes only |
| `files.write` cap URL shared without warning | Recipient can delete/overwrite any file in session working dir | Prominent "write access granted" warning in re-share modal; not default-on |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No confirmation before delete | Accidental deletion of source files | Modal confirmation (web) / `d` → `y` confirm flow (TUI) on every surface |
| Editor buffer cleared on 401 during remote save | User loses unsaved edits when cap expires mid-edit | Preserve buffer; show "access expired — copy your changes" with editor still open |
| No visual distinction between read-only and edit mode in FileBrowserTab | User attempts to edit but `files.write` cap not granted | "View only" badge when `files.write` absent; "Editable" badge/edit button when present |
| `$EDITOR` not set, TUI shows unhelpful error | User cannot edit files via TUI on a fresh system | `resolveEditor` fallback; show "Set $EDITOR in your shell profile" if nothing found |
| Upload replaces existing file silently | User overwrites working config without warning | If upload target already exists, show "This will overwrite `config.json` (last modified: 3 minutes ago). Continue?" |
| Rename succeeds but directory listing does not refresh | User sees old filename in listing | After rename completes, refresh directory listing unconditionally |
| Remote write failure with generic error | User cannot determine whether file was saved | Distinguish `401 cap expired` / `403 no write perm` / `network error` with specific messages |
| Delete of directory with children silently fails | User is confused by "directory not empty" error | Show directory entry count before deleting a directory; offer recursive delete only with explicit confirmation |

---

## "Looks Done But Isn't" Checklist — Mandatory Before Merge

### Write-Side Sandbox (`internal/files/sandbox.go`)
- [ ] `Sandbox.Rename(oldRelPath, newRelPath)` validates BOTH paths through `validateAndClean` before calling `os.Rename`
- [ ] `Sandbox.MkdirAll` uses `root.Mkdir` per component, never `os.MkdirAll` with constructed absolute path
- [ ] `Sandbox.WriteFileAtomic` uses temp file + `f.Sync()` + `Sandbox.Rename` (NOT `O_TRUNC` in-place)
- [ ] Write-path symlink escape test: symlink inside sandbox pointing outside → write returns `403`, not `200`
- [ ] Fuzz corpus extended with rename destination traversal payloads
- [ ] Denylist for shell RC files: `~/.bashrc` within a home-directory sandbox → write returns `403 Protected system file`

### Capability and Middleware (`internal/capability/`, `internal/webserver/`)
- [ ] `PermFilesWrite = "files.write"` added alongside `PermFilesRead` in `capability.go`
- [ ] `requireFilesWrite` middleware uses `HasPerm`, NOT `strings.Contains`
- [ ] Static grep gate: `TestHasPerm_NoStringsContains_Write` source-inspects `requireFilesWrite` for `strings.Contains`
- [ ] Origin check in `requireFilesWrite` for POST/PUT/PATCH/DELETE (parallel to Phase 88 WS Origin check)
- [ ] `files.write` NOT in default session-owner token; NOT in default web-share URL
- [ ] `schemaVersion` bumped to `4`; `TestSettingsMigration_FilesWriteDefaultsFalse` fixture test passes
- [ ] `POST /api/files/write` → `405` when called as `GET`; `GET /api/files/read` → `405` when called as `DELETE`

### Upload Endpoint
- [ ] `FileHeader.Filename` NOT used directly as upload destination path — goes through `filepath.Base` + `validateAndClean`
- [ ] `http.MaxBytesReader` applied before `r.ParseMultipartForm`
- [ ] Upload writes to temp file + atomic rename (not directly to destination)
- [ ] Partial upload temp file cleaned up on `r.Context().Done()` (client disconnect)
- [ ] Oversized upload (> configured max) returns `413` before any bytes written to disk
- [ ] Upload to existing file shows "will overwrite" confirmation in UI before request is issued

### Editor Integration
- [ ] File size threshold: no edit button shown for files > 1 MB; warning shown for 256 KB – 1 MB
- [ ] Binary files (`IsBinary: true`) do NOT show an "Edit" button — download only
- [ ] `vendor_drift_test.go` passes after adding editor npm packages
- [ ] Zero CSP violations in Playwright after editor ships
- [ ] Tab key in editor inserts indentation (does not navigate browser focus) in Wails WebView on macOS
- [ ] Cmd-V paste in editor does not double-paste on macOS
- [ ] Monaco Web Worker embedding verified in production build (`-tags wailsassets`) IF Monaco is chosen over CodeMirror 6

### TUI `$EDITOR` Shell-Out
- [ ] `resolveEditor()` checks `$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi` via `exec.LookPath`
- [ ] Non-zero exit from editor shows toast "Editor exited with error" — does NOT mark file as saved
- [ ] `tea.ClearScreen` issued in `tea.Exec` completion handler to restore terminal
- [ ] `loadDirCmd` dispatched unconditionally after `tea.Exec` returns (listing always refreshed)
- [ ] `$EDITOR` unset produces actionable error "Set $EDITOR in your shell profile", not a crash

### Cross-Surface Parity
- [ ] Delete confirmation required on ALL surfaces: web modal, TUI `d` → `y`, remote same as local
- [ ] Rename available on all surfaces (web, TUI, remote)
- [ ] Remote write `401` (cap expiry) preserves editor buffer and shows "access expired" message
- [ ] Post-write directory listing refresh consistent across all surfaces
- [ ] Remote write endpoints go through same `requireFilesWrite` middleware as local (no bypass in proxy path)

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Rename destination escape shipped | HIGH | Hotfix: add `validateAndClean` on destination path in `Sandbox.Rename`; audit logs for escaped write requests; security advisory |
| Shell RC file overwritten via write API | HIGH | Hotfix: add denylist to `Sandbox.Write*` methods; restore affected files from backup; investigate what was written |
| `O_TRUNC` in-place save causing corruption on crash | MEDIUM | Hotfix: replace with atomic temp+rename; affected users see empty/partial files on crash; advise restore from version control |
| Upload endpoint without size limit ships | MEDIUM | Hotfix: add `http.MaxBytesReader`; manual disk cleanup on affected instances |
| `files.write` in default web-share token | HIGH | Hotfix: remove from default token issuance; existing shared URLs with `files.write` must be re-issued |
| Editor opens binary files, corruption on save | LOW | Fix: add `IsBinary` check before showing edit button; restore affected files from version control |
| TUI terminal state corruption after `$EDITOR` crash | LOW | Fix: add `tea.ClearScreen` in completion handler; workaround: `reset` command in the terminal |
| `$EDITOR` unset crash | LOW | Fix: add `resolveEditor` fallback; workaround: `export EDITOR=nano` before launching TUI |
| Orphaned temp files after interrupted upload | LOW | Startup-time orphan scan in next patch; manual cleanup: `find . -name "*.agenthub-tmp-*" -delete` |

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Write-side sandbox extension (`Sandbox.Rename`, `Sandbox.MkdirAll`, `Sandbox.WriteFileAtomic`) | `os.Root` API gaps; rename destination traversal; non-atomic writes | Implement all three as `Sandbox` methods; validate BOTH paths in Rename; use temp+rename always |
| `files.write` capability and middleware | `strings.Contains` instead of `HasPerm`; no Origin check; accidental default-on | `HasPerm` + static grep gate + Origin check in `requireFilesWrite`; never default-on |
| Upload endpoint | Filename path injection; unbounded size; partial upload corruption | `filepath.Base` + `validateAndClean`; `http.MaxBytesReader`; temp+rename; context-aware cleanup |
| Shell RC file denylist | Missing protection when session cwd is `$HOME` | Absolute-path denylist in `Sandbox.Write*` covering `.bashrc`, `.zshrc`, `.ssh/authorized_keys`, `.claude/CLAUDE.md`, daemon config |
| Editor integration (CodeMirror 6 / Monaco) | `vendor_drift_test.go` failure; Wails Tab/paste conflict; Monaco worker not embedded in prod | Bundle via Vite (not CDN); run CI gate before merge; test Tab and Cmd-V in Wails WebView; verify `-tags wailsassets` if Monaco |
| TUI `$EDITOR` shell-out | Unset `$EDITOR`; terminal corruption; stale listing after edit | `resolveEditor` helper; `tea.ClearScreen` in completion; `loadDirCmd` unconditionally post-exec |
| Delete and rename confirmation | Inconsistent across surfaces; web surface deletes without confirmation | Define confirmation contract before implementation; parity test across web/TUI/remote |
| Remote write parity | Cap expiry mid-edit; partial upload on Tailscale drop | Preserve editor buffer on 401; context-aware cleanup in upload handler; startup orphan scan |
| `schemaVersion: 4` migration | `FilesWrite` zero-value defaults to false (correct); migration test missing | `TestSettingsMigration_FilesWriteDefaultsFalse` fixture test; verify `FilesRead` default preserved |
| Cross-surface parity UAT | One surface has write ops, another doesn't; remote write not tested two-machine | Cross-surface parity is release-blocking; all write ops must exist on all surfaces before release gate |

---

## Sources

- [Go Blog: Traversal-resistant file APIs (os.Root, Go 1.24)](https://go.dev/blog/osroot) — HIGH confidence, official
- [golang/go#67002: os: safer file open functions (os.Root design — Rename/MkdirAll gaps)](https://github.com/golang/go/issues/67002)
- [golang/go#69462: os: Root.Rename (proposed)](https://github.com/golang/go/issues/69462) — MEDIUM confidence, issue tracker (proposal, not yet shipped as of Go 1.26)
- [OWASP Path Traversal — rename destination as second attack vector](https://owasp.org/www-community/attacks/Path_Traversal)
- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [OWASP File Upload Cheat Sheet — filename injection, zip bomb, size limits](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html)
- [CodeMirror 6 System Guide — bundling, language packages, no CDN required](https://codemirror.net/docs/guide/)
- [Monaco Editor ESM Integration — Web Worker embedding requirement](https://github.com/microsoft/monaco-editor/blob/main/docs/integrate-esm.md)
- [Go atomic file write pattern — temp+rename semantics](https://pkg.go.dev/os#Rename)
- [Bubble Tea v2 `tea.Exec` — suspend/resume pattern for external process](https://pkg.go.dev/github.com/charmbracelet/bubbletea/v2#Exec)
- [CVE-2026-27976: Zed code editor symlink escape (write-path TOCTOU class)](https://www.thehackerwire.com/zed-code-editor-sandbox-escape-via-symlink-traversal-cve-2026-27976/) — class applies to write path equally
- AgentHub source (verified from live v3.4 codebase): `internal/files/sandbox.go`, `internal/files/handler.go`, `internal/files/types.go`, `internal/capability/capability.go`, `internal/webserver/capability_mw.go`, `internal/webserver/server.go`, `internal/tui/update.go`, `internal/tui/files_cmds.go`, `internal/tui/cmds.go`
- [v3.4 PITFALLS.md — read-side research, path traversal menagerie, fuzz corpus skeleton](.planning/milestones/v3.4-research/PITFALLS.md) — prior art, not repeated here

---
*Pitfalls research for: v3.5 Write-Side File Browser + In-App Code Editor — adding write operations + editor to AgentHub's sandboxed read-only FS API*
*Researched: 2026-06-14*
