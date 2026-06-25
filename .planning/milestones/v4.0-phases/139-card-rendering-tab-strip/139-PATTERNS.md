# Phase 139: Card Rendering & Tab Strip - Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 9 new/modified files
**Analogs found:** 9 / 9

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/daemon/engine.go` (modify: add `GetSessionStyledTailLines`) | service | transform | `GetSessionTailLines` in same file (lines 555–579) | exact |
| `internal/daemon/types.go` (modify: add `StyledSpan`, `StyledTailLinesResponse`) | model | — | `TailLinesResponse` struct in same file (lines 62–66) | exact |
| `internal/daemon/api.go` (modify: add `handleGetSessionStyledTailLines` + route) | controller | request-response | `handleGetSessionTailLines` in same file (lines 618–634) | exact |
| `internal/daemon/client.go` (modify: add `GetSessionStyledTailLines`) | service | request-response | `GetSessionTailLines` in same file (lines 104–110) | exact |
| `app.go` (modify: add `GetSessionStyledTailLines` Wails binding) | controller | request-response | `GetSessionTailLines` in same file (lines 434–449) | exact |
| `internal/daemon/engine_test.go` (modify: add VT styled-tail tests) | test | — | `TestGetSessionTailLines_*` block (lines 1579–1704) | exact |
| `frontend/src/components/Hub/MiniPreview.tsx` (modify: consume `StyledSpan[][]`) | component | transform | itself — current `string[]` render (lines 1–38) | exact |
| `frontend/src/components/Hub/HubBriefingModal.tsx` (modify: replace regex tail) | component | request-response | itself — current local/remote tail branches (lines 86–150) | exact |
| `frontend/src/components/TabBar.tsx` (modify: chevrons + shrink state) | component | event-driven | itself + `TerminalPanel.tsx` ResizeObserver block (lines 678–687) | role-match |
| `frontend/src/style.css` (modify: `.tab` flex-shrink, icon-only floor, chevron rules) | config | — | itself — `.tab` block (lines 108–124) | exact |

---

## Pattern Assignments

### `internal/daemon/engine.go` — add `GetSessionStyledTailLines` (service, transform)

**Analog:** `GetSessionTailLines` in `internal/daemon/engine.go`, lines 555–579

**Imports to add** (Go side — new packages alongside existing `relay`, `strings`, `regexp`):
```go
// Add to existing import block in engine.go
import (
    // ... existing imports ...
    "fmt"
    "image/color"

    "github.com/charmbracelet/x/ansi"
    ultraviolet "github.com/charmbracelet/ultraviolet"
    xvt "github.com/charmbracelet/x/vt"
)
```

**Core pattern — copy from `GetSessionTailLines` (engine.go:555–579), replacing the regex strip:**
```go
// EXISTING (lines 555–579) — the full pattern to copy, then modify:
func (e *SessionEngine) GetSessionTailLines(id string, n int) []string {
    hub, ok := e.manager.Get(id)
    if !ok {
        return []string{} // IN-01: defensive — never nil; callers need not nil-guard
    }
    raw := hub.ScrollbackSnapshot()
    // Strip relay.MsgOutput (0x01) framing bytes
    stripped := make([]byte, 0, len(raw))
    for _, b := range raw {
        if b != relay.MsgOutput {
            stripped = append(stripped, b)
        }
    }
    // Strip ANSI escape sequences.
    text := ansiEscape.ReplaceAllString(string(stripped), "")
    lines := strings.Split(text, "\n")
    // Remove empty trailing lines.
    for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
        lines = lines[:len(lines)-1]
    }
    if len(lines) > n {
        lines = lines[len(lines)-n:]
    }
    return lines
}
```

**New method — replaces regex step with charmbracelet/x/vt emulator:**
- Keep: `hub, ok := e.manager.Get(id)` guard (line 556–558)
- Keep: `hub.ScrollbackSnapshot()` (line 560)
- Keep: framing-byte strip loop (lines 562–566)
- Replace: `ansiEscape.ReplaceAllString(...)` with `xvt.NewEmulator(cols, rows)` + `emu.Write(stripped)` + cell grid extraction
- Return `[][]StyledSpan` instead of `[]string`

**ANSI-16 escape used in existing tests (engine_test.go:1643):**
```go
content := "\x1b[32mgreen text\x1b[0m\n\x1b]0;title\x07\nplain line\n"
```
The new `GetSessionStyledTailLines` must handle this same content and return cell structs with `FG:"ansi:2"` (green) on the first line.

---

### `internal/daemon/types.go` — add `StyledSpan` and `StyledTailLinesResponse` (model)

**Analog:** `TailLinesResponse` in `internal/daemon/types.go`, lines 62–66

**Existing pattern to copy:**
```go
// TailLinesResponse is the response body for GET /sessions/{id}/tail.
// Phase 132 / CARD-07.
type TailLinesResponse struct {
    Lines []string `json:"lines"`
}
```

**New types — copy the doc+struct shape, substitute the field types:**
```go
// StyledSpan is one styled cell in the VT grid — the wire type for
// GET /sessions/{id}/styled-tail. All color values are hex strings
// ("#rrggbb") or ANSI index markers ("ansi:N") or "" for terminal default.
// Phase 139 / CARD-05.
type StyledSpan struct {
    Char string `json:"c"`            // Unicode grapheme cluster
    FG   string `json:"fg,omitempty"` // "#rrggbb", "ansi:N", or "" (default fg)
    BG   string `json:"bg,omitempty"` // "#rrggbb", "ansi:N", or "" (default bg)
    Bold bool   `json:"b,omitempty"`  // true when bold attribute set
}

// StyledTailLinesResponse is the response body for GET /sessions/{id}/styled-tail.
// Phase 139 / CARD-05.
type StyledTailLinesResponse struct {
    Lines [][]StyledSpan `json:"lines"`
}
```

**Rule:** All fields must be exported with explicit JSON tags (no `image/color.Color` on the wire — convert in engine). Wails binding introspects via reflection; unexported fields produce `Promise<any>` (Pitfall 3 in RESEARCH.md).

---

### `internal/daemon/api.go` — add `handleGetSessionStyledTailLines` + route (controller, request-response)

**Analog:** `handleGetSessionTailLines` in `internal/daemon/api.go`, lines 618–634

**Route registration pattern (copy from lines 98–105, add alongside `tail`):**
```go
// EXISTING (api.go:105):
a.mux.HandleFunc("GET /sessions/{id}/tail", a.handleGetSessionTailLines)

// NEW — add immediately after:
a.mux.HandleFunc("GET /sessions/{id}/styled-tail", a.handleGetSessionStyledTailLines)
```

**Handler pattern — copy from `handleGetSessionTailLines` (lines 613–634), substitute engine call and response type:**
```go
// EXISTING (lines 618–634) — exact shape to copy:
func (a *API) handleGetSessionTailLines(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    n := 4 // default
    if nStr := r.URL.Query().Get("n"); nStr != "" {
        if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
            n = parsed
        }
    }
    if n > 20 {
        n = 20 // mirror app.go clamp — enforce [1..20] at the daemon HTTP boundary
    }
    lines := a.engine.GetSessionTailLines(id, n)
    if lines == nil {
        lines = []string{}
    }
    writeJSON(w, http.StatusOK, TailLinesResponse{Lines: lines})
}
```

**New handler — identical shape, different engine call and response struct:**
- Copy `id := r.PathValue("id")` (same path param)
- Copy `n` parsing block with same clamp `n > 20` (same security contract)
- Replace `a.engine.GetSessionTailLines(id, n)` with `a.engine.GetSessionStyledTailLines(id, n)`
- Replace nil guard and response: `if spans == nil { spans = [][]StyledSpan{} }; writeJSON(w, http.StatusOK, StyledTailLinesResponse{Lines: spans})`

**`writeJSON` helper (api.go:462–467) — already exists, use as-is:**
```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}
```

---

### `internal/daemon/client.go` — add `GetSessionStyledTailLines` (service, request-response)

**Analog:** `GetSessionTailLines` in `internal/daemon/client.go`, lines 101–110

**Existing pattern — copy exactly:**
```go
// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer, with ANSI escape sequences and framing bytes stripped.
// Phase 132 / CARD-07.
func (c *DaemonClient) GetSessionTailLines(id string, n int) ([]string, error) {
    var resp TailLinesResponse
    if err := c.doJSON(http.MethodGet, fmt.Sprintf("/sessions/%s/tail?n=%d", id, n), nil, &resp); err != nil {
        return nil, err
    }
    return resp.Lines, nil
}
```

**New client method — copy, substitute path and response type:**
```go
// GetSessionStyledTailLines returns the last n styled-cell lines from the
// session's scrollback buffer. Phase 139 / CARD-05.
func (c *DaemonClient) GetSessionStyledTailLines(id string, n int) ([][]StyledSpan, error) {
    var resp StyledTailLinesResponse
    if err := c.doJSON(http.MethodGet, fmt.Sprintf("/sessions/%s/styled-tail?n=%d", id, n), nil, &resp); err != nil {
        return nil, err
    }
    return resp.Lines, nil
}
```

---

### `app.go` — add `GetSessionStyledTailLines` Wails binding (controller, request-response)

**Analog:** `GetSessionTailLines` in `app.go`, lines 429–449

**Existing binding — copy exactly (same nil-guard, same n-clamp, same empty-return):**
```go
// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer, with ANSI escape sequences and relay framing bytes stripped.
// Returns an empty slice if the session has no scrollback (e.g. remote sessions)
// or if the daemon is unreachable. n is clamped to [1..20] to bound response size.
// Used by the Hub mini-preview poller (CARD-07). Phase 132.
func (a *App) GetSessionTailLines(id string, n int) []string {
    if a.client == nil {
        return []string{}
    }
    if n < 1 {
        n = 1
    }
    if n > 20 {
        n = 20
    }
    lines, err := a.client.GetSessionTailLines(id, n)
    if err != nil || lines == nil {
        return []string{}
    }
    return lines
}
```

**New binding — copy, substitute return type and client call:**
- Change return type: `[]string` → `[][]daemon.StyledSpan`
- Change empty return: `return []string{}` → `return [][]daemon.StyledSpan{}`
- Change client call: `a.client.GetSessionTailLines(id, n)` → `a.client.GetSessionStyledTailLines(id, n)`
- Keep: `a.client == nil` guard, `n` clamp `[1..20]`, `err != nil || lines == nil` guard
- After adding the method, run `wails dev` to regenerate `wailsjs/go/main/App.ts` and `App.d.ts`

---

### `internal/daemon/engine_test.go` — add `TestGetSessionStyledTailLines_*` (test)

**Analog:** `TestGetSessionTailLines_*` block in `internal/daemon/engine_test.go`, lines 1579–1704

**Test helper to reuse (lines 1595–1607):**
```go
// makeTailHub creates a hub for the given session ID in the engine's manager,
// writes rawContent as PTY output (hub.Run frames it as [0x01 | payload]),
// closes the pipe, and waits for the Run goroutine to finish draining.
func makeTailHub(t *testing.T, e *SessionEngine, sessionID, rawContent string) {
    t.Helper()
    pr, pw := io.Pipe()
    hub := e.manager.Create(sessionID, pr, pw, nil)
    _, err := pw.Write([]byte(rawContent))
    if err != nil {
        t.Fatalf("makeTailHub write: %v", err)
    }
    pw.Close()
    <-hub.Done()
}
```

**Test structure to copy (lines 1609–1636 as template):**
```go
func TestGetSessionTailLines_StripsFramingBytes(t *testing.T) {
    e := NewSessionEngine()
    makeTailHub(t, e, "framing-test", "hello world\n")

    lines := e.GetSessionTailLines("framing-test", 10)
    if lines == nil {
        t.Fatal("GetSessionTailLines returned nil, expected lines")
    }
    // ... assertions ...
}
```

**New test names required (from RESEARCH.md validation table):**
- `TestGetSessionStyledTailLines_ColorBold` — CSI color sequence → correct `FG:"ansi:2"` + `Bold:true` in span
- `TestGetSessionStyledTailLines_TUI` — cursor-reposition overwrite (e.g., `\r`) → no doubled lines in result
- `TestGetSessionStyledTailLines_Unknown` — unknown session ID → `[][]StyledSpan{}` (not nil, not panic)
- `TestHandleGetSessionStyledTailLines` — HTTP GET `/sessions/{id}/styled-tail` returns 200 + valid JSON (copy from `TestAPIGetSessionStatus` at api_test.go:258–279 for the `rawGet` + JSON-decode pattern)

**API test helper pattern (api_test.go:258–279) — for the HTTP-layer test:**
```go
func TestAPIGetSessionStatus(t *testing.T) {
    _, _, socketPath := testDaemon(t)
    _, createBody := rawPost(t, socketPath, "/sessions", `{"cli":"cat","name":"st-tab","workDir":""}`)
    var cr CreateResponse
    if err := json.Unmarshal(createBody, &cr); err != nil {
        t.Fatalf("decode create response: %v", err)
    }
    t.Cleanup(func() { rawDelete(t, socketPath, fmt.Sprintf("/sessions/%s", cr.ID)) })

    status, body := rawGet(t, socketPath, fmt.Sprintf("/sessions/%s/status", cr.ID))
    if status != 200 {
        t.Errorf("GET /sessions/%s/status: want 200, got %d", cr.ID, status)
    }
    // ... decode + assert ...
}
```

---

### `frontend/src/components/Hub/MiniPreview.tsx` (component, transform)

**Analog:** itself — current `string[]` render (lines 1–38)

**Current prop and render pattern (lines 1–38) — complete file to modify:**
```tsx
/* CARD-07: mini preview is plain text snapshot — NO xterm instance; polling interval 3s shared across all cards */
import React from 'react'

export interface MiniPreviewProps {
  lines: string[] | undefined
  dimmed?: boolean
}

export function MiniPreview({ lines }: MiniPreviewProps): React.ReactElement {
  if (lines === undefined) {
    return (
      <div className="hub-card__preview hub-card__preview--loading" aria-hidden="true">
        <span className="hub-card__preview-line">Loading…</span>
      </div>
    )
  }
  if (lines.length === 0) {
    return (
      <div className="hub-card__preview hub-card__preview--empty" aria-hidden="true">
        <span className="hub-card__preview-line">No output yet</span>
      </div>
    )
  }
  return (
    <div className="hub-card__preview" aria-hidden="true">
      {lines.map((line, i) => (
        <div key={i} className="hub-card__preview-line">{line || ' '}</div>
      ))}
    </div>
  )
}
```

**What changes:**
- Import `StyledSpan` type from `../../wailsjs/go/main/App` (after `wails dev` regenerates)
- Change prop: `lines: string[] | undefined` → `lines: StyledSpan[][] | undefined`
- Add `theme: ITheme` prop (already threaded to `HubPanel` via `ITheme` — see RESEARCH.md D-03)
- Keep: `undefined` → loading div, `length === 0` → empty div, `aria-hidden="true"` on all states
- Change: inner `<div key={i} className="hub-card__preview-line">{line || ' '}</div>` becomes a row of `<span>` elements (one per `StyledSpan`), each with inline `style={{ color, background, fontWeight }}`
- Add `resolveColor` helper (see Shared Patterns below) to map `"ansi:N"` / `"#rrggbb"` / `""` through `ITheme`
- MUST NOT import or mount `Terminal` from `@xterm/xterm` (CARD-07 hard constraint)

**Call site in `SessionCard.tsx` (line 548) — prop type change flows here:**
```tsx
<MiniPreview lines={previewLines} />
```

**`usePreviewPoller` in `HubPanel.tsx` (lines 65–117) — must switch from `GetSessionTailLines` to `GetSessionStyledTailLines`:**
```tsx
// CURRENT (HubPanel.tsx:87):
GetSessionTailLines(s.id, 4).catch(() => [] as string[])

// AFTER — call new binding; type becomes Map<string, StyledSpan[][]>:
GetSessionStyledTailLines(s.id, 4).catch(() => [] as StyledSpan[][])
```
The `tails` state changes from `Map<string, string[]>` to `Map<string, StyledSpan[][]>`. The `previewTails` prop threading through `SessionCardGrid` → `SessionCard` → `MiniPreview` all change type in one atomic wave.

---

### `frontend/src/components/Hub/HubBriefingModal.tsx` (component, request-response)

**Analog:** itself — existing local/remote branches (lines 86–150)

**Local branch (lines 144–150) — current pattern:**
```tsx
// Local path: unchanged — GetSessionTailLines is fast, synchronous on Go side.
GetSessionTailLines(session.id, 20)
  .then((lines) => setTailLines(lines))
  .catch(() => setTailLines([]))
```

**Local branch after change:**
- Replace `GetSessionTailLines` import with `GetSessionStyledTailLines`
- `tailLines` state changes from `string[] | null` to `StyledSpan[][] | null`
- Render: replace `<pre>{tailLines.join('\n')}</pre>` (line 238) with structured span rows (same `resolveColor` helper as MiniPreview, same `theme` prop pattern)
- Security note (RESEARCH.md): local path renders via React children (auto-escaped) — no `dangerouslySetInnerHTML`

**Remote branch (lines 86–143) — current pattern that accumulates chunks:**
```tsx
const chunks: Uint8Array[] = []
// ... RelayClient onOutput handler:
chunks.push(data)
// ... finish():
setTailLines(extractTailLines(chunks, 20))
```

**Remote branch after change:**
- Remove `stripAnsi` function (lines 21–29) and `extractTailLines` function (lines 33–45) — both replaced
- Import `Terminal` from `@xterm/xterm` and `SerializeAddon` from `@xterm/addon-serialize` (already installed per RESEARCH.md Standard Stack)
- In `finish()`: replace `setTailLines(extractTailLines(chunks, 20))` with headless xterm render:
  ```tsx
  const term = new Terminal({ cols: 80, rows: 50, allowProposedApi: true, theme })
  const serAddon = new SerializeAddon()
  term.loadAddon(serAddon)
  const merged = mergeUint8Arrays(chunks)
  term.write(merged)
  const html = serAddon.serializeAsHTML({ scrollback: 20, includeGlobalBackground: false })
  term.dispose()
  setRemoteHtml(html) // separate state — string, not StyledSpan[][]
  ```
- Remote tail renders as: `<div className="hub-modal__tail" dangerouslySetInnerHTML={{__html: remoteHtml}} />`
- Do NOT call `term.open()` (Pitfall 5 in RESEARCH.md — headless use skips DOM attachment)

**`theme` prop threading:** `HubBriefingModal` currently does NOT accept `theme`. It needs to receive `ITheme` from its parent (`HubPanel` or the card-click handler). The analog for how `theme` is passed is `TerminalPanel` (TerminalPanel.tsx:61) which already receives `theme: ITheme` as a prop.

---

### `frontend/src/components/TabBar.tsx` (component, event-driven)

**Analog:** itself (lines 1–308) + `TerminalPanel.tsx` ResizeObserver pattern (lines 678–687)

**Current state declarations (lines 110–113) — add chevron state alongside:**
```tsx
// EXISTING:
const [editingId, setEditingId] = useState<string | null>(null)
const [editValue, setEditValue] = useState('')
const inputRef = useRef<HTMLInputElement>(null)
const [contextMenu, setContextMenu] = useState<{ tabId: string; x: number; y: number } | null>(null)

// ADD after line 113:
const listRef = useRef<HTMLDivElement>(null)
const [canScrollLeft, setCanScrollLeft] = useState(false)
const [canScrollRight, setCanScrollRight] = useState(false)
```

**ResizeObserver pattern to copy from `TerminalPanel.tsx` (lines 678–687):**
```tsx
// TerminalPanel.tsx:678-687 — the ResizeObserver setup pattern:
const ro = new ResizeObserver(() => { if (termRef.current) fitTerminal(termRef.current) })
ro.observe(container)

return () => {
  cancelled = true
  if (rafId !== undefined) cancelAnimationFrame(rafId)
  ro.disconnect()
}
```

**New `checkScroll` + `useEffect` for TabBar (after existing useEffects at lines 116–138):**
```tsx
function checkScroll() {
  const el = listRef.current
  if (!el) return
  setCanScrollLeft(el.scrollLeft > 0)
  setCanScrollRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 1)
}

useEffect(() => {
  const el = listRef.current
  if (!el) return
  el.addEventListener('scroll', checkScroll, { passive: true })
  const ro = new ResizeObserver(checkScroll)
  ro.observe(el)
  checkScroll()
  return () => {
    el.removeEventListener('scroll', checkScroll)
    ro.disconnect()
  }
}, [])
```

**JSX changes — current return (lines 173–307):**
```tsx
// EXISTING (line 174):
return (
  <div className="tab-bar">
    <div className="tab-list">
      {tabs.map((tab) => { ... })}
    </div>
    {contextMenu && ...}
  </div>
)
```

**Modified return — add `ref={listRef}` to `.tab-list`, wrap with chevron buttons:**
```tsx
return (
  <div className="tab-bar">
    {canScrollLeft && (
      <button
        className="tab-bar__chevron tab-bar__chevron--left"
        onClick={() => { listRef.current?.scrollBy({ left: -160, behavior: 'smooth' }) }}
        aria-label="Scroll tabs left"
        tabIndex={0}
      >‹</button>
    )}
    <div className="tab-list" ref={listRef}>
      {tabs.map((tab) => { /* existing — unchanged */ })}
    </div>
    {canScrollRight && (
      <button
        className="tab-bar__chevron tab-bar__chevron--right"
        onClick={() => { listRef.current?.scrollBy({ left: 160, behavior: 'smooth' }) }}
        aria-label="Scroll tabs right"
        tabIndex={0}
      >›</button>
    )}
    {contextMenu && ...}  {/* contextMenu block — unchanged */}
  </div>
)
```

**D-07 tooltip — `title` on the outer `.tab` div (not just `.tab__name`):**
Currently `title={titleText}` is on the inner `<span className="tab__name">` (line 223). Move/copy to the outer `<div className="tab ...">` (line 193) so the tab name is discoverable at icon-only floor when `.tab__name` is hidden.

**D-08 — close × and `tab__progress` already survive CSS hide of `.tab__name`:** Both `tab__close` (line 230) and `tab__progress` div (line 245) are siblings of `.tab__name`, not children. The `@container` rule that hides `.tab__name` does not affect them.

---

### `frontend/src/style.css` — `.tab` flex-shrink, icon-only floor, chevron rules (config)

**Analog:** itself — `.tab` block (lines 108–124), `.tab-bar` block (lines 82–92)

**Current `.tab` rule (lines 108–124) — the lines to change:**
```css
.tab {
  display: flex;
  align-items: center;
  padding: 0 10px;
  height: 100%;
  min-width: 80px;       /* CHANGE: new icon-only floor */
  max-width: 180px;
  cursor: pointer;
  background-color: transparent;
  border-right: 1px solid #292e42;
  color: #9aa5ce;
  font-size: 13px;
  gap: 6px;
  transition: background-color 0.1s;
  flex-shrink: 0;         /* CHANGE: allow shrink */
  position: relative;
}
```

**After changes:**
- `flex-shrink: 0` → `flex-shrink: 1`
- `min-width: 80px` → `min-width: 32px` (icon-only floor: planner may adjust after visual review)
- Add `container-type: inline-size;` to enable `@container` queries on `.tab`

**New CSS to add after existing `.tab` block:**
```css
/* D-05/D-06: hide name text at icon-only floor — CSS container query */
.tab {
  container-type: inline-size;
}
@container (max-width: 59px) {
  .tab__name {
    display: none;
  }
  .tab__rename-input {
    display: none;   /* inline rename hidden at floor; D-06: use context menu */
  }
}

/* D-07: title attribute on .tab (outer div) covers icon-only state */
/* (no CSS needed — applied in TabBar.tsx JSX) */

/* D-09: chevron buttons for overflow — appear only when canScrollLeft/Right */
.tab-bar__chevron {
  flex-shrink: 0;
  width: 24px;
  height: 100%;
  background: #16161e;   /* matches .tab-bar background */
  border: none;
  color: #9aa5ce;        /* matches inactive tab color */
  font-size: 16px;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-right: 1px solid #292e42;
}
.tab-bar__chevron--right {
  border-right: none;
  border-left: 1px solid #292e42;
}
.tab-bar__chevron:hover {
  color: #c0caf5;        /* matches active tab color */
}
```

**Preserved (do not touch):**
- `scrollbar-width: none` (style.css:102) — chevrons depend on hidden native scrollbar
- `.tab-list::-webkit-scrollbar { display: none }` (lines 104–106)
- `overflow-x: auto` on `.tab-list` (line 98) — chevron `scrollBy` drives this

---

## Shared Patterns

### `resolveColor` — ANSI/hex color mapping through `ITheme`
**Apply to:** `MiniPreview.tsx` and `HubBriefingModal.tsx` local path
**Source reference:** RESEARCH.md Pattern 2 + `@xterm/xterm/typings/xterm.d.ts` (confirmed in node_modules)

```typescript
import type { ITheme } from '@xterm/xterm'

const ANSI_THEME_KEYS: (keyof ITheme)[] = [
  'black', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'white',        // 0-7
  'brightBlack', 'brightRed', 'brightGreen', 'brightYellow',                     // 8-11
  'brightBlue', 'brightMagenta', 'brightCyan', 'brightWhite',                    // 12-15
]

function resolveColor(val: string | undefined, theme: ITheme, isFg: boolean): string | undefined {
  if (!val) return isFg ? (theme.foreground ?? undefined) : undefined
  if (val.startsWith('ansi:')) {
    const idx = parseInt(val.slice(5), 10)
    if (idx < 16) return theme[ANSI_THEME_KEYS[idx]] as string | undefined
    return theme.extendedAnsi?.[idx - 16] ?? undefined
  }
  return val // already "#rrggbb"
}
```

This function may be extracted to `frontend/src/lib/vtColor.ts` if used in both `MiniPreview` and `HubBriefingModal` (avoids duplication). The RESEARCH.md validation table references `frontend/src/lib/vtColor.test.ts` — implies a standalone lib file is expected.

### `framing-byte strip` — relay byte removal
**Apply to:** `GetSessionStyledTailLines` in `engine.go`
**Source:** `GetSessionTailLines` in `engine.go`, lines 562–566

```go
stripped := make([]byte, 0, len(raw))
for _, b := range raw {
    if b != relay.MsgOutput {
        stripped = append(stripped, b)
    }
}
```
Copy exactly — same framing byte `relay.MsgOutput` (0x01), same pre-allocation pattern.

### `n`-clamp `[1..20]` — response-size bound
**Apply to:** `handleGetSessionStyledTailLines` (api.go), `GetSessionStyledTailLines` (app.go)
**Source:** `handleGetSessionTailLines` (api.go:626–628) and `GetSessionTailLines` (app.go:438–441)

```go
// api.go layer:
if n > 20 {
    n = 20
}
// app.go layer:
if n < 1 { n = 1 }
if n > 20 { n = 20 }
```
Both clamps apply independently (defense in depth). New endpoint/binding must replicate both.

### ResizeObserver cleanup pattern
**Apply to:** `TabBar.tsx` chevron `useEffect`
**Source:** `TerminalPanel.tsx`, lines 679–687

```tsx
const ro = new ResizeObserver(checkFn)
ro.observe(el)
return () => {
  el.removeEventListener('scroll', checkFn)
  ro.disconnect()
}
```
Always call `ro.disconnect()` in cleanup. Always return a cleanup function from the `useEffect`.

### Wails binding empty-return convention
**Apply to:** `GetSessionStyledTailLines` in `app.go`
**Source:** `GetSessionTailLines` in `app.go`, lines 434–448

```go
if a.client == nil {
    return []string{} // or [][]StyledSpan{} for new binding
}
// ...
if err != nil || lines == nil {
    return []string{} // or [][]StyledSpan{}
}
```
Never return `nil` from a Wails binding — return empty slice. Mirrors the IN-01 convention from `GetSessionTailLines`.

---

## No Analog Found

All files in Phase 139 have close analogs. The `charmbracelet/x/vt` cell-extraction logic inside `GetSessionStyledTailLines` is genuinely new (no prior VT emulator usage in the codebase), but the surrounding method structure, test helpers, HTTP handler, client method, and Wails binding all follow exact analogs above.

| File | Role | Data Flow | Notes |
|------|------|-----------|-------|
| `internal/daemon/engine.go` `colorToHex()` helper | utility | transform | No prior `image/color.Color` → string conversion in codebase. Use RESEARCH.md Pattern 1 `colorToHex` function as the pattern. |
| `frontend/src/lib/vtColor.ts` (new lib file, if extracted) | utility | transform | No prior ITheme color-resolution utility in codebase. Use `resolveColor` from Shared Patterns above. |

---

## Metadata

**Analog search scope:** `internal/daemon/` (engine.go, api.go, client.go, types.go, engine_test.go, api_test.go), `app.go`, `frontend/src/components/Hub/` (MiniPreview.tsx, HubBriefingModal.tsx, HubPanel.tsx, SessionCard.tsx), `frontend/src/components/TabBar.tsx`, `frontend/src/components/TerminalPanel.tsx`, `frontend/src/style.css`
**Files scanned:** 14
**Pattern extraction date:** 2026-06-20
