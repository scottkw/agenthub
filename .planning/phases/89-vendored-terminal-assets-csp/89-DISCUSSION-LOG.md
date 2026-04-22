# Phase 89: Vendored Terminal Assets + CSP - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-22
**Phase:** 89-vendored-terminal-assets-csp
**Areas discussed:** Vendoring strategy, CSP strictness & inline handling, CSP scope (which pages), Asset serving + regression guards

---

## Vendoring Strategy

### Q1: How should xterm JS/CSS get into the embedded binary?

| Option | Description | Selected |
|--------|-------------|----------|
| Commit files to web/vendor/ | Copy xterm.js, addon-fit.js, xterm.css into web/vendor/xterm/ and commit. Extend //go:embed. Zero build-time network. ~500KB committed. | ✓ |
| Build-step copy from node_modules | build.sh copies from frontend/node_modules right before go build. Not committed; pnpm install is source of truth. | |
| go:generate with SHA verification | Download from npm registry with SHA-256 verification. Explicit 'make vendor' step. Most auditable. | |

**User's choice:** Commit files to web/vendor/
**Notes:** Reproducible, offline-buildable, single source of truth in git.

### Q2: Which xterm version to pin?

| Option | Description | Selected |
|--------|-------------|----------|
| Match frontend (^6.0.0 / ^0.11.0) | Track whatever pnpm-lock.yaml resolves for frontend. One source of truth across GUI + web terminal. | ✓ |
| Pin exact latest stable | Lock to dedicated web-side manifest. Decouples cadence. | |

**User's choice:** Match frontend (^6.0.0 / ^0.11.0)
**Notes:** Avoids dual-version drift.

### Q3: How to keep vendored assets in sync with frontend's pnpm version?

| Option | Description | Selected |
|--------|-------------|----------|
| Manual update + grep test | Two-step manual update; Go test reads pnpm-lock.yaml and web/vendor/xterm/VERSION, fails on drift. | ✓ |
| Script-driven copy | scripts/sync-web-vendor.sh copies from node_modules/. No drift test. | |
| Just vendor, no sync check | Commit once, upgrade when noticed. Silent drift risk. | |

**User's choice:** Manual update + grep test
**Notes:** CI catches drift; keeps the copy step explicit and reviewable.

### Q4: Where do vendored files live in the repo?

| Option | Description | Selected |
|--------|-------------|----------|
| web/vendor/xterm/ | Per-lib subdir under web/vendor/. Clear origin boundary vs first-party. | ✓ |
| web/assets/ | Flat directory mixing first-party and third-party. | |

**User's choice:** web/vendor/xterm/

---

## CSP Strictness & Inline Handling

### Q1: How should the ~200 lines of inline `<script>` in terminal.html be handled?

| Option | Description | Selected |
|--------|-------------|----------|
| Extract to terminal.js | Move all inline JS into web/terminal.js, embed, serve at /assets/terminal.js. CSP becomes pure 'script-src self'. No per-request work. | ✓ |
| Per-request nonce | Middleware generates random nonce per request, injects into HTML + CSP. Keeps inline. Adds per-request crypto. | |
| SHA-256 hash in CSP | Hash inline block at build time, put sha256-... in CSP. Fragile (any edit breaks hash). | |

**User's choice:** Extract to terminal.js

### Q2: How should the ~65 lines of inline `<style>` be handled?

| Option | Description | Selected |
|--------|-------------|----------|
| Extract to terminal.css | Move inline styles into web/terminal.css, embed, serve at /assets/terminal.css. Pure 'style-src self'. | ✓ |
| Keep inline with 'unsafe-inline' | Allows style-src 'self' 'unsafe-inline'. Weaker policy. | |
| Keep inline with nonce | Same per-request nonce mechanism. Requires HTML rewrite per request. | |

**User's choice:** Extract to terminal.css

### Q3: What exact CSP directives should the terminal page carry?

| Option | Description | Selected |
|--------|-------------|----------|
| Strict: self only + explicit WSS | default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self' wss://<host>; img-src 'self' data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'. Literal reading of SEC-08. | ✓ |
| Strict: self only (no explicit WSS) | Same but connect-src 'self' alone. Violates literal reading of SEC-08. | |
| Minimal: just script-src/style-src/connect-src | Only the three SEC-08 names. No default-src 'none' belt. | |

**User's choice:** Strict: self only + explicit WSS
**Notes:** Full belt (default-src 'none') + all hardening directives.

### Q4: How should the WSS origin get into connect-src dynamically?

| Option | Description | Selected |
|--------|-------------|----------|
| Compute from ws.BaseURL() per request | Middleware reads BaseURL() under RLock, rewrites scheme (https→wss), splices into header. Mirrors Phase 88 D-01/D-11. | ✓ |
| 'self' only + wss: scheme | connect-src 'self' wss: — allows WSS to any TLS host. Loosens policy. | |

**User's choice:** Compute from ws.BaseURL() per request

---

## CSP Scope (Which Pages)

### Q1: Which embedded HTML pages should carry a CSP header?

| Option | Description | Selected |
|--------|-------------|----------|
| All three: terminal + dashboard + join | Uniform strict CSP on every embedded HTML response. Single middleware. Matches defense-in-depth. | ✓ |
| Terminal + dashboard only | SC-4 names those two. /join uncovered. | |
| Terminal only | Minimum literal reading of SEC-08. | |

**User's choice:** All three: terminal + dashboard + join

### Q2: Same CSP or page-specific?

| Option | Description | Selected |
|--------|-------------|----------|
| Single shared CSP for all three pages | One strict policy, applied by shared middleware. Uniform prevents drift. | ✓ |
| Per-page CSP | Terminal gets WSS in connect-src; dashboard/join get 'self' only. Tighter but more moving parts. | |

**User's choice:** Single shared CSP for all three pages

### Q3: Extract dashboard and join inline blocks too?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, extract all three | web/terminal.js+css, web/dashboard.js+css, web/join.js+css. Every HTML becomes presentation-only. Uniform 'self' everywhere. | ✓ |
| Yes but keep dashboard/join CSS inline with hash | Only JS extracted; small stable CSS covered by SHA-256. | |
| No — allow 'unsafe-inline' for dashboard/join | Terminal pure 'self'; dashboard/join weaker. Violates 'uniform strict'. | |

**User's choice:** Yes, extract all three

### Q4: connect-src for dashboard/join — they don't open WSS

| Option | Description | Selected |
|--------|-------------|----------|
| Include WSS in shared CSP anyway | Uniform policy; same header for all three pages. Cost is zero. | ✓ |
| Strip WSS from dashboard/join CSP | Two CSP builders. Tighter but doubled audit surface. | |

**User's choice:** Include WSS in shared CSP anyway

---

## Asset Serving + Regression Guards

### Q1: How should embedded static assets be served?

| Option | Description | Selected |
|--------|-------------|----------|
| http.FileServerFS at /assets/ | `http.StripPrefix("/assets/", http.FileServerFS(assetsFS))`. One route handles everything. Content-Type auto. | ✓ |
| Dedicated handler per asset | mux.HandleFunc per file. Greppable but near-identical handlers. | |
| Inline at embed time | Concatenate xterm.js + terminal.js into HTML. Breaks caching. | |

**User's choice:** http.FileServerFS at /assets/

### Q2: Should /assets/* be capability-gated or public?

| Option | Description | Selected |
|--------|-------------|----------|
| Public (no auth) | Assets same across sessions; contain no secrets. No 401 on first load. | ✓ |
| Capability-gated | Any valid cap for any session grants access. Adds UX failure on cap expiry. | |
| Public via /assets but Basic Auth in LAN-fallback mode | Match /dashboard's Basic Auth layering. | |

**User's choice:** Public (no auth)
**Notes:** Decision captured in CONTEXT D-15 with a note that LAN-fallback Basic Auth middleware should still apply to /assets if it currently applies to /dashboard, for same-level treatment.

### Q3: What cache headers should /assets/* carry?

| Option | Description | Selected |
|--------|-------------|----------|
| Cache-Control: no-store (during v3.1) | Disable browser cache. Simplest; avoids stale xterm until cache-busting designed. | ✓ |
| Content-addressed URLs + long cache | /assets/xterm/xterm.<sha>.js + immutable long cache. Best perf but adds build-time hash-rewriting. | |
| Default (no explicit header) | Let FileServerFS emit Last-Modified / ETag only. | |

**User's choice:** Cache-Control: no-store (during v3.1)

### Q4: What regression guards should the phase ship? (multi-select)

| Option | Description | Selected |
|--------|-------------|----------|
| Source-grep: no CDN refs in web/ | Test reads every web/*.html, web/*.js; fails on cdn.jsdelivr, unpkg.com, ://cdn., <script src="http, <link href="http. | ✓ |
| Integration test: CSP header present + strict | httptest server; assert CSP header exists, contains script-src 'self'; no unsafe-inline, no *. | ✓ |
| Integration test: browser loads without console violations | Headless chromedp navigates each page, asserts 0 CSP violations. Strong end-to-end. | ✓ |
| Integration test: network isolation proof | NXDOMAIN stub on cdn.jsdelivr.net, assert terminal still renders. | |

**User's choice:** Source-grep + CSP-header-present + browser-console-violations (three of four)
**Notes:** Network-isolation test declined — source-grep + chromedp already cover it. Added D-20 for version-drift test as mandatory alongside.

### Q5: Where should the CSP header be emitted from?

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated cspHeaders(next) middleware | New middleware analogous to Phase 88 requireAllowedOrigin. Composable, testable in isolation. | ✓ |
| Inline in each handler | Each handler sets CSP directly. Three near-identical writes — drift risk. | |
| Global middleware (all routes) | Apply CSP to JSON/API too. Harmless but noise. | |

**User's choice:** Dedicated cspHeaders(next) middleware

### Q6: Should the CSP include extra hardening directives?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — add frame-ancestors 'none' + base-uri 'none' + form-action 'self' | Blocks clickjacking, <base> injection, form hijack. Cheap, widely supported. | ✓ |
| Only what SEC-08 literally names | script-src, style-src, connect-src. Nothing else. | |

**User's choice:** Yes — add frame-ancestors 'none' + base-uri 'none' + form-action 'self'

### Q7: form-action scope for /join's POST to /join/exchange

| Option | Description | Selected |
|--------|-------------|----------|
| form-action 'self' | Covers /join/exchange same-origin POST. Blocks third-party form hijack. | ✓ |
| form-action 'none' + override on /join | Tightest default, but requires per-route CSP — scope creep. | |

**User's choice:** form-action 'self'

---

## Claude's Discretion

- Exact file name for CSP middleware (csp_mw.go recommended)
- Per-request vs cached CSP header string construction
- assetsFS sub-FS construction path
- Extracted-JS/CSS file naming
- Test file organization and build-tag usage for browser tests
- Error message text in regression tests

## Deferred Ideas

- Subresource Integrity for optional offline mirrors
- Content-addressed URLs + immutable long-cache
- CSP reporting (Report-Only + report-to)
- xterm major-version upgrade
- Shared JS utility module across extracted files
- CSP on JSON/API routes
- Network-isolation proof test (NXDOMAIN stub)
