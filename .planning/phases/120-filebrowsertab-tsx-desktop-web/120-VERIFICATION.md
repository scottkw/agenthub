---
phase: 120
verified: 2026-05-20T16:50:00Z
status: human_needed
score: 5/5 must-haves verified
requirements_covered: 14/14
overrides_applied: 0
human_verification:
  - test: "Open file browser tab in Wails desktop build and confirm CR-01 fix"
    expected: "Launch `wails dev` or production build → Sessions panel → right-click session → 'Open file browser' → tab opens, breadcrumb shows session cwd, file list populates from the real session cwd (not a 404/NetworkErrorState), text/markdown/image previews render."
    why_human: "Vitest mounts components in jsdom and Playwright e2e (per its own architecture note at files-browser.spec.ts:8-44) exercises only the HTTP API surface — not the live Wails webview hitting 127.0.0.1:relayPort. CR-01's fix is unit-tested at internal/relay/server_files_test.go but the end-to-end Wails desktop click-path was never auto-mode-verifiable. A human must launch the GUI to confirm the file browser actually loads in production."
  - test: "Web-share viewer parity gap — Browse files via web-share works only at the API level in v3.4"
    expected: "Acknowledge that loading /app/?session=…&cap=… in a regular browser (e.g. via the web-share or Tailscale path) yields a partially-functional React shell that imports wailsjs/runtime and does NOT read session/cap from window.location. Per Plan 04 SUMMARY decision #5 (\"Remote browse via web-share is a v3.5 follow-on — the component supports it via props, but the integration trigger is deferred\"), this is a documented v3.5 deferral, not a v3.4 regression. The /app/ route serves the bundle and the API enforces capability gating; only the React shell-level URL-param parsing is deferred."
    why_human: "User awareness gate — confirm this parity gap is acceptable for v3.4 release. If unacceptable, file an issue against scottkw/agenthub to wire web-mode detection into App.tsx (read ?session=&cap= from window.location, skip wailsjs imports under web mode)."
  - test: "Code review WARNINGS (WR-01..WR-07) — defer or fix-now decision"
    expected: "Review the 7 WARNINGS in 120-REVIEW.md (directory listings under /app/, cacheable /app/ bundle assets, joinPath sanitization, formatRowMtime fallback, humanSize truncation comment, useRefreshedText interval cleanup [downgraded — no action], empty alt on image preview). None are merge-blockers but each is a quality issue. Decide: ship as-is (defer to v3.5) or fix-now."
    why_human: "Severity is WARNING (not BLOCKER); the user's call whether to ship and address in v3.5 or fix before tagging v3.4. WR-06 was already analyzed and downgraded to no-action."
---

# Phase 120: FileBrowserTab (TSX) Desktop + Web Verification Report

**Phase Goal:** Users can open a file browser tab for any session, navigate the session's cwd tree, preview text/markdown/image files, download any file, and receive explicit error states for binary/over-cap/permission-denied cases — on both the desktop app and the web frontend.

**Verified:** 2026-05-20T16:50:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria + Phase Goal)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Open file browser tab via session context menu, file list (name/size/mtime), keyboard nav | VERIFIED | `FileBrowserTab.tsx` (590 LOC, full implementation), `fileBrowserTabId()` singleton helper at App.tsx:48 / 893, TabBar context-menu entry + DaemonManagerPanel "Browse files" button wired via `onOpenFileBrowser` callback. Keyboard nav (ArrowUp/Down/Home/End/PgUp/PgDn/Enter/Backspace/`/`) in `FileListPane.tsx` exercised by 17 vitest cases. File list with name/size/mtime columns in `FileRow.tsx` + 3 columnheader buttons in `FileListPane.tsx`. |
| 2 | Text (≤5MB) → monospaced preview; .md → react-markdown + remark-gfm (no raw HTML, no syntax highlighting) | VERIFIED | `TextPreview.tsx` is `<pre>` monospaced. `MarkdownPreview.tsx` uses `<Markdown remarkPlugins={[remarkGfm]}>` — `rehype-raw` / `rehypePlugins` / `dangerouslySetInnerHTML` all banned by source-inspection vitest at `__tests__/FileBrowserTab.no-rehype-raw.test.tsx` (5 cases). 5 MB cap enforced server-side via Phase 118 413 response; client maps to `over-cap` PreviewState. No syntax highlighting (UI-08 — no `highlight.js` / `shiki` / `prismjs` deps present). |
| 3 | PNG/JPEG/WebP/GIF/SVG → `<img src="/api/files/read?...">` (no base64); binary or >5MB → "can't display" + Download | VERIFIED | `ImagePreview.tsx` line 36-…: `<img src={url}>` with direct URL (verified by grep — no `btoa`, no `data:image/`, no `FileReader`, no `toDataURL`, no `URL.createObjectURL`). Source-inspection vitest at `__tests__/FileBrowserTab.no-base64.test.tsx` (6 cases) makes regression detectable. `UnsupportedFile.tsx` renders for both `kind='unsupported'` and `kind='over-cap'` with Download button bound to /api/files/read URL. |
| 4 | Viewer without files.read → "files.read permission required" (not generic 403); breadcrumb bounded to session cwd | VERIFIED | `PermissionDeniedTakeover.tsx` line 28: `<h2>files.read permission required</h2>` — verbatim match. Triggered by `useFilesCapability` 4-state machine (line 46-50): 403 + body contains 'files.read' → `'denied'` state → `FileBrowserTab.tsx:442` full-tab takeover. Breadcrumb sandbox guard: `isPrefixOrEqual(candidate, current)` at `FileBrowserTab.tsx:88-…` + `navigateTo` uses it at line 369. Server-side sandbox is the actual security boundary (Phase 118); UI is defense-in-depth. |
| 5 | Playwright e2e (Chromium + Firefox + WebKit) passes 12 scenarios — merge gate | VERIFIED (with caveat) | `frontend/e2e/files-browser.spec.ts` — 13 `test(...)` calls (12 UI-14 scenarios + 1 API smoke), × 3 browser projects = 39 cells, all PASS per Plan 05 SUMMARY. **Architectural caveat:** the 12 scenarios are API-surface tests against the HTTPS webserver fixture (not React DOM tests through /app/). Each UI-14 scenario maps to a specific API behavior — see Architecture section below. Justification documented in spec header (lines 8-44) and 120-05-SUMMARY.md. **This is the architectural decision flagged for user awareness** (see Human Verification #2). |

**Score:** 5/5 truths verified

### Required Artifacts (3-level verification: exists, substantive, wired)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/server.go` | Mounts /api/files/* on relay TCP listener (CR-01 fix) | VERIFIED | Lines 41-46: `s.mux.HandleFunc("GET /api/files/{list,stat,read}", filesHandler.X)` + HEAD. Wired at api.go:185 `relay.NewServer(..., a.filesHandler)`. Companion regression test at `server_files_test.go:26-72` (TestServer_FilesAPI_MountedOnRelay) passes. |
| `internal/relay/server_files_test.go` | CR-01 regression test exists | VERIFIED | 93 LOC, two test cases — TestServer_FilesAPI_MountedOnRelay (real fetch through relay reaches handler returns FileListResponse) + TestServer_FilesAPI_NilHandler_404 (nil-deref guard). Both PASS via `go test ./internal/relay/...`. |
| `internal/webserver/server.go` | SetStaticAppFS + /app/ route | VERIFIED | Plan 04 added SetStaticAppFS setter + `GET /app/` route with SPA fallback. WR-01 (directory-listing leak under /app/assets/) and WR-02 (cache-control inconsistency) flagged in REVIEW.md as warnings. |
| `internal/daemon/api.go` | SetStaticAppFS wired at both webserver-start sites | VERIFIED | Lines 375 (Local mode start) + 865 (Tailscale mode start): `ws.SetStaticAppFS(getStaticAppFS())`. Verified by grep. |
| `internal/daemon/staticapp.go` | Daemon-side package-level FS holder | VERIFIED | Plan 04 created — thread-safe getter/setter pattern; main.go calls `daemon.SetStaticAppFS(staticAppForDaemon())` so dev builds return nil (503 fallback, not working-dir leak), prod returns the wails embed.FS. |
| `frontend/src/components/FileBrowserTab.tsx` | Orchestrator (590 LOC) | VERIFIED | 590 LOC, full implementation. Wires breadcrumb + list + preview + status, owns directory-list effect + per-file preview effect (both with AbortController + requestId race protection per RESEARCH Pitfall 3). Capability dispatch at line 442. |
| `frontend/src/components/FileBrowser/*.tsx` | 8 leaf components | VERIFIED | All 8 present: BreadcrumbBar, FileListPane, FileRow, StatusLine, PreviewPane, TextPreview, MarkdownPreview, ImagePreview, UnsupportedFile, EmptyDirectoryState, PermissionDeniedTakeover, NetworkErrorState (12 leaf components — exceeds requirement). All have data-testid attribution per UI-SPEC. |
| `frontend/src/lib/filesApi.ts` | FilesApiClient class (not stub) | VERIFIED | 166 LOC. `class FilesApiClient` at line 76, `class FilesApiError` with 4 `is*()` predicates at line 46. Wire shape mirrors `internal/files/types.go`. URLSearchParams (no manual concat). Capability token threaded as `cap` query param when set. |
| `frontend/src/lib/useFilesCapability.ts` | 4-state hook | VERIFIED | 60 LOC. `CapabilityState = 'unknown' \| 'present' \| 'denied' \| 'probe-failed'`. AbortController-style `cancelled` flag for race protection. `isMissingFilesReadPerm()` predicate distinguishes 'denied' (403 + files.read body) from 'probe-failed' (any other error). |
| `frontend/e2e/files-browser.spec.ts` | 12+ Playwright scenarios | VERIFIED | 13 test cases (12 UI-14 + 1 smoke). 39 cells (3 browsers × 13) all PASS per Plan 05 SUMMARY. Suite is API-surface coverage by architectural decision (Plan 04 decision #5 + spec header lines 8-44). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| FileBrowserTab.tsx | /api/files/list,stat,read | FilesApiClient → fetch | WIRED | Constructed in FileBrowserTab; orchestrator effects call client.listFiles / stat / read with AbortController. |
| Wails desktop FileBrowserTab | 127.0.0.1:relayPort/api/files/* | relay HTTP mux | WIRED (CR-01 fix landed) | App.tsx:1124 `fbBaseURL = "http://127.0.0.1:${relayPort}"` → relay.NewServer at api.go:185 mounts files handler → handler at internal/files/handler.go. Regression test at server_files_test.go. |
| Web-share viewer | /api/files/* | HTTPS webserver (TLS + capability middleware) | WIRED (API only) | webserver SetFilesHandler at api.go:375/865; capability middleware fronts the routes; e2e suite proves the path against the playwright-fixture HTTPS listener. **React shell under /app/ does NOT yet read session/cap from URL params — see Human Verification #2.** |
| MarkdownPreview | remark-gfm | remarkPlugins prop | WIRED | `<Markdown remarkPlugins={[remarkGfm]}>{source}</Markdown>` line 30. |
| Capability denial | PermissionDeniedTakeover | useFilesCapability + FilesApiError.isMissingFilesReadPerm() | WIRED | useFilesCapability:46 dispatch → FileBrowserTab:442 full-tab takeover render. |
| App.tsx file-browser tab | Tab.type==='file-browser' render | activeId.startsWith('__files__') | WIRED | App.tsx:1117 gate. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| FileBrowserTab (list) | entries[] | client.listFiles(sessionId, path) → fetch GET /api/files/list → files.Handler.List → sandbox.ReadDir | YES — real fs.ReadDir | FLOWING |
| PreviewPane (text/markdown/image) | state.text / state.url | client.readFile / object URL of /api/files/read | YES — real bytes from files.Handler.Read | FLOWING |
| BreadcrumbBar | refreshedAt | X-Refreshed-At header from /api/files/list | YES — server-supplied RFC3339 | FLOWING |
| useFilesCapability | state | probe `client.listFiles(sessionId, '.')` → real fetch | YES | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go relay tests pass | `go test ./internal/relay/...` | `ok ... 1.996s` | PASS |
| Go webserver tests pass | `go test ./internal/webserver/...` | `ok ... 2.620s` | PASS |
| Go daemon tests pass | `go test ./internal/daemon/...` | `ok ... 8.603s` | PASS |
| Go files tests pass | `go test ./internal/files/...` | `ok (cached)` | PASS |
| Frontend vitest passes | `pnpm test` | `Test Files 72 passed, Tests 1019 passed` | PASS |
| TypeScript clean | `pnpm exec tsc --noEmit` | exit 0, no output | PASS |
| FilesApiClient class exists | `grep "class FilesApiClient" filesApi.ts` | line 76 | PASS |
| FilesApiError class exists | `grep "class FilesApiError" filesApi.ts` | line 46 | PASS |
| No rehype-raw in components | `grep "rehype-raw" components/FileBrowser/` | only doc-comments in MarkdownPreview.tsx + test file | PASS |
| No base64 in components | `grep "base64\|data:image\|btoa\|FileReader\|toDataURL\|createObjectURL"` | only doc-comments in ImagePreview.tsx + test file | PASS |
| E2E scenarios count | `grep -c "test(" files-browser.spec.ts` | 13 (≥12) | PASS |
| SetStaticAppFS wired at both sites | `grep "SetStaticAppFS" api.go` | lines 376 and 866 | PASS |
| Relay /api/files/* mounted | `grep "/api/files/" relay/server.go` | lines 42-45 | PASS |

### Requirements Coverage (UI-01..UI-14)

| Req | Description | Status | Evidence |
|-----|-------------|--------|----------|
| UI-01 | FileBrowserTab registered in tab system; opens via session context menu; per-session singleton | SATISFIED | TabBar context-menu, DaemonManagerPanel button, App.tsx singleton via fileBrowserTabId; vitest FileBrowserTab.singleton.test.tsx |
| UI-02 | Single-pane list (left) + preview (right); 60/40 split; no left tree | SATISFIED | FileBrowserTab.tsx layout, style.css class file-browser__main with grid template |
| UI-03 | Sort by name/size/mtime asc/desc; directories sticky; columnheader toggle | SATISFIED | sortEntries.ts (11 vitest), FileListPane.tsx columnheader buttons |
| UI-04 | Type-ahead filter via `/`; Escape clears | SATISFIED | FileListPane keyDown `/` handler + StatusLine search input (auto-focus via useEffect+ref) |
| UI-05 | Breadcrumb bar; cwd as root; user cannot escape above cwd | SATISFIED | BreadcrumbBar.tsx + isPrefixOrEqual guard in FileBrowserTab; server-side sandbox.ResolvePath is authoritative |
| UI-06 | Text up to 5 MB; over-cap/binary → "can't display" + Download | SATISFIED | UnsupportedFile.tsx (kind='unsupported' or 'over-cap'); 5 MB cap enforced server-side (Phase 118 FS-08); client maps 413 to over-cap |
| UI-07 | Markdown via react-markdown + remark-gfm; NO rehype-raw | SATISFIED | MarkdownPreview.tsx line 30; no-rehype-raw source-inspection vitest |
| UI-08 | Source code as monospaced plain text (no syntax highlighting) | SATISFIED | TextPreview.tsx is `<pre>`; no highlight.js/shiki/prismjs imported |
| UI-09 | Images via `<img src="/api/files/read?...">`; no base64 | SATISFIED | ImagePreview.tsx + no-base64 source-inspection vitest |
| UI-10 | Download via Range-capable /api/files/read | SATISFIED | PreviewPane download button binds href to /read URL; server supports Range (Phase 118); e2e scenario 9 verifies 206 + Content-Range |
| UI-11 | Works against local AND remote sessions; uses fetch() (not Wails binding) | PARTIAL — see human verification #2 | FilesApiClient uses fetch (not Wails). Local path WIRED via relay (CR-01 fix). Remote-viewer path: API gated by capability middleware works (e2e proves it), but React shell App.tsx does not yet detect web mode + read session/cap from URL. Plan 04 SUMMARY decision #5 documents deferral. |
| UI-12 | ARIA semantics; keyboard-only operation; WCAG AA contrast | SATISFIED | role=listbox / option / columnheader / navigation / region throughout; 17 vitest cases on keyboard nav; UI-SPEC color palette reused (no new hex tokens introduced) |
| UI-13 | Empty / network-error / permission-denied each render explicit copy | SATISFIED | EmptyDirectoryState, NetworkErrorState, PermissionDeniedTakeover all present with explicit copy (verified against UI-SPEC) |
| UI-14 | Playwright e2e covers 12 scenarios as merge gate | SATISFIED (API-surface) | 12 scenarios + 1 smoke × 3 browsers = 39 cells PASS. Architectural decision: API-surface tests (justified at spec lines 8-44 + 120-05-SUMMARY); DOM-level testid coverage handled by 75+ vitest cases |

**Coverage:** 14/14 SATISFIED (UI-11 marked PARTIAL but the API path that satisfies the contract is wired; web-share React-shell parity is the v3.5 follow-on flagged for human awareness).

### Anti-Patterns Found

From 120-REVIEW.md adversarial review:

| Severity | ID | File | Issue | Status in this verification |
|----------|-----|------|-------|---------------------------|
| Critical (was) | CR-01 | `frontend/src/App.tsx:1124` + `internal/relay/server.go` | FileBrowserTab hits relay TCP port; relay did not mount /api/files/* → 404 in Wails | **FIXED** — relay/server.go now mounts /api/files/{list,stat,read} (line 41-46) + regression test at server_files_test.go |
| Warning | WR-01 | webserver/server.go:555-578 | /app/ allows directory listings (no assetsNoStore equivalent) | OPEN (defer to v3.5 or fix-now decision — see human verification #3) |
| Warning | WR-02 | webserver/server.go:555-578 | /app/ bundle responses are aggressively cacheable | OPEN |
| Warning | WR-03 | FileBrowserTab.tsx:58-61 | joinPath does not validate name containing `/` | OPEN — server is authoritative, defense-in-depth |
| Warning | WR-04 | FileRow.tsx:57-65 | formatRowMtime malformed-string fallback dumps RFC3339 raw | OPEN |
| Warning | WR-05 | humanSize.ts:7-22 | Comment about truncation rationale is misleading | OPEN |
| Warning | WR-06 | BreadcrumbBar.tsx:49-58 | useRefreshedText interval drift — analyzed and downgraded to no-action | NO ACTION |
| Warning | WR-07 | PreviewPane.tsx:144 | Empty alt on image preview hurts a11y | **FIXED** — line 151 now passes `filename ?? 'image'` |
| Info | IN-01..IN-06 | various | Quality/perf/test-coverage informational items | OPEN (defer to v3.5) |

### Probe Execution

No project-conventional `scripts/*/tests/probe-*.sh` discovered (`find scripts -path '*/tests/probe-*.sh'` returned empty). PLAN/SUMMARY files do not declare probe paths. Phase 120 is a feature-implementation phase (not migration/tooling), so probe execution is not required.

### Architecture Note: API-Surface vs DOM-Surface E2E

Per the spec header at `frontend/e2e/files-browser.spec.ts:8-44` and Plan 05 SUMMARY:

> Plan 05 as originally written expected to exercise the React component tree by navigating to `/app/?session=…&cap=…` in three browsers and matching `data-testid="file-browser-row-*"` etc. against a live React DOM. This is architecturally infeasible in v3.4 because:
> 1. `App.tsx` is Wails-bound — imports `wailsjs/wailsjs/runtime/runtime`, calls `GetRelayPort()` and `GetWebServerMode()`, reads no URL parameters.
> 2. No remote-viewer SPA entry exists yet (v3.5 follow-on, per Plan 04 SUMMARY decision #5).
> 3. Component testids are already covered by 75+ vitest cases in Plans 03 + 04.

This is the **second human verification item** below — a documented parity gap requiring user sign-off.

### Human Verification Required

#### 1. Wails desktop click-path verification (CR-01 fix)

**Test:** Launch `wails dev` or production build, open Sessions panel → right-click session → "Open file browser" (or click DaemonManagerPanel "Browse files"). Confirm the tab opens, breadcrumb shows session cwd, file list populates from the real session cwd (not a 404/NetworkErrorState), text/markdown/image previews render.
**Expected:** Full file browser functionality in the live Wails GUI.
**Why human:** Vitest mounts in jsdom and Playwright e2e is API-surface only (per its own architecture note). The Wails-desktop click-path through 127.0.0.1:relayPort was never auto-mode-verifiable — only the unit-level CR-01 regression test at `internal/relay/server_files_test.go` proves the routing.

#### 2. Web-share viewer parity gap acknowledgment (deferred to v3.5)

**Test:** Acknowledge the documented v3.5 deferral.
**Expected:** Loading `/app/?session=…&cap=…` in a regular browser yields a partially-functional React shell that imports `wailsjs/runtime` and does NOT read session/cap from `window.location`. The `/app/` route serves the bundle and the HTTP API enforces capability gating end-to-end (verified by the 39 e2e cells), but the React-shell wiring for URL-param-driven session/cap binding is a v3.5 follow-on. Plan 04 SUMMARY decision #5 documents this explicitly.
**Why human:** User awareness gate — confirm this parity gap is acceptable for v3.4 release. If unacceptable, file an issue against scottkw/agenthub to wire web-mode detection into App.tsx (read `?session=&cap=` from `window.location`, skip `wailsjs` imports under web mode). The cross-surface parity memory note (`feedback_cross_surface_parity.md`) makes this a release-blocking decision unless explicitly signed off.

#### 3. WARNINGS triage (WR-01..WR-05, WR-07 is fixed)

**Test:** Review the 7 WARNINGS in `120-REVIEW.md` and decide ship-as-is (defer to v3.5) vs fix-now.
**Expected:** Severity is WARNING (not BLOCKER); all together do not block the phase goal. WR-07 is already fixed (PreviewPane.tsx:151 now passes filename). The remaining 6 are quality/hygiene issues.
**Why human:** Triage decision rests with the user.

### Gaps Summary

**No BLOCKER-level gaps.** All 5 ROADMAP success criteria are verified in code, all 14 UI requirements are satisfied (UI-11 partially — its API path is wired; the web-share React-shell wiring for URL-param-based session selection is the deferred v3.5 follow-on documented in Plan 04 SUMMARY decision #5).

CR-01 (the originally-blocking review finding — Wails desktop FileBrowserTab 404'd on relay because relay mux didn't mount /api/files/*) is **fixed**: relay/server.go lines 41-46 now mount the routes, sharing the same `*files.Handler` instance as the daemon Unix socket, with a regression test at `server_files_test.go` (both cases PASS).

The phase is functionally complete pending two human-verification gates:
- (a) live Wails desktop click-path UAT confirming the CR-01 fix works in the real GUI;
- (b) explicit user sign-off on the v3.5 web-share React-shell parity deferral (documented and intentional per Plan 04 SUMMARY decision #5, but per the cross-surface parity memory note, requires explicit acknowledgment).

A third advisory item (WR-01..WR-05 + WR-07-fixed warnings triage) is included for completeness but is not goal-blocking.

---

_Verified: 2026-05-20T16:50:00Z_
_Verifier: Claude (gsd-verifier)_
