# Milestone v3.5 Requirements — File Browser: Write Operations & Editor

**Defined:** 2026-06-14
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

**Milestone Goal:** Ship the write-side half of the file-browser epic (GitHub Issue #24) with full cross-surface parity — write/upload/delete/rename/move/mkdir endpoints, an in-app CodeMirror 6 editor with syntax highlighting (replacing v3.4 plain-text rendering), TUI `$EDITOR` shell-out, an opt-in `files.write` web-share capability, and remote tailnet peer write parity. Folds in carried tech-debt TD-4 and TD-5.

**Closes GitHub Issues:** #63 (editing + upload) + #64 (TUI edit parity). Umbrella epic #24 closes when v3.5 ships.

**Cross-surface parity is a release-blocking contract** (established v3.3 Phases 107/108, reaffirmed v3.4 Phase 122). Every write operation must behave equivalently across GUI / web-share / TUI, with the one intentional gap documented (TUI upload — see TUIW-06).

---

## Scope Decisions (ratified 2026-06-14 at milestone scoping)

| Decision | Choice | Implication |
|----------|--------|-------------|
| Web-share writes | IN scope | New opt-in `files.write` capability bit grantable to web-share viewers; requires a dedicated security phase (SEC). |
| Remote tailnet peer writes | IN scope | Full write parity on remote sessions, mirroring v3.4 read parity (RMW). |
| Editor library | CodeMirror 6 (research-ratified) | Monaco rejected — its `worker-src blob:` CSP requirement violates the vendored-only/single-binary discipline. Zero new CSP amendments. |
| **Owner `files.write` default** | **Default-ON for the session owner** | Owner token carries `files.write` by default (mirrors `files.read`); web-share viewers remain opt-in. **Makes the server-side shell-RC denylist (FSW-06) and home-dir write warning (CAP-06) release-critical**, since an owner's home-directory session now carries write by default. |
| Multi-file upload | IN scope (P1) | Batched upload-queue UI with per-file progress in the React editor phase. |
| Cross-directory move | IN scope (P1) | "Move to…" picker UI; the `rename` endpoint validates both paths and is already move-capable. |
| Upload size cap | Hardcoded 50 MiB | `http.MaxBytesReader` before `ParseMultipartForm`. Configurable `daemonSettings.UploadMaxBytes` deferred to on-demand. |
| Auto-save | OUT (anti-feature) | AI agents watch the filesystem; auto-saving partial edits would inject corrupt files into a live coding session. Explicit Cmd/Ctrl+S only. |

**Carry-forward operator one-time tasks (still required before next release):**

- `RELEASE_PUBLISH_TOKEN` PAT (`Contents: read/write` on `scottkw/agenthub`) — `gh secret set RELEASE_PUBLISH_TOKEN`
- `WINGET_FIRST_SUBMISSION=true` (one-time, first submission only) — `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Unset after winget-pkgs accepts first submission.

---

## v3.5 Requirements

### FSW — Filesystem Write Primitives + Daemon Routes + TD Cleanup

The load-bearing security foundation. Extends the v3.4 `internal/files/` `Sandbox` (wrapping `*os.Root`) with write methods, all kernel-sandboxed via `os.Root` / `os.OpenInRoot` (TOCTOU-free). Daemon local-socket write routes are auth-less (loopback trust, same as v3.4 reads — WEB-01 precedent). No capability model changes in this category. TD-5 lands first because the desktop GUI cannot acquire a remote cap today, blocking all remote-write testing.

- [ ] **FSW-01** — `Sandbox.WriteFileAtomic(relPath, content)` writes to a sibling temp file inside the sandbox root (`relPath + ".agenthub-tmp-" + randomHex()`), calls `f.Sync()`, then atomic-renames to the target. No `O_TRUNC` in-place writes — a concurrent AI-agent reader must never observe an empty or partial file. Temp file stays within the sandbox (never a system temp dir — that escapes the sandbox and loses cross-filesystem rename atomicity).
- [ ] **FSW-02** — `Sandbox.Rename(oldRel, newRel)` validates **both** source AND destination relative paths through `validateAndClean` before constructing sandbox-absolute paths and calling `os.Rename`. `os.Root.Rename` does NOT exist in Go 1.26 (golang/go#69462). Destination-path traversal is rejected identically to source — failing to validate the destination is the #1 write-side traversal risk. Supports same-directory rename AND cross-directory move (both paths sandbox-validated).
- [ ] **FSW-03** — `Sandbox.Mkdir` / `Sandbox.MkdirAll` create directories within the sandbox via iterative `root.Mkdir`; traversal payloads rejected.
- [ ] **FSW-04** — `Sandbox.Delete(relPath)` removes a file or recursively removes a directory subtree; the recursive walk is guaranteed to stay within the sandbox root.
- [ ] **FSW-05** — Upload write path streams a multipart part to disk via `Sandbox.WriteFileAtomic`; the destination filename is sanitized via `filepath.Base(header.Filename)` then `validateAndClean` (never trust `multipart.FileHeader.Filename` — it can contain `../`).
- [ ] **FSW-06** — Server-side sensitive-file denylist enforced inside ALL `Sandbox` write methods (write/rename/delete/mkdir): reject operations targeting shell RC files (`~/.bashrc`, `~/.zshrc`, `~/.profile`, `~/.bash_profile`), `~/.ssh/*`, `~/.claude/*`, and the daemon's own config dir, returning `403 Protected system file`. **Load-bearing** — owner `files.write` defaults on, and AI sessions commonly run with `cwd=$HOME`, so the sandbox is technically correct but the consequences (overwriting SSH keys / shell init) are severe without this control.
- [ ] **FSW-07** — `FuzzSandboxWrite` (`testing.F`) extends the v3.4 `FuzzSandboxPath` corpus with write-path, rename-destination-traversal, and upload-filename-injection payloads; `go test -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/...` reporting 0 crashes is a merge gate.
- [ ] **FSW-08** — Daemon local-socket HTTP API exposes write routes for in-process GUI/TUI consumers, auth-less on the local Unix-socket / named-pipe surface (WEB-01 trust precedent): `PUT /api/files/write`, `POST /api/files/upload`, `DELETE /api/files/delete`, `POST /api/files/rename`, `POST /api/files/mkdir`.
- [ ] **FSW-09** — `DaemonClient` gains `WriteFile`, `UploadFile`, `DeleteFile`, `RenameFile`, `MkdirFile` methods consuming the daemon write routes.
- [ ] **FSW-10** — **TD-5 cleanup:** fix `DaemonClient.ExchangeJoinCodeAtURL` 303-shim — it currently JSON-decodes a 303 redirect body (which has none) and silently fails, so the desktop GUI cannot acquire a remote cap at all. Disable auto-redirect follow, detect `StatusCode == 303`, parse `?cap=<token>` from the `Location` header (the TUI `exchangeJoinCodeCmd` already does this correctly — reference implementation). Prerequisite for all remote-write work (RMW).
- [ ] **FSW-11** — **TD-4 cleanup:** Phase 120 WR-01..05 file-browser hardening — `/app/` directory listings, write-route cache-control headers, `joinPath` sanitization, mtime fallback, and comment clarity addressed while the file layer is open.
- [ ] **FSW-12** — 50 MiB upload size cap enforced via `http.MaxBytesReader` applied BEFORE `r.ParseMultipartForm`; over-cap upload returns a clear error, not a truncated file. Cap is hardcoded for v3.5 (configurable `daemonSettings.UploadMaxBytes` deferred).

### CAP — `files.write` Capability + Webserver Write Routes + Web-Share Opt-In

Wires the write handler into `internal/webserver` behind a new capability bit and CSRF protection. The capability model mirrors v3.4's `files.read`/`requireFilesRead` exactly, with one ratified difference: the owner token carries `files.write` by default.

- [ ] **CAP-01** — `PermFilesWrite = "files.write"` constant added to `internal/capability/capability.go`; gated via the existing `HasPerm` whole-token comma-split helper (NOT `strings.Contains` — `"no-files.write"` would false-positive).
- [ ] **CAP-02** — New `requireFilesWrite` middleware (separate from both `requireCapability` and `requireFilesRead`) gates all five webserver write routes. Adding `files.write` to `requireCapability`'s switch is explicitly rejected (would risk breaking terminal relay routes — same separation that was load-bearing in v3.4).
- [ ] **CAP-03** — `requireFilesWrite` adds a CSRF `Origin` check for state-changing methods (POST/PUT/PATCH/DELETE): reject when an `Origin` header is present and does not match the server FQDN (mirrors the Phase 88 WebSocket Origin check). Desktop GUI Wails `fetch()` sends no `Origin`, so local requests pass vacuously. v3.4's read surface was GET-only and CSRF-safe by convention; write verbs are not.
- [ ] **CAP-04** — Session-owner cap-token issuance includes `files.write` in `Perms` by default (ratified owner-default-on decision). Web-share viewer token issuance does NOT include `files.write` unless the share grant explicitly enables it (default OFF for viewers).
- [ ] **CAP-05** — Web-share grant UI exposes an explicit `files.write` opt-in toggle (default OFF), separate from `files.read`, with a confirmation explaining: "This will allow the recipient to create, edit, delete, rename, and upload files in this session's working directory."
- [ ] **CAP-06** — Home-directory write warning: when `files.write` is active for a session whose resolved cwd is `$HOME` (or a sensitive ancestor), the GUI and TUI surface a visible warning that file writes can affect dotfiles, SSH keys, and shell config. Release-critical given owner-default-on.
- [ ] **CAP-07** — Webserver write routes mounted via the established `SetFilesHandlerProvider` pattern (no direct coupling between `internal/webserver` and `internal/files/`); daemon and webserver share the same handler, capability middleware sits in front on the webserver side only.
- [ ] **CAP-08** — Settings `schemaVersion: 4` migration via the `defaultSettings()` constructor-merge pattern; per-field assertions in the migration test; web-share `files.write` opt-in persists with default `false`.
- [ ] **CAP-09** — Capability-denied integration tests: a viewer token without `files.write` receives 403 on all five write endpoints (correct method coverage per endpoint); an owner/granted token receives 2xx. Static-grep gate asserts `HasPerm` (not `strings.Contains`) on the write path.
- [ ] **CAP-10** — Remote daemon-proxy write routes: `/api/files/remote/{sid}/{write,upload,delete,rename,mkdir}`; `proxyRemoteFiles` forwards `r.Body` for PUT/POST/PATCH methods (it currently passes `nil`, correct only for GET/HEAD).

### EDIT — React Editor (CodeMirror 6) — Desktop + Web

The milestone centrepiece. CodeMirror 6 replaces the v3.4 plain-text `<pre>` preview for editable files. Library selection is research-ratified — no decision gate at plan time. Depends on FSW (write API frozen) + CAP (capability live).

- [ ] **EDIT-01** — CodeMirror 6 (`@codemirror/view`, `/state`, `/commands`, `/lang-*`, `/language-data`) installed via pnpm and vendored under `web/vendor/codemirror/`; `vendor_drift_test.go` extended to enforce CodeMirror version parity between `package.json` and vendored assets. Zero new CSP amendments (no `worker-src` needed; `style-src 'unsafe-inline'` already covers CodeMirror's inline style injection).
- [ ] **EDIT-02** — `Editor.tsx` wrapper mounts CodeMirror 6 and toggles read-only↔editable via `Compartment.reconfigure()` (no remount); replaces the v3.4 plain-text preview pane for editable text files.
- [ ] **EDIT-03** — Explicit read-only→edit toggle (pencil button) in the preview pane header; NO auto-edit-on-open. The Edit button is hidden when `IsBinary` is true or when the caller lacks `files.write`.
- [ ] **EDIT-04** — Syntax highlighting via CodeMirror language packs with extension-based language detection (Go, TS/TSX, JS/JSX, Python, JSON, YAML, Markdown, Bash/shell, HTML, CSS, and other common languages).
- [ ] **EDIT-05** — Cmd/Ctrl+S save intercepts the browser default and calls `PUT /api/files/write` with an `If-Match: <etag>` header (ETag = mtime + size, not full-content hash).
- [ ] **EDIT-06** — Dirty-state indicator (modified bullet/asterisk in the editor header) keyed off `EditorState.doc` vs the saved-content snapshot; three-state save feedback (idle / saving / saved, ~1.5s transient — matches the Settings Save pattern).
- [ ] **EDIT-07** — Unsaved-changes warning on navigate-away — a React-level guard (NOT browser `beforeunload`, which Wails blocks) on file-switch within the tab, navigating up the tree, and tab close: "You have unsaved changes. Save or discard?"
- [ ] **EDIT-08** — 412 conflict UX: on `If-Match` mismatch the editor surfaces "This file was modified by another process" with [Force overwrite] / [Save as new file] / [Discard my changes].
- [ ] **EDIT-09** — Write actions in `FileBrowserTab`: create file (inline name input), mkdir (inline name input), delete (file + directory; directory confirmation states the recursive file count — "Delete `dir` and all N files inside? This cannot be undone."), rename (inline F2), and cross-directory move via a "Move to…" picker. Rename/move name collision returns 409 → "A file named X already exists. Replace it?" (Cancel default / Replace).
- [ ] **EDIT-10** — Upload: single + multiple files via `<input type="file" multiple>` and drag-and-drop into the directory listing; an upload queue shows per-file progress; existing-name collision shows a 409 overwrite warning before completing.
- [ ] **EDIT-11** — Large-file editor guard: open-for-edit on files > 500 KB warns "This is a large file (N MB). Edits may be slow." (warn-then-proceed, not block); files near the 5 MB cap disable syntax highlighting (plain-text mode) with "Syntax highlighting disabled for large files."
- [ ] **EDIT-12** — `useFilesWrite` hook + `useFilesCapability.canWrite`; all write affordances (Edit button, create/delete/rename/move/mkdir/upload controls) are gated on `canWrite`.
- [ ] **EDIT-13** — Playwright cross-browser e2e (Chromium + Firefox + WebKit) — Phase merge gate — covers: local write-and-save, web-share write with a `files.write` cap, 403 without the cap, create file, mkdir, delete file, delete directory (recursive confirm), rename, cross-directory move, single upload, multi-file upload, 412 conflict flow, binary-file no-edit, large-file guard.

### TUIW — TUI Write Parity (`$EDITOR` Shell-Out)

Extends the shared `FilesClient` interface and the TUI Files view with write operations. Edit is via `$EDITOR` shell-out (the established Unix pattern — `git commit`, `crontab -e`); no in-TUI CodeMirror. Depends only on FSW; can run in parallel with EDIT.

- [ ] **TUIW-01** — The shared `FilesClient` interface grows from 4 read methods to 8 (add `WriteFile`, `DeleteFile`, `RenameFile`, `MkdirFile`); both `*daemon.DaemonClient` and `*tui.RemoteFilesClient` satisfy all 8, preserving the duck-typed parity contract that drives one pipeline for local AND remote.
- [ ] **TUIW-02** — TUI edit on the `e` key uses `tea.Exec` to suspend the TUI, spawn `$EDITOR` with the file's sandbox-absolute path, and resume; the write-back / refresh routes through a `tea.Cmd` (never synchronous I/O in `Update`).
- [ ] **TUIW-03** — `resolveEditor()` fallback chain (`$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi`); if none resolves, show a clear error: "`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)."
- [ ] **TUIW-04** — `tea.ClearScreen` issued on editor-exit completion; the directory listing refresh (`loadDirCmd`) runs unconditionally post-exec so the listing is never stale after an edit.
- [ ] **TUIW-05** — TUI delete (`d` + confirmation dialog reusing the existing kill-session confirmation pattern), rename (`r` inline), and mkdir (`m` inline name input) — full cross-surface parity for these three operations.
- [ ] **TUIW-06** — TUI upload is formally descoped with an on-screen "Use desktop or web to upload files." message (the one intentional parity gap, mirroring the v3.4 image-preview gap); a follow-up GitHub issue is filed.
- [ ] **TUIW-07** — `TestFiles_NoSyncFSCalls` static-grep merge gate extended to cover the new write commands — all write filesystem I/O must route through `tea.Cmd`.

### SEC — Web-Share Write Security Hardening

The dedicated security-audit phase for the most-exposed surface. All four research files flag web-share write as requiring its own phase. Depends on CAP + EDIT (write routes live).

- [ ] **SEC-01** — Write-path symlink-escape test: a write/rename/mkdir whose resolved target escapes the sandbox via a symlink returns 403, not 200.
- [ ] **SEC-02** — Shell-RC denylist enforcement tests: attempts to write or rename into `~/.bashrc`, `~/.ssh/authorized_keys`, `~/.claude/CLAUDE.md`, and the daemon config within a home-directory sandbox return `403 Protected system file`.
- [ ] **SEC-03** — Upload abuse coverage: multipart filename injection (`../` in `FileHeader.Filename`) is sanitized; over-cap upload (> 50 MiB) is rejected by `MaxBytesReader`; directory/zip upload is not silently accepted (no zip-slip surface introduced).
- [ ] **SEC-04** — Capability-escalation audit: no token lacking `files.write` reaches any write endpoint on any surface; `files.write` does not leak across sessions; findings documented in a SECURITY artifact.
- [ ] **SEC-05** — Data-integrity tests: concurrent-write race (two writers + If-Match) and atomic-rename failure paths (disk-full / interrupted write) leave no corrupt or partial target file; the original is preserved on failure.
- [ ] **SEC-06** — `FuzzSandboxWrite` corpus finalized with rename-destination traversal, denylist-bypass, and upload-filename-injection payloads; merge gate reports 0 crashes.
- [ ] **SEC-07** — Playwright web-share write e2e: a viewer granted `files.write` writes successfully; a viewer without it gets 403; a CSRF Origin-mismatch request is rejected.

### RMW — Remote Write Parity + Cross-Surface Integration

The final integration phase. Mirrors the v3.4 Phase 122 read-parity proof with a 3-observer write-parity test. Depends on the full local write stack (FSW + CAP + EDIT + TUIW + SEC) and on FSW-10 (TD-5).

- [ ] **RMW-01** — Remote write parity proven by 3 independent observers — daemon-proxy Go + `tui.RemoteFilesClient` Go + Playwright HTTPS browser — establishing byte-equivalent behavior, mirroring the Phase 122 read-parity proof.
- [ ] **RMW-02** — Desktop GUI remote write end-to-end via the daemon proxy: edit/save, upload, delete, rename, cross-directory move, and mkdir on a remote tailnet peer session.
- [ ] **RMW-03** — TUI remote write end-to-end via `RemoteFilesClient` over HTTPS (TLS 1.2+ pinned; cap token redacted from error messages).
- [ ] **RMW-04** — Remote peer version gate: a write attempt against a v3.4 peer (no write endpoints) returns 405 → the client shows "The remote session is running an older version of AgentHub that does not support file writes." (NOT a generic network error).
- [ ] **RMW-05** — Cap-expiry-mid-edit behavior: the editor buffer is preserved and an "access expired" message is shown (no silent buffer loss); any orphaned partial upload is cleaned up.
- [ ] **RMW-06** — No regression on the Phase 122 remote-read test suite; a two-machine tailnet write UAT checklist is prepared (Machine A web-share + Machine B GUI + Machine B TUI; cross-surface write parity + cap-expiry failure mode).

---

## Future Requirements (Deferred)

Tracked for visibility; not in the v3.5 roadmap.

- **Filesystem-change auto-refresh** (inotify / FSEvents / ReadDirectoryChangesW push via SSE/WS) — refresh-on-navigate covers the MVP; trigger: user-reported friction with stale listings. (v3.6+)
- **Concurrent-edit viewer-count indicator** ("N others editing this file") — If-Match already covers safety; this is a DX nice-to-have. Trigger: multi-viewer write collisions reported.
- **Right-click context menu for write ops** — discoverability polish; all actions reachable via keyboard + buttons at launch.
- **Configurable upload size cap** (`daemonSettings.UploadMaxBytes` + Settings UI) — 50 MiB hardcoded ships first; add on demand.
- **Drag-and-drop move within the file list** — explicit "Move to…" picker ships first; gesture-based move is polish.
- **TUI upload** — formally descoped in v3.5 (TUIW-06) with a follow-up issue; revisit on demand.

## Out of Scope (v3.5)

| Feature | Reason |
|---------|--------|
| Monaco Editor | Requires `worker-src blob:` CSP amendment incompatible with the vendored-only/single-binary discipline; ~2.4–6 MB bundle vs CodeMirror 6 ~135 KB. CodeMirror 6 ratified. |
| In-TUI CodeMirror rendering | TUI is a real terminal, not a WebView; `$EDITOR` shell-out is the correct Unix pattern. |
| Auto-save (keystroke/timer) | AI agents watch the filesystem; auto-saving partial edits would feed corrupt files into a live coding session. Explicit Cmd/Ctrl+S only. |
| Real-time multi-cursor collab (CRDT/OT) | Wrong architecture for AgentHub's single-operator + AI model; If-Match conflict detection covers the actual concurrency risk. A separate milestone if ever. |
| Git integration (diff/stage/commit) | Separate epic; terminal tabs already provide full git CLI access. |
| Recursive project-wide file search | Current-directory `/` filter (v3.4) + terminal `rg`/`find` cover the cases; recursive search over remote relay is expensive and unscoped. |
| Binary hex editor | Separate product-grade feature; binary files keep the download-only affordance. |
| Upload entire directory (zip/unzip) | Browser multi-file drag loses directory structure; server-side zip extraction adds a zip-slip attack surface. Multi-file-within-directory upload covers the need. |
| Per-session write audit log | New persistent store + UI not justified for v3.5; bash history + git log cover the audit need. |
| Drag-out-to-filesystem download | Blocked by the browser/WebView security model; the Range-capable download button is sufficient. |
| CLI file-browse/write commands | CLI interacts with files via the terminal session itself; consistent with v3.4 (no CLI file commands). Not a parity gap. |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| _All FSW / CAP / EDIT / TUIW / SEC / RMW requirements_ | _assigned by roadmapper_ | Pending |

> Traceability is filled by the roadmapper (`/gsd:new-milestone` step 10) when phases are derived. Phase numbering continues from v3.4 (last phase 122) — v3.5 begins at Phase 123.

**Coverage (pre-roadmap):**

- v3.5 requirements: 55 total (FSW: 12, CAP: 10, EDIT: 13, TUIW: 7, SEC: 7, RMW: 6)
- Carried tech-debt folded in: TD-4 (FSW-11), TD-5 (FSW-10)

---

*Requirements defined: 2026-06-14*
*Research basis: `.planning/research/SUMMARY.md` (HIGH confidence) + 4 dimension files (STACK / FEATURES / ARCHITECTURE / PITFALLS).*
