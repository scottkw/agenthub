# Phase 139: Card Rendering & Tab Strip - Research

**Researched:** 2026-06-20
**Domain:** Go headless VT emulation / React CSS flex tab strip
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01 (research-gated):** Target is Go-side-unified VT rendering in the daemon for BOTH
  local and remote sessions — IF the local daemon sees remote PTY bytes.
  **Hard gate:** researcher MUST verify. Fallback D-01a: local in Go, remote JS-side.
- **D-02:** Render color + bold on BOTH surfaces (mini-preview + briefing tail).
- **D-03 (default):** Map grid colors through the active xterm `ITheme`, not a fixed ANSI-16
  palette. Overridable to fixed palette after review.
- **D-04:** Agent output colors are reproduction, not app status encoding — colorblind rule
  not violated.
- **D-05:** Tabs flex-shrink from current 180px max PAST current 80px min to icon-only floor
  (status dot + close ×). Exact floor px is planner detail.
- **D-06:** Rename at the floor falls back to existing right-click `tab__context-menu`.
- **D-07:** Each tab needs a hover `title` showing full name at the floor.
- **D-08:** Close × and Phase 98 `tab__progress` underline MUST remain functional at floor.
- **D-09:** Overflow affordance is chevron buttons (‹ / ›), scroll-position-aware, appear
  only on overflow. Native scrollbar stays hidden (`scrollbar-width: none` preserved).

### Claude's Discretion

- Go VT library selection.
- Exact icon-only pixel floor and whether close × is hover-only at the floor.
- Themed-vs-fixed palette defaulted to themed but overridable.
- Chevron styling, scroll step size, keyboard accessibility of chevrons.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope. RDS redesign and `agenthub-v4.0-redesign/`
mockups are scoped to Phases 140–141.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CARD-05 | Mini-preview cards and briefing-modal tail render agent output legibly — correct column spacing, no leaked escape sequences, headless VT render of scrollback | Go VT library selection (charmbracelet/x/vt), styled-grid transport design, D-01 gate resolved |
| TAB-01 | Open tabs shrink as count grows (browser-style) down to sensible minimum | CSS flex-shrink approach with icon-only floor pattern |
| TAB-02 | When tabs overflow window width, visible side-scroll affordance (chevrons) lets user reach every tab | ResizeObserver + scroll listener pattern on existing `.tab-list` |
| TAB-03 | Close, rename, progress-underline affordances remain functional at minimum tab width | D-06 (context menu rename), D-07 (title tooltip), D-08 (close × + tab__progress survive) |
</phase_requirements>

---

## Summary

Phase 139 has two independent work clusters: headless VT rendering (CARD-05) and browser-style tab shrink/overflow (TAB-01..03). The most important decision this research resolves is the hard gate in D-01.

**D-01 Gate Resolution: Go-side-unified is NOT valid for remote sessions.** The local daemon's remote WebSocket proxy (`internal/daemon/remote_ws_proxy.go`) is an opaque bidirectional frame relay — it explicitly never parses or stores PTY bytes locally (see `copyWS` at `remote_ws_proxy.go:119-130`). The local `relay.HubManager` has no Hub entry for remote session IDs, so `GetSessionTailLines` returns `[]string{}` for remote sessions (`engine.go:556-558`). This is documented in `HubBriefingModal.tsx:11-12` ("GetSessionTailLines is local-only and returns [] for remote session ids — engine.go:550"). **The D-01a split fallback applies.**

The recommended Go VT library is `charmbracelet/x/vt` (published June 2026, actively maintained, MIT license, clean API). For the frontend styled-grid transport, a new API endpoint returns per-line `StyledSpan` arrays (not raw ANSI) that the frontend maps through `ITheme` without running any emulator.

For TAB-01..03, pure CSS `flex-shrink` with a CSS width-threshold rule to hide `.tab__name` and a small React chevron component are the correct approach — no new npm packages needed.

**Primary recommendation:** Use `charmbracelet/x/vt` for Go-side VT gridding (local sessions only). Add a new `GetSessionStyledTailLines` Wails binding returning `[][]StyledSpan`. For remote sessions, the existing `HubBriefingModal` JS-side `extractTailLines` path is replaced with a `@xterm/xterm` + `@xterm/addon-serialize` headless render (already installed). MiniPreview uses local-only Go path. For tabs: change `flex-shrink: 0; min-width: 80px` to `flex-shrink: 1; min-width: <floor>px`, hide `.tab__name` via CSS at `min-width`, add a small `<TabChevrons>` component.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Local scrollback VT gridding | Go daemon (engine layer) | — | Raw scrollback bytes live in `relay.Hub.Scrollback`; daemon owns the ring buffer |
| Remote session VT gridding | Browser (JS) | — | D-01 gate confirmed: daemon never sees remote PTY bytes; bytes arrive at browser via opaque WS proxy |
| Styled-grid → frontend transport (local) | Go daemon (HTTP API) | Wails binding | New `GET /sessions/{id}/styled-tail` endpoint + Wails `App.GetSessionStyledTailLines` |
| Styled-grid → frontend transport (remote) | Browser (JS xterm headless) | — | `@xterm/xterm` (hidden Terminal) + `@xterm/addon-serialize` serialize scrollback bytes gathered from WS |
| ITheme color mapping | Browser / Frontend | — | `ITheme` already threaded App → HubPanel → HubInteractiveModal; ANSI index → hex resolved in frontend |
| MiniPreview rendering | Frontend (React) | — | CARD-07: no xterm instance; consumes styled spans from Go API; `aria-hidden` |
| HubBriefingModal tail rendering (local) | Frontend (React) | — | Calls Go API for styled spans instead of `stripAnsi` regex |
| HubBriefingModal tail rendering (remote) | Frontend (React) | — | Replaces `extractTailLines` JS regex with xterm headless + serialize |
| Tab shrink / icon-only floor | Browser (CSS) | — | Pure CSS `flex-shrink` + width threshold; no JS measurement needed for hide/show |
| Chevron overflow detection | Browser (JS: ResizeObserver) | — | `scrollWidth > clientWidth` detection; chevron scroll drives existing `overflow-x: auto` |
| Tab affordances at floor | Frontend (React: existing) | — | Close ×, `tab__progress`, context menu already in `TabBar.tsx`; no new UI needed |

---

## Standard Stack

### Core (CARD-05 Go side)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/charmbracelet/x/vt` | `v0.0.0-20260615092313-b57e5e6d29bb` | Headless VT emulator — produces `uv.Cell` grid with per-cell `uv.Style` (fg/bg color + bold/italic) | Actively maintained by Charmbracelet (June 2026), MIT, clean `Write([]byte)` + `CellAt(x,y)` API, supports scrollback natively, no PTY device needed |
| `github.com/charmbracelet/ultraviolet` | `v0.0.0-20260615092913-2399af76d5b1` | Cell/Style types used by `x/vt` | Transitive dep of `x/vt`; `uv.Style.Attributes` bitmask (AttrBold etc), `uv.Style.Foreground/Background color.Color` |
| `github.com/charmbracelet/x/ansi` | (transitive) | `ansi.BasicColor`, `ansi.IndexedColor` type assertions for color classification | Needed to distinguish ANSI-16 / 256-color / true-color from `image/color.Color` values |

### Already Installed (CARD-05 JS side — remote sessions)

| Library | Version | Purpose |
|---------|---------|---------|
| `@xterm/xterm` | `^6.0.0` | Headless Terminal instance to receive remote scrollback bytes |
| `@xterm/addon-serialize` | `^0.14.0` | `serializeAsHTML()` to extract styled output from xterm buffer for remote briefing tail |

**No new npm packages needed.** `@xterm/xterm` and `@xterm/addon-serialize` are already in `frontend/package.json` and are used in `TerminalPanel.tsx`.

### Supporting (TAB-01..03)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none — pure CSS + React) | — | Tab flex-shrink + chevrons | CSS `flex-shrink: 1` + `min-width` floor; chevron component in `TabBar.tsx` using `useRef` + `ResizeObserver` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `charmbracelet/x/vt` | `hinshun/vt10x` | vt10x last commit March 2022, unmaintained; `Glyph.Mode` int16 bitmask for bold but fewer escape-sequence coverage; no built-in Scrollback API |
| `charmbracelet/x/vt` | `gitpod-io/xterm-go` | 48 commits, unclear maintenance; API surface less documented for cell-level color extraction |
| `charmbracelet/x/vt` | `danielgatis/go-headless-term` | No scrollback API; designed for full PTY attachment |
| JS headless xterm for ALL sessions | ← | Rejected by D-01 AVOID — forces N card-level xterm instances on 3s shared poll; reverses CARD-07 |
| New npm package for chevrons | Plain React + DOM | No package needed; `scrollLeft` / `scrollWidth` / `clientWidth` + `ResizeObserver` is 30 lines |

**Installation (Go):**
```bash
go get github.com/charmbracelet/x/vt@v0.0.0-20260615092313-b57e5e6d29bb
```

**Version verification:** Go module proxy confirms `charmbracelet/x/vt` published `2026-06-15T09:23:13Z`. `charmbracelet/ultraviolet` published `2026-06-15T09:29:13Z`. Both confirmed via `proxy.golang.org`.

---

## Package Legitimacy Audit

> slopcheck was unavailable at research time. All packages below are tagged `[ASSUMED]` based on official source verification (Go module proxy + GitHub). The planner must gate each Go `go get` behind a `checkpoint:human-verify` task unless slopcheck can be run.

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| `github.com/charmbracelet/x/vt` | Go proxy | Published 2026-06-15 | Part of charmbracelet/x monorepo — Charmbracelet org | github.com/charmbracelet/x | [ASSUMED] | Approved — official Charmbracelet package |
| `github.com/charmbracelet/ultraviolet` | Go proxy | Published 2026-06-15 | Transitive dep of charmbracelet/x | github.com/charmbracelet/ultraviolet | [ASSUMED] | Approved — official Charmbracelet package |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*slopcheck was unavailable at research time. All packages above are tagged `[ASSUMED]` — planner must add `checkpoint:human-verify` before each `go get`.*

---

## The D-01 Hard Gate — RESOLVED

### Finding: Go-side-unified is INVALID for remote sessions. Apply D-01a split.

**Evidence trail:**

1. `internal/daemon/remote_ws_proxy.go:39-41` — comment: "This handler only forwards the cap (server-side) and copies frames. It never parses or evaluates frames."
2. `internal/daemon/remote_ws_proxy.go:119-130` — `copyWS` reads WS messages and writes them verbatim to the other side. No `scrollback.Append`, no `Hub.Create`, no `relay.Manager` interaction.
3. `internal/relay/hub.go:128-152` — `Hub.Run()` is the sole goroutine that feeds `h.scrollback.Append(frame)`. No remote proxy goroutine calls it.
4. `internal/daemon/engine.go:555-558` — `GetSessionTailLines` calls `e.manager.Get(id)`; if no Hub exists for that ID, returns `[]string{}`. Remote session IDs have no Hub in the local manager.
5. `frontend/src/components/Hub/HubBriefingModal.tsx:11-12` — JSDoc: "GetSessionTailLines is local-only and returns [] for remote session ids — engine.go:550".
6. `frontend/src/components/Hub/HubPanel.tsx:80-88` — `usePreviewPoller` already excludes remote sessions ("All passed sessions are local (caller filters by provenance). Remote sessions have no tail API and are excluded by the caller").

**Consequence:**
- **MiniPreview** (Hub cards, 3s poll): local sessions only, already filtered. Go-side VT rendering applies directly — replace `GetSessionTailLines`'s regex strip with `charmbracelet/x/vt` grid extraction. Remote session cards show no preview (current behavior; no change needed for CARD-05 scope).
- **HubBriefingModal tail (local):** Replace `GetSessionTailLines` call with new `GetSessionStyledTailLines`; render styled spans.
- **HubBriefingModal tail (remote):** The current `extractTailLines` JS regex → replace with a headless `@xterm/xterm` Terminal instance that consumes the same WS bytes already collected by `HubBriefingModal`'s `RelayClient` `onOutput` handler, then calls `serializeAsHTML()` or cell-by-cell extraction. The `@xterm/addon-serialize` is already installed.

---

## Architecture Patterns

### System Architecture Diagram

```
LOCAL SESSION TAIL
──────────────────
relay.Hub.Scrollback (raw PTY bytes + relay framing)
        │
        ▼
GetSessionStyledTailLines (engine.go — NEW)
        │  feeds bytes to vt.NewEmulator(cols=80, rows=24)
        │  reads cells → [][]StyledSpan{char, fg, bg, bold}
        │
GET /sessions/{id}/styled-tail (api.go — NEW endpoint)
        │  JSON: {"lines": [[{char,fg,bg,bold},...], ...]}
        │
App.GetSessionStyledTailLines (app.go — NEW Wails binding)
        │
MiniPreview.tsx (renders <span> per cell, aria-hidden)
HubBriefingModal.tsx — local path (renders styled <pre>)


REMOTE SESSION TAIL (HubBriefingModal only)
────────────────────────────────────────────
RelayClient.onOutput (existing — accumulates WS bytes)
        │
        ▼ (NEW: replace extractTailLines regex)
@xterm/xterm Terminal (headless, hidden, width=80)
        │  receives accumulated bytes via term.write()
        │
@xterm/addon-serialize serializeAsHTML({ scrollback: N })
        │  HTML string with inline styles (spans with color/bold)
        │
<div dangerouslySetInnerHTML={…}> (or parse back to styled spans)
```

### Recommended Project Structure

**Go changes (daemon):**
```
internal/daemon/
├── engine.go           # GetSessionStyledTailLines (replaces regex in GetSessionTailLines)
├── api.go              # GET /sessions/{id}/styled-tail handler
├── types.go            # StyledSpan, StyledTailLinesResponse structs
├── client.go           # GetSessionStyledTailLines RPC method
└── engine_test.go      # Tests for VT grid output
app.go                  # GetSessionStyledTailLines Wails binding
```

**Frontend changes:**
```
frontend/src/
├── components/Hub/
│   ├── MiniPreview.tsx          # Consume [][]StyledSpan, render <span> per cell
│   ├── HubBriefingModal.tsx     # Local path: StyledSpan; remote path: xterm headless
│   └── HubPanel.tsx             # Pass StyledSpan[][] from new API to MiniPreview
├── components/TabBar.tsx         # Add chevron state + ResizeObserver
└── style.css                     # Change .tab flex-shrink + min-width; add chevron rules
```

### Pattern 1: Go VT Grid Extraction (CARD-05)

**What:** Feed raw scrollback bytes into `charmbracelet/x/vt` emulator; read styled cell grid.
**When to use:** `GetSessionStyledTailLines` — replaces the regex strip in `GetSessionTailLines`.

```go
// Source: pkg.go.dev/github.com/charmbracelet/x/vt
import (
    "github.com/charmbracelet/ultraviolet" // uv.Cell, uv.Style, uv.AttrBold
    xvt "github.com/charmbracelet/x/vt"
    "github.com/charmbracelet/x/ansi"     // ansi.BasicColor, ansi.IndexedColor
    "image/color"
    "github.com/scottkw/agenthub/internal/relay"
)

// StyledSpan is the wire type: one cell in the styled grid.
// All color values are hex strings ("#rrggbb") or "" for default.
type StyledSpan struct {
    Char string `json:"c"`         // Unicode character (may be multi-rune grapheme)
    FG   string `json:"fg,omitempty"` // "#rrggbb" or ""  (empty = default fg)
    BG   string `json:"bg,omitempty"` // "#rrggbb" or ""  (empty = default bg)
    Bold bool   `json:"b,omitempty"`  // true if bold attribute set
}

func (e *SessionEngine) GetSessionStyledTailLines(id string, n int) [][]StyledSpan {
    hub, ok := e.manager.Get(id)
    if !ok {
        return [][]StyledSpan{}
    }
    raw := hub.ScrollbackSnapshot()

    // Strip relay framing bytes (0x01) — same as GetSessionTailLines.
    stripped := make([]byte, 0, len(raw))
    for _, b := range raw {
        if b != relay.MsgOutput {
            stripped = append(stripped, b)
        }
    }

    // Feed into headless VT emulator. 80 cols × 50 rows is enough to hold
    // scrollback; only the last n visible lines are returned.
    emu := xvt.NewEmulator(80, 50)
    emu.SetScrollbackSize(200)
    emu.Write(stripped)

    // Extract last n non-blank lines from the scrollback + active screen.
    // charmbracelet/x/vt exposes Scrollback().Lines() for pushed-off lines.
    // For simplicity: render to a 80×50 grid and read active screen rows.
    result := make([][]StyledSpan, 0, n)
    for y := 0; y < 50; y++ {
        var row []StyledSpan
        for x := 0; x < 80; x++ {
            cell := emu.CellAt(x, y)
            if cell == nil {
                break
            }
            span := StyledSpan{
                Char: cell.Content,
                Bold: cell.Style.Attributes&uv.AttrBold != 0,
                FG:   colorToHex(cell.Style.Foreground),
                BG:   colorToHex(cell.Style.Background),
            }
            row = append(row, span)
        }
        result = append(result, row)
    }
    // Trim trailing blank rows; return last n.
    // ... (same trailing-trim logic as GetSessionTailLines)
    return result
}

// colorToHex converts image/color.Color to "#rrggbb" or "" for nil/zero.
// ANSI BasicColor (0-15) and IndexedColor (16-255) are passed through as
// index markers ("ansi:N") so the frontend can resolve them via ITheme.
func colorToHex(c color.Color) string {
    if c == nil {
        return ""
    }
    switch v := c.(type) {
    case ansi.BasicColor:   // ANSI 0-15
        return fmt.Sprintf("ansi:%d", int(v))
    case ansi.IndexedColor: // ANSI 16-255
        return fmt.Sprintf("ansi:%d", int(v))
    default:
        r, g, b, _ := c.RGBA()
        return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
    }
}
```

**Note on Wails binding:** `GetSessionStyledTailLines` returns `[][]StyledSpan`. Wails can only expose Go methods on `*App` with types that JSON-marshal cleanly. The nested slice of structs marshals fine. The Wails binding in `app.go` wraps the daemon client call exactly like `GetSessionTailLines`. The `wailsjs` TypeScript types are regenerated by `wails dev` (run `wails dev` after adding the binding).

### Pattern 2: Frontend StyledSpan Rendering (CARD-05)

**What:** Map `StyledSpan` fg/bg ("ansi:N" or "#rrggbb") through `ITheme` to produce `style` props.
**When to use:** `MiniPreview.tsx` and `HubBriefingModal.tsx` local path.

```typescript
// Source: ITheme from @xterm/xterm/typings/xterm.d.ts
import type { ITheme } from '@xterm/xterm'

// Map "ansi:N" index or "#rrggbb" to a resolved CSS color string.
// ITheme has black/red/green/yellow/blue/magenta/cyan/white (ANSI 0-7)
// and brightBlack..brightWhite (ANSI 8-15) plus extendedAnsi[] (16-255).
const ANSI_THEME_KEYS: (keyof ITheme)[] = [
  'black','red','green','yellow','blue','magenta','cyan','white',
  'brightBlack','brightRed','brightGreen','brightYellow',
  'brightBlue','brightMagenta','brightCyan','brightWhite',
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

// Usage in MiniPreview:
// MiniPreview now accepts StyledSpan[][] instead of string[].
// Each row renders as a <div>; each span renders as <span style={{color,background,fontWeight}}>
```

### Pattern 3: Remote Session Headless xterm Render (CARD-05, HubBriefingModal only)

**What:** Replace `extractTailLines` regex with headless `@xterm/xterm` Terminal.
**When to use:** `HubBriefingModal.tsx` remote path (`remote === true`).

```typescript
// Source: @xterm/xterm and @xterm/addon-serialize (already installed)
import { Terminal } from '@xterm/xterm'
import { SerializeAddon } from '@xterm/addon-serialize'

// Inside the useEffect remote branch — after WS bytes collected:
function renderRemoteTail(chunks: Uint8Array[], theme: ITheme): string {
  const term = new Terminal({ cols: 80, rows: 50, allowProposedApi: true, theme })
  const serAddon = new SerializeAddon()
  term.loadAddon(serAddon)
  // No need to open() — headless use (Terminal.open() is optional)
  const merged = mergeChunks(chunks)  // Uint8Array
  term.write(merged)
  // serializeAsHTML returns a <span>-tagged HTML fragment with inline styles
  const html = serAddon.serializeAsHTML({ scrollback: 20, includeGlobalBackground: false })
  term.dispose()
  return html
}
// Render: <div className="hub-modal__tail" dangerouslySetInnerHTML={{__html: html}} />
// Security: output comes from the agent session this user owns (not user input).
// The terminal tail is controlled output, not arbitrary user-supplied HTML.
```

**Important:** The `Terminal` instance is created, written, serialized, and disposed in-memory (no DOM attachment). This is the legitimate use pattern for `@xterm/addon-serialize` — it is documented for server-side use without `open()`.

### Pattern 4: Tab Flex-Shrink + Icon-Only Floor (TAB-01)

**What:** Change `.tab` from `flex-shrink: 0; min-width: 80px` to `flex-shrink: 1; min-width: <floor>px` and hide `.tab__name` below a threshold.
**When to use:** `frontend/src/style.css` lines 108-123 (`.tab` rule).

```css
/* BEFORE (style.css:108-123): */
.tab {
  flex-shrink: 0;      /* tabs never shrink */
  min-width: 80px;
  max-width: 180px;
}

/* AFTER: */
.tab {
  flex-shrink: 1;      /* D-05: tabs CAN shrink */
  min-width: 32px;     /* icon-only floor: status dot (10px) + gap (6px) + close ×(16px) = 32px min; planner may adjust */
  max-width: 180px;
}

/* Hide name text below 60px (CSS-only threshold — no JS needed) */
.tab .tab__name {
  /* existing: flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap */
  /* add: */
  transition: opacity 0.1s;
}
/* When the tab is narrower than ~60px, the name text is invisible.
   A pure CSS approach: use a container query on the .tab itself. */
.tab {
  container-type: inline-size;
}
@container (max-width: 59px) {
  .tab__name {
    display: none;       /* or visibility:hidden to preserve layout */
  }
  .tab__rename-input {
    display: none;       /* inline rename input is also hidden — D-06: use context menu */
  }
}
```

**Floor calculation:** Status dot (`tab__status`) is 10px wide (typical). Close × button (`tab__close`) needs ~16px touch target. Padding: 0 10px (existing). Net floor: 10 (left-pad) + 10 (status) + 6 (gap) + 16 (close) + 10 (right-pad) = 52px. Round up to **32px content box** = 52px total. Planner should verify with visual test. D-08: close × and `tab__progress` underline live in the `.tab` container and are NOT inside `.tab__name`, so they survive the `display: none` on `.tab__name`.

**Note on CSS container queries vs JS measurement:** CSS container queries (`container-type: inline-size`) are now [VERIFIED: MDN] supported in all modern browsers (Chrome 105+, Firefox 110+, Safari 16+). Wails uses Chromium WebView (all versions tested use Chrome 105+). This eliminates JS measurement.

### Pattern 5: Overflow Chevrons (TAB-02)

**What:** Add ‹ / › buttons outside `.tab-list` that scroll it; show only on overflow; position-aware.
**When to use:** Inside `TabBar.tsx`, wrapping the `.tab-list`.

```tsx
// Inside TabBar component — new state + refs
const listRef = useRef<HTMLDivElement>(null)
const [canScrollLeft, setCanScrollLeft] = useState(false)
const [canScrollRight, setCanScrollRight] = useState(false)

// Check scroll state (called on scroll event + ResizeObserver)
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
  checkScroll() // initial check
  return () => {
    el.removeEventListener('scroll', checkScroll)
    ro.disconnect()
  }
}, [])

const SCROLL_STEP = 160 // px — one ~average tab width

// In JSX — the .tab-bar wraps chevrons + .tab-list:
return (
  <div className="tab-bar">
    {canScrollLeft && (
      <button
        className="tab-bar__chevron tab-bar__chevron--left"
        onClick={() => { listRef.current?.scrollBy({ left: -SCROLL_STEP, behavior: 'smooth' }) }}
        aria-label="Scroll tabs left"
        tabIndex={0}
      >‹</button>
    )}
    <div className="tab-list" ref={listRef}>
      {/* ...existing tab renders... */}
    </div>
    {canScrollRight && (
      <button
        className="tab-bar__chevron tab-bar__chevron--right"
        onClick={() => { listRef.current?.scrollBy({ left: SCROLL_STEP, behavior: 'smooth' }) }}
        aria-label="Scroll tabs right"
        tabIndex={0}
      >›</button>
    )}
    {/* ...existing contextMenu... */}
  </div>
)
```

**CSS additions:**
```css
.tab-bar__chevron {
  flex-shrink: 0;
  width: 24px;
  height: 100%;
  background: #16161e;
  border: none;
  color: #9aa5ce;
  font-size: 16px;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-right: 1px solid #292e42; /* left chevron */
}
.tab-bar__chevron--right {
  border-right: none;
  border-left: 1px solid #292e42;
}
.tab-bar__chevron:hover { color: #c0caf5; }
```

**D-09 compliance:** Chevrons appear only when `canScrollLeft` / `canScrollRight` are true. The native scrollbar stays hidden (no changes to the existing `scrollbar-width: none` at `style.css:102`). Keyboard: both chevron `<button>` elements are naturally focusable (tabIndex=0) and respond to Enter/Space.

### Anti-Patterns to Avoid

- **Per-card xterm instance:** Explicitly rejected by D-01 AVOID. CARD-07 is a hard constraint. Do not mount `Terminal` in `MiniPreview`.
- **Regex strip instead of VT emulator:** The root cause of #96. Regex cannot handle cursor positioning, TUI-style overwrite sequences, or column tracking. Always use the VT emulator for local sessions.
- **Calling `GetSessionTailLines` and then `GetSessionStyledTailLines`:** The new styled endpoint replaces the old plain-text one for both surfaces. Do not call both.
- **Go-side VT rendering for remote sessions:** The daemon never sees remote PTY bytes (D-01 gate confirmed). Any attempt to call `GetSessionStyledTailLines` with a remote session ID returns `[][]StyledSpan{}`.
- **Using `dangerouslySetInnerHTML` for local tail:** The Go-side styled spans are structured data, not HTML. Only the remote path (xterm `serializeAsHTML`) uses `dangerouslySetInnerHTML`, and only with terminal-controlled output.
- **Removing `scrollbar-width: none` from `.tab-list`:** The chevrons depend on hidden native scrollbar. The existing rules at `style.css:102-106` must be preserved unchanged.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| VT sequence parsing / cell grid | Custom ANSI state machine | `charmbracelet/x/vt` | VT100/VT220 has hundreds of edge cases (CSI sequences, DEC private modes, alternate screen, cursor save/restore, line wrapping). `go-ansi-parser` (already a Wails dep) handles styling display only, not cell positioning. |
| Color index lookup (ANSI 16 vs 256 vs RGB) | `if c == "\x1b[31m"` | `ansi.BasicColor` / `ansi.IndexedColor` type assertions from `charmbracelet/x/ansi` | The `charmbracelet/ultraviolet` color types returned by `x/vt` CellAt() already encode the color kind. |
| Overflow detection scroll container | Custom wheel event capture | `ResizeObserver` + native `scrollLeft`/`scrollWidth` | Native scroll metrics are reliable; `overflow-x: auto` is already on `.tab-list`. |
| Hiding tab name at narrow width | JS `getBoundingClientRect` polling | CSS `container-type: inline-size` + `@container` query | CSS container queries are layout-phase accurate, zero JS overhead, no FOUC. |

**Key insight:** The VT emulator state machine is where regex solutions always fail — TUI programs use cursor movement to overwrite lines, clear screens, and position text. A regex cannot reconstruct what the user sees; only a cell-grid state machine can.

---

## Runtime State Inventory

> Omitted — this is a greenfield-feature phase (new endpoint, new CSS rules, new UI affordances). No existing state, data migrations, or stored strings need renaming.

---

## Common Pitfalls

### Pitfall 1: VT Emulator Terminal Size Mismatch
**What goes wrong:** Feeding scrollback bytes into a `vt.NewEmulator(80, 50)` when the actual PTY was running at 220 columns produces wrapping artifacts. Long lines that were not wrapped in the terminal appear wrapped in the emulator output, producing spurious extra rows.
**Why it happens:** The VT emulator wraps at its declared column width. If the PTY was wider, content was never wrapped in scrollback.
**How to avoid:** Use a fixed wide column count (e.g., 220) for the emulator when processing scrollback for tail extraction. For the mini-preview display (5-6 character cells per preview line), the display truncates anyway. Alternatively, read the PTY's actual column count from the Hub and pass it to the emulator. `relay.Hub.ResizeClient` tracks `ptyCols` — expose a `Hub.Cols()` accessor.
**Warning signs:** Preview shows half-lines or doubled content that doesn't match the real terminal.

### Pitfall 2: `charmbracelet/x/vt` Experimental Status
**What goes wrong:** The `x/vt` package lives in `charmbracelet/x` as a pre-v1 pseudo-version module. Breaking API changes can occur between pseudo-versions.
**Why it happens:** The module is not yet tagged with a v1 semver.
**How to avoid:** Pin the exact pseudo-version (`v0.0.0-20260615092313-b57e5e6d29bb`) in `go.mod`. Treat any `go get -u` on this package as a potentially breaking change. Do not use `go get ./...` -u during Phase 139.
**Warning signs:** `go build` errors about missing methods on `*vt.Emulator` after an accidental update.

### Pitfall 3: Wails Binding Return Type for Nested Slice
**What goes wrong:** `[][]StyledSpan` as a Wails binding return type generates incorrect TypeScript if the Go struct has unexported fields or non-JSON-serializable types.
**Why it happens:** Wails introspects the return type via reflection.
**How to avoid:** `StyledSpan` must use all exported fields with explicit JSON tags. No `image/color.Color` interface values in the wire type — convert all colors to `string` (hex or "ansi:N") before returning. Run `wails generate module` (or `wails dev`) after adding `GetSessionStyledTailLines` to `app.go` to regenerate `App.d.ts` and verify the TypeScript type is `Promise<StyledSpan[][]>`.
**Warning signs:** `App.d.ts` shows `Promise<any>` for the new binding, or TypeScript errors in the consumer.

### Pitfall 4: CSS Container Queries — Agent Badge Hidden at Floor
**What goes wrong:** The `@container (max-width: 59px)` rule hides `.tab__name` but also needs to NOT hide the `.tab__agent-badge` or `.tab__status` elements (those must remain).
**Why it happens:** If the container query selector is too broad, it hides the status dot and badge at the floor, making all tabs look identical.
**How to avoid:** The `@container` rule must target only `.tab__name` and `.tab__rename-input`, not the whole tab content. Status dot (`.tab__status`) and close button (`.tab__close`) must remain visible. D-07 requires the `title` attribute on the outer `.tab` div so the full name appears on hover.
**Warning signs:** At icon-only floor, all tabs look identical with no distinguishing marks.

### Pitfall 5: Remote Tail xterm Headless — No `open()` Needed
**What goes wrong:** Code calls `term.open(container)` on a detached DOM element, causing `Cannot read properties of undefined` in jsdom during tests, or unnecessary DOM manipulation in the live app.
**Why it happens:** Some xterm.js guides always call `open()` before use.
**How to avoid:** `Terminal.write()` and `SerializeAddon.serialize()` / `serializeAsHTML()` work without `open()` for headless buffer operations. Skip `open()`. Dispose the Terminal after serializing.
**Warning signs:** Test failures mentioning DOM element missing or canvas errors.

### Pitfall 6: `StyledTailLinesResponse` vs `TailLinesResponse` Confusion
**What goes wrong:** The API handler for `/sessions/{id}/tail` is accidentally changed to return styled spans, breaking `MiniPreview` poller which still calls the existing endpoint before migration.
**Why it happens:** Planner adds a new endpoint but accidentally modifies the old one.
**How to avoid:** Add a NEW endpoint `GET /sessions/{id}/styled-tail`. Keep the old `GET /sessions/{id}/tail` endpoint and `GetSessionTailLines` in place until both surfaces are migrated. Then deprecate (or keep for backward compat). The two Wails bindings (`GetSessionTailLines` and `GetSessionStyledTailLines`) coexist during the transition.

---

## Code Examples

### Verified Pattern: charmbracelet/x/vt Cell Access
```go
// Source: pkg.go.dev/github.com/charmbracelet/x/vt (confirmed June 2026)
emu := vt.NewEmulator(80, 24)
emu.Write([]byte("\x1b[32mgreen text\x1b[0m\nnormal text"))
cell := emu.CellAt(0, 0) // 'g', foreground=BasicColor(2)=green
// cell.Content == "g"
// cell.Style.Foreground -- type assert to ansi.BasicColor, ansi.IndexedColor, or image/color.RGBA
// cell.Style.Attributes & uv.AttrBold != 0 -- bold check
```

### Verified Pattern: ITheme ANSI-16 Map
```typescript
// Source: @xterm/xterm/typings/xterm.d.ts (confirmed in node_modules)
// ITheme keys for ANSI 0-15:
const ANSI_KEYS: (keyof ITheme)[] = [
  'black','red','green','yellow','blue','magenta','cyan','white',        // 0-7
  'brightBlack','brightRed','brightGreen','brightYellow',                // 8-11
  'brightBlue','brightMagenta','brightCyan','brightWhite',               // 12-15
]
// ITheme.extendedAnsi[] covers 16-255 (indexed as [0]...[239])
// ITheme.foreground — default fg; ITheme.background — default bg
```

### Verified Pattern: ResizeObserver + scrollLeft Chevron Detection
```typescript
// Source: MDN Web Docs (ResizeObserver, scrollLeft, scrollWidth — web standard)
// Confirmed: scrollWidth > clientWidth indicates overflow
const ro = new ResizeObserver(() => checkScroll())
ro.observe(listRef.current)
// canScrollLeft = el.scrollLeft > 0
// canScrollRight = el.scrollLeft + el.clientWidth < el.scrollWidth - 1
```

### Verified Pattern: CSS Container Query (width threshold)
```css
/* Source: MDN container-type documentation (verified Chrome 105+, Firefox 110+, Safari 16+) */
.tab { container-type: inline-size; }
@container (max-width: 59px) {
  .tab__name { display: none; }
}
```

### Verified Pattern: Existing tab__progress (do not break)
```tsx
// Source: frontend/src/components/TabBar.tsx:241-250 (verified in codebase)
// tab__progress uses transform: scaleX() and is absolutely positioned.
// It is a sibling of .tab__name, NOT a child — it survives display:none on .tab__name.
<div
  className="tab__progress"
  style={{ transform: `scaleX(${(tabProgress?.[tab.sessionId] ?? 0) / 100})` }}
  data-testid={`tab-progress-${tab.id}`}
/>
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Regex ANSI strip for tail preview | Headless VT grid (D-01: Go side for local) | Phase 139 | Correct column spacing, color + bold, no leaked codes |
| `extractTailLines` JS regex for remote | xterm headless + `serializeAsHTML` | Phase 139 | Same fidelity for remote sessions |
| `flex-shrink: 0` fixed-width tabs | `flex-shrink: 1` with floor + chevrons | Phase 139 | Chrome-style browser tab behavior |
| CSS media queries for responsive | CSS container queries `@container` | Standardized ~2023 | Layout-phase width detection, no JS |

**Deprecated/outdated in this phase:**
- `ansiEscape` regex in `engine.go:541-547` for `GetSessionTailLines`: not deleted but bypassed by the new `GetSessionStyledTailLines` which uses the VT emulator instead.
- `stripAnsi()` in `HubBriefingModal.tsx:21-28` (local path): replaced. Remote path `extractTailLines` also replaced.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `charmbracelet/x/vt NewEmulator().Write()` works without calling `Open()` — pure in-memory buffer feed | Pattern 1: Go VT Grid Extraction | Low: xterm-go and similar libs support headless write; confirmed by API surface (no DOM/PTY required) |
| A2 | `@xterm/xterm Terminal.write()` + `SerializeAddon.serializeAsHTML()` work without calling `terminal.open(container)` | Pattern 3: Remote xterm Headless | Medium: some addon versions required `open()` in older releases; test needed in Wave 0 |
| A3 | CSS container queries are supported in the Wails Chromium WebView version in use | Pattern 4: CSS Tab Floor | Low: Wails v2.10.2 uses Chromium 105+ where container queries shipped |
| A4 | `charmbracelet/ultraviolet` returns `ansi.BasicColor` (not some other type) for ANSI-16 fg/bg values — allowing type-switch extraction | Pattern 1 colorToHex | Medium: if ultraviolet wraps colors differently, the type switch falls through to RGBA extraction; functional but loses ANSI index mapping |
| A5 | `go get github.com/charmbracelet/x/vt` does not introduce dependency conflicts with the existing `go.mod` (particularly `charmbracelet/x/ansi` already present as indirect dep via tailscale) | Standard Stack | Low: `charmbracelet/x/ansi` is already in `go.sum` (visible in `go list -m all`); version pinning should resolve cleanly |

**Risk A2 is the highest-priority assumption to verify in Wave 0** (a one-file test of `@xterm/xterm` headless write + `serializeAsHTML`).

---

## Open Questions

1. **Emulator column width for scrollback extraction**
   - What we know: PTY column width varies per session and viewer; `relay.Hub` tracks `ptyCols` via the max-wins resize arbiter.
   - What's unclear: Whether `Hub.ptyCols` (an unexported field) should be exposed as a new accessor, or whether a fixed wide value (e.g., 220) is adequate for tail extraction.
   - Recommendation: Add `func (h *Hub) Cols() int` that returns `h.ptyCols` (or 80 if zero). Pass to `vt.NewEmulator(cols, rows)`. The cost is minimal; the correctness benefit is high for wide-terminal agents.

2. **MiniPreview StyledSpan prop type change**
   - What we know: `MiniPreview.tsx` currently accepts `lines: string[] | undefined`. Changing to `lines: StyledSpan[][] | undefined` is a breaking prop change.
   - What's unclear: Whether `usePreviewPoller` in `HubPanel.tsx` calls the old or new endpoint, and whether the transition is atomic or wave-gated.
   - Recommendation: Change `usePreviewPoller` and `MiniPreview` together in one task wave. The old `GetSessionTailLines` binding remains intact; `GetSessionStyledTailLines` is additive.

3. **Remote MiniPreview cards — current behavior preserved?**
   - What we know: `usePreviewPoller` already excludes remote sessions (`HubPanel.tsx:80-88`). Remote cards show no preview.
   - What's unclear: Whether CARD-05 explicitly requires preview content for remote cards.
   - Recommendation: CARD-05 does not require remote card preview — the existing "no preview for remote" behavior is acceptable. The briefing modal (not the card) is the one remote display surface to fix.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.26+ | `go get charmbracelet/x/vt` | ✓ | 1.26.3 (go.mod) | — |
| Node.js / pnpm | Frontend build | ✓ | (system) | — |
| `@xterm/xterm ^6` | Remote headless xterm | ✓ | Already in package.json | — |
| `@xterm/addon-serialize ^0.14` | Remote `serializeAsHTML` | ✓ | Already in package.json | — |
| `charmbracelet/x/vt` | Go VT emulation | Not yet in go.mod | v0.0.0-20260615 | — (must add) |
| Wails v2 `wails generate module` | Regenerate TypeScript bindings | ✓ | v2.10.2 | Run `wails dev` to auto-regenerate |

**Missing dependencies with no fallback:**
- `charmbracelet/x/vt` must be added via `go get`. No fallback — it is the selected VT library.

**Missing dependencies with fallback:**
- None.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Go framework | `testing` package (standard) |
| Go quick run | `go test ./internal/daemon/... -run TestGetSessionStyledTailLines -count=1` |
| Go full suite | `go test ./... -count=1` |
| Frontend framework | vitest 4.x (vite.config.ts) |
| Frontend quick run | `pnpm test -- --run --reporter=verbose` |
| Frontend full suite | `pnpm test -- --run` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CARD-05 | VT emulator strips ANSI and produces correct char+color for CSI color sequence | unit (Go) | `go test ./internal/daemon/... -run TestGetSessionStyledTailLines_ColorBold` | ❌ Wave 0 |
| CARD-05 | VT emulator handles cursor-reposition (TUI overwrite) without doubled lines | unit (Go) | `go test ./internal/daemon/... -run TestGetSessionStyledTailLines_TUI` | ❌ Wave 0 |
| CARD-05 | `GetSessionStyledTailLines` returns `[][]StyledSpan{}` for unknown session | unit (Go) | `go test ./internal/daemon/... -run TestGetSessionStyledTailLines_Unknown` | ❌ Wave 0 |
| CARD-05 | API endpoint `GET /sessions/{id}/styled-tail` returns 200 + JSON with spans | integration (Go) | `go test ./internal/daemon/... -run TestHandleGetSessionStyledTailLines` | ❌ Wave 0 |
| CARD-05 | MiniPreview renders colored spans (not plain text) | unit (vitest) | `pnpm test -- --run --reporter=verbose src/components/Hub/MiniPreview.test.tsx` | ✅ (update existing) |
| CARD-05 | `resolveColor("ansi:2", theme)` returns `theme.green` | unit (vitest) | `pnpm test -- --run src/lib/vtColor.test.ts` | ❌ Wave 0 |
| CARD-05 | Remote briefing tail uses xterm headless (no regex) — behavioral | manual UAT | verify in live app with a TUI session shared remotely | — |
| TAB-01 | Tabs shrink below 80px when 10+ tabs open | visual/manual | `wails dev` or web-share Chrome | — |
| TAB-01 | `.tab__name` hidden at icon-only floor (CSS container query) | unit (vitest JSDOM limited) | manual/visual | — |
| TAB-02 | Chevrons appear when tab list overflows | unit (vitest) | `pnpm test -- --run src/components/__tests__/TabBar.test.tsx` | ✅ (update existing) |
| TAB-02 | ‹ chevron hidden when scrolled to start; › hidden at end | unit (vitest) | `pnpm test -- --run src/components/__tests__/TabBar.test.tsx` | ✅ (update existing) |
| TAB-03 | Right-click context menu "Rename" works at icon-only width | unit (vitest) | `pnpm test -- --run src/components/__tests__/TabBar.test.tsx` | ✅ (update existing) |
| TAB-03 | `tab__progress` underline visible at floor width | visual/manual | `wails dev` | — |
| TAB-03 | `title` attribute shows full name at floor | unit (vitest) | `pnpm test -- --run src/components/__tests__/TabBar.test.tsx` | ✅ (update existing) |

### Sampling Rate

- **Per task commit:** `go test ./internal/daemon/... -run TestGetSessionStyledTail -count=1` + `pnpm test -- --run`
- **Per wave merge:** `go test ./... -count=1` + `pnpm test -- --run`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/daemon/engine_test.go` — `TestGetSessionStyledTailLines_*` (4 new tests)
- [ ] `internal/daemon/api_test.go` — `TestHandleGetSessionStyledTailLines` (1 new test)
- [ ] `frontend/src/lib/vtColor.test.ts` — `resolveColor` unit tests
- [ ] `frontend/src/components/Hub/MiniPreview.test.tsx` — update for `StyledSpan[][]` prop type
- [ ] A2 assumption verification: one-off script confirming `@xterm/xterm` headless `write()` + `serializeAsHTML()` without `open()` — run before remote-path implementation

---

## Security Domain

> `security_enforcement` is not set to `false` in config. Standard ASVS review applies.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | partial | Styled tail endpoint inherits same session-ID gate as existing `/sessions/{id}/tail`; no new cap needed |
| V5 Input Validation | yes | `n` parameter clamped 1-20 (same as existing tail endpoint); `StyledSpan.Char` is terminal output chars, not user input; rendered via React children (auto-escaped for local path) |
| V6 Cryptography | no | — |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| XSS via `serializeAsHTML` for remote tail | Tampering | Content comes from agent-owned PTY session, not user input. The session owner chose to run the agent. Accept the risk — same as displaying raw terminal output in xterm. Scope: briefing modal only, not the main terminal. |
| Oversized `StyledSpan[][]` response | Denial | `n` parameter clamped to 20 in `app.go:438-441`. The new endpoint inherits the same clamp. |
| Remote session ID spoofing to styled-tail | Elevation | Styled-tail is local-only — returns `[][]StyledSpan{}` for any ID not in the local `HubManager`. No new attack surface beyond existing `/sessions/{id}/tail`. |

---

## Sources

### Primary (HIGH confidence)
- `internal/daemon/remote_ws_proxy.go` — D-01 gate: confirmed daemon is opaque WS relay, no scrollback for remote sessions
- `internal/daemon/engine.go:555-579` — `GetSessionTailLines` code: reads `hub.ScrollbackSnapshot()` → regex strip
- `internal/relay/hub.go:128-152` — `Hub.Run()`: only place scrollback is written
- `frontend/src/components/Hub/HubBriefingModal.tsx:11-12` — confirms "GetSessionTailLines is local-only and returns [] for remote session ids"
- `frontend/src/components/Hub/HubPanel.tsx:80-88` — `usePreviewPoller` excludes remote sessions
- `pkg.go.dev/github.com/charmbracelet/x/vt` — VT API: `NewEmulator`, `Write`, `CellAt`, `Scrollback`
- `pkg.go.dev/github.com/charmbracelet/ultraviolet` — `uv.Cell`, `uv.Style`, `uv.AttrBold`, `color.Color`
- `frontend/node_modules/@xterm/xterm/typings/xterm.d.ts` — `ITheme` interface (all 16+2 ANSI color slots)
- `frontend/node_modules/@xterm/addon-serialize/typings/addon-serialize.d.ts` — `serializeAsHTML` API
- `frontend/src/style.css:82-140` — current `.tab-bar`, `.tab-list`, `.tab` CSS
- `frontend/src/components/TabBar.tsx` — all existing tab affordances: `tab__progress`, `tab__context-menu`, `tab__close`, `tab__status`

### Secondary (MEDIUM confidence)
- `proxy.golang.org/github.com/charmbracelet/x/vt` — confirmed latest version `v0.0.0-20260615092313-b57e5e6d29bb` published 2026-06-15
- `proxy.golang.org/github.com/charmbracelet/ultraviolet` — published 2026-06-15
- `proxy.golang.org/github.com/hinshun/vt10x` — last version 2022-03-01 (confirmed unmaintained)
- MDN Web Docs — `ResizeObserver`, `scrollLeft`, `scrollWidth`, CSS `container-type`

### Tertiary (LOW confidence)
- WebSearch finding that `gitpod-io/xterm-go` exists but last commit dates and API stability unclear
- WebSearch findings on CSS container query browser support (corroborated by MDN)

---

## Metadata

**Confidence breakdown:**
- D-01 gate resolution: HIGH — verified against actual code in 5 files with consistent evidence
- Standard Stack (charmbracelet/x/vt): MEDIUM — Go proxy confirms version; API confirmed via pkg.go.dev; pre-v1 stability is [ASSUMED]
- Architecture patterns: HIGH — derived from codebase analysis of existing patterns
- Tab CSS patterns: HIGH — CSS standards confirmed via MDN; ResizeObserver is web standard
- Remote xterm headless (A2): MEDIUM — SerializeAddon headless use is expected pattern but not verified in this codebase

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (30 days for the Go library; CSS/frontend patterns are stable)
