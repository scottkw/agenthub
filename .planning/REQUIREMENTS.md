# Milestone v3.4 Requirements — File Browser (Read-Only) + TUI Parity

**Defined:** 2026-05-20
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

**Milestone Goal:** Ship the read-only half of the file browser epic (GitHub Issue #24) — sandboxed filesystem API, desktop/web file browser tab, and TUI browse+preview parity. Let the sandboxed FS API bake under real use before the v3.5 write-side work lands.

**Closes GitHub Issues:** #62 (read-only file browser) + v3.4 slice of #64 (TUI browse+preview parity). Umbrella epic #24 stays open across v3.4 + v3.5; closes when v3.5 ships.

**Scope discipline (locked, deferred to v3.5):** write operations (upload/delete/rename/mkdir/edit); CodeMirror 6 vs Monaco editor library decision; TUI shell-out to `$EDITOR`; syntax highlighting in preview pane (markdown rendering is in v3.4, code-file highlighting is v3.5).

**Carry-forward from v3.3 (operator one-time, before next release):**

- `RELEASE_PUBLISH_TOKEN` PAT (`Contents: read/write` on `scottkw/agenthub`) — `gh secret set RELEASE_PUBLISH_TOKEN`
- `WINGET_FIRST_SUBMISSION=true` (one-time, first submission only) — `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Unset after winget-pkgs accepts first submission.

---

## v3.4 Requirements

### FS — Sandboxed Filesystem API (Phase 1)

The load-bearing foundation. `internal/files/` is a new sub-package with zero coupling to `internal/daemon`, `internal/relay`, or `internal/webserver`. Injected into both muxes via the `SetFilesHandlerProvider` pattern already established by `SetPluginSettingsProvider`. Sandboxing uses Go 1.24+ `os.OpenRoot` / `os.OpenInRoot` exclusively — the two-step `filepath.EvalSymlinks` + `os.Open` pattern has a TOCTOU race window demonstrated exploitable by CVE-2026-27976 (Zed) and CVE-2026-43998 (vm2). All file ops ride existing HTTPS over Tailscale; the relay's PTY fan-out protocol is NOT extended.

- [x] **FS-01** — `internal/files/` package exists with `Sandbox` type wrapping `*os.Root` scoped to a session's WorkDir; `List`, `Stat`, and `Read` methods reject paths via `os.OpenInRoot` (kernel-level TOCTOU-free sandbox), not via the legacy `filepath.EvalSymlinks` + `os.Open` two-step.
- [x] **FS-02** — `SessionEngine` gains a `sessionWorkDirs map[string]string` field (mirroring the established `tabNames` / `sessionCLIs` pattern) populated at `CreateSession` time with the resolved-absolute WorkDir (after `$HOME` substitution). Plugs the existing gap where WorkDir is passed to `cmd.Dir` and discarded after spawn.
- [x] **FS-03** — `GET /api/files/list?session=<id>&path=<rel>` returns directory listing as JSON array of `FileEntry{Name, Size, Mtime, Mode, IsDir, IsSymlink, IsBinary, MIME}`; directory entries are not stat'd recursively; uses `os.ReadDir` (streaming) not `ioutil.ReadDir`.
- [x] **FS-04** — `GET /api/files/stat?session=<id>&path=<rel>` returns single `FileEntry` for the named path.
- [x] **FS-05** — `GET /api/files/read?session=<id>&path=<rel>` streams file bytes via `http.ServeContent` with Range support; honors `If-Modified-Since` and Last-Modified.
- [x] **FS-06** — `HEAD /api/files/read?session=<id>&path=<rel>` is supported and returns Content-Length + Content-Type without body; frontend uses it for preflight before deciding inline-preview vs download warning. (Resolved OQ-1.)
- [x] **FS-07** — 0-byte files served by `/read` return 200 with empty body (NOT 416). Explicit unit test required — `http.ServeContent`'s default behavior is wrong here.
- [x] **FS-08** — Path sandbox rejects: absolute paths (including `C:\...` and `\\?\...` on Windows); paths containing `..` after `filepath.Clean`; encoded variants (`%2e%2e%2f`, `%252e%252e%252f`); Unicode path-traversal variants (U+FF0F fullwidth slash, U+2024 one-dot-leader); null bytes (`\x00`); Windows reserved device names (`CON`, `NUL`, `PRN`, `AUX`, `COM1`–`COM9`, `LPT1`–`LPT9`) cross-platform; Windows alternate data streams (`:` in path); Windows 8.3 short names (`PROGRA~1`); trailing dots/spaces on Windows; symlinks whose resolved target is outside the sandbox.
- [x] **FS-09** — `internal/files/sandbox_test.go` includes a `testing.F` fuzz test (`FuzzSandboxPath`) seeded with the 40+ payload corpus from `.planning/research/PITFALLS.md` §Fuzz Corpus. Fuzz run is a merge gate (`go test -fuzz=FuzzSandboxPath -fuzztime=60s ./internal/files/...` must report 0 crashes).
- [x] **FS-10** — New capability bit `files.read` added to the comma-separated `Claims.Perms` string in `internal/capability/capability.go`; new `HasPerm(perms, "files.read") bool` helper splits on commas (NOT `strings.Contains` — `"no-files.read"` would false-positive).
- [x] **FS-11** — New `requireFilesRead` middleware wrapper (separate from `requireCapability`) gates all three file endpoints. Adding `files.read` to `requireCapability`'s switch is explicitly rejected — would risk breaking existing terminal relay routes.
- [x] **FS-12** — Session-owner cap token issuance includes `files.read` in `Perms` by default; web-share viewer token issuance does NOT include `files.read` unless the share grant explicitly enables it (default OFF for viewers).
- [x] **FS-13** — Capability-denied test: a viewer token without `files.read` gets 403 on `/api/files/list`, `/api/files/stat`, and `/api/files/read`; verified for both GET and HEAD on `/read`.
- [x] **FS-14** — Settings `schemaVersion: 3` migration via the established `defaultSettings()` constructor-merge pattern from v3.2; per-field assertions in migration test for any new persisted file-browser settings (preview cap, default sort, etc.).

### WEB — WebServer Routes + `files.read` Capability Plumbing (Phase 2)

Wires the new `internal/files/` handler into `internal/webserver` and into the daemon's local-socket HTTP API via the `SetFilesHandlerProvider` pattern. The daemon and the webserver share the same handler; capability middleware sits in front of it on the webserver side only.

- [x] **WEB-01** — Daemon's local-socket HTTP API exposes `/api/files/list`, `/stat`, `/read` (GET + HEAD) for in-process GUI/TUI/CLI consumers; no cap-token middleware on this surface (local Unix-socket / named-pipe is already trusted).
- [x] **WEB-02** — Webserver mux exposes the same three endpoints under `/api/files/...` wrapped by `requireFilesRead`; routes are mounted via `SetFilesHandlerProvider` (no direct coupling between `internal/webserver` and `internal/files/`).
- [x] **WEB-03** — Read-only web-share viewer cannot use file browser endpoints: an explicit integration test asserts 403 with a viewer cap token across all three endpoints + both methods on `/read`.
- [x] **WEB-04** — Web-shared file browser works against tailnet-remote sessions via Tailscale HTTPS (NOT via the relay's binary frame protocol — relay is PTY fan-out only); the frontend uses `fetch()` against the remote peer's HTTPS base URL, same channel already used for tailnet peer discovery.
- [x] **WEB-05** — Zero new CSP amendments: existing `script-src 'self'` + `style-src 'self' 'unsafe-inline'` + `'wasm-unsafe-eval'` policy is sufficient for the new tab. Cross-browser Playwright e2e (Chromium + Firefox + WebKit) reports zero CSP violations from file browser flows.

### UI — FileBrowserTab.tsx (Desktop + Web) (Phase 3)

Single-pane list + side-by-side preview (NOT tree+list — collides with AgentHub's existing left sidebar). Filter activation key is `/` (NOT `Cmd-F` — keeps parity with TUI and the existing scrollback-search Cmd-F handler in xterm.js). Code-file syntax highlighting is OUT of v3.4 (deferred to v3.5 when CodeMirror 6 lands for editing); markdown rendering via `react-markdown@10.1.0` + `remark-gfm@^4` is in v3.4. (Resolved OQ-3.)

- [x] **UI-01** — New `frontend/src/components/FileBrowserTab.tsx` registered in the tab system; opens via session context menu ("Open file browser") and from the Sessions panel; singleton find-or-add per-session pattern consistent with Settings/DaemonManager/Remote tabs.
- [x] **UI-02** — Single-pane file list (left) + preview pane (right) layout; resize via splitter or fixed 60/40 split; no left tree pane (no nested sidebar inside the tab).
- [x] **UI-03** — File list sort by name / size / mtime, ascending or descending; directories sticky at top of each sort; column headers clickable to toggle sort.
- [x] **UI-04** — Type-ahead filter activated by `/` key (when tab is focused), Escape clears + dismisses. Matches against displayed names in the current directory; filter scope is current-directory-only (no recursive search in v3.4).
- [x] **UI-05** — Breadcrumb path bar at top of tab; each segment is clickable to navigate up; root segment is the session cwd (NOT filesystem root); user cannot navigate above cwd via any path (typed, pasted, or clicked).
- [x] **UI-06** — Preview pane renders text files up to a 5 MB server-enforced cap (Content-Length check before stream); over-cap or binary files show GitHub-style "Sorry, we can't display this file. View raw / Download." copy with a Download button wired to the Range-capable `/read` endpoint.
- [x] **UI-07** — Markdown files (`.md`, `.markdown`) render via `react-markdown@10.1.0` + `remark-gfm@^4` (GFM tables/task lists); NO `rehype-raw` (raw HTML passthrough — XSS risk in preview pane).
- [x] **UI-08** — Source code files (`.go`, `.ts`, `.tsx`, `.py`, etc.) render as monospaced plain text — NO syntax highlighting in v3.4. Code highlighting deferred to v3.5 when the editor lands.
- [x] **UI-09** — Image previews use `<img src="/api/files/read?session=...&path=...&cap=...">` (direct stream); explicitly NOT base64-in-state (33% overhead + GC pressure). Common types only: PNG, JPEG, WebP, GIF, SVG (rendered as text — never as embedded SVG).
- [x] **UI-10** — Download button per file uses the Range-capable `/api/files/read` endpoint; for files larger than the preview cap, the download path is the only way to retrieve full contents from the browser.
- [x] **UI-11** — File browser tab works against local AND remote (tailnet) sessions; the React component uses `fetch()` directly (NOT a new Wails binding — follows the precedent set by `TerminalPanel.tsx` connecting to the relay WSS endpoint directly).
- [x] **UI-12** — ARIA semantics: file list has `role="grid"` or `role="listbox"`, preview pane has `role="region"` with aria-label, breadcrumb has `role="navigation"`; keyboard-only operation works end-to-end (arrow keys, Enter into dir, Backspace/Cmd-Up to parent, Tab between panes); WCAG AA 4.5:1 contrast on selection states.
- [x] **UI-13** — Empty directory, network-error, and permission-denied states each render with explicit user-readable copy (NOT raw "403 Forbidden"). Permission-denied surfaces the missing `files.read` capability explicitly.
- [x] **UI-14** — Playwright e2e covers desktop + web paths for: open tab, list cwd, navigate into subdirectory, preview text file, preview markdown file, preview image, attempt to preview binary (refusal), attempt to preview over-cap file (refusal), download file (full + Range), capability-denied viewer (403). Required as Phase 3 merge gate.

### TUI — TUI Files View (Phase 4)

Custom `bubbles list.Model` + `viewport.Model` joined via `lipgloss.JoinHorizontal` (NOT `bubbles/filepicker` — that's a selection-dialog primitive, not a browse pane). All filesystem I/O via `tea.Cmd` — synchronous `os.ReadDir` freezes the Bubble Tea render loop. Path truncation in the status line is left-truncated (`…/utils/helper.ts`) — preserves the high-information leaf-end. (Resolved OQ-2.) Can run in parallel with Phase 3 once Phase 1 freezes.

- [x] **TUI-01** — New `internal/tui/files.go` Bubble Tea sub-model with custom `bubbles list.Model` (file list) + `viewport.Model` (preview pane) joined via `lipgloss.JoinHorizontal`; bordered lipgloss frame with TokyoNight palette consistent with existing TUI tabs.
- [x] **TUI-02** — New "Files" sidebar entry (or reachable per-session via the Sessions list); opens scoped to the selected session's cwd; closing the file view returns to the prior TUI tab.
- [x] **TUI-03** — Navigation: Up/Down arrow keys move the list cursor; PageUp/PageDown jump a page; Enter enters a directory; Backspace or Left arrow goes up; user cannot navigate above session cwd (Backspace at cwd root is a no-op, NOT a parent traversal).
- [x] **TUI-04** — Read-only preview pane shows text files (5 MB server-enforced cap); markdown files rendered via `charmbracelet/glamour` (promoted from indirect to direct dep); binary files show "Use desktop or web to preview"; over-cap files show "Too large to preview, use desktop or web to download."
- [x] **TUI-05** — Type-ahead filter activated by `/` key (parity with desktop); current-directory only; Escape clears and dismisses.
- [x] **TUI-06** — Status line shows session-cwd-relative path (left-truncated when wider than the pane: `…/utils/helper.ts`) + file count + selection position.
- [x] **TUI-07** — ALL filesystem I/O routed through `tea.Cmd` (returning `tea.Msg`); a synchronous `os.ReadDir` in Update is a merge-gate failure.
- [x] **TUI-08** — TUI Files view works against local AND remote (tailnet) sessions; uses the same daemon-local HTTP API for local and Tailscale HTTPS for remote (no relay frames).
- [x] **TUI-09** — Help overlay (`?` key) updated with file-browser keybindings: `↑/↓` `PgUp/PgDn` `Enter` `Backspace` `/` `?` `Esc` `q`.
- [x] **TUI-10** — Key-dispatch priority updated in `internal/tui/update.go` to handle file-browser modal correctly (above main view but below kill-confirm/new-session/QR overlay/help).

## v3.5 Requirements (Deferred, not in v3.4 roadmap)

Tracked here for visibility; will be promoted to active during v3.5 milestone start.

### Write Operations

- **WRITE-01** — `POST /api/files/write` with If-Match conflict detection
- **WRITE-02** — `POST /api/files/upload` (chunked) with capability bit `files.upload`
- **WRITE-03** — Rename / delete / mkdir endpoints with capability bit `files.write`

### Editor Integration

- **EDIT-01** — CodeMirror 6 (subject to ratification at v3.5 plan time) integrated into FileBrowserTab.tsx
- **EDIT-02** — Cmd/Ctrl+S save; conflict UI on If-Match 412 mismatch
- **EDIT-03** — Syntax highlighting via CodeMirror language packs (replaces v3.4 plain-text code rendering)

### Upload UI + Remote Parity

- **UPLD-01** — Drag-and-drop chunked upload with progress
- **UPLD-02** — Verified denial for read-only web-share viewers

### TUI Edit

- **TUI-EDIT-01** — TUI shells out to `$EDITOR` on Enter/`e` against a text file
- **TUI-EDIT-02** — `$EDITOR` unset → clear error message pointing to docs
- **TUI-EDIT-03** — Upload-from-TUI formally implemented or formally descoped with follow-up issue filed

## Out of Scope (v3.4)

| Feature | Reason |
|---------|--------|
| Recursive search across the cwd subtree | Current-directory-only filter ships first; recursion adds complexity (perf + cap-respect) better validated after read-only bakes |
| Path-paste input box for breadcrumb | Click-segment navigation only; paste-jump can land in v3.5 with the editor |
| Right-click context menu | All actions reachable via keyboard + buttons; context-menu UX better lands with write ops in v3.5 |
| Drag-out file download | Browser-native download via Range endpoint is sufficient; drag-out adds platform-specific complexity |
| Filesystem-watcher push notifications | Refresh-on-click is the v3.4 coherence contract; SSE/WS push is a v3.5+ concern |
| Cloud Commander integration | Native UI locked in epic — Cloud Commander's standalone-Node-server auth model conflicts with daemon-centric capability model |
| Monaco Editor | CodeMirror 6 recommended in epic (~200KB vs ~5MB, CSP-clean); decision ratified at v3.5 plan time, not v3.4 |
| Syntax highlighting for code files | Deferred to v3.5 when editor lands — preview-only highlighting via shiki adds ~200KB bundle for v3.4-only value, low ROI |
| In-TUI text editor | Shell-out to `$EDITOR` is the v3.5 path (matches `git commit` / `crontab -e` ergonomics) |
| Path-paste input in TUI breadcrumb | TUI navigation is keyboard-driven by design |
| HEAD on `/api/files/list` or `/stat` | Stat already returns full metadata; HEAD on `/read` is the only preflight needed |
| `Claims.Perms` schema change (struct/array vs comma-string) | Established v3.1+ pattern is comma-string; `HasPerm` helper handles whole-token match cleanly |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| FS-01 | Phase 118 | Complete |
| FS-02 | Phase 118 | Complete |
| FS-03 | Phase 118 | Complete |
| FS-04 | Phase 118 | Complete |
| FS-05 | Phase 118 | Complete |
| FS-06 | Phase 118 | Complete |
| FS-07 | Phase 118 | Complete |
| FS-08 | Phase 118 | Complete |
| FS-09 | Phase 118 | Complete |
| FS-10 | Phase 118 | Complete |
| FS-11 | Phase 118 | Complete |
| FS-12 | Phase 118 | Complete |
| FS-13 | Phase 118 | Complete |
| FS-14 | Phase 118 | Complete |
| WEB-01 | Phase 119 | Complete |
| WEB-02 | Phase 119 | Complete |
| WEB-03 | Phase 119 | Complete |
| WEB-04 | Phase 119 | Complete |
| WEB-05 | Phase 119 | Complete |
| UI-01 | Phase 120 | Complete |
| UI-02 | Phase 120 | Complete |
| UI-03 | Phase 120 | Complete |
| UI-04 | Phase 120 | Complete |
| UI-05 | Phase 120 | Complete |
| UI-06 | Phase 120 | Complete |
| UI-07 | Phase 120 | Complete |
| UI-08 | Phase 120 | Complete |
| UI-09 | Phase 120 | Complete |
| UI-10 | Phase 120 | Complete |
| UI-11 | Phase 120 | Complete |
| UI-12 | Phase 120 | Complete |
| UI-13 | Phase 120 | Complete |
| UI-14 | Phase 120 | Complete |
| TUI-01 | Phase 121 | Complete |
| TUI-02 | Phase 121 | Complete |
| TUI-03 | Phase 121 | Complete |
| TUI-04 | Phase 121 | Complete |
| TUI-05 | Phase 121 | Complete |
| TUI-06 | Phase 121 | Complete |
| TUI-07 | Phase 121 | Complete |
| TUI-08 | Phase 121 | Complete |
| TUI-09 | Phase 121 | Complete |
| TUI-10 | Phase 121 | Complete |
| REMOTE-01 | Phase 122 | Pending |
| REMOTE-02 | Phase 122 | Pending |
| REMOTE-03 | Phase 122 | Pending |
| REMOTE-04 | Phase 122 | Pending |
| REMOTE-05 | Phase 122 | Pending |

## REMOTE — Cross-Surface Remote-Session File Browse (Phase 122)

- **REMOTE-01:** Desktop GUI's React `FileBrowserTab` opens against a remote tailnet session by passing `baseURL = <remote-tailnet-URL>` + `capToken = <session's existing web-share cap>`. No new cap-minting flow — reuse the session's web-share cap.
- **REMOTE-02:** If a remote session has NOT been web-shared, the desktop GUI shows "Enable web sharing to browse this session's files" instead of opening a broken tab.
- **REMOTE-03:** TUI Files view opens against a remote tailnet session by fetching from the remote webserver over HTTPS with the session's cap token (not the local Unix socket). The previous "File browser not available for remote sessions" toast is removed.
- **REMOTE-04:** TUI behavior is identical between local and remote sessions: keyboard nav, preview, filter, status line, glamour markdown, binary/over-cap refusals.
- **REMOTE-05:** Cross-surface parity: a viewer with `files.read` on a session can browse that session's files from desktop GUI, web browser, OR TUI with the same observable behavior.

**Coverage:**

- v3.4 requirements: 48 total (FS: 14, WEB: 5, UI: 14, TUI: 10, REMOTE: 5)
- Mapped to phases: 48/48
- Unmapped: 0 ✓

---

*Requirements defined: 2026-05-20*
*Last updated: 2026-05-21 — REMOTE-01..05 added for Phase 122 (cross-surface remote browse wiring).*
