# Phase 8: Per-Tab Status Bar - Research

**Researched:** 2026-03-19
**Domain:** React component layout, CSS flexbox, Wails desktop UI
**Confidence:** HIGH

## Summary

Phase 8 moves the web-serving controls from a floating header overlay (`.web-serving-bar` currently positioned at the TOP of `.terminal-wrapper`) to a permanent, fixed-height status strip anchored to the BOTTOM of every tab. The second requirement removes the overlay entirely, leaving clean terminal content above and the status bar below.

The technical work is almost entirely within `frontend/src/App.tsx` and `frontend/src/style.css`. The current `.web-serving-bar` renders above the `<TerminalPanel>` inside a flex column `.terminal-wrapper`, meaning the terminal fills whatever vertical space remains after the bar takes its share. Moving the bar to the bottom means reordering the JSX (put status bar after `<TerminalPanel>`), giving it a fixed height, and ensuring `flex: 1; min-height: 0` on the terminal keeps working correctly in its new position. No new dependencies are required.

The critical architectural insight is that the status bar must be **always present** (permanent) rather than conditionally rendered — it should show a "Web serving off" or similar neutral state when web serving is disabled or the web server is not running, rather than disappearing. This avoids the layout jump that occurs when the bar appears/disappears and ensures the terminal always has a constant viewport size. The terminal's `ResizeObserver`-driven `fitAddon.fit()` will detect any layout dimension change, but eliminating layout jumps is still best practice.

**Primary recommendation:** Reorder JSX in `App.tsx` so the status bar renders below `<TerminalPanel>`, give it a fixed height in CSS, make it always-visible (not conditionally rendered on `webServerRunning`), and remove the old `.web-serving-bar` conditional render.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| UILAY-02 | Each tab displays a status bar at the bottom showing web serving state, URL, and controls | Always-visible fixed-height bar below TerminalPanel in terminal-wrapper flex column; contains web state, URL, and action buttons |
| UILAY-03 | Web status/URL header overlay is removed from tab content area | Remove the conditional `{webServerRunning && <div className="web-serving-bar">...}` block from App.tsx; delete associated CSS |
</phase_requirements>

## Standard Stack

### Core (already installed — no new dependencies needed)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | ^19.2.4 | UI framework | Already in use; JSX reorder + new component |
| TypeScript | ^5.9.3 | Type safety | Already in use |

### No New Dependencies

Phase 8 requires zero new npm packages. All changes are JSX restructuring, CSS additions, and a new `StatusBar` component extracted from `App.tsx`.

**Installation:**
```bash
# Nothing to install
```

## Architecture Patterns

### Current Layout (before Phase 8)

```
.app  (flex column)
  .tab-bar  (flex-shrink: 0; height: 42px)
  .terminal-container  (flex: 1; min-height: 0)
    .terminal-wrapper  (flex column; width:100%; height:100%)
      .web-serving-bar  [TOP — conditionally rendered when webServerRunning]
      TerminalPanel > div  (flex:1; minHeight:0)
```

### Target Layout (after Phase 8)

```
.app  (flex column)
  .tab-bar  (flex-shrink: 0; height: 42px)
  .terminal-container  (flex: 1; min-height: 0)
    .terminal-wrapper  (flex column; width:100%; height:100%)
      TerminalPanel > div  (flex:1; minHeight:0)
      .tab-status-bar  [BOTTOM — always present, fixed height]
```

### Pattern 1: Always-Present Status Bar

**What:** The status bar is rendered unconditionally for every tab wrapper. It shows different content depending on state (web serving off, web serving on + URL, web server not running). A conditionally rendered bar causes layout reflow when it appears/disappears, which forces `fitAddon.fit()` to recalculate terminal dimensions on every toggle.

**When to use:** Any time a UI element needs to be stable to preserve terminal viewport size.

**Example:**
```tsx
// Always present — never conditionally rendered
<div className="tab-status-bar">
  {webServerRunning ? (
    webEnabled[tab.sessionId] ? (
      // Serving: show URL and controls
      <>
        <span className="tab-status-bar__state tab-status-bar__state--on">Web On</span>
        <a className="tab-status-bar__url" href={sessionURLs[tab.sessionId]} target="_blank" rel="noreferrer">
          {sessionURLs[tab.sessionId]}
        </a>
        <button onClick={() => void handleToggleWeb(tab.sessionId)}>Disable</button>
        <button onClick={() => void handleCopyTokenLink(tab.sessionId)}>Copy Link</button>
        <button onClick={() => setQrSessionId(tab.sessionId)}>QR</button>
      </>
    ) : (
      // Server running but session not sharing
      <>
        <span className="tab-status-bar__state tab-status-bar__state--off">Web Off</span>
        <button onClick={() => void handleToggleWeb(tab.sessionId)}>Enable</button>
      </>
    )
  ) : (
    // Web server not running
    <span className="tab-status-bar__state tab-status-bar__state--inactive">Web server not running</span>
  )}
</div>
```

### Pattern 2: Fixed-Height Bottom Bar in Flex Column

**What:** The status bar must not shrink the terminal unpredictably. Use `flex-shrink: 0` and an explicit height so the terminal above always gets `flex: 1` of the remaining space.

**Example CSS:**
```css
.tab-status-bar {
  flex-shrink: 0;
  height: 32px;          /* fixed — never auto-sizes to content */
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  background-color: #16161e;
  border-top: 1px solid #292e42;
  overflow: hidden;      /* prevent content from expanding height */
  font-size: 12px;
  color: #565f89;
}
```

**Why fixed height matters:** If the bar uses `height: auto`, adding or removing URL text causes the bar to grow/shrink, which triggers a `ResizeObserver` callback on the terminal container and a `fitAddon.fit()` recalculation. With a fixed height the terminal viewport stays stable across state changes within the status bar.

### Pattern 3: Extracting StatusBar as a Component

**What:** The status bar logic currently lives in App.tsx as inline JSX within the `tabs.map()`. It can remain inline (simpler) or be extracted to `StatusBar.tsx` (cleaner, testable). Given the amount of conditional rendering logic and the need for unit tests, extraction is recommended.

**Proposed component interface:**
```tsx
// frontend/src/components/StatusBar.tsx
interface StatusBarProps {
  sessionId: string
  webServerRunning: boolean
  webEnabled: boolean
  sessionURL: string | undefined
  onToggleWeb: () => void
  onCopyTokenLink: () => void
  onShowQR: () => void
}
export function StatusBar(props: StatusBarProps): React.ReactElement
```

**App.tsx usage:**
```tsx
<div className="terminal-wrapper" style={{ display: isActive ? 'flex' : 'none' }}>
  <TerminalPanel sessionId={tab.sessionId} isActive={isActive} relayPort={relayPort} />
  <StatusBar
    sessionId={tab.sessionId}
    webServerRunning={webServerRunning}
    webEnabled={!!webEnabled[tab.sessionId]}
    sessionURL={sessionURLs[tab.sessionId]}
    onToggleWeb={() => void handleToggleWeb(tab.sessionId)}
    onCopyTokenLink={() => void handleCopyTokenLink(tab.sessionId)}
    onShowQR={() => setQrSessionId(tab.sessionId)}
  />
</div>
```

### Anti-Patterns to Avoid

- **Conditional render of the status bar:** `{webServerRunning && <StatusBar ...>}` causes layout jumps when web server starts/stops. Always render the bar; change its content based on state.
- **`height: auto` on the status bar:** Allows content changes (e.g., URL appearing) to resize the bar, triggering unnecessary `fit()` calls and terminal viewport jitter.
- **Moving web-serving logic to TerminalPanel:** TerminalPanel is intentionally a pure terminal component with no app-layer concerns. Status bar state (webEnabled, sessionURLs) stays in App.tsx, passed down as props.
- **Removing `.web-serving-bar` CSS before deleting the JSX that uses it:** Will cause broken styles during transition. Remove both in the same commit.

### Recommended Project Structure

```
frontend/src/
├── App.tsx                          # Remove old web-serving-bar JSX; add StatusBar usage
├── style.css                        # Remove .web-serving-bar styles; add .tab-status-bar styles
└── components/
    ├── StatusBar.tsx                # New component extracted from App.tsx inline JSX
    └── __tests__/
        └── StatusBar.test.tsx       # Unit tests for StatusBar rendering
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Stable terminal height | Custom dimension recalculation after status bar toggle | Always-present status bar with fixed height | ResizeObserver + fitAddon already handles resize; avoid triggering it unnecessarily |
| URL truncation | Custom ellipsis JS | CSS `overflow: hidden; text-overflow: ellipsis; white-space: nowrap` | Handles all cases including long tunnel URLs |
| Status indicator icons | SVG icon library | Unicode/text chars or CSS-only dot indicators | Already used in tab status dots; consistent with existing style |

**Key insight:** The project already has a working pattern for fixed-height chrome strips (`.tab-bar`, `.web-serving-bar` with `flex-shrink: 0`). The status bar reuses this exact same CSS flex pattern — just at the bottom instead of the top.

## Common Pitfalls

### Pitfall 1: Status Bar Causes Terminal Viewport Jitter on State Change

**What goes wrong:** When `webEnabled` or `webServerRunning` changes, the bar content changes, and if height is not fixed, the bar grows or shrinks, triggering `ResizeObserver` on the terminal container, causing `fit()` to run and possibly snapping the terminal to different col/row counts.

**Why it happens:** `height: auto` combined with content-driven rendering.

**How to avoid:** Set `height: 32px` (or similar fixed value) on `.tab-status-bar` and `overflow: hidden`. The bar's visible content changes, but its height does not.

**Warning signs:** Terminal cursor position shifts or text reflows when toggling web serving.

### Pitfall 2: Forgetting to Remove Old `.web-serving-bar` from CSS

**What goes wrong:** The old selector `.web-serving-bar` remains in `style.css` with `flex-shrink: 0` but no corresponding DOM element. This is harmless but is dead code and will confuse future readers.

**How to avoid:** Remove the `.web-serving-bar`, `.web-toggle-btn`, `.web-session-url`, `.copy-token-btn` CSS rules in the same commit as the JSX removal. The `.qr-btn` style may be reused in the status bar, or renamed.

### Pitfall 3: Layout Chain Break After Reorder

**What goes wrong:** After moving `<TerminalPanel>` above the status bar, the terminal no longer fills the correct space — it either overflows or shrinks to minimum.

**Why it happens:** JSX reorder changes the DOM order, and if `.terminal-wrapper` was implicitly relying on the bar being first to set a height anchor, the chain could break.

**How to avoid:** Ensure `.terminal-wrapper` remains `display: flex; flex-direction: column; height: 100%` (unchanged), `<TerminalPanel>` container has `flex: 1; min-height: 0` (unchanged), and the status bar has `flex-shrink: 0; height: 32px` (new). This is the same pattern as the tab bar above the terminal container.

**Warning signs:** Terminal has no height or shows dead space at bottom.

### Pitfall 4: Status Bar Hidden on Inactive Tabs

**What goes wrong:** The status bar is inside the `display: none` tab wrapper for inactive tabs — it is not visible when a tab is inactive, which is correct. But if it were OUTSIDE the wrapper, it would show for non-active sessions.

**How to avoid:** Keep the status bar inside `.terminal-wrapper` which already has `display: isActive ? 'flex' : 'none'`. No extra visibility logic needed.

### Pitfall 5: `qr-btn` CSS Lost in Cleanup

**What goes wrong:** The `.qr-btn` CSS currently handles the QR button in the old `.web-serving-bar`. If the button moves to `.tab-status-bar`, the style still applies (CSS class is still present), but the visual context changes. Check that `qr-btn` sizing and border fit the thinner status bar.

**How to avoid:** Audit button padding when moving to 32px height bar. May need to reduce padding from `4px 10px` to `2px 8px`.

## Code Examples

### Complete terminal-wrapper structure after Phase 8

```tsx
// Source: App.tsx — target state after Phase 8
tabs.map((tab) => {
  const isActive = tab.id === activeId
  return (
    <div
      key={tab.sessionId}
      className="terminal-wrapper"
      style={{ display: isActive ? 'flex' : 'none' }}
    >
      <TerminalPanel
        sessionId={tab.sessionId}
        isActive={isActive}
        relayPort={relayPort}
      />
      <StatusBar
        sessionId={tab.sessionId}
        webServerRunning={webServerRunning}
        webEnabled={!!webEnabled[tab.sessionId]}
        sessionURL={sessionURLs[tab.sessionId]}
        onToggleWeb={() => void handleToggleWeb(tab.sessionId)}
        onCopyTokenLink={() => void handleCopyTokenLink(tab.sessionId)}
        onShowQR={() => setQrSessionId(tab.sessionId)}
      />
    </div>
  )
})
```

### tab-status-bar CSS

```css
/* Source: project CSS patterns + this phase design */

/* Remove old .web-serving-bar rule — replaced by .tab-status-bar */

.tab-status-bar {
  flex-shrink: 0;
  height: 32px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  background-color: #16161e;
  border-top: 1px solid #292e42;
  overflow: hidden;
  font-size: 12px;
  color: #565f89;
  font-family: '"Cascadia Code"', '"MesloLGS NF"', '"Fira Code"', monospace;
}

.tab-status-bar__state {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  flex-shrink: 0;
}
.tab-status-bar__state--on  { color: #9ece6a; }  /* green — web serving active */
.tab-status-bar__state--off { color: #565f89; }  /* muted — web off */
.tab-status-bar__state--inactive { color: #414868; }  /* very muted — server not running */

.tab-status-bar__url {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #7aa2f7;
  font-size: 12px;
  text-decoration: none;
  min-width: 0;  /* critical for flex text-overflow to work */
}
.tab-status-bar__url:hover {
  text-decoration: underline;
}

.tab-status-bar__btn {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 4px;
  border: 1px solid #292e42;
  background: transparent;
  color: #a9b1d6;
  cursor: pointer;
  font-size: 11px;
  font-family: inherit;
  white-space: nowrap;
}
.tab-status-bar__btn:hover {
  background-color: #1e2030;
  color: #c0caf5;
}
.tab-status-bar__btn--active {
  border-color: #7aa2f7;
  color: #7aa2f7;
}
```

### StatusBar component unit test pattern

```tsx
// Source: project test pattern (TabBar.test.tsx)
// frontend/src/components/__tests__/StatusBar.test.tsx

import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { StatusBar } from '../StatusBar'

function renderStatusBar(props: Partial<React.ComponentProps<typeof StatusBar>> = {}) {
  const defaults = {
    sessionId: 'test-session',
    webServerRunning: false,
    webEnabled: false,
    sessionURL: undefined,
    onToggleWeb: vi.fn(),
    onCopyTokenLink: vi.fn(),
    onShowQR: vi.fn(),
  }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(StatusBar, { ...defaults, ...props }))
  })
  return { container, root }
}

describe('StatusBar', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders tab-status-bar root element', () => {
    ;({ container, root } = renderStatusBar())
    expect(container.querySelector('.tab-status-bar')).not.toBeNull()
  })

  it('shows inactive state when web server not running', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: false }))
    expect(container.querySelector('.tab-status-bar__state--inactive')).not.toBeNull()
  })

  it('shows off state when server running but web disabled', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: false }))
    expect(container.querySelector('.tab-status-bar__state--off')).not.toBeNull()
  })

  it('shows on state and URL when web enabled', () => {
    ;({ container, root } = renderStatusBar({
      webServerRunning: true,
      webEnabled: true,
      sessionURL: 'https://example.com/sessions/test-session',
    }))
    expect(container.querySelector('.tab-status-bar__state--on')).not.toBeNull()
    expect(container.querySelector('.tab-status-bar__url')).not.toBeNull()
  })
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Floating overlay bar above terminal | Fixed-height bottom strip always present | Phase 8 | No layout jump; consistent terminal viewport |
| Conditional render on `webServerRunning` | Always rendered; content changes by state | Phase 8 | Terminal height stable across server state changes |

**Deprecated/outdated after Phase 8:**
- `.web-serving-bar` CSS class: replaced by `.tab-status-bar`
- `.web-toggle-btn`, `.web-toggle-btn--active`: replaced by `.tab-status-bar__btn`
- `.web-session-url`: replaced by `.tab-status-bar__url`
- `.copy-token-btn`: replaced by `.tab-status-bar__btn`

## Open Questions

1. **Height of the status bar: 32px vs 28px vs 36px**
   - What we know: Tab bar is 42px. Status bar is supplementary chrome and should be visually secondary — smaller than the tab bar.
   - What's unclear: Whether 32px fits all button content comfortably at 11px font on Windows/Linux.
   - Recommendation: Start with 32px (matching common terminal status bars like VS Code's). If buttons feel cramped on Windows, bump to 34px.

2. **Should the status bar render when no tabs are open (empty state)?**
   - What we know: The entire `.terminal-container` shows nothing when `tabs` is empty; the map produces nothing.
   - What's unclear: Nothing — the status bar is inside each tab wrapper, so it only renders when a tab exists.
   - Recommendation: No action needed.

3. **Does `.qr-btn` CSS need to be renamed?**
   - What we know: `.qr-btn` is currently defined on its own in `style.css` (not nested under `.web-serving-bar`), so it applies by class name.
   - Recommendation: Reuse `.tab-status-bar__btn` for the QR button too — consistent with the new naming. Remove the standalone `.qr-btn` rule.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (vitest configured inline) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UILAY-02 | Status bar renders for each tab | unit | `pnpm test -- --reporter=verbose` | Wave 0 — needs StatusBar.test.tsx |
| UILAY-02 | Status bar shows inactive state when server not running | unit | `pnpm test -- --reporter=verbose` | Wave 0 — needs StatusBar.test.tsx |
| UILAY-02 | Status bar shows off state when server running, web disabled | unit | `pnpm test -- --reporter=verbose` | Wave 0 — needs StatusBar.test.tsx |
| UILAY-02 | Status bar shows URL and controls when web enabled | unit | `pnpm test -- --reporter=verbose` | Wave 0 — needs StatusBar.test.tsx |
| UILAY-02 | Status bar is at bottom of tab (position/layout) | manual-only | Visual inspection in running app | N/A — layout, not logic |
| UILAY-03 | Old web-serving-bar overlay is absent from DOM | unit | Source code inspection via ?raw import | Wave 0 — needs test in App.test.tsx or StatusBar.test.tsx |
| UILAY-03 | Terminal fills full height with no dead space above status bar | manual-only | Visual inspection | N/A — CSS layout |

**Note on UILAY-03 automation:** Verifying the old overlay is absent can be done by asserting the App source does not contain the `.web-serving-bar` class name after Phase 8.

### Wave 0 Gaps

- [ ] `frontend/src/components/__tests__/StatusBar.test.tsx` — covers UILAY-02 (rendering, state variants)
- [ ] `frontend/src/components/StatusBar.tsx` — the component itself (extracted from App.tsx)

*(Existing vitest infrastructure covers all tooling needs — only new source + test files are missing)*

## Sources

### Primary (HIGH confidence)

- Codebase direct read: `frontend/src/App.tsx` — complete web-serving-bar implementation, all props passed, all handlers
- Codebase direct read: `frontend/src/style.css` — current `.web-serving-bar`, `.web-toggle-btn`, `.web-session-url`, `.copy-token-btn`, `.qr-btn` rules
- Codebase direct read: `frontend/src/components/TerminalPanel.tsx` — inline styles confirming flex chain
- Codebase direct read: `.planning/phases/07-layout-baseline/07-RESEARCH.md` — flex chain patterns established in Phase 7
- Codebase direct read: `frontend/package.json` — confirmed zero new deps needed

### Secondary (MEDIUM confidence)

- MDN Flexbox: `flex-shrink: 0` on fixed-height chrome strips; `min-width: 0` for truncated flex text
- CSS `text-overflow: ellipsis` requires `overflow: hidden; white-space: nowrap; min-width: 0` on flex children

### Tertiary (LOW confidence)

- None — all critical findings from direct codebase inspection

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new deps; all from direct package.json inspection
- Architecture: HIGH — full source code available; layout chain fully traceable; Phase 7 established the flex patterns
- Pitfalls: HIGH — identified from code inspection and established CSS flex behavior
- CSS values: HIGH for structure; MEDIUM for exact pixel heights (32px is a reasonable estimate, may need minor tuning)

**Research date:** 2026-03-19
**Valid until:** Stable — CSS flexbox behavior and existing component interfaces are stable
