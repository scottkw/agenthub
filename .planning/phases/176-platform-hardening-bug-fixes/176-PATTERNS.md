# Phase 176: Platform & Hardening Bug Fixes - Pattern Map

**Mapped:** 2026-07-08
**Files analyzed:** 5 (3 modified source files + 2 likely new/extended test files)
**Analogs found:** 5 / 5

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `main.go` (`appMenu()`, `runGUI()`) | config/bootstrap | event-driven (menu callbacks, process env setup) | `main.go` itself (no prior platform guard exists — self is the only structural analog) | partial (first `runtime.GOOS` guard in this file) |
| `internal/webserver/server.go` (`/app/` route registration, ~line 1034) | route | request-response | `internal/webserver/server.go` lines 869 (`/dashboard`), 874 (`/join`), 979 (`/sessions/{id}`) — same file, sibling routes | exact |
| `internal/webserver/csp_integration_test.go` (extend or add `TestCSPHeaderStrict_App`) | test | request-response | `internal/webserver/csp_integration_test.go` (`TestCSPHeaderStrict_Dashboard`, lines 103-115) | exact |
| `frontend/src/components/Hub/MiniPreview.tsx` (only if BUG-07 reproduces) | component | transform (styled-run → DOM) | itself — no code change unless true root cause is confirmed live; see D-01/D-02 | n/a (conditional) |
| `frontend/src/components/Hub/MiniPreview.test.tsx` (extend, only if BUG-07 fix lands) | test | transform | `frontend/src/components/Hub/MiniPreview.test.tsx` (existing `StyledSpan[][] rendering` describe block, lines 117-169) | exact |

## Pattern Assignments

### `main.go` — BUG-05 darwin-guard + DMABUF env guard

**Analog:** same file, no cross-file analog exists (first platform guard). Use the existing `appMenu()` structure and `os` import idiom already present.

**Current unconditional role-menu block** (`main.go:104-116`):
```go
func appMenu() *menu.Menu {
	m := menu.NewMenu()
	// 1. AppMenu MUST be first on macOS (STATE.md pitfall)
	m.Append(menu.AppMenu())
	// 2. File menu (custom — FileMenuRole is commented out in v2.10.2)
	fileMenu := m.AddSubmenu("File")
	fileMenu.AddText("New Session", keys.CmdOrCtrl("n"), nil)
	fileMenu.AddSeparator()
	fileMenu.AddText("Close Tab", keys.CmdOrCtrl("w"), nil)
	// 3. EditMenu — enables Cmd+C/V/X/Z via native NSMenu (MENU-02)
	m.Append(menu.EditMenu())
	// 4. Window menu
	m.Append(menu.WindowMenu())
	// 5. Help menu (custom — HelpSubMenuRole is commented out in v2.10.2)
	helpMenu := m.AddSubmenu("Help")
	helpMenu.AddText("AgentHub on GitHub", nil, openGitHubCallback)
	helpMenu.AddText("Check for Updates", nil, checkForUpdatesCallback)
	return m
}
```
Per D-08: wrap `menu.AppMenu()`, `menu.EditMenu()`, `menu.WindowMenu()` each in `if runtime.GOOS == "darwin" { ... }` (stdlib `runtime`, aliased to avoid collision with the already-imported `github.com/wailsapp/wails/v2/pkg/runtime` — e.g. `import goruntime "runtime"` and call `goruntime.GOOS`). File/Help submenus stay unconditional on all platforms.

**Existing import block** (`main.go:13-24`, note the Wails `runtime` alias collision to avoid):
```go
import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)
```
Add the stdlib import here (aliased), e.g. `goruntime "runtime"`.

**`runGUI()` env-guard pattern** (`main.go:65-93`) — no existing `os.LookupEnv` guard in this file to copy verbatim, but `os` is already imported and used elsewhere (`os.Args` in `main()`). Follow the same defensive style as `cspHeaders`'s fail-closed pattern (guard before side effect, don't silently skip):
```go
func runGUI() {
	daemon.AugmentServicePath() // Ensure CLIs in /usr/local/bin, Homebrew, volta, nvm are on PATH
	app := NewApp()

	err := wails.Run(&options.App{
		...
```
Per D-10: insert, before `wails.Run(...)` and guarded `if goruntime.GOOS == "linux"`, an `os.LookupEnv("WEBKIT_DISABLE_DMABUF_RENDERER")`-gated `os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")` call — only set if not already present, per D-10.

---

### `internal/webserver/server.go` — BUG-06 apply `cspHeaders` to `/app/`

**Analog:** sibling route registrations in the same file.

**Pattern already used 3x** (`server.go:869, 874, 979`):
```go
mux.HandleFunc("GET /dashboard", ws.cspHeaders(ws.handleDashboard))
mux.HandleFunc("GET /join", ws.cspHeaders(ws.handleJoin))
...
mux.HandleFunc("GET /sessions/{id}",
	ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage)))
```

**Current `/app/` registration to modify** (`server.go:1034` onward, function-literal handler — no named handler function exists, so the fix wraps the whole `func(w http.ResponseWriter, r *http.Request) {...}` literal, not a named `ws.handleXxx`):
```go
mux.HandleFunc("GET /app/", func(w http.ResponseWriter, r *http.Request) {
	appFS := ws.staticAppFS
	if appFS == nil {
		http.Error(w, "app bundle not configured", http.StatusServiceUnavailable)
		return
	}
	...
})
```
Per D-05: wrap the anonymous function literal in `ws.cspHeaders(...)`:
```go
mux.HandleFunc("GET /app/", ws.cspHeaders(func(w http.ResponseWriter, r *http.Request) {
	...
}))
```
Note the composition ordering convention from `/sessions/{id}` (`cspHeaders` OUTERMOST, wraps auth/capability middlewares) — for `/app/` there is no capability gate to nest inside, so `cspHeaders` wraps the handler literal directly.

**Middleware being reused verbatim — no changes needed** (`internal/webserver/csp_mw.go:93-127`):
```go
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
		b.WriteString("script-src 'self' 'wasm-unsafe-eval'; ")
		b.WriteString("style-src 'self' 'unsafe-inline'; ")
		b.WriteString("connect-src 'self' ")
		b.WriteString(wssOrigin)
		b.WriteString("; ")
		b.WriteString("img-src 'self' data:; ")
		b.WriteString("font-src 'self'; ")
		b.WriteString("base-uri 'none'; ")
		b.WriteString("form-action 'self'; ")
		b.WriteString("frame-ancestors 'none'")
		w.Header().Set("Content-Security-Policy", b.String())
		w.Header().Set("Cache-Control", "no-store")
		next(w, r)
	}
}
```

---

### `internal/webserver/csp_integration_test.go` — BUG-06 test coverage

**Analog:** `TestCSPHeaderStrict_Dashboard` (lines 103-115), the simplest of the three existing per-route tests (no capability token, no redirect-following complexity — `/app/` is also an open, non-redirecting GET route once a static bundle is wired).

```go
// TestCSPHeaderStrict_Dashboard asserts D-18's five assertions on GET /dashboard.
func TestCSPHeaderStrict_Dashboard(t *testing.T) {
	ws, client := testServer(t)
	resp, err := client.Get(ws.BaseURL() + "/dashboard")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /dashboard, got %d", resp.StatusCode)
	}
	assertCSPHeaderStrict(t, resp, ws.BaseURL(), "/dashboard")
}
```

**Shared assertion helper to reuse (do not duplicate)** (`csp_integration_test.go:18-69`, `assertCSPHeaderStrict`) — runs the canonical D-18 five-assertion suite (header present, required tokens, no-unsafe tokens, wss connect-src, frame-ancestors/base-uri). Call it with `"/app/"` as `routeName`.

**Caveat (D-06 in CONTEXT.md):** `/app/` returns 503 unless `ws.staticAppFS` is wired (only true under `wails build -tags "webkit2_41,wailsassets"` prod builds — the daemon test harness `testServer(t)` likely does NOT wire a static app FS by default). Check whether `testServer`/`testServerWithHub` (in `internal/webserver/*_test.go` helpers) already sets a stub `SetStaticAppFS` for `/app/`-route tests — grep for `SetStaticAppFS` in test helper files before assuming a 200. If no stub exists, the new test may need to call `ws.SetStaticAppFS(...)` with an in-memory `fstest.MapFS` (or similar) to get past the 503 guard and exercise `cspHeaders`. Alternatively, an existing test may already assert the current no-CSP-header gap or 503 status — check `internal/webserver/*_test.go` for any existing `"/app/"` route test before adding a new one (avoid duplicating coverage; D-12 in CONTEXT.md allows folding into an existing test file).

**Related "absence" test pattern to mirror inversely** (`internal/webserver/files_routes_test.go:331-340`, `TestFilesRoutes_NoCSPHeader`) — this is the mirror-image pattern (asserting NO CSP header on files routes that intentionally omit it). Useful as a structural template if the new test needs to assert transition from absent (before fix) to present (after fix), but the primary analog remains `TestCSPHeaderStrict_Dashboard`.

---

### `frontend/src/components/Hub/MiniPreview.tsx` — BUG-07 (conditional)

**No code-change analog needed unless D-01/D-02 repro confirms a true root cause.** Per D-01, the CSS fix hypothesized by the issue is already present at `frontend/src/style.css:6020-6028`:
```css
.hub-card__preview-line {
  font-family: var(--hub-font-mono);
  font-size: 10px;
  line-height: 1.3;
  color: var(--hub-preview-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
```
Current `MiniPreview.tsx` renders one `<span>` per `StyledSpan` (per-character run) inside each `.hub-card__preview-line` row (`MiniPreview.tsx:48-59`):
```tsx
row.map((span, j) => (
  <span
    key={j}
    style={{
      color: resolveColor(span.fg, theme, true),
      background: resolveColor(span.bg, theme, false),
      fontWeight: span.b ? 'bold' : undefined,
    }}
  >
    {span.c || ' '}
  </span>
))
```
This is a plain inline `<span>` (no `display` override) — default `display: inline`, so multiple spans should flow horizontally within the `nowrap` parent. If live repro shows stacking, per D-02 the true cause is likely NOT here but in a global CSS reset forcing `span { display: block }` or similar, OR the `.hub-card__preview` / `.hub-card__preview-line` parent picking up an unintended `flex-direction: column` from a shared utility class. No global `span` display-override rule was found via `grep -n "^span"` in `style.css` — if repro occurs, broaden the search to descendant selectors (e.g., `.hub-card__preview span`, a global reset stylesheet, or a CSS module leak) before touching `MiniPreview.tsx`.

### `frontend/src/components/Hub/MiniPreview.test.tsx` — BUG-07 test extension (conditional)

**Analog:** the existing `StyledSpan[][] rendering` describe block (lines 117-169), specifically `'renders hub-card__preview-line rows for each StyledSpan row'` (lines 152-161) — extend with a horizontal-layout assertion (e.g., assert computed style `display` of the row is not `block`/`column`, or assert all sibling `<span>` elements share the same `offsetTop` via a jsdom-limited proxy, or simply assert `row.length` renders as a single row without `.hub-card__preview-line` count exceeding the input row count). jsdom does not compute real layout, so any assertion of "one row not many" must be structural (DOM node count / class checks), not pixel-based — mirror the existing pattern of querying `.hub-card__preview-line` counts and `container.querySelectorAll('span')`.

```tsx
function renderPreviewStyled(lines: StyledSpan[][] | undefined, theme?: ITheme) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(React.createElement(MiniPreview as any, { lines, theme }))
  })
  return { container, root }
}
```

## Shared Patterns

### Middleware composition ordering (webserver)
**Source:** `internal/webserver/server.go` (route registrations, lines 869/874/979/1034) and `csp_mw.go:69-77` package comment (D-13: `cspHeaders` ALWAYS delegates, is outermost).
**Apply to:** the `/app/` route wrap in BUG-06.
```go
mux.HandleFunc("GET /sessions/{id}",
	ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage)))
```

### CSP assertion helper (webserver tests)
**Source:** `internal/webserver/csp_integration_test.go:18-69` (`assertCSPHeaderStrict`)
**Apply to:** any new `/app/` CSP test — call the existing helper, do not hand-roll new assertions.

### Platform guard style (Go, stdlib runtime.GOOS)
**Source:** none pre-existing in this codebase (`main.go` is the first). Follow Go idiom: `if runtime.GOOS == "darwin" { ... }` with an aliased import to avoid the Wails `runtime` package collision.
**Apply to:** both BUG-05 fix sites (`appMenu()` role-menu guards, `runGUI()` DMABUF env guard).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `main.go` platform-guard idiom | config | event-driven | First `runtime.GOOS` guard in this codebase; no prior Go file in the repo was found with an OS-conditional Wails menu/env pattern to copy verbatim — follow standard Go idiom instead (see Shared Patterns above) |
| `TESTING.md` M-NN manual checklist item (BUG-05 Linux/Wayland) | doc | n/a | Not a code file; follow the existing Section 5 M-NN entry format directly in TESTING.md (see repo convention, no separate pattern-map entry needed) |

## Metadata

**Analog search scope:** `main.go`, `internal/webserver/` (server.go, csp_mw.go, csp_mw_test.go, csp_integration_test.go, files_routes_test.go, browser_csp_e2e_test.go), `frontend/src/components/Hub/` (MiniPreview.tsx, MiniPreview.test.tsx), `frontend/src/style.css` (lines 5995-6045)
**Files scanned:** ~10
**Pattern extraction date:** 2026-07-08
</content>
