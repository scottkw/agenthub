# Feature Research

**Domain:** Write-side file browser + in-app code editor — AgentHub v3.5
**Researched:** 2026-06-14
**Confidence:** HIGH — based on v3.4 shipped API contract (authoritative), v3.4 FEATURES.md prior research,
PROJECT.md milestone decisions, and confirmed existing capability/sandbox architecture.

> Scope: v3.5 adds write operations (create/edit-save/upload/delete/rename/mkdir), an in-app code
> editor replacing the v3.4 plain-text preview, TUI `$EDITOR` shell-out parity, opt-in `files.write`
> capability for web-share, remote tailnet peer write parity, and carried tech-debt TD-4/TD-5.
> This file focuses on the NEW write-side capability set only.

---

## Feature Landscape

### Table Stakes — Write Operations (Users Expect These)

These are expected of any embedded file browser that supports writes. Missing them makes the write
side feel broken or half-shipped.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Edit-and-save existing file** | Every file manager with write access lets you edit an existing file. This is the primary motivation for Issue #63. | MEDIUM | Opens file in editor; Cmd/Ctrl+S saves. Dirty-state indicator required. Requires new `POST /api/files/write` endpoint + If-Match for conflict detection. Depends on editor library landing (CodeMirror 6). |
| **Create new file (touch)** | VS Code Explorer, GitHub web editor, Finder, and every other file manager let you create a new file in the current directory. | LOW | Inline rename-on-creation UX (name input appears in file list, Enter confirms, Esc cancels). Maps to `POST /api/files/create` or `write` with empty body. |
| **mkdir — create new directory** | Counterpart to file creation. Any file manager with create-file also has create-folder. | LOW | Same inline-creation UX as file create. Maps to `POST /api/files/mkdir`. |
| **Delete file** | Users need to remove files. Conspicuously absent without it. | LOW-MEDIUM | Confirmation modal required (see Safety UX). Keyboard shortcut: Delete/Backspace with confirmation prompt. Maps to `DELETE /api/files/delete`. |
| **Delete directory (recursive)** | Follows from delete-file. VS Code Explorer, Finder, GitHub Codespaces all delete directories recursively with confirmation. | MEDIUM | Needs explicit recursive-delete warning in confirmation modal ("This will delete 14 files. This cannot be undone."). Higher complexity than file delete — server must walk and delete entire subtree. Shares endpoint with file delete but daemon-side logic is distinct. |
| **Rename file or directory** | Core file management. GitHub web editor, VS Code, and every other manager support rename. | LOW-MEDIUM | Inline rename UX (F2 or click-to-rename on name cell). Maps to `PATCH /api/files/rename` with `{from, to}`. Rename within same directory only in v3.5 (move across directories is a differentiator). |
| **Upload single file** | Every hosted-code tool (GitHub Codespaces, VS Code Server, Cyberduck) allows uploading a file from the local machine into the browser session. | MEDIUM | File picker via `<input type="file">` or drag-and-drop into the file list. Streamed multipart upload to `POST /api/files/upload`. Progress indicator for files > ~100KB. |
| **`files.write` capability bit** | The v3.4 `files.read` pattern established that write access is capability-gated. Web-share viewers and remote sessions need explicit grant control. | MEDIUM | New `files.write` comma-separated capability bit following the `HasPerm` whole-token pattern. Session-owner cap includes `files.write` by default. Web-share viewer cap excludes by default (opt-in). Maps to new `requireFilesWrite` middleware separate from `requireFilesRead`. |
| **Dirty-state indicator in editor** | Every editor (VS Code, CodeMirror, Monaco, GitHub web editor) shows an unsaved-changes indicator — typically a dot on the tab or in the header. Users cannot safely switch away without it. | LOW | Modified indicator (bullet or asterisk) on the file browser tab or editor header. Keyed off CodeMirror's `EditorState.doc` !== saved content snapshot. |
| **Unsaved-changes warning on navigate-away** | VS Code, GitHub web editor, and every browser-based editor warn before discarding edits on tab close or navigation. | LOW-MEDIUM | Intercept navigation away from the editor (clicking another file, closing the tab, navigating up the directory tree). Show: "You have unsaved changes. Save or discard?" modal. Does NOT use browser `beforeunload` on desktop (Wails blocks this); must be React-level guard. |
| **Keyboard save (Cmd/Ctrl+S)** | Universal keyboard shortcut for save. Users muscle-memory this. Its absence is immediately felt. | LOW | `Cmd+S` on macOS, `Ctrl+S` on Windows/Linux. Must intercept before browser default. CodeMirror provides extension hooks for this. |
| **Save confirmation / feedback** | After save, users need to know the save succeeded. Matches the established three-state Save button pattern from Settings (idle/saving/saved). | LOW | Show "Saved" transient indicator (1.5s matches Settings pattern) or a brief status-line message. |
| **Error on save failure** | If the save fails (permissions changed, disk full, concurrent-edit conflict), users need an actionable error, not silence. | LOW-MEDIUM | Surface specific error: "Save failed: permission denied", "Save failed: file was modified by another process (409)." With option to force-overwrite or save as new file. |
| **Syntax highlighting in editor** | As of v3.4, code files display as plain monospace. v3.5 is explicitly the milestone where the editor library lands. Users expect syntax highlighting for code files once an editor is present. | MEDIUM | Provided by CodeMirror 6 language packs. Requires language detection from file extension. Common languages: Go, TypeScript/TSX, JavaScript/JSX, Python, JSON, YAML, Markdown, Bash, HTML, CSS. |
| **Read-only to edit toggle** | Users expect a "view first, edit intentionally" model, not auto-edit-mode on every file open. GitHub web editor, Gitea, and VS Code all require an explicit edit action. | LOW | Edit button (pencil icon) in the preview pane header. Switches preview pane to editor. The toggle is explicit and visible — avoids accidental edits. |

---

### Table Stakes — Editor UX (In-App Code Editor)

The specific UX behaviors expected of any embedded code editor.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **CodeMirror 6 as editor library** | Decision pre-ratified at v3.5 plan time per v3.4 REQUIREMENTS.md OQ-3 / PROJECT.md scope decisions. ~200KB vs Monaco ~5MB. CSP-clean (no `eval`). | MEDIUM | `@codemirror/view`, `@codemirror/state`, `@codemirror/commands`. Must be vendored under `web/vendor/` to satisfy `vendor_drift_test.go` CI gate. Language packs: `@codemirror/lang-*`. |
| **Line numbers** | Users expect line numbers in any code editor — CodeMirror 6, Monaco, ACE all show them by default. | LOW | CodeMirror `lineNumbers()` extension. |
| **Cursor position in status line** | VS Code shows `Ln N, Col N` in the status bar. GitHub Codespaces and CodeMirror-based editors all show cursor position. | LOW | CodeMirror `EditorView.updateListener` exposes selection state. Show in file browser status line. |
| **Large-file handling — open-but-disable-some-features** | VS Code warns for files over ~2MB (syntax highlighting disabled). GitHub web editor refuses files over 1MB. | MEDIUM | v3.4 established a 5MB preview cap. Editor should use the same cap. For files near the cap (> ~500KB), disable syntax highlighting (plain text mode) to avoid parser perf cliff. Show: "Syntax highlighting disabled for large files." |
| **Large-file warning before editing** | Users opening a 4MB log file in an editor should know what they're doing — editing a 4MB file in-browser is unusual. | LOW | Warn at open-for-edit time for files > 500KB: "This is a large file (N MB). Edits may be slow." Warn-then-proceed, not block. |
| **Binary file — no edit mode** | Binary files cannot be meaningfully edited in a text editor. GitHub web editor, VS Code both block this. | LOW | If `IsBinary = true` in the `FileEntry` stat response, hide the Edit button entirely. Show only "Download" affordance. |

---

### Table Stakes — Safety UX

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Delete confirmation modal** | Every file manager (Finder, VS Code, Windows Explorer) requires confirmation before delete. Missing it = user anger the first time they fat-finger Delete. | LOW | Modal: "Delete `filename`? This cannot be undone." Two buttons: Cancel (default focus) / Delete (destructive, red). For directories: "Delete `dirname` and all N files inside? This cannot be undone." |
| **Overwrite-on-save conflict detection (If-Match)** | When two web-share viewers or GUI + terminal are editing the same file, last-writer-wins silently discards edits. Every server-side editor (GitHub, Gitea, Codespaces) uses ETag/If-Match to detect this. | MEDIUM | `POST /api/files/write` accepts `If-Match: <etag>` header. Returns 412 Precondition Failed when ETag mismatches. Client shows: "This file was modified by another process. [Force overwrite] [Save as new file] [Discard my changes]". ETag is file mtime + size, not full content hash (performance). |
| **Atomic-write (temp file + rename)** | Writing directly to the target file risks partial writes corrupting the file on daemon crash or disk-full. VS Code uses atomic writes. | MEDIUM | Daemon writes to `filename.tmp` in the same directory, then `os.Rename` to target. `os.Rename` is atomic on POSIX. On Windows, Go's `os.Rename` calls `MoveFileEx(MOVEFILE_REPLACE_EXISTING)`. The sandboxed `os.OpenRoot` constraint means the temp file must be within the sandbox — use a sibling `.agenthubtmp_<random>` name. |
| **Rename collision warning** | Attempting to rename a file to a name that already exists should warn, not silently overwrite. | LOW | Daemon returns 409 Conflict if target exists. Client shows: "A file named `filename` already exists. Replace it?" with Cancel (default) / Replace. |
| **Upload overwrite warning** | Uploading a file whose name already exists in the directory should warn. GitHub web editor shows "A file with this name already exists. Replace it?" | LOW | Same 409 pattern as rename collision. Show warning before completing the upload. |

---

### Differentiators (Competitive Advantage in AgentHub's Context)

These features are not universally expected in embedded file browsers but fit AgentHub's specific
Tailscale/AI-session context and elevate the experience above a generic file manager.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Upload multiple files** | VS Code drag-and-drop to Explorer, GitHub web uploader, Cyberduck all do multi-file upload. Single-file is table stakes; multi-file is the power-user differentiator. | MEDIUM | Multi-select via `<input type="file" multiple>` or drag-and-drop of multiple files into the directory listing. Shows upload queue with per-file progress bars. Chunked upload stream to avoid timeout on large files. |
| **`files.write` opt-in for web-share viewers** | AgentHub's Tailscale model lets you share a session URL with a teammate who can then edit files in the session's working directory. No other terminal-sharing tool does this with sandboxed, capability-gated file writes. | HIGH | Requires a dedicated security phase. The web-share grant flow must present explicit `files.write` opt-in (separate from `files.read`). Default OFF. Dangerous without explicit user understanding — requires a confirmation dialog: "This will allow the recipient to create, edit, delete, and upload files in this session's working directory." |
| **Remote tailnet peer write parity** | Editing files on a remote peer session over Tailscale with the same UI as local — no other tool does this. Directly extends the v3.4 remote-read parity. | HIGH | Follows the v3.4 Phase 122 proxy pattern: desktop GUI uses `daemon proxy /api/files/remote/{sid}/write` etc. TUI uses direct HTTPS to the remote peer. Requires `files.write` cap on the remote session's cap token. |
| **TUI edit via `$EDITOR` shell-out** | Matches `git commit`, `crontab -e`, `kubectl edit` — the Unix-native way to edit files from a TUI. Power users expect pressing `e` on a file in the TUI file browser to open `$EDITOR`. | MEDIUM | TUI shell-out using `tea.Exec` (suspend TUI, spawn `$EDITOR` with file path, resume TUI). Requires `$EDITOR` to be set; if unset, show clear error: "`$EDITOR` is not set. Set it in your shell profile (e.g., `export EDITOR=nano`)." No in-TUI CodeMirror — shell-out is the correct TUI edit pattern. |
| **Move file across directories** | Renaming within the same directory is table stakes; moving across directories (drag-and-drop or cut-paste) is a power-user feature that VS Code and GitHub Codespaces support. | HIGH | Path: `PATCH /api/files/rename` already accepts `{from, to}` — daemon validates both paths are within the sandbox. UI: drag-and-drop from list to breadcrumb segment, or a "Move to..." picker modal. Explicitly a v3.5 stretch goal — confirm scope at plan time. |
| **Concurrent-edit viewer count indicator** | When multiple web-share viewers have the same file open in edit mode, showing "2 others editing this file" prevents overwrite surprises. | MEDIUM | Requires daemon-side per-file edit-session tracking (a map of `filepath → [capToken]`). Not required for v3.5 correctness (If-Match covers safety), but a meaningful DX improvement for collaborative sessions. Explicitly a stretch goal. |
| **Filesystem-change auto-refresh** | When the AI agent running in the terminal modifies a file (writes a new component, patches a config), the file browser should reflect the change without manual refresh. | HIGH | Requires inotify (Linux) / FSEvents (macOS) / ReadDirectoryChangesW (Windows) push via SSE or WebSocket from daemon. High implementation complexity; not required for v3.5 correctness (refresh-on-navigate covers the basic case). Explicitly deferred to v3.6+. |

---

### Anti-Features (Explicitly Not This Milestone)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Monaco Editor** | VS Code uses it; power users know it; rich built-in features. | ~5MB bundle (vs CodeMirror 6 ~200KB); requires `eval`-based CSP carve-outs (conflicts with existing strict `script-src 'self'` policy); complex vendoring. The v3.4 REQUIREMENTS.md explicitly deferred this at OQ-3 resolution; CodeMirror 6 was selected. | CodeMirror 6 — CSP-clean, ~200KB, extensible language packs, wide adoption (GitHub, Replit, Glitch all use it). Decision ratified. |
| **In-TUI CodeMirror rendering** | "Why not show a full editor inside the TUI?" | TUI runs in a real terminal (not WebView/browser). CodeMirror is a browser-DOM library. Embedding it in Bubble Tea would require xterm.js-style WebView passthrough which is out of scope. | TUI shells out to `$EDITOR` via `tea.Exec` — the established Unix pattern. Matches `git commit -e`, `kubectl edit`. |
| **Drag-out to download from editor** | "Let me drag the file out to my filesystem." | Browser security model prevents drag-out-to-filesystem from non-native browser windows. Wails WebView has the same restriction. Complex and unreliable cross-platform. | Download button in the preview/editor header wired to the Range-capable `/api/files/read` endpoint (already shipping in v3.4). |
| **Real-time multi-cursor collaborative editing (CRDT/OT)** | Google Docs model; multiple cursors editing simultaneously. | Requires CRDT (e.g., Yjs) or Operational Transform on the server — a fundamentally different architecture from single-writer ETag/If-Match. Would take a full separate milestone. The AgentHub use case is one operator + AI agent, not multi-human real-time collab. | If-Match conflict detection (last-writer-wins with 412 warning) covers the actual concurrency risk in AgentHub sessions. |
| **Auto-save (save on every keystroke or timer)** | VS Code has auto-save. Users sometimes expect it. | AI agents are actively watching the filesystem. Auto-saving partial edits would inject corrupted half-finished files into a live AI coding session. This is dangerous in AgentHub's specific context — the AI agent might immediately read and act on the partial edit. | Explicit Cmd/Ctrl+S save only. The save action is intentional, not ambient. |
| **Git integration (diff view, stage/unstage, commit)** | VS Code Source Control panel does this. Embedded git UI is popular. | Out of scope for the file browser epic. Would require a separate `internal/git/` package, new API surface, and a multi-phase implementation. The terminal tabs already give direct access to git CLI. | Users already have full git via terminal sessions. File browser + editor covers "view and edit" without git semantics. |
| **Recursive file search** | "Find all files matching pattern across the project" | Expensive server-side walk for large projects over remote relay; requires streaming response; complex UI to surface search results. No foundation laid in v3.4 read-only. | Current-directory `/` filter (v3.4) covers the in-directory case. Full-project search via `rg`/`find` in the terminal tab covers the recursive case. |
| **Drag-and-drop move within the file list** | VS Code drag to reorder / move files in Explorer. | Requires implementing drag sources and drop targets in the React file list — significant event plumbing on top of the current simple list. Drag-and-drop across directories adds sandbox validation complexity. First-release writes should use explicit named actions (rename/move via modal), not gestures. | `PATCH /api/files/rename` with a "Move to..." picker modal is a v3.5 stretch. Drag-and-drop is a v3.6 polish item. |
| **Binary file hex editor** | Power users editing ELF binaries, compiled assets. | A hex editor is a separate product-grade feature. The binary-refusal path (v3.4) is correct for a file browser embedded in an AI session management tool. | Download the binary, edit with a native tool. |
| **Upload entire directory (zip/unzip)** | "Let me upload my whole project." | Browser drag-and-drop of directories produces a list of individual File objects (no directory structure preserved in the multipart boundary). Server-side zip extraction adds a new attack surface (zip-slip vulnerabilities). | Multi-file upload within a directory. The AI agent running in the terminal can handle bulk operations via shell commands. |
| **Per-session write audit log** | "Show me what files were changed in this session." | Requires a new persistent audit store, API surface, and UI panel. Not justified for v3.5 scope. | Bash history in the terminal and git log cover the audit-trail need for the primary AI-coding use case. |

---

## Feature Dependencies

```
files.write capability bit (new, mirrors files.read)
    └──required by──> POST /api/files/write (edit-save)
    └──required by──> POST /api/files/upload
    └──required by──> DELETE /api/files/delete
    └──required by──> PATCH /api/files/rename
    └──required by──> POST /api/files/mkdir
    └──required by──> web-share write access (opt-in grant)
    └──required by──> remote tailnet peer write parity
    └──depends on──> v3.4 HasPerm whole-token comma-split (already shipped)
    └──depends on──> v3.4 requireFilesRead middleware pattern (new parallel middleware)

os.OpenRoot sandbox (v3.4 internal/files/ Sandbox, already shipped)
    └──required by──> all write endpoints (sandbox must wrap write ops too)
    └──note: write ops (mkdir, delete, rename across dirs) need new sandbox methods
    └──extends: Read/List/Stat -> add Write/Upload/Delete/Rename/Mkdir methods

Atomic-write (temp + rename)
    └──required by──> POST /api/files/write (data integrity)
    └──requires──> temp file within sandbox bounds (sibling .agenthubtmp_<random>)

If-Match / ETag conflict detection
    └──required by──> POST /api/files/write (concurrent-edit safety)
    └──depends on──> GET /api/files/stat response (mtime + size as ETag, already shipped)

CodeMirror 6 (new frontend dep)
    └──required by──> in-app editor UI (FileBrowserTab.tsx edit mode)
    └──required by──> syntax highlighting
    └──required by──> Cmd/Ctrl+S save shortcut
    └──must be──> vendored under web/vendor/ (vendor_drift_test.go CI gate)
    └──enables──> dirty-state tracking (EditorState.doc !== saved snapshot)

Editor (CodeMirror 6) in FileBrowserTab
    └──required by──> edit-and-save workflow
    └──required by──> dirty-state indicator
    └──required by──> unsaved-changes warning on navigate-away
    └──depends on──> files.write capability (must be present to enable Edit button)
    └──replaces──> v3.4 plain-text preview pane for code files

Delete endpoint (DELETE /api/files/delete)
    └──required by──> delete-file UX
    └──required by──> delete-directory UX (recursive, server-side walk)

Rename endpoint (PATCH /api/files/rename)
    └──required by──> inline rename UX
    └──required by──> move-across-directories (stretch; same endpoint, both paths validated)

Upload endpoint (POST /api/files/upload)
    └──required by──> single-file upload
    └──required by──> multi-file upload (same endpoint, batched)

TUI $EDITOR shell-out
    └──requires──> tea.Exec suspend/resume (already used in TUI attach; pattern established)
    └──requires──> $EDITOR env var detection at shell-out time
    └──note: local-only; remote TUI write uses HTTPS stream, not $EDITOR

web-share files.write opt-in
    └──requires──> files.write capability bit
    └──requires──> explicit user grant UI (new confirmation modal at share time)
    └──requires──> security-focused phase (separate from write endpoint phase)
    └──requires──> FuzzSandboxWrite corpus extending v3.4 FuzzSandboxPath

remote tailnet peer writes
    └──requires──> write endpoints on the remote peer's webserver (Phase 119 parallel)
    └──requires──> files.write cap on the remote session's cap token
    └──follows──> v3.4 Phase 122 proxy pattern (desktop GUI via daemon proxy, TUI via direct HTTPS)
    └──version constraint──> remote peer must run v3.5+ (write endpoints absent on v3.4 peers)
```

### Dependency Notes

- **Write ops and the v3.4 sandbox:** The v3.4 `Sandbox` type wraps `*os.Root` and has `List`, `Stat`, `Read` methods. v3.5 must add `Write`, `Upload`, `Delete`, `Rename`, and `Mkdir` methods to the same type. All use `os.OpenInRoot` for TOCTOU-free path resolution — the same kernel guarantee that secured the read side covers writes automatically when using `os.OpenRoot.OpenFile()` with write flags.

- **ETag strategy:** Using mtime + size (not content hash) for ETags means a file edited and restored to the same content within a 1-second window could cause a false-positive 412. This is acceptable — false-positives on conflict detection are safe (user sees "conflict" but there is not one); false-negatives (silent overwrite) are dangerous. Content hashing would require reading the full file on every stat call — too expensive.

- **CodeMirror 6 vendoring:** The existing `vendor_drift_test.go` CI gate enforces version parity between `package.json` and vendored assets for every `@xterm/addon-*`. The same gate pattern must be extended to enforce CodeMirror version parity. CodeMirror packages (`@codemirror/view`, `@codemirror/state`, `@codemirror/commands`, `@codemirror/lang-*`) must all land in `web/vendor/codemirror/` or equivalent flat structure under `web/vendor/`.

- **Unsaved-changes guard on desktop (Wails):** Wails v2 does not expose `window.onbeforeunload` in a way that reliably intercepts window close or tab navigation. The guard must be a React-level concern: a `useEffect` that registers against the existing `app:quit-requested` event pathway when the editor is dirty. Navigation to another file within the file browser tab must also check dirty state before replacing the editor content.

- **TUI shell-out file path:** The TUI `$EDITOR` shell-out passes the file's absolute path (within the sandbox) to the editor. The daemon resolves this from the session's WorkDir + relative path before handing it to the TUI. This is local-only — remote TUI writes stream file content via HTTPS to the remote peer, not via `$EDITOR` on the remote machine.

- **web-share files.write security phase:** The `files.write` opt-in for web-share is the highest-risk surface in v3.5. It MUST be a separate phase with dedicated security review. The security concerns: (1) path traversal on writes (sandbox handles this, but new write code paths need the same fuzz testing as v3.4 read paths); (2) upload size limits and DoS via large uploads; (3) the capability grant UI must make the risk explicit to the session owner. This phase should include a new `FuzzSandboxWrite` fuzz corpus extending v3.4's `FuzzSandboxPath`.

- **Remote peer version gate:** Remote write attempts against a v3.4 peer (which has no write endpoints) return 405 Method Not Allowed. The client must detect 405 and show: "The remote session is running an older version of AgentHub that does not support file writes." Do not surface a generic network error.

---

## Cross-Surface Parity Analysis

The release-blocking parity contract (established in v3.3 Phases 107/108, reaffirmed in v3.4
Phase 122) requires write operations to behave equivalently across all surfaces.

### What Parity Means for Write Operations

| Operation | Desktop GUI | Web-share Browser | TUI | CLI |
|-----------|-------------|-------------------|-----|-----|
| Edit-and-save | CodeMirror 6 editor in FileBrowserTab, Cmd+S | Same editor via web-share HTTPS, Ctrl+S | `$EDITOR` shell-out (local); HTTPS stream (remote) | Out of scope — use terminal tab |
| Upload | File picker + drag-and-drop in FileBrowserTab | Same UI via web-share | No in-TUI upload; TUI shows "use desktop/web" message | Out of scope — use terminal tab |
| Delete | Delete key + confirmation modal | Same UI via web-share | `d` key + confirmation modal (Bubble Tea dialog) | Out of scope — use terminal tab |
| Rename | F2 inline edit | Same UI via web-share | `r` key + inline rename | Out of scope — use terminal tab |
| Mkdir | Toolbar button + inline name input | Same UI via web-share | `m` key + inline name input | Out of scope — use terminal tab |
| Remote session | daemon proxy `/api/files/remote/{sid}/write` | N/A (remote web-share has own HTTPS) | Direct HTTPS to remote peer with `files.write` cap | N/A |

### TUI Parity Notes

- **Edit:** `$EDITOR` shell-out is the TUI parity for the GUI editor. The parity is behavioral (edit-and-save is possible on all surfaces) not visual (TUI does not embed CodeMirror).
- **Upload:** TUI has no upload mechanism. This is the one intentional parity gap — TUI shows "Use desktop or web to upload files." (same pattern as image preview in v3.4). Follow-up issue may be filed if demand arises.
- **Delete/Rename/Mkdir:** These use standard Bubble Tea interaction patterns (key + confirmation dialog using the existing kill-session confirmation dialog pattern). Full TUI parity for these three operations.
- **Remote writes in TUI:** Direct HTTPS to remote peer with `files.write` cap (same as v3.4 TUI remote read pattern via `RemoteFilesClient`).

### CLI Parity Notes

CLI does not expose file browser commands — users interact with files via the terminal session itself. This is consistent with v3.4 (CLI exposes no file-browse commands either). Not a parity gap.

---

## Remote Writes — Behavior and Constraints

| Scenario | Expected Behavior | Constraint |
|----------|-------------------|------------|
| Local session (desktop GUI) | Write directly via daemon Unix socket; cap not required on socket side (trusted local surface, WEB-01 decision from v3.4) | None — full writes available |
| Local session (TUI) | Same daemon Unix socket path | None |
| Remote session (desktop GUI) | GUI hits `daemon proxy /api/files/remote/{sid}/write`, daemon proxies to remote peer's `/api/files/write` with `files.write` cap | Remote peer must have web-sharing enabled; remote session owner must have granted `files.write` in cap |
| Remote session (TUI) | `RemoteFilesClient` dials remote peer HTTPS directly, sends `files.write`-bearing cap token | Same as above; TLS 1.2+ pinned |
| Web-share viewer (local session) | Browser hits `/api/files/write` on the hosting peer; `requireFilesWrite` middleware gates | Viewer must have been granted `files.write` in the web-share cap grant |
| Web-share viewer (remote session) | Browser accesses the remote peer's HTTPS directly with cap | `files.write` cap must be granted on the remote session's web-share grant |

---

## MVP Definition for v3.5

### Launch With (Core Write + Editor)

- [ ] `files.write` capability bit + `requireFilesWrite` middleware (mirrors `files.read`/`requireFilesRead`)
- [ ] `POST /api/files/write` with atomic temp+rename and If-Match ETag conflict detection
- [ ] `DELETE /api/files/delete` (file + directory recursive, sandbox-validated)
- [ ] `PATCH /api/files/rename` (within sandbox, same directory)
- [ ] `POST /api/files/mkdir` (within sandbox)
- [ ] `POST /api/files/upload` (single file, streamed multipart)
- [ ] CodeMirror 6 vendored + integrated into FileBrowserTab (replaces plain-text preview for editable files)
- [ ] Read-only to edit toggle (explicit Edit button; no auto-edit)
- [ ] Dirty-state indicator in editor header
- [ ] Unsaved-changes warning on navigate-away (React-level guard)
- [ ] Cmd/Ctrl+S save shortcut
- [ ] Save success/failure feedback (three-state pattern)
- [ ] 412 conflict UX (force-overwrite / save-as / discard options)
- [ ] Delete confirmation modal (file + directory variants)
- [ ] Rename collision warning (409)
- [ ] TUI `$EDITOR` shell-out on `e` key (local sessions only)
- [ ] `$EDITOR` unset error message
- [ ] Syntax highlighting via CodeMirror language packs (common languages)
- [ ] Binary file — no Edit button (IsBinary = true from stat)
- [ ] Large-file editor guard (warn + disable syntax highlight > 500KB)
- [ ] `files.write` opt-in for web-share (security-focused phase, explicit confirmation, FuzzSandboxWrite corpus)
- [ ] Remote tailnet peer write parity (GUI + TUI, following v3.4 Phase 122 proxy pattern)
- [ ] TD-4 (Phase 120 WR-01..05 hardening) and TD-5 (ExchangeJoinCode shim cleanup) folded in

### Add After Validation (v3.5.x or v3.6)

- [ ] Multi-file upload — trigger: single-file upload bakes under real use
- [ ] Move across directories — trigger: rename lands cleanly; confirm demand
- [ ] Concurrent-edit viewer count indicator — trigger: multi-viewer write collision reported
- [ ] Filesystem-change auto-refresh (inotify/FSEvents/ReadDirectoryChanges) — trigger: user-reported friction with stale listings
- [ ] Right-click context menu with write ops — trigger: write ops are table stakes; context menu is discoverability polish

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| `files.write` capability bit + middleware | HIGH (gating all writes) | LOW | P1 |
| `POST /api/files/write` + atomic + If-Match | HIGH | MEDIUM | P1 |
| `DELETE /api/files/delete` (file + dir) | HIGH | MEDIUM | P1 |
| `PATCH /api/files/rename` | HIGH | LOW | P1 |
| `POST /api/files/mkdir` | HIGH | LOW | P1 |
| `POST /api/files/upload` (single) | HIGH | MEDIUM | P1 |
| CodeMirror 6 integration | HIGH (milestone centrepiece) | MEDIUM | P1 |
| Edit toggle (explicit, not auto) | HIGH | LOW | P1 |
| Dirty-state indicator | HIGH | LOW | P1 |
| Unsaved-changes warning | HIGH | LOW-MEDIUM | P1 |
| Cmd/Ctrl+S save | HIGH | LOW | P1 |
| Save feedback (three-state) | MEDIUM | LOW | P1 |
| 412 conflict UX | HIGH (data safety) | MEDIUM | P1 |
| Delete confirmation modal | HIGH (safety) | LOW | P1 |
| Syntax highlighting (common languages) | HIGH (reason for editor library) | MEDIUM | P1 |
| Binary file — no edit button | HIGH (correctness) | LOW | P1 |
| TUI `$EDITOR` shell-out | HIGH (parity contract) | MEDIUM | P1 |
| `$EDITOR` unset error | MEDIUM | LOW | P1 |
| web-share `files.write` opt-in | HIGH (most-exposed surface) | HIGH (dedicated security phase) | P1 |
| Remote tailnet peer writes | HIGH (release-blocking parity) | HIGH | P1 |
| Large-file editor guard | MEDIUM | LOW | P1 |
| Rename collision warning (409) | MEDIUM | LOW | P1 |
| Upload overwrite warning (409) | MEDIUM | LOW | P1 |
| Multi-file upload | MEDIUM | MEDIUM | P2 |
| Move across directories | MEDIUM | MEDIUM-HIGH | P2 |
| Concurrent-edit viewer count indicator | LOW-MEDIUM | MEDIUM | P3 |
| Filesystem-change auto-refresh | MEDIUM | HIGH | P3 (v3.6) |
| Right-click context menu (write ops) | LOW-MEDIUM | MEDIUM | P3 |
| Git diff integration | LOW | HIGH | OUT |
| Real-time multi-cursor (CRDT/OT) | LOW (wrong use case) | VERY HIGH | OUT |
| Auto-save | LOW (dangerous in AI session context) | LOW | OUT |
| Binary hex editor | LOW | HIGH | OUT |

**Priority key:**
- P1: Must have for v3.5 release (closes Issues #63, #64, #24)
- P2: Should have; add when core is validated
- P3: Nice to have; v3.6+
- OUT: Explicitly out of scope

---

## Competitor Feature Analysis

| Feature | VS Code / Codespaces | GitHub Web Editor | Gitea/Forgejo Web | AgentHub v3.5 (proposed) |
|---------|----------------------|-------------------|-------------------|--------------------------|
| Edit-and-save | CodeMirror (web) / Monaco (desktop) | CodeMirror 6 (github.dev) | CodeMirror 6 | CodeMirror 6 |
| Conflict detection | None (last-write-wins in Codespaces) | ETag/If-Match on raw file API | Optimistic locking via commit SHA | If-Match on mtime+size ETag (412 UX prompt) |
| Atomic write | Yes (VS Code uses tmp+rename) | Server-side (GitHub storage layer) | Server-side | Yes (tmp+rename within sandbox) |
| Delete | Confirmation modal | Confirmation modal | Confirmation with commit message | Confirmation modal (no commit) |
| Upload | Drag-and-drop, multi-file | Drag-and-drop | Web UI upload | Single file (v3.5); multi-file (v3.5 stretch) |
| Rename/move | Drag + F2, cross-directory | Web UI (limited) | Web UI | F2 inline, same-directory (v3.5); cross-dir stretch |
| TUI equivalent | Terminal + `$EDITOR` | N/A | N/A | `$EDITOR` shell-out (tea.Exec) |
| Remote writes | Local only in Codespaces | N/A (always remote) | N/A | Remote tailnet peer writes via HTTPS |
| Capability gating | Role-based (repo permissions) | Repo write permission | Repo write permission | `files.write` capability bit, per-session, opt-in for web-share |
| Auto-save | Yes (configurable) | Yes | No | No (intentionally — AI session context) |
| Syntax highlighting | Extensive language packs | Extensive language packs | CodeMirror 6 built-ins | CodeMirror 6 common language packs |

---

## Sources

**Architecture (HIGH confidence — authoritative project source):**
- v3.4 REQUIREMENTS.md (shipped, 48/48 requirements satisfied) — existing API contract, capability model, sandbox design, `os.OpenRoot` pattern, `requireFilesRead` middleware, `HasPerm` whole-token split
- PROJECT.md v3.5 milestone scoping — ratified scope, `files.write` design, editor library decision, phase numbering, TD-4/TD-5 carry
- v3.4 FEATURES.md (prior research) — read-side feature landscape and dependency patterns that v3.5 extends

**CodeMirror 6 (MEDIUM-HIGH confidence — well-documented community knowledge):**
- CodeMirror 6 is used by GitHub (github.dev, github.com file editor), Replit, and Glitch. ~200KB bundle, CSP-clean (no `eval`), extensible language packs. PROJECT.md and v3.4 REQUIREMENTS.md explicitly name CodeMirror 6 as the ratified choice, closing the Monaco vs CodeMirror decision.

**Atomic-write patterns (HIGH confidence — Go stdlib + POSIX spec):**
- `os.Rename` is atomic on POSIX when source and destination are on the same filesystem (POSIX `rename(2)` spec). Same-directory temp file satisfies this within the sandboxed `os.OpenRoot` scope. Go's `os.Rename` calls `MoveFileEx(MOVEFILE_REPLACE_EXISTING)` on Windows since Go 1.5+.

**ETag/If-Match conflict detection (HIGH confidence — HTTP spec + GitHub API patterns):**
- RFC 7232 defines `If-Match` / `ETag` semantics and 412 Precondition Failed. GitHub Content API uses ETag on file SHA for conflict detection. mtime+size shortcut is a standard REST API implementation simplification.

**`tea.Exec` for `$EDITOR` shell-out (HIGH confidence — established in AgentHub TUI attach):**
- The `tea.Exec` suspend/resume pattern is already used in AgentHub's TUI attach (`internal/tui/update.go`). The `$EDITOR` shell-out is structurally identical. No new primitives required.

---
*Feature research for: AgentHub v3.5 write-side file browser + in-app code editor*
*Researched: 2026-06-14*
