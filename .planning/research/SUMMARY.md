# Project Research Summary

**Project:** AgentHub — v3.5 File Browser: Write Operations & Editor
**Domain:** Write-side filesystem API + in-app code editor for a sandboxed AI-session management desktop app
**Researched:** 2026-06-14
**Confidence:** HIGH

## Executive Summary

v3.5 adds the write half of the file-browser epic to an existing, fuzz-proven, `os.OpenRoot`-sandboxed read-only filesystem API. The surface is well-understood: five write endpoints (write, upload, delete, rename, mkdir), a new `files.write` capability bit that mirrors the v3.4 `files.read` pattern exactly, and an in-app code editor. The editor library question is research-ratified: **use CodeMirror 6, not Monaco.** Monaco requires `worker-src blob:` in the Content Security Policy — a new directive that violates project security discipline and has no workaround that preserves the single-binary architecture. CodeMirror 6 bundles cleanly via Vite (~135 KB gzipped), uses no web workers, and fits the existing `style-src 'unsafe-inline'` CSP policy without amendment. All four research areas (STACK, FEATURES, ARCHITECTURE, PITFALLS) converge on the same recommendation.

The biggest technical risk cluster is in write-side sandbox correctness, not in the editor integration. Three issues require careful implementation: (1) `os.Root.Rename` does **not** exist in Go 1.26 — `Sandbox.Rename` must validate both source AND destination paths through `validateAndClean` before calling `os.Rename` with constructed absolute paths; (2) atomic write (temp file + `f.Sync()` + rename) is mandatory — `O_TRUNC` in-place writes corrupt files visible to concurrent AI agent readers; (3) CSRF Origin checks are required on all state-changing write endpoints because v3.4's GET-only read surface was CSRF-safe by convention, but POST/PUT/DELETE are not. The `files.write` capability must be explicit opt-in — never default-on — because a home-directory session (cwd=`$HOME`) combined with a default-on write cap creates a path to overwrite shell RC files and SSH keys without a sandbox escape.

Two carried tech-debt items shape the phase order. TD-5 (`ExchangeJoinCodeAtURL` 303-shim in the desktop GUI) must be fixed in Phase 123 — before remote write testing — because the desktop GUI cannot acquire a cap for any remote session today. TD-4 (WR-01..05 FileBrowserTab hardening from Phase 120) bundles cleanly into the same phase as a "while we're touching the file layer" sweep. Auto-save is an explicit anti-feature: AI agents are actively watching the filesystem, and auto-saving partial edits would inject corrupted half-finished files into a live coding session. The concurrency story is If-Match/ETag on mtime+size — not CRDT, not last-writer-wins.

## Key Findings

### Recommended Stack

v3.5 adds zero new Go modules. All write operations use `os.Root` methods (`Create`, `OpenFile`, `Remove`, `Mkdir`) already available in Go 1.26.3, plus stdlib `mime/multipart` and `net/http.MaxBytesReader` for upload. The TUI `$EDITOR` shell-out uses `tea.Exec` + `tea.ExecCommand`, the same pattern already in production in `internal/tui/attach.go` — no new API to learn.

The only new production dependencies are frontend: CodeMirror 6 packages, all Vite-bundled and CSP-clean.

**Core technologies:**
- **CodeMirror 6** (`codemirror@6.0.2` + `@codemirror/*`): in-app editor — CSP-clean (no workers, no eval), ~135 KB gzipped, Lezer-based incremental syntax highlighting, runtime read-only toggle via `Compartment.reconfigure()` without remount, lazy language loading via `@codemirror/language-data`
- **`os.Root` write methods** (Go stdlib 1.24+): `Create`, `OpenFile`, `Remove`, `Mkdir` — same sandbox boundary as v3.4 reads; `Rename` is NOT on `*os.Root` and requires a validated `Sandbox.Rename` wrapper calling `os.Rename` with sanitized absolute paths
- **`mime/multipart` + `http.MaxBytesReader`** (Go stdlib): multipart upload with size-capped body — no new Go module
- **`tea.Exec` + `tea.ExecCommand`** (Bubble Tea v2, already in go.mod): TUI `$EDITOR` shell-out, identical pattern to `attachCmd` in `internal/tui/attach.go`
- **Monaco Editor**: explicitly rejected — requires `worker-src blob:` CSP amendment (confirmed: payloadcms/payload#10229, keycloak/keycloak#32901), 2.4–6+ MB bundle, Vite 8 Oxc minifier regression (vitejs/vite#22009)

**No new CSP amendments required.** `style-src 'unsafe-inline'` (Amendment 1, Phase 89) already covers CodeMirror 6's inline style injection. Zero `worker-src`, `script-src`, or other CSP changes needed.

### Expected Features

**Must have (table stakes — P1, closes Issues #63, #64, #24):**
- `files.write` capability bit + `requireFilesWrite` middleware (mirrors `files.read`/`requireFilesRead`)
- `PUT /api/files/write` with atomic temp+rename and If-Match/ETag conflict detection (412 UX)
- `DELETE /api/files/delete` (file + directory recursive)
- `POST /api/files/rename` (within sandbox; both source and destination validated)
- `POST /api/files/mkdir`
- `POST /api/files/upload` (single file, streamed multipart, `MaxBytesReader`-capped)
- CodeMirror 6 integrated into `FileBrowserTab.tsx` — replaces plain-text `<pre>` preview for editable files
- Explicit read-only to edit toggle (pencil button) — no auto-edit-on-open
- Dirty-state indicator + unsaved-changes warning on navigate-away
- Cmd/Ctrl+S save shortcut; three-state save feedback; 412 conflict UX
- Delete confirmation modal (file + directory variants)
- Syntax highlighting via CodeMirror language packs (common languages)
- Binary file exclusion from edit mode (`IsBinary: true` — no Edit button)
- TUI `$EDITOR` shell-out on `e` key; `resolveEditor()` fallback chain
- `files.write` opt-in for web-share (dedicated security phase, `FuzzSandboxWrite` corpus)
- Remote tailnet peer write parity (GUI daemon proxy + TUI `RemoteFilesClient`)
- TD-4 (WR-01..05) + TD-5 (`ExchangeJoinCode` 303-shim) folded into Phase 123

**Should have (competitive differentiators — P2):**
- Multi-file upload — trigger: single-file upload bakes under real use
- Move across directories — trigger: rename lands cleanly; confirm demand

**Defer (v3.6 or later — P3/OUT):**
- Filesystem-change auto-refresh (inotify/FSEvents) — high complexity; refresh-on-navigate covers MVP
- Concurrent-edit viewer count indicator — If-Match covers safety; count is a DX nice-to-have
- Auto-save — **explicit anti-feature**: AI agents watch the FS; partial edits would corrupt live coding sessions
- CRDT/real-time multi-cursor collab — wrong architecture for AgentHub's single-operator + AI model
- Git diff integration — separate epic; terminal tab covers git CLI use cases
- Binary hex editor — out of scope

### Architecture Approach

The write side extends the existing `internal/files/` package with a new `write.go` file containing `Handler.Write`, `Handler.Delete`, `Handler.Rename`, and `Handler.Mkdir` HTTP handlers. The existing `Sandbox` struct gains new methods (`WriteFileAtomic`, `Rename`, `MkdirAll`, `Delete`) that wrap `*os.Root` operations — or in the case of `Rename`, use `os.Rename` with double-validated absolute paths. The existing `FilesClient` interface in `internal/tui/files_client.go` grows from 4 read methods to 8, with `*daemon.DaemonClient` and `*tui.RemoteFilesClient` both satisfying the extended interface. The React `FileBrowserTab.tsx` editor toggle follows a `mode: 'preview' | 'edit'` prop pattern on `PreviewPane`, with CodeMirror 6 mounted in a `useEffect`. Write parity is proven via the same 3-observer pattern established in Phase 122: daemon-proxy Go, `RemoteFilesClient` Go, and Playwright HTTPS browser.

**Major components:**
1. `internal/files/write.go` — HTTP handlers for Write/Upload/Delete/Rename/Mkdir; atomic write-temp-rename; 50 MiB body cap; CSRF Origin check in `requireFilesWrite`
2. `internal/capability/capability.go` + `internal/webserver/capability_mw.go` — `PermFilesWrite` constant + `requireFilesWrite` middleware (parallel to `requireFilesRead`; uses `HasPerm`, not `strings.Contains`; includes Origin check)
3. `frontend/src/components/Editor.tsx` — CodeMirror 6 wrapper; `Compartment`-based read-only toggle; Cmd/Ctrl+S; dirty-state tracking; If-Match ETag header on save
4. `internal/tui/files.go` + `internal/tui/files_cmds.go` — `tea.Exec` $EDITOR shell-out; `writeBackCmd` tea command; `resolveEditor()` fallback chain
5. `internal/daemon/remote_files.go` — extended `proxyRemoteFiles` to forward `r.Body` for PUT/POST write ops
6. `internal/files/sandbox.go` — `Sandbox.Rename` (double-validated), `Sandbox.MkdirAll` (iterative `root.Mkdir`), shell-RC denylist on all write paths

### Critical Pitfalls

1. **`os.Root.Rename` does not exist** — `*os.Root` in Go 1.24-1.26 exposes no `Rename` method (golang/go#69462, still open). `Sandbox.Rename` must validate both `oldRelPath` AND `newRelPath` through `validateAndClean`, then construct absolute paths and call `os.Rename`. Failing to validate the destination enables path traversal to `~/.ssh/authorized_keys` or `~/.bashrc` with write consequences. Fuzz corpus must include rename-destination traversal payloads. (PITFALLS.md §1, §2)

2. **Atomic write is mandatory** — `O_TRUNC` in-place writes create a window where concurrent AI agent readers see empty or partial files. `Sandbox.WriteFileAtomic` must write to a sibling temp file (`relPath + ".agenthub-tmp-" + randomHex()`), call `f.Sync()` (fdatasync), then call `Sandbox.Rename(tmp, relPath)`. The temp file must be inside the sandbox root (not `os.TempFile` to a system temp dir — that escapes the sandbox AND loses atomicity on cross-filesystem rename). (PITFALLS.md §5)

3. **CSRF Origin checks on all write endpoints** — v3.4's read surface was GET-only and CSRF-safe by convention. POST/PUT/DELETE are not. `requireFilesWrite` must check the `Origin` header and reject requests where Origin is present but does not match the server's FQDN, mirroring the Phase 88 WebSocket Origin check. Desktop GUI Wails fetch sends no Origin header — the check passes vacuously for local requests. (PITFALLS.md §4)

4. **Shell RC file denylist** — when a session's `cwd` is `$HOME` (Claude Code's default), the sandbox root IS the home directory. A `files.write` token for this session can correctly-but-dangerously overwrite `~/.bashrc`, `~/.zshrc`, `~/.ssh/authorized_keys`, `~/.claude/CLAUDE.md`, and the daemon's own config. A server-side absolute-path denylist in `Sandbox.Write*` methods is load-bearing. Show a warning in GUI/TUI when `files.write` is enabled for a home-directory session. (PITFALLS.md §8)

5. **`files.write` must be explicit opt-in — never default-on** — the `HasPerm` whole-token semantics must be used (not `strings.Contains`). `files.write` must NOT appear in the default session-owner token or the default web-share URL. TD-5 (`ExchangeJoinCodeAtURL` 303-shim) must be fixed before remote write testing, as the desktop GUI silently fails to acquire any remote cap today. (PITFALLS.md §7; ARCHITECTURE.md §9)

6. **Upload filename injection** — `mime/multipart`'s `FileHeader.Filename` can contain `../../.bashrc`. Always run `filepath.Base(header.Filename)` then `validateAndClean` before using the filename as a destination path. Apply `http.MaxBytesReader` before `r.ParseMultipartForm` to enforce the 50 MiB cap. (PITFALLS.md §6)

7. **TD-5 is a prerequisite for remote write** — `ExchangeJoinCodeAtURL` in `internal/daemon/client_remote_files.go` currently attempts JSON decode on the 303 redirect body (which has no JSON body), causing silent failure on the success path. Remote write testing is blocked until this is fixed. Fix: disable auto-redirect follow on the HTTP client, detect `resp.StatusCode == 303`, extract the `?cap=<token>` query parameter from the `Location` header. The TUI `exchangeJoinCodeCmd` already does this correctly — use it as the reference. (ARCHITECTURE.md §9)

## Implications for Roadmap

Phase numbering continues from v3.4's last phase (122). Suggested phases: **6** (Phase 123–128).

---

### Phase 123: TD Cleanup + Write Sandbox Primitives + Daemon Routes

**Rationale:** TD-5 (`ExchangeJoinCodeAtURL` 303-shim) must be fixed first — remote write testing is impossible without it. TD-4 (WR-01..05 hardening) bundles here as a "while touching the file layer" sweep. The write sandbox primitives (`Sandbox.Rename`, `Sandbox.WriteFileAtomic`, `Sandbox.MkdirAll`, shell-RC denylist) establish the security foundation before any write endpoint goes live. No capability model changes yet — daemon socket write routes are auth-less (loopback trust, same as reads).

**Delivers:** All write sandbox primitives in `internal/files/sandbox.go` and `internal/files/write.go`; `DaemonClient` write methods; daemon socket write routes (auth-less); fuzz corpus extended with write-path and rename-destination traversal payloads; TD-4 + TD-5 closed.

**Addresses:** Edit-and-save, delete, rename, mkdir, upload (server side); all P1 write ops at the sandbox layer.

**Avoids:** `os.Root.Rename` gap (Pitfall 1); non-atomic writes (Pitfall 5); rename destination traversal (Pitfall 2); shell RC file overwrite (Pitfall 8).

**Gate:** `go test ./internal/files/... ./internal/daemon/...` green; `FuzzSandboxWrite` reports zero crashes; TD-5 integration test passes (303 Location header parsed correctly).

**Research flag:** Standard patterns — no deeper research needed.

---

### Phase 124: `files.write` Capability + Webserver Write Routes

**Rationale:** Once sandbox primitives are frozen, the capability model and webserver route registration can land independently. The `requireFilesWrite` middleware (with Origin check and `HasPerm` semantics) must exist before any browser-facing write surface can be tested.

**Delivers:** `PermFilesWrite` constant; `requireFilesWrite` middleware (with CSRF Origin check); five webserver write routes under `requireFilesWrite`; `issueCapabilitiesForSession` updated (`files.write` opt-in, never default-on); `daemonSettings.FilesWrite` + `schemaVersion: 4` migration; share-panel opt-in toggle (off by default); remote proxy write routes + `r.Body` forwarding in `remote_files.go`.

**Addresses:** `files.write` capability bit; web-share opt-in; all P1 capability/middleware requirements.

**Avoids:** `strings.Contains` vs `HasPerm` foot-gun (Pitfall 7); CSRF on write endpoints (Pitfall 4); `files.write` default-on (Pitfall 7).

**Gate:** 403/200 capability integration tests; `TestHasPerm_NoStringsContains_Write` static grep gate; zero new CSP violations; `TestSettingsMigration_FilesWriteDefaultsFalse` fixture test.

**Research flag:** Standard patterns — capability model mirrors `files.read` exactly.

---

### Phase 125: React Editor (Desktop + Web)

**Rationale:** Depends on Phases 123 + 124 (write API frozen, capability model live). This is the milestone centrepiece — CodeMirror 6 integration replacing the v3.4 plain-text `<pre>` preview. Library selection is research-ratified (CodeMirror 6); no decision gate needed at plan time.

**Delivers:** CodeMirror 6 installed (pnpm) + `vendor_drift_test.go` gate updated; `Editor.tsx` with syntax highlighting, dirty-state tracking, Cmd/Ctrl+S, If-Match ETag header, and 412 conflict UX; `useFilesWrite` hook; `useFilesCapability` extended with `canWrite`; `FileBrowserTab.tsx` edit/delete/rename/mkdir triggers; Playwright cross-browser e2e (local write, web-share write with `files.write` cap, 403 without cap); large-file + binary-file guards.

**Addresses:** Editor integration (CodeMirror 6); edit-and-save workflow; dirty state; unsaved-changes warning; syntax highlighting; binary exclusion; large-file guard.

**Avoids:** Monaco CSP blocker (CodeMirror 6 chosen); `vendor_drift_test.go` violation; Tab/paste conflict in Wails WebView (verify in UAT checklist).

**Gate:** Cross-browser Playwright e2e; `vendor_drift_test.go` passes; zero CSP violations in browser console; Tab key inserts indentation (not browser focus change) on macOS in Wails WebView; Cmd-V paste does not double-paste.

**Research flag:** Standard patterns. If Wails WebView Tab/paste conflicts surface during UAT, flag as a targeted research spike — not a pre-phase blocker.

---

### Phase 126: TUI `$EDITOR` Shell-Out

**Rationale:** Depends only on Phase 123 (`FilesClient` interface extended, `DaemonClient` write methods available). Can run in parallel with Phase 125 if capacity allows. `$EDITOR` shell-out follows the identical pattern as `attachCmd` in `attach.go` — lower risk than the React editor phase.

**Delivers:** `FilesClient` interface extended to 8 methods; `RemoteFilesClient` write methods; `files.go` edit mode + `tea.Exec` shell-out + `writeBackCmd`; `resolveEditor()` helper (`$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi`); `tea.ClearScreen` in completion handler; `loadDirCmd` unconditionally post-exec; `TestFiles_NoSyncFSCalls` gate extended.

**Addresses:** TUI `$EDITOR` shell-out (P1 cross-surface parity); `$EDITOR` unset error; stale listing after edit; terminal state restoration.

**Avoids:** Synchronous I/O in TUI Update loop; terminal state corruption on editor exit; stale listing.

**Gate:** All TUI tests green; `TestFiles_NoSyncFSCalls` passes; `$EDITOR` shell-out integration test with override env var; `RemoteFilesClient.WriteFile` against `httptest.TLSServer`.

**Research flag:** No research needed. `tea.Exec` pattern is established in production.

---

### Phase 127: Web-Share Write Security Hardening

**Rationale:** Depends on Phases 124 + 125 (write routes live, capability model complete). This is the dedicated security audit phase for the most-exposed surface. All three research files flag web-share write as requiring a separate phase.

**Delivers:** Write security audit documentation; body size cap enforcement tests; atomic rename failure path tests; concurrent write race tests; capability escalation audit; `FuzzSandboxWrite` corpus with rename-destination traversal payloads, shell-RC denylist bypass attempts, upload filename injection; Playwright e2e for web-share write scenarios.

**Addresses:** Web-share `files.write` opt-in UX finalization; upload abuse prevention; CSRF validation; `FuzzSandboxWrite` corpus.

**Avoids:** Capability scope creep; multipart abuse; write TOCTOU; shell RC denylist bypass.

**Gate:** Security audit documented; `FuzzSandboxWrite` zero crashes; Playwright web-share write e2e; write-path symlink escape test (`403`, not `200`); `~/.bashrc` within home-dir sandbox returns `403 Protected system file`.

**Research flag:** No additional research needed.

---

### Phase 128: Remote Write Parity + Cross-Surface Integration

**Rationale:** Final phase — depends on the full stack (Phases 123–127 live locally). Mirrors the Phase 122 read parity proof pattern with a 3-observer write parity test.

**Delivers:** Remote write parity proof (daemon-proxy Go + `RemoteFilesClient` Go + Playwright HTTPS browser); TUI remote write end-to-end test; GUI remote write via daemon proxy end-to-end test; cross-surface parity verification; remote peer version gate (405 → "older version" user message); cap expiry mid-edit behavior (preserve editor buffer, show "access expired").

**Addresses:** Remote tailnet peer write parity (release-blocking cross-surface parity contract); remote write failure modes.

**Avoids:** Cap expiry buffer loss; partial upload orphan files; generic 401 error message.

**Gate:** 3-observer write parity test passes; no regression on Phase 122 remote read tests; two-machine UAT checklist ready.

**Research flag:** No additional research needed. Proxy pattern is established (Phase 122).

---

### Phase Ordering Rationale

- **123 first:** TD-5 unblocks all remote write testing. Sandbox primitives establish the security foundation before any write endpoint is callable.
- **124 before 125/126:** Capability model and webserver routes must exist before browser-facing or TUI write surfaces can be tested end-to-end.
- **125 and 126 can parallelize:** React editor (125) and TUI shell-out (126) both depend on Phase 123 write methods but not on each other.
- **127 after 125:** Security hardening phase audits the browser-facing write surface — requires Phase 125 to be complete.
- **128 last:** Remote parity is the integration milestone requiring all local write functionality to be stable.
- **No `--research-phase` inserts needed** for any of these phases. All use well-documented patterns from the existing v3.4 codebase.

### Open Questions for Plan-Time Resolution

1. **Multi-file upload P1 vs P2** — Research places it at P2. Confirm scope at Phase 123/125 boundary. If elevated to P1, it adds a batched-upload queue UI component to Phase 125.
2. **Move-across-directories scope** — ARCHITECTURE.md scopes rename to same-directory in v3.5 MVP; move-across-dir is a stretch. Server side is ready (both paths validated); scope question is UI only. Resolve at Phase 123 plan time.
3. **Upload size cap default** — 50 MiB is the research recommendation. Product may prefer `daemonSettings.UploadMaxBytes` configurable field. Resolve at Phase 123 plan time.
4. **Owner `files.write` default** — Research strongly recommends opt-in only (never default-on for any token). Confirm this matches product intent at Phase 124 plan time. If the owner token should include `files.write` by default (matching how `files.read` defaults on for the owner), that is a single-line change but has significant security implications for web-share re-sharing flows.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All versions verified against npm registry and go.mod; CSP impact verified against live `csp_mw.go`; Monaco CSP blocker confirmed against multiple tracked GitHub issues; `tea.Exec` API confirmed in production use in `internal/tui/update.go` |
| Features | HIGH | Based on authoritative v3.4 shipped API contract and PROJECT.md scope decisions; CodeMirror 6 ratified as editor; `files.write` capability design mirrors `files.read` exactly |
| Architecture | HIGH | All claims verified against actual source files; component boundaries, interface signatures, and route registration patterns taken directly from live codebase; TD-5 bug shape confirmed from Phase 122 recovery notes |
| Pitfalls | HIGH (Go stdlib/sandbox), MEDIUM (CodeMirror Wails WebView quirks) | `os.Root.Rename` gap verified against golang/go#69462; atomic write, upload abuse, and CSRF patterns are well-documented security classes; Wails-specific Tab/paste interaction not exhaustively source-confirmed |

**Overall confidence: HIGH**

### Gaps to Address

- **CodeMirror 6 Tab/Cmd-V behavior in Wails WebView**: The existing `app.go` Cmd-V clipboard handler (Phase 49) may conflict with CodeMirror's paste handler. Verify during Phase 125 UAT; conditionally disable the Wails handler when CodeMirror has focus if conflict occurs. UAT discovery item, not a pre-phase blocker.
- **Upload size cap final value**: 50 MiB is the research recommendation. Product may prefer a configurable `daemonSettings.UploadMaxBytes` setting. Resolve at Phase 123 plan time.
- **Multi-file upload scope**: P2 per research. Escalate to P1 only if confirmed at milestone kickoff.
- **Move-across-directories**: P2 per research. Server-side endpoint is already capable; scope question is entirely UI. Resolve at Phase 123 plan time.

## Sources

### Primary (HIGH confidence)
- `/Users/ken/dev/agenthub/internal/files/sandbox.go` — verified `os.OpenRoot` usage, `validateAndClean`, sandbox boundary patterns
- `/Users/ken/dev/agenthub/internal/files/handler.go` — verified `Handler` struct and read-side method signatures
- `/Users/ken/dev/agenthub/internal/capability/capability.go` — verified `PermFilesRead`, `HasPerm` whole-token semantics
- `/Users/ken/dev/agenthub/internal/webserver/capability_mw.go` — verified `requireFilesRead` separation pattern
- `/Users/ken/dev/agenthub/internal/webserver/csp_mw.go` — verified existing CSP: `style-src 'self' 'unsafe-inline'` set, no `worker-src` directive
- `/Users/ken/dev/agenthub/internal/tui/attach.go` + `update.go` — confirmed `tea.Exec` + `tea.ExecCommand` pattern in production
- `/Users/ken/dev/agenthub/internal/daemon/client_remote_files.go` — verified TD-5 bug: JSON decode on 303 redirect body
- `.planning/milestones/v3.4-phases/122-remote-session-file-browse-wiring/122-01-RECOVERY-SUMMARY.md` — TD-5 concrete shape confirmed
- [Go Blog: os.Root traversal-resistant file APIs](https://go.dev/blog/osroot) — `Create`, `Mkdir`, `OpenFile`, `Remove` availability; Go 1.24+
- [golang/go#69462: os: Root.Rename (proposed)](https://github.com/golang/go/issues/69462) — confirmed `Rename` absent from `*os.Root` in Go 1.26
- npm registry (`npm view codemirror`, `npm view monaco-editor`) — version verification
- [CodeMirror bundle size docs](https://codemirror.net/examples/bundle/) — ~135 KB gzipped confirmed
- [pkg.go.dev Bubble Tea v2 `tea.Exec`](https://pkg.go.dev/github.com/charmbracelet/bubbletea/v2) — API name confirmed

### Secondary (MEDIUM confidence)
- [payloadcms/payload#10229](https://github.com/payloadcms/payload/issues/10229) — Monaco `worker-src blob:` requirement confirmed
- [keycloak/keycloak#32901](https://github.com/keycloak/keycloak/issues/32901) — Monaco CSP incompatibility; recommends CSP-compliant replacement
- [monaco-editor#5154](https://github.com/microsoft/monaco-editor/issues/5154) — Monaco bundle size: ts.worker.js 6.68 MB, main bundle 3.88 MB
- [vitejs/vite#22009](https://github.com/vitejs/vite/issues/22009) — Vite 8 Oxc minifier regression with Monaco
- [Sourcegraph migration: Monaco to CodeMirror](https://sourcegraph.com/blog/migrating-monaco-codemirror) — 43% JS reduction; Monaco alone was 2.4 MB
- [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html) — filename injection, zip bomb, size limits
- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [golang/go#67002: os.Root design discussion (Rename/MkdirAll gaps)](https://github.com/golang/go/issues/67002)

---
*Research completed: 2026-06-14*
*Ready for roadmap: yes*
