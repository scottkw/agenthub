# Phase 34: Terminal Fill Fix - Research

**Researched:** 2026-03-26
**Domain:** xterm.js FitAddon layout timing, PTY initial dimensions, Wails WebView rendering
**Confidence:** HIGH

## Summary

The terminal fill bug has two independent root causes. The first is a layout-timing problem: `FitAddon.fit()` is called before CSS layout has committed pixel dimensions, so it measures zero or incorrect values and sets wrong cols/rows. The second is an initial-dimensions problem: PTY sessions are always spawned at 80x24 regardless of container size, so even when the correct resize fires later the PTY's initial output was formatted for the wrong column width.

The fix requires two coordinated changes:

1. **Frontend (TerminalPanel.tsx):** Replace the direct `fit()` call in the `isActive` effect with a double-requestAnimationFrame deferral. `document.fonts.ready` alone is insufficient because font readiness does not guarantee the Wails WebView has finished its first layout pass. A double-rAF defers execution until after the browser has painted at least one frame, ensuring `getComputedStyle` returns real pixel dimensions for the now-visible container.

2. **Backend (engine.go):** The `daemon.CreateRequest` struct does not currently include cols/rows. The daemon API and engine both hardcode 80x24. To fix TERM-03, cols/rows must be threaded from the frontend (where the container pixel size is known at the time the user clicks "Create") through the Wails binding, daemon client, API handler, and down to `SessionEngine.CreateSession`.

**Primary recommendation:** Add double-rAF deferral for the initial fit AND thread initial cols/rows through the full stack from frontend to PTY spawn.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TERM-01 | Terminal fills correctly on initial load for Claude CLI sessions (no resize needed) | Double-rAF deferral in TerminalPanel + correct initial PTY dims |
| TERM-02 | Terminal fills correctly on initial load for Gemini CLI sessions (no resize needed) | Same fix as TERM-01; agent-agnostic |
| TERM-03 | PTY sessions spawn at appropriate initial dimensions instead of hardcoded 80x24 | Add cols/rows to daemon CreateRequest, thread from frontend |
| TERM-04 | Double-rAF deferral on `fit()` ensures layout is committed before terminal sizing | Replace `document.fonts.ready.then(fit)` with double-rAF in isActive effect |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @xterm/addon-fit | 0.11.0 | Measure container and call terminal.resize() | Official xterm.js addon; already installed |
| @xterm/xterm | 6.0.0 | Terminal emulator | Already in use |
| ResizeObserver | browser built-in | Fire fit() on container dimension changes | Already wired in TerminalPanel |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| requestAnimationFrame | browser built-in | Defer fit() until after layout paint | Initial fit only — not resize events |

**Installation:** No new dependencies required.

**Version verification:** `@xterm/addon-fit` 0.11.0 and `@xterm/xterm` 6.0.0 are current stable releases (verified against npm registry 2026-03-26).

## Architecture Patterns

### How FitAddon.proposeDimensions() Works

From the minified source at `frontend/node_modules/@xterm/addon-fit/lib/addon-fit.js`:

```javascript
proposeDimensions() {
  // Returns undefined if terminal element or parent is missing
  if (!this._terminal?.element?.parentElement) return;
  const dims = this._terminal._core._renderService.dimensions;
  // Returns undefined if character cell measurements are zero
  if (dims.css.cell.width === 0 || dims.css.cell.height === 0) return;
  // Reads COMPUTED style of the PARENT element for container height/width
  const parentStyle = window.getComputedStyle(this._terminal.element.parentElement);
  const height = parseInt(parentStyle.getPropertyValue('height'));
  const width = Math.max(0, parseInt(parentStyle.getPropertyValue('width')));
  // ...computes cols/rows from container size and cell size
}
```

**Key insight:** FitAddon reads `getComputedStyle` on the parent. When the parent has `display: none`, `getComputedStyle` returns `height: 0px` and `width: 0px`. When the parent switches from `display: none` to `display: flex`, there is a race: the DOM property changes immediately but the browser's layout engine hasn't yet recalculated pixel dimensions. Calling `fit()` synchronously on the same microtask as the display change will measure zero.

### Pattern 1: Double-rAF for Initial Fit

**What:** Defer `fit()` using two nested `requestAnimationFrame` calls.

**When to use:** For the initial fit when a panel becomes visible after being hidden. Not needed for ResizeObserver callbacks (those already fire after layout).

**Why double, not single rAF:**
- Single rAF fires before the browser paints; layout may not be committed yet.
- Double rAF fires in the next frame after the first paint; layout is guaranteed committed.
- The STATE.md note confirms: "double-rAF vs. single-rAF — test both in production binary, behavior differs from wails dev." The Wails WebView (Chromium-based) may need the extra frame.

**Example:**
```typescript
// Source: standard xterm.js community pattern, verified against xterm.js issues
useEffect(() => {
  if (!isActive || !containerRef.current) return

  let cancelled = false
  const fit = () => fitAddonRef.current?.fit()

  // Double-rAF: wait for the browser to paint the first frame
  // after display:none -> display:flex before measuring.
  const rafId1 = requestAnimationFrame(() => {
    const rafId2 = requestAnimationFrame(() => {
      if (!cancelled) fit()
    })
    return rafId2  // Note: inner rAF id for potential cancellation
  })

  const ro = new ResizeObserver(fit)
  ro.observe(containerRef.current)

  return () => {
    cancelled = true
    cancelAnimationFrame(rafId1)
    ro.disconnect()
  }
}, [isActive])
```

Note: The `document.fonts.ready.then(fit)` approach in the current code is insufficient for the Wails case because font readiness does not imply CSS layout completion. The double-rAF replaces it. If font loading is still a concern, `document.fonts.ready` can be awaited inside the second rAF callback.

### Pattern 2: Threading Initial Dimensions Through the Stack

**What:** Read the container pixel dimensions in the frontend at session creation time, convert to cols/rows via FitAddon, and pass them to the daemon when creating the session.

**The stack:** `TerminalPanel (read dims)` -> `App.createTab(cols, rows)` -> `CreateSession Wails binding` -> `daemon.Client.CreateSession` -> `POST /sessions` body -> `SessionEngine.CreateSession` -> `pty.CreateRequest{Cols, Rows}`.

**Current gap:**
- `daemon.CreateRequest` (types.go) has no cols/rows fields.
- `SessionEngine.CreateSession` hardcodes `Cols: 80, Rows: 24`.
- The Wails `App.CreateSession` method signature has no cols/rows parameters.

**Backend change (Go):**

```go
// internal/daemon/types.go — add to CreateRequest
type CreateRequest struct {
    CLI     string   `json:"cli"`
    Name    string   `json:"name"`
    WorkDir string   `json:"workDir"`
    Args    []string `json:"args,omitempty"`
    Cols    int      `json:"cols,omitempty"`  // 0 = use default (80)
    Rows    int      `json:"rows,omitempty"`  // 0 = use default (24)
}
```

```go
// internal/daemon/engine.go — apply sensible defaults
cols, rows := req.Cols, req.Rows
if cols <= 0 { cols = 80 }
if rows <= 0 { rows = 24 }

sess, err := e.backend.Create(ctx, pty.CreateRequest{
    CLI:     cliPath,
    Args:    req.Args,
    Cols:    cols,
    Rows:    rows,
    WorkDir: req.WorkDir,
})
```

**Wire format:** Adding `omitempty` means old clients that don't send cols/rows get 80x24 (backward compatible).

**Wails binding change (app.go):**

```go
// app.go — add cols, rows parameters
func (a *App) CreateSession(cli, name, workDir string, args []string, cols, rows int) (string, error) {
    return a.client.CreateSession(cli, name, workDir, args, cols, rows)
}
```

**Frontend change — where to get cols/rows:**

Option A: Read from FitAddon.proposeDimensions() at tab creation time.
Option B: Read from the container's offsetWidth/offsetHeight and calculate manually.

Option A is preferable because FitAddon already performs the exact measurement including padding. However, the TerminalPanel for the NEW session hasn't been mounted yet at creation time — the panel only exists after `setTabs` adds it and React re-renders.

The correct approach: pass a default (e.g., 220x50 based on typical viewport) or measure the active container's size at the time the user clicks "Create". The most reliable approach is to measure the existing `.terminal-container` div (the parent of all panels) in `App.tsx` when `createTab` is called, since it is always visible.

```typescript
// App.tsx — createTab callback
const createTab = useCallback(async (cliName: string, workDir: string, args: string[]) => {
  // Estimate initial dimensions from the terminal container
  const container = document.querySelector('.terminal-container') as HTMLElement | null
  let cols = 220, rows = 50  // Reasonable fallback
  if (container) {
    // Use character approximation: 8px wide, 17px tall at 14px font
    const w = container.clientWidth
    const h = container.clientHeight - 32  // subtract status bar height
    if (w > 0 && h > 0) {
      cols = Math.max(80, Math.floor(w / 8))
      rows = Math.max(24, Math.floor(h / 17))
    }
  }
  const sessionId = await CreateSession(cliName, defaultName, workDir, args, cols, rows)
  // ...
}, [tabCounter])
```

**Alternatively (cleaner):** Wait for the panel to mount, then send the real resize from the `isActive` effect. This is the double-rAF path. The initial spawn with approximate dims reduces visual "jump" from 80x24 to correct size.

### Pattern 3: Tab Activation Resize (existing, already wired)

The `term.onResize` handler in TerminalPanel already sends a resize frame to the daemon when xterm.js's cols/rows change. So the double-rAF fit fires, xterm.js resizes, `onResize` fires, and the daemon receives the correct size. This path already works for subsequent resizes — the bug is only the initial measurement timing.

### Anti-Patterns to Avoid

- **Calling fit() synchronously in the isActive useEffect:** The container is transitioning from `display:none` to `display:flex`; layout hasn't committed yet.
- **Using `setTimeout(fit, 0)`:** Better than synchronous, but less reliable than rAF in Wails' Chromium — 0ms setTimeout fires before paint in some browser versions.
- **Using only `document.fonts.ready`:** Font readiness ≠ layout completeness. Fonts may be ready before the Wails window's first paint, or the promise may already be resolved on subsequent tab switches.
- **Adding cols/rows to CreateRequest without omitempty:** Would break the wire format for clients that don't send cols/rows.
- **Passing 0 as cols/rows to the daemon:** Must default to 80x24 on the daemon side when 0 is received (guard in engine.go).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Measure container → cols/rows | Manual pixel math | FitAddon.proposeDimensions() | Handles padding, scrollbar width, cell measurement |
| Track container size changes | setInterval polling | ResizeObserver | Already wired; fires on layout changes |
| PTY resize signaling | Custom protocol | Existing MSG_RESIZE2 frame + onResize handler | Already implemented and working |

**Key insight:** The entire resize plumbing already exists and works. The bug is purely in timing — fit() fires before the browser has committed layout for the newly-visible container.

## Common Pitfalls

### Pitfall 1: cancelAnimationFrame Across Two rAFs

**What goes wrong:** The outer rAF id is cancelled, but the inner rAF (registered inside the outer callback) is not. If the component unmounts between the two frames, the inner rAF fires after cleanup.

**Why it happens:** The inner `requestAnimationFrame` call returns a separate id that must also be tracked.

**How to avoid:** Store both ids:
```typescript
let rafId2: number | undefined
const rafId1 = requestAnimationFrame(() => {
  rafId2 = requestAnimationFrame(() => { if (!cancelled) fit() })
})
return () => {
  cancelled = true
  cancelAnimationFrame(rafId1)
  if (rafId2 !== undefined) cancelAnimationFrame(rafId2)
  ro.disconnect()
}
```

**Warning signs:** Console warning about calling `fit()` on a disposed terminal.

### Pitfall 2: wails dev vs. wails build Behavior

**What goes wrong:** The single-rAF fix appears to work in `wails dev` (hot-reload Vite dev server) but fails in the production binary (`wails build`).

**Why it happens:** `wails dev` uses a separate Vite dev server with slightly different timing characteristics than the embedded WebView. The Wails production build embeds the frontend as a static asset served from memory, and the WebView's first layout pass may take an extra frame.

**How to avoid:** Always validate terminal fill in a production binary (`wails build -tags wailsassets`), not just in `wails dev`. STATE.md already flags this.

**Warning signs:** "Works in dev, broken in prod."

### Pitfall 3: FitAddon Returns undefined When Container Has No Height

**What goes wrong:** `fitAddon.fit()` silently does nothing — no error, no resize. The terminal stays at 80x24.

**Why it happens:** `proposeDimensions()` returns `undefined` when `parentElement` height is 0 or cell width/height is 0. `fit()` exits early without calling `terminal.resize()`.

**How to avoid:** After the double-rAF, verify the terminal actually resized by checking `termRef.current?.cols` before and after. Log a warning if cols/rows didn't change from the spawn defaults.

**Warning signs:** Terminal appears at 80 cols even after double-rAF; check if the container has `height: 0` in the browser dev tools.

### Pitfall 4: Wails TypeScript Binding Regeneration Required

**What goes wrong:** After changing `App.CreateSession` Go method signature (adding cols, rows int), the Wails-generated TypeScript binding at `frontend/src/wailsjs/go/main/App.ts` still has the old signature. The build compiles but sends wrong data.

**Why it happens:** Wails bindings are generated at `wails generate` time, not on every save.

**How to avoid:** After changing any Go method signature exposed to the frontend, run `wails generate` or rebuild with `wails build -tags wailsassets` which regenerates bindings as part of the build. Alternatively verify the generated `App.ts` matches the new signature.

**Warning signs:** TypeScript compilation succeeds but the Wails call sends 0 for cols/rows at runtime.

### Pitfall 5: ResizeObserver Fires Before Fonts Are Ready

**What goes wrong:** ResizeObserver's initial observation fires with correct container dimensions but incorrect cell dimensions (font not loaded), producing wrong cols/rows.

**Why it happens:** ResizeObserver fires when the container has layout dimensions but the FitAddon's character cell measurement uses a fallback font if `@font-face` hasn't loaded yet.

**How to avoid:** The double-rAF approach implicitly handles this in most cases because fonts load within the first frame. If fonts are custom/remote, consider awaiting `document.fonts.ready` inside the second rAF callback before calling fit.

## Code Examples

### Double-rAF Fit Pattern (isActive effect)

```typescript
// Source: derived from xterm.js community patterns + STATE.md decision
useEffect(() => {
  if (!isActive || !containerRef.current) return

  const container = containerRef.current
  let cancelled = false
  let rafId2: number | undefined
  const fit = () => fitAddonRef.current?.fit()

  // Double-rAF: ensures CSS layout is committed for the newly-visible container
  // before FitAddon measures it. Single rAF is insufficient in Wails production build.
  const rafId1 = requestAnimationFrame(() => {
    rafId2 = requestAnimationFrame(() => {
      if (!cancelled) fit()
    })
  })

  // ResizeObserver handles all subsequent size changes (window resize, etc.)
  const ro = new ResizeObserver(fit)
  ro.observe(container)

  return () => {
    cancelled = true
    cancelAnimationFrame(rafId1)
    if (rafId2 !== undefined) cancelAnimationFrame(rafId2)
    ro.disconnect()
  }
}, [isActive])
```

### Daemon CreateRequest with Cols/Rows (Go)

```go
// internal/daemon/types.go
type CreateRequest struct {
    CLI     string   `json:"cli"`
    Name    string   `json:"name"`
    WorkDir string   `json:"workDir"`
    Args    []string `json:"args,omitempty"`
    Cols    int      `json:"cols,omitempty"`
    Rows    int      `json:"rows,omitempty"`
}
```

### SessionEngine with Dimension Defaults (Go)

```go
// internal/daemon/engine.go - CreateSession
func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string, args []string, cols, rows int, onStatus func(string, status.SessionStatus)) (string, error) {
    cliPath := e.ResolveCLI(cli)
    if cols <= 0 { cols = 80 }
    if rows <= 0 { rows = 24 }

    sess, err := e.backend.Create(ctx, pty.CreateRequest{
        CLI:     cliPath,
        Args:    args,
        Cols:    cols,
        Rows:    rows,
        WorkDir: workDir,
    })
    // ...
}
```

### Daemon Client Threading (Go)

```go
// internal/daemon/client.go - CreateSession
func (c *Client) CreateSession(cli, name, workDir string, args []string, cols, rows int) (string, error) {
    req := CreateRequest{CLI: cli, Name: name, WorkDir: workDir, Args: args, Cols: cols, Rows: rows}
    // ... rest unchanged
}
```

### Wails App.go Binding (Go)

```go
// app.go
func (a *App) CreateSession(cli, name, workDir string, args []string, cols, rows int) (string, error) {
    return a.client.CreateSession(cli, name, workDir, args, cols, rows)
}
```

### Frontend createTab with Dimension Estimation (TypeScript)

```typescript
// App.tsx — createTab measures terminal container at creation time
const createTab = useCallback(async (cliName: string, workDir: string, args: string[]) => {
  const defaultName = `${cliName} ${tabCounter}`
  setTabCounter((n) => n + 1)

  // Estimate initial PTY dimensions from current container size.
  // These are approximations — the double-rAF fit will send the exact resize
  // once the panel is mounted. Using real estimates reduces initial "jump".
  const container = document.querySelector('.terminal-container') as HTMLElement | null
  let cols = 220, rows = 50
  if (container && container.clientWidth > 0 && container.clientHeight > 0) {
    const statusBarHeight = 32  // .tab-status-bar height from style.css
    cols = Math.max(80, Math.floor(container.clientWidth / 8))
    rows = Math.max(24, Math.floor((container.clientHeight - statusBarHeight) / 17))
  }

  try {
    const sessionId = await CreateSession(cliName, defaultName, workDir, args, cols, rows)
    // ... rest unchanged
  } catch (err) {
    console.error('[App] CreateSession failed:', err)
  }
}, [tabCounter])
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Terminal.open() + immediate fit() | Deferred fit via fonts.ready + ResizeObserver | Phase ~8 | Better but still misses initial layout timing |
| Synchronous resize on creation | Double-rAF deferred resize | Phase 34 (this phase) | Correct initial dimensions |
| Hardcoded 80x24 spawn | Caller-supplied dims with 80x24 fallback | Phase 34 (this phase) | PTY output formatted for actual viewport |

**Deprecated/outdated:**
- `document.fonts.ready.then(fit)` as the primary initial fit trigger: replaced by double-rAF.

## Open Questions

1. **Exact character cell size for dimension estimation**
   - What we know: FitAddon measures actual rendered cell size from `_renderService.dimensions`. The frontend estimation in `createTab` uses hardcoded 8px x 17px approximations.
   - What's unclear: Whether approximation is "good enough" given that the double-rAF resize will correct it immediately after mount.
   - Recommendation: Use the approximation for spawn dims. The double-rAF correction happens within one frame so visual flash is imperceptible. If flash is visible in QA, read dims from the active FitAddon of the currently-focused panel instead.

2. **Whether to await document.fonts.ready inside double-rAF**
   - What we know: Custom fonts (`Cascadia Code`, `MesloLGS NF`) are listed in the terminal font stack. If neither is installed, the fallback monospace is used immediately — no loading delay.
   - What's unclear: Whether the Wails production build bundles these fonts or relies on system fonts.
   - Recommendation: Await `document.fonts.ready` inside the second rAF for correctness; this adds no latency when fonts are already loaded.

## Environment Availability

Step 2.6: The phase is purely frontend TypeScript + Go changes. No new external dependencies.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js / pnpm | Frontend build | Already in use | — | — |
| Go toolchain | Backend changes | Already in use | — | — |
| wails CLI | Production build validation | Already in use | — | — |

**Missing dependencies with no fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` |
| Quick run command | `cd frontend && npm test` |
| Full suite command | `cd frontend && npm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TERM-04 | TerminalPanel source uses double-rAF pattern | unit (source inspection) | `cd frontend && npm test` | ✅ `__tests__/TerminalPanel.test.tsx` |
| TERM-03 | daemon.CreateRequest includes cols/rows fields | unit | `go test ./internal/daemon/...` | ✅ `internal/daemon/engine_test.go` |
| TERM-03 | engine.go applies 80/24 default when cols/rows = 0 | unit | `go test ./internal/daemon/...` | needs new test |
| TERM-01/02 | Initial fit happens after double-rAF (source inspection) | unit (source inspection) | `cd frontend && npm test` | needs addition to TerminalPanel.test.tsx |

### Sampling Rate
- **Per task commit:** `cd frontend && npm test && go test ./internal/daemon/... ./internal/pty/...`
- **Per wave merge:** same
- **Phase gate:** Full suite green before `/gsd:verify-work` + manual production binary validation per STATE.md note

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — needs new tests: TERM-04 double-rAF pattern check, TERM-01 initial-fit-not-synchronous check
- [ ] `internal/daemon/engine_test.go` — needs new test: TERM-03 default dimension guard (cols=0 → 80, rows=0 → 24)
- [ ] No new test files needed — extend existing files

## Sources

### Primary (HIGH confidence)
- Direct source inspection: `frontend/node_modules/@xterm/addon-fit/lib/addon-fit.js` — FitAddon.proposeDimensions() reads `getComputedStyle` on parent element
- Direct source inspection: `frontend/src/components/TerminalPanel.tsx` — current fit strategy
- Direct source inspection: `internal/daemon/engine.go` lines 49-55 — hardcoded 80x24
- Direct source inspection: `internal/daemon/types.go` — CreateRequest has no cols/rows
- Project STATE.md accumulated decisions — double-rAF requirement documented

### Secondary (MEDIUM confidence)
- xterm.js community pattern: double-rAF for FitAddon when container transitions from display:none — widely used pattern in xterm.js issues/discussions

### Tertiary (LOW confidence)
- Character cell size approximation (8px x 17px at 14px font size) — reasonable heuristic, unverified against exact Cascadia Code metrics

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already installed, no new dependencies
- Architecture: HIGH — root cause confirmed from source inspection, fix pattern well-established
- Pitfalls: HIGH — cancelAnimationFrame gap and wails dev vs. prod difference confirmed by STATE.md
- Initial dimension estimation: MEDIUM — approximation heuristic, may need adjustment

**Research date:** 2026-03-26
**Valid until:** 2026-06-26 (stable domain, xterm.js FitAddon API unlikely to change)
