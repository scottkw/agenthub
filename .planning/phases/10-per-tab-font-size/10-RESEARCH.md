# Phase 10: Per-Tab Font Size - Research

**Researched:** 2026-03-19
**Domain:** xterm.js font size control, keyboard event interception, per-tab React state
**Confidence:** HIGH

## Summary

Phase 10 adds keyboard-driven font size adjustment to every terminal tab. Users press SHIFT+= (which produces `+` on most keyboards) to increase font size and SHIFT+- (which produces `_` on most keyboards, though the actual key value is `-`) to decrease font size. Each tab maintains its own independent font size so switching tabs does not reset other tabs' sizes. Font size changes must not inject characters into the PTY — a critical requirement that xterm.js satisfies via `attachCustomKeyEventHandler`.

The implementation has two interlocking mechanisms: (1) `attachCustomKeyEventHandler` returns `false` for the SHIFT+= and SHIFT+- events so xterm suppresses their default "forward to PTY" behavior, and (2) after mutating `terminal.options.fontSize`, `fitAddon.fit()` must be called immediately so the terminal reflows to fill its container with the new cell dimensions. Both halves are mandatory — omitting either causes a visible defect.

Per-tab font size state lives in `App.tsx` as a `Record<string, number>` (sessionId → fontSize), mirroring the existing patterns for `webEnabled`, `sessionURLs`, and `sessionStatuses`. `TerminalPanel` receives a `fontSize` prop and an `onFontSizeChange` callback. The terminal's `attachCustomKeyEventHandler` intercepts SHIFT+= / SHIFT+- inside `TerminalPanel` and calls the callback.

**Primary recommendation:** Store `fontSizes: Record<string, number>` in `App.tsx`. Pass `fontSize` and `onFontSizeChange` as props to `TerminalPanel`. Inside `TerminalPanel`, call `attachCustomKeyEventHandler` during terminal setup to intercept SHIFT+= / SHIFT+-, call `onFontSizeChange`, and return `false`. In the `isActive` effect, apply `term.options.fontSize = fontSize` then `fitAddon.fit()` whenever `fontSize` prop changes.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TERM-02 | User can press SHIFT+ to increase font size in the active terminal tab | `attachCustomKeyEventHandler` intercepts Shift+= (key `=`, shiftKey `true`); `terminal.options.fontSize += 1`; `fitAddon.fit()` |
| TERM-03 | User can press SHIFT- to decrease font size in the active terminal tab | `attachCustomKeyEventHandler` intercepts Shift+- (key `-`, shiftKey `true`); `terminal.options.fontSize -= 1`; `fitAddon.fit()` |
| TERM-04 | Font size changes persist per tab and do not leak characters to the PTY | Per-tab `fontSizes` Record in App.tsx; handler returns `false` to suppress PTY injection |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @xterm/xterm | 6.0.0 | `terminal.options.fontSize` write; `attachCustomKeyEventHandler` | Already installed, already used in TerminalPanel |
| @xterm/addon-fit | 0.11.0 | `fitAddon.fit()` after font size change to reflow terminal | Already installed, already used in TerminalPanel |
| React | 19.2.4 | `fontSizes` state in App.tsx; prop drilling to TerminalPanel | Already used for all tab state |
| Vitest | 4.1.0 | Unit tests verifying source-level key handler and font size logic | Already configured |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jsdom | 29.0.0 | Simulates DOM in Vitest test environment | All component tests already use this |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `attachCustomKeyEventHandler` | `window` keydown listener | Window listener fires regardless of which terminal is focused; `attachCustomKeyEventHandler` is scoped to the specific terminal instance, correct isolation |
| Props for font size | `useRef` or `useContext` | Ref mutation does not trigger re-render; Context adds indirection for a small per-tab value. Props keep the pattern consistent with existing per-tab state (`webEnabled`, `sessionURLs`) |
| `term.options.fontSize = N` | Dispose + recreate terminal | Recreation destroys PTY output buffer; options mutation is instant and preserves state |

**Installation:** No new packages required.

## Architecture Patterns

### Relevant Project Structure
```
frontend/src/
├── App.tsx                          # MODIFY — add fontSizes state, pass fontSize + onFontSizeChange to TerminalPanel
├── components/
│   ├── TerminalPanel.tsx            # MODIFY — accept fontSize prop, attachCustomKeyEventHandler, apply font on isActive change
│   ├── __tests__/
│   │   └── TerminalPanel.test.tsx   # MODIFY — add source-inspection tests for key handler and font size logic
```

### Pattern 1: Per-Tab Font Size State in App.tsx
**What:** `fontSizes: Record<string, number>` mirrors the existing per-tab state pattern (`webEnabled`, `sessionURLs`, `sessionStatuses`).
**When to use:** Any per-tab value that needs to survive tab switching and be independent per session.
**Example:**
```typescript
// Source: mirrors existing webEnabled pattern in App.tsx
const DEFAULT_FONT_SIZE = 14  // matches current TerminalPanel hardcoded value

const [fontSizes, setFontSizes] = useState<Record<string, number>>({})

// Getter with fallback:
const getFontSize = (sessionId: string) => fontSizes[sessionId] ?? DEFAULT_FONT_SIZE

// Callback passed to TerminalPanel:
const handleFontSizeChange = useCallback((sessionId: string, delta: number) => {
  setFontSizes((prev) => {
    const current = prev[sessionId] ?? DEFAULT_FONT_SIZE
    const next = Math.max(6, Math.min(32, current + delta))
    return { ...prev, [sessionId]: next }
  })
}, [])

// In TerminalPanel render:
<TerminalPanel
  sessionId={tab.sessionId}
  isActive={isActive}
  relayPort={relayPort}
  fontSize={getFontSize(tab.sessionId)}
  onFontSizeChange={(delta) => handleFontSizeChange(tab.sessionId, delta)}
/>
```

### Pattern 2: Key Event Interception in TerminalPanel
**What:** `attachCustomKeyEventHandler` is called once during terminal setup. It intercepts SHIFT+= and SHIFT+- events, calls the `onFontSizeChange` callback, and returns `false` to suppress PTY injection.
**When to use:** Any keyboard shortcut that should be consumed by the app layer, not forwarded to the PTY.
**Critical detail:** Return value `false` = suppress PTY forwarding; `true` = pass through to terminal normally.
**Example:**
```typescript
// Source: @xterm/xterm 6.0.0 typings/xterm.d.ts
term.attachCustomKeyEventHandler((ev: KeyboardEvent): boolean => {
  if (ev.type !== 'keydown') return true  // only intercept keydown, not keyup/keypress
  if (ev.shiftKey && ev.key === '=') {
    // SHIFT+= produces '+' on US keyboards
    onFontSizeChange(+1)
    return false  // suppress: do NOT send '+' to the PTY
  }
  if (ev.shiftKey && ev.key === '-') {
    // SHIFT+- produces '_' on US keyboards — but ev.key is '-' when shiftKey is true
    onFontSizeChange(-1)
    return false  // suppress: do NOT send character to the PTY
  }
  return true  // all other keys pass through normally
})
```

**Note on keyboard keys:** When SHIFT is held, `ev.key` reports the base key (`=` or `-`), not the shifted character (`+` or `_`). Always check `ev.shiftKey && ev.key === '='` — not `ev.key === '+'`.

### Pattern 3: Apply Font Size Change with Reflow
**What:** When the `fontSize` prop changes (detected via a dedicated `useEffect`), write to `terminal.options.fontSize` and immediately call `fitAddon.fit()`.
**Why fit() is mandatory:** `FitAddon.fit()` calculates cols/rows from `containerWidth / cellWidth`. `cellWidth` depends on font size. Without `fit()` after a size change, the terminal uses stale col/row counts — text wraps incorrectly and the terminal dims no longer match the container.
**Example:**
```typescript
// Source: @xterm/addon-fit 0.11.0 typings, @xterm/xterm 6.0.0 typings
// Separate effect — runs when fontSize prop changes AND terminal is mounted
useEffect(() => {
  if (!termRef.current || !fitAddonRef.current) return
  termRef.current.options.fontSize = fontSize
  // Must call fit() immediately after — FitAddon recalculates cols/rows
  // using the new cell dimensions from the updated font size.
  fitAddonRef.current.fit()
}, [fontSize])
```

### Anti-Patterns to Avoid
- **Checking `ev.key === '+'` for SHIFT+=:** `ev.key` reports the unshifted key value when `shiftKey` is true in xterm's key event model. Use `ev.shiftKey && ev.key === '='`.
- **Attaching a `window.addEventListener('keydown', ...)` instead of `attachCustomKeyEventHandler`:** Window listeners fire regardless of focus and do not suppress PTY injection — the character will still be written to the terminal.
- **Calling `fit()` only in the `isActive` effect:** Font size changes on an inactive tab (if possible) would not reflow. The `fontSize` effect should call `fit()` unconditionally; `fit()` is safe to call on hidden panels (it may produce zero dimensions but won't corrupt state).
- **Not clamping font size:** Without min/max bounds, repeated presses produce unusably tiny (< 6px) or enormous (> 32px) font. Clamp to `[6, 32]`.
- **Storing font size in TerminalPanel local state:** Local state is lost if the component unmounts. Per-tab font size belongs in `App.tsx` alongside other per-tab state.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Keyboard shortcut interception | `window.addEventListener` + `ev.preventDefault()` | `term.attachCustomKeyEventHandler` returning `false` | xterm routes keys to PTY after DOM events; window listener cannot prevent PTY injection; `attachCustomKeyEventHandler` runs inside xterm's pipeline before PTY write |
| Terminal reflow after resize | Manual col/row math | `fitAddon.fit()` | FitAddon already handles container dimension measurement + `term.resize()` correctly; manual math misses DPR scaling, scrollbar width, padding |

**Key insight:** The PTY injection problem is the core technical risk of this phase. `attachCustomKeyEventHandler` is the only API in xterm that prevents a keydown from reaching the PTY. `ev.preventDefault()` on DOM events does NOT stop xterm from sending the character.

## Common Pitfalls

### Pitfall 1: SHIFT+= Injects '+' into the PTY
**What goes wrong:** Pressing SHIFT+= increases font size visually but also types `+` into the running shell session.
**Why it happens:** `onData` in xterm fires for every key that passes through xterm's key processing pipeline. If `attachCustomKeyEventHandler` returns `true` (or is not registered for this key), xterm writes the key's character to the PTY.
**How to avoid:** `attachCustomKeyEventHandler` must return `false` for the SHIFT+= and SHIFT+- events. Return `true` for all other keys.
**Warning signs:** Shell history shows `++++++` or `______` characters; running commands show unexpected `+` prepended.

### Pitfall 2: Font Size Changes Cause Garbled/Clipped Terminal Output
**What goes wrong:** After a font size change, text overflows or is clipped; the terminal doesn't fill its container.
**Why it happens:** `FitAddon` calculated cols/rows based on the old font size. After `terminal.options.fontSize` changes, cell dimensions change but cols/rows are not updated until `fit()` is called.
**How to avoid:** Always call `fitAddonRef.current.fit()` immediately after `termRef.current.options.fontSize = newSize`. The two calls must be adjacent.
**Warning signs:** Terminal appears to have wrong line widths; text wraps mid-word; blank area appears at bottom or right edge of terminal.

### Pitfall 3: Font Size Does Not Persist When Switching Tabs
**What goes wrong:** Switching away from a tab and returning resets its font size to 14.
**Why it happens:** Font size stored in component-local state (`useState` inside `TerminalPanel`) is not preserved across visibility cycles if the component remounts. Or the `terminal.options.fontSize` is reset when the terminal setup effect re-runs.
**How to avoid:** Store font size in `App.tsx` as `fontSizes: Record<string, number>`. Apply it via a `useEffect([fontSize])` inside `TerminalPanel` — separate from the setup effect that runs only on `[sessionId]`.
**Warning signs:** `console.log` shows font size resetting to 14 on tab switch.

### Pitfall 4: ev.key Confusion for SHIFT Keys
**What goes wrong:** Handler checks `ev.key === '+'` and never matches, so font size never changes.
**Why it happens:** xterm passes the raw DOM `KeyboardEvent`. On a US keyboard with SHIFT held: `ev.key` is `'='` (the base key), `ev.shiftKey` is `true`. The shifted character `+` is NOT what `ev.key` contains.
**How to avoid:** Check `ev.shiftKey && ev.key === '='` for increase, `ev.shiftKey && ev.key === '-'` for decrease.
**Warning signs:** Key handler never fires; adding `console.log(ev.key)` shows `'='` not `'+'`.

### Pitfall 5: attachCustomKeyEventHandler Replaces Previous Handler
**What goes wrong:** If `attachCustomKeyEventHandler` is called multiple times (e.g., on prop change), only the last handler is active — previous handlers are silently replaced.
**Why it happens:** xterm holds a single custom key handler reference. Calling it again overwrites the previous.
**How to avoid:** Call `attachCustomKeyEventHandler` exactly once during terminal setup (inside the `[sessionId]` effect). Do not call it in the `[fontSize]` effect or elsewhere.
**Warning signs:** Rapid font-size changes stop working after component re-renders; adding a `console.log` shows the handler fires once then stops.

## Code Examples

Verified patterns from official sources and existing codebase:

### Full TerminalPanel Props Extension
```typescript
// Source: TerminalPanel.tsx current interface + new props
interface TerminalPanelProps {
  sessionId: string
  isActive: boolean
  relayPort: number
  fontSize: number             // NEW — controlled by App.tsx
  onFontSizeChange: (delta: number) => void  // NEW — called on SHIFT+= / SHIFT+-
}
```

### Key Handler Registration (inside terminal setup effect)
```typescript
// Source: @xterm/xterm 6.0.0 typings/xterm.d.ts — attachCustomKeyEventHandler
// Place after term.open(containerRef.current), before relay client setup
term.attachCustomKeyEventHandler((ev: KeyboardEvent): boolean => {
  if (ev.type !== 'keydown') return true
  if (ev.shiftKey && ev.key === '=') { onFontSizeChange(+1); return false }
  if (ev.shiftKey && ev.key === '-') { onFontSizeChange(-1); return false }
  return true
})
```

### Font Size Apply Effect (separate from setup effect)
```typescript
// Source: @xterm/xterm 6.0.0 terminal.options mutation pattern
// @xterm/addon-fit 0.11.0 fit() API
useEffect(() => {
  if (!termRef.current || !fitAddonRef.current) return
  termRef.current.options.fontSize = fontSize
  fitAddonRef.current.fit()
}, [fontSize])
```

### Font Size State in App.tsx
```typescript
// Source: mirrors webEnabled pattern in current App.tsx
const DEFAULT_FONT_SIZE = 14

const [fontSizes, setFontSizes] = useState<Record<string, number>>({})

const handleFontSizeChange = useCallback((sessionId: string, delta: number) => {
  setFontSizes((prev) => {
    const current = prev[sessionId] ?? DEFAULT_FONT_SIZE
    const next = Math.max(6, Math.min(32, current + delta))
    return { ...prev, [sessionId]: next }
  })
}, [])
```

### Cleanup on Tab Close (in handleCloseTab)
```typescript
// Remove font size entry when tab closes — mirrors sessionStatuses cleanup pattern
setFontSizes((prev) => { const n = { ...prev }; delete n[id]; return n })
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded `fontSize: 14` in Terminal constructor | Controlled `fontSize` prop + `terminal.options.fontSize` mutation | Phase 10 | Per-tab font size becomes independent and persistent |
| No keyboard shortcut interception | `attachCustomKeyEventHandler` for SHIFT+= / SHIFT+- | Phase 10 | Clean key interception without PTY injection |

**Deprecated/outdated:**
- Hardcoded `fontSize: 14` in TerminalPanel's `new Terminal({...})`: will become the fallback default only, applied from `App.tsx` state.

## Open Questions

1. **Should font size reset when a session is closed and re-created with the same CLI?**
   - What we know: `fontSizes` is keyed by `sessionId` (a UUID). New sessions get new IDs.
   - What's unclear: Whether "same slot" memory is expected.
   - Recommendation: No persistence needed — a fresh session always starts at 14px. Closing a tab removes its entry from `fontSizes`.

2. **Non-US keyboard layouts: will SHIFT+= / SHIFT+- match correctly?**
   - What we know: `ev.key` is layout-aware. On UK keyboards, `=` is on a different key.
   - What's unclear: Full cross-layout coverage.
   - Recommendation: Implement for US layout first (SHIFT+`=` and SHIFT+`-`). Document in plan as known gap. Broad keyboard support is a future enhancement, not in scope for TERM-02/03/04.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` (test: { environment: 'jsdom' }) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TERM-02 | Source contains attachCustomKeyEventHandler with shiftKey && key === '=' | unit (source inspection) | `pnpm test -- TerminalPanel` | ✅ (file exists, new cases needed) |
| TERM-02 | Source applies fontSize prop to terminal.options.fontSize | unit (source inspection) | `pnpm test -- TerminalPanel` | ✅ (file exists, new cases needed) |
| TERM-03 | Source contains handler for shiftKey && key === '-' | unit (source inspection) | `pnpm test -- TerminalPanel` | ✅ (file exists, new cases needed) |
| TERM-04 | Key handler returns false for SHIFT+= and SHIFT+- (PTY suppression) | unit (source inspection) | `pnpm test -- TerminalPanel` | ✅ (file exists, new cases needed) |
| TERM-04 | App.tsx source contains fontSizes Record and handleFontSizeChange | unit (source inspection) | `pnpm test -- App` (new file) | ❌ Wave 0 |

**Note on test strategy:** `TerminalPanel` mounts xterm.js which requires a real canvas context — jsdom does not support it. The established project pattern (see existing `TerminalPanel.test.tsx`) is to import the source file as `?raw` string and assert on code patterns. This works for verifying the key handler logic, options mutation, and fit() call co-location. Behavioral xterm tests (actually firing key events) require a real browser — those are manual UAT, not automated.

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] New test cases in `frontend/src/components/__tests__/TerminalPanel.test.tsx` — covers TERM-02, TERM-03, TERM-04 via `?raw` source inspection
- [ ] New test file `frontend/src/components/__tests__/App.test.tsx` — covers TERM-04 (fontSizes state in App.tsx) via `?raw` source inspection

*(Vitest, jsdom, and React testing infrastructure already configured. No new framework setup needed.)*

## Sources

### Primary (HIGH confidence)
- Direct read of `/Users/ken/dev/agenthub/frontend/node_modules/@xterm/xterm/typings/xterm.d.ts` — `attachCustomKeyEventHandler` API, `terminal.options.fontSize` mutation pattern, return value semantics
- Direct read of `/Users/ken/dev/agenthub/frontend/node_modules/@xterm/addon-fit/typings/addon-fit.d.ts` — `FitAddon.fit()` API
- Direct read of `/Users/ken/dev/agenthub/frontend/src/components/TerminalPanel.tsx` — current terminal setup, existing refs, existing fit pattern
- Direct read of `/Users/ken/dev/agenthub/frontend/src/App.tsx` — existing per-tab state patterns (`webEnabled`, `sessionURLs`, `sessionStatuses`)
- Direct read of `/Users/ken/dev/agenthub/frontend/src/components/__tests__/TerminalPanel.test.tsx` — established `?raw` source inspection test pattern for xterm components
- Direct read of `/Users/ken/dev/agenthub/.planning/STATE.md` — explicit callout: "Phase 10 (Font size): Two interlocking pitfalls require explicit manual verification — key event suppression (attachCustomKeyEventHandler) and fit() call after every size change"
- Direct read of `/Users/ken/dev/agenthub/frontend/package.json` — confirmed @xterm/xterm 6.0.0, @xterm/addon-fit 0.11.0, no new packages required

### Secondary (MEDIUM confidence)
- None required — all critical information sourced directly from project files and installed package typings

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — confirmed from installed package.json + node_modules typings
- Architecture: HIGH — key handler API and options mutation pattern read directly from @xterm/xterm 6.0.0 type definitions; per-tab state pattern derived from existing App.tsx code
- Pitfalls: HIGH — two pitfalls explicitly called out in STATE.md; additional pitfalls derived from reading actual xterm API signatures and existing codebase patterns

**Research date:** 2026-03-19
**Valid until:** 2026-04-18 (stable — xterm.js and React already pinned in project)
