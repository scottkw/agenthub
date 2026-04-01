# Phase 37: Splash Screen - Research

**Researched:** 2026-03-31
**Domain:** Wails v2 window lifecycle, React splash overlay, WebKit init latency masking
**Confidence:** HIGH

## Summary

Phase 37 adds a branded splash screen that appears immediately on app launch and dismisses when the daemon connection is confirmed (or after a 3-second timeout). The core challenge is preventing a white-flash before the window is ready: WebKit (WKWebView on macOS) takes 100–400ms to render the first frame, and without mitigation the window appears white before the React tree paints.

The canonical solution in Wails v2 is `StartHidden: true` + `OnDomReady` → `runtime.WindowShow(ctx)`. This keeps the OS window invisible until the WebView has finished its first DOM paint. A static HTML splash (inline in `index.html`, removed by React on mount) covers the blank-canvas period between DOM-ready and React's first meaningful render.

The daemon-dismissal signal comes from the same `GetDaemonError()` / `RetryDaemon()` path already present in `App.tsx`. No new Wails bindings are needed: the splash dismisses when `init()` completes without error, or after a 3-second `setTimeout` fallback.

**Primary recommendation:** Static HTML splash in `index.html` + `StartHidden: true` + `OnDomReady` show + React `SplashScreen` component that removes itself via `useState` once daemon init resolves.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BRND-02 | Splash screen shows full title logo during app startup, dismissed when daemon connection confirmed (no artificial delay, 3s timeout fallback) | Static HTML splash covers WebKit gap; React overlay covers daemon-init gap; `OnDomReady` + `StartHidden` eliminates dock-flash |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/wailsapp/wails/v2` | v2.10.2 (locked) | `StartHidden` + `OnDomReady` lifecycle hooks | Already in use; these fields exist in `pkg/options/options.go` |
| `github.com/wailsapp/wails/v2/pkg/runtime` | same | `runtime.WindowShow(ctx)` call | Already used in `app.go` for `WindowHide` |
| React | 18.x (existing) | `SplashScreen` component manages dismiss state | Already in use |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Vitest + jsdom | existing | Unit tests for splash component | Raw-source static analysis tests (same pattern as `App.test.tsx`) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `StartHidden` + `OnDomReady` show | Transparent window + opacity animation | Transparent windows on macOS require disabling the native shadow; adds complexity |
| Inline CSS in index.html | CSS import or JS injection | Inline CSS is synchronous and zero-latency; imported CSS may load after first paint |
| React state for dismiss | CSS animation + `transitionend` | CSS transition is smoother but harder to test; state is simpler and reliable |

**Installation:** No new packages required. All dependencies already present.

---

## Architecture Patterns

### Recommended Project Structure

No new directories needed. Changes span these existing locations:

```
frontend/
├── index.html                      # Add static splash HTML + inline CSS
└── src/
    ├── App.tsx                     # Add splash state, pass splashDone to SplashScreen
    ├── components/
    │   └── SplashScreen.tsx        # New: splash overlay component
    └── components/__tests__/
        └── SplashScreen.test.tsx   # New: vitest static-analysis tests

main.go                             # Add StartHidden: true, OnDomReady hook
app.go                              # Add domReady lifecycle + WindowShow call
```

### Pattern 1: StartHidden + OnDomReady WindowShow

**What:** Set `StartHidden: true` in `wails.Run` options. Register `OnDomReady` callback that calls `runtime.WindowShow(ctx)`. The window remains hidden at the OS level until the WebView DOM fires `DOMContentLoaded`.

**When to use:** Any Wails app that needs to prevent the blank-canvas flash on startup.

**Example:**
```go
// Source: /Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.10.2/pkg/options/options.go
// StartHidden and OnDomReady are confirmed fields in options.App.

err := wails.Run(&options.App{
    // ... existing fields ...
    StartHidden: true,
    OnDomReady:  app.domReady,
    OnStartup:   app.startup,
    // ...
})
```

```go
// In app.go — new method
func (a *App) domReady(ctx context.Context) {
    runtime.WindowShow(ctx)
}
```

**Critical:** `OnStartup` runs in a goroutine (daemon startup may block for hundreds of ms). `OnDomReady` is the correct hook because it fires after the WebView finishes the initial DOM parse — the window is visually ready at this point.

### Pattern 2: Static HTML Splash in index.html

**What:** Place a splash `<div>` with inline CSS directly in `index.html`. React replaces `#root`'s children when it mounts, which removes the static splash. This covers the gap between DOM-ready (window shows) and React's first paint.

**When to use:** Whenever you need zero-latency splash display — no JS bundle load required.

**Example:**
```html
<!-- frontend/index.html -->
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>AgentHub</title>
    <style>
      /* Inline — synchronous, no import latency */
      #splash-static {
        position: fixed;
        inset: 0;
        background: #1a1b26;  /* matches body background-color in style.css */
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9999;
      }
      #splash-static img {
        width: 320px;
        max-width: 80%;
      }
    </style>
  </head>
  <body>
    <div id="splash-static">
      <!-- Inline data URI or asset reference — React will hide/remove this -->
      <img src="/src/assets/agenthub-title-logo.png" alt="AgentHub" />
    </div>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

**Note on asset path:** In dev mode (`wails dev`), Vite serves assets directly. In production (`wails build`), assets are embedded via the `go:embed` asset server. The path `/src/assets/agenthub-title-logo.png` works in dev. For production, the Vite build copies assets to `dist/assets/` with a content-hash filename. The static `index.html` splash img will not resolve in production unless the image is also placed in `frontend/public/` (which Vite copies verbatim to `dist/` without hashing). **Resolution: copy `agenthub-title-logo.png` to `frontend/public/` and reference it as `/agenthub-title-logo.png`** — this path resolves correctly in both dev and production.

### Pattern 3: React SplashScreen Component with Daemon-Dismiss

**What:** A React component that renders on top of the app until daemon init completes. `App.tsx` passes a `done` prop (set after `init()` resolves). The component stays visible for a brief CSS fade-out, then unmounts.

**When to use:** When the dismiss trigger is async (daemon connection) rather than a fixed timer.

**Example:**
```tsx
// frontend/src/components/SplashScreen.tsx
interface SplashScreenProps {
  done: boolean
}

export function SplashScreen({ done }: SplashScreenProps) {
  const [visible, setVisible] = useState(!done)

  useEffect(() => {
    if (done) {
      // Optional: short delay for CSS fade, then unmount
      const t = setTimeout(() => setVisible(false), 300)
      return () => clearTimeout(t)
    }
  }, [done])

  if (!visible) return null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: '#1a1b26',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9999,
        opacity: done ? 0 : 1,
        transition: done ? 'opacity 0.3s ease' : 'none',
      }}
    >
      <img src="/agenthub-title-logo.png" alt="AgentHub" style={{ width: 320, maxWidth: '80%' }} />
    </div>
  )
}
```

```tsx
// In App.tsx — add splash state
const [splashDone, setSplashDone] = useState(false)

// In init() — after daemon check resolves (success or error):
async function init() {
  const startupErr = await GetDaemonError()
  if (startupErr) {
    setDaemonError(startupErr)
    setSplashDone(true)  // dismiss splash so error banner is visible
    return
  }
  // ... existing Promise.all ...
  setSplashDone(true)
}

// Fallback timeout — dismiss regardless after 3s
useEffect(() => {
  const t = setTimeout(() => setSplashDone(true), 3000)
  return () => clearTimeout(t)
}, [])

// In JSX — render SplashScreen as sibling to app content:
return (
  <div className="app">
    <SplashScreen done={splashDone} />
    {/* ... existing content ... */}
  </div>
)
```

### Anti-Patterns to Avoid

- **`OnStartup` for WindowShow:** `OnStartup` runs in a goroutine before the WebView finishes loading — calling `WindowShow` there shows a white window. Use `OnDomReady` instead.
- **`sleep`/`time.After` in OnDomReady:** Never add artificial delays in the Go show hook — the requirement explicitly forbids minimum-duration delays.
- **Hardcoded asset path with Vite hash:** Vite appends content hashes to asset filenames in production (`agenthub-title-logo-a1b2c3d4.png`). Static HTML cannot know this hash. Use `frontend/public/` to bypass hashing.
- **`display:none` on root instead of splash overlay:** Hiding `#root` until ready causes React to render into a hidden container, which can break xterm.js dimension calculations. Use a fixed overlay instead so React renders normally.
- **Not resetting `splashDone` to true on all code paths:** If `init()` throws unexpectedly, the splash would stay forever. Always call `setSplashDone(true)` in the `catch` block too.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Window show timing | Custom IPC message from JS to Go to show window | `StartHidden` + `OnDomReady` | Wails already handles the platform-specific WebView readiness signal |
| Asset path stability | Complex JS to discover hashed asset path | `frontend/public/` directory | Vite copies `public/` verbatim without hashing |
| Fade-out animation | Manual `requestAnimationFrame` loop | CSS `transition: opacity` | Single CSS property, handles all timing |

---

## Common Pitfalls

### Pitfall 1: White Flash on Production Build (Asset Path Breaks)
**What goes wrong:** Static HTML splash image 404s in production because Vite hashes the filename. Window shows with a visible broken-image icon instead of the logo.
**Why it happens:** Vite rewrites imports in JS but cannot rewrite literal strings in `index.html`'s `<img src>` that point into `src/assets/`. Files in `frontend/public/` are copied as-is.
**How to avoid:** Place splash logo in `frontend/public/agenthub-title-logo.png`. Reference it as `/agenthub-title-logo.png` in `index.html` and in `SplashScreen.tsx`.
**Warning signs:** Logo visible in `wails dev` but missing in `wails build` production test.

### Pitfall 2: Splash Stays Forever if init() Throws
**What goes wrong:** An unexpected exception in `init()` (e.g., `Environment()` fails) bypasses `setSplashDone(true)`, leaving the splash permanently over the UI.
**Why it happens:** `setSplashDone` is only called on the happy path and the daemon-error path.
**How to avoid:** Add `setSplashDone(true)` in the `catch (err)` block of `init()`, and keep the 3-second fallback `setTimeout` as a belt-and-suspenders guard.
**Warning signs:** App appears frozen at splash during testing.

### Pitfall 3: `OnStartup` vs `OnDomReady` Confusion
**What goes wrong:** `WindowShow` called in `OnStartup` shows the window before the WebView has loaded, producing a blank/white flash — exactly what `StartHidden` was meant to prevent.
**Why it happens:** `OnStartup` fires when the Wails app object is initialized, not when the WebView DOM is ready.
**How to avoid:** Always use `OnDomReady` for the show call. `OnStartup` is for Go-side initialization only.
**Warning signs:** White flash visible before splash during production build test.

### Pitfall 4: Dock Flash When Using HideWindowOnClose + StartHidden
**What goes wrong:** On macOS, the Dock icon briefly flashes even with `StartHidden: true`.
**Why it happens:** The Dock icon is controlled by `LSUIElement` in `Info.plist`, not by `StartHidden`. The two concerns are orthogonal.
**How to avoid:** This is expected behavior until Phase 41 (LSUIElement). Phase 37 does not add `LSUIElement` — that is Phase 41's responsibility.
**Warning signs:** Dock flash during testing is expected and acceptable for Phase 37.

### Pitfall 5: xterm.js Dimension Miscalculation
**What goes wrong:** `TerminalPanel` mounts while the splash overlay covers the viewport, and `FitAddon.proposeDimensions()` returns wrong values because the container is visually hidden or zero-sized.
**Why it happens:** The splash is a `position:fixed` overlay with `z-index:9999` — it does NOT hide the content underneath from the DOM. The terminal container is still rendered and has real dimensions.
**How to avoid:** Using `position:fixed` overlay (not `display:none` on the container) means xterm.js sees correct dimensions. No extra handling needed.
**Warning signs:** Terminal appears with wrong column/row count after splash dismisses.

---

## Code Examples

### Go: Adding StartHidden + OnDomReady to main.go

```go
// Source: confirmed field names from
// /Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.10.2/pkg/options/options.go
// Lines 45, 62: StartHidden bool; OnDomReady func(ctx context.Context)

err := wails.Run(&options.App{
    Title:             "AgentHub",
    Width:             1200,
    Height:            800,
    MinWidth:          800,
    MinHeight:         600,
    StartHidden:       true,           // NEW: prevent blank-window flash
    HideWindowOnClose: true,
    BackgroundColour:  &options.RGBA{R: 0x1a, G: 0x1b, B: 0x26, A: 0xff},
    AssetServer: &assetserver.Options{
        Assets: assets,
    },
    OnStartup:     app.startup,
    OnDomReady:    app.domReady,       // NEW
    OnShutdown:    app.shutdown,
    OnBeforeClose: app.beforeClose,
    Bind: []interface{}{
        app,
    },
})
```

### Go: domReady method in app.go

```go
// Source: runtime.WindowShow confirmed in
// /Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.10.2/pkg/runtime/window.go

// domReady is called by Wails after the WebView DOM is ready.
// Shows the window now that the splash is rendered and visible.
func (a *App) domReady(ctx context.Context) {
    runtime.WindowShow(ctx)
}
```

### Frontend: SplashScreen component

```tsx
// frontend/src/components/SplashScreen.tsx
import React, { useState, useEffect } from 'react'

interface SplashScreenProps {
  done: boolean
}

export function SplashScreen({ done }: SplashScreenProps): React.ReactElement | null {
  const [visible, setVisible] = useState(true)

  useEffect(() => {
    if (done) {
      const t = setTimeout(() => setVisible(false), 300)
      return () => clearTimeout(t)
    }
  }, [done])

  if (!visible) return null

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: '#1a1b26',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9999,
        opacity: done ? 0 : 1,
        transition: done ? 'opacity 0.3s ease' : 'none',
        pointerEvents: 'none',
      }}
    >
      <img
        src="/agenthub-title-logo.png"
        alt="AgentHub"
        style={{ width: 320, maxWidth: '80%' }}
        draggable={false}
      />
    </div>
  )
}
```

### Frontend: App.tsx splash state additions

```tsx
// Add to state declarations in App():
const [splashDone, setSplashDone] = useState(false)

// Add fallback timeout (belt-and-suspenders, covers all failure modes):
useEffect(() => {
  const t = setTimeout(() => setSplashDone(true), 3000)
  return () => clearTimeout(t)
}, [])

// Modify init() to always call setSplashDone(true) on ALL exit paths:
async function init() {
  try {
    const startupErr = await GetDaemonError()
    if (startupErr) {
      setDaemonError(startupErr)
      setSplashDone(true)   // dismiss so error banner is visible
      return
    }
    const [port, clis, sessions, running, health, env] = await Promise.all([...])
    // ... existing setters ...
    setSplashDone(true)     // dismiss when UI data is ready
  } catch (err) {
    console.error('[App] init failed:', err)
    setDaemonError(String(err))
    setSplashDone(true)     // dismiss even on unexpected errors
  }
}

// In JSX — add SplashScreen as first child of app div:
return (
  <div className="app">
    <SplashScreen done={splashDone} />
    <TabBar ... />
    <div className="terminal-container">
      ...
    </div>
    ...
  </div>
)
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `OnStartup` + immediate show | `StartHidden: true` + `OnDomReady` show | Wails v2.x (present in v2.10.2) | Eliminates blank-window flash |
| Splash as separate OS window | Splash as React overlay inside single window | Wails v2 (single-window only) | Simpler; no second window API needed |
| Timed minimum display | Event-driven dismiss with fallback timeout | Best practice | No artificial delays |

**Out-of-scope for Phase 37:**
- `LSUIElement` / Dock hiding: deferred to Phase 41 per STATE.md decisions
- Tray icon integration: Phase 41

---

## Open Questions

1. **Fade-out duration (300ms)**
   - What we know: 300ms is a common animation duration that feels responsive without being jarring
   - What's unclear: Whether the product prefers an instant cut or a fade
   - Recommendation: Default to 300ms fade; trivially adjustable in `SplashScreen.tsx`

2. **Static splash in index.html vs. React-only splash**
   - What we know: There is a ~50–200ms gap between `OnDomReady` (window shows) and React's first paint where the screen could be blank
   - What's unclear: Whether this gap is perceptible in practice on the target hardware
   - Recommendation: Include the static HTML splash for correctness; it is low-risk and handles the worst case

---

## Environment Availability

Step 2.6: SKIPPED (no external tool dependencies — changes are code/config only within the existing Wails + React stack)

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 2.x + jsdom |
| Config file | `frontend/vite.config.ts` (test section present) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BRND-02 | SplashScreen component exists and renders logo img | unit (raw source) | `pnpm test -- SplashScreen` | ❌ Wave 0 |
| BRND-02 | App.tsx declares `splashDone` state | unit (raw source) | `pnpm test -- App` | ✅ (extend existing) |
| BRND-02 | App.tsx renders `<SplashScreen` | unit (raw source) | `pnpm test -- App` | ✅ (extend existing) |
| BRND-02 | App.tsx has 3-second fallback timeout | unit (raw source) | `pnpm test -- App` | ✅ (extend existing) |
| BRND-02 | `setSplashDone(true)` called on error path | unit (raw source) | `pnpm test -- App` | ✅ (extend existing) |
| BRND-02 | main.go has `StartHidden: true` | manual / visual | `wails build` production test | manual |
| BRND-02 | No white flash before splash | manual / visual | launch built app | manual |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full vitest suite green + manual production build visual verification

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/SplashScreen.test.tsx` — covers BRND-02 (component structure, img src, visible/hidden state)

*(Existing `App.test.tsx` will be extended — not a gap, but requires new test cases)*

---

## Project Constraints (from CLAUDE.md)

- **Node package manager:** `pnpm` preferred (all frontend commands use `pnpm`)
- **TypeScript:** `camelCase` variables, `PascalCase` components, ESLint + Prettier, TypeScript types
- **No artificial delays:** REQUIREMENTS.md explicitly lists "Splash screen with fixed minimum duration" as out of scope
- **Wails v3 migration:** Out of scope — staying on v2; v3 is alpha
- **Second OS-level window:** Not possible — Wails v2 is single-window only
- **`LSUIElement`:** Do not add in Phase 37 — deferred to Phase 41 per STATE.md decisions
- **`defer setInitComplete(true)`:** STATE.md decision for Phase 37 says `defer setInitComplete(true)` on all code paths — implement as `setSplashDone(true)` on all `init()` exit paths (success, daemon error, unexpected error)

---

## Sources

### Primary (HIGH confidence)
- `/Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.10.2/pkg/options/options.go` — confirmed `StartHidden bool` (line 45), `OnDomReady func(ctx context.Context)` (line 62)
- `/Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.10.2/pkg/runtime/window.go` — confirmed `func WindowShow(ctx context.Context)` and `func WindowHide(ctx context.Context)`
- `/Users/ken/dev/agenthub/main.go` — existing `wails.Run` call structure; confirms current options in use
- `/Users/ken/dev/agenthub/app.go` — confirms `OnStartup`, `OnShutdown`, `OnBeforeClose` patterns; `runtime.WindowHide` already called
- `/Users/ken/dev/agenthub/frontend/src/App.tsx` — confirmed init flow, `GetDaemonError()` call, error state handling pattern
- `/Users/ken/dev/agenthub/frontend/index.html` — confirmed current structure (no splash yet)
- `/Users/ken/dev/agenthub/frontend/src/assets/agenthub-title-logo.png` — confirmed asset exists (169KB, Phase 36 output)
- `/Users/ken/dev/agenthub/frontend/src/components/__tests__/App.test.tsx` — confirmed raw-source test pattern (vitest, no DOM rendering)
- `.planning/STATE.md` — locked decision: `StartHidden: true` + `OnDomReady` → `runtime.WindowShow()` + `defer setInitComplete(true)` on all code paths

### Secondary (MEDIUM confidence)
- Vite `public/` directory behavior — documented Vite convention: files in `public/` are copied verbatim to `dist/` without hashing, accessible at root `/`

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all APIs verified in local Wails v2.10.2 source
- Architecture: HIGH — patterns derived from existing codebase + verified Wails API
- Pitfalls: HIGH — asset-path issue is a known Vite behavior; other pitfalls derived from reading the existing code

**Research date:** 2026-03-31
**Valid until:** 2026-06-30 (stable stack, no fast-moving dependencies)
