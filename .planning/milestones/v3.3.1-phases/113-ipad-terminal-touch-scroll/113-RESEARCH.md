# Phase 113: iPad terminal touch-scroll - Research

**Researched:** 2026-05-18
**Domain:** Frontend / xterm.js touch-event plumbing on iPad Safari + iPad Chrome
**Confidence:** HIGH on root-cause + chosen approach; MEDIUM on test-surface coverage; UAT path is physical-iPad-only

## Summary

xterm.js 6.0.0 ships with **zero touch handling** wired up to the terminal screen element. The `Gesture` singleton in `vs/base/browser/touch.ts` exists but is dormant — nothing in xterm calls `Gesture.addTarget()` or `Gesture.ignoreTarget()`, so the document-level touch listeners are never instantiated. The Viewport in v6 uses VSCode's `SmoothScrollableElement`, which listens only for trusted `wheel` events on the screen element. Touch drags on iPad therefore fall through to default browser behavior: iOS Safari pans the page because xterm's rendered content (canvas or DOM rows inside `.xterm-screen`) is not in a natively scrollable overflow region.

The CONTEXT.md root-cause statement ("xterm-helper-textarea captures touch events and prevents scrollback navigation") is **inaccurate at the DOM level** — that element is positioned at `left: -9999em; width: 0; height: 0; z-index: -5; opacity: 0` and cannot intercept visual touches. The observable bug (page panning instead of scrollback) is real; the mechanism is "no touch listeners exist anywhere on the terminal," not "an offscreen textarea is greedy."

Pure CSS `touch-action: pan-y` (Option A) **will not work** as the sole fix, because xterm's content is not in a native scrollable overflow — pan-y would let the iPad pan something, but the closest scrollable ancestor is the page/window, not the xterm buffer. Synthetic `WheelEvent` dispatch also doesn't help: untrusted wheel events don't trigger default scroll actions in any browser. We need **Option B: explicit `touchstart` / `touchmove` handlers on the terminal container that call `term.scrollLines(N)`**. A `touch-action: pan-y` CSS rule is still a useful **companion** to Option B — it suppresses iOS double-tap-to-zoom and signals to the browser that we're claiming the vertical drag gesture, reducing the chance of a competing browser-level pan animation fighting our handler.

**Primary recommendation:** Implement Option B (explicit touch handlers) with a `touch-action: pan-y` CSS companion on `.terminal-session-container`. Single tap on link cells continues to dispatch a synthetic click via the existing OSC 8 / WebLinks path (tap is `touchstart`+`touchend` with sub-threshold movement; we ignore those and only call `scrollLines` once cumulative movement exceeds one cell height).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Touch event capture on terminal container | Browser (DOM event listeners) | — | iPad delivers `TouchEvent` only to a real DOM node; xterm.js has no built-in hook for this in v6 |
| Touch-to-scrollback translation | Frontend (TerminalPanel.tsx) | xterm.js public API | `scrollLines(N)` is the only public scroll primitive; deltaY → lines conversion happens in our React component |
| Mouse-wheel scrollback (existing) | xterm.js internal (vscode SmoothScrollableElement) | — | Unchanged; lives entirely inside xterm |
| OSC 8 link tap | xterm.js WebLinksAddon (existing) | TerminalPanel.tsx handler | Already wired; touch must not preventDefault on taps below the drag threshold |
| Two-finger pan suppression on terminal | Browser (touch-action CSS) | Frontend touch handler (ignore multi-touch) | CSS `touch-action: pan-y` declares single-finger-vertical as our gesture; multi-touch we bail out and let browser/iOS handle |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/xterm` | 6.0.0 | Terminal emulator | Already shipped; no upgrade needed `[VERIFIED: npm view @xterm/xterm version → 6.0.0; matches frontend/package.json]` |
| `@xterm/addon-web-links` | 0.12.0 | OSC 8 / URL detection + click handler | Already shipped; tap-on-link path lives in TerminalPanel.tsx WebLinks handler `[VERIFIED: npm view @xterm/addon-web-links version → 0.12.0]` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none) | — | — | Pure DOM `addEventListener('touchstart'/'touchmove'/'touchend', …, { passive: false })` from React's `useEffect` is sufficient. No new dependency. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Plain DOM `addEventListener` | Hammer.js / @use-gesture/react | More code, new dep, dead-weight for what is ~30 lines of `touchmove` math. Rejected. |
| Manual `scrollLines` math | Synthesize `WheelEvent` and dispatch on screen element | Untrusted wheel events do not trigger default scroll action `[CITED: developer.mozilla.org/Web/API/WheelEvent — "Untrusted wheel events never cause any default action"]`. Rejected — would silently do nothing. |
| Explicit JS handler | Pure CSS `touch-action: pan-y` only | xterm content lives inside `.xterm-screen` (canvas/DOM rows), not in a natively scrollable overflow. `pan-y` would scroll the page, not the buffer. Rejected as sole fix. |

**Installation:** No package changes required.

**Version verification:**
```
npm view @xterm/xterm version      → 6.0.0  (verified 2026-05-18)
npm view @xterm/addon-web-links version → 0.12.0  (verified 2026-05-18)
```

## Architecture Patterns

### System Architecture Diagram

```
iPad single-finger drag
        │
        ▼
    TouchEvent (touchstart, touchmove, touchend)
        │  delivered to DOM node under finger
        ▼
.terminal-session-container  ◄── React ref (TerminalPanel.tsx)
    │  CSS: touch-action: pan-y (declares gesture intent)
    │  JS:  useEffect attaches passive:false listeners
    ▼
TouchScrollHandler (new helper module OR inline in useEffect)
    │
    ├─ touchstart  → record startY, startTime, mark "tracking"
    ├─ touchmove   → if single touch + Δy ≥ cellHeight → scrollLines(-Δy / cellHeight), preventDefault
    │                if multi-touch → release tracking (let iOS pan)
    └─ touchend    → if sub-threshold movement → release without preventDefault
                     → TouchEnd propagates → WebLinksAddon click handler fires for taps on links
        │
        ▼
term.scrollLines(N)  ← xterm.js public API (typings/xterm.d.ts:1211)
        │
        ▼
xterm Viewport → SmoothScrollableElement → buffer.ydisp updates
        │
        ▼
Renderer (WebGL or DOM) repaints scrollback
```

### Recommended Project Structure
```
frontend/src/
├── components/
│   └── TerminalPanel.tsx                  # add touch-handler useEffect here
├── lib/
│   └── touchScrollHandler.ts              # NEW — pure function: (term, container) → cleanup
└── style.css                              # add `.terminal-session-container { touch-action: pan-y; }`
```

A standalone `lib/touchScrollHandler.ts` module is preferred over inline code because (a) it isolates the math + state machine for unit testing, (b) it matches the established pattern (`lib/openLink.ts`, `lib/urlSafety.ts`, `lib/webglProbe.ts`).

### Pattern 1: Touch handler attachment via useEffect

```typescript
// Source: TerminalPanel.tsx — new useEffect, mirrors the mount-effect lifecycle
// pattern at line 195 (single-shot, sessionId dependency, returns cleanup).

useEffect(() => {
  const container = containerRef.current
  const term = termRef.current
  if (!container || !term) return

  const cleanup = attachTouchScroll(container, term)
  return cleanup
}, [sessionId])
```

```typescript
// Source: frontend/src/lib/touchScrollHandler.ts — new file
// Encapsulates the touch state machine; pure function for testability.

import type { Terminal } from '@xterm/xterm'

const TAP_THRESHOLD_PX = 8  // anything below this we treat as a tap, not a drag
                            //  — must be < 30 to coexist with future Gesture (xterm
                            //  Gesture HOLD_DELAY uses 30px tap-window per touch.ts:194)

export function attachTouchScroll(container: HTMLElement, term: Terminal): () => void {
  let trackingId: number | null = null
  let lastY = 0
  let startY = 0
  let accumulatedDy = 0

  const onTouchStart = (e: TouchEvent): void => {
    if (e.touches.length !== 1) {
      // Multi-touch — release; let iOS handle pinch / two-finger pan.
      trackingId = null
      return
    }
    const t = e.changedTouches[0]
    trackingId = t.identifier
    startY = t.clientY
    lastY = t.clientY
    accumulatedDy = 0
  }

  const onTouchMove = (e: TouchEvent): void => {
    if (trackingId === null) return
    if (e.touches.length !== 1) {
      trackingId = null
      return
    }
    const t = Array.from(e.changedTouches).find(x => x.identifier === trackingId)
    if (!t) return

    const dy = t.clientY - lastY
    lastY = t.clientY
    accumulatedDy += dy

    // Read live cell height (theme / font-size changes invalidate cached values).
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const cellH: number = (term as any)._core?._renderService?.dimensions?.css?.cell?.height ?? 17
    const lines = Math.trunc(accumulatedDy / cellH)
    if (lines !== 0) {
      // scrollLines: positive=down, negative=up. Finger drag DOWN should reveal
      // older content (scroll UP in xterm terminology) → negate.
      term.scrollLines(-lines)
      accumulatedDy -= lines * cellH
      e.preventDefault()  // suppress page-pan once we know we're scrolling
    }
  }

  const onTouchEnd = (e: TouchEvent): void => {
    // Sub-threshold movement (a tap) is left alone: do NOT preventDefault.
    // The WebLinksAddon's synthetic click handler must still receive its click.
    const totalDelta = Math.abs(lastY - startY)
    if (totalDelta < TAP_THRESHOLD_PX) {
      // Pure tap — no-op; let the addon receive its event.
    }
    trackingId = null
  }

  container.addEventListener('touchstart', onTouchStart, { passive: true })
  // touchmove must be non-passive so we can preventDefault on confirmed scroll.
  container.addEventListener('touchmove', onTouchMove, { passive: false })
  container.addEventListener('touchend', onTouchEnd, { passive: true })
  container.addEventListener('touchcancel', onTouchEnd, { passive: true })

  return () => {
    container.removeEventListener('touchstart', onTouchStart)
    container.removeEventListener('touchmove', onTouchMove)
    container.removeEventListener('touchend', onTouchEnd)
    container.removeEventListener('touchcancel', onTouchEnd)
  }
}
```

### Pattern 2: CSS companion

```css
/* style.css — add to .terminal-session-container rule (~line 20) */
.terminal-session-container {
  padding: 8px;
  overflow: hidden;
  position: relative;
  /* Phase 113 UI-03: declare single-finger vertical drag as ours, so iOS
     Safari doesn't compete with the touchmove handler to pan the page.
     Two-finger gestures (pinch-zoom) still get default browser handling
     (browser intersects with ancestors). */
  touch-action: pan-y;
}
```

iOS Safari 13+ has full `touch-action` support `[CITED: caniuse.com/css-touch-action — "Versions 13–26.5: Supported"]`. The legacy iOS 9.3–12.5 partial support (auto/manipulation only) is a non-concern: v3.3 UAT-04 hardware runs iOS 16+.

### Anti-Patterns to Avoid

- **Synthesizing WheelEvent**: dispatching a synthetic `WheelEvent` on `.xterm-screen` will NOT trigger xterm's `SmoothScrollableElement` scroll because untrusted events bypass the default scroll action. Don't go this route — looks correct, silently fails.
- **Attaching to `.xterm-helper-textarea`**: the textarea is at `left: -9999em; width: 0; height: 0` per `@xterm/xterm/css/xterm.css:61-77`. It's not the touch target on the visible terminal. Attaching there does nothing visible.
- **Touch-action: none**: would block all gestures including two-finger pinch-zoom, breaking accessibility. Use `pan-y` to whitelist the gestures we don't claim.
- **`passive: true` on touchmove**: cannot call `preventDefault()` on passive listeners. We need to suppress competing page-pan once we've detected a drag, so touchmove must be `passive: false`.
- **Eager `preventDefault` in `touchstart`**: blocks the tap-to-click path. Don't preventDefault until movement exceeds tap threshold.
- **Caching `cellHeight` at mount**: font-size changes (the SHIFT+= / SHIFT+- intercept at TerminalPanel.tsx:258) and theme changes mutate cell dimensions. Read fresh from `term._core._renderService.dimensions.css.cell.height` each touchmove (already-private path is used elsewhere in the file — `fitTerminal()` line 29).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Scroll the xterm buffer programmatically | DOM scroll manipulation, scroll-into-view, manual buffer rewrites | `term.scrollLines(N)` | Public API since xterm 2.x; integrates with smooth-scroll, alt-buffer guards, ydisp/buffer state |
| Detect single-vs-multi finger | Custom pointer-event tracking | `e.touches.length` check | TouchEvent API has this for free; no library needed |
| Tap-vs-drag disambiguation | Reuse `Gesture` singleton from xterm internals | Local threshold check (8px) | xterm's `Gesture` is dormant in v6 and uses 30px window for tap which is too loose for terminal cells (~9px wide). Local threshold is simpler and matches the cell-row resolution we need anyway |
| Inertial / "ballistic" scrolling | Velocity tracker + RAF loop | (defer) | Not in UI-03/UI-04 acceptance criteria. Issue #594 documents that xterm's architecture (canvas under DOM rows) makes inertia hard. **Out of scope for v3.3.1.** |

**Key insight:** xterm.js's lack of touch handling is well-known (Issues #594, #1007, #3613, #5377 all open or duplicates of "no touch") and the maintainers have not fixed it in v6 because the v6 Viewport rewrite chose `SmoothScrollableElement` (vscode component) which only listens for wheel. We are not "hand-rolling around" a library deficiency — we are filling a documented gap with the publicly-supported `scrollLines` primitive.

## Runtime State Inventory

> Phase is a pure code addition (new useEffect + new module + CSS rule). No rename, refactor, or migration. Runtime state inventory N/A.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — verified by inspecting CONTEXT.md scope ("Pure frontend bug. Web only — Wails desktop has no touchscreen path.") | — |
| Live service config | None | — |
| OS-registered state | None | — |
| Secrets/env vars | None | — |
| Build artifacts | None (Vite rebuild picks up TS + CSS changes automatically) | — |

## Common Pitfalls

### Pitfall 1: touchmove handler accidentally suppresses tap-on-link

**What goes wrong:** Calling `e.preventDefault()` on every touchmove fires even for a sub-threshold tap (touchmove can fire once or twice between touchstart and touchend even with no real drag motion). If preventDefault has been called, iOS Safari may suppress the synthetic `click` event, breaking the OSC 8 link tap.

**Why it happens:** iOS Safari's heuristic for "fire click after touchend" includes "no preventDefault was called on the touch sequence."

**How to avoid:** Only call `preventDefault()` once `accumulatedDy >= cellHeight` (i.e., we've decided this is a scroll, not a tap). Below threshold: do nothing. This is encoded in Pattern 1 above — `preventDefault` is inside the `if (lines !== 0)` arm.

**Warning signs:** UAT-04 carry-over fails: tapping an OSC 8 link in scrollback does nothing. Console shows no error.

### Pitfall 2: Stale `cellHeight` after font-size / theme change

**What goes wrong:** Caching the cell height at mount time means after a SHIFT+= font bump, one finger-distance unit no longer equals one cell row — drag-scrolling becomes too fast or too slow.

**Why it happens:** Cell dimensions are recomputed inside xterm's `RenderService` on font-size and theme change.

**How to avoid:** Read `term._core._renderService.dimensions.css.cell.height` inside `touchmove` (live), not in `touchstart` (cached). The `_core` accessor is already used elsewhere in TerminalPanel.tsx (`fitTerminal`, line 29) — this is the established escape hatch for v6's reduced public API.

### Pitfall 3: Multi-touch pinch-zoom is suppressed unintentionally

**What goes wrong:** Two-finger gesture on the terminal (zoom) triggers our single-finger handler with stale state, scrolls wildly, then iOS gives up on the zoom.

**Why it happens:** `touchstart` records `trackingId` from `changedTouches[0]` without re-checking `e.touches.length`.

**How to avoid:** First check in both `touchstart` and `touchmove`: `if (e.touches.length !== 1) { trackingId = null; return }`. Released tracking → no preventDefault → iOS gets to handle the pinch.

### Pitfall 4: Desktop browsers with touch-emulated DevTools accidentally route through this code

**What goes wrong:** Chrome DevTools "toggle device toolbar" emulates touch events while still firing real mouse events; in some configurations both fire, our handler scrolls AND mouse-wheel scrolls.

**Why it happens:** DevTools touch-emulation does not perfectly mirror real-device behavior. UAT must be on the physical iPad — but devs in Chrome DevTools toggling touch-emulation might see this.

**How to avoid:** Accept this as a DevTools-emulation-only artifact. UAT spec calls out physical hardware. Document for dev hygiene only — not a code change. Desktop browsers without touch hardware never trigger the handler.

### Pitfall 5: React 19 effect cleanup ordering vs term.dispose()

**What goes wrong:** Mount-effect cleanup at TerminalPanel.tsx:290 calls `term.dispose()` which removes the entire `.xterm` subtree from the container. If our new touchscroll-effect runs cleanup AFTER the mount-effect, the listeners are removed from a container that's already empty of xterm DOM — but the container itself still exists (we only remove from `containerRef.current`, which is the React-owned outer `<div ref=…>`, not the xterm subtree). Safe, but the ordering deserves to be documented in the code comment.

**Why it happens:** React effect cleanup order is reverse of registration order, but both effects depend on `sessionId` — they'll both fire cleanup together on session change. The container `<div>` itself is a React-owned ref and persists across xterm re-mount.

**How to avoid:** Attach the listeners to `containerRef.current` (the outer React div), not to anything inside `.xterm` / `.xterm-screen`. The container outlives `term.dispose()`. Pattern 1 above already does this correctly.

## Code Examples

Already documented above in **Architecture Patterns / Pattern 1 + Pattern 2**. The `attachTouchScroll` function and the CSS rule are the complete implementation surface.

Reference for `term.scrollLines` signature:
```typescript
// Source: frontend/node_modules/@xterm/xterm/typings/xterm.d.ts:1207-1211
/**
 * Scroll the display of the terminal
 * @param amount The number of lines to scroll down (negative scroll up).
 */
scrollLines(amount: number): void;
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| xterm.js v4/v5 had `.xterm-viewport` with native `overflow-y: scroll` that iOS panned correctly when content overflowed | xterm.js v6 swapped to vscode `SmoothScrollableElement` — `.xterm-viewport` overflow still set but no longer contains scrollable content | xterm 6.0.0 release (Viewport.ts rewrite) | iPad scroll silently broke for any v5→v6 upgrader |
| Touch handlers in xterm core | Dormant `Gesture` singleton, no callers | Same v6 rewrite | Consumers must add their own touch handlers |

**Deprecated/outdated:**
- The old advice "set scrollback high and let `.xterm-viewport` overflow do the work" — only works on v4/v5. Does not work on v6.
- Synthesizing `WheelEvent` to drive xterm scroll — never worked (untrusted events).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | iPad Chrome (which is WebKit-based / Blink-based depending on iPadOS 17+) handles `touch-action: pan-y` identically to iPad Safari | CSS companion section | Low — both use WKWebView underneath on iPadOS; if it diverges, the fix degrades to "JS handler works, browser may double-pan." UAT will catch. |
| A2 | xterm.js will not gain native touch support during v3.3.1 timeframe (so our handler will not collide with theirs) | Don't Hand-Roll | Low — checked v6.0.0 latest; Issue #5377 still open with no PR. |
| A3 | The `_core._renderService.dimensions.css.cell.height` private path remains stable in xterm 6.0.0 | Pattern 1 | Low — same path already in production at TerminalPanel.tsx:29 (`fitTerminal`), so a v6 patch breaking it would break us with or without this phase. |
| A4 | `touchmove` with `passive: false` on the container, when no drag is in progress, has no measurable performance impact on iPad | Pitfall 1 mitigation | Very low — single React-owned div; passive:false only matters at preventDefault-call time. |

## Open Questions

1. **Should we add a small velocity tracker for inertia ("ballistic" scrolling)?**
   - What we know: Issue #594 documents xterm's architectural difficulty with inertia (canvas under row divs); CONTEXT.md does not require it; UI-03 says "matching desktop wheel-scroll behavior" — desktop wheel-scroll is not inertial either.
   - What's unclear: whether iPad users will find non-inertial scroll feel "broken" vs. desktop parity.
   - Recommendation: **Skip for v3.3.1.** Non-inertial = honest parity with desktop wheel. If users complain, file a v3.4 enhancement issue. Documented in "Don't Hand-Roll" as deferred.

2. **Does the existing OSC 8 / WebLinksAddon click handler actually fire on iPad today (pre-fix)?**
   - What we know: `STATE.md:44` says "iPad tap-on-link captured by xterm-helper-textarea instead of firing link click handler. Pre-existing iPad-touch polish cluster. (UAT-04)." This implies it **currently does not fire** on iPad — the "must not regress" framing in CONTEXT.md is aspirational, not a current baseline.
   - What's unclear: whether the Phase 113 fix incidentally repairs the tap path (a tap below 8px threshold lets the touch sequence reach the WebLinks handler with no preventDefault — potentially fixing UAT-04 as a free side-effect) OR whether tap-on-link needs a separate fix and we just need to confirm "Phase 113 does not make it worse."
   - Recommendation: **Test both during physical iPad UAT.** Add a UAT step for "tap on a known-good `https://` link in scrollback → confirm link opens in new tab (or popover fires)." If it works post-fix, note as bonus fix; if it doesn't work pre OR post, document as still-broken-but-not-regressed.

3. **Does macOS executor have any way to validate this fix programmatically?**
   - What we know: vitest + jsdom — jsdom's TouchEvent support is limited (Issue jsdom#1508 still open). The existing TerminalPanel test pattern (`?raw` import + string-grep on source) is the established way around jsdom gaps for this project (`TerminalPanel.test.tsx`).
   - What's unclear: whether a string-grep test that asserts "the touch handler exists and calls scrollLines" is the right granularity, or whether we should mock a TouchEvent-like object with `touches: [{ clientY }]` and `dispatchEvent` it on a mocked container.
   - Recommendation: **Both, in different files.** See "Test Surface" section below.

## Environment Availability

> Phase requires no new external tools — Node/pnpm/vitest already present.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Vite build, vitest | ✓ | (project-pinned via volta/pnpm — already in use) | — |
| pnpm | Frontend deps | ✓ | (already in use) | — |
| vitest | Unit tests | ✓ | 4.1.0 | — |
| jsdom | Test DOM env | ✓ | 29.0.0 | TouchEvent simulation via mocked event objects |
| Physical iPad | UAT-04 carryover + UI-03 verification | external (operator hardware) | iOS 16+ assumed | none — UAT cannot be automated |

**Missing dependencies with no fallback:** Physical iPad for UAT. Same hardware as v3.3 UAT-04.

**Missing dependencies with fallback:** TouchEvent simulation in jsdom — fallback is mocked event objects (see Test Surface below).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 + jsdom 29.0.0 |
| Config file | `frontend/vitest.config.ts` (exists; verified via project conventions) |
| Quick run command | `pnpm --filter frontend test -- TerminalPanel.touchscroll` |
| Full suite command | `pnpm --filter frontend test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UI-03 | Touch handler attached to terminal container; reads cell height live; calls `term.scrollLines(-N)` on Δy ≥ cellH | unit (string-grep `?raw`) | `pnpm --filter frontend test TerminalPanel.touchscroll.source` | ❌ Wave 0 |
| UI-03 | `attachTouchScroll(container, term)` pure function: synthetic TouchEvent sequence → mocked `term.scrollLines` called with expected arg | unit (mock TouchEvent) | `pnpm --filter frontend test touchScrollHandler` | ❌ Wave 0 |
| UI-03 | iPad Safari + iPad Chrome — single-finger drag scrolls scrollback | manual UAT (physical iPad) | (none) | n/a — physical hardware |
| UI-03 | Two-finger drag on terminal does NOT trigger our handler (lets browser pinch-zoom) | unit (mock TouchEvent with 2 touches → assert scrollLines NOT called) | `pnpm --filter frontend test touchScrollHandler` | ❌ Wave 0 |
| UI-04 | Desktop mouse-wheel still scrolls (no regression) | manual smoke (Chrome desktop) | (none) | n/a — visual confirm |
| UI-04 | `.terminal-session-container` has `touch-action: pan-y` in style.css | unit (CSS grep) | `pnpm --filter frontend test TerminalPanel.touchscroll.source` | ❌ Wave 0 |
| UI-04 | Sub-threshold tap (< 8px movement) does NOT call `preventDefault` (preserves OSC 8 click path) | unit (mock TouchEvent + spy on preventDefault) | `pnpm --filter frontend test touchScrollHandler` | ❌ Wave 0 |
| UI-04 | OSC 8 / web-link tap-on-link path: physical iPad — tap a link, link opens | manual UAT (physical iPad) | (none) | n/a — physical hardware |

### Sampling Rate
- **Per task commit:** `pnpm --filter frontend test -- touchScrollHandler TerminalPanel.touchscroll`
- **Per wave merge:** `pnpm --filter frontend test`
- **Phase gate:** Full suite green + manual iPad UAT before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/TerminalPanel.touchscroll.test.tsx` — string-grep (`?raw`) tests that assert the touch-handler useEffect exists, references `scrollLines`, attaches with `passive: false` for touchmove. Mirrors the existing `TerminalPanel.test.tsx` style.
- [ ] `frontend/src/lib/__tests__/touchScrollHandler.test.ts` — unit tests for the pure `attachTouchScroll` function: mock Terminal + container, synthesize TouchEvent-like objects (since jsdom's TouchEvent is incomplete, build raw `{ type, touches, changedTouches }` objects and call the handler directly OR via `dispatchEvent(new Event('touchstart'))` then patch in custom properties).
- [ ] CSS grep test inside `TerminalPanel.touchscroll.test.tsx` confirms `touch-action: pan-y` rule is present on `.terminal-session-container`.

*(Framework already installed; no install step required.)*

## Security Domain

> Default `security_enforcement: true` (not overridden in config.json).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Phase only touches local DOM event handling |
| V3 Session Management | no | No session-token plumbing changed |
| V4 Access Control | no | Browser-local UI behavior; no auth surface |
| V5 Input Validation | yes (lightweight) | TouchEvent `touches[i].clientY` is a number from a trusted browser API; we use it in arithmetic only (no eval, no DOM injection, no fetch). Math.trunc + bounded by clientY (small int) — no overflow risk. |
| V6 Cryptography | no | No crypto |

### Known Threat Patterns for Frontend / React + xterm

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Touch-event handler is a footgun for XSS-via-link if it accidentally upgrades a tap to "follow URL without WebLinks risk check" | Tampering | We do NOT bypass the existing WebLinksAddon click path; we only call `scrollLines`. Tap-on-link still routes through `getRisk` + `LinkConfirmPopover` (TerminalPanel.tsx:516-525). No new attack surface. |
| Synthetic events bypassing security checks | Tampering | Inert — we do not synthesize any events. We consume real browser TouchEvents and call a public xterm API. |

**Conclusion:** No new security surface. The fix is mechanically isolated to DOM event listeners on a React-owned `<div>` and one CSS property.

## Sources

### Primary (HIGH confidence)
- xterm.js source code, local checkout:
  - `frontend/node_modules/@xterm/xterm/src/browser/CoreBrowserTerminal.ts:436-441` — confirms `.xterm-helper-textarea` is appended to `_helperContainer`, an offscreen helper
  - `frontend/node_modules/@xterm/xterm/css/xterm.css:61-77` — confirms textarea is at `left: -9999em`, `width: 0`, `height: 0`, `z-index: -5`
  - `frontend/node_modules/@xterm/xterm/src/browser/Viewport.ts` — confirms v6 uses `SmoothScrollableElement`, no touch listeners attached
  - `frontend/node_modules/@xterm/xterm/src/vs/base/browser/touch.ts` — confirms `Gesture` is a dormant singleton; no `addTarget` callers in xterm
  - `frontend/node_modules/@xterm/xterm/src/vs/base/browser/ui/scrollbar/scrollableElement.ts:387` — confirms only `MOUSE_WHEEL` listener, no touch
  - `frontend/node_modules/@xterm/xterm/typings/xterm.d.ts:1207-1233` — confirms `scrollLines`, `scrollPages`, `scrollToTop`, `scrollToBottom`, `scrollToLine` are public API
- AgentHub source:
  - `frontend/src/components/TerminalPanel.tsx:801-841` — current `.terminal-session-container` JSX; the mount point for our useEffect
  - `frontend/src/style.css:20-27` — current `.terminal-session-container` CSS rule
  - `frontend/src/components/TerminalPanel.tsx:489-558` — existing WebLinksAddon click handler (the OSC 8 / link tap path that must not regress)
  - `frontend/src/components/__tests__/TerminalPanel.test.tsx:1-30` — established `?raw` + CSS-grep test pattern
- xterm.js public docs: https://xtermjs.org/docs/api/terminal/interfaces/iterminaloptions/
- MDN: https://developer.mozilla.org/en-US/docs/Web/CSS/touch-action — confirms `pan-y` semantics + iOS Safari 13+ support
- caniuse: https://caniuse.com/css-touch-action — confirms iOS Safari 13+ full support

### Secondary (MEDIUM confidence)
- GitHub issues confirming "xterm has no built-in touch support, custom handler required":
  - https://github.com/xtermjs/xterm.js/issues/5377 — "Limited touch support on mobile devices" (open, help wanted)
  - https://github.com/xtermjs/xterm.js/issues/594 — "Support ballistic scrolling via touch" (closed, duplicate)
  - https://github.com/xtermjs/xterm.js/issues/3613 — "[iOS] Scroll problems if you starts on text" (open, duplicate)
  - https://github.com/xtermjs/xterm.js/issues/1007 — "Touch scrolling should send arrow keys" (open)
- AgentHub Issue #56: https://github.com/scottkw/agenthub/issues/56 — confirms the bug, mentions suggested approach (touch handlers OR touch-action)
- jsdom TouchEvent gap: https://github.com/jsdom/jsdom/issues/1508 — confirms unit testing needs mocked event objects

### Tertiary (LOW confidence)
- WebKit `touch-action` partial-support warnings (iOS 9.3–12.5): irrelevant for v3.3.1's iPad UAT hardware (iOS 16+). Flagged in Assumption A1.

## Metadata

**Confidence breakdown:**
- Root cause (xterm v6 has no touch handlers; helper-textarea claim is wrong): **HIGH** — direct source inspection
- Standard stack (no new deps; use built-in API): **HIGH** — typings + npm registry verified
- Architecture (Option B + CSS companion): **HIGH** — Option A definitively rejected via MDN/WheelEvent docs
- Pitfalls (tap-vs-drag, multi-touch, cell-height cache): **HIGH** — direct mapping from xterm internals
- Test surface (vitest unit tests viable with mocked events): **MEDIUM** — jsdom limitations require care, but the project has the `?raw` pattern as a fallback
- iPad UAT path: **HIGH** — same hardware/operator as v3.3 UAT-04

**Research date:** 2026-05-18
**Valid until:** 2026-06-17 (30 days; xterm 6.0.0 is stable, no major release expected)
