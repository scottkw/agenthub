---
phase: 120
status: findings
critical_count: 1
warning_count: 7
info_count: 6
---

# Phase 120: Code Review Report

**Depth:** deep
**Files Reviewed:** 22
**Status:** findings

## Summary

Adversarial review of the Phase 120 FileBrowserTab implementation (React orchestrator + 8 leaf components, FilesApiClient + capability hook + helpers + types module, webserver `/app/` route + staticAppFS plumbing, Playwright merge-gate, and main.go embed wiring).

**Headline finding:** the file browser appears to be broken end-to-end in the Wails desktop mode. App.tsx wires `FileBrowserTab` against `http://127.0.0.1:${relayPort}`, but the relay HTTP server (internal/relay/server.go) only exposes `/sessions/{id}/ws` and `/sessions` — it does NOT mount `/api/files/*`. The daemon's `/api/files/*` routes live on `a.mux` and are served on the Unix domain socket, which the Wails-hosted browser cannot reach via TCP. The FilesApiClient will hit 404 on every list/stat/read in desktop mode. The Playwright merge-gate cannot detect this regression because (per the suite's own architecture note at files-browser.spec.ts:11-44) the v3.4 suite only exercises the HTTPS webserver path against the fixture, not the desktop path.

Focus-area assessment:
- XSS/markdown: MarkdownPreview correctly omits rehype-raw and never enables raw HTML.
- Image preview: ImagePreview is `<img src={url}>` with no base64/blob/FileReader pipeline.
- CSP: no new directives needed; `/app/` route returns its bundle without `data:` URLs.
- Path safety (breadcrumb): `isPrefixOrEqual` correctly rejects siblings/parents.
- Capability detection: 4-state machine in useFilesCapability correctly distinguishes `denied` (403+files.read substring) from `probe-failed`.
- 5 MiB cap: enforced server-side; client surfaces 413 → `over-cap` state.
- Wails-vs-web mode detection: **NOT wired (BLOCKER, see CR-01).**
- SetStaticAppFS at both daemon construction sites: confirmed (api.go:371 and api.go:861).
- staticapp.go concurrency: correct RWMutex pattern.
- Worktree stubs: none observed.

## Critical Issues

### CR-01: FileBrowserTab is wired to the relay port, which does not serve `/api/files/*`

**Files:**
- `frontend/src/App.tsx:1124`
- `internal/relay/server.go:22-31`
- `internal/daemon/api.go:131-134`

**Issue:** In App.tsx, the Wails-mode FileBrowserTab is constructed with:
```ts
const fbBaseURL = `http://127.0.0.1:${relayPort ?? 0}`
return <FileBrowserTab ... baseURL={fbBaseURL} />
```
`relayPort` comes from `GetRelayPort()`, which is the port the relay HTTP server listens on (relay.NewServer, started in api.go:177-189). That relay mux only registers two routes:
```go
s.mux.HandleFunc("GET /sessions/{id}/ws", s.handleSession)
s.mux.HandleFunc("GET /sessions", s.handleListSessions)
```
The `/api/files/{list,stat,read}` handlers are registered on `a.mux` (api.go:131-134), which is served on the **Unix domain socket** at `daemon.DefaultSocketPath()` (api.go:204-205) — not on TCP `127.0.0.1:relayPort`.

Net effect in Wails desktop mode:
- `useFilesCapability` probe `GET http://127.0.0.1:RELAY/api/files/list?...` → 404 from relay mux.
- 404 is not `files.read`-bodied, so capability resolves to `probe-failed`, not `denied`.
- The tab body becomes `<NetworkErrorState scope="directory" />` forever; clicking Retry re-issues the same 404.
- No file in the session is ever listable, statable, or readable through the desktop GUI.

The Playwright merge-gate (files-browser.spec.ts) explicitly disclaims coverage of the desktop path (lines 11-44: "Plan 04's SUMMARY acknowledges this explicitly: 'Remote browse via web-share is a v3.5 follow-on'"), so CI is green against the HTTPS webserver fixture even though desktop is broken. The 75+ vitest cases all mount components against jsdom — none of them exercise the real `fetch('http://127.0.0.1:RELAY/...')` call chain.

**Fix:** One of:

1. Mount `/api/files/*` on the relay server's mux as well — minimal patch, since the same `*files.Handler` instance is already in scope:
```go
// internal/relay/server.go — accept a filesHandler argument
func NewServer(manager *HubManager, backend pty.SessionBackend, filesHandler http.Handler) *Server {
    ...
    if filesHandler != nil {
        s.mux.Handle("GET /api/files/list", filesHandler)
        s.mux.Handle("GET /api/files/stat", filesHandler)
        s.mux.Handle("GET /api/files/read", filesHandler)
        s.mux.Handle("HEAD /api/files/read", filesHandler)
    }
    return s
}
```
Then update the api.go:177 `StartRelay()` call site and the cmd/playwright-fixture caller.

2. Expose a Wails-bound IPC method (`ListFiles`, `StatFile`, `ReadFile`) that goes through the Unix socket via `DaemonClient`, and have `FilesApiClient` detect a "wails" environment and route through those bindings instead of `fetch`. Heavier lift but keeps the loopback trust boundary clean.

3. Add a small TCP listener on the daemon that mirrors `a.mux` (or at least the files routes) on `127.0.0.1:randomPort`, return that port through `GetRelayPort` (or a new `GetFilesPort` RPC), and use it from the frontend.

Whichever path is chosen, a desktop-mode integration test that actually mounts the React tree and fires a real fetch is required to prevent this from regressing.

## Warnings

### WR-01: `/app/` route allows directory listings (no `assetsNoStore` equivalent)

**File:** `internal/webserver/server.go:555-578`

**Issue:** The other static-asset routes (`/assets/`, `/assets/xterm/`) are wrapped in `assetsNoStore` (server.go:920-931) which both sets `Cache-Control: no-store` AND 404s any URL that ends with `/` (to block `http.FileServerFS`'s default directory-index behavior). The new `/app/` handler is not wrapped — it uses `http.FileServerFS` directly via `stripped.ServeHTTP(w, r)`. The `/app/` and `/app` cases are explicitly routed to `serveIndex`, but a URL like `/app/assets/` (or any directory that physically exists in the React bundle, such as `/app/locales/`) will reach `http.FileServerFS`. `http.FileServerFS` either serves `index.html` from that subdirectory if one exists, or renders a styled HTML directory listing of the bundle contents.

The React bundle is not particularly sensitive (it's already served as the SPA bundle), but a directory listing leaks file names + sizes + mtimes of every bundled asset, which is unintended.

**Fix:** Apply the same directory-listing guard before delegating to `FileServerFS`:
```go
rel := path[len("/app/"):]
if rel != "" && rel[len(rel)-1] == '/' {
    serveIndex(w, r, appFS) // SPA fallback for nested "directory" requests
    return
}
if _, err := fs.Stat(appFS, rel); err != nil {
    serveIndex(w, r, appFS)
    return
}
stripped.ServeHTTP(w, r)
```

### WR-02: `/app/` bundle responses are aggressively cacheable (no Cache-Control on bundled assets)

**File:** `internal/webserver/server.go:555-578`

**Issue:** `serveIndex` correctly emits `Cache-Control: no-store` for `index.html`, but every other asset under `/app/` (the hashed JS/CSS bundles) reaches `http.FileServerFS` without any `Cache-Control` header. The default response Go's `FileServerFS` writes includes `Last-Modified` from the embed FS, which is usually fine for hashed assets — but `/assets/` is wrapped in `assetsNoStore` for the documented reason "keeps embedded xterm + extracted JS/CSS fresh across deploys without content-hashing the URLs." Inconsistency aside, the React Vite bundle does content-hash its asset URLs, so caching is acceptable there. But if a future deploy ships an unhashed asset under `/app/something.js`, stale caches would silently break the page.

**Fix:** Either explicitly document the caching policy of `/app/*` in the handler comment ("hashed asset filenames carry the cache key — no Cache-Control needed"), or apply `Cache-Control: no-store` consistently with the other static-asset routes.

### WR-03: `joinPath` does not normalize duplicate slashes or `.` components

**File:** `frontend/src/components/FileBrowserTab.tsx:58-61`

**Issue:**
```ts
function joinPath(base: string, name: string): string {
  if (base === '.' || base === '') return name
  return `${base}/${name}`
}
```
If `base === 'subdir/'` (a trailing slash from any source — currently impossible by construction, but defensive) or `name === ''`, the result is malformed (`subdir//` or `subdir/`). More importantly, no validation prevents `name` from containing `/` itself, so navigating "into" an entry whose `name` is the literal string `..` would produce `subdir/..`, which is exactly the sandbox-escape pattern the server is supposed to reject. The server-side guard is the actual security boundary (per the comment at FileBrowserTab.tsx:347-348), but the UI should not depend on the server returning 403 to catch client-side construction of escape paths — a future bug where the server reduces `subdir/..` to `subdir` server-side and serves it would silently bypass UI intent.

**Fix:** Reject `name` values containing `/`, `\`, or equal to `.`/`..` in `navigateInto`:
```ts
const navigateInto = useCallback((name: string) => {
  if (name === '.' || name === '..' || name.includes('/') || name.includes('\\')) return
  setPath((p) => joinPath(p, name))
}, [])
```

### WR-04: `formatRowMtime` malformed-string fallback dumps raw RFC3339 into the row

**File:** `frontend/src/components/FileBrowser/FileRow.tsx:57-65`

**Issue:** When `mtime.length >= 10 && mtime[4] === '-' && mtime[7] === '-'` is false, the function returns the raw `mtime` string. If the server ever returns a malformed mtime (e.g. `"0001-01-01T00:00:00Z"` for missing files, or anything non-RFC3339), the entire string is rendered into the cell, breaking the column width and the row layout. The em-dash fallback at line 58 only triggers on `!mtime`.

**Fix:** Treat any non-conforming string as malformed:
```ts
function formatRowMtime(mtime: string): string {
  if (!mtime) return '—'
  if (mtime.length >= 10 && mtime[4] === '-' && mtime[7] === '-') {
    return mtime.slice(0, 10)
  }
  return '—'
}
```

### WR-05: `humanSize` truncation produces "0.0 KB" for small non-zero KB values

**File:** `frontend/src/lib/humanSize.ts:7-22`

**Issue:** `humanSize(1024)` returns `1.0 KB`. `humanSize(1025)` returns `1.0 KB`. `humanSize(1)` returns `1 B`. `humanSize(50)` returns `50 B`. So far OK. But: `humanSize(102)` (B, under threshold) returns `102 B`. `humanSize(1023)` returns `1023 B`. `humanSize(1024)` returns `1.0 KB`. What about a file slightly under the next unit: `humanSize(1024 * 1024)` returns `1.0 MB`. `humanSize(1024 * 1024 - 1)` returns `1023.9 KB`. OK.

The real issue: the comment claims "truncate (not round half-up) to avoid '5.0 MB' → '5.1 MB' jitter on the 5-MiB cap boundary check used downstream." But there is no downstream `humanSize()` boundary check — the cap check is server-side at exactly 5*1024*1024+1 bytes, returning HTTP 413. The truncation rationale doesn't apply. More importantly, the truncation produces consistently low-biased numbers users won't expect (`5.99 MB` reads as `5.9 MB`). This is a quality issue, not a correctness bug, but the comment is now misleading.

**Fix:** Either switch to `Math.round` for a more natural display, or update the comment to reflect actual intent (display stability across CI).

### WR-06: `useRefreshedText` setInterval drifts when refreshedAt is null

**File:** `frontend/src/components/FileBrowser/BreadcrumbBar.tsx:49-58`

**Issue:** When `refreshedAt` transitions from a value back to `null` (e.g. on capability re-probe), the `useEffect` early-returns at line 52 without clearing a previously-set interval. Subsequent renders will create new intervals on every `refreshedAt` value change, leaking timers. Specifically:
- mount with `refreshedAt = "2026-05-20T12:00:00Z"` → interval#1 created, cleanup registered
- update to `refreshedAt = null` → effect re-runs; line 52 returns BEFORE registering a new cleanup. The prior cleanup runs (clears interval#1) — OK, cleanup runs on every dep change. So actually the cleanup DOES run on the way out. The leak does NOT occur.

Verified: React runs the previous effect's cleanup before running the new effect body, so the early `return` on line 53 is preceded by the prior cleanup. Downgraded from BLOCKER to WARNING because there's still a subtle issue: when `refreshedAt === null`, the displayed text comes from `formatRefreshedAt(null, now)` which returns `''` (correct), but `now` is no longer being updated. If `refreshedAt` later toggles back to a non-null value, the immediate `setNow(Date.now())` at line 53 re-syncs. Not actually broken.

**Fix:** No action required. Verify the cleanup-before-rerun ordering with a unit test if not already present (BreadcrumbBar.test.tsx may already cover this).

### WR-07: Empty `alt` attribute on the image preview hurts accessibility

**File:** `frontend/src/components/FileBrowser/PreviewPane.tsx:144`

**Issue:** `<ImagePreview src={state.url} filename={''} />` — the filename is intentionally elided here because the header already shows it, but inside ImagePreview this becomes `alt=""`. An empty alt tells screen readers to ignore the image entirely, which is fine for purely decorative images but wrong for previewed user content. A blind user has no way to know what file is currently previewed if they navigate directly to the image element.

**Fix:** Thread the filename through:
```tsx
case 'image':
  return <ImagePreview src={state.url} filename={filename ?? 'image'} />
```
Then in ImagePreview, set `alt={filename}` (already done at line 39).

## Info

### IN-01: `compareBySize` integer overflow risk on huge files

**File:** `frontend/src/components/FileBrowser/sortEntries.ts:40-43`

**Issue:** `return a.size - b.size` can lose precision when `size` exceeds 2^53. Realistically the 5 MiB preview cap and 10,000-entry truncation bound the input, but `FileEntry.size` is `number` (JSON-typed via JS), and a directory entry can still be >2^53 (the server doesn't reject the listing). For comparators, use:
```ts
if (a.size !== b.size) return a.size < b.size ? -1 : 1
```

### IN-02: `joinPath` second argument `name` is unsanitized on key path

**File:** `frontend/src/components/FileBrowserTab.tsx:214, 228, 244, 289, 474`

**Issue:** `joinPath(path, selected)` (and similar) embeds the file name as-is in a URL via `URLSearchParams.set('path', path)`. URLSearchParams correctly encodes the path query parameter, so this is safe at the wire level — but if a filename contains ` ` or other control characters, those will be transmitted to the server and the server's response behavior is undefined (sandbox.ResolvePath should reject them but this is a defense-in-depth gap). Logging would also surface raw bytes.

### IN-03: `headFile` returns size from `content-length` which is wrong for compressed transports

**File:** `frontend/src/lib/filesApi.ts:144-152`

**Issue:** `content-length` returned by an HTTP HEAD reflects the on-wire byte count, which can be smaller than the actual file size after gzip/deflate encoding. The webserver doesn't currently set `Content-Encoding`, so this is dormant, but it would silently understate file sizes if a future ServeContent change enables compression. Document or use the `X-File-Size` custom header (or `stat` endpoint) for the authoritative size.

### IN-04: `displayEntries` filter pass is O(n) per render and runs twice

**File:** `frontend/src/components/FileBrowser/FileListPane.tsx:110-117` and `frontend/src/components/FileBrowserTab.tsx:407-411, 415-418`

**Issue:** Both `FileListPane.displayEntries` (memoized) and `FileBrowserTab.visibleCount`/`sortedEntries` (memoized) compute filter+sort independently. With the 10,000-entry cap this is 20,000 extra string comparisons per filter keystroke. Performance is technically out of v3.4 scope, but: the orchestrator could compute the filtered list once and thread `visibleCount` through, eliminating the duplicate pass.

### IN-05: `formatRowSize` em-dashes any file with `size === 0`, not just "size unknown"

**File:** `frontend/src/components/FileBrowser/FileRow.tsx:42-51`

**Issue:** The comment says "files where size==0 (List endpoint default before lazy-stat fires) both render as an em-dash." But a legitimately empty file (`empty.txt` from the Playwright fixture) is indistinguishable from "size unknown" in the row. The UI-SPEC contract may be intentional, but a screen reader sees "size unknown" via `rowAriaLabel` at line 81, which is wrong for a known-zero-byte file.

### IN-06: `humanSize` test claim of unit boundary handling is undocumented

**File:** `frontend/src/lib/humanSize.ts`

**Issue:** No unit test was found for `humanSize.ts`; only behavioral tests exist downstream in FileRow/PreviewPane. The function's invariants ("Negative / NaN / Infinity collapse to '—'", "truncates to one decimal", "1024 boundary") are documented in the file comment but unverified by direct tests. Add a small `humanSize.test.ts` with edge cases (0, 1, 1023, 1024, 1024*1024-1, 1024*1024, -1, NaN, Infinity).

---

_Reviewed: 2026-05-20T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

## REVIEW COMPLETE
