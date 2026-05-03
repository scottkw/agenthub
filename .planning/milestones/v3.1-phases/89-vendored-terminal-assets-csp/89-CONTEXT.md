# Phase 89: Vendored Terminal Assets + CSP - Context

**Gathered:** 2026-04-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Close security-review Finding 4 (SEC-07 + SEC-08) by eliminating runtime dependency on `cdn.jsdelivr.net` for the interactive terminal page and layering a strict Content-Security-Policy on every embedded HTML response. After this phase:

1. The terminal page loads xterm JS/CSS only from the embedded app binary — no third-party origins touched at runtime.
2. The terminal page, dashboard page, and join page each carry a strict `Content-Security-Policy` response header that restricts `script-src` / `style-src` / `connect-src` to `'self'` (+ the explicit WSS origin for connect-src), with no `'unsafe-inline'` and no `*` wildcards.
3. Browser devtools shows zero requests to `cdn.jsdelivr.net` (or any third-party origin) during attach, resize, scrollback, detach.
4. Chromium and Safari render all three embedded HTML pages without CSP console violations in both Tailscale-mode FQDN serving and local-network-fallback HTTPS serving.

Out of scope: subresource integrity for optional offline mirrors, CSP reporting endpoints, CSP on JSON/API routes, xterm major-version upgrade, content-addressed cache-busting (deferred to a future perf phase).

Only the tailnet- and LAN-facing `internal/webserver/` plus the `web/` package are modified. The localhost-only `internal/relay/server.go` is out of scope for this phase.

</domain>

<decisions>
## Implementation Decisions

### Vendoring Strategy
- **D-01:** xterm JS/CSS are served from the app binary via **committed vendored files** under `web/vendor/xterm/`. Files tracked in git, embedded via `//go:embed` extension in `web/embed.go`. Offline-buildable, reproducible, no build-time network dependency.
- **D-02:** Files vendored (must match this naming so route patterns are stable):
  - `web/vendor/xterm/xterm.js` — from `frontend/node_modules/@xterm/xterm/lib/xterm.js`
  - `web/vendor/xterm/xterm.css` — from `frontend/node_modules/@xterm/xterm/css/xterm.css`
  - `web/vendor/xterm/addon-fit.js` — from `frontend/node_modules/@xterm/addon-fit/lib/addon-fit.js`
  - `web/vendor/xterm/VERSION` — plain text file with the exact resolved versions from `frontend/pnpm-lock.yaml` for both `@xterm/xterm` and `@xterm/addon-fit`.
- **D-03:** Versions track **`frontend/package.json`** (`@xterm/xterm@^6.0.0`, `@xterm/addon-fit@^0.11.0`). The resolved exact versions from `frontend/pnpm-lock.yaml` are the source of truth; `web/vendor/xterm/VERSION` records them.
- **D-04:** Drift prevention is a **source-parse test**. A Go test reads `frontend/pnpm-lock.yaml`, extracts the resolved versions for `@xterm/xterm` and `@xterm/addon-fit`, reads `web/vendor/xterm/VERSION`, and fails if they differ. Updating xterm becomes a two-step `pnpm update` + copy-files-to-`web/vendor/` workflow; CI catches drift.
- **D-05:** Vendoring is a one-shot copy. No build-time fetch, no `go:generate`, no tarball download step in CI. When the version test fails, a developer manually copies the three files from `node_modules/` and commits both the new files and an updated `VERSION`.

### CSP Strictness & Inline Handling
- **D-06:** **All inline `<script>` and `<style>` blocks in `terminal.html`, `dashboard.html`, and `join.html` are extracted to external files.** No nonces, no hashes, no `'unsafe-inline'`. CSP becomes pure `'self'` for both script-src and style-src everywhere. This is a one-time refactor that eliminates per-request CSP work and is the easiest policy to audit.
- **D-07:** Extracted file layout:
  - `web/terminal.js` + `web/terminal.css` (from `terminal.html`'s ~200 lines inline JS + ~65 lines inline CSS)
  - `web/dashboard.js` + `web/dashboard.css` (from `dashboard.html`'s inline `<script>` at line 73 + inline `<style>`)
  - `web/join.js` + `web/join.css` (from `join.html`'s inline `<script>` at line 156 + inline `<style>`)
  Each HTML file references its companions via `<script src="/assets/terminal.js">` / `<link rel="stylesheet" href="/assets/terminal.css">` etc. No cross-page JS sharing in v3.1.
- **D-08:** All first-party extracted assets + vendored xterm files are added to `//go:embed` in `web/embed.go`. The embed.FS becomes the source of truth for every byte served at `/assets/*`.

### CSP Header Content
- **D-09** (amended 2026-04-22 after e2e finding)**:** Shared **uniform CSP** applied to all three embedded HTML pages. Policy string (single line in the header):
  ```
  default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self' wss://<host>; img-src 'self' data:; font-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'
  ```
  **Amendment rationale:** The original D-09 specified `style-src 'self'` with no `'unsafe-inline'`. `TestBrowserCSP_TerminalNoViolations` (Phase 89 Plan 05 e2e, Chromium) surfaced 12 × `style-src-elem 'inline'` violations on `/sessions/{id}`, caused by xterm.js injecting `<style>` elements at runtime (cursor, selection, theme hooks). The user dispositioned this by allowing `'unsafe-inline'` for style-src only; `script-src 'self'` stays strict — Finding 4's CDN-injection class remains blocked. Updated tests: csp_mw_test.go (NoUnsafeTokens now asserts script-src clause only) and csp_integration_test.go (D-18.3 checks script-src clause only).
  - `default-src 'none'` is the belt that blocks everything not explicitly allowed.
  - `connect-src 'self' wss://<host>` satisfies the literal reading of SEC-08 ("`'self'` plus the explicit WebSocket origin"). The WSS origin is composed from `ws.BaseURL()` per request (see D-10). Dashboard and join never open WSS in practice; including it costs nothing and keeps the header uniform.
  - `img-src 'self' data:` permits inline data-URI PNGs (QR codes rendered client-side, plus xterm's built-in data-URI glyphs).
  - `font-src 'self'` keeps the default browser system stack working; no external fonts are loaded.
  - `base-uri 'none'` blocks injected `<base>` tags; `form-action 'self'` limits the `/join` POST target to same-origin; `frame-ancestors 'none'` blocks clickjacking.
- **D-10:** The `wss://<host>` portion of `connect-src` is **composed dynamically from `ws.BaseURL()` per request**, mirroring Phase 88 D-01/D-11. The CSP middleware reads `BaseURL()` under `ws.mu.RLock()` on every request, substitutes the scheme (`https://` → `wss://`), and splices it into the header string. Handles random-port fallback automatically — whatever port the listener holds is what connect-src reflects.
- **D-11:** The Content-Security-Policy header is the only CSP mechanism. **No `Content-Security-Policy-Report-Only`, no `report-uri`, no `report-to`.** SEC-08 names the enforcement header specifically; reporting infrastructure is deferred (matches Phase 87 D-22 / Phase 88 D-14 minimal-observability stance in security code).

### CSP Scope (Which Pages)
- **D-12:** **All three embedded HTML pages get the CSP header**: `/sessions/{id}`, `/dashboard`, `/join`. Same uniform policy (D-09). Applied by a shared middleware (D-13). API/JSON routes do NOT get a CSP — irrelevant for JSON, uniform across those is noise.
- **D-13:** A dedicated middleware **`cspHeaders(next http.HandlerFunc) http.HandlerFunc`** composes the CSP string and sets the response header before delegating. Mirrors the Phase 88 `requireAllowedOrigin` shape (`func(http.HandlerFunc) http.HandlerFunc`). Wrapped around exactly these three route registrations:
  - `GET /sessions/{id}` — innermost chain `cspHeaders → requireCapability → handleTerminalPage`
  - `GET /dashboard` — `cspHeaders → handleDashboard`
  - `GET /join` — `cspHeaders → handleJoin`
  Middleware file location: `internal/webserver/csp_mw.go` (alongside `capability_mw.go` and any origin middleware from Phase 88).

### Asset Serving
- **D-14:** All embedded static assets (vendored xterm files + first-party extracted JS/CSS) are served via **`http.FileServerFS` mounted at `/assets/`**. A single route: `mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assetsFS)))` where `assetsFS` is a sub-FS built from `webfs.WebFS` rooted at a flat layout so both `/assets/xterm/xterm.js` and `/assets/terminal.js` resolve. Content-Type is inferred from file extension by Go's stdlib.
- **D-15:** `/assets/*` is **public — no capability gate, no Basic Auth gate**. Assets contain no secrets and are identical across sessions. Gating them would cost a 401 on every first page load without adding real confidentiality. Public `/assets/` sits alongside `/dashboard` and `/join` in the non-gated route tier. Exception for local-network fallback mode: assets should ALSO pass through Basic Auth middleware exactly like `/dashboard` currently does — same-level as the other public-but-basic-auth-gated pages.
- **D-16:** **`Cache-Control: no-store`** on every `/assets/*` response during v3.1. Disables browser caching so xterm upgrades and extracted-JS changes take effect immediately on next page load. Cost is small (assets are small, single page load per session open). Content-addressed URLs with long cache TTLs are a deferred perf optimization (NOT this phase).

### Regression Guards
- **D-17:** **Source-grep regression test** (mirrors Phase 88 D-13 / Phase 87 constant-time regression pattern). A Go test reads every `.html`, `.js`, and `.css` file under `web/` and asserts:
  1. No occurrence of `cdn.jsdelivr` anywhere
  2. No occurrence of `unpkg.com` anywhere
  3. No occurrence of `://cdn.` anywhere (catches other CDNs)
  4. No `<script src="http` or `<link href="http` in any `.html` (catches any future cross-origin tag)
  Low-tech, high-signal, lives forever. Fails immediately if a future maintainer pastes a CDN tag back.
- **D-18:** **Integration test: CSP header present + strict**. Against a running `httptest` server, `GET /sessions/{id}` (with valid cap), `GET /dashboard`, `GET /join` each assert:
  1. `Content-Security-Policy` response header is present and non-empty
  2. The header contains `script-src 'self'` and `style-src 'self'` tokens
  3. The header does NOT contain `'unsafe-inline'`, `'unsafe-eval'`, or `*` (outside of data: specifiers)
  4. The header contains `connect-src 'self' wss://` followed by the test server's host:port
  5. The header contains `frame-ancestors 'none'` and `base-uri 'none'`
- **D-19:** **Integration test: browser loads without console violations**. A headless Chromium test (via `github.com/chromedp/chromedp`, already a go.mod-friendly dependency — verify during research phase) navigates to each of the three pages on a test server and asserts that `page.Metrics().ConsoleErrors()` equivalent (specifically CSP violation reports intercepted via CDP's `Log.violationReported` or console `securitypolicyviolation` event) is zero after the page has rendered and the WSS handshake completed. This is the only test that proves SC-4 end-to-end.
- **D-20:** **Source-parse version-drift test** (D-04). Reads `frontend/pnpm-lock.yaml` resolved versions for `@xterm/xterm` and `@xterm/addon-fit`, reads `web/vendor/xterm/VERSION`, fails on mismatch.

### Scope & Constraints
- **D-21:** `internal/webserver/` and `web/` are the only packages modified. The daemon API surface (Unix socket IPC used by GUI/CLI/TUI) is unaffected — this phase has no Daemon-client impact.
- **D-22:** `internal/relay/server.go` is **unchanged** by this phase. The relay serves no HTML and has no CDN dependency. Phase 88 already handled the relay's WSS policy.
- **D-23:** No changes to TLS, capability, or Origin-allowlist mechanics. Phases 87 and 88 remain the authz / handshake layers; Phase 89 layers content policy on top.

### Claude's Discretion
- Exact file name for the CSP middleware (`csp_mw.go` recommended, but `middleware_csp.go` or inline in `server.go` are acceptable if trivial).
- Whether to cache the CSP header string in a `*WebServer` field vs rebuilding per request (per-request is fine; BaseURL already uses an RLock).
- How `assetsFS` (sub-FS rooted at the right path) is constructed from `webfs.WebFS` — `fs.Sub` is the stdlib answer; naming up to planner.
- The exact layout inside the embed.FS — whether extracted files live at `web/terminal.js` (same directory as HTML) or `web/assets/terminal.js`; a `fs.Sub` rebase normalizes URL paths either way. Recommended: keep all first-party assets at `web/` top level, vendored ones at `web/vendor/xterm/`, and use `fs.Sub` to present `/assets/terminal.js` and `/assets/xterm/xterm.js`.
- Naming convention for extracted JS/CSS files (`terminal.js` vs `terminal-page.js` etc.) — planner decides.
- Test file organization (unit vs integration split, one file per concern).
- Whether the chromedp CSP-violation test lives under `internal/webserver/` or a new `internal/webserver/browser_test.go` — planner decides based on build-tag conventions (may need a `//go:build e2e` tag to exclude from fast `go test ./...` runs).
- Error message text when regression tests fail (actionable wording, pointing the reader to D-17 and the Phase 89 context).

### Folded Todos
None — no pending backlog items matched Phase 89's scope.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Threat Model
- `.planning/REQUIREMENTS.md` §Security Hardening — SEC-07 and SEC-08 acceptance criteria
- `.planning/ROADMAP.md` §Phase 89 — Goal + 4 success criteria
- `security-review/SECURITY_REVIEW.md` §Finding 4 — "Terminal page executes third-party CDN JavaScript with no integrity pinning" — exploit scenario and recommended fix (vendor locally + CSP)

### Existing Code That Must Change
- `web/terminal.html:7` — `<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@6/css/xterm.css">` → `<link rel="stylesheet" href="/assets/xterm/xterm.css">`
- `web/terminal.html:65-66` — `<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@6/lib/xterm.js">` + addon-fit → `/assets/xterm/xterm.js` + `/assets/xterm/addon-fit.js`
- `web/terminal.html:67-276` — inline `<script>` block → extract to `web/terminal.js`, reference via `<script src="/assets/terminal.js">`
- `web/terminal.html:8-57` (approximately) — inline `<style>` block → extract to `web/terminal.css`, reference via `<link rel="stylesheet" href="/assets/terminal.css">`
- `web/dashboard.html:73+` — inline `<script>` → extract to `web/dashboard.js`; inline `<style>` → extract to `web/dashboard.css`
- `web/join.html:156+` — inline `<script>` → extract to `web/join.js`; inline `<style>` → extract to `web/join.css`
- `web/embed.go` — extend `//go:embed` to include `terminal.js terminal.css dashboard.js dashboard.css join.js join.css vendor/xterm/*`
- `internal/webserver/server.go:407-415` (`handleDashboard`) — no logic change, but route wiring now wraps in `cspHeaders`
- `internal/webserver/server.go:423-431` (`handleJoin`) — same: wrap in `cspHeaders`
- `internal/webserver/server.go:569-579` (`handleTerminalPage`) — wrap in `cspHeaders`; keep `requireCapability` wrapper
- `internal/webserver/server.go` route registration (around `setupRoutes`) — add `mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assetsFS)))`

### Phase 87/88 Handoffs (patterns to mirror)
- `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-CONTEXT.md` — Middleware composition model; constant-time source-grep regression test pattern
- `.planning/milestones/v3.1-phases/88-websocket-handshake-security/88-CONTEXT.md` — Dynamic per-request allowlist pattern (D-01/D-11) — Phase 89 connect-src composition mirrors this exactly
- `.planning/milestones/v3.1-phases/88-websocket-handshake-security/88-CONTEXT.md` §D-13 — Source-grep regression guard pattern — Phase 89 D-17 mirrors this

### Existing Patterns to Mirror
- `internal/webserver/capability_mw.go` — Middleware shape `func(http.HandlerFunc) http.HandlerFunc` — `cspHeaders` follows the same shape
- Phase 88 origin-allowlist middleware (in `internal/webserver/`, composed outside `requireCapability`) — `cspHeaders` composes *outside* origin and capability middleware for the terminal route, outside any auth for dashboard/join
- `internal/webserver/server.go:321` (`BaseURL()`) — returns canonical `scheme://host:port`; source for `connect-src wss://<host>` after scheme rewrite

### Library / Stack References
- `github.com/coder/websocket` — not modified by this phase but relevant to the CSP `connect-src wss://` composition contract
- `github.com/chromedp/chromedp` — candidate dependency for D-19's browser-level CSP-violation test. Phase 89 research step MUST verify license / go.mod impact / CI viability before locking this choice
- MDN CSP reference for `connect-src` and WSS — confirm that `connect-src 'self'` alone is NOT sufficient for `wss://` in some browsers (Safari historically required explicit `wss://`); this is the load-bearing rationale for D-09's explicit WSS inclusion

### No external ADRs
No external specs, ADRs, or feature docs beyond the above. Requirements are fully captured in `REQUIREMENTS.md`, `ROADMAP.md`, `security-review/SECURITY_REVIEW.md`, and the decisions above.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/embed.go` — `embed.FS` with `//go:embed dashboard.html terminal.html join.html` — extend this list to include new extracted `.js`/`.css` files and the `vendor/xterm/*` tree.
- `frontend/package.json` — `@xterm/xterm@^6.0.0` + `@xterm/addon-fit@^0.11.0` already installed via pnpm. `frontend/pnpm-lock.yaml` holds the resolved exact versions. Vendoring is a one-time copy from `frontend/node_modules/@xterm/...`.
- `internal/webserver/capability_mw.go` — Middleware shape `func(http.HandlerFunc) http.HandlerFunc`. `cspHeaders` is a direct sibling.
- `internal/webserver/server.go:321` `BaseURL()` — already returns canonical `scheme://host:port` form. Reads `ws.listener.Addr()` under `ws.mu.RLock()`. Source for `wss://<host>` composition in the per-request CSP.
- `http.FileServerFS` + `fs.Sub` (stdlib) — native stdlib idiom for serving embed.FS under a URL prefix.

### Established Patterns
- Middleware layering outside-in: Basic Auth (local mode only) → Origin → Capability → handler. Phase 89 adds CSP header-setter outside that chain for the terminal route and as the sole non-auth middleware for dashboard/join.
- Source-grep / source-parse regression tests for security-sensitive decisions (Phase 87 constant-time, Phase 88 origin allowlist, Phase 89 no-CDN + version-drift).
- Minimal observability in security-layer code — Phase 87 D-22 and Phase 88 D-14 stance carries forward: no logs on CSP violations, no report-uri (D-11).

### Integration Points
- `web/embed.go` — central embed-FS declaration; extending it lands every new file under the binary.
- `internal/webserver/server.go` `setupRoutes` — central route registration; add `/assets/` here and rewrap the three HTML handlers.
- `frontend/pnpm-lock.yaml` — version-drift test reads this file directly (no dependency on pnpm CLI at test time); parse the YAML keys for `@xterm/xterm` and `@xterm/addon-fit` resolved versions.

### Creative Options Enabled
- Because all three HTML pages get identical CSP policy and all inline blocks are extracted, the planner can build one shared middleware, one shared `assetsFS` sub-filesystem, and zero per-page special cases. The diff is mostly mechanical.
- The `no-store` cache policy makes upgrades lossless — developers can iterate on `web/terminal.js` without fighting browser cache.

### Constraints
- Safari historically required explicit `wss://<host>` in `connect-src` even when same-origin — Phase 89 research step should verify current Safari (≥ 17) behavior and either confirm `'self'` covers it or keep the explicit `wss://<host>` belt (D-09). When in doubt, keep the explicit WSS clause — SEC-08 literally names it.
- `chromedp` browser tests need headless Chromium available; CI images typically have it but verify during research phase. If unavailable, the planner may move D-19 behind a `//go:build e2e` tag and document how to run it locally.
- The 500KB-ish vendor blob is committed to git; verify no `.gitignore` exclusion under `web/` would silently drop it.

</code_context>

<specifics>
## Specific Ideas

- **Mirror Phase 88's pattern wholesale.** CSP middleware = Origin middleware with a different header being set; dynamic per-request composition from `ws.BaseURL()` = same RLock / substring substitute pattern; source-grep regression guard = same pattern applied to a new forbidden string list.
- **One-time refactor preferred over per-request work.** Extracting inline `<script>`/`<style>` once is a cleaner long-term footprint than nonce generation + HTML rewriting per request, which would also need its own test matrix for randomness and header-HTML sync.
- **Uniform policy over per-page tuning.** Dashboard and join don't use `connect-src wss://` in practice, but giving them the same string as the terminal page eliminates three code paths' worth of drift risk. One CSP string, three routes, done.
- **Strict by default, widen only with a new phase.** `default-src 'none'` + `base-uri 'none'` + `form-action 'self'` + `frame-ancestors 'none'` are table stakes for a content policy; their cost is zero and their blast-radius on future misuse is real.

</specifics>

<deferred>
## Deferred Ideas

- **Subresource Integrity for optional offline mirrors.** If a future deployment wants to load xterm from an internal CDN (not jsdelivr), SRI hashes would be needed. Out of scope — vendoring eliminates the need.
- **Content-addressed URLs + immutable long-cache.** `/assets/xterm/xterm.<sha>.js` with `Cache-Control: public, max-age=31536000, immutable` is a perf win but requires build-time hash-rewriting of HTML references. Worth a dedicated perf phase if page-load latency becomes a complaint.
- **CSP reporting (`Content-Security-Policy-Report-Only` + `report-to`).** Useful for observing what would break before enforcing, and for detecting probe attempts in production. Matches our minimal-observability stance to skip for v3.1; candidate for a v3.2 observability phase.
- **xterm major-version upgrade (6 → 7 or later).** Phase 89 preserves the existing version; upgrades belong to a frontend phase.
- **Shared JS utility module across `terminal.js`, `dashboard.js`, `join.js`.** Small duplication between `dashboard.js` and `join.js` (both render the join-code input) is acceptable; consolidating would be a tech-debt phase.
- **CSP on JSON/API routes.** Harmless but noise; skipped to keep the middleware scope crisp.
- **Network-isolation proof test** (stub NXDOMAIN on cdn.jsdelivr.net, confirm terminal still renders). Declined in favor of the source-grep guard (D-17) and browser-violation test (D-19), which together give the same guarantee with less CI infrastructure.

No todos were reviewed-but-deferred — none matched Phase 89's scope.

</deferred>

---

*Phase: 89-vendored-terminal-assets-csp*
*Context gathered: 2026-04-22*
