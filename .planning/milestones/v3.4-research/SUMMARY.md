# Research Summary — v3.4 File Browser (Read-Only) + TUI Parity

**Project:** AgentHub v3.4
**Domain:** Sandboxed read-only filesystem API + desktop/web file browser tab + TUI browse/preview parity
**Researched:** 2026-05-20
**Confidence:** HIGH (all four research files verified against official docs, Go stdlib issue tracker, and actual AgentHub source)

---

## Executive Summary

AgentHub v3.4 adds a sandboxed read-only file browser — a new capability on top of a mature daemon-centric, capability-token-authed, Tailscale-TLS stack that has shipped 22 milestones without a file API. The architecture is well-constrained: a new `internal/files/` package (pure, zero coupling) provides `List`, `Stat`, and `Read` HTTP handlers; the daemon and webserver each mount these handlers with their respective auth layers (Unix-socket loopback = no auth; Tailscale HTTPS = `requireFilesRead` capability middleware); a new `FileBrowserTab.tsx` provides the desktop/web surface; and a new `internal/tui/files.go` provides TUI parity. All four research documents converge on the same build order: sandbox core first, then webserver capability plumbing, then React tab, then TUI view — with TUI parallelizable against the React tab after Phase 1 freezes.

The recommended stack is narrow. On the Go side, `os.OpenRoot` (stdlib, Go 1.24+, project uses 1.26.1) is the only viable sandbox primitive — it eliminates the TOCTOU race that makes every two-step EvalSymlinks+Open approach exploitable. `http.ServeContent` (stdlib) handles Range requests. `github.com/wailsapp/mimetype` is promoted from indirect to direct dep for accurate MIME detection. On the frontend, `react-markdown@10.1.0` + `remark-gfm@^4` render markdown without CSP impact, and `shiki@4.1.0` with `createJavaScriptRegexEngine()` provides CSP-clean syntax highlighting if the planner opts to ship it in v3.4. The TUI uses custom `list.Model` + `viewport.Model` (NOT `filepicker`) paired with `charmbracelet/glamour@v0.8.0` for markdown. Zero new CSP amendments are required under any of these choices.

The most significant risks are all in Phase 1: the TOCTOU symlink race (must use `os.OpenRoot` exclusively, never two-step), Windows reserved device names and alternate data streams (require explicit cross-platform rejection before `os.OpenRoot`), and the `HasPerm` whole-token match for `files.read` (using `strings.Contains` introduces a prefix-match false-positive). The fuzz corpus must ship in the same PR as the first file endpoint — it is a merge gate. Every other pitfall (TUI async I/O, image-via-src-URL not base64, `rehype-raw` exclusion, 0-byte 416 fix) is a design-in concern for the phase that introduces the relevant surface, with concrete checklist items the planner can encode directly.

---

## Key Findings

### Recommended Stack

The v3.4 stack is almost entirely stdlib and already-in-go.mod promotions. The only net-new Go change is promoting two transitive deps to direct. On the frontend, three npm packages are genuinely new (`react-markdown`, `remark-gfm`, `shiki`) but none require CSP changes and none touch the `web/vendor/` vendored xterm pipeline, so `vendor_drift_test.go` is unaffected.

**Core technologies (new or promoted for v3.4):**
- `os.OpenRoot` / `os.OpenInRoot` (Go stdlib, 1.24+): sandbox primitive — atomic TOCTOU-safe open, blocks traversal at kernel level
- `http.ServeContent` (Go stdlib): Range-capable file streaming — handles 206, ETags, If-Modified-Since, no new dep
- `github.com/wailsapp/mimetype@v1.4.1`: promote indirect→direct — 200+ MIME types vs stdlib's ~15, already in go.sum
- `charmbracelet/glamour@v0.8.0`: promote indirect→direct — ANSI markdown rendering for TUI preview pane
- `charm.land/bubbles/v2` `list.Model` + `viewport.Model`: already direct dep — custom TUI file browser (NOT `filepicker`)
- `react-markdown@10.1.0` + `remark-gfm@^4`: markdown rendering in React preview pane — virtual DOM, no `dangerouslySetInnerHTML`, no CSP impact
- `shiki@4.1.0` with `createJavaScriptRegexEngine()`: syntax highlighting — JS RegExp engine avoids Oniguruma WASM, zero CSP impact; `shiki/core` fine-grained bundle with dynamic `import()` (conditional on OQ-3)
- `testing.F` (Go stdlib): fuzz testing — seed corpus in `testdata/fuzz/FuzzSandboxPath/`, merge gate for Phase 1

**What not to use:**
- `filepath-securejoin`: legacy API is TOCTOU-unsafe; modern API is Linux-only; MPL-2.0 copyleft concern
- `go-billy/v5 ChrootOS`: CVE-2023-49569 path traversal; not a syscall-level sandbox
- `shiki` with default Oniguruma engine: requires `wasm-unsafe-eval` CSP amendment
- `rehype-raw` with `react-markdown`: allows raw HTML passthrough from working-dir files (XSS via `style-src unsafe-inline`)
- CodeMirror 6 / Monaco: v3.5 decision — editor lib is only relevant for the write side

### Expected Features

The feature scope is bounded by "embedded read-only browser inside a session-management app" — higher bar than a tree sidebar, lower bar than macOS Finder.

**Must-have (table stakes):**
- Directory listing: name/size/mtime columns, directories-first, sort asc/desc
- Keyboard navigation: Up/Down/Enter/Backspace/Home/End/PageUp/PageDown
- Breadcrumb path bar: clickable, sandboxed to session cwd, remote indicator
- Type-ahead filter: `/` key activation (NOT Cmd-F — collides with xterm find bar), current-dir scope, ESC to clear
- Text file preview: raw monospace, 5 MB cap, size in header
- Markdown rendered preview: `react-markdown` + `remark-gfm`, same 5 MB cap
- Image preview: PNG/JPEG/WebP/GIF/SVG via `<img>` with `object-fit: contain`
- Binary refusal + Download button; "Too large" message with size + Download button
- Empty directory, loading, permission-denied, network-error states
- Broken symlink annotation in listing
- `files.read` capability gate: tab-level message for viewers lacking the bit
- TUI Files view: `list.Model` + `viewport.Model`, TokyoNight palette, text/markdown preview, image/binary referral, type-ahead, breadcrumb status line
- Cross-surface parity (GUI + Web + TUI), ARIA accessibility

**Should-have (differentiators):**
- Remote session `via tailnet` / hostname badge — low cost, high value for Tailscale users
- "Refreshed N seconds ago" staleness indicator + manual Refresh button (no inotify in v3.4, but staleness must be visible)
- `X-Directory-Truncated: true` header + frontend signal for 10,000+ entry directories
- Preview size indicator in header (KB/MB)
- Broken symlink `⚠ broken symlink` dimmed row annotation

**Defer to v3.5:**
- Write operations (upload/delete/rename/mkdir)
- Syntax highlighting (research complete; policy deferred — see OQ-3)
- Recursive filter / file search
- Auto-refresh via `fsnotify` SSE
- TUI `$EDITOR` shell-out
- Sort persistence per session

### Architecture Approach

All file operations route through a new `internal/files/` package (zero coupling — imports nothing from daemon, relay, or webserver). The daemon socket mux and webserver mux each mount the same `FileHandler` with different auth. `SessionEngine` gains a `sessionWorkDirs map[string]string` field (parallel to existing `tabNames`/`sessionCLIs` maps) — this is the gap that must be patched in Phase 1. Remote session file access goes over Tailscale HTTPS directly to the remote daemon's webserver, NOT through a new relay frame type.

**Major components:**
1. `internal/files/` — `Sandbox.Resolve()` using `os.OpenRoot` + `Handler.{List,Stat,Read}` — pure package, fuzz-tested before any consumer merges
2. `internal/daemon/engine.go` + `api.go` — `sessionWorkDirs` map, daemon-socket file routes, `DaemonClient.{ListFiles,StatFile,ReadFile}` methods
3. `internal/webserver/server.go` + `capability_mw.go` — `requireFilesRead` wrapper (separate from `requireCapability`), webserver file routes, `files.read` in `Claims.Perms`, `HasPerm` helper in `internal/capability`
4. `frontend/src/components/FileBrowserTab.tsx` — single-pane list + side-by-side preview, breadcrumb, sort, filter, ARIA semantics, per-session tab (NOT singleton)
5. `internal/tui/files.go` + `files_fetch.go` — `FilesModel` with all FS I/O via `tea.Cmd` (never sync in `Update`)

### Critical Pitfalls

1. **TOCTOU symlink race** — Use `os.OpenRoot` / `os.OpenInRoot` exclusively. Pre-resolve cwd via `EvalSymlinks` once at session creation; never call `EvalSymlinks` on user-supplied path components. Two-step EvalSymlinks+Open has a race window exploitable by any process with write access to cwd subdirs (CVE-2026-27976 class, 8.8 CVSS).

2. **Windows reserved device names + ADS** — `os.OpenRoot` does not prevent `CON`, `NUL`, `COM1`-`COM9`, `LPT1`-`LPT9` or NTFS Alternate Data Streams on all platforms. Add explicit cross-platform rejection (not Windows-build-only): null bytes → absolute paths → device names (case-insensitive, with/without extension) → ADS colon syntax → UNC paths — before `filepath.Clean`. CVE-2025-27210 (Node.js) was exactly this class.

3. **`HasPerm` whole-token match** — `strings.Contains(claims.Perms, "files.read")` matches hypothetical `"no-files.read"`. Add `HasPerm(perms, perm string) bool` to `internal/capability` that splits on commas and checks for exact token match. Use in `requireFilesRead` wrapper, not inside shared `requireCapability`.

4. **0-byte file → 416** — `http.ServeContent` returns `416 Requested Range Not Satisfiable` for any Range header on a 0-byte file (golang/go#54794). Special-case: respond `200` with empty body before calling `ServeContent` if `stat.Size() == 0`.

5. **Base64 image anti-pattern** — Fetching image bytes into React state as base64 adds 33% memory overhead. Use `<img src="/api/files/{id}/read?path=...&cap=TOKEN" />` directly — browser native resource loader, no React state, no base64.

6. **TUI synchronous I/O in `Update`** — All filesystem I/O in the TUI must be dispatched via `tea.Cmd` functions returning result messages. Calling `os.ReadDir` directly in `Update` freezes the entire render loop.

---

## Convergence: The 6 Highest-Impact Decisions

All four research documents arrived at the same conclusions. These are locked:

1. **`os.Root` as the sandbox primitive** — Not `filepath-securejoin`, not `go-billy`, not EvalSymlinks+prefix-check. `os.OpenRoot` is the only TOCTOU-safe choice in Go 1.24+. Project uses Go 1.26.1. Locked.

2. **REST over HTTPS (Tailscale) for both local and remote file access** — NOT a new relay frame type. The relay is PTY fan-out with no request-response semantics. For remote sessions, `RemoteSession.url` + capability token provides the HTTPS transport. Locked.

3. **Single-pane list + preview layout (GitHub web viewer model)** — NOT VS Code tree-in-sidebar (creates dual-sidebar conflict with AgentHub's existing left nav), NOT miller columns (too wide, poor fit for 2-4 level project trees). Locked.

4. **Custom `list.Model` + `viewport.Model` for TUI (NOT `filepicker`)** — `filepicker` is a selection-dialog pattern with no browse-pane mode and no preview integration. Custom delegate costs ~200 LoC but gives full TokyoNight palette control and composable split-pane layout. Locked.

5. **No new CSP carve-out** — `shiki` with `createJavaScriptRegexEngine()` eliminates WASM `wasm-unsafe-eval`. `react-markdown` without `rehype-raw` is safe under `script-src 'self'`. Images served via direct endpoint URL avoid any `img-src blob:` amendment. Locked.

6. **Shiki with JS engine + 10 curated languages (if syntax highlighting ships in v3.4)** — `shiki/core` + `createJavaScriptRegexEngine()` + dynamic `import()` per language + `tokyo-night` theme. ~200-300 KB fine-grained bundle before Vite tree-shaking. Locked conditional on OQ-3.

---

## Tensions and Contradictions Between Research Docs

**T-1: WorkDir gap + fuzz corpus must both land in Phase 1**
ARCHITECTURE identifies `sessionWorkDirs` as a required Phase 1 addition. PITFALLS mandates the fuzz corpus ships in the same PR as the first endpoint. Both belong in Phase 1 — the planner must encode both as explicit tasks, not separated.

**T-2: `filepicker` disagreement between STACK.md and FEATURES.md**
STACK.md recommends `bubbles/v2/filepicker` with a wrapper. FEATURES.md argues against it in favor of custom `list.Model` + `viewport.Model`. FEATURES.md is correct: `filepicker` is dialog-oriented and provides no preview pane integration. Use `list.Model` + `viewport.Model` as specified in FEATURES.md and confirmed by ARCHITECTURE.md Decision 8.

**T-3: `/` filter key vs. Cmd-F**
FEATURES.md recommends `/` as the primary filter activation key to avoid collision with the xterm.js find bar (SRC-01 pattern from v3.2 Phase 94). ARCHITECTURE.md confirms the collision is real. Encode `/` as the filter activation key. Cmd-F can be added as a secondary shortcut only if `fileBrowserTabActive && !xterm.hasFocus` is enforced.

**T-4: Remote session scope for v3.4**
ARCHITECTURE.md provides the full call-path trace for remote file access (works — same HTTPS + cap token already used for tailnet peer discovery). PITFALLS.md Pitfall 13 suggests "local sessions only for v3.4." The milestone spec says "works against local AND remote sessions." Treat remote file browse as in-scope per the milestone spec, but flag as a testing checkpoint: both ends must be on v3.4+ for `files.read` to be present in the remote cap token.

---

## Implications for Roadmap

### Phase 1: Sandbox Core + Daemon Routes + WorkDir Gap

**Rationale:** `internal/files/` is dependency-free and must be proven correct before any surface depends on it. The `sessionWorkDirs` gap in `SessionEngine` blocks all file endpoints. The fuzz corpus is a merge gate per the milestone spec.

**Delivers:**
- `internal/files/sandbox.go` — `Sandbox.Resolve()` using `os.OpenRoot`; defense-in-depth pre-checks (null bytes → absolute paths → Windows device names → ADS colons → UNC paths)
- `internal/files/handler.go` — `List` (10,000-entry cap + `X-Directory-Truncated` header + Windows path separator normalization + macOS `._` filter), `Stat`, `Read` (`http.ServeContent` + 0-byte special-case + server-side 5 MB cap → 413)
- `internal/files/sandbox_test.go` + `handler_test.go` — fuzz tests, Range edge cases, MIME cascade tests
- `internal/daemon/engine.go` — `sessionWorkDirs map[string]string` + `GetSessionWorkDir(id string) string`; EvalSymlinks-resolved WorkDir cached at `CreateSession`
- `internal/daemon/api.go` — `GET /sessions/{id}/files/list`, `/stat`, `/read` (method-prefixed, loopback = no auth)
- `internal/daemon/types.go` — `FileEntry`, `FileListResponse`, `FileStatResponse` (forward-slash normalized)
- `internal/daemon/client.go` — `ListFiles`, `StatFile`, `ReadFile` methods
- `internal/capability/capability.go` — `const PermFilesRead = "files.read"` + `HasPerm(perms, perm string) bool` helper
- `settings.json` migration: `filesRead bool` in `daemonSettings`, `defaultSettings()` explicit `true`, `schemaVersion: 3`, fixture test `TestSettingsMigration_FilesReadDefaultsTrue`

**Avoids:** TOCTOU race (Pitfall 1), Windows device names (Pitfall 2), `HasPerm` prerequisite (Pitfall 4), 0-byte 416 (Pitfall 5), directory memory blowup (Pitfall 6), schema migration default (Pitfall 16)

**Research flag:** Standard patterns — no additional pre-phase research needed.

---

### Phase 2: WebServer Routes + `files.read` Capability Bit

**Rationale:** Depends on Phase 1's `internal/files` and `HasPerm`. Must be verified with integration tests before the React tab lands.

**Delivers:**
- `internal/webserver/capability_mw.go` — `requireFilesRead` wrapper (chains `requireCapability` + `HasPerm(claims.Perms, PermFilesRead)`); returns `403 "files.read permission required"` — never generic "forbidden"; separate from shared `requireCapability`
- `internal/webserver/server.go` — `filesHandlerProvider` injection; `GET /api/files/list`, `/stat`, `/read` routes; explicit `POST /api/files/*` → 405; HEAD behavior resolved (see OQ-1)
- `internal/daemon/api.go` — `issueCapabilitiesForSession`: owner token gets `read,write,files.read`; viewer token gets `read` only
- Integration tests: viewer cap + file endpoint = 403 with message; owner cap = 200; no-cap = 401 (not 404); POST = 405

**Avoids:** `HasPerm` false-positive (Pitfall 4), middleware contamination (Pitfall 4), 401 vs 404 route existence leak (Pitfall 8)

**Research flag:** Standard patterns — existing `requireCapability` is a direct template.

---

### Phase 3: FileBrowserTab.tsx (Desktop + Web)

**Rationale:** Depends on Phase 1 (daemon API shape frozen) and Phase 2 (capability token plumbing). Phase 3 and Phase 4 can run in parallel.

**Delivers:**
- `frontend/src/components/FileBrowserTab.tsx` — single-pane list + side-by-side preview; breadcrumb; sort; `/` type-ahead filter; text/markdown/image/binary preview; Download button; ARIA `listbox` + `region` + breadcrumb `<nav>` + `aria-live` preview
- `react-markdown@10.1.0` + `remark-gfm@^4` — no `rehype-raw`; `.html` files force `text/plain` + source label
- `shiki@4.1.0` (JS engine, fine-grained, dynamic import) — conditional on OQ-3 resolution
- Image preview via `<img src="...endpoint URL...">` — NOT base64 through React state
- `X-Refreshed-At` header → "Refreshed N seconds ago" + manual Refresh button (R key)
- `frontend/src/App.tsx` — `'file-browser'` tab type; `handleOpenFileBrowser`; per-session find-or-add (NOT singleton)
- `DaemonManagerPanel.tsx` — "Browse Files" button; `TabBar.tsx` — context menu item
- Playwright e2e: local + remote browse, viewer 403 with message, markdown render, image preview, binary refusal, empty dir, network error

**Avoids:** Base64 anti-pattern (Pitfall 10), `rehype-raw` CSP bypass (Pitfall 9), singleton tab bug (Pitfall 13), HTML file rendering (Pitfall 9), no staleness indicator (Pitfall 11)

**Research flag:** OQ-3 (syntax highlight scope) must be resolved before this phase spec is written. Otherwise standard patterns.

---

### Phase 4: TUI Files View

**Rationale:** Depends only on Phase 1 (`DaemonClient.{ListFiles,ReadFile}` methods). Parallelizable against Phase 3 after Phase 1 freezes. No new API work needed.

**Delivers:**
- `internal/tui/files.go` — `FilesModel`: `list.Model` with `FileItem` delegate (icon/name/size/mtime, TokyoNight palette), `viewport.Model` preview pane, breadcrumb, focus toggle, built-in `/` filter
- `internal/tui/files_fetch.go` — `fetchDirCmd`, `fetchPreviewCmd` as `tea.Cmd` functions (all FS I/O async)
- `internal/tui/model.go` — `tabFiles tabID` constant, `filesModel *FilesModel` (lazily initialized)
- `internal/tui/update.go` — key dispatch priority: `editing_filter > kill confirm > new session modal > QR overlay > files > help > main view`; `f` key on Sessions tab opens Files for selected session
- `internal/tui/view.go` — `filesModel.View()` for `tabFiles`
- Layout: 40/60 split >120 cols; 50/50 at 80-119; full-width <80 cols
- `glamour.WithAutoStyle()` for `.md` files; binary → "Use the desktop or web app to preview"
- Help overlay (`?`) updated with Files view keybindings
- Breadcrumb truncated from left: `…/utils/helper.ts`

**Avoids:** TUI sync I/O freeze (Pitfall 7), key dispatch conflict (Pitfall 12), navigation above cwd (milestone spec)

**Research flag:** Standard patterns — existing TUI `tea.Cmd` architecture is a direct template.

---

### Phase Ordering Rationale

- Phase 1 is a hard blocker for all other phases. The `sessionWorkDirs` gap makes all file endpoints non-functional until patched. The fuzz corpus is a merge gate.
- Phase 2 can immediately follow or be combined with Phase 1.
- Phases 3 and 4 can run in parallel after Phase 1 freezes the `DaemonClient` API shape. Phase 3 additionally needs Phase 2 capability plumbing.
- Phase 3 Playwright e2e is the v3.4 release gate.

### Research Flags

All phases use well-documented standard patterns. No pre-phase research passes are needed. The full implementation detail is in STACK.md, ARCHITECTURE.md, and PITFALLS.md.

---

## Open Questions for the Planner

**OQ-1: HEAD /api/files/read — 405 or supported?**
`http.ServeContent` handles HEAD correctly (headers without body), enabling Content-Length preflights. Decision: register `HEAD /api/files/{id}/read` explicitly (recommended — enables frontend size preflight before loading large files), or explicitly return 405 and document it. Encode as a concrete spec item in Phase 2.

**OQ-2: TUI breadcrumb truncation direction**
Left truncation (`…/utils/helper.ts`) is the correct UX choice — shows the most specific path component. Confirm in Phase 4 spec.

**OQ-3: Syntax highlighting scope — 10 languages or plain text in v3.4?**
FEATURES.md defers syntax highlighting to v3.5 (plain monospace text sufficient for read-only preview). STACK.md fully specifies `shiki@4.1.0` with JS engine — zero CSP impact, ~200-300 KB fine-grained bundle, research complete. Options:
- **Ship plain text** (FEATURES.md): simpler Phase 3, no `shiki` dep, consistent with "preview establishes API shape" framing. Syntax highlighting deferred to v3.5 with editor library decision.
- **Ship shiki with 10 languages** (STACK.md readiness): better source code preview UX, all implementation detail is pre-specced.
- Synthesis recommendation: ship plain text in v3.4. The `shiki` research is ready for v3.5 if the planner decides otherwise.

---

## "Watch Out For" — Merge-Gate Items for the Planner

| Item | Phase | Merge Gate Condition |
|------|-------|----------------------|
| Fuzz corpus 30s clean | 1 | `go test -fuzz=FuzzSandboxPath -fuzztime=30s` passes with no panics or escapes |
| Windows device name rejection | 1 | `?path=CON`, `?path=NUL.txt`, `?path=COM1.txt` all return 400 on all platforms |
| ADS colon rejection | 1 | `?path=file.txt:hidden` returns 400 on all platforms |
| 0-byte file returns 200 not 416 | 1 | `GET /read` on 0-byte file returns `200` with empty body |
| Server-side 5 MB cap | 1 | 5,000,001-byte file returns `413` from `/read` endpoint |
| `HasPerm` whole-token match | 1 | `HasPerm("read,write", "files.read")` = false; `HasPerm("read,files.read", "files.read")` = true |
| No-cap = 401 not 404 | 2 | `GET /api/files/{id}/list` without `?cap=` returns 401 |
| Viewer cap without `files.read` = 403 | 2 | Response body contains "files.read" |
| `POST /api/files/list` = 405 | 2 | Method not allowed for non-GET methods |
| `rehype-raw` excluded | 3 | Source-inspection test: `rehype-raw` NOT imported in `FileBrowserTab.tsx` |
| Image via `<img src>` not base64 | 3 | Source-inspection test: no `btoa` or base64 string in image preview code |
| TUI I/O via `tea.Cmd` only | 4 | Source-inspection test: no direct `os.ReadDir` or `os.Open` inside `Update` method |
| TUI never above cwd | 4 | Backspace at root does NOT navigate above session cwd |
| Kill confirm takes priority over Files | 4 | Kill-confirm modal still works with Files view underneath it |
| Help overlay updated | 4 | `?` shows Files view keybindings in `tabFiles` mode |

---

## Minimal New-Deps Roster

**Go (go.mod only — no new binaries downloaded):**
- `github.com/wailsapp/mimetype@v1.4.1` — promote indirect → direct
- `github.com/charmbracelet/glamour@v0.8.0` — promote indirect → direct

**Frontend (pnpm add — Vite-bundled, NOT served to web terminal page, no `vendor_drift_test.go` entries):**
- `react-markdown@10.1.0`
- `remark-gfm@^4`
- `shiki@4.1.0` (conditional on OQ-3)

No `web/vendor/` changes. No `vendor_drift_test.go` changes. No `embed.go` changes. No CSP changes.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All Go choices verified against go.dev docs and issue tracker; npm versions verified against npm/GitHub; dep status verified via `go list -m all` and `frontend/package.json` |
| Features | HIGH | Primary sources: MDN/ARIA spec, charmbracelet official docs, milestone spec; competitor analysis MEDIUM |
| Architecture | HIGH | All claims verified against actual AgentHub source files (`engine.go`, `capability.go`, `server.go`, `App.tsx`, `tui/model.go`) |
| Pitfalls | HIGH | Go stdlib behavior verified against go.dev docs and issue tracker; AgentHub-specific integration verified from source; CVE class confidence HIGH, individual CVE NVD cross-reference MEDIUM |

**Overall confidence:** HIGH

### Gaps to Address During Planning

- **Remote cap token availability:** Verify the remote session's cap token includes `files.read` after Phase 2 is deployed. Both ends must be v3.4+. Encode as a concrete test checkpoint in Phase 3 spec for remote file browse.
- **Windows path prefix check case-sensitivity:** The defense-in-depth pre-check layer may need case-folded comparison on Windows NTFS (case-insensitive FS). `os.OpenRoot` handles this at the kernel level, but worth a Windows-specific test in Phase 1.
- **macOS `._` resource fork filtering platform flag:** Filter `._` prefixed files on `runtime.GOOS == "darwin"` only (not a build tag). Enumerate as an explicit task in Phase 1 spec.
- **`schemaVersion: 3` migration fixture:** `settings_v3.3.json` fixture file must be created as a concrete deliverable in Phase 1; it is not implied.

---

## Sources

### Primary (HIGH confidence)

- [Go 1.24 os.Root blog post](https://go.dev/blog/osroot) — traversal-resistant semantics, TOCTOU elimination, Windows device name blocking
- [golang/go#71165](https://github.com/golang/go/issues/71165) — `filepath.EvalSymlinks` Windows link-type bug (open)
- [golang/go#54794](https://github.com/golang/go/issues/54794) — `http.ServeContent` 416 on 0-byte file
- [golang/go#50905](https://github.com/golang/go/issues/50905) — `ServeContent` wrong headers on invalid range
- [charm.land/bubbles/v2@v2.1.0](https://pkg.go.dev/charm.land/bubbles/v2@v2.1.0/filepicker) — `filepicker` component confirmed
- [charmbracelet/glamour v0.8.0](https://pkg.go.dev/github.com/charmbracelet/glamour@v0.8.0) — ANSI markdown renderer
- [react-markdown GitHub](https://github.com/remarkjs/react-markdown) — v10.1.0 stable, virtual DOM, no `dangerouslySetInnerHTML`
- [Shiki JavaScript RegExp engine](https://shiki.style/guide/regex-engines) — native RegExp, no WASM
- [Shiki bundle sizes](https://shiki.style/guide/bundles) — full 6.4 MB; core + per-lang imports
- [W3C WAI-ARIA Tree View Pattern](https://www.w3.org/WAI/ARIA/apg/patterns/treeview/) — ARIA semantics reference
- AgentHub source: `internal/capability/capability.go`, `internal/webserver/capability_mw.go`, `internal/webserver/server.go`, `internal/daemon/engine.go`, `internal/daemon/api.go`, `internal/relay/protocol.go`, `frontend/src/App.tsx`, `internal/tui/model.go`, `internal/tui/update.go`

### Secondary (MEDIUM confidence)

- [Shiki CSP/eval issue — vercel/streamdown#384](https://github.com/vercel/streamdown/issues/384) — WASM engine requires `wasm-unsafe-eval`; JS engine fixes it
- [CVE-2026-27976: Zed sandbox escape via symlink TOCTOU](https://www.thehackerwire.com/zed-code-editor-sandbox-escape-via-symlink-traversal-cve-2026-27976/) — 8.8 CVSS, same class as Pitfall 1
- [CVE-2025-27210: Node.js Windows device name path traversal](https://zeropath.com/blog/cve-2025-27210-nodejs-path-traversal-windows) — same class as Pitfall 2
- [go-billy CVE-2023-49569](https://dailycve.com/go-git-go-billy-path-traversal-symlink-following-cve-2023-49569-critical/) — ChrootOS path traversal risk
- [Microsoft: Naming Files, Paths, and Namespaces](https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file) — Windows reserved device names, ADS
- [VS Code User Interface docs](https://code.visualstudio.com/docs/getstarted/userinterface) — Explorer layout reference
- [About large files on GitHub](https://docs.github.com/en/repositories/working-with-files/managing-large-files/about-large-files-on-github) — file size limit UX reference

---
*Research completed: 2026-05-20*
*Ready for roadmap: yes*
