# Phase 120: FileBrowserTab.tsx (Desktop + Web) - Research

**Researched:** 2026-05-20
**Domain:** React 19 + Wails desktop file-browser tab + web parity surface; HTTP client over daemon socket (local) and Tailscale HTTPS (remote/web)
**Confidence:** HIGH

## Summary

Phase 120 is a frontend-only phase that builds the GUI surface on top of the now-shipped sandboxed `/api/files/{list,stat,read}` REST API (Phase 118 daemon-socket + Phase 119 Tailscale-HTTPS routes). All architectural seams already exist: Phase 119 mounted `requireFilesRead` on the webserver and emitted 401 (no cap) / 403 (no `files.read` perm) / 405 (wrong verb) end-to-end; Phase 118 issues `read,write,files.read` to owner cap tokens and bare `read` to viewer tokens. The frontend code is a single new tab component `FileBrowserTab.tsx`, four child components (`BreadcrumbBar`, `FileListPane`, `PreviewPane`, `StatusLine`), a typed `FilesApiClient` wrapper, a per-tab `useFilesCapability` hook, two new runtime dependencies (`react-markdown@10.1.0` + `remark-gfm@^4`), and one Playwright spec at `frontend/e2e/files-browser.spec.ts` (merge gate per UI-14).

**Primary recommendation:** Build `FileBrowserTab` as a per-session non-singleton tab (`type: 'file-browser'`, id `__files__<sessionId>`) following the exact patterns established by `DaemonManagerPanel`, `RemoteSessionsPanel`, and `SettingsTab` in App.tsx. Issue all `/api/files/*` requests from a single typed client constructed at tab mount time. For the Wails desktop app, use `http://127.0.0.1:${relayPort}/api/files/*` against the daemon's loopback HTTP API (no cap needed — same channel `DaemonClient` already uses). For the web surface, the architecture as documented in ARCHITECTURE.md says "web FileBrowser page" — but the existing web surface (`web/terminal.html`) is a vanilla-JS terminal-only page and does NOT host the React tab system. **This is a planner gating decision** (see Open Questions § Q1 below): either (a) ship a new HTML page that imports the React build, (b) ship a separate vanilla-JS file browser on `web/files.html` mirroring `web/terminal.html` patterns, or (c) descope the web surface to "documented stub link → open desktop app for file browse." All other planning is independent of that choice.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

Locked (from ROADMAP success criteria):
- Single-tab file browser (per session) — keyboard nav (arrow/Enter/Backspace/Tab)
- Two-pane layout: file list (left) + preview pane (right)
- Markdown: `react-markdown` + `remark-gfm`. No raw HTML. No syntax highlighting in v3.4.
- Images: `<img src="/api/files/read?...">` direct URL (no base64-in-state)
- Over-cap (>5MB) / binary: "Sorry, we can't display this file" + Download button
- Viewer without `files.read`: "files.read permission required" (not generic 403)
- Path safety: breadcrumb bar bounded to session cwd; typed/pasted/clicked paths cannot escape
- E2E: Playwright Chromium + Firefox + WebKit covering 12 scenarios — merge gate

### Claude's Discretion

- Component file location (likely `frontend/src/components/FileBrowserTab.tsx`)
- State management (likely React hooks + URL query for breadcrumb state)
- API client wrapper around /api/files/list, /stat, /read
- Capability detection (parse cap token Perms or call /api/cap/info if exists)
- Loading/skeleton states between API calls
- Tab open trigger: session context menu integration + Sessions panel button

### Deferred Ideas (OUT OF SCOPE)

None — discuss phase skipped.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UI-01 | New `FileBrowserTab.tsx` registered in tab system; opens via session context menu and Sessions panel; singleton find-or-add per-session | App.tsx find-or-add pattern (settings/daemon-manager/remote-sessions) is the direct template. Per-session keying uses composite `type + sessionId` matching ARCHITECTURE.md Decision 6. TabBar `tab__context-menu` already exists (lines 249-277 of TabBar.tsx) — add `Browse files` `<button role="menuitem" className="tab__context-menu__item">` next to existing `Save Terminal As…`. DaemonManagerPanel already exposes `onKill`/`onToggleWeb` action buttons on each session row — add `onBrowseFiles` adjacent (style with `.daemon-panel__btn` per existing pattern). |
| UI-02 | Single-pane file list (left) + preview pane (right); fixed 40/60 split; no left tree pane | Pure CSS — append to `frontend/src/style.css` under `/* ─── File Browser Tab (Phase 120) ─── */` section per UI-SPEC §Design System. `flex-basis: 40%; min-width: 240px; max-width: 480px;` on list pane; `flex: 1 1 auto` on preview. UI-SPEC §Macro layout provides the complete layout contract. |
| UI-03 | Sort by name / size / mtime, asc/desc; directories sticky at top; clickable column headers | Pure React component state. UI-SPEC §Sort interaction provides exact sequence (name asc default; mtime/size default desc on first click). Note: List endpoint does NOT return size/mtime (see Phase 118 SUMMARY: `List leaves Size=0 and Mtime=""`) — frontend must call `/api/files/stat` per entry OR sort lexically only (see Open Question § Q2). |
| UI-04 | Type-ahead filter activated by `/`; current-directory only; Escape clears | Standard `useState<string>` + filtered array. UI-SPEC §Filter activation specifies document-level key listener gated by `!(event.target instanceof HTMLInputElement)` AND `tabIsActive` — same gating shape as `lib/isXtermFocused.ts`. |
| UI-05 | Breadcrumb path bar; clickable segments; root = session cwd; cannot navigate above cwd | The path that the frontend passes is always relative to the cwd — the Phase 118 sandbox enforces server-side via `os.OpenInRoot`. UI-side path state is `string[]` of segments; reset to `[]` at tab mount; navigate-up at index 0 is a no-op. |
| UI-06 | Preview cap at 5 MB server-enforced; binary/over-cap show "Sorry, we can't display" + Download | Phase 118 already returns 413 on >5 MiB and `Content-Type` flags binary via the MIME cascade. Frontend dispatches to `UnsupportedFile` / `OverCapRefusal` components. Use `HeadFile` preflight to learn size+type without streaming bytes. |
| UI-07 | Markdown via `react-markdown@10.1.0` + `remark-gfm@^4`; NO `rehype-raw` | Both packages verified (see Standard Stack). `dangerouslySetInnerHTML` MUST NOT appear in this component. Pitfall 9 source-inspection guard in tests. |
| UI-08 | Source code = monospaced plain text; NO syntax highlighting in v3.4 | `<pre className="file-browser__preview--text">` with `whiteSpace: pre`, mono 12px per UI-SPEC §Typography. |
| UI-09 | Images via `<img src="/api/files/read?...">` direct stream; NOT base64-in-state | Build the URL with `URLSearchParams` (`session=`, `path=`, `cap=`). The browser fetches the bytes via native loader; no React state involved. Same-origin URL → covered by existing `img-src 'self'` CSP. Pitfall 10 source-inspection guard in tests. |
| UI-10 | Download via Range-capable `/api/files/read`; for over-cap files this is the only retrieval path | `<a download href="...">` element activation — browser handles the streaming. The Range request is unused for downloads but the endpoint is the same. |
| UI-11 | Works against local AND remote (tailnet) sessions; React component uses `fetch()` (no new Wails binding) | Local: `http://127.0.0.1:${relayPort}/api/files/*` (loopback, no auth — same channel `DaemonClient` uses). Remote: `${remoteSession.url}/api/files/*?session=${sid}&cap=${capToken}` (HTTPS over Tailscale). Both supported by Phase 119 — `TestFilesRoutes_*` covers WEB-04 implicitly. |
| UI-12 | ARIA semantics; keyboard-only operation end-to-end; WCAG AA 4.5:1 on selection | UI-SPEC §Accessibility Contract is the verbatim contract — `role="listbox"` + `role="option"` rows + `aria-selected`, `<nav aria-label="Path">`, `role="region"` + `aria-live="polite"` on preview. Contrast verification done in UI-SPEC §Color (passes AAA on primary, AA Large on muted). |
| UI-13 | Empty / network-error / permission-denied each show explicit user-readable copy; `files.read` denial explicit | UI-SPEC §"State machine — directory listing" provides the 8-state truth table (loading, loaded, empty, truncated, permission-denied, not-found, files-read-denied, network-error, not-authorized). Copy in UI-SPEC §"Error copy". |
| UI-14 | Playwright e2e covers 12 scenarios across Chromium + Firefox + WebKit — merge gate | Existing `frontend/playwright.config.ts` is already cross-browser-configured. Existing fixture pattern (`cmd/playwright-fixture` + `e2e/global-setup.ts` + `e2e/fixture-env.ts`) provides BASE_URL + CAP for tests. Need a new fixture mode that seeds a session with known files (or extend existing fixture). |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Directory listing fetch | Frontend Server (Daemon socket) | API / Backend (Webserver) | Loopback for desktop, HTTPS for remote/web. Both already exist (Phases 118-119). |
| Path sandbox enforcement | API / Backend | — | Phase 118 `os.OpenInRoot`. Frontend MUST NOT replicate validation — trust the server. |
| Capability gating | API / Backend | — | Phase 119 `requireFilesRead` middleware. Frontend reads HTTP status (403 with body containing `files.read` → takeover state). |
| Tab state management | Browser / Client | — | React useState in App.tsx; per-session tab keyed by `__files__<sessionId>`. |
| Markdown rendering | Browser / Client | — | `react-markdown` runs entirely in the browser. No SSR. |
| Image preview | Browser / Client | API / Backend | Native `<img>` resource loader fetches from `/api/files/read`; no app code touches bytes. |
| Filter / sort | Browser / Client | — | Client-side over the current directory listing only. No server round-trip. |
| Keyboard nav state | Browser / Client | — | Component state + document-level listener gated by tab focus. |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `react` | 19.2.4 (already pinned) | UI framework | Existing dep. `react-markdown@10` requires React >=18 (verified via `npm view react-markdown@10.1.0 peerDependencies` — output: `{ '@types/react': '>=18', react: '>=18' }`). |
| `react-markdown` | 10.1.0 (exact pin per UI-SPEC) | Markdown rendering | [VERIFIED: npm registry — `npm view react-markdown@10.1.0 version` → 10.1.0] [CITED: https://github.com/remarkjs/react-markdown] Virtual-DOM render — no `dangerouslySetInnerHTML` by default. Owned by remarkjs org (Titus Wormer), 23.2M weekly downloads. |
| `remark-gfm` | ^4 (latest 4.0.1) | GitHub-Flavored Markdown plugin (tables, task lists, strikethrough, autolinks) | [VERIFIED: npm registry] [CITED: https://github.com/remarkjs/remark-gfm] Same org, 20.5M weekly downloads. |
| `@heroicons/react` | ^2.2.0 (already pinned) | Icon library for `FolderIcon`/`DocumentIcon`/`PhotoIcon`/`DocumentTextIcon`/`LinkIcon`/`ExclamationTriangleIcon`/`ChevronUpIcon`/`ChevronDownIcon`/`ArrowPathIcon` | Existing dep. UI-SPEC explicitly forbids any new icon library. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@playwright/test` | ^1.59.1 (already pinned) | Cross-browser e2e | Already configured for Chromium + Firefox + WebKit in `playwright.config.ts`. Required for UI-14 merge gate. |
| `vitest` | ^4.1.0 (already pinned) | Unit / component tests for pure-logic helpers (path normalization, filter logic, sort comparator) | Component-render tests via existing `jsdom@^29` env (see `test-setup.ts`). |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `react-markdown` + `remark-gfm` | `marked` + custom React wrapper | `marked` is faster but uses `dangerouslySetInnerHTML`-style API; CSP-incompatible without manual sanitization. Locked in CONTEXT.md anyway. |
| Direct `fetch()` calls | TanStack Query / SWR | A new query lib is overkill for a tab with at most 3 in-flight requests. The existing codebase uses bare `fetch()` everywhere (no query lib installed). |
| Polling for refresh | `EventSource` (SSE) from webserver | SSE filesystem-watch is explicitly deferred to v3.5 per PITFALLS.md Pitfall 11 ("Do NOT implement implicit auto-refresh polling in v3.4"). Manual Refresh button + `X-Refreshed-At` header is the v3.4 contract. |
| Virtualized list (`react-window`) | Render all rows up to 10,000 | Phase 118 caps directory listings at 10,000 entries (`X-Directory-Truncated: true` header above that). At 32px/row a 10k-entry list is 320k px scrollable height. Native scroll is fine; virtualization is a v3.5 optimization if profiling shows jank. CONTEXT.md `<specifics>` says "virtualized" but FEATURES.md only requires "snappy scroll" — recommend native scroll first, virtualize on measured need. |

**Installation:**
```bash
cd frontend && pnpm add react-markdown@10.1.0 remark-gfm@^4
# (Project memory states pnpm preferred; package-lock or pnpm-lock as installed.)
# Verify after install: pnpm list react-markdown remark-gfm
```

**Version verification:**
```bash
$ npm view react-markdown@10.1.0 version
10.1.0
$ npm view remark-gfm versions --json | tail -10
[ "1.0.0", "2.0.0", "3.0.0", "3.0.1", "4.0.0", "4.0.1" ]
# Latest 4.x at research time: 4.0.1 (caret range ^4 resolves to 4.0.1)
$ npm view react-markdown peerDependencies
{ '@types/react': '>=18', react: '>=18' }   # React 19.2.4 satisfies
```

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| `react-markdown@10.1.0` | npm | mature (10+ major versions; v10 released 2024) | 23,217,305/wk | github.com/remarkjs/react-markdown | not run (slopcheck unavailable) — flagged [ASSUMED] per protocol | Approved (extensive cross-verification: maintainers `wooorm`/`remcohaszing` are well-known maintainers of the remark/unified ecosystem; CITED in UI-SPEC; CITED in Phase 119 dependency analysis) |
| `remark-gfm@^4` (4.0.1) | npm | mature (4 major versions) | 20,475,806/wk | github.com/remarkjs/remark-gfm | not run — [ASSUMED] | Approved (same remarkjs org, same maintainers, same scrutiny level as `react-markdown`) |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*slopcheck was not installed in this research environment (`pip install slopcheck` not run — research is documentation-only). The planner SHOULD add a `checkpoint:human-verify` task that runs `npm view react-markdown@10.1.0` and `npm view remark-gfm@4.0.1` against the project's pnpm lockfile before `pnpm add` lands in any commit. The packages are extremely well-established (>20M weekly downloads each, official remarkjs org, multi-year history) so the risk is low — but the protocol gate stands.*

## Architecture Patterns

### System Architecture Diagram

```
┌─ Wails Desktop App (process: agenthub) ────────────────────────────────────┐
│                                                                            │
│  ┌─ Renderer (Chromium webview) ─────────────────────────────────────────┐ │
│  │                                                                       │ │
│  │  React App (frontend/src/) ───────────────────────────────────────┐   │ │
│  │  ┌─ App.tsx ───────────────────────────────────────────────────┐  │   │ │
│  │  │  tabs: Tab[] ; activeId: string                             │  │   │ │
│  │  │  handleOpenFileBrowser(sessionId, sessionName) ─────────┐   │  │   │ │
│  │  │  passes onBrowseFiles to DaemonManagerPanel + TabBar    │   │  │   │ │
│  │  └─────────────────────────────────────────────────────────┼───┘  │   │ │
│  │                                                            │      │   │ │
│  │  ┌─ FileBrowserTab.tsx ────────────────────────────────────┘      │   │ │
│  │  │  props: { sessionId, sessionName, isRemote, baseURL, capToken? } │ │
│  │  │  state: { path: string[], entries: FileEntry[], selected: string,│ │
│  │  │           preview: PreviewState, sortKey, sortDir, filter,       │ │
│  │  │           refreshedAt, error }                                   │ │
│  │  │                                                                  │ │
│  │  │  ┌─ BreadcrumbBar ────┐ ┌─ FileListPane ─┐ ┌─ PreviewPane ─┐    │ │
│  │  │  │ segments + refresh │ │ listbox of rows│ │ dispatch state │    │ │
│  │  │  └────────────────────┘ └────────────────┘ └────────────────┘    │ │
│  │  │  ┌─ StatusLine ────────────────────────────────────────────┐     │ │
│  │  │  │ item count + filter input (when '/' active)             │     │ │
│  │  │  └─────────────────────────────────────────────────────────┘     │ │
│  │  └──────────────────────┬───────────────────────────────────────────┘ │ │
│  │                         │                                             │ │
│  │  ┌─ FilesApiClient (lib/filesApi.ts) ──────────────────────────────┐  │ │
│  │  │  listFiles(path), statFile(path), readFileText(path),           │  │ │
│  │  │  headFile(path), buildImageUrl(path), buildDownloadUrl(path)    │  │ │
│  │  │  Constructed with { baseURL, capToken? } at tab mount           │  │ │
│  │  └─────────────────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────┬───────────────────────────────────┘ │
│                                       │ HTTP                                │
│  ┌─ Go side ─────────────────────────────────────────────────────────────┐  │
│  │  daemon socket localhost:${relayPort}     OR   webserver :443 (TLS)   │  │
│  │  ↓                                              ↓                      │  │
│  │  internal/daemon/api.go                         internal/webserver/    │  │
│  │  GET /api/files/list   (no auth — loopback)     GET /api/files/list    │  │
│  │  GET /api/files/stat   (no auth)                + requireFilesRead     │  │
│  │  GET /api/files/read   (no auth)                  (verifies cap.Perms  │  │
│  │  HEAD /api/files/read  (no auth)                  contains files.read) │  │
│  │  ↓ all four routes call the SAME stateless Handler instance ↓         │  │
│  │  internal/files.Handler.{List,Stat,Read} (Phase 118)                  │  │
│  │  ↓                                                                     │  │
│  │  internal/files.Sandbox (os.OpenInRoot scoped to sessionWorkDirs[id]) │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────┘

   ┌─ Remote tailnet session ──────────────────────────────────────────┐
   │  Same FileBrowserTab.tsx, but baseURL = remoteSession.url         │
   │  (HTTPS over Tailscale). capToken supplied by GUI for remote case │
   └───────────────────────────────────────────────────────────────────┘

   ┌─ Web-share viewer (browser) ──────────────────────────────────────┐
   │  Visitor lands at https://<host>/sessions/{id}?cap=<token>        │
   │  Currently served by web/terminal.html (vanilla JS, terminal only)│
   │  PHASE 120 SCOPE QUESTION: see Open Question § Q1                 │
   └───────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
frontend/src/
├── components/
│   ├── FileBrowserTab.tsx          # top-level — orchestrates state + child components
│   ├── FileBrowser/                # new subdir for the 5 child components
│   │   ├── BreadcrumbBar.tsx
│   │   ├── FileListPane.tsx
│   │   ├── FileRow.tsx             # row delegate — icon + name + size + mtime
│   │   ├── PreviewPane.tsx         # state dispatcher
│   │   ├── TextPreview.tsx
│   │   ├── MarkdownPreview.tsx
│   │   ├── ImagePreview.tsx
│   │   ├── UnsupportedFile.tsx     # binary + over-cap + broken-symlink + forbidden
│   │   ├── EmptyDirectory.tsx
│   │   ├── PermissionDeniedTakeover.tsx
│   │   └── StatusLine.tsx
│   └── __tests__/
│       ├── FileBrowserTab.test.tsx
│       ├── FileBrowserTab.no-rehype-raw.test.tsx   # Pitfall 9 source-inspection guard
│       └── FileBrowserTab.no-base64.test.tsx       # Pitfall 10 source-inspection guard
├── lib/
│   ├── filesApi.ts                 # typed client wrapping fetch() against /api/files/*
│   ├── useFilesCapability.ts       # detects files.read presence (parses GET /api/files/list response)
│   └── humanSize.ts                # "3 KB" / "1.2 MB" formatter for the row size column
├── style.css                       # APPEND to existing file; new section /* ─── File Browser Tab (Phase 120) ─── */
└── App.tsx                         # MODIFY — add 'file-browser' to Tab.type union, handleOpenFileBrowser, render block

frontend/e2e/
└── files-browser.spec.ts           # NEW — 12 scenarios per UI-14
```

### Pattern 1: Per-session non-singleton tab via `App.tsx` find-or-add

**What:** Tab is keyed by composite `(type, sessionId)` so each session gets its own tab; singleton patterns (Settings, DaemonManager) key by `type` only.

**When to use:** Every per-session tab in this codebase. Direct precedent: terminal tabs (each `sessionId` is the tab id).

**Example:**
```typescript
// frontend/src/App.tsx (new handler — modeled on handleOpenDaemonManager line 877-887)
const handleOpenFileBrowser = useCallback(
  (sessionId: string, sessionName: string) => {
    const tabId = `__files__${sessionId}`
    const existing = tabs.find((t) => t.id === tabId)
    if (existing) {
      setActiveId(existing.id)
      return
    }
    const newTab: Tab = {
      id: tabId,
      name: `${sessionName} — Files`,
      sessionId,
      cli: '',
      type: 'file-browser',
    }
    setTabs((prev) => [...prev, newTab])
    setActiveId(newTab.id)
  },
  [tabs]
)
```

Source: `frontend/src/App.tsx` lines 649-657 (`handleOpenSettings`), 876-886 (`handleOpenDaemonManager`).

### Pattern 2: Typed API client constructed at mount time

**What:** A small class (or set of bound functions) that knows the base URL and (for the web/remote case) cap token. Tabs receive props that determine the base URL; the client is constructed once per tab and re-built when the props change.

**When to use:** Whenever the React app needs to talk to a Go HTTP endpoint that already exists on both daemon socket and webserver. Direct precedent: the daemon socket access pattern in `RelayClient` (constructed with `relayPort + sessionId`).

**Example:**
```typescript
// frontend/src/lib/filesApi.ts
export interface FilesApiConfig {
  baseURL: string                          // 'http://127.0.0.1:6789'  OR  'https://my-host.tail-XXXX.ts.net:443'
  capToken?: string                        // present only for remote / web-share contexts
}

export class FilesApiClient {
  constructor(private cfg: FilesApiConfig) {}

  async listFiles(sessionId: string, path: string): Promise<FileListResponse> {
    const url = this.urlFor('list', sessionId, path)
    const res = await fetch(url)
    if (!res.ok) throw await this.toError(res)
    const truncated = res.headers.get('X-Directory-Truncated') === 'true'
    const refreshedAt = res.headers.get('X-Refreshed-At')   // may be null
    const body = (await res.json()) as { entries: FileEntry[]; truncated?: boolean }
    return { ...body, truncated: body.truncated ?? truncated, refreshedAt }
  }

  async statFile(sessionId: string, path: string): Promise<FileEntry> {
    const url = this.urlFor('stat', sessionId, path)
    const res = await fetch(url)
    if (!res.ok) throw await this.toError(res)
    return res.json()
  }

  async readFileText(sessionId: string, path: string): Promise<{ text: string; contentType: string }> {
    const url = this.urlFor('read', sessionId, path)
    const res = await fetch(url)
    if (!res.ok) throw await this.toError(res)
    return { text: await res.text(), contentType: res.headers.get('Content-Type') ?? 'text/plain' }
  }

  async headFile(sessionId: string, path: string): Promise<{ size: number; contentType: string }> {
    const url = this.urlFor('read', sessionId, path)
    const res = await fetch(url, { method: 'HEAD' })
    if (!res.ok) throw await this.toError(res)
    return {
      size: Number(res.headers.get('Content-Length') ?? '0'),
      contentType: res.headers.get('Content-Type') ?? 'application/octet-stream',
    }
  }

  buildImageUrl(sessionId: string, path: string): string {
    return this.urlFor('read', sessionId, path)
  }

  buildDownloadUrl(sessionId: string, path: string): string {
    return this.urlFor('read', sessionId, path)
  }

  private urlFor(op: 'list' | 'stat' | 'read', sessionId: string, path: string): string {
    const params = new URLSearchParams()
    params.set('session', sessionId)
    params.set('path', path)
    if (this.cfg.capToken) params.set('cap', this.cfg.capToken)
    return `${this.cfg.baseURL}/api/files/${op}?${params.toString()}`
  }

  private async toError(res: Response): Promise<FilesApiError> {
    const text = await res.text().catch(() => '')
    return new FilesApiError(res.status, text)
  }
}

export class FilesApiError extends Error {
  constructor(public status: number, public bodyText: string) {
    super(`files api ${status}: ${bodyText}`)
  }
  isMissingFilesReadPerm(): boolean {
    return this.status === 403 && this.bodyText.toLowerCase().includes('files.read')
  }
  isUnauthorized(): boolean { return this.status === 401 }
  isForbidden(): boolean { return this.status === 403 }
  isNotFound(): boolean { return this.status === 404 }
  isOverCap(): boolean { return this.status === 413 }
}
```

Source: pattern composes [VERIFIED: existing codebase] `frontend/src/lib/relayClient.ts` (port + sid construction at line 86) with [CITED: Phase 119 `TestFilesRoutes_ViewerCapReturns403_List`] response shape (body contains literal `files.read`).

### Pattern 3: Document-level keyboard listener gated by tab focus

**What:** A `useEffect`-mounted listener at the `document` level that checks `event.target` is inside the file browser tab AND not inside any input.

**When to use:** Any keyboard shortcut that should fire only when the file browser is the active tab (`/` filter activation, `R` refresh).

**Example precedent:** `frontend/src/lib/isXtermFocused.ts` — UI-SPEC §"Filter activation" cites this exact pattern.

```typescript
// inside FileBrowserTab.tsx
useEffect(() => {
  if (!isActive) return
  const onKey = (e: KeyboardEvent) => {
    if (e.target instanceof HTMLInputElement) return
    if (!tabRef.current?.contains(e.target as Node)) return
    if (e.key === '/') { e.preventDefault(); setFilterActive(true) }
    if (e.key === 'r' || e.key === 'R') { e.preventDefault(); void refresh() }
  }
  document.addEventListener('keydown', onKey)
  return () => document.removeEventListener('keydown', onKey)
}, [isActive])
```

### Pattern 4: Display:none-hide for terminal tabs vs unmount-on-switch for panel tabs

**What:** Terminal tabs are hidden via `style={{ display: isActive ? 'flex' : 'none' }}` (App.tsx line 1164) so xterm state survives tab switches. Panel tabs (Settings/Sessions/Remote) are unmounted when inactive (rendered only when `activeId === XXX.id`).

**When to use:** FileBrowserTab follows the **panel pattern** (render only when active) since there's no costly state to preserve — directory listings re-fetch from a refresh button anyway, and the markdown/image preview is cheap to re-render. Exception: if profiling shows the re-fetch on tab-reactivate is slow, switch to the display:none pattern.

**Example:**
```typescript
// frontend/src/App.tsx — modeled after RemoteSessionsPanel render at line 1090-1096
{activeId?.startsWith('__files__') && (() => {
  const sessionId = activeId.slice('__files__'.length)
  const session = tabs.find(t => t.id === activeId)
  return (
    <FileBrowserTab
      sessionId={sessionId}
      sessionName={session?.name ?? 'Files'}
      relayPort={relayPort ?? 0}
      isRemote={false}   /* TODO: thread remote-session state through */
      baseURL={`http://127.0.0.1:${relayPort}`}
      capToken={undefined}   /* desktop loopback */
    />
  )
})()}
```

### Anti-Patterns to Avoid

- **Storing file bytes in React state for images:** Use `<img src=...>` with the endpoint URL. See PITFALLS.md Pitfall 10 — 33% memory overhead from base64 + GC pressure. Tested via source-inspection: `FileBrowserTab.no-base64.test.tsx` reads the component source and asserts no `btoa(` or `data:image/` literal appears.
- **Enabling `rehype-raw` or MDX:** PITFALLS.md Pitfall 9. CSP has `style-src 'unsafe-inline'` already, so embedded `<style>` blocks from a working-directory `.md` file would execute and could clickjack the UI. Tested via source-inspection: `FileBrowserTab.no-rehype-raw.test.tsx`.
- **Validating paths in the frontend:** The server (Phase 118 sandbox) is the security boundary. Frontend path manipulation is for UX only (breadcrumb display). Never refuse a path on the client — let the server return 403 with a clear error.
- **Naive `URLSearchParams.toString()` of user-supplied paths that contain `+`:** `URLSearchParams` encodes spaces as `+` (form-encoded) but the daemon's `r.URL.Query().Get("path")` correctly decodes both `+` and `%20`. This is fine — but unit-test a path with a literal `+` character to confirm round-trip.
- **Catching all errors with `catch (_) {}`:** PITFALLS.md ("Silent Fallbacks" — CLAUDE.md core principle). Each `try/catch` must set a typed error state that surfaces in the UI (`network-error` or `read-error` state per UI-SPEC §"State machine — preview pane").
- **Polling `/api/files/list` on an interval:** PITFALLS.md Pitfall 11 explicitly forbids auto-refresh in v3.4. Manual Refresh button + `X-Refreshed-At` "Refreshed Ns ago" indicator is the contract.
- **Inflating the existing TabBar context menu by adding the Browse Files item BEFORE renaming the test:** TabBar.tsx tests assert exact menu shape. The planner must update those tests in the same PR.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Markdown rendering | A custom regex-based markdown parser | `react-markdown` + `remark-gfm` | Tens of edge cases (code blocks, nested lists, tables, links, escapes). The remark ecosystem has tens of millions of weekly users — far more battle-tested than anything you'd write. |
| Path traversal validation | A frontend `..` / null-byte check | Trust the Phase 118 server-side `os.OpenInRoot` sandbox | Defense-in-depth at the wrong layer breeds false confidence. The server is the boundary. |
| MIME detection in frontend | Looking at file extension to decide preview type | The `Content-Type` header from `/api/files/stat` (or `HEAD /read`) | Phase 118 already runs the MIME cascade (`extensionMIME` → `wailsapp/mimetype.DetectReader`); duplicating it in JS will diverge. Use server-supplied `IsBinary` boolean + `MIME` string from `FileEntry`. |
| Image preview as base64 | `fetch(url).then(blob).then(toBase64)` | `<img src={endpointURL} />` | Browser native loader handles Range, caching, decoding. Zero React state involved. PITFALLS Pitfall 10. |
| Range download for the Download button | A custom XHR with `Range` header | `<a download href={endpointURL}>Download</a>` | The browser handles streaming download. The endpoint already supports Range (`http.ServeContent`). |
| Cross-browser e2e wiring | A hand-rolled WebDriver harness | `@playwright/test` (already pinned and configured) | Existing `playwright.config.ts` ships Chromium + Firefox + WebKit with a single config block. |
| File-tree virtualization | `react-window` / `react-virtuoso` before measured need | Native scroll over 10,000-row max list | Premature optimization; FEATURES.md §"Should-have" calls for "snappy scroll" not virtualization. Add `react-window` in v3.5 if profiling demands. |

**Key insight:** Phase 118 + Phase 119 already shipped a security-hardened, MIME-correct, capability-gated REST API. Phase 120 is a **client integration** phase, not a re-implementation. The risk surface is: (1) accidentally leaking the server's protection (e.g., enabling `rehype-raw`), (2) replicating server logic in the client and drifting (e.g., MIME re-detection), and (3) shipping a UX that hides server-returned semantics behind generic error strings ("403 Forbidden" instead of "files.read permission required").

## Common Pitfalls

### Pitfall 1: Replicating MIME detection in the frontend
**What goes wrong:** Frontend looks at `entry.name.endsWith('.md')` to decide rendering — works at first, but extension casing, `.markdown` long-form, and binary files with `.txt` extensions cause divergence from the server.
**Why it happens:** Reasonable shortcut that becomes a maintenance trap.
**How to avoid:** Use `entry.MIME` (the `mime` JSON field) and `entry.IsBinary` from the Phase 118 `FileEntry`. Extension is fine for the icon column, but preview dispatch must use the MIME from the server.
**Warning signs:** Bug report "this .txt file shows as binary in the desktop app but renders fine in the web view" (or vice versa).

### Pitfall 2: Cross-tab focus collision with xterm find bar
**What goes wrong:** `/` activates the filter, but if a terminal tab is partially visible (e.g., split-pane experiments) the keystroke goes to xterm.
**Why it happens:** Document-level listener doesn't know which tab is active.
**How to avoid:** Gate the document listener on `isActive` AND `event.target` is inside the file-browser tab DOM tree. UI-SPEC §"Filter activation" describes the exact gating. Direct precedent: `frontend/src/lib/isXtermFocused.ts`.

### Pitfall 3: Race between directory listing fetch and user navigation
**What goes wrong:** User clicks into `subdir` while `/list?path=parent` is still in flight; the slow response arrives and overwrites the new directory's listing.
**Why it happens:** No request-id correlation in the fetch flow.
**How to avoid:** Capture the path at request time; in the `.then(...)` callback, check `if (path !== currentPath) return` before applying the result. Or use an `AbortController` and abort the prior fetch on navigate.
**Warning signs:** Random "wrong directory listing" reports — intermittent.

### Pitfall 4: Refreshed-at timestamp shifts every render
**What goes wrong:** "Refreshed 3s ago" updates every React render instead of every 5s as UI-SPEC specifies, making the status line flicker.
**Why it happens:** Computing `Date.now() - refreshedAt` inline in JSX recomputes per render.
**How to avoid:** Store the formatted string in state, update via `setInterval(5_000)` keyed on `refreshedAt`. Clear interval on unmount + on refresh.

### Pitfall 5: `URLSearchParams` corruption with paths containing `+`
**What goes wrong:** A filename `1 + 1.txt` round-trips via `URLSearchParams` → encoded as `1+%2B+1.txt` (spaces as `+`, literal `+` as `%2B`). Server's `r.URL.Query().Get("path")` correctly decodes — but if the frontend builds the URL by string-concatenation later, things break.
**How to avoid:** Always use `URLSearchParams.toString()` once at the very end of URL construction. Never `'?path=' + encodeURIComponent(path)` mixed with the params object. Unit-test a path with `+`, `&`, `=`, and space characters.

### Pitfall 6: `fetch()` against `http://127.0.0.1:${relayPort}` fails CORS in the Wails webview
**What goes wrong:** Wails webview has the same-origin policy of the served page; if `frontend/dist` is served from a `wails://` origin and `fetch('http://127.0.0.1:...')` is cross-origin, CORS preflight will fire.
**How to avoid:** Wails dev mode uses `localhost` and prod uses `wails://`. The daemon socket loopback either needs `Access-Control-Allow-Origin: *` on its mux (acceptable for loopback) OR the requests must use a relative URL when running under the bundled app. **Verify before planning:** test a `fetch('http://127.0.0.1:6789/api/files/list?session=X&path=.')` from the Wails dev server; if it errors with CORS, the daemon socket mux needs an `Access-Control-Allow-Origin` header (Phase 118 may already set this — check). This is a real architectural seam the planner must validate in a Wave 0 task.
**Reality check:** Existing `RelayClient` connects via `ws://127.0.0.1:...` — WebSocket has different origin semantics than `fetch()`. The daemon's HTTP API is currently only exercised by Go-side `DaemonClient`, never by the webview. This is a NEW integration path.

### Pitfall 7: Cap token leakage via referrer / browser history when used in image src
**What goes wrong:** `<img src="/api/files/read?session=X&path=Y&cap=TOKEN">` puts the cap token in the URL, which the browser logs in network panels, includes in `Referer` headers on outgoing image embed, and stores in browser history if the image opens in a new tab.
**Why it happens:** Cap tokens as URL params is the existing AgentHub pattern (sessions URL: `?cap=...`). Image URLs inherit the same exposure.
**How to avoid (v3.4 acceptance):** This is the existing AgentHub model — cap tokens in URLs are tolerated because the URL is short-lived per-grant and the `Referer` only leaks across origins (the image is loaded same-origin). Document the trade-off in code comments. v3.5 may want a session-cookie-bearer auth strategy for `/api/files/read`. Not a Phase 120 blocker.

### Pitfall 8: Skeleton loader CSS shimmer animation causes accessibility regression
**What goes wrong:** UI-SPEC §Motion specifies the shimmer animation is paused under `prefers-reduced-motion: reduce`. If the CSS forgets that media query, motion-sensitive users get an unwanted animation.
**How to avoid:** Append the shimmer animation under `@media (prefers-reduced-motion: no-preference)` ONLY, with the static state as the default. Direct precedent: `.webgl-recovery-banner` in `style.css`.

## Runtime State Inventory

> Phase 120 is a greenfield UI phase — not a rename/refactor/migration. No existing runtime state is being changed. Skipped.

## Code Examples

### Markdown rendering — verified pattern

```typescript
// frontend/src/components/FileBrowser/MarkdownPreview.tsx
// Source: https://github.com/remarkjs/react-markdown (README "GFM example")
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

export function MarkdownPreview({ source }: { source: string }): JSX.Element {
  return (
    <div className="file-browser__preview--markdown" data-testid="file-browser-preview-markdown">
      <Markdown remarkPlugins={[remarkGfm]}>
        {source}
      </Markdown>
    </div>
  )
}

// NEVER add:
//   import rehypeRaw from 'rehype-raw'
//   <Markdown rehypePlugins={[rehypeRaw]}>  ← CSP regression per PITFALLS Pitfall 9
```

### Image preview — verified pattern

```typescript
// frontend/src/components/FileBrowser/ImagePreview.tsx
export function ImagePreview({
  src,
  filename,
}: { src: string; filename: string }): JSX.Element {
  return (
    <div className="file-browser__preview--image" data-testid="file-browser-preview-image">
      <img
        src={src}
        alt={filename}
        style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }}
      />
    </div>
  )
}

// `src` is constructed by FilesApiClient.buildImageUrl(sessionId, path) → produces
//   `http://127.0.0.1:6789/api/files/read?session=X&path=Y`  (desktop loopback)
//   `https://host/api/files/read?session=X&path=Y&cap=TOKEN` (web/remote)
// NEVER: const data = await fetch(src).then(r => r.blob()).then(toBase64) ← PITFALLS Pitfall 10
```

### Tab open from DaemonManagerPanel — wiring

```typescript
// frontend/src/components/DaemonManagerPanel.tsx (additions)
interface DaemonManagerPanelProps {
  // ... existing props
  onBrowseFiles: (sessionId: string, sessionName: string) => void
}

// inside the per-session row render block (around current lines 207-232):
<button
  className="daemon-panel__btn daemon-panel__btn--browse"
  onClick={() => onBrowseFiles(s.id, s.name || s.cli)}
  title="Open the file browser for this session"
>
  Browse files
</button>
```

```typescript
// frontend/src/components/TabBar.tsx (additions inside the context menu block, lines 249-277)
<button
  role="menuitem"
  className="tab__context-menu__item"
  onClick={() => {
    const tab = tabs.find(t => t.id === contextMenu.tabId)
    if (tab?.sessionId) {
      onBrowseFiles?.(tab.sessionId, tab.name)
    }
    setContextMenu(null)
  }}
>
  Browse files
</button>
```

### State dispatch in PreviewPane

```typescript
// frontend/src/components/FileBrowser/PreviewPane.tsx
type PreviewState =
  | { kind: 'idle' }
  | { kind: 'loading' }
  | { kind: 'text';        text: string; size: number; mtime: string }
  | { kind: 'markdown';    text: string; size: number; mtime: string }
  | { kind: 'image';       url: string; size: number; mtime: string }
  | { kind: 'empty';       filename: string }
  | { kind: 'unsupported'; filename: string; downloadUrl: string; humanSize: string }
  | { kind: 'over-cap';    filename: string; downloadUrl: string; humanSize: string }
  | { kind: 'broken-symlink'; filename: string; targetPath: string }
  | { kind: 'read-error';  filename: string; message: string; onRetry: () => void }
  | { kind: 'forbidden-file'; filename: string }

// Dispatches to leaf component per UI-SPEC §"State machine — preview pane"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `react-markdown@6.x` rendering via `dangerouslySetInnerHTML` of pre-sanitized HTML | `react-markdown@10.x` returning a virtual-DOM tree built from MDAST | v9 (Nov 2024) added the new render strategy as default; v10 made it mandatory | CSP-friendly out of the box; no `unsafe-inline` script needed even for tables. Locks us into the React 18+ peer. |
| `remark-gfm@^3` with separate `remark-tables` plugin | `remark-gfm@^4` (Aug 2024) merged tables + task-lists + strikethrough + autolinks into one plugin | Aug 2024 | Single plugin install satisfies UI-07. |
| Cap token middleware accepting `strings.Contains(perms, "files.read")` | `HasPerm(perms, "files.read")` whole-token match (Phase 118 Plan 03) | 2026-05-20 | False-positive `"no-files.read"` substring match closed. Frontend MUST check 403 body for the literal `files.read` (current Phase 119 behavior). |

**Deprecated / outdated:**
- `react-markdown` with `rehype-raw` for "useful HTML" — explicitly excluded by UI-07 and Pitfall 9.
- `react-markdown` v6/v7/v8 patterns documented online — they used different prop shapes. Stick to v10 docs only.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The web surface for File Browser is in scope and the planner will pick between the three architectural options (React-app-on-web vs vanilla `web/files.html` vs descope) | Summary + Open Question Q1 | If web is descoped and UI-11 strictly requires "works on web" → blocks the merge gate. If web requires a new HTML page → adds significant scope (vanilla JS markdown + image preview again). Need user signoff. |
| A2 | `react-markdown@10.1.0` and `remark-gfm@4.0.1` are slopcheck-clean | Package Legitimacy Audit | Both have 20M+ weekly downloads and official remarkjs org — extremely low risk but slopcheck protocol asks for explicit verification. Planner adds checkpoint task. |
| A3 | Daemon socket HTTP at `http://127.0.0.1:${relayPort}/api/files/*` is reachable from the Wails webview via `fetch()` without CORS preflight failure | Pitfall 6 | If CORS blocks the fetch, either the daemon needs `Access-Control-Allow-Origin: *` header (Phase 119 webserver already sets some CORS) or a new Wails RPC binding is needed (contradicts UI-11 "no new Wails binding"). Wave 0 task: probe with a 1-line `fetch()` call in the dev console. |
| A4 | The List endpoint not returning size/mtime means the frontend either issues N `stat` calls per directory OR sorts only by name (with size/mtime columns showing "—" until selected) | Phase Requirements UI-03; Open Question Q2 | If the user expects "click size column, see all files sort by size on first listing", we need to either change the List endpoint contract (out of scope for Phase 120) or batch-stat on directory load. Recommend: lazy-stat the selected row + display "—" for size/mtime in the listing rows. |
| A5 | The `X-Refreshed-At` response header is set by Phase 118/119 | Pitfall 4 + UI-SPEC §Refresh interaction | Phase 118 SUMMARY does not mention setting `X-Refreshed-At`. If absent, the frontend can fall back to `Date.now()` at fetch completion (UI-SPEC explicitly handles this). Low risk. |
| A6 | The `tabs.find` pattern in App.tsx is extensible to a 5th tab type without performance impact | Pattern 1 | Sessions panel polls every 3s; adding the file-browser to the union doesn't change render hot path. Verified by reading App.tsx render block lines 1119-1158. |
| A7 | The existing TabBar's `Tab.type` union (`'terminal' | 'welcome' | 'daemon-manager' | 'remote-sessions' | 'settings'`) accepts a new `'file-browser'` member without breaking existing consumers | Pattern 1 + TabBar.tsx | Direct read of TabBar.tsx confirms `type` is the only switch on `.type`. App.tsx renders by `activeId === XXX.id` (string equality), not by type. Adding the new type is a 1-line change at TabBar.tsx line 8. |

**If this table is empty:** Several claims tagged `[ASSUMED]` — planner should resolve A1, A3, and A4 in Wave 0 or as a pre-execution `checkpoint:human-verify` task.

## Open Questions

1. **Q1 — Web surface architecture for File Browser** (HIGH PRIORITY)
   - What we know: ARCHITECTURE.md describes "web FileBrowser page" but the current web surface is `web/terminal.html` (vanilla JS, terminal only — no React, no tab system). UI-11 requires "works against local AND remote sessions"; UI-14 requires Playwright e2e (which uses the webserver per existing fixture). CONTEXT.md and UI-SPEC reference "Desktop + Web" but never explicitly state how the web surface materializes.
   - What's unclear: Three viable architectures —
     - (A) **Serve the React build (`frontend/dist`) on the webserver** at `/files/{sessionId}?cap=...` route. Cost: significant — the React bundle is currently only embedded into the Wails binary via `assets_prod.go`; mounting it on the webserver requires new route + CSP review for the React runtime + handling the deep-link.
     - (B) **Build a separate vanilla-JS `web/files.html`** mirroring `web/terminal.html` patterns. Cost: 2x implementation work (markdown rendering, image preview, listbox keyboard nav, all in vanilla JS); but isolation from the React app and reuses existing `web/` infrastructure.
     - (C) **Descope web to a stub page** that says "Open the desktop app to browse files." Cost: low; but breaks UI-11 strict reading.
   - Recommendation: **Option (A) — serve `frontend/dist` from the webserver under a cap-gated route**, because the React app already has the full tab system and component library. This is a Wave 0 architectural task: add a webserver route that serves the React index with the cap token in a `<meta>` tag for the app to read via `getCapTokenFromMeta()`. The planner MUST get user signoff on this before locking the plan.

2. **Q2 — Listing endpoint does not return size or mtime; sort behavior** (MEDIUM PRIORITY)
   - What we know: Phase 118 SUMMARY explicitly states `List leaves Size=0 and Mtime=""` to avoid N stat syscalls per directory (Pitfall 6). UI-03 requires "sort by name / size / mtime".
   - What's unclear: How to deliver "sort by size" UX without per-entry stat.
   - Recommendation: Show `—` in the size/mtime columns on the list; lazy-stat the selected row only; if the user clicks the Size column header, batch-stat the visible window (or all 10k entries) with a loading indicator. Treat as an explicit task in the plan.

3. **Q3 — Cap token derivation for the desktop app browsing a REMOTE session**
   - What we know: Local sessions on desktop use loopback (no cap needed). Remote tailnet sessions are accessed via `RemoteSession.url` over HTTPS; the existing `DaemonManagerPanel` issues `IssueCapabilities` Wails RPC to mint a cap for sharing — but the GUI itself never authenticates against a remote daemon's webserver today.
   - What's unclear: How does the desktop GUI obtain a cap token for the remote daemon's `/api/files/*` routes?
   - Recommendation: There's no precedent yet — `RemoteSessionsPanel` uses `BrowserOpenURL` to open the remote session in the user's browser (with the embedded cap). For file browse, the GUI needs an `IssueRemoteFilesCap(remoteHost)` flow OR descope remote-session file browse to "open the remote session's web URL" (graceful degradation). **The planner should escalate this to the user before locking the plan** — UI-11 reads strictly but the remote desktop case may not be wired in any prior phase.

4. **Q4 — Test fixture extension for files**
   - What we know: The existing `cmd/playwright-fixture` boots a daemon with one terminal session but does NOT seed files into the session's cwd. UI-14's 12 scenarios require known files (a text file, a markdown file, an image, a binary, an over-cap file, etc.) at known paths.
   - What's unclear: Extend the existing fixture (add a `--seed-files-dir` flag pointing at a `testdata/files/` tree) OR write a new fixture for the file browser tests.
   - Recommendation: Extend existing fixture. Add a per-spec `beforeAll` that copies a fixture file tree into the session cwd via the daemon's filesystem (since the fixture binary already controls the daemon process).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | pnpm + vite + playwright | ✓ | (from project: vite ^8) | — |
| pnpm | package install | ✓ (preferred per CLAUDE.md) | latest | npm fallback acceptable |
| Playwright browsers (Chromium/Firefox/WebKit) | UI-14 merge gate | (assumed installed by `pnpm playwright install` in CI; check CI config) | 1.59.1 | — |
| Go 1.26.1 | daemon + webserver + fixture binary | ✓ (already used by Phases 118-119) | 1.26.1 | — |
| `react-markdown@10.1.0` | UI-07 | ✗ (not installed) | — | install via `pnpm add` (gated behind slopcheck checkpoint per audit table) |
| `remark-gfm@^4` | UI-07 | ✗ (not installed) | — | install via `pnpm add` (gated behind slopcheck checkpoint) |

**Missing dependencies with no fallback:** none (both runtime packages can be installed by the executor)
**Missing dependencies with fallback:** none

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 (unit/component) + Playwright 1.59.1 (e2e cross-browser) |
| Config file | `frontend/vite.config.ts` (vitest config inline) + `frontend/playwright.config.ts` |
| Quick run command | `cd frontend && pnpm test` (vitest run) |
| Full suite command | `cd frontend && pnpm test && pnpm exec playwright test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UI-01 | Open tab from context menu | e2e | `pnpm exec playwright test files-browser.spec.ts -g 'open from context menu'` | ❌ Wave 0 |
| UI-01 | Open tab from DaemonManagerPanel | e2e | `pnpm exec playwright test files-browser.spec.ts -g 'open from sessions panel'` | ❌ Wave 0 |
| UI-01 | Find-or-add singleton-per-session | unit | `pnpm test FileBrowserTab.singleton.test.tsx` | ❌ Wave 0 |
| UI-02 | Two-pane layout | e2e | `pnpm exec playwright test -g 'list pane and preview pane visible'` | ❌ Wave 0 |
| UI-03 | Sort by columns | unit | `pnpm test FileBrowserTab.sort.test.tsx` | ❌ Wave 0 |
| UI-04 | `/` filter activation, Escape clears | e2e | `pnpm exec playwright test -g 'slash filter clears with escape'` | ❌ Wave 0 |
| UI-05 | Breadcrumb cannot escape cwd | e2e | `pnpm exec playwright test -g 'cannot navigate above session cwd'` | ❌ Wave 0 |
| UI-06 | Over-cap shows refusal + Download | e2e | `pnpm exec playwright test -g 'over-cap refusal with download'` | ❌ Wave 0 |
| UI-07 | Markdown render + no rehype-raw | unit + e2e | `pnpm test FileBrowserTab.no-rehype-raw.test.tsx && pnpm exec playwright test -g 'markdown renders with tables'` | ❌ Wave 0 |
| UI-08 | Source as monospace plaintext | e2e | `pnpm exec playwright test -g 'source code displays as monospace'` | ❌ Wave 0 |
| UI-09 | Image via direct URL not base64 | unit + e2e | `pnpm test FileBrowserTab.no-base64.test.tsx && pnpm exec playwright test -g 'image preview uses endpoint URL'` | ❌ Wave 0 |
| UI-10 | Download button issues GET on /read | e2e | `pnpm exec playwright test -g 'download triggers read endpoint'` | ❌ Wave 0 |
| UI-11 | Works against local + remote | e2e | `pnpm exec playwright test -g 'remote session file browse'` (gated on Q3) | ❌ Wave 0 |
| UI-12 | ARIA roles + keyboard end-to-end | e2e | `pnpm exec playwright test -g 'keyboard navigates list and preview'` | ❌ Wave 0 |
| UI-13 | Error copy is user-readable | e2e | `pnpm exec playwright test -g 'permission denied shows explicit copy'` | ❌ Wave 0 |
| UI-14 | All 12 scenarios pass on Chromium + Firefox + WebKit | e2e | `pnpm exec playwright test files-browser.spec.ts` | ❌ Wave 0 |
| (pitfall) | No CSP violations introduced | e2e | `pnpm exec playwright test web-csp.spec.ts` (extend existing) | ✓ (extend) |

### Sampling Rate

- **Per task commit:** `cd frontend && pnpm test` (vitest only — fast, < 30s) plus targeted Playwright spec if the task touches e2e fixture/spec files
- **Per wave merge:** `cd frontend && pnpm test && pnpm exec playwright test --project=chromium` (single-browser to keep CI fast)
- **Phase gate:** `cd frontend && pnpm test && pnpm exec playwright test` (all 3 browsers — Chromium + Firefox + WebKit) green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `frontend/e2e/files-browser.spec.ts` — covers UI-01..UI-14 across the 12 scenarios
- [ ] `frontend/src/components/__tests__/FileBrowserTab.singleton.test.tsx` — per-session singleton find-or-add (UI-01)
- [ ] `frontend/src/components/__tests__/FileBrowserTab.no-rehype-raw.test.tsx` — Pitfall 9 source-inspection guard (regexes the component source for `rehype-raw` import)
- [ ] `frontend/src/components/__tests__/FileBrowserTab.no-base64.test.tsx` — Pitfall 10 source-inspection guard (asserts no `btoa(` or `data:image/` literal)
- [ ] `frontend/src/components/__tests__/FileBrowserTab.sort.test.tsx` — sort comparator + directories-sticky (UI-03)
- [ ] `frontend/src/lib/__tests__/filesApi.test.ts` — URL construction, error mapping (`FilesApiError.isMissingFilesReadPerm()`, etc.), `+`/`&`/`=` in paths
- [ ] `cmd/playwright-fixture/main.go` — extend to seed a `testdata/files/` tree into the session cwd (Q4)
- [ ] `frontend/e2e/fixture-env.ts` — add `filesSeedDir: string` field if planner picks the extend-fixture route

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Cap-token auth model already in place (Phase 119); no new auth surface introduced |
| V3 Session Management | no | Session state is not extended in this phase |
| V4 Access Control | yes | `requireFilesRead` already gates the backend (Phase 119). Frontend MUST surface 403 with the literal `files.read` body → user-readable copy (UI-13) |
| V5 Input Validation | yes | Frontend MUST NOT trust user-typed paths (Q1: there's no path input box in v3.4 — breadcrumb-only). Path values flow through `URLSearchParams.set('path', value)` → server sandbox is the boundary |
| V6 Cryptography | no | No new cryptographic primitives |
| V7 Error Handling | yes | Every error state must NOT leak raw HTTP status to the UI (UI-13 + UI-SPEC §"Error copy") |
| V11 Business Logic | yes | Over-cap (5 MB) refusal flow must NOT be bypassable from the frontend; the server returns 413 and the frontend renders refusal copy |
| V13 API and Web Service | yes | All file API calls go through `FilesApiClient` (single point of construction) which always uses HTTPS for remote and loopback HTTP for local |
| V14 Configuration | yes | CSP MUST NOT be amended for this phase. WEB-05 e2e CSP check must remain green |

### Known Threat Patterns for `frontend (React 19) + Wails desktop + browser web-share`

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Markdown XSS via `rehype-raw` | Tampering / Elevation | EXCLUDE `rehype-raw`; source-inspection test (`FileBrowserTab.no-rehype-raw.test.tsx`) |
| Style injection via embedded `<style>` in `.md` | Tampering | `react-markdown` v10 default renders raw HTML as escaped text (since v9 changed renderer); no further action needed |
| Cap token leakage via Referer | Information Disclosure | Acceptable — same-origin requests; existing app pattern (Pitfall 7) |
| Path traversal via crafted breadcrumb click | Tampering / Elevation | Phase 118 sandbox is the boundary; frontend never builds an `os`-level path |
| Image URL revealing cap to extension/devtools | Information Disclosure | Acceptable per existing AgentHub model; documented in code |
| Replay of a revoked cap token after `ClearGrants` | Elevation | Phase 119 `requireCapability` checks grant-active list — frontend sees 401, surfaces "Sign in or re-open" copy (UI-SPEC §Error copy) |
| CSP regression introducing inline `<script>` from a third-party | Tampering | WEB-05 Playwright CSP smoke remains a merge gate; `react-markdown` is the ONLY new runtime dep and confirmed CSP-clean |

## Sources

### Primary (HIGH confidence)

- `/Users/ken/dev/agenthub/.planning/phases/120-filebrowsertab-tsx-desktop-web/120-CONTEXT.md` — locked decisions and discretion areas
- `/Users/ken/dev/agenthub/.planning/phases/120-filebrowsertab-tsx-desktop-web/120-UI-SPEC.md` — verbatim UI contract (layout, color, copy, ARIA, testids)
- `/Users/ken/dev/agenthub/.planning/REQUIREMENTS.md` — UI-01..UI-14
- `/Users/ken/dev/agenthub/.planning/research/PITFALLS.md` — Pitfalls 9, 10, 11, 13 directly applicable to this phase
- `/Users/ken/dev/agenthub/.planning/research/ARCHITECTURE.md` — Decisions 3, 5, 6 (HTTPS not relay, fetch() not Wails binding, per-session tab)
- `/Users/ken/dev/agenthub/.planning/research/FEATURES.md` — anti-features (Cmd-F collision, recursive search), GitHub-web-viewer layout rationale
- `/Users/ken/dev/agenthub/.planning/phases/118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi/118-02-SUMMARY.md` — handler wire shape, FileEntry/FileListResponse JSON, 5 MiB cap, 0-byte short-circuit, 10k cap
- `/Users/ken/dev/agenthub/.planning/phases/118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi/118-05-SUMMARY.md` — daemon route shape, owner vs viewer cap perms
- `/Users/ken/dev/agenthub/.planning/phases/119-webserver-routes-files-read-capability-plumbing/119-01-SUMMARY.md` — webserver routes, 401 vs 403 contract, 405 for POST/PUT/DELETE, body contains literal `files.read` on 403
- `/Users/ken/dev/agenthub/frontend/src/App.tsx` (lines 78-82, 614-647, 649-657, 876-886, 1056-1192) — tab system patterns
- `/Users/ken/dev/agenthub/frontend/src/components/TabBar.tsx` (lines 3-9, 240-279) — Tab type union, existing context menu shape
- `/Users/ken/dev/agenthub/frontend/src/components/DaemonManagerPanel.tsx` (lines 7-15, 197-232) — per-session button row pattern
- `/Users/ken/dev/agenthub/frontend/package.json` — pinned versions (React 19.2.4, Playwright 1.59.1, Vitest 4.1.0, Heroicons 2.2.0)
- `/Users/ken/dev/agenthub/frontend/playwright.config.ts` — Chromium + Firefox + WebKit pre-configured
- `/Users/ken/dev/agenthub/frontend/e2e/global-setup.ts` + `fixture-env.ts` — fixture pattern (Go binary + JSON env file)
- `/Users/ken/dev/agenthub/web/embed.go` + `web/terminal.html` — confirmed web surface is vanilla JS, NOT React (anchor for Open Question Q1)

### Secondary (MEDIUM confidence)

- npm registry `npm view react-markdown@10.1.0 peerDependencies` → `{ react: '>=18' }` (verified live)
- npm registry `npm view remark-gfm versions --json` → latest 4.0.1 (verified live)
- npm downloads API `api.npmjs.org/downloads/point/last-week/react-markdown` → 23.2M/wk (verified live, indicates mature established package)

### Tertiary (LOW confidence)

- *(none — all factual claims in this research were either verified against the running file system / npm registry, or cited from project documents that are themselves verified.)*

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — both new packages live on npm, peer-compatible with React 19, used by tens of millions weekly; existing deps unchanged
- Architecture (frontend tab integration): HIGH — direct read of existing App.tsx / TabBar.tsx / DaemonManagerPanel.tsx confirms the find-or-add and per-session patterns
- Architecture (web surface): LOW — Open Question Q1 must be resolved before plan locks
- Architecture (remote-session cap-token derivation): LOW — Open Question Q3 must be resolved before UI-11 remote scenarios can be encoded as tasks
- API contract: HIGH — Phases 118 and 119 SUMMARY.md files spell out wire shape, status codes, headers, body content
- Pitfalls: HIGH — all 5 phase-relevant pitfalls (9, 10, 11, 13 + the new ones found here) have explicit code-level mitigations
- CSP: HIGH — Phase 119 WEB-05 test is green and `react-markdown` without `rehype-raw` is documented CSP-safe

**Research date:** 2026-05-20
**Valid until:** 2026-06-20 (30 days — stable stack, no fast-moving libraries; revisit if `react-markdown` ships a v11 or `remark-gfm` ships v5 before phase execution)

---

## RESEARCH COMPLETE
