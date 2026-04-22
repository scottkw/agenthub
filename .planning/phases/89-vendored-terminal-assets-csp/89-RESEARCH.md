# Phase 89: Vendored Terminal Assets + CSP — Research

**Researched:** 2026-04-22
**Domain:** Go static-asset serving via embed.FS + HTTP middleware composition + browser CSP
**Confidence:** HIGH (all 9 research questions verified against primary sources; the only remaining uncertainty is CI-side Chromium availability, which is a planner/executor concern, not a research gap)

## Summary

Phase 89 is a mechanical layering phase with well-understood primitives. The Go stdlib (1.22+ `http.FileServerFS` + `io/fs.Sub`) covers all asset-serving needs with zero third-party dependencies. The xterm vendor payload is just 492 KB across three files (no workers, no WASM) resolved to pinned exact versions in `pnpm-lock.yaml` (`@xterm/xterm@6.0.0` + `@xterm/addon-fit@0.11.0`). CSP middleware composition mirrors Phase 88's `requireAllowedOrigin` pattern verbatim — same `func(http.HandlerFunc) http.HandlerFunc` shape, same `ws.BaseURL()` RLock for dynamic per-request `wss://<host>` composition. All three inline-script/style blocks are small and unconditional on DOM state at parse time (the terminal page's IIFE self-bootstraps), so extraction to external files is safe without adding `defer` or `DOMContentLoaded` wrappers.

Two items needed real verification: (1) chromedp viability and (2) Safari's CSP `connect-src 'self'` behavior for wss://.

- **chromedp is viable** but requires a separately-installed Chromium or Chrome binary at test time — it does NOT bundle one. MIT-licensed, actively maintained (releases in April 2026), pure-Go with no CGO. Recommended approach for D-19: gate behind `//go:build e2e` with clear docs in the plan. CSP violations are captured via `chromedp.ListenTarget` on CDP `log.EventEntryAdded` events (Chromium emits "Refused to ..." messages for CSP violations to the console) combined with a JS-injected `window.addEventListener('securitypolicyviolation', ...)` listener read back via `chromedp.Evaluate`. The latter is the cleaner primitive; the former is the belt-and-suspenders backup.
- **Safari still requires explicit `wss://<host>`** in `connect-src` for maximum compatibility. WebKit merged a fix in April 2022 (r292266) in a nightly build, but MDN's current `connect-src` documentation still warns: "`connect-src 'self'` does not resolve to websocket schemes in all browsers." Keeping D-09's explicit `wss://<host>` is the correct defensive choice.

**Primary recommendation:** Implement as CONTEXT.md specifies; no decisions need revisiting. The plan can proceed with a clear 5-wave shape (see `## Validation Architecture` / Nyquist below).

## User Constraints (from CONTEXT.md)

The 23 decisions D-01 through D-23 in `89-CONTEXT.md` are LOCKED. This research confirms all of them are implementable as written and flags no course corrections. Key anchors:

- **Vendoring**: committed files at `web/vendor/xterm/{xterm.js,xterm.css,addon-fit.js,VERSION}`, tracked in git, embedded via `//go:embed vendor/xterm/*` extension.
- **Inline extraction**: all three HTML pages get their `<script>` + `<style>` extracted to sibling `.js`/`.css` files under `web/`.
- **CSP**: uniform policy, dynamic `wss://<host>` composition via `ws.BaseURL()`, no reporting, no `Content-Security-Policy-Report-Only`.
- **Asset serving**: `http.FileServerFS` + `fs.Sub` mounted at `/assets/` — public (not capability-gated), `no-store` cache policy.
- **Regression tests**: source-grep (Phase 87/88 style), integration-header, chromedp-CSP-violation (behind `//go:build e2e`).
- **Scope**: `internal/webserver/` + `web/` only. Relay untouched.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SEC-07 | Terminal page loads xterm JS and CSS from assets served by the embedded app binary (no `https://cdn.jsdelivr.net/...` at runtime) | Q3 (xterm vendoring mechanics), Q4 (FileServerFS), Q8 (gitignore) |
| SEC-08 | Terminal page sets a `Content-Security-Policy` response header restricting `script-src` / `style-src` / `connect-src` to `self` plus the WebSocket origin | Q2 (Safari wss://), Q5 (BaseURL RLock composition), Q6 (inline extraction) |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Embed static assets | Backend build (Go) | — | `embed.FS` + `//go:embed` at compile time; no runtime fetch |
| Serve `/assets/*` | Backend (Go `internal/webserver/`) | — | `http.FileServerFS` + `fs.Sub`; public route, same mux as HTML pages |
| Apply CSP header | Backend (middleware) | Browser (enforcement) | Middleware sets header; browser enforces. Per-request dynamic composition. |
| Extract inline JS/CSS | Static content (`web/*.{js,css}`) | — | Pure refactor; no runtime difference beyond an extra HTTP request per file |
| Origin allowlist for WSS | Backend (Phase 88 layer, unchanged) | Library layer (`coder/websocket.AcceptOptions`) | Phase 89 does not touch this; CSP's `connect-src` is a parallel layer |
| Vendor version drift detection | Backend test | — | Go test reads `pnpm-lock.yaml` + `web/vendor/xterm/VERSION`, no network call |

## Findings

### Q1. chromedp viability for D-19 (browser CSP-violation test)

**License & maintenance:**
- **MIT license** `[VERIFIED: pkg.go.dev/github.com/chromedp/chromedp]` — compatible with the repo's existing licensing stance (coder/websocket is ISC, go-qrcode is MIT, godbus is BSD-2).
- **Actively maintained** — `github.com/chromedp/cdproto/network` published 2026-04-05; core chromedp packages updated through March 2026.

**Go module impact:**
- **No CGO**, pure Go. Implements the async Chrome DevTools Protocol entirely in Go per the official project description.
- Adds `github.com/chromedp/chromedp` + `github.com/chromedp/cdproto` + `github.com/chromedp/sysutil` + `github.com/gobwas/ws` (transitive). Total `go.mod` addition is ~4 entries; none overlap with existing deps.
- Module size is negligible to the build; binary size delta for tests (`go test ./...`) is local-only — chromedp is test-only, not shipped in the production binary.

**Binary dependency:**
- chromedp does NOT bundle Chromium. It requires Chrome, Chromium, or `chromedp/headless-shell` to be available at runtime `[VERIFIED: pkg.go.dev/github.com/chromedp/chromedp]`.
- **Local dev**: Google Chrome.app is already installed on the dev machine (verified via `ls /Applications/`). chromedp auto-detects standard install paths.
- **CI**: GitHub Actions Ubuntu runners include Chromium; macOS runners include Safari but NOT Chromium by default — a CI step would need `brew install chromium` or use `chromedp/headless-shell` via docker. This is a Phase 89 plan decision, not a research blocker.

**CSP violation capture mechanism — two candidates, both verified:**

**Option A — CDP `log.EventEntryAdded` (browser-level console stream):**
```go
// Source: Context7 /chromedp/chromedp docs on ListenTarget (verified 2026-04-22)
chromedp.ListenTarget(ctx, func(ev interface{}) {
    switch e := ev.(type) {
    case *log.EventEntryAdded:
        if e.Entry.Source == log.SourceSecurity || strings.Contains(e.Entry.Text, "Content Security Policy") {
            violations = append(violations, e.Entry.Text)
        }
    }
})
// Must call: chromedp.Run(ctx, log.Enable()) first to receive Log.* events.
```

**Option B — JS `securitypolicyviolation` event (DOM-level):**
```go
// Source: MDN "SecurityPolicyViolationEvent" + chromedp Evaluate pattern
// [VERIFIED: developer.mozilla.org/en-US/docs/Web/API/SecurityPolicyViolationEvent]
const initScript = `
  window.__cspViolations = [];
  document.addEventListener('securitypolicyviolation', (e) => {
    window.__cspViolations.push({
      directive: e.violatedDirective,
      blockedURI: e.blockedURI,
      lineNumber: e.lineNumber,
    });
  });
`
chromedp.Run(ctx,
    page.AddScriptToEvaluateOnNewDocument(initScript),
    chromedp.Navigate(url),
    // ... wait for load + WS handshake ...
)
var violations []map[string]interface{}
chromedp.Run(ctx, chromedp.Evaluate(`window.__cspViolations`, &violations))
```

**Recommendation:** Use **Option B as primary**, Option A as defense-in-depth. Option B reads like a proper assertion in Go test output (`len(violations) == 0` reads cleanly); Option A's console-text matching is fragile across Chrome versions. Both available via `chromedp.ListenTarget` / `page.AddScriptToEvaluateOnNewDocument` in the public chromedp API.

**Build tag gate — YES, use `//go:build e2e`:**
- Default `go test ./...` must stay fast (Phases 87 + 88 run in <5s locally). chromedp tests take 2-10 seconds per page load.
- CI invokes the tag explicitly via `go test -tags=e2e ./...` in a dedicated job.
- Matches Phase 87's precedent of `phase87_wave1` / `phase87_wave2` build tags for gating test categories.

**Fallback if chromedp proves unviable on CI:**
- Source-grep (D-17) + integration-header (D-18) already provide strong coverage; D-19 is the SC-4 end-to-end proof but its absence would not leave security behavior unverified — it would leave the UAT step manual.
- Fallback: document D-19 as a manual UAT item in `89-HUMAN-UAT.md` (mirrors Phase 88 SC-2), run against both Chromium and Safari on the dev Mac before milestone completion.

**Verdict:** chromedp is the right tool; D-19 is implementable as CONTEXT specifies. Gate behind `//go:build e2e` tag; plan should include the CI workflow change (install chromium on Ubuntu runner) as an explicit task, OR document the manual-UAT fallback.

[VERIFIED: pkg.go.dev/github.com/chromedp/chromedp — license, maintenance]
[VERIFIED: Context7 /chromedp/chromedp — ListenTarget, Evaluate API]
[CITED: developer.mozilla.org/en-US/docs/Web/API/SecurityPolicyViolationEvent]

### Q2. Safari CSP behavior for `connect-src` with WSS

**Answer: Keep explicit `wss://<host>` in `connect-src`. D-09 is correct.**

**Evidence trail:**

1. **The spec (CSP3) says `'self'` SHOULD match wss://same-origin** `[CITED: developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/Sources]`. Secure upgrades are allowed: "If the document is served from `ws://example.org`, then a CSP of `'self'` will also permit resources from `wss://example.org`."

2. **BUT MDN's `connect-src` page has an explicit warning** `[CITED: developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/connect-src]`: "`connect-src 'self'` does not resolve to websocket schemes in all browsers, more info in [W3C webappsec-csp issue #7]."

3. **WebKit bug history** `[CITED: bugs.webkit.org/show_bug.cgi?id=201591]`:
   - Filed Sep 2019. Still broken in Safari 15.3 (Mar 2022).
   - Resolved DUPLICATE → merged into bug 235873 `[CITED: bugs.webkit.org/show_bug.cgi?id=235873]` which was RESOLVED FIXED with commit r292266 on 2022-04-02 in a WebKit Nightly Build.
   - **The fix was merged in a nightly build; no definitive Safari stable release is named in either bug tracker.** Safari 16 (Sep 2022) is the earliest stable release that COULD contain the fix, but WebKit bug trackers don't publicly confirm the shipping version.

4. **Current ecosystem guidance (2024-2025)**: Multiple independent blogs and Stack Overflow answers from June 2025 still recommend explicitly listing `wss://<host>` in `connect-src` for Safari/iOS compatibility `[VERIFIED: accent.github.io/security/csp/headers/2025/06/01/safari-connect-src.html — confirmed via WebSearch]`.

5. **Chromium behavior for comparison:** Chromium matches CSP3 spec and treats `'self'` as covering wss://same-origin. So the explicit `wss://<host>` is redundant on Chromium but harmless.

**Conclusion:** D-09 is correct. The explicit `wss://<host>` clause is a defensive belt that costs nothing on Chromium and unlocks Safari compatibility. SEC-08's literal reading ("`'self'` plus the explicit WebSocket origin") also matches this choice.

**Confirm D-09's construction**: `wss://<host>` where `<host>` = authority (host:port) from `ws.BaseURL()` with scheme replaced. No trailing slash, no path. `connect-src` source expressions are origin tuples per CSP spec — no path component accepted. `ws.BaseURL()` returns `https://<host>:<port>` → string-replace `https://` with `wss://` and use the result directly. Verified against Q5 implementation sketch below.

[CITED: developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/connect-src]
[CITED: bugs.webkit.org/show_bug.cgi?id=201591 — original Safari bug]
[CITED: bugs.webkit.org/show_bug.cgi?id=235873 — merged/fixed bug]

### Q3. xterm vendoring mechanics

**Exact resolved versions from `frontend/pnpm-lock.yaml`** `[VERIFIED: read directly at lines 17-19, 26-28, 359-369, 1043-1049]`:

- `@xterm/xterm@6.0.0` (lockfile version resolution)
- `@xterm/addon-fit@0.11.0`

**Exact file paths (after `pnpm install`)** `[VERIFIED: ls frontend/node_modules/@xterm/... on 2026-04-22]`:

```
frontend/node_modules/@xterm/xterm/lib/xterm.js          (487,763 bytes = 476 KB)
frontend/node_modules/@xterm/xterm/css/xterm.css         ( 7,112 bytes =   7 KB)
frontend/node_modules/@xterm/addon-fit/lib/addon-fit.js  ( 1,521 bytes =   2 KB)
```

**Total vendor payload: 496,396 bytes ≈ 485 KB** committed to git. Each file is a single self-contained UMD bundle (no separate maps shipped — `.js.map` siblings stay in node_modules).

**Additional xterm files at runtime?** NO.
- `grep -i "worker|wasm|importScripts|new Worker"` on `xterm.js` returned 1 hit on the minified line — manually verified to be a substring match inside an unrelated identifier, not an actual `new Worker()` call. [VERIFIED by source inspection]
- xterm v6 for browser renders via DOM + canvas/webgl — no separate worker script files needed.
- Optional addons like `@xterm/addon-webgl` are installed in `node_modules` but NOT referenced by `terminal.html`. Scope stays at the 3 files CONTEXT lists.

**pnpm-lock.yaml structure for Go test parsing** `[VERIFIED: sampled lines 359-369 and 1043-1049 of pnpm-lock.yaml]`:

```yaml
# Top-level metadata block (line ~359):
  '@xterm/addon-fit@0.11.0':
    resolution: {integrity: sha512-jYcgT6...}

  '@xterm/xterm@6.0.0':
    resolution: {integrity: sha512-TQwDdQ...}

# Importers block (line ~17):
importers:
  .:
    dependencies:
      '@xterm/addon-fit':
        specifier: ^0.11.0
        version: 0.11.0
      '@xterm/xterm':
        specifier: ^6.0.0
        version: 6.0.0
```

**Recommended Go parser approach for D-04 drift test:**

Rather than parsing YAML, `grep`-style line matching is simpler and sufficient because pnpm's output is stable for top-level package keys:

```go
// Parse resolved version from top-level "  '@xterm/xterm@X.Y.Z':" line pattern.
// Regexp: ^  '@xterm/(xterm|addon-fit)@([0-9.]+)':$
// The top-level block is unambiguous because pnpm-lock.yaml uses that exact
// 2-space indent + quoted package@version + trailing colon pattern for every
// resolution entry.
```

Alternative: pull in `gopkg.in/yaml.v3` (already in `go.mod` as indirect) and parse the whole doc — overkill. The grep approach is the Phase 87/88 pattern.

**VERSION file format (D-02)** — recommend plaintext KV:
```
@xterm/xterm@6.0.0
@xterm/addon-fit@0.11.0
```
One line per package. Test reads both lines, verifies each matches the same-named pnpm-lock entry.

[VERIFIED: pnpm-lock.yaml direct read]
[VERIFIED: ls frontend/node_modules on 2026-04-22]

### Q4. `http.FileServerFS` + `fs.Sub` idioms

**Confirmed stdlib availability** `[VERIFIED: go doc net/http.FileServerFS, go doc io/fs.Sub]`:
- `http.FileServerFS(root fs.FS) Handler` — added Go 1.22.
- `fs.Sub(fsys FS, dir string) (FS, error)` — present since Go 1.16.
- **Current toolchain: Go 1.26.2** (`go.mod` line 3: `go 1.26.1`; active toolchain 1.26.2 verified). Both APIs available.

**Existing repo precedent** — `assets_prod.go:13`:
```go
var assets, _ = fs.Sub(embeddedAssets, "frontend/dist")
```
Same pattern applied at package level. Phase 89 can mirror this exactly.

**Minimal working pattern for Phase 89:**

```go
// In web/embed.go (extended):
package web

import "embed"

//go:embed dashboard.html terminal.html join.html
//go:embed terminal.js terminal.css dashboard.js dashboard.css join.js join.css
//go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
var WebFS embed.FS

// In internal/webserver/server.go setupRoutes():
import (
    "io/fs"
    "net/http"
    webfs "github.com/scottkw/agenthub/web"
)

// AssetsFS presents files under /assets/... flattened so that
//   /assets/terminal.js         → webfs.WebFS/terminal.js
//   /assets/xterm/xterm.js      → webfs.WebFS/vendor/xterm/xterm.js
// requires a tiny custom fs.FS wrapper (see below) OR two separate route mounts.
```

**Layout decision** — CONTEXT.md Claude's Discretion leaves this to the planner. Two viable options:

**Option 1 (recommended) — Two mounts, one FS each:**
```go
// /assets/xterm/xterm.js served from vendor/xterm subtree
xtermFS, _ := fs.Sub(webfs.WebFS, "vendor/xterm")
mux.Handle("GET /assets/xterm/", http.StripPrefix("/assets/xterm/", http.FileServerFS(xtermFS)))

// /assets/terminal.js served from top-level web/ (same dir as HTML)
assetsFS, _ := fs.Sub(webfs.WebFS, ".")  // or build a minimal fs that only exposes the .js/.css siblings
mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assetsFS)))
```
Downside: `fs.Sub(WebFS, ".")` exposes `terminal.html` at `/assets/terminal.html`, which is ugly (and still works via the HTML route), but harmless.

**Option 2 — Single mount + fs.FS wrapper that rewrites paths:**

Small custom `fs.FS` that maps incoming URL paths onto the right real paths. More code; less duplication in route table. Probably over-engineered for v3.1.

**Option 3 (simplest, recommended) — Normalize embed layout:**

Move all extracted JS/CSS under `web/assets/` so the embed-FS layout mirrors the URL:
```
web/
├── dashboard.html
├── terminal.html
├── join.html
└── assets/
    ├── terminal.js
    ├── terminal.css
    ├── dashboard.js
    ├── dashboard.css
    ├── join.js
    ├── join.css
    └── xterm/
        ├── xterm.js
        ├── xterm.css
        ├── addon-fit.js
        └── VERSION

// Then:
assetsFS, _ := fs.Sub(webfs.WebFS, "assets")
mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assetsFS)))
```

**Recommend Option 3.** It's the simplest mental model ("URL path mirrors FS layout"), matches how the `assets_prod.go` precedent sub-rooted at `frontend/dist`, and needs one route + one `fs.Sub`. It slightly contradicts CONTEXT.md's "Recommended: keep all first-party assets at web/ top level" suggestion but that was a Claude's-Discretion recommendation, not a locked decision — planner can choose Option 1 or 3 based on aesthetic preference.

**Content-Type auto-detection** `[CITED: Go net/http source — FileServerFS delegates to ServeFile which sniffs via mime.TypeByExtension]`:
- `.js` → `text/javascript; charset=utf-8`
- `.css` → `text/css; charset=utf-8`
- `.png` → `image/png`
- `VERSION` (no ext) → `text/plain; charset=utf-8` (via content sniffing, since there's no registered MIME for no-ext files)

VERSION file exposure at `/assets/xterm/VERSION` is fine — public, read-only, intentional (lets a curious user view exactly what xterm version is deployed).

**Gotchas with nested subdirs (`/assets/xterm/xterm.js`):**
- `fs.Sub` + `http.FileServerFS` handles nested paths natively; no custom FS code needed.
- `http.StripPrefix` MUST match the mount point exactly — "/assets/" (with trailing slash), or the inner `FileServerFS` will see paths with a leading slash and 404.
- Precedent in the repo (`assets_prod.go` + Wails serving) proves this pattern works in production.

**Cache-Control: no-store (D-16):** Apply in a tiny wrapper middleware, not in FileServerFS (which doesn't accept header overrides):
```go
noCache := func(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "no-store")
        h.ServeHTTP(w, r)
    })
}
mux.Handle("GET /assets/", noCache(http.StripPrefix("/assets/",
    http.FileServerFS(assetsFS))))
```

[VERIFIED: go doc net/http.FileServerFS]
[VERIFIED: go doc io/fs.Sub]
[VERIFIED: assets_prod.go:13 — existing repo usage]

### Q5. CSP header composition under `ws.BaseURL()` RLock

**Verified `BaseURL()` shape** `[VERIFIED: internal/webserver/server.go lines 318-336]`:

```go
// BaseURL returns the base HTTPS URL for the server.
func (ws *WebServer) BaseURL() string {
    ws.mu.RLock()
    ln := ws.listener
    ws.mu.RUnlock()
    if ln == nil {
        return ""
    }
    _, port, err := net.SplitHostPort(ln.Addr().String())
    if err != nil {
        return ""
    }
    if ws.config.Mode == "local" {
        return fmt.Sprintf("https://%s:%s", ws.config.BindIP, port)
    }
    return fmt.Sprintf("https://%s:%s", ws.config.FQDN, port)
}
```

Returns canonical `https://<host>:<port>` (no trailing slash, no path). To derive `wss://<host>:<port>`:

```go
base := ws.BaseURL()
if base == "" {
    // Fail closed — same pattern as origin_mw.go
    http.Error(w, "internal error", http.StatusInternalServerError)
    return
}
wssOrigin := "wss://" + strings.TrimPrefix(base, "https://")
// wssOrigin is now "wss://<host>:<port>" — exactly what connect-src wants.
```

**Per-request CSP composition** — `strings.Builder` with ~8 tokens. Negligible cost:

```go
func (ws *WebServer) composeCSP() string {
    wssOrigin := "wss://" + strings.TrimPrefix(ws.BaseURL(), "https://")
    var b strings.Builder
    b.Grow(256)
    b.WriteString("default-src 'none'; ")
    b.WriteString("script-src 'self'; ")
    b.WriteString("style-src 'self'; ")
    b.WriteString("connect-src 'self' ")
    b.WriteString(wssOrigin)
    b.WriteString("; ")
    b.WriteString("img-src 'self' data:; ")
    b.WriteString("font-src 'self'; ")
    b.WriteString("base-uri 'none'; ")
    b.WriteString("form-action 'self'; ")
    b.WriteString("frame-ancestors 'none'")
    return b.String()
}
```

**Performance:** `BaseURL()` takes one `RLock`/`RUnlock` pair + one `SplitHostPort` + one `Sprintf`. Measured microbenchmark not necessary — the call is on the order of 1-5 µs per request, and page loads are infrequent. HTML response body writes (~10 KB) dominate by 100x. **No need to cache** — per-request composition is fine, matches Phase 88's approach.

**CONTEXT Claude's Discretion check**: D-10 says "per-request is fine; BaseURL already uses an RLock." Confirmed — no cached field on `*WebServer` needed.

**Fail-closed when BaseURL() == "":** This is theoretically unreachable (handlers don't run until `Start()` succeeds and the listener is set), but origin_mw.go still checks defensively (returns 403). The CSP middleware should mirror: if BaseURL() is empty, either serve a 500 or serve the page with a CSP that omits the `wss://` clause but still includes `default-src 'none'` (which blocks everything by default). **Recommend 500** — fail-closed consistency with Phase 88.

**Locking cost under RLock:** `ws.mu.RLock()` allows concurrent reads from `requireCapability`, `BaseURL()`, `isGrantActive`, and `IsSessionEnabled`. A write on `ws.mu.Lock()` (from `SetSigningKey`, `EnableSession`, `AddGrant`) briefly blocks all of them, but these writes are infrequent (session enables are measured in per-user-click, not per-request). Adding one more RLock per HTML request is statistically invisible.

[VERIFIED: internal/webserver/server.go:318-336 BaseURL() source]
[VERIFIED: internal/webserver/origin_mw.go:60-66 — existing similar pattern]

### Q6. Inline script/style extraction mechanics

**Line counts per file (verified by Read tool, 2026-04-22):**

#### `web/terminal.html` (279 lines total)
- **Inline `<style>` block:** lines 8-57 (50 lines, ~1.2 KB)
- **Inline `<script>` block:** lines 67-276 (210 lines, ~9.5 KB)
- **External CDN refs to replace:**
  - Line 7: `<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@6/css/xterm.css">` → `/assets/xterm/xterm.css`
  - Line 65: `<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@6/lib/xterm.js"></script>` → `/assets/xterm/xterm.js`
  - Line 66: `<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.11/lib/addon-fit.js"></script>` → `/assets/xterm/addon-fit.js`
- **Parse-time DOM dependency check:**
  - The inline `<script>` is **at the bottom of `<body>`** (line 67, after `<div id="terminal">` at line 64). By the time the script executes, all DOM nodes it references (`#status-dot`, `#session-info`, `#web-status-bar`, `#terminal`) exist.
  - The script uses an **async IIFE** `(async function initTerminal() { ... })()` at line 162 — self-bootstrapping, does NOT assume `DOMContentLoaded`.
  - **SAFE to extract verbatim** with `<script src="/assets/terminal.js"></script>` at the same position. No `defer` attribute needed (though adding `defer` for belt-and-suspenders is also safe).

#### `web/dashboard.html` (108 lines total)
- **Inline `<style>` block:** lines 7-51 (45 lines, ~1.1 KB)
- **Inline `<script>` block:** lines 73-105 (33 lines, ~1.1 KB)
- **No external CDN refs.**
- **Parse-time DOM dependency check:**
  - Script is at the bottom of `<body>` (after line 71's `</form>` + hint text).
  - Script uses a plain IIFE `(function () { ... })()` that reads `document.getElementById('code')`. The element exists (line 65) before the IIFE runs.
  - **SAFE to extract verbatim.**

#### `web/join.html` (214 lines total)
- **Inline `<style>` block:** lines 7-101 (95 lines, ~2.4 KB)
- **Inline `<script>` block:** lines 156-211 (56 lines, ~2.1 KB)
- **No external CDN refs.**
- **Parse-time DOM dependency check:**
  - Script is at the bottom of `<body>` (after the 5 state divs, last `</div>` at line 154).
  - IIFE wires `document.getElementById('code-b')` conditionally inside `showState('b')` + `wireCodeInput()`. All DOM references resolve to elements defined earlier in the file.
  - **SAFE to extract verbatim.**

**All three files:** inline scripts run AFTER the DOM is fully parsed (scripts at end of `<body>`). Extraction to external files preserves this timing because the browser loads external scripts at the `<script src>` position, not at `<head>`. No `defer` / `async` / `DOMContentLoaded` wrapping needed — the existing IIFE pattern is self-sufficient.

**One subtle consideration for `terminal.html`:**
- The three xterm tags (lines 65-66) currently load BEFORE the inline script (line 67). After extraction, the order must stay: `<script src="/assets/xterm/xterm.js">` → `<script src="/assets/xterm/addon-fit.js">` → `<script src="/assets/terminal.js">`. Script tags with `src=` load in document order (unless `async`/`defer` is set), so preserving source order preserves load order. The inline code at line 186 does `new Terminal(...)` (global `Terminal` from xterm.js) and line 198 does `new FitAddon.FitAddon()` (global `FitAddon` from addon-fit.js). Both globals must be defined before `terminal.js` runs — and they will be, because the three tags are sequential.

**No script restructuring required.** Pure copy-paste refactor.

[VERIFIED: web/terminal.html, web/dashboard.html, web/join.html — Read tool on 2026-04-22]

### Q7. Regression test patterns (Phase 87 / 88 templates)

**Phase 87 constant-time regression test** `[VERIFIED: internal/capability/capability_test.go:157-173]`:

```go
func TestVerify_ConstantTimeComparison(t *testing.T) {
    data, err := os.ReadFile("capability.go")
    if err != nil {
        t.Fatalf("ReadFile capability.go: %v", err)
    }
    src := string(data)
    if !strings.Contains(src, "hmac.Equal") {
        t.Error("capability.go must call hmac.Equal for signature comparison")
    }
    if strings.Contains(src, "bytes.Equal") {
        t.Error("capability.go must not use bytes.Equal on signature bytes (timing side channel)")
    }
}
```

**Phase 88 origin allowlist regression test** `[VERIFIED: internal/webserver/security_regression_test.go:19-56]`:

```go
func TestSecurity_NoAcceptAllOriginInWebserver(t *testing.T) {
    data, err := os.ReadFile("server.go")
    if err != nil {
        t.Fatalf("ReadFile server.go: %v", err)
    }
    src := string(data)
    if strings.Contains(src, `OriginPatterns: []string{"*"}`) {
        t.Error(`server.go must not contain OriginPatterns: []string{"*"} — Phase 88 SC-4 anti-regression; use ws.allowedOrigins() instead`)
    }
}
```

**The pattern**: `os.ReadFile` the target file → `strings.Contains` for forbidden strings → `t.Error` with actionable message referencing the phase decision and the fix direction. Always positive + negative assertions (require present tokens; forbid absent tokens) when possible.

**Phase 89 applies this pattern to walk a DIRECTORY** (all `.html`/`.js`/`.css` under `web/`). Sketch:

```go
func TestSecurity_NoCDNReferencesInWebAssets(t *testing.T) {
    forbidden := []string{
        "cdn.jsdelivr",
        "unpkg.com",
        "://cdn.",
        `<script src="http`,
        `<link href="http`,
    }
    err := filepath.WalkDir("../../web", func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        if d.IsDir() { return nil }
        ext := filepath.Ext(path)
        if ext != ".html" && ext != ".js" && ext != ".css" { return nil }
        data, err := os.ReadFile(path)
        if err != nil { return err }
        src := string(data)
        for _, needle := range forbidden {
            if strings.Contains(src, needle) {
                t.Errorf("%s contains forbidden string %q (Phase 89 D-17 anti-CDN regression)", path, needle)
            }
        }
        return nil
    })
    if err != nil { t.Fatal(err) }
}
```

Note the relative path `../../web` because the test file lives in `internal/webserver/` and needs to reach `web/`. Alternative: put the test in a new package `web/regression_test.go` where `web/` is `.`. Planner's choice.

**Integration header test pattern** — mirror the existing `internal/webserver/origin_integration_test.go` style: use `testServerWithHub` + `http.Client.Do` to observe raw response headers:

```go
func TestCSPHeader_PresentAndStrictOnTerminalPage(t *testing.T) {
    ws, client, _, _ := testServerWithHub(t, "sess-89-csp")
    ws.SetSigningKey(capTestKey)
    token := issueCapFor(t, ws, "sess-89-csp", "read,write")

    url := ws.BaseURL() + "/sessions/sess-89-csp?cap=" + token
    req, _ := http.NewRequest("GET", url, nil)
    resp, err := client.Do(req)
    if err != nil { t.Fatalf("Do: %v", err) }
    defer resp.Body.Close()

    csp := resp.Header.Get("Content-Security-Policy")
    if csp == "" { t.Fatal("expected CSP header, got empty") }
    for _, token := range []string{"script-src 'self'", "style-src 'self'",
        "frame-ancestors 'none'", "base-uri 'none'"} {
        if !strings.Contains(csp, token) {
            t.Errorf("CSP missing required token %q: %s", token, csp)
        }
    }
    for _, forbidden := range []string{"'unsafe-inline'", "'unsafe-eval'"} {
        if strings.Contains(csp, forbidden) {
            t.Errorf("CSP contains forbidden token %q: %s", forbidden, csp)
        }
    }
    // connect-src wss://<host:port> check:
    host := strings.TrimPrefix(ws.BaseURL(), "https://")
    expectedWSS := "wss://" + host
    if !strings.Contains(csp, "connect-src 'self' "+expectedWSS) {
        t.Errorf("CSP connect-src missing %q: %s", expectedWSS, csp)
    }
}
```

[VERIFIED: internal/capability/capability_test.go:157-173]
[VERIFIED: internal/webserver/security_regression_test.go]
[VERIFIED: internal/webserver/origin_integration_test.go — integration helper style]

### Q8. git tracking + .gitignore check

**`.gitignore` inspection** `[VERIFIED: /Users/ken/dev/agenthub/.gitignore, 45 lines total]`:

Rules relevant to `web/vendor/`:

- Line 8: `frontend/node_modules/` — doesn't affect `web/vendor/`.
- Line 9: `frontend/dist/` — doesn't affect `web/vendor/`.
- Line 22: `build/bin/` — Wails build artifacts; not `web/`.
- No rule under `web/`, no `*.js` / `*.css` / `vendor/` pattern.

**Result: `web/vendor/xterm/*.js`, `*.css`, and `VERSION` will commit cleanly.** Nothing in `.gitignore` excludes them.

**Sanity check against the entire `.gitignore`:**
```
/agenthub, /agenthub.exe, /agenthub-cli, /agenthub-cli.exe    (root binaries)
*.exe                                                          (any exe)
frontend/node_modules/                                         (pnpm install output)
frontend/dist/                                                 (vite build output)
.idea/, .vscode/, *.swp, .DS_Store, Thumbs.db                 (IDE/OS)
build/bin/, build/darwin/*                                     (Wails outputs)
frontend/package.json.md5                                      (Wails state file)
frontend/src/wailsjs/wailsjs/go/                               (generated bindings)
*.test, *.out, coverage.html                                   (test artifacts)
.superpowers/                                                  (internal)
*.ts.net.crt, *.ts.net.key                                     (Tailscale local certs)
security-review/                                               (gitignored audit)
```

No pattern matches `web/vendor/` or the vendor file names.

**Recommendation for the plan**: add a guard comment at the top of `web/vendor/xterm/VERSION`:
```
# Do not edit by hand. Tracks resolved versions from frontend/pnpm-lock.yaml.
# Drift is caught by TestXtermVendorVersionsMatchPnpmLock (Phase 89 D-04/D-20).
```
— plaintext, ignored by any parser, useful for future maintainers.

[VERIFIED: .gitignore 45-line direct read]

### Q9. Validation Architecture (Nyquist Dimension 8)

See dedicated section below.

## Validation Architecture

Phase 89 has four validation sampling points. Three are automated, one is manual (browser-level Safari render check).

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` stdlib + `github.com/chromedp/chromedp` (new dep for e2e) |
| Config files | None (Go convention) |
| Quick run command | `go test ./internal/webserver/... -run 'TestCSP\|TestSecurity_No' -count=1` |
| Full suite (unit+integration) | `go test ./... -count=1` |
| E2E (browser CSP check) | `go test -tags=e2e ./internal/webserver/... -run 'TestBrowserCSP' -count=1` |
| Phase gate | Full suite green + e2e green + manual Safari check before `/gsd-verify-work` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SEC-07 | Terminal page has no CDN refs | source-grep | `go test ./internal/webserver/ -run TestSecurity_NoCDNReferencesInWebAssets` | ❌ Wave 0 |
| SEC-07 | xterm files vendored at `web/vendor/xterm/` | file-existence | `go test ./internal/webserver/ -run TestXtermVendorFilesPresent` | ❌ Wave 0 |
| SEC-07 | Vendor versions match pnpm-lock.yaml | source-parse | `go test ./internal/webserver/ -run TestXtermVendorVersionsMatchPnpmLock` | ❌ Wave 0 |
| SEC-07 | `/assets/xterm/xterm.js` serves from embed | HTTP integration | `go test ./internal/webserver/ -run TestAssets_XtermJSServedFromEmbed` | ❌ Wave 0 |
| SEC-08 | CSP header present on /sessions/{id} | HTTP integration | `go test ./internal/webserver/ -run TestCSPHeader_PresentOnTerminalPage` | ❌ Wave 0 |
| SEC-08 | CSP header present on /dashboard | HTTP integration | `go test ./internal/webserver/ -run TestCSPHeader_PresentOnDashboard` | ❌ Wave 0 |
| SEC-08 | CSP header present on /join | HTTP integration | `go test ./internal/webserver/ -run TestCSPHeader_PresentOnJoin` | ❌ Wave 0 |
| SEC-08 | CSP includes script-src/style-src 'self' | HTTP integration | (same tests as above, token assertions) | ❌ Wave 0 |
| SEC-08 | CSP excludes 'unsafe-inline' | HTTP integration | (same tests, negative assertion) | ❌ Wave 0 |
| SEC-08 | CSP has connect-src wss://host:port | HTTP integration | (same tests, BaseURL host assertion) | ❌ Wave 0 |
| SEC-08 | Chromium loads terminal with zero CSP violations | browser E2E | `go test -tags=e2e ./internal/webserver/ -run TestBrowserCSP_TerminalNoViolations` | ❌ Wave 0 |
| SC-4 | Safari loads all 3 pages with zero CSP violations | manual UAT | (checklist in 89-HUMAN-UAT.md) | ❌ Wave 0 (UAT doc) |

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver/... -count=1` (unit + source-grep + integration headers, <5s)
- **Per wave merge:** `go test ./... -count=1` (full repo suite, <30s)
- **Pre-phase-gate:** `go test -tags=e2e ./...` (adds ~30s for headless Chromium) + manual Safari UAT
- **Manual UAT:** on macOS dev machine, open terminal.html and dashboard.html via both Tailscale-mode AND local-network-fallback mode in Safari, visually verify no CSP violations in devtools console. Checklist lives in `89-HUMAN-UAT.md` (plan creates this). Run once before merging to main.

### Reference Dataset / Fixture Needs
- **Existing test capability infrastructure:** `issueCapFor(t, ws, sessionID, "read,write")` helper (from `internal/webserver/capability_test_helpers.go`). Reuse verbatim.
- **Existing test server:** `testServerWithHub(t, sessionID)` helper. Reuse verbatim. Provides TLS, HTTP client, mock PTY pipe.
- **New: Phase 89 fixture** — none needed. All tests drive the real embed.FS (not mocks) because that's the code under test.
- **E2E Chromium binary:** planner must either install Chromium in CI or fallback to manual UAT. Dev machine has Chrome.app installed.

### Wave 0 Gaps
- [ ] `web/vendor/xterm/{xterm.js,xterm.css,addon-fit.js,VERSION}` — vendored files (copy from `frontend/node_modules/`)
- [ ] `web/{terminal,dashboard,join}.{js,css}` — extracted inline blocks (6 files total)
- [ ] `web/embed.go` — extend `//go:embed` directive
- [ ] `internal/webserver/csp_mw.go` — new middleware file
- [ ] `internal/webserver/csp_mw_test.go` — unit tests for compose logic
- [ ] `internal/webserver/csp_integration_test.go` — integration header tests
- [ ] `internal/webserver/assets_test.go` — `/assets/*` route tests
- [ ] `internal/webserver/vendor_drift_test.go` — D-04 / D-20 source-parse test
- [ ] `internal/webserver/no_cdn_regression_test.go` — D-17 source-grep test
- [ ] `internal/webserver/browser_csp_test.go` — D-19 chromedp test (behind `//go:build e2e`)
- [ ] `go.mod` / `go.sum` — add `github.com/chromedp/chromedp` + transitive deps
- [ ] `.planning/phases/89-vendored-terminal-assets-csp/89-HUMAN-UAT.md` — manual Safari checklist

## Project Constraints (from CLAUDE.md)

Extracted from `/Users/ken/dev/agenthub/CLAUDE.md` + `/Users/ken/dev/CLAUDE.md`:

- **Go code**: `go fmt`, `golangci-lint`, context-aware functions. Phase 89 adds HTTP handlers — follow existing `handleXxx(w, r)` shape. No `ctx context.Context` param on `http.HandlerFunc`s (stdlib interface).
- **Commit style**: Conventional Commits (`docs(89):`, `feat(89):`, `test(89):` etc.) — matches Phase 87/88 precedent.
- **Tech stack**: Go backend, not Python/JS for this phase. No CGO.
- **Testing**: `go test`, 80%+ coverage in critical components. CSP middleware qualifies as critical.
- **LSP over grep**: use `gopls` for navigation during implementation.
- **Chesterton's Fence**: existing `OriginPatterns: ws.allowedOrigins()` + `requireCapability` stay untouched by Phase 89.
- **Silent Fallbacks Forbidden**: CSP middleware MUST fail-closed if `BaseURL()` returns "" (no silent header omission).
- **Make beliefs pay rent**: predict CSP behavior before implementing. This research does that.
- **RULE 0 (catastrophic failures)**: N/A for Phase 89 — no data-loss risk in a content-header change.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CSP nonces per request | Full inline extraction + pure `'self'` | Ongoing best practice | Cheaper runtime (no nonce gen), easier audit |
| `http.FileServer(http.FS(...))` | `http.FileServerFS` | Go 1.22 | Cleaner stdlib API, same functionality |
| Manual MIME guessing | `mime.TypeByExtension` built into `FileServerFS` | Go 1.22 | Zero-config content-types |
| Report-Only CSP rollout | Enforcement from day one | Phase 89 D-11 | Matches minimal-observability stance |
| WebSocket CSP `'self'` reliance | Explicit `wss://<host>` in `connect-src` | Cross-browser reality 2019-now | Safari still requires explicit per MDN |

**Deprecated / outdated** to watch for:
- `xterm-addon-fit@0.11` (the non-`@xterm/` scoped variant) — we use `@xterm/addon-fit@0.11.0`. Do NOT re-pin to the unscoped one.
- `ws://` scheme — never appropriate for this app; all serving is TLS. `connect-src` only lists `wss://`, not `ws://`.
- `Content-Security-Policy-Report-Only` — intentionally skipped (D-11).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | CI runners have Chromium installable (GitHub Actions) | Q1 | Moderate — fallback to manual UAT if chromedp can't run in CI |
| A2 | Safari 16+ may have WebKit fix r292266 but not universal across iOS/macOS versions | Q2 | Low — D-09 keeps explicit `wss://<host>` regardless |
| A3 | xterm.js v6 has no runtime worker/wasm file beyond the three we vendor | Q3 | Low — verified via grep + package docs; if later a worker is added, vendor list extends |
| A4 | pnpm-lock.yaml structure is stable for `grep`-based parsing | Q3 | Low — format pinned since pnpm v7 (lockfileVersion 9 in use); v10 would require test update |
| A5 | `fs.Sub` path conventions match `http.FileServerFS` expectations with `StripPrefix` | Q4 | Very low — existing `assets_prod.go` already proves this pattern works in production |

**User confirmation needed:** None. All assumptions are either low-risk or defensively bounded by CONTEXT.md's locked decisions.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build + test | ✓ | 1.26.2 | — |
| pnpm-installed xterm deps in node_modules | vendoring step | ✓ | `@xterm/xterm@6.0.0`, `@xterm/addon-fit@0.11.0` | `pnpm install` from `frontend/` |
| Google Chrome / Chromium | D-19 chromedp e2e test (local) | ✓ (Chrome.app in /Applications) | (macOS managed) | Manual Safari UAT |
| Chromium on CI | D-19 chromedp e2e test (CI) | ✗ (not in default Ubuntu/macOS runners) | — | `apt-get install -y chromium-browser` step OR skip D-19 in CI and make it local-only |
| github.com/chromedp/chromedp | D-19 test | ✗ (new dep) | v0.14.x latest Mar 2026 | N/A — must be added to go.mod |
| `gopkg.in/yaml.v3` | D-04 version-drift test (if YAML parsing chosen) | ✓ (indirect in go.mod) | v3.0.1 | grep-based parsing — preferred, no new dep |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:**
- **Chromium on CI**: plan can add `install-chromium` action step OR document D-19 as local-only + manual-UAT fallback. Recommend local-only to keep CI fast.

## Common Pitfalls

### Pitfall 1: `fs.Sub` silently returns an FS that 404s on nested directory access

**What goes wrong:** `fs.Sub(WebFS, "assets")` works only if `WebFS`'s `//go:embed` directive actually captures `assets/` as a directory — i.e., the embed directive must explicitly name `assets/*` or include `all:` prefix.

**Why it happens:** `//go:embed some.html` only embeds `some.html`, not the directory around it. Nested files need `//go:embed vendor/xterm/*` or `//go:embed assets/*`.

**How to avoid:** Use `//go:embed all:assets` or explicitly list `vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION`. Verified via `go doc embed`: `//go:embed path/*` captures all top-level files in path but NOT nested dirs unless `all:` prefix is used.

**Warning signs:** `404 Not Found` on `/assets/xterm/xterm.js` even though the file exists on disk — embed didn't pick it up.

### Pitfall 2: Inline script extraction breaks if the file is opened directly by the browser (file://)

**What goes wrong:** After extraction, a developer double-clicking `web/terminal.html` in Finder no longer loads the script (file:// URL can't fetch `/assets/terminal.js`).

**Why it happens:** The extracted tags assume absolute URLs (`/assets/...`). They only resolve when served through the Go webserver.

**How to avoid:** Document in a comment at the top of terminal.html: `<!-- These assets resolve only when served via the embedded webserver. Do not open this file directly. -->`. Also, Phase 89 tests run against `testServerWithHub`, not against raw HTML, so tests are unaffected.

### Pitfall 3: Browser caches the old CDN-pinned HTML

**What goes wrong:** A developer upgrades xterm, redeploys, and the browser still hits `cdn.jsdelivr.net` because it cached the old HTML.

**Why it happens:** Old HTML already shipped WITHOUT `Cache-Control: no-store` on the HTML route. Browsers default to heuristic caching of HTML.

**How to avoid:** D-16 already specifies `no-store` on `/assets/*`. For the HTML pages (`/sessions/{id}`, `/dashboard`, `/join`), Phase 89 should ALSO set `Cache-Control: no-store` — but this was not explicitly decided. **Recommend the planner add this** as a trivial one-liner in the same CSP middleware (set both `Content-Security-Policy` AND `Cache-Control: no-store` — two headers, one place). Alternatively, leave HTML cacheable — in practice, tab refresh reloads it, and the CSP update is a one-time milestone change. Planner's call.

### Pitfall 4: Wrong `<meta charset>` or trailing newline in extracted `.css` breaks Content-Type sniffing

**What goes wrong:** `http.FileServerFS` uses `mime.TypeByExtension(".css")` → `text/css; charset=utf-8`. No issue in practice. But if a developer saves `.css` with a BOM, some browsers get confused.

**Why it happens:** Editor auto-adds BOM on save (Windows notepad, rare but real).

**How to avoid:** Not a real risk on the dev Mac. Flag for awareness only.

### Pitfall 5: `wss://host:port` in connect-src with the raw port uncorrected

**What goes wrong:** Browser reports CSP violation for `wss://hostname.ts.net:12345/sessions/x/ws` because the CSP had `wss://hostname.ts.net` (no port).

**Why it happens:** Forgetting that `ws.BaseURL()` returns `https://<host>:<port>` WITH the port, and the CSP source expression must match scheme + host + port exactly.

**How to avoid:** The composition sketch in Q5 above does `"wss://" + strings.TrimPrefix(base, "https://")` — this preserves the port. Verified correct.

**Warning signs:** Terminal page renders, xterm appears, but WebSocket never connects; console shows "Refused to connect to wss://..." violation.

### Pitfall 6: Case-sensitivity in CSP source expressions across browsers

**What goes wrong:** CSP spec says source expressions are case-INSENSITIVE for scheme/host, but some browsers historically were case-sensitive for host.

**Why it happens:** Spec vs. implementation drift.

**How to avoid:** `ws.BaseURL()` returns lowercase (scheme + MagicDNS is lowercase by convention). No uppercase input path. Low risk.

### Pitfall 7: The embed package and the test package see DIFFERENT file paths

**What goes wrong:** `os.ReadFile("../../web/terminal.html")` works from `internal/webserver/foo_test.go` but `webfs.WebFS.ReadFile("terminal.html")` also works from the same package — different paths to the same byte contents.

**Why it happens:** `go:embed` paths are relative to the file declaring the directive, not the caller. Test code reading files from the FS must use the embed-relative path.

**How to avoid:** Use `webfs.WebFS.ReadFile("assets/terminal.js")` (if Option 3 layout is chosen) in integration tests, OR walk the physical filesystem with `os.ReadFile` paths when doing source-grep regression (where the point is to check the source, not the embedded copy). Either is fine; pick one per test.

## Code Examples

### Embed directive extension
```go
// web/embed.go
// Source: Go stdlib embed + existing assets_prod.go pattern
package web

import "embed"

//go:embed dashboard.html terminal.html join.html
//go:embed assets/terminal.js assets/terminal.css
//go:embed assets/dashboard.js assets/dashboard.css
//go:embed assets/join.js assets/join.css
//go:embed assets/xterm/xterm.js assets/xterm/xterm.css assets/xterm/addon-fit.js assets/xterm/VERSION
var WebFS embed.FS
```

### Assets route registration
```go
// internal/webserver/server.go — setupRoutes() additions
// Source: Go stdlib FileServerFS + fs.Sub + existing capability_mw wrapping pattern
assetsFS, err := fs.Sub(webfs.WebFS, "assets")
if err != nil {
    panic(fmt.Sprintf("webserver: fs.Sub assets: %v", err))
}
noStore := func(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "no-store")
        h.ServeHTTP(w, r)
    })
}
mux.Handle("GET /assets/", noStore(http.StripPrefix("/assets/",
    http.FileServerFS(assetsFS))))
```

### CSP middleware
```go
// internal/webserver/csp_mw.go
// Source: mirrors internal/webserver/origin_mw.go shape (Phase 88)
package webserver

import (
    "net/http"
    "strings"
)

// cspHeaders composes and sets the Content-Security-Policy response header
// before delegating to next. Policy is uniform across /sessions/{id},
// /dashboard, and /join per D-09 and D-12.
//
// connect-src's wss://<host> clause is composed from ws.BaseURL() per
// request, mirroring Phase 88's per-request allowlist composition (D-10).
//
// Fail-closed: if ws.BaseURL() returns "" (listener not yet ready — in
// practice unreachable because handlers don't run until Start() succeeds),
// respond 500 rather than serve a CSP missing its wss:// clause.
func (ws *WebServer) cspHeaders(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        base := ws.BaseURL()
        if base == "" {
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }
        wssOrigin := "wss://" + strings.TrimPrefix(base, "https://")
        var b strings.Builder
        b.Grow(256)
        b.WriteString("default-src 'none'; ")
        b.WriteString("script-src 'self'; ")
        b.WriteString("style-src 'self'; ")
        b.WriteString("connect-src 'self' ")
        b.WriteString(wssOrigin)
        b.WriteString("; ")
        b.WriteString("img-src 'self' data:; ")
        b.WriteString("font-src 'self'; ")
        b.WriteString("base-uri 'none'; ")
        b.WriteString("form-action 'self'; ")
        b.WriteString("frame-ancestors 'none'")
        w.Header().Set("Content-Security-Policy", b.String())
        next(w, r)
    }
}
```

### Source-grep regression test (D-17)
```go
// internal/webserver/no_cdn_regression_test.go
// Source: mirrors Phase 88 security_regression_test.go pattern
package webserver

import (
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestSecurity_NoCDNReferencesInWebAssets(t *testing.T) {
    forbidden := []struct{ needle, reason string }{
        {"cdn.jsdelivr", "Phase 89 D-17: xterm must be vendored, not fetched from jsDelivr"},
        {"unpkg.com", "Phase 89 D-17: no CDN dependencies in web assets"},
        {"://cdn.", "Phase 89 D-17: no CDN dependencies (catches generic cdn.* hostnames)"},
        {`<script src="http`, "Phase 89 D-17: all script tags must use relative /assets/ paths"},
        {`<link href="http`, "Phase 89 D-17: all link tags must use relative /assets/ paths"},
    }
    err := filepath.WalkDir("../../web", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            return nil
        }
        ext := filepath.Ext(path)
        if ext != ".html" && ext != ".js" && ext != ".css" {
            return nil
        }
        data, err := os.ReadFile(path)
        if err != nil {
            return err
        }
        src := string(data)
        for _, f := range forbidden {
            if strings.Contains(src, f.needle) {
                t.Errorf("%s contains forbidden string %q — %s", path, f.needle, f.reason)
            }
        }
        return nil
    })
    if err != nil {
        t.Fatalf("walk web/: %v", err)
    }
}
```

### Version-drift test (D-04 / D-20)
```go
// internal/webserver/vendor_drift_test.go
// Reads frontend/pnpm-lock.yaml (grep-style) and web/vendor/xterm/VERSION.
package webserver

import (
    "os"
    "regexp"
    "strings"
    "testing"
)

var pnpmKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-fit))@([0-9.]+)':`)

func TestXtermVendorVersionsMatchPnpmLock(t *testing.T) {
    lock, err := os.ReadFile("../../frontend/pnpm-lock.yaml")
    if err != nil {
        t.Fatalf("ReadFile pnpm-lock.yaml: %v", err)
    }
    pnpmVersions := map[string]string{}
    for _, line := range strings.Split(string(lock), "\n") {
        if m := pnpmKeyRe.FindStringSubmatch(line); m != nil {
            pnpmVersions[m[1]] = m[2]
        }
    }
    if len(pnpmVersions) < 2 {
        t.Fatalf("failed to parse @xterm/xterm and @xterm/addon-fit from pnpm-lock.yaml: found %v", pnpmVersions)
    }

    vendor, err := os.ReadFile("../../web/assets/xterm/VERSION")
    if err != nil {
        t.Fatalf("ReadFile VERSION: %v", err)
    }
    vendorLines := strings.Split(strings.TrimSpace(string(vendor)), "\n")
    vendorVersions := map[string]string{}
    for _, line := range vendorLines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "#") || line == "" {
            continue
        }
        parts := strings.SplitN(line, "@", 3) // "@xterm/xterm@6.0.0" -> ["", "xterm/xterm", "6.0.0"]
        if len(parts) == 3 {
            vendorVersions["@"+parts[1]] = parts[2]
        }
    }

    for pkg, wantV := range pnpmVersions {
        gotV, ok := vendorVersions[pkg]
        if !ok {
            t.Errorf("web/assets/xterm/VERSION missing entry for %s (Phase 89 D-04: update VERSION to match pnpm-lock.yaml)", pkg)
            continue
        }
        if gotV != wantV {
            t.Errorf("version drift: %s pnpm-lock=%s vendor-VERSION=%s — re-copy from frontend/node_modules/%s/ and update VERSION (Phase 89 D-05)", pkg, wantV, gotV, pkg)
        }
    }
}
```

### Browser E2E test (D-19, e2e-tagged)
```go
//go:build e2e

// internal/webserver/browser_csp_test.go
// Source: chromedp Context7 docs — ListenTarget, Evaluate, page.AddScriptToEvaluateOnNewDocument
package webserver

import (
    "context"
    "testing"
    "time"

    "github.com/chromedp/cdproto/page"
    "github.com/chromedp/cdproto/runtime"
    "github.com/chromedp/chromedp"
)

func TestBrowserCSP_TerminalNoViolations(t *testing.T) {
    ws, _, _, _ := testServerWithHub(t, "sess-89-chromedp")
    ws.SetSigningKey(capTestKey)
    token := issueCapFor(t, ws, "sess-89-chromedp", "read,write")

    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", true),
        chromedp.Flag("ignore-certificate-errors", true), // self-signed TLS
    )
    allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
    defer cancel()
    ctx, cancel := chromedp.NewContext(allocCtx)
    defer cancel()
    ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    var consoleMsgs []string
    chromedp.ListenTarget(ctx, func(ev interface{}) {
        if msg, ok := ev.(*runtime.EventConsoleAPICalled); ok {
            for _, arg := range msg.Args {
                if arg.Value != nil {
                    consoleMsgs = append(consoleMsgs, string(arg.Value))
                }
            }
        }
    })

    initScript := `
      window.__cspViolations = [];
      document.addEventListener('securitypolicyviolation', (e) => {
        window.__cspViolations.push({
          directive: e.violatedDirective,
          blocked: e.blockedURI,
        });
      });
    `
    url := ws.BaseURL() + "/sessions/sess-89-chromedp?cap=" + token
    var violations []map[string]interface{}
    err := chromedp.Run(ctx,
        page.AddScriptToEvaluateOnNewDocument(initScript),
        chromedp.Navigate(url),
        chromedp.WaitVisible(`#terminal`, chromedp.ByID),
        chromedp.Sleep(2*time.Second), // let WS handshake complete
        chromedp.Evaluate(`window.__cspViolations`, &violations),
    )
    if err != nil {
        t.Fatalf("chromedp.Run: %v", err)
    }
    if len(violations) > 0 {
        t.Errorf("CSP violations reported: %v", violations)
    }
}
```

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | Capability tokens (Phase 87, unchanged) |
| V3 Session Management | no | Capability TTL (Phase 87, unchanged) |
| V4 Access Control | no | Origin allowlist (Phase 88, unchanged) |
| V5 Input Validation | yes (indirect) | CSP prevents injected script execution — complements input validation by neutralizing XSS payloads that reach the browser |
| V6 Cryptography | no | HMAC signing (Phase 87, unchanged) |
| **V14 Configuration** | **yes** | **CSP header configuration** — ASVS 14.4.5 "all responses contain a Content Security Policy (CSP) header compatible with the application's functional requirements" |
| V14 Configuration (cont.) | yes | ASVS 14.4.2: "prevent click-jacking attacks by using `frame-ancestors 'none'`" — D-09 compliant |
| V14 Configuration (cont.) | yes | ASVS 14.4.3: "verify that a content type header with a safe character set is specified" — `http.FileServerFS` auto-sets via `mime.TypeByExtension`, compliant |

### Known Threat Patterns for this Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| CDN supply-chain injection (Finding 4 / SEC-07) | Tampering | Vendor locally + CSP blocks external script fetches even if HTML is modified |
| Cross-site script injection in terminal output | Tampering | `default-src 'none'` + `script-src 'self'` blocks inline + external scripts |
| Click-jacking on dashboard | Elevation of Privilege | `frame-ancestors 'none'` blocks embedding in frame/iframe |
| `<base>` tag injection rewriting asset URLs | Tampering | `base-uri 'none'` blocks injected `<base href>` tags |
| Form submission hijack (join flow) | Elevation of Privilege | `form-action 'self'` restricts `<form action>` targets to same-origin |
| Mixed content (http asset in https page) | Tampering | CSP source expressions are explicit schemes (https, wss) — no ambiguity |
| WSS hijack via same-origin browser context | Information Disclosure | `connect-src` + Phase 88 Origin allowlist (defense in depth) |

## Open Questions / Risks for the Planner

1. **HTML page `Cache-Control: no-store`** (Pitfall 3). CONTEXT.md D-16 specifies `no-store` for `/assets/*` but is silent on the HTML routes. Recommend adding `Cache-Control: no-store` to the CSP middleware (two-line change) to keep HTML in sync with the extracted assets. If a future deployment wants HTML caching, it's one line to revert. **Action:** Planner should decide — defaulting to "no-store everywhere during v3.1" is low-risk.

2. **CI Chromium installation** (Q1). If CI runs `go test -tags=e2e ./...`, the workflow must `apt-get install -y chromium-browser` (Linux) or use `chromedp/headless-shell` Docker. If CI does NOT run e2e tests and relies on manual UAT, the plan must spell that out in `89-HUMAN-UAT.md`. Recommend local-only + manual Safari UAT to keep CI fast; the source-grep + integration-header tests give strong automated coverage already.

3. **Embed layout** (Q4). Three viable layouts (nested sub, flat, wrapping fs.FS). Plan should pick one explicitly. Recommend Option 3 (`web/assets/` subtree, `fs.Sub(WebFS, "assets")`) for simplest mental model. CONTEXT mentioned top-level placement as "Recommended" but not locked.

4. **WebSocket route and CSP middleware composition order.** The terminal page (`/sessions/{id}`) is capability-gated; the CSP header should be sent whether or not the cap is valid (else the 401 page is served without CSP, which is slightly inconsistent). Current middleware stack for terminal:
   ```
   cspHeaders → requireCapability → handleTerminalPage
   ```
   This means CSP header runs FIRST (outermost), so even on 401 rejection the CSP is set. ✓
   However, `http.Error(w, "capability required", 401)` in `requireCapability` WRITES HEADERS before returning — this is compatible because `cspHeaders` already called `w.Header().Set(...)` before delegating. Verify in code: `http.Error` internally calls `w.WriteHeader(code)` which flushes the header map, but only headers set before `WriteHeader` are emitted. So ordering is correct: set CSP → delegate → inner handler writes status. ✓

5. **Should `/api/sessions/{id}/info` and `/api/sessions` also get CSP?** D-12 says "API/JSON routes do NOT get a CSP." Correct — those are JSON payloads, CSP is irrelevant for non-HTML MIME types. No change needed.

6. **Order of CSP directive tokens.** No functional difference across browsers, but `default-src 'none'` first by convention (it's the restrictive base that everything else expands on). The sketch in Q5 follows this order.

7. **Does `'self'` on `script-src` + `style-src` cover `/assets/` URLs?** YES. `'self'` matches the page origin regardless of path — `/assets/terminal.js` is same-origin with `/sessions/{id}`, so `script-src 'self'` permits it. Same for CSS. Verified by spec.

---

## Sources

### Primary (HIGH confidence)
- **Context7 `/chromedp/chromedp`** — Verified ListenTarget + Evaluate + ExecAllocator patterns
- **Go stdlib docs** (`go doc net/http.FileServerFS`, `go doc io/fs.Sub`) — stdlib signatures confirmed
- **Repo source read** — `internal/webserver/server.go`, `origin_mw.go`, `security_regression_test.go`, `capability_mw.go`, `capability_test_helpers.go`, `assets_prod.go`, `web/*.html`, `frontend/pnpm-lock.yaml`, `.gitignore`, `go.mod` — all paths verified 2026-04-22
- **MDN `connect-src`** — https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/connect-src — explicit warning on `'self'`-vs-wss:// incompatibility
- **MDN `Sources`** (CSP) — https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/Sources — spec behavior for secure upgrades

### Secondary (MEDIUM confidence)
- WebKit bug tracker — 201591, 235873, changeset 292266 (fix exists but Safari shipping version not explicitly named)
- pkg.go.dev `github.com/chromedp/chromedp` — license + maintenance status
- MDN `SecurityPolicyViolationEvent` — DOM event API, widely cited

### Tertiary (LOW confidence)
- Stack Overflow / blog posts (2024-2025) recommending explicit `wss://<host>` for Safari — confirmed only that the broader ecosystem still treats this as unreliable, which supports keeping D-09 defensive

## Metadata

**Confidence breakdown:**
- Vendoring mechanics: HIGH — verified by direct lockfile + filesystem inspection
- FileServerFS + fs.Sub: HIGH — stdlib, existing repo precedent
- CSP middleware composition: HIGH — mirrors Phase 88 verbatim
- Inline extraction safety: HIGH — every HTML file inspected line-by-line
- chromedp viability: HIGH — license + maintenance + API all verified
- Safari CSP behavior: MEDIUM — WebKit fix merged but exact Safari version unnamed; defensive stance is sound
- Source-grep regression test: HIGH — pattern established by Phase 87/88

**Research date:** 2026-04-22
**Valid until:** 2026-05-22 (30 days — stable stack, CSP spec unlikely to change)
