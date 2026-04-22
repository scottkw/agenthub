# Phase 89: Vendored Terminal Assets + CSP - Pattern Map

**Mapped:** 2026-04-22
**Files analyzed:** 20 (15 new, 5 modified)
**Analogs found:** 17 / 20 (3 have no analog — first-of-kind in this repo)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `web/vendor/xterm/xterm.js` | vendored-asset | static-bytes | — | none (first of kind) |
| `web/vendor/xterm/xterm.css` | vendored-asset | static-bytes | — | none (first of kind) |
| `web/vendor/xterm/addon-fit.js` | vendored-asset | static-bytes | — | none (first of kind) |
| `web/vendor/xterm/VERSION` | vendored-asset | static-bytes | — | none (first of kind) |
| `web/terminal.js` | first-party-asset | static-bytes | `web/terminal.html:67-275` (inline `<script>`) | exact (lift-and-shift) |
| `web/terminal.css` | first-party-asset | static-bytes | `web/terminal.html:8-57` (inline `<style>`) | exact (lift-and-shift) |
| `web/dashboard.js` | first-party-asset | static-bytes | `web/dashboard.html:73-105` (inline `<script>`) | exact (lift-and-shift) |
| `web/dashboard.css` | first-party-asset | static-bytes | `web/dashboard.html:7-51` (inline `<style>`) | exact (lift-and-shift) |
| `web/join.js` | first-party-asset | static-bytes | `web/join.html:156+` (inline `<script>`) | exact (lift-and-shift) |
| `web/join.css` | first-party-asset | static-bytes | `web/join.html:7-101` (inline `<style>`) | exact (lift-and-shift) |
| `internal/webserver/csp_mw.go` | middleware | request-response | `internal/webserver/origin_mw.go` | exact (BaseURL RLock + per-request header composition) |
| `internal/webserver/csp_test.go` | test-integration | request-response | `internal/webserver/origin_integration_test.go` + `origin_mw_test.go` | exact (httptest + header/status assertions) |
| `internal/webserver/vendor_drift_test.go` | test-source-parse | file-I/O | `internal/webserver/security_regression_test.go` | role-match (stdlib file read + string parsing) |
| `internal/webserver/regression_test.go` (extend existing `security_regression_test.go`) | test-source-grep | file-I/O | `internal/webserver/security_regression_test.go` | exact (same pattern, new forbidden strings) |
| `internal/webserver/browser_csp_e2e_test.go` | test-browser-e2e | event-driven | — | none (first chromedp test) |
| `web/embed.go` (modify) | embed-fs | build-time | `web/embed.go` (existing) + `assets_prod.go` (precedent for `fs.Sub`) | exact (one-line `//go:embed` extension) |
| `web/terminal.html` (modify) | html-page | static-bytes | `web/dashboard.html` (clean external references) | role-match (references external assets) |
| `web/dashboard.html` (modify) | html-page | static-bytes | — | self (extracting from itself) |
| `web/join.html` (modify) | html-page | static-bytes | — | self (extracting from itself) |
| `internal/webserver/server.go` (modify) | handler+routing | request-response | `internal/webserver/server.go:338-405` (`setupRoutes`) + `:569-578` (`handleTerminalPage`) | self (extending existing setupRoutes + wrapping 3 handlers) |

---

## Pattern Assignments

### `web/vendor/xterm/xterm.js`, `xterm.css`, `addon-fit.js`

**Role:** vendored-asset
**Classification:** new

**Closest analog:** None in this repo — first vendored third-party browser JS/CSS.

**Pattern:** Byte-for-byte copy from `frontend/node_modules/@xterm/xterm/lib/xterm.js`, `frontend/node_modules/@xterm/xterm/css/xterm.css`, and `frontend/node_modules/@xterm/addon-fit/lib/addon-fit.js`. No modifications. Total ~485 KB committed to git.

**Key things to preserve:**
- Exact file contents from `frontend/node_modules/@xterm/...` (integrity-critical; D-02 / D-04)
- File names `xterm.js`, `xterm.css`, `addon-fit.js` — route patterns depend on these
- No source-map siblings (`.js.map` files stay in node_modules per Q3 research)

**Key things to change:**
- None — this is a one-shot copy step (D-05)

---

### `web/vendor/xterm/VERSION`

**Role:** vendored-asset
**Classification:** new

**Closest analog:** None in this repo (first plaintext version manifest file of this shape).

**Pattern:** Two-line plaintext KV file. Per research Q3:
```
@xterm/xterm@6.0.0
@xterm/addon-fit@0.11.0
```

**Key things to preserve:**
- Exact resolved versions from `frontend/pnpm-lock.yaml` (the drift-test in D-04 asserts equality)
- Line-per-package format (simplest parser shape)

**Key things to change:**
- Update whenever xterm is bumped via `pnpm update` (manual two-step workflow, D-05)

---

### `web/terminal.js`

**Role:** first-party-asset
**Classification:** new

**Closest analog:** `web/terminal.html:67-275` — the existing inline `<script>` block that will be lifted out.

**Pattern to mirror:** Pure lift-and-shift. The existing IIFE self-bootstraps on parse (no `DOMContentLoaded` wrapper needed per Research Q6); the only required change is adding a `defer` OR moving the `<script>` tag to the end of body in the HTML so the DOM is ready — which it already is in the current layout (script tag is already after `<div id="terminal">`).

**Code excerpt (lines 67-100 of current `web/terminal.html`):**
```javascript
<script>
    // Binary framing protocol constants (must match relay/protocol.go)
    const MsgOutput  = 0x01;
    const MsgInput   = 0x10;
    const MsgResize2 = 0x11;
    const MsgPing    = 0x12;

    function makeFrame(type, payload) {
      const frame = new Uint8Array(1 + payload.length);
      frame[0] = type;
      frame.set(payload, 1);
      return frame;
    }

    function makeInputFrame(text) {
      const enc = new TextEncoder().encode(text);
      return makeFrame(MsgInput, enc);
    }
    // ... ~210 more lines through line 275 ...
</script>
```

**Key things to preserve:**
- All logic byte-for-byte (binary framing constants, capability URL parsing, xterm wiring, fail-safe `perms = 'read'` default, WSS connect, onData/onResize/resize handlers)
- The `(async function initTerminal() { ... })();` IIFE entrypoint — still self-bootstraps when loaded as external script
- The `const` / `var` / `let` mix as-is (the IIFE scope is preserved; top-level `const MsgOutput` etc. become module-scope globals, which is fine because there's exactly one `<script>` on the page)

**Key things to change:**
- Strip the surrounding `<script>` and `</script>` tags
- No other changes

---

### `web/terminal.css`

**Role:** first-party-asset
**Classification:** new

**Closest analog:** `web/terminal.html:8-57` — the existing inline `<style>` block.

**Pattern to mirror:** Pure lift-and-shift.

**Code excerpt (lines 8-35 of current `web/terminal.html`):**
```css
<style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    html, body { width: 100%; height: 100%; background: #1a1b26; overflow: hidden; }
    body { display: flex; flex-direction: column; }
    #web-status-bar {
      flex-shrink: 0;
      height: 32px;
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 0 12px;
      background-color: #16161e;
      border-bottom: 1px solid #292e42;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Menlo, Monaco, monospace;
      font-size: 12px;
      color: #565f89;
    }
    /* ... ~30 more lines ... */
    #terminal { flex: 1; min-height: 0; width: 100%; }
</style>
```

**Key things to preserve:**
- All selectors and declarations byte-for-byte
- Tokyo Night palette colors (`#1a1b26`, `#16161e`, `#9ece6a`, etc.)

**Key things to change:**
- Strip `<style>` / `</style>` tags

---

### `web/dashboard.js`, `web/dashboard.css`

**Role:** first-party-asset
**Classification:** new

**Closest analog:** `web/dashboard.html:73-105` (script) + `web/dashboard.html:7-51` (style).

**Pattern to mirror:** Lift-and-shift. Same shape as `terminal.js` / `terminal.css` extraction.

**Code excerpt (script IIFE, dashboard.html:73-105):**
```javascript
<script>
    (function () {
      var input = document.getElementById('code');
      if (!input) return;

      function formatCode(raw) {
        var upper = (raw || '').toUpperCase();
        var cleaned = '';
        for (var i = 0; i < upper.length && cleaned.length < 8; i++) {
          var ch = upper.charAt(i);
          if ((ch >= 'A' && ch <= 'Z') || (ch >= '2' && ch <= '7')) {
            cleaned += ch;
          }
        }
        if (cleaned.length <= 4) return cleaned;
        return cleaned.slice(0, 4) + '-' + cleaned.slice(4);
      }

      input.addEventListener('input', function () {
        var formatted = formatCode(input.value);
        if (formatted !== input.value) input.value = formatted;
      });
      // ... paste handler ...
    })();
</script>
```

**Key things to preserve:**
- The IIFE pattern (`(function () { ... })();`) — matches the page's expectation that DOM is present at parse
- Base32 alphabet filter logic (A-Z, 2-7)
- Paste handler with `e.preventDefault()` + `formatCode`

**Key things to change:**
- Strip `<script>` / `<style>` tag wrappers

---

### `web/join.js`, `web/join.css`

**Role:** first-party-asset
**Classification:** new

**Closest analog:** `web/join.html:156+` (script) + `web/join.html:7-101` (style).

**Pattern to mirror:** Lift-and-shift. The join script has more state (5 UI state variants A–E + `showState('...')` dispatcher) but the same IIFE shape.

**Code excerpt (join.html:156-200):**
```javascript
<script>
    (function () {
      function showState(id) {
        var all = document.querySelectorAll('.state');
        for (var i = 0; i < all.length; i++) all[i].classList.remove('active');
        var el = document.getElementById('state-' + id);
        if (el) el.classList.add('active');
      }

      function formatCode(raw) { /* same as dashboard */ }
      function wireCodeInput(input) { /* input + paste listeners */ }

      var params = new URLSearchParams(window.location.search);
      var code = params.get('code');
      var err = params.get('error');

      if (err === 'expired') { showState('c'); }
      else if (err === 'invalid') { showState('d'); }
      else if (err === 'session-gone') { showState('e'); }
      // ... etc
    })();
</script>
```

**Key things to preserve:**
- IIFE pattern
- State A-E dispatcher (class-toggle pattern on `.state` elements)
- Duplicated `formatCode` helper (deferred consolidation — see CONTEXT.md Deferred)

**Key things to change:**
- Strip wrappers

---

### `internal/webserver/csp_mw.go`

**Role:** middleware
**Classification:** new

**Closest analog:** `internal/webserver/origin_mw.go` — same `func(http.HandlerFunc) http.HandlerFunc` method on `*WebServer`, same `ws.BaseURL()` RLock pattern, same fail-closed-on-empty-BaseURL semantics. Phase 88 is the direct template.

**Pattern to mirror:** Per-request header composition that reads `ws.BaseURL()` (which internally RLocks `ws.mu`), rewrites `https://` → `wss://`, and splices into a pre-built CSP string. No caching — matches Phase 88 research recommendation (Q5).

**Code excerpt — `origin_mw.go:31-51` (the middleware shape to mirror):**
```go
// requireAllowedOrigin gates the WebSocket upgrade route on an exact
// match between r.Header.Get("Origin") and ws.BaseURL(). Missing or
// mismatched Origin -> 403 "forbidden".
func (ws *WebServer) requireAllowedOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		allowed := ws.BaseURL()
		if allowed == "" || origin != allowed {
			// Pitfall 1: BaseURL() == "" means listener-not-ready — fail
			// closed, never silently pass.
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
```

**Code excerpt — `server.go:321-336` (BaseURL(), the read-side source):**
```go
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

**Key things to preserve:**
- Method-on-`*WebServer` receiver (so it can call `ws.BaseURL()` without import cycles)
- `func(http.HandlerFunc) http.HandlerFunc` signature — so it wraps mux.HandleFunc registrations the same way `requireCapability` + `requireAllowedOrigin` do
- Fail-closed when `BaseURL()` returns `""` — 500 per Research Q5 recommendation (matches Phase 88 origin_mw fail-closed stance, differs only in choosing 500 vs 403 because CSP absence is a server config failure, not a client-supplied Origin failure)
- No logging of CSP events (minimal-observability per D-11 / Phase 87 D-22 / Phase 88 D-14)

**Key things to change:**
- Instead of comparing `r.Header.Get("Origin")` against `BaseURL()`, **compose a CSP string** and call `w.Header().Set("Content-Security-Policy", cspString)` before delegating to `next`
- Use `strings.Builder` with `b.Grow(256)` for the 9-token CSP string per Research Q5 sketch
- Rewrite `https://<host>:<port>` → `wss://<host>:<port>` via `"wss://" + strings.TrimPrefix(base, "https://")`
- Always `next(w, r)` after setting the header (never short-circuit — this middleware sets a response header, it does not gate)

**Token order for the CSP string (D-09):**
```
default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self' wss://<host>; img-src 'self' data:; font-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'
```

---

### `internal/webserver/csp_test.go`

**Role:** test-integration
**Classification:** new

**Closest analog:** `internal/webserver/origin_integration_test.go` (for full httptest-server integration) + `internal/webserver/origin_mw_test.go` (for httptest.NewRecorder-only unit tests).

**Pattern to mirror:**
- For unit-level (`TestCSPHeader_Strict`): use `httptest.NewRecorder()` + the middleware applied to a stub handler, assert `rec.Header().Get("Content-Security-Policy")` contains the required tokens (and does NOT contain forbidden tokens like `'unsafe-inline'`).
- For full integration (`TestCSPHeaderStrict`): spin up a real `testServer(t)`, do `client.Get(baseURL + "/sessions/{id}?cap=...")` with valid cap, assert `resp.Header.Get("Content-Security-Policy")` on the response.
- For `TestAssetsRoute`: hit `client.Get(baseURL + "/assets/xterm/xterm.js")` and assert 200 + `Content-Type` is `text/javascript; charset=utf-8`.

**Code excerpt — `origin_mw_test.go:14-33` (unit test shape to mirror):**
```go
func TestRequireAllowedOrigin_MatchingOriginPasses(t *testing.T) {
	ws, _ := testServer(t)
	called := false
	handler := ws.requireAllowedOrigin(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", ws.BaseURL()+"/sessions/x/ws", nil)
	req.Header.Set("Origin", ws.BaseURL())
	rec := httptest.NewRecorder()
	handler(rec, req)
	if !called {
		t.Fatal("expected inner handler to be called for matching Origin")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for matching Origin, got %d", rec.Code)
	}
}
```

**Code excerpt — `origin_integration_test.go:25-54` (integration test shape to mirror, with header assertion):**
```go
func TestSecurity_WebSocketRejectsCrossSiteOrigin(t *testing.T) {
	ws, client, _, _ := testServerWithHub(t, "sess-88-xsite")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-88-xsite", "read,write")

	wsURL := ws.BaseURL() + "/sessions/sess-88-xsite/ws?cap=" + token
	req, err := http.NewRequest("GET", wsURL, nil)
	// ... set headers ...
	resp, err := client.Do(req)
	if err != nil { t.Fatalf("client.Do: %v", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for cross-site Origin, got %d", resp.StatusCode)
	}
}
```

**Key things to preserve:**
- `testServer(t)` / `testServerWithHub(t, sid)` helpers (already defined in `capability_test_helpers.go` and `server_test.go`)
- `ws.SetSigningKey(capTestKey)` + `issueCapFor(t, ws, sid, perms)` pattern for minting caps (from `capability_test.go`)
- The `capForSession` / `capTestKey` helpers live in the `webserver` internal test package; the `_test.go` external-package helpers live in `server_test.go`
- Response assertions via `resp.Header.Get(name)` (stdlib pattern) — mirrors Phase 88's status-code assertions but inspects header key instead

**Key things to change:**
- Test the **presence + content** of `Content-Security-Policy` header, not 403/401 status codes
- Assert positive tokens: `script-src 'self'`, `style-src 'self'`, `frame-ancestors 'none'`, `base-uri 'none'`, and `connect-src 'self' wss://` + host
- Assert negative tokens: no `'unsafe-inline'`, no `'unsafe-eval'`, no bare `*` (the `data:` specifier after `img-src 'self'` is allowed; the test must not false-positive on it — substring-search each forbidden token with surrounding whitespace)
- D-18 exhaustive list: test runs against all three routes `/sessions/{id}`, `/dashboard`, `/join` (three subtests or a table-driven test)
- For `TestAssetsRoute`: assert 200 + correct `Content-Type` for each of `/assets/xterm/xterm.js`, `/assets/xterm/xterm.css`, `/assets/xterm/addon-fit.js`, `/assets/terminal.js`, `/assets/terminal.css`, etc.
- Also assert `Cache-Control: no-store` on `/assets/*` responses (D-16)

---

### `internal/webserver/vendor_drift_test.go`

**Role:** test-source-parse
**Classification:** new

**Closest analog:** `internal/webserver/security_regression_test.go` — same "read a source file + string-grep for a pattern" shape. The drift test extends the pattern with cross-file comparison (pnpm-lock.yaml ↔ VERSION).

**Pattern to mirror:** `os.ReadFile` a source asset, apply a simple regex or line-scan, compare against a second source asset.

**Code excerpt — `security_regression_test.go:22-31` (file-read + string-check shape to mirror):**
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

**Key things to preserve:**
- Naked `os.ReadFile` + `t.Fatalf` on I/O error (no stdlib YAML parser needed per Research Q3)
- Actionable error message pointing at the CONTEXT decision number and the remediation (e.g., "Update `web/vendor/xterm/VERSION` or re-vendor from `frontend/node_modules/@xterm/...` — see Phase 89 D-04/D-05")
- Relative paths from the test's working dir (`internal/webserver/`): `"../../frontend/pnpm-lock.yaml"` and `"../../web/vendor/xterm/VERSION"` (Research Q3 validated these paths exist)

**Key things to change:**
- Read two files: `frontend/pnpm-lock.yaml` and `web/vendor/xterm/VERSION`
- Use the grep approach from Research Q3:
  ```go
  // Match lines like:  '@xterm/xterm@6.0.0':
  re := regexp.MustCompile(`^  '@xterm/(xterm|addon-fit)@([0-9.]+)':$`)
  ```
- Iterate resolved-versions map; compare to VERSION file's two lines
- Fail with diff-showing message: "`@xterm/xterm` drift: pnpm-lock=6.0.0, VERSION=5.9.0"

---

### `internal/webserver/regression_test.go` (extend existing `security_regression_test.go`)

**Role:** test-source-grep
**Classification:** new (either extend existing file or new file)

**Closest analog:** `internal/webserver/security_regression_test.go` — exact pattern. Phase 89 D-17 explicitly mirrors Phase 88 D-13.

**Pattern to mirror:** Walk a directory tree, read every matching file, `strings.Contains` check against a forbidden-strings list.

**Code excerpt — `security_regression_test.go:37-56` (existing shape):**
```go
func TestSecurity_WebserverOriginAllowlistWiredToBaseURL(t *testing.T) {
	data, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("ReadFile server.go: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "ws.allowedOrigins()") && !strings.Contains(src, "ws.BaseURL()") {
		t.Error(`server.go handleWSSRelay AcceptOptions must reference ws.allowedOrigins() or ws.BaseURL() — Phase 88 SC-4 positive guard`)
	}
	if !strings.Contains(src, "OriginPatterns:") {
		t.Error(`server.go must still set OriginPatterns on websocket.AcceptOptions for handleWSSRelay (belt-and-suspenders D-12)`)
	}
}
```

**Canonical walk pattern (from Research Q8 sketch — 89-RESEARCH.md:491 / 878):**
```go
err := filepath.WalkDir("../../web", func(path string, d fs.DirEntry, err error) error {
	if err != nil { return err }
	if d.IsDir() { return nil }
	ext := filepath.Ext(path)
	if ext != ".html" && ext != ".js" && ext != ".css" { return nil }
	data, err := os.ReadFile(path)
	if err != nil { return err }
	src := string(data)
	for _, forbidden := range []string{
		"cdn.jsdelivr",
		"unpkg.com",
		"://cdn.",
	} {
		if strings.Contains(src, forbidden) {
			t.Errorf("%s contains forbidden substring %q — Phase 89 D-17", path, forbidden)
		}
	}
	return nil
})
```

**Key things to preserve:**
- Actionable error message referencing D-17 and "re-vendor or remove the CDN reference"
- Skip the `vendor/xterm/*` subtree if xterm's own source contains any of the tokens (verify during implementation — xterm.js is minified but should not contain `cdn.jsdelivr` or `unpkg.com` as substrings; if it does, the walker must skip `vendor/` so the test doesn't self-fire)
- `filepath.WalkDir` (stdlib) — no third-party dep

**Key things to change:**
- New forbidden-strings list: `cdn.jsdelivr`, `unpkg.com`, `://cdn.`, `<script src="http`, `<link href="http`
- New `TestNoInlineScriptOrStyle` sub-test: walk the three HTML files and assert no `<script>` tag without a `src=` attribute (and no `<style>` tag). Regex: `<script\b[^>]*>(?!\s*</script>)` (detect script blocks with non-empty bodies) — simpler variant: assert the HTML files contain ZERO occurrences of the literal string `<script>` (inline) but MAY contain `<script src=` (external)

---

### `internal/webserver/browser_csp_e2e_test.go`

**Role:** test-browser-e2e
**Classification:** new

**Closest analog:** None — first `chromedp`-based browser test in this repo.

**Pattern to use (from Research Q1 Option B — the recommended approach):**
- `//go:build e2e` build tag at the top of the file (matches Phase 87 build-tag precedent)
- Use `chromedp.NewContext(...)` + `chromedp.Run(ctx, page.AddScriptToEvaluateOnNewDocument(listenerScript), chromedp.Navigate(url), chromedp.WaitReady(...), chromedp.Evaluate("window.__cspViolations", &violations))`
- Test against all three pages with a real `testServer(t)` (reusing the existing `testServer` helper that returns a `ws` + `client`)

**JS listener script injected into each page load (from Research Q1):**
```javascript
window.__cspViolations = [];
document.addEventListener('securitypolicyviolation', (e) => {
  window.__cspViolations.push({
    directive: e.violatedDirective,
    blockedURI: e.blockedURI,
    lineNumber: e.lineNumber,
  });
});
```

**Key things to preserve:**
- The `//go:build e2e` tag so the default `go test ./...` run stays fast
- Fail-skip pattern if Chromium is unavailable: `chromedp.Run` returns an error if no browser binary is found — catch it and `t.Skip("chromedp: Chromium unavailable")` (keeps CI jobs without Chromium green)

**Key things to change:**
- Real test assertion: `if len(violations) > 0 { t.Errorf("CSP violations: %+v", violations) }`
- Navigate to all three pages sequentially or in sub-tests (one per page) for clean failure attribution

---

### `web/embed.go` (modify)

**Role:** embed-fs
**Classification:** modify

**Closest analog:** `web/embed.go` (current) + `assets_prod.go` (the top-level file showing `fs.Sub` on `embed.FS` — the stdlib pattern Phase 89 will mirror in `setupRoutes`).

**Current code:**
```go
package web

import "embed"

//go:embed dashboard.html terminal.html join.html
var WebFS embed.FS
```

**`assets_prod.go` precedent for `fs.Sub`:**
```go
//go:build wailsassets

package main

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/dist
var embeddedAssets embed.FS

var assets, _ = fs.Sub(embeddedAssets, "frontend/dist")
```

**Key things to preserve:**
- Package `web` with `var WebFS embed.FS` as the exported handle (consumed by `internal/webserver/server.go:15` as `webfs "github.com/scottkw/agenthub/web"`)
- The existing `//go:embed` line that includes the three HTML files

**Key things to change (per D-08):**
- Extend the `//go:embed` directive(s) to include the six new first-party files and the vendor/xterm tree:
  ```go
  //go:embed dashboard.html terminal.html join.html
  //go:embed terminal.js terminal.css dashboard.js dashboard.css join.js join.css
  //go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
  var WebFS embed.FS
  ```
- Optionally: expose a helper like `func AssetsFS() fs.FS { sub, _ := fs.Sub(WebFS, "."); return sub }` if the planner chooses to centralize the sub-FS construction here rather than inline in `setupRoutes` — this is Claude's-discretion per CONTEXT D-14

---

### `web/terminal.html` (modify)

**Role:** html-page
**Classification:** modify

**Closest analog:** `web/dashboard.html` after Phase 89 extraction (a clean external-reference HTML page with no inline JS/CSS). During Phase 89 itself there is no prior-art "fully-external" HTML page — all three HTML pages are extracted in parallel.

**Current code at lines 7, 65-66 (the CDN references to remove):**
```html
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm@6/css/xterm.css">
...
<script src="https://cdn.jsdelivr.net/npm/@xterm/xterm@6/lib/xterm.js"></script>
<script src="https://cdn.jsdelivr.net/npm/@xterm/addon-fit@0.11/lib/addon-fit.js"></script>
```

**Current code at lines 8-57 (`<style>` block) and 67-275 (`<script>` block):**
Already excerpted above in `terminal.js` / `terminal.css` sections.

**Key things to preserve:**
- The `<!DOCTYPE html>` / `<meta charset>` / `<meta viewport>` / `<title>AgentHub Terminal</title>` head structure
- The body DOM: `<div id="web-status-bar">` with nested span elements, then `<div id="terminal">` (xterm mount point)
- Tag ordering: xterm lib script before addon-fit script before terminal.js (xterm must parse first so `Terminal` and `FitAddon` globals are defined)

**Key things to change (per D-07):**
- Delete lines 8-57 (inline `<style>`) — replace with `<link rel="stylesheet" href="/assets/terminal.css">` in `<head>`
- Replace line 7 (CDN CSS) with `<link rel="stylesheet" href="/assets/xterm/xterm.css">`
- Replace lines 65-66 (CDN JS) with `<script src="/assets/xterm/xterm.js"></script>` + `<script src="/assets/xterm/addon-fit.js"></script>`
- Delete lines 67-275 (inline `<script>`) — replace with `<script src="/assets/terminal.js"></script>` at the end of `<body>`

---

### `web/dashboard.html` (modify) and `web/join.html` (modify)

**Role:** html-page
**Classification:** modify

**Closest analog:** Each page is its own "self" — extracting inline to external.

**Key things to preserve:**
- All HTML structure, IDs, class names, form actions (`/join`, `/join/exchange`, `/dashboard`)
- The State A-E `<div>` structure in `join.html` (dispatched by the extracted `join.js`)

**Key things to change:**
- `dashboard.html`: delete lines 7-51 (inline `<style>`) → add `<link rel="stylesheet" href="/assets/dashboard.css">` in `<head>`. Delete lines 73-105 (inline `<script>`) → add `<script src="/assets/dashboard.js"></script>` at end of `<body>`.
- `join.html`: delete lines 7-101 (inline `<style>`) → add `<link rel="stylesheet" href="/assets/join.css">`. Delete the `<script>...</script>` block at line 156+ → add `<script src="/assets/join.js"></script>` at end of `<body>`.

---

### `internal/webserver/server.go` (modify)

**Role:** handler + routing
**Classification:** modify

**Closest analog:** itself — the existing `setupRoutes` at lines 338-405 and the existing three handlers (`handleDashboard` at 407-416, `handleJoin` at 423-431, `handleTerminalPage` at 569-578) are the exact patterns being extended.

**Code excerpt — `setupRoutes` route registration pattern (lines 366-404) showing the middleware-composition style to mirror:**
```go
// GET /dashboard — open landing page (no session list per D-17, finalized
// by Plan 06).
mux.HandleFunc("GET /dashboard", ws.handleDashboard)

// GET /join — open join-flow page (Plan 06, D-09).
mux.HandleFunc("GET /join", ws.handleJoin)

// GET /sessions/{id} — capability-gated terminal HTML page.
mux.HandleFunc("GET /sessions/{id}", ws.requireCapability(ws.handleTerminalPage))

// GET /sessions/{id}/ws — Origin allowlist + capability-gated WebSocket upgrade.
mux.HandleFunc("GET /sessions/{id}/ws",
	ws.requireAllowedOrigin(ws.requireCapability(ws.handleWSSRelay)))
```

**Code excerpt — `handleTerminalPage` (lines 569-578) showing the handler shape unchanged:**
```go
// handleTerminalPage serves the embedded terminal.html.
func (ws *WebServer) handleTerminalPage(w http.ResponseWriter, r *http.Request) {
	data, err := webfs.WebFS.ReadFile("terminal.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data) //nolint:errcheck
}
```

**Key things to preserve:**
- The three handler bodies (`handleDashboard`, `handleJoin`, `handleTerminalPage`) — unchanged, they still read from `webfs.WebFS.ReadFile` and write HTML
- The `requireCapability` wrapper on `/sessions/{id}` (SEC-03 is Phase 87; Phase 89 only adds `cspHeaders` *outside* it)
- The `requireAllowedOrigin(ws.requireCapability(...))` composition on `/sessions/{id}/ws` (Phase 88; Phase 89 does NOT touch this — WSS is not an HTML page and has no CSP)
- Route registration in `setupRoutes` using `mux.HandleFunc` (stdlib `http.ServeMux`, no router library)

**Key things to change (per D-13 and D-14):**
1. Wrap each of the three HTML route registrations with `ws.cspHeaders(...)`:
   ```go
   mux.HandleFunc("GET /dashboard", ws.cspHeaders(ws.handleDashboard))
   mux.HandleFunc("GET /join", ws.cspHeaders(ws.handleJoin))
   mux.HandleFunc("GET /sessions/{id}", ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage)))
   ```
   Note: `cspHeaders` is outermost (composition order: cspHeaders → requireCapability → handleTerminalPage). The CSP header is set before the capability check runs, so even a 401 "capability required" response carries the CSP header. This is safe and desirable.

2. Add the `/assets/` route registration (single line using stdlib `http.FileServerFS` + `fs.Sub` + `http.StripPrefix`):
   ```go
   assetsFS, _ := fs.Sub(webfs.WebFS, ".")  // or "assets" if the planner chose Option 3 layout
   mux.Handle("GET /assets/", noCache(http.StripPrefix("/assets/",
       http.FileServerFS(assetsFS))))
   ```
   Where `noCache` is a tiny `func(h http.Handler) http.Handler` that sets `Cache-Control: no-store` (D-16). Could be inline `http.HandlerFunc` or a named helper.

3. Add `"io/fs"` to the import block (for `fs.Sub`).

---

## Shared Patterns

### Middleware shape: `func(http.HandlerFunc) http.HandlerFunc` on `*WebServer`

**Source:** `internal/webserver/capability_mw.go:37`, `internal/webserver/origin_mw.go:31`
**Apply to:** `csp_mw.go` (the new `cspHeaders` middleware)

**Why this shape:**
- Method on `*WebServer` so the middleware can read server state (`ws.BaseURL()`, `ws.mu`)
- `func(http.HandlerFunc) http.HandlerFunc` composes cleanly with `mux.HandleFunc` registrations: `mux.HandleFunc("GET /x", ws.cspHeaders(ws.handler))`
- Nests with existing middleware: `ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage))` is outside-in

**Concrete excerpt (from `origin_mw.go:31-51`):**
```go
func (ws *WebServer) requireAllowedOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		allowed := ws.BaseURL()
		if allowed == "" || origin != allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
```

### BaseURL() RLock read pattern for dynamic per-request composition

**Source:** `internal/webserver/server.go:321-336`
**Apply to:** `csp_mw.go` for deriving `wss://<host>:<port>` in the `connect-src` clause

**Why per-request (not cached):** Listener port is assigned at `Start()` and can change if `Restart()` is called; BaseURL() is cheap (one RLock + Sprintf, ~1-5 µs), negligible vs response-body writes.

### Source-grep regression test pattern

**Source:** `internal/webserver/security_regression_test.go`
**Apply to:** `regression_test.go` (Phase 89 D-17 — forbidden CDN strings) and `vendor_drift_test.go` (Phase 89 D-20 — pnpm-lock vs VERSION)

**Shape:** `os.ReadFile(<relative path>)` + `strings.Contains(src, forbidden)` + `t.Errorf(...)` with actionable message referencing the Phase 89 decision number.

### httptest + testServer(t) pattern for integration tests

**Source:** `internal/webserver/server_test.go:242-316` (`TestWebServerWSS`) + `internal/webserver/origin_integration_test.go:25-54`
**Apply to:** `csp_test.go`

**Shape:**
1. `ws, client := testServer(t)` or `testServerWithHub(t, sid)` helper
2. `ws.SetSigningKey(ssExtTestKey)` if testing a capability-gated route
3. Mint a cap via `capForSession(t, ws, sid)` or `issueCapFor(t, ws, sid, perms)`
4. `client.Get(baseURL + "/<path>?cap=" + token)`
5. Assert `resp.StatusCode`, `resp.Header.Get(name)`, and/or body bytes

### Fail-closed on BaseURL() == ""

**Source:** `internal/webserver/origin_mw.go:42-47`
**Apply to:** `csp_mw.go` — if `BaseURL()` returns `""` (listener not ready, theoretically unreachable), return 500 "internal error" rather than silently serving a page without `connect-src wss://`. Matches CLAUDE.md's "Silent Fallbacks Forbidden" principle.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `web/vendor/xterm/xterm.js` + siblings | vendored-asset | static-bytes | First committed third-party browser JS/CSS in this repo. Byte-for-byte copy; no pattern needed. |
| `web/vendor/xterm/VERSION` | vendored-asset | static-bytes | First plaintext version manifest. Two-line format specified in CONTEXT D-02 / Research Q3. |
| `internal/webserver/browser_csp_e2e_test.go` | test-browser-e2e | event-driven | First `chromedp` test in repo. Pattern fully specified in Research Q1 (Option B `securitypolicyviolation` listener). |

For these, the planner should follow the concrete code examples in **89-RESEARCH.md** (Q1 for chromedp, Q3 for VERSION format, Q8 for filepath.WalkDir walker) rather than look for a codebase analog.

---

## Summary

### Mechanical work (strong analog, lift-and-shift)

These files have a direct, concrete analog and should be mostly copy-paste-with-strip-wrappers or copy-pattern-with-minor-tweaks work:

1. `web/terminal.js`, `terminal.css` — lift from `terminal.html:8-57`, `67-275`
2. `web/dashboard.js`, `dashboard.css` — lift from `dashboard.html:7-51`, `73-105`
3. `web/join.js`, `join.css` — lift from `join.html:7-101`, `156+`
4. `web/terminal.html`, `dashboard.html`, `join.html` — strip inline blocks, add `<link>` and `<script src>` references
5. `internal/webserver/csp_mw.go` — mirror `origin_mw.go` verbatim, swap the gate body for a header-set body
6. `internal/webserver/csp_test.go` — mirror `origin_mw_test.go` (unit) + `origin_integration_test.go` (integration)
7. `internal/webserver/regression_test.go` (or extend `security_regression_test.go`) — mirror existing with new forbidden-string list
8. `web/embed.go` — one-line `//go:embed` extension
9. `internal/webserver/server.go` — 3 route-wrapping lines + 1 new `mux.Handle("GET /assets/", ...)` line + `io/fs` import

### Novel design (no analog — follow Research spec)

These have no close codebase analog and should follow RESEARCH.md's concrete sketches:

1. `web/vendor/xterm/*` — byte-for-byte copy from `frontend/node_modules/@xterm/...`; format specified in D-02 and Research Q3
2. `web/vendor/xterm/VERSION` — two-line KV format specified in Research Q3
3. `internal/webserver/vendor_drift_test.go` — stdlib `os.ReadFile` + regex parse of pnpm-lock.yaml top-level keys, per Research Q3
4. `internal/webserver/browser_csp_e2e_test.go` — `//go:build e2e` + `chromedp.Run` + `page.AddScriptToEvaluateOnNewDocument(...)` with a `securitypolicyviolation` listener, per Research Q1 Option B

The CSP middleware + its tests are the conceptual centerpiece of this phase, and they have the tightest analog fit (Phase 88's `origin_mw.go` / `origin_integration_test.go`). The remaining work is mechanical file-extraction and a single stdlib `http.FileServerFS` mount.

---

## Metadata

**Analog search scope:** `internal/webserver/`, `web/`, `assets_prod.go` (top-level embed precedent), `frontend/pnpm-lock.yaml` (source-of-truth for version drift)
**Files scanned:** ~18 Go files + 3 HTML files + 1 embed.go
**Pattern extraction date:** 2026-04-22
