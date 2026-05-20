---
phase: 120
plan: 04
subsystem: file-browser-tab
tags: [filebrowser, react, preview-pane, webserver, capability]
requires:
  - 120-01 (SetStaticAppFS — landed here as part of this plan)
  - 120-02 (FilesApiClient, FilesApiError, useFilesCapability, filesTypes)
  - 120-03 (BreadcrumbBar, FileListPane, FileRow, StatusLine, sortEntries)
provides:
  - FileBrowserTab orchestrator (top-level tab component)
  - PreviewPane dispatcher + 7 leaf components (Text/Markdown/Image/Unsupported/Empty/PermissionDenied/NetworkError)
  - App.tsx find-or-add singleton tab + DaemonManagerPanel + TabBar context-menu integration
  - webserver SetStaticAppFS setter + /app/ route with SPA fallback
  - daemon staticapp.go package-level FS holder (decouples wails main embed from daemon process)
affects:
  - frontend/src/App.tsx (handleOpenFileBrowser + render block)
  - frontend/src/components/DaemonManagerPanel.tsx (Browse files button)
  - frontend/src/components/TabBar.tsx (context-menu Browse files item)
  - frontend/src/style.css (preview pane visual contract)
  - internal/webserver/server.go (SetStaticAppFS + /app/ route)
  - internal/daemon/api.go (SetStaticAppFS wiring at both web-start sites)
  - main.go + assets_prod.go + assets_stub.go (staticAppForDaemon plumbing)
  - internal/release/no_autosave_test.go (skip-list fix for built dist)
tech-stack:
  added: []
  patterns:
    - "Discriminated-union preview state with kind-switch dispatcher"
    - "Per-session singleton tab pattern (__files__{sessionId} id)"
    - "Source-inspection vitest guards for security invariants (no rehype-raw, no base64)"
    - "Read-at-request-time fs.FS / handler closure pattern (mirrors Phase 119)"
    - "SPA fallback for /app/* unknown paths → serve index.html"
key-files:
  created:
    - frontend/src/components/FileBrowserTab.tsx
    - frontend/src/components/FileBrowser/PreviewPane.tsx
    - frontend/src/components/FileBrowser/TextPreview.tsx
    - frontend/src/components/FileBrowser/MarkdownPreview.tsx
    - frontend/src/components/FileBrowser/ImagePreview.tsx
    - frontend/src/components/FileBrowser/UnsupportedFile.tsx
    - frontend/src/components/FileBrowser/EmptyDirectoryState.tsx
    - frontend/src/components/FileBrowser/PermissionDeniedTakeover.tsx
    - frontend/src/components/FileBrowser/NetworkErrorState.tsx
    - frontend/src/components/FileBrowser/__tests__/PreviewPane.test.tsx
    - frontend/src/components/FileBrowser/__tests__/FileBrowserTab.no-rehype-raw.test.tsx
    - frontend/src/components/FileBrowser/__tests__/FileBrowserTab.no-base64.test.tsx
    - frontend/src/components/FileBrowser/__tests__/FileBrowserTab.singleton.test.tsx
    - internal/daemon/staticapp.go
    - internal/webserver/app_bundle_test.go
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/DaemonManagerPanel.tsx
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/__tests__/DaemonManagerPanel.test.tsx
    - frontend/src/style.css
    - internal/webserver/server.go
    - internal/daemon/api.go
    - internal/release/no_autosave_test.go
    - main.go
    - assets_prod.go
    - assets_stub.go
decisions:
  - "Plan 01's SetStaticAppFS landed in this plan (Task 4) rather than in a prior wave — Wave 1 commits never authored it. Implemented in-place with the same signature/timing as SetFilesHandler so no plan-01 prerequisite is left dangling."
  - "Daemon binary cannot import package main, so introduced internal/daemon/staticapp.go as a package-level FS holder + thread-safe getter/setter. main.go wires the FS via build-tagged staticAppForDaemon() — assets_stub returns nil (dev /app/ → 503), assets_prod returns the same embed.FS that Wails uses."
  - "Webserver /app/ does SPA fallback: any path under /app/ that isn't a real file in the embed serves index.html, so React Router (or any future client-side routing) just works."
  - "App.tsx render gates the file-browser tab strictly on activeId.startsWith('__files__') and the terminal-render loop skips tab.type==='file-browser' so terminal panels do not double-render under a file-browser tab."
  - "Pragmatic capToken handling: FileBrowserTab accepts capToken? but App.tsx only wires the local-loopback path (isRemote=false, no cap). Remote browse via web-share is a v3.5 follow-on — the component supports it via props, but the integration trigger is deferred."
metrics:
  duration: 32m
  completed: 2026-05-20T21:10:51Z
  tasks_total: 4
  tasks_completed: 4
  files_created: 15
  files_modified: 11
---

# Phase 120 Plan 04: FileBrowserTab Orchestrator + Preview Pane + Webserver Bundle Mount Summary

FileBrowserTab orchestrator composing breadcrumb + list + preview + status into a per-session tab, with 7 preview leaf components dispatched by PreviewPane on PreviewState.kind, App.tsx + DaemonManagerPanel + TabBar wiring for the find-or-add singleton tab pattern, and webserver-side `/app/` route mounting the React bundle so remote/web-share viewers can actually load the SPA. Closes the bulk of Phase 120 — Plan 05 covers e2e validation.

## Tasks completed

| Task | Description | Commit |
|------|-------------|--------|
| 1 | Preview leaf components + PreviewPane dispatcher + source-inspection guards | `f9c20cf` |
| 2 | FileBrowserTab orchestrator + singleton helper test | `7efe938` |
| 3 | App.tsx / DaemonManagerPanel / TabBar wiring for find-or-add tab | `b7f5e7b` |
| 4 | webserver `/app/` route + daemon SetStaticAppFS plumbing | `2c10f35` |

## Verification results

- **Frontend tests:** 1019 / 1019 pass (`pnpm test`). FileBrowser tree alone: 75+ subtests across PreviewPane (14 cases), no-rehype-raw guard (5 cases), no-base64 guard (6 cases), singleton helper (5 cases), plus prior Plan 03 BreadcrumbBar/FileListPane/sortEntries coverage.
- **TypeScript:** `pnpm exec tsc --noEmit` clean.
- **Dev Go build:** `go build ./...` clean.
- **Prod Go build:** `go build -tags wailsassets ./...` clean.
- **Go tests:** all packages green (`go test ./...`). New webserver `TestAppBundle_*` (4 cases) all pass via fstest.MapFS — exercises the 503 fallback, index serve, SPA route fallback, and real-asset serve without depending on a real frontend/dist embed.
- **Phase 119 regression:** `TestFilesRoutes_*` continue to pass.

## Source-inspection security guards

The MarkdownPreview / ImagePreview / FileBrowserTab source bytes are scanned at test time for banned substrings; CI fails before any of these can ship:

| Guard | Banned patterns | Status |
|-------|-----------------|--------|
| no-rehype-raw | `rehype-raw`, `rehypePlugins`, `dangerouslySetInnerHTML` | PASS (only doc-comment mentions remain) |
| no-base64 | `btoa(`, `data:image/`, `FileReader`, `.toDataURL`, `URL.createObjectURL` | PASS (only doc-comment mentions remain) |

## Architecture notes

### Per-session singleton tab pattern

Each session can have at most one file browser tab open. The id is computed by the exported `fileBrowserTabId(sessionId)` helper as `__files__{sessionId}`, mirroring the existing `__welcome__` / `__settings__` / `__daemon_manager__` / `__remote_sessions__` namespace. App.tsx's `handleOpenFileBrowser` checks `tabs.find(t => t.id === tabId)` before creating a new tab — opening the file browser twice for the same session focuses the existing tab rather than creating a duplicate (UI-01 success criterion).

### Preview state dispatch

`PreviewState` is a discriminated union over `kind: 'idle'|'loading'|'text'|'markdown'|'image'|'empty'|'unsupported'|'over-cap'|'broken-symlink'|'read-error'|'forbidden-file'`. `PreviewPane` switches on `kind` and renders the appropriate leaf with the payload from that variant. Each leaf is pure-presentational — no fetching, no effects, no state — so the orchestrator owns the entire data lifecycle. This makes the leaves trivially unit-testable (the PreviewPane.test.tsx suite renders each variant in isolation and asserts the right testid / copy / wired callback).

### Race protection

Both the directory-list effect and the per-file preview effect capture their respective `requestPath` / `requestSelected` at dispatch time and ignore resolutions whose captured value no longer matches the current state. AbortController cancels in-flight fetches on cleanup. Together these eliminate the Pitfall 3 race where rapid navigation between directories or files produces stale results stacked on top of fresh ones.

### Static app FS routing — Plan 01's deliverable, landed here

Plan 01 was supposed to add `SetStaticAppFS` to webserver/server.go, but the commit log shows it never landed (no SetStaticAppFS reference in any prior commit). To unblock Plan 04 Task 4, I implemented Plan 01's contract in-place with the same signature and timing semantics as the Phase 119 SetFilesHandler. The daemon-side wiring went through `internal/daemon/staticapp.go` because the daemon binary cannot import package `main` (it's the same binary invoked with `agenthub daemon`, but Go's package isolation means the embed.FS in `assets_prod.go` is unreachable from `internal/daemon/`). `main.go` calls `daemon.SetStaticAppFS(staticAppForDaemon())` at startup; the build-tagged `staticAppForDaemon()` returns nil for dev builds (so `/app/` → 503 instead of leaking the working directory) and the embed.FS for `wails build` / `go build -tags wailsassets`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-existing test failure on built dist scan**
- **Found during:** Task 4 verification (after running `pnpm build` to satisfy `go build -tags wailsassets`)
- **Issue:** `TestSER03_NoAutoSavePatterns` in `internal/release/no_autosave_test.go` walks the entire repo scanning .ts/.tsx/.js/.go files for "auto-save" vocabulary, and the skip-list's `"dist"` entry only matches `dist/...` paths at the repo root, not `frontend/dist/...`. The minified bundle's `xterm-theme` scaffolding contains `auto_` tokens that trigger the regex.
- **Fix:** Added `"frontend/dist"` to the skip-list with explanatory comment.
- **Files modified:** `internal/release/no_autosave_test.go`
- **Commit:** `2c10f35`

**2. [Rule 3 - Blocker] Plan 01 deliverable (SetStaticAppFS) missing from server.go**
- **Found during:** Task 4 initial read
- **Issue:** `git log --oneline --all | grep -i "static\|appFS"` returned no commits implementing Plan 01's SetStaticAppFS method on the webserver. Without it, Task 4 had no setter to call. The plan acknowledges it as an existing surface (see plan §interfaces "the webserver setter to plug a live React bundle FS into") but it wasn't present in the codebase.
- **Fix:** Implemented SetStaticAppFS + the `GET /app/` route + SPA fallback + 4 tests in this plan rather than blocking on a missing prior wave. Same signature and timing as Phase 119's SetFilesHandler so no contract drift.
- **Files modified:** `internal/webserver/server.go`, `internal/webserver/app_bundle_test.go`
- **Commit:** `2c10f35`

**3. [Rule 2 - Critical functionality] `assets_stub.go` would have leaked working directory under /app/**
- **Found during:** Task 4 design (when first proposing to pass `assets` directly to the daemon)
- **Issue:** `assets_stub.go`'s `var assets fs.FS = os.DirFS(".")` is fine for Wails (its dev server overrides it) but if passed to the webserver as the /app/ mount in a non-wailsassets build, it would serve the working directory tree (source code, .git, secrets, etc.) over HTTPS.
- **Fix:** Added a build-tagged `staticAppForDaemon()` accessor that returns nil in dev and the real embed.FS in prod. `main.go` calls `daemon.SetStaticAppFS(staticAppForDaemon())` so the webserver receives nil in dev → answers /app/ with 503.
- **Files modified:** `assets_stub.go`, `assets_prod.go`, `main.go`
- **Commit:** `2c10f35`

### Scope adjustments documented in code

- **Remote browse capToken:** FileBrowserTab accepts `capToken?` so the remote path is wireable, but App.tsx currently only wires the local-loopback path (`isRemote=false`, no cap). Remote browse via web-share is a v3.5 follow-on — the component is ready for it, but the integration trigger (Sessions panel → "Browse remote files") is deferred. The wiring in App.tsx uses the existing daemon-loopback baseURL (`http://127.0.0.1:{relayPort}`); the daemon-loopback /api/files/* routes are open per Phase 118 (no cap gate on the loopback path).
- **Lazy stat for size/mtime in list:** The plan calls for lazy per-row stat on selection + opt-in batch-stat on Size/Modified column header click. This plan didn't add the per-row stat backfill (entries arrive from /list with size+mtime populated by Phase 118 anyway — the field is always present in FileEntry). Sort by Size/Modified works against the listed values directly. The lazy-stat optimization is deferred to a future plan if profiling shows /list latency is the bottleneck.

## Threat surface review

No new threat surface introduced beyond what `<threat_model>` covers:

- T-120-13 (XSS via .md) — mitigated by no-rehype-raw source-inspection guard (Task 1 tests)
- T-120-14 (cap token leak via img src) — accepted (same-origin pattern, documented in ImagePreview.tsx)
- T-120-15 (path traversal via breadcrumb click) — mitigated by `isPrefixOrEqual` defense-in-depth in FileBrowserTab `navigateTo`
- T-120-16 (5 MiB cap bypass) — mitigated by relying on server 413 (Phase 118 enforced); client renders refusal copy without attempting to stream-and-discard
- T-120-17 (cap leak via Referer) — accepted, same-origin model
- T-120-SC (package install supply chain) — mitigated (no new installs in this plan; react-markdown@10.1.0 + remark-gfm^4 already installed in Plan 02 with checkpoint:human-verify)

## Self-Check: PASSED

Verified created files exist on disk:
- `frontend/src/components/FileBrowserTab.tsx` FOUND
- `frontend/src/components/FileBrowser/PreviewPane.tsx` FOUND
- `frontend/src/components/FileBrowser/TextPreview.tsx` FOUND
- `frontend/src/components/FileBrowser/MarkdownPreview.tsx` FOUND
- `frontend/src/components/FileBrowser/ImagePreview.tsx` FOUND
- `frontend/src/components/FileBrowser/UnsupportedFile.tsx` FOUND
- `frontend/src/components/FileBrowser/EmptyDirectoryState.tsx` FOUND
- `frontend/src/components/FileBrowser/PermissionDeniedTakeover.tsx` FOUND
- `frontend/src/components/FileBrowser/NetworkErrorState.tsx` FOUND
- `frontend/src/components/FileBrowser/__tests__/PreviewPane.test.tsx` FOUND
- `frontend/src/components/FileBrowser/__tests__/FileBrowserTab.no-rehype-raw.test.tsx` FOUND
- `frontend/src/components/FileBrowser/__tests__/FileBrowserTab.no-base64.test.tsx` FOUND
- `frontend/src/components/FileBrowser/__tests__/FileBrowserTab.singleton.test.tsx` FOUND
- `internal/daemon/staticapp.go` FOUND
- `internal/webserver/app_bundle_test.go` FOUND

Verified commits exist:
- `f9c20cf` FOUND (Task 1)
- `7efe938` FOUND (Task 2)
- `b7f5e7b` FOUND (Task 3)
- `2c10f35` FOUND (Task 4)
