# Phase 12: Tab Rename + Web Dashboard - Research

**Researched:** 2026-03-20
**Domain:** React tab interaction, Go HTTP API, vanilla-JS dashboard HTML
**Confidence:** HIGH

---

## Summary

Phase 12 has two parallel workstreams that must coordinate through a shared data contract: (1) adding right-click rename to the existing double-click rename in `TabBar.tsx`, and (2) upgrading the `/api/sessions` endpoint and `dashboard.html` so session names flow through correctly.

The Go backend already stores and exposes tab names via `tabNames map[string]string` and `RenameSession`. The gap is that `handleListSessions` returns raw `[]string` IDs instead of rich objects. The dashboard HTML's `renderSessions` function already branches on `typeof s === 'object'` and reads `s.name` and `s.cli_type` — it was written for the richer shape but the backend never sent it. Closing this gap means changing one Go handler and adding no new frontend state.

The frontend rename work requires only a context-menu addition to `TabBar.tsx`. Double-click rename is already implemented. The `onRename` prop and `handleRenameTab` callback in `App.tsx` are already wired to `RenameSession` on the Go side.

**Primary recommendation:** Change `handleListSessions` in `webserver/server.go` to return `[]SessionObject{id, name, cli_type}` (requiring a `NameResolver` callback injection), add right-click context menu to `TabBar.tsx`, and refresh `dashboard.html` styling.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| UILAY-04 | User can rename a tab by double-clicking or right-clicking the tab label | Double-click already works; right-click context menu needed in TabBar.tsx |
| UILAY-05 | Renamed tab name is used as the session name in the web dashboard | RenameSession + tabNames already exist in Go; /api/sessions must emit name field |
| WEBUI-01 | Web dashboard has an improved visual design with better styling | Pure dashboard.html CSS/HTML change; no Go backend changes needed |
| WEBUI-02 | Web dashboard displays session names (from tab renames) instead of raw session IDs | handleListSessions must return objects with name field; dashboard already reads s.name |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.x (project) | Tab bar component tree | Existing; all components use React |
| Vitest | 4.x (project) | Frontend unit tests | Existing test runner |
| Go stdlib `net/http` | go1.x (project) | API handler changes | Existing; no new dep needed |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jsdom | 29.x (project) | DOM testing in Vitest | All existing component tests use it |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Vanilla context menu div | `@radix-ui/react-context-menu` | No external dep fits the project's zero-library UI pattern; hand-rolled div is fine here |
| JSON object list in `/api/sessions` | Separate `/api/sessions/{id}/name` endpoint | Object list is simpler and matches what dashboard.html already expects |

**No new npm installs required.** All needed packages are present.

---

## Architecture Patterns

### Recommended Project Structure

No new files/folders needed. Changes touch:
```
frontend/src/components/TabBar.tsx        # right-click context menu
frontend/src/style.css                    # context menu CSS + dashboard is separate
web/dashboard.html                        # visual redesign (self-contained HTML/CSS/JS)
internal/webserver/server.go              # handleListSessions returns objects
internal/webserver/server_test.go         # update TestWebServerSessionListAPI
```

### Pattern 1: Right-click context menu in TabBar

**What:** Render a floating `div` anchored to the tab on `onContextMenu`. Show "Rename" action.
**When to use:** Right-click (contextmenu) event fires; dismiss on any outside click or Escape.
**Example:**
```tsx
// In TabBar component state
const [contextMenu, setContextMenu] = useState<{ tabId: string; x: number; y: number } | null>(null)

// On tab span
onContextMenu={(e) => {
  e.preventDefault()
  e.stopPropagation()
  setContextMenu({ tabId: tab.id, x: e.clientX, y: e.clientY })
}}

// Floating menu (rendered in tab-bar div or portal)
{contextMenu && (
  <div
    className="tab__context-menu"
    style={{ position: 'fixed', top: contextMenu.y, left: contextMenu.x }}
  >
    <button onClick={() => { startEditById(contextMenu.tabId); setContextMenu(null) }}>
      Rename
    </button>
  </div>
)}
```

**Dismiss pattern:** `useEffect` adds a `mousedown` listener on `document` when `contextMenu !== null`, clears it on outside click or Escape keydown.

### Pattern 2: `/api/sessions` returning named objects

**What:** Change `handleListSessions` to emit `[]SessionObject` where each has `id`, `name`, and `cli_type`.
**When to use:** Always — backward-compatible since dashboard.html already handles both shapes.

The webserver has no direct access to `tabNames` (owned by `App` in `app.go`). The current pattern uses constructor injection via the `relay.HubManager`. The cleanest approach is to inject a `NameResolver func(sessionID string) string` into `WebServer` at construction time — or add a `SetNameResolver` method.

**Option A (constructor injection):**
```go
type Config struct {
    BindIP      string
    Port        int
    ConfigDir   string
    NameResolver func(sessionID string) string // nil = use sessionID as name
}
```

**Option B (method on WebServer):**
```go
func (ws *WebServer) SetNameResolver(fn func(string) string) {
    ws.mu.Lock()
    ws.nameResolver = fn
    ws.mu.Unlock()
}
```

Option B is lower-blast-radius since it doesn't touch `Config`. In `App.startup` or `StartWebServer`, call `ws.SetNameResolver(func(id string) string { a.mu.RLock(); defer a.mu.RUnlock(); return a.tabNames[id] })`.

**Session object shape:**
```go
type sessionListItem struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cli_type"`
}
```

The `cli` type is stored in `pty.Session`. The webserver currently has no access to it. Two options: (a) extend `NameResolver` to return a `SessionInfo` struct, or (b) add a second `CLIResolver`. Simplest: use a single `SessionResolver func(string) (name, cli string)` callback.

### Pattern 3: Dashboard visual redesign

**What:** CSS-only overhaul of `dashboard.html`. The file is a self-contained HTML file embedded via Go's `embed.FS` in `web/embed.go`. Changes are pure HTML/CSS.
**Key improvements for WEBUI-01:**
- Status color indicators: use the same color palette as `style.css` (running=`#3b82f6`, idle=`#22c55e`, waiting=`#f59e0b`, errored=`#ef4444`)
- CLI badges: colored pill badges next to session names
- Card layout with better spacing and visual hierarchy
- Consistent dark theme matching the desktop app (`#1e1e1e` background, `#1a1b26`-family palette)

**Session status in dashboard:** The `/api/sessions` endpoint can also include a `status` field from `App.sessionStatuses`. This is optional for the redesign but easy to add with the same resolver pattern.

### Anti-Patterns to Avoid

- **Mutating `contextMenu` state inside the tab click handler:** The click event fires after mousedown; closing the menu with a mousedown listener is the correct dismiss pattern. If you close on `click` instead, you get a race where opening and closing happen on the same click.
- **Using `innerHTML` in dashboard for user content without escaping:** The existing `escHtml()` helper in dashboard.html must be used for all user-sourced content (name, cli_type). Already present.
- **Adding `session_name` as a separate API endpoint:** Unnecessary complexity. Return it inline.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Context menu positioning at screen edges | Custom edge-detection logic | Use `fixed` positioning with `clientX/Y` | The tab bar is at the top; menus open downward, edge cases rare; YAGNI |
| Context menu library | `@radix-ui/react-context-menu` | Simple `div` with `position: fixed` | Matches existing zero-dep UI pattern in project |

---

## Common Pitfalls

### Pitfall 1: Context menu stays open after tab close
**What goes wrong:** User right-clicks, tab is closed via keyboard shortcut, context menu hangs.
**Why it happens:** `contextMenu` state holds a `tabId` that no longer exists in `tabs`.
**How to avoid:** In the menu render, look up the tab: if `tabs.find(t => t.id === contextMenu.tabId)` is undefined, set `contextMenu(null)`. Or clear `contextMenu` in `handleCloseTab`.
**Warning signs:** Clicking "Rename" on a stale menu triggers `onRename` with a dead session ID.

### Pitfall 2: `handleListSessions` test breaks on type change
**What goes wrong:** `TestWebServerSessionListAPI` decodes into `[]string`, which fails when the API now returns `[]object`.
**Why it happens:** The existing test asserts the old JSON shape.
**How to avoid:** Update `TestWebServerSessionListAPI` to decode into `[]struct{ ID string }` or a map slice. Also update the `ws.EnableSession` test helper to pass a `SessionResolver` that returns the name.

### Pitfall 3: WebServer constructed before tabNames exist
**What goes wrong:** `SetNameResolver` is called after `StartWebServer` but the closure captures the map correctly since it reads under `a.mu.RLock()` at call time, not at setup time.
**Why it happens:** Confusion about closure semantics.
**How to avoid:** The resolver closure accesses `a.tabNames` at query time (inside the closure body), not at definition time. This is correct. No pitfall if written properly.

### Pitfall 4: Right-click opens native browser context menu too
**What goes wrong:** Both the custom menu AND the OS/browser context menu appear.
**Why it happens:** `e.preventDefault()` must be called on the `contextmenu` event.
**How to avoid:** Always call `e.preventDefault()` in `onContextMenu`.

### Pitfall 5: Dashboard name missing for sessions created before rename
**What goes wrong:** A session created as "claude 1" that was never renamed falls back to showing the raw UUID in the dashboard.
**Why it happens:** `tabNames[id]` is populated at `CreateSession` with the default name (e.g. "claude 1"), so the name is always present. But if the name is empty string (possible race or bug), the dashboard would show nothing.
**How to avoid:** In `handleListSessions`, fall back to session ID if name is empty: `if name == "" { name = id }`.

---

## Code Examples

### Right-click dismiss effect (verified React pattern)
```tsx
// Source: standard React event listener cleanup pattern
useEffect(() => {
  if (contextMenu === null) return
  function handleOutsideClick(e: MouseEvent) {
    setContextMenu(null)
  }
  function handleEscape(e: KeyboardEvent) {
    if (e.key === 'Escape') setContextMenu(null)
  }
  document.addEventListener('mousedown', handleOutsideClick)
  document.addEventListener('keydown', handleEscape)
  return () => {
    document.removeEventListener('mousedown', handleOutsideClick)
    document.removeEventListener('keydown', handleEscape)
  }
}, [contextMenu])
```

### Updated `/api/sessions` handler
```go
// Source: app.go pattern (tabNames access via mu.RLock)
func (ws *WebServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
    ids := ws.webEnabledSessions()
    items := make([]sessionListItem, 0, len(ids))
    for _, id := range ids {
        name, cliType := id, ""
        if ws.sessionResolver != nil {
            name, cliType = ws.sessionResolver(id)
        }
        if name == "" {
            name = id
        }
        items = append(items, sessionListItem{ID: id, Name: name, CLIType: cliType})
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items)
}
```

### Dashboard session card with status dot and CLI badge
```html
<!-- Source: existing dashboard.html renderSessions pattern, extended -->
li.innerHTML = `
  <span class="session-dot session-dot--${escHtml(status)}"></span>
  <span class="session-name">${escHtml(name)}</span>
  ${cli ? '<span class="session-badge">' + escHtml(cli) + '</span>' : ''}
  ...
`;
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `/api/sessions` returns `[]string` IDs | Returns `[]{id, name, cli_type}` objects | Phase 12 | Dashboard shows names instead of UUIDs |
| Tab rename via double-click only | Double-click + right-click context menu | Phase 12 | UILAY-04 requirement satisfied |

---

## Open Questions

1. **Should `/api/sessions` include session status (running/idle/waiting/errored)?**
   - What we know: `App.sessionStatuses` tracks per-session status; dashboard.html redesign calls for "status color indicators" per WEBUI-01
   - What's unclear: Whether WEBUI-01 requires live status polling or just static display
   - Recommendation: Include `status` in the session list API response for free (no polling infrastructure needed); dashboard can render color dots without websocket complexity

2. **SessionResolver: name only, or name+cli+status?**
   - What we know: Three separate maps in App: `tabNames`, `sessionStatuses`, and session registry for CLI type
   - What's unclear: Whether to unify into one callback or three
   - Recommendation: Single `func(id string) (name, cliType, status string)` tuple resolver — simpler injection, single mutex acquire

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (frontend) | Vitest 4.x |
| Config file | `frontend/vite.config.ts` (or inline in package.json test script) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test` |
| Go test command | `go test ./internal/webserver/...` |
| Go full command | `go test ./...` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UILAY-04 | Right-click on tab label opens rename context menu | unit (DOM) | `cd frontend && pnpm test` | ❌ Wave 0 — new test in TabBar.test.tsx |
| UILAY-04 | Double-click on tab label triggers inline rename input | unit (DOM) | `cd frontend && pnpm test` | ❌ Wave 0 — new test in TabBar.test.tsx |
| UILAY-05 | /api/sessions returns objects with name field | unit (Go) | `go test ./internal/webserver/... -run TestWebServerSessionListAPI` | existing — needs update |
| WEBUI-01 | Dashboard renders status color dots per session | manual-only | n/a | Manual: open browser to /dashboard |
| WEBUI-02 | Dashboard renders session name not raw ID | unit (Go) | `go test ./internal/webserver/... -run TestWebServerSessionListAPI` | existing — needs update |
| WEBUI-02 | Dashboard JS renderSessions uses s.name when present | unit (raw string check) | `cd frontend && pnpm test` | ❌ Wave 0 — raw-string test in App.test.tsx pattern |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test` (frontend) + `go test ./internal/webserver/... -count=1` (backend)
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test` + `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] New tests in `frontend/src/components/__tests__/TabBar.test.tsx` — UILAY-04 right-click and double-click rename
- [ ] Update `internal/webserver/server_test.go` `TestWebServerSessionListAPI` — decode `[]struct{ID,Name,CLIType string}` instead of `[]string`
- [ ] Optional: raw-string test for dashboard `renderSessions` in a new `web/dashboard_test` or verify via Go test

*(Existing test infrastructure covers all other phase requirements.)*

---

## Sources

### Primary (HIGH confidence)
- Direct code read: `frontend/src/components/TabBar.tsx` — existing rename mechanism
- Direct code read: `app.go` `RenameSession`, `CreateSession`, `tabNames` map
- Direct code read: `internal/webserver/server.go` `handleListSessions`, `webEnabledSessions`
- Direct code read: `web/dashboard.html` `renderSessions` — already handles `s.name` and `s.cli_type`
- Direct code read: `frontend/src/style.css` — color palette for dashboard consistency

### Secondary (MEDIUM confidence)
- React `onContextMenu` / `e.preventDefault()` pattern: standard DOM event API, no library needed
- Go struct JSON serialization with `encoding/json`: standard library behavior

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all existing; no new deps
- Architecture: HIGH — code read directly; all key files inspected
- Pitfalls: HIGH — identified from actual code paths (stale context menu, test type mismatch, empty name fallback)

**Research date:** 2026-03-20
**Valid until:** 2026-04-20 (stable codebase, no fast-moving deps)
