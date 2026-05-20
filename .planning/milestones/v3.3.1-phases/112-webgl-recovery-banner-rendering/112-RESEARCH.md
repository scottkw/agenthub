# Phase 112: WebGL recovery banner rendering - Research

**Researched:** 2026-05-18
**Domain:** React component lifecycle + xterm.js addon-webgl context-loss handler ordering
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **Root cause (suspected, per v3.3.1-ROADMAP.md Phase 112)** — `frontend/src/components/TerminalPanel.tsx:391-395`. The CONTEXT calls this "closure rot." This research **refutes** the closure-rot framing and identifies a different (but related) root cause inside the same five lines — see §1 below. The locked file region (391–395) is correct; the proposed pattern (`useRef` to keep latest setter fresh) is one valid fix, but is not necessary given React `useState` setter stability. A simpler fix exists and is recommended in §5.
- **DOM fallback still works** — independent path; not regressed by any fix in this phase.
- **Fix shape** — researcher's discretion between `useRef`-for-setter vs. effect-dep restructure vs. handler-reordering. Pick what fits the file. **Recommendation:** handler reordering + try/catch hardening (§5) — the pattern most consonant with the rest of `TerminalPanel.tsx`.
- **Verify with DevTools** — `WEBGL_lose_context.loseContext()` via the canvas; banner must appear in `.banner-stack` and auto-dismiss after 8s.

### Cross-surface verification (release gate)

- **Desktop:** Wails dev mode (`wails dev`) — prod app has no DevTools per project memory `project_wails_devtools_disabled_in_prod`. Web-share to Chrome is the alternative path.
- **Web:** browser at local web-share URL, DevTools, trigger context loss, observe banner.
- **DOM fallback regression smoke:** terminal content readable after loss event.
- **Single fix covers both surfaces** — same React tree.

### Claude's Discretion

- Choice of fix pattern (ref vs. reorder vs. try/catch) — recommended choice is in §5.
- Regression-test shape — recommended source-inspection + behavioral fixture in §7.

### Deferred Ideas (OUT OF SCOPE)

- xterm.js renderer fallback logic — already works.
- Banner styling / copy — locked by Phase 93 UI-SPEC.
- Other banner types in `.banner-stack`.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UI-01 | After `WEBGL_lose_context.loseContext()`, `document.querySelector('.webgl-recovery-banner')` is non-null; auto-dismisses 8s later | §1 (root cause), §2 (banner contract), §3 (.banner-stack render gating), §5 (fix pattern) |
| UI-02 | DOM fallback continues to work after WebGL loss; terminal content remains readable | §8 (fallback independence) |
</phase_requirements>

## Summary

Issue #55 reports that after `WEBGL_lose_context.loseContext()`:
- xterm's internal logs fire correctly (`webglcontextlost event received` then 3s later `webgl context not restored; firing onContextLoss`).
- DOM fallback works.
- But `webglContextLost` React state never flips → no banner.

The CONTEXT.md hypothesis is "closure rot on the banner-state setter." **This research refutes that specific framing** [VERIFIED: source read of `frontend/src/App.tsx:172-318` + React 19.2 useState semantics]: the App-level setter is a `useState` setter (stable by React contract), and `handleWebGLContextLost` is a `useCallback(..., [])` that only calls setters — there is no stale setter to rot.

The actual smoking gun is **call ordering inside the `onContextLoss` handler in `TerminalPanel.tsx:391-395`**:

```ts
webglAddon.onContextLoss(() => {
  webglAddon.dispose()             // ← (1) tears down emitters mid-fire
  webglAddonRef.current = null
  onWebGLContextLost?.('context-loss')   // ← (2) may never reach this line
})
```

`dispose()` on `WebglAddon` runs the disposable chain registered via `_register` [VERIFIED: `node_modules/@xterm/addon-webgl/src/WebglAddon.ts:84-97`]. One of those disposables resets the render service to a fresh DOM renderer mid-fire (`renderService.setRenderer((this._terminal as any)._core._createRenderer())` — `WebglAddon.ts:90-97`), and another disposes the `Emitter<void>` chain whose `.fire()` is still on the call stack. The forwarded event listener that *we* registered is itself a disposable owned by the emitter being torn down. The result, exactly matching Issue #55's observation, is that the synchronous call to our `onWebGLContextLost?.('context-loss')` line never executes — or executes against a partially-disposed addon and throws silently.

**Primary recommendation:** Reorder the handler so the React notify call fires **before** any disposal work, and defer the addon teardown to a microtask. Pattern is local to the five-line block and follows the same defensive-style other callbacks in the file already use (e.g., `try { webLinksAddonRef.current.dispose() } catch { /* ignore */ }` at line 320).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Detect WebGL context loss | Browser (xterm.js addon) | — | xterm listens on `canvas.webglcontextlost` directly; not our concern |
| Surface event to React tree | Frontend Server (React component — `TerminalPanel`) | — | The bridge from xterm's emitter to React state is `onContextLoss` callback in `TerminalPanel.tsx` |
| Banner state | Frontend Server (React component — `App.tsx`) | — | `webglContextLost`/`webglBannerDismissed` live at App level so banner-stack ordering works |
| Banner rendering | Frontend Server (React component — `WebGLRecoveryBanner`) | — | Pure presentational; 8s auto-dismiss timer is local to the component |
| DOM fallback rendering | Browser (xterm.js core) | — | xterm internally calls `renderService.setRenderer(_createRenderer())` on context loss; orthogonal to banner state |

**No cross-tier concerns.** This is a one-file fix inside the React presentation layer.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @xterm/xterm | 6.0.0 | Terminal emulator | Already in use |
| @xterm/addon-webgl | 0.19.0 | GPU-accelerated rendering | Already in use; current per npm (published 2025-12-22) [VERIFIED: `npm view @xterm/addon-webgl version`] |
| react | 19.2.4 | UI tree | Already in use; setState setters are guaranteed stable [CITED: React docs — useState setter identity] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| vitest | 4.1.0 | Unit/component test runner | Already used for all frontend tests [VERIFIED: `frontend/package.json:39`] |
| jsdom | 29.0.0 | DOM emulation for vitest | Already configured [VERIFIED: `vite.config.ts:8`] |
| react-dom/client + flushSync | 19.2.3 | Synchronous render for tests | Pattern already used in `WebGLRecoveryBanner.test.tsx` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Handler reorder + try/catch | useRef-for-setter pattern | Works but doesn't address the real root cause (call ordering vs. closure rot); adds unused indirection |
| Handler reorder + try/catch | Effect dep-array restructure | Doesn't fix the in-fire disposal problem either |

**No new installs required.** The fix is purely a code change in one file.

**Version verification:**
- `@xterm/addon-webgl@0.19.0` confirmed current on npm (published 2025-12-22) [VERIFIED: `npm view @xterm/addon-webgl version`].

## Architecture Patterns

### System Architecture Diagram

```
User triggers context loss in DevTools console
  │
  ▼
canvas.dispatchEvent('webglcontextlost')           [browser/xterm internals]
  │
  ▼
WebglRenderer event listener (line 110-121)        [xterm internals]
  │  • e.preventDefault()
  │  • setTimeout(3000) ────► fires _onContextLoss [3s LATER — note timing]
  │
  ▼
WebglAddon._onContextLoss emitter (Event.forward)  [xterm internals]
  │
  ▼
OUR HANDLER in TerminalPanel.tsx:391-395           [BUG SITE — fix here]
  │  Current (buggy) order:
  │    1. webglAddon.dispose() ◄── tears down emitter mid-fire
  │    2. webglAddonRef.current = null
  │    3. onWebGLContextLost?.('context-loss') ◄── never reached
  │
  │  Fixed order:
  │    1. onWebGLContextLost?.('context-loss') ◄── notify React first
  │    2. queueMicrotask(() => { addon.dispose(); ref = null }) ◄── defer
  │
  ▼
App.tsx: handleWebGLContextLost → setWebglContextLost(true)
  │
  ▼
App re-renders, condition (line 981 + 1015) is true
  │
  ▼
<WebGLRecoveryBanner reason='context-loss' /> mounts in .banner-stack
  │
  ▼
WebGLRecoveryBanner.tsx:36-40: useEffect setTimeout(8000)
  │
  ▼ 8 seconds later
onDismiss() → setWebglBannerDismissed(true) → banner unmounts
```

### Recommended Project Structure

No new files. Touch points:

```
frontend/src/components/
├── TerminalPanel.tsx                        # SOLE FIX SITE (lines 389-403)
└── __tests__/
    ├── TerminalPanel.hot-swap.test.tsx     # ADD: source-inspection tests for handler order
    └── TerminalPanel.test.tsx              # OR ADD: behavioral test using mock WebglAddon
```

### Pattern 1: React setState setter stability (refutes closure-rot hypothesis)

**What:** `useState` setters are guaranteed stable across renders.
**When to use:** When deciding whether a setter passed into a callback needs a ref wrapper.
**Why it matters here:** `setWebglContextLost` in `App.tsx:172` is `useState` setter; passed to `handleWebGLContextLost` (`App.tsx:312-318`) which is `useCallback(..., [])`; passed to `TerminalPanel` as `onWebGLContextLost`. None of these references can go stale.

```ts
// React docs (https://react.dev/reference/react/useState):
// "React guarantees that setState function identity is stable and will not
//  change on re-renders."
```

[CITED: https://react.dev/reference/react/useState — "setState function" section]

### Pattern 2: Event-emitter handler must not synchronously dispose the emitter

**What:** When a callback is invoked from inside an emitter's `.fire()`, calling `.dispose()` on a chain that includes that emitter from within the callback aborts the remaining work on the call stack.

**Verified mechanism:** [VERIFIED: read of `@xterm/addon-webgl/src/WebglAddon.ts:84-97`]

```ts
// In WebglAddon.activate():
this._register(Event.forward(this._renderer.onContextLoss, this._onContextLoss));
// ...
this._register(toDisposable(() => {
  // Runs on WebglAddon.dispose() — resets renderService to a fresh DOM renderer
  renderService.setRenderer((this._terminal as any)._core._createRenderer());
  renderService.handleResize(terminal.cols, terminal.rows);
}));
```

When our callback calls `webglAddon.dispose()`, the `Event.forward` and the `toDisposable` block above are both torn down while we are still on the emitter's call stack.

**Fix:** Defer disposal to a microtask or rAF so it runs **after** the emitter's `.fire()` returns.

```ts
// Source pattern (recommended):
const webglAddon = new WebglAddon()
webglAddon.onContextLoss(() => {
  // 1. Notify React FIRST (synchronous, no addon teardown yet).
  onWebGLContextLost?.('context-loss')
  // 2. Defer the addon dispose to a microtask so the emitter chain finishes
  //    firing cleanly before we tear it down. Pitfall: synchronous dispose
  //    inside the callback aborts subsequent listeners and can throw —
  //    Issue #55 root cause.
  queueMicrotask(() => {
    try { webglAddon.dispose() } catch { /* ignore — render service may already be torn down */ }
    if (webglAddonRef.current === webglAddon) {
      webglAddonRef.current = null
    }
  })
})
term.loadAddon(webglAddon)
webglAddonRef.current = webglAddon
```

[VERIFIED: this is the established defensive-dispose pattern already in use at `TerminalPanel.tsx:320` — `try { webLinksAddonRef.current.dispose() } catch { /* ignore */ }`]

### Anti-Patterns to Avoid

- **`useRef`-wrap a stable React setter** — adds indirection without fixing the real bug. The `useState` setter is already stable; wrapping it in a ref masks the actual ordering bug while paying complexity tax.
- **Disposing an event-emitter chain synchronously inside its own fire path** — current behavior, exactly what Issue #55 hits.
- **Calling `term.clear()` or `term.reset()` from the loss handler** — destroys scrollback (the Phase 93 contract explicitly forbids this; `TerminalPanel.hot-swap.test.tsx:54-60` source-asserts this).
- **Catching the dispose throw but leaving `onWebGLContextLost` unreached** — the React notify must happen before any disposal work that can throw.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 8s auto-dismiss timer | Custom setTimeout in App.tsx | `WebGLRecoveryBanner.tsx:36-40` already owns it | Component-local timer with cleanup on unmount [VERIFIED] |
| Banner stacking / ordering | Custom z-index logic | Existing `.banner-stack` flex container | Established Phase 81 vocabulary [VERIFIED: `App.tsx:984`] |
| Setter stability for React handlers | Manual ref wrapper for `useState` setters | React's built-in stability | React guarantees this [CITED: react.dev/useState] |
| DOM renderer fallback after WebGL loss | Custom restart logic | xterm.js handles internally | `WebglAddon.ts:90-97` already calls `renderService.setRenderer(_createRenderer())` on dispose [VERIFIED] |

**Key insight:** Every component this phase touches already exists and is correct. The bug is one ordering mistake in one five-line block.

## Common Pitfalls

### Pitfall 1: Mistaking xterm's internal logs for our handler running

**What goes wrong:** Issue #55 reports `webglcontextlost event received` and `webgl context not restored; firing onContextLoss` in the console, which suggests the chain is firing. But these logs come from xterm's `WebglRenderer.ts:111,118` — they print **before** our callback is invoked. Their presence does NOT prove our handler ran.

**Why it happens:** xterm logs eagerly; our `onContextLoss(() => {...})` is one of N subscribers fired by the emitter, after those logs.

**How to avoid:** Add a `console.debug('[TerminalPanel] onContextLoss callback fired')` as the first line of our handler during UAT to confirm reach. Remove before merge.

**Warning signs:** xterm logs present, but `document.querySelector('.webgl-recovery-banner')` is null.

### Pitfall 2: The 3-second delay confuses reproduction

**What goes wrong:** [VERIFIED: `WebglRenderer.ts:114-120`] xterm waits **3000ms** after the `webglcontextlost` browser event before firing `onContextLoss` (it's checking for context restore first). Testers who only watch for 1–2s and conclude "no banner" may have given up before the chain can fire.

**Why it happens:** Documented in xterm comments: "Wait a few seconds to see if the 'webglcontextrestored' event is fired. If not, dispatch the onContextLoss notification."

**How to avoid:** UAT script must wait **≥ 3.5s** between `loseContext()` call and the `document.querySelector` check.

**Warning signs:** Repro feels intermittent — sometimes the banner shows up "late," sometimes not at all.

### Pitfall 3: React 19 concurrent rendering does NOT cause stale state setters

**What goes wrong:** Reading "concurrent React" docs and assuming `setWebglContextLost` may be captured stale at mount. False.

**Why it happens:** Concurrent rendering re-orders renders, not setter identity. `useState` setters are identity-stable across all React versions.

**How to avoid:** Don't add `useRef` for setters unless verified necessary; see Pattern 1.

**Warning signs:** Fix involves `latestSetterRef = useRef(setX); useEffect(() => { latestSetterRef.current = setX })` — code smell here because `setX` is already stable.

### Pitfall 4: HMR can mask the bug in `wails dev`

**What goes wrong:** Vite HMR reloads `TerminalPanel.tsx` on save; the WebglAddon is re-attached fresh, the bug "fixes itself" between code changes, then re-appears on a clean reload.

**Why it happens:** A fresh addon instance with a fresh emitter chain hasn't accumulated the dispose-from-fire problem yet — but the bug is structural and will recur on every context-loss event.

**How to avoid:** Hard reload (Cmd-R) between tests; Issue #55 confirmed the bug persists after `location.reload()`.

**Warning signs:** Bug "fixed" by editing unrelated code, then returns.

### Pitfall 5: Two different banner-trigger paths (context-loss vs. software-rasterized)

**What goes wrong:** Fixing only the context-loss path leaves the software-rasterized startup path untouched. The startup path (`TerminalPanel.tsx:384-388`) calls `onWebGLContextLost?.('software-rasterized')` directly from the hot-swap effect — no emitter, no closure-rot concern, already works.

**Why it happens:** Two different code paths share one banner prop signature.

**How to avoid:** Don't refactor the software-rasterized path. Issue #55 is context-loss only.

**Warning signs:** Touching `TerminalPanel.tsx:384-388` in the fix diff.

## Code Examples

### The fix (recommended pattern)

```ts
// File: frontend/src/components/TerminalPanel.tsx
// Replace lines 389-395 (current onContextLoss registration).
// Source: this research, derived from defensive patterns at lines 320, 326.

try {
  const webglAddon = new WebglAddon()
  webglAddon.onContextLoss(() => {
    // Issue #55: notify React FIRST so the banner-state setter runs
    // synchronously while the emitter is still alive. Disposing the addon
    // synchronously here would tear down the emitter chain mid-fire (see
    // node_modules/@xterm/addon-webgl/src/WebglAddon.ts:84-97 — Event.forward
    // + renderer-reset disposables) and the React notify call below would
    // not be reached.
    onWebGLContextLost?.('context-loss')
    // Defer the addon teardown to a microtask: emitter fire returns first,
    // then we tear down cleanly. try/catch matches the established pattern
    // at line 320 (WebLinksAddon dispose).
    queueMicrotask(() => {
      try { webglAddon.dispose() } catch { /* ignore */ }
      if (webglAddonRef.current === webglAddon) {
        webglAddonRef.current = null
      }
    })
  })
  term.loadAddon(webglAddon)
  webglAddonRef.current = webglAddon
} catch (err) {
  console.warn(`[TerminalPanel] WebGL unavailable for session ${sessionId}:`, err)
}
```

### Manual UAT repro

```js
// DevTools console — paste while a shell session tab is active.
// Adapted from Issue #55 repro.
(() => {
  for (const c of document.querySelectorAll('canvas')) {
    const gl = c.getContext('webgl2') || c.getContext('webgl');
    if (gl) {
      const e = gl.getExtension('WEBGL_lose_context');
      if (e) { e.loseContext(); return 'triggered — wait 3.5s, then check .banner-stack'; }
    }
  }
  return 'no WebGL canvas found';
})()

// After ≥ 3.5s:
({
  hasBanner: !!document.querySelector('.webgl-recovery-banner'),
  hasStack: !!document.querySelector('.banner-stack'),
  text: document.querySelector('.webgl-recovery-banner__message')?.textContent
})
// Expected (post-fix): { hasBanner: true, hasStack: true,
//   text: 'Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact.' }

// After ≥ 11s total (3.5s emitter + 8s auto-dismiss):
({ hasBanner: !!document.querySelector('.webgl-recovery-banner') })
// Expected: { hasBanner: false }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Pre-Phase 93: no banner; silent fallback | Phase 93 WGL-02: `WebGLRecoveryBanner` with 8s auto-dismiss | v3.0 → v3.1 | Phase 93 spec defines the contract; this phase restores its live behavior |
| (none) | xterm `addon-webgl` 0.19.0 keeps internal 3s wait-for-restore window | already in use | UAT timing must account for the 3s delay |

**Deprecated/outdated:** Nothing. xterm.js addon-webgl 0.19.0 is the latest stable (published 2025-12-22).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `queueMicrotask` is the right scheduler vs. `setTimeout(_, 0)` | §5 / Code Examples | Low — both work; microtask is closer to the call site and avoids a tick of latency. If `queueMicrotask` is unavailable in any target browser, `setTimeout(_, 0)` is a drop-in replacement (it is available on all modern browsers AgentHub targets — both Wails WebView and current Chrome/Safari/Firefox; documented in MDN since 2021) |
| A2 | The `dispose()` mid-fire is the actual root cause | §Summary, §Pattern 2 | Medium — strongly suggested by source read of `WebglAddon.ts:84-97` and Issue #55's diagnostic suggestion ("If callback runs but throws, wrap `webglAddon.dispose()` in try/catch and move the React notify call first") but not behaviorally proven without instrumentation in `wails dev`. **Mitigation:** the recommended fix moves the React notify first regardless of which sub-cause (sync dispose vs. throw) is correct — so the fix is robust even if the precise mechanism is something else inside those 5 lines |

## Open Questions

1. **Does `dispose()` actually throw, or merely abort the synchronous continuation?**
   - What we know: Issue #55 author suggested either mechanism; reading `WebglAddon.ts` makes "throw" plausible (the `_terminal._core._store._isDisposed` guard at line 91 only protects against one specific case; the renderer reset at line 94-96 has no try/catch).
   - What's unclear: behavior of `Event.forward` on a disposed source emitter — does the forwarding subscription's `.dispose()` re-fire the original .fire()?
   - Recommendation: instrument with `console.debug` in `wails dev` during the plan's Wave 0 spike if planner wants behavioral confirmation. **Not blocking** — the recommended fix works regardless because it moves the React-side work before any teardown.

2. **Does `term.dispose()` in the unmount cleanup (TerminalPanel.tsx:359) have the same in-fire issue?**
   - What we know: unmount cleanup runs from React, not from inside an xterm emitter fire — so the call stack origin is different.
   - What's unclear: nothing actionable here; this is a separate code path and not regressed by the recommended fix.
   - Recommendation: out of scope — Issue #55 is specifically about the context-loss handler, not unmount.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `wails` CLI (for `wails dev`) | UAT repro on desktop | Assumed ✓ (already used for v3.3 release) | — | Web-share to Chrome |
| Chrome / Chromium with DevTools | UAT repro on web surface | ✓ (any modern Chrome) | — | — |
| `npm` / vitest | Regression test | ✓ | vitest 4.1.0 already installed | — |
| `WEBGL_lose_context` extension | DevTools repro | ✓ in all WebGL-enabled browsers | (browser API) | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 [VERIFIED: `frontend/package.json:39`] |
| Config file | `frontend/vite.config.ts` (vitest config inline at top of file) |
| Quick run command | `cd frontend && pnpm test -- TerminalPanel` (or `npm test -- TerminalPanel`) |
| Full suite command | `cd frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UI-01 (source) | `onContextLoss` handler calls `onWebGLContextLost('context-loss')` **before** any `dispose()` call | unit (source-inspection) | `cd frontend && pnpm test -- TerminalPanel.hot-swap` | ✅ extend existing file |
| UI-01 (behavioral) | Simulated context-loss event causes parent `onWebGLContextLost` callback to be invoked with `'context-loss'` | unit (mocked addon) | `cd frontend && pnpm test -- TerminalPanel.contextLoss` | ❌ Wave 0 — new file |
| UI-01 (DOM end-to-end) | After dispatching the event, `.webgl-recovery-banner` exists in rendered DOM | unit (vitest + jsdom + react-dom/client + flushSync) | `cd frontend && pnpm test -- App.contextLoss` | ❌ Wave 0 — new file (or append to App.test.tsx) |
| UI-01 (8s auto-dismiss) | After 8000ms fake-timers tick, banner is removed | unit | (covered by existing `WebGLRecoveryBanner.test.tsx`) | ✅ — already locked at line 76-82 |
| UI-02 (DOM fallback) | xterm internally calls `renderService.setRenderer(_createRenderer())` on `WebglAddon.dispose()` | source-inspection of `node_modules/@xterm/addon-webgl/src/WebglAddon.ts` | source check; reference in commit message | (xterm library, locked) |
| UI-01/UI-02 (manual UAT) | Real WebGL context loss in `wails dev` produces visible banner + readable terminal | manual | `wails dev` + DevTools script (see §Code Examples) | manual-only |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm test -- TerminalPanel` (~5s)
- **Per wave merge:** `cd frontend && pnpm test` (full frontend suite)
- **Phase gate:** full suite green + manual UAT log captured before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/components/__tests__/TerminalPanel.contextLoss.test.tsx` — behavioral test with mocked `@xterm/addon-webgl`: mock `WebglAddon` to capture the `onContextLoss` listener, render `<TerminalPanel>` inside a wrapper that captures `onWebGLContextLost` calls, invoke the captured listener synchronously, assert the wrapper's callback was called with `'context-loss'` **before** any `dispose()` was invoked on the mock.
- [ ] Source-inspection assertions in existing `TerminalPanel.hot-swap.test.tsx` — add tests that the substring `onWebGLContextLost?.('context-loss')` appears **before** `webglAddon.dispose()` inside the `onContextLoss(...)` callback body (regex match on the raw source).

*(Existing `WebGLRecoveryBanner.test.tsx` already covers the banner-component contract; no changes needed there.)*

## Security Domain

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | no | Banner copy is verbatim from Phase 93 UI-SPEC; no user input renders |
| V6 Cryptography | no | — |
| V7 Error Handling | yes | The `try/catch` around `dispose()` matches established pattern; do NOT swallow `onWebGLContextLost` notification errors silently in any new code path |
| V14 Configuration | no | — |

### Known Threat Patterns for {react + xterm}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Information disclosure via renderer string | I | `webglProbe.ts` already mitigates — renderer string never leaves the function; banner copy never names "SwiftShader" etc. [VERIFIED: `webglProbe.ts:14-17` comment + verbatim copy contract] |
| XSS via banner text | T | Banner copy is a static string literal in `WebGLRecoveryBanner.tsx:42-45`; no interpolation; not a vector |

**No new security surface introduced by this phase.** The fix is a re-order of existing calls.

## Project Constraints (from CLAUDE.md)

| Directive | Source | Application to This Phase |
|-----------|--------|----------------------------|
| JS/TS: `camelCase`, ESLint + Prettier, TypeScript types | CLAUDE.md §Code Conventions | Fix must keep existing formatting and pass the project's linter/typecheck |
| Frontend testing: vitest | CLAUDE.md §Testing | Already used; new test files must match existing patterns (see `WebGLRecoveryBanner.test.tsx`) |
| Cross-surface parity is release-blocking | MEMORY.md `feedback_cross_surface_parity` | Single React fix covers both desktop (Wails) and web — manually verify both per phase CONTEXT |
| Verify color-based UAT at source (user is colorblind) | MEMORY.md `user_colorblind` | Source-inspection tests required — don't rely on "looks blue enough." Already inline in §Validation Architecture |
| Don't delete test artifacts early | MEMORY.md `feedback_dont_delete_test_artifacts_early` | Keep any DevTools repro screenshots / console captures until user confirms UAT pass |
| Cross-check GitHub issues during UAT | MEMORY.md `feedback_check_github_issues_during_uat` | Recheck Issue #55 + scan for new bug comments before declaring phase complete |
| Wails DevTools disabled in prod | MEMORY.md `project_wails_devtools_disabled_in_prod` | UAT must use `wails dev` OR web-share + Chrome — never expect DevTools on the prod build |
| Wails build requires `-tags wailsassets` | MEMORY.md `project_wails_build_requires_tags` | Not relevant unless this phase builds prod; if planner adds a build step, must include the tag |
| LSP over Grep/Read for code navigation | CLAUDE.md §Code Navigation | Use LSP `goToDefinition` / `findReferences` when investigating xterm types during plan execution |
| `pnpm` preferred for Node packages | CLAUDE.md §Tech Stack | Test commands above use `pnpm` (npm acceptable if pnpm absent locally) |
| Make beliefs pay rent | CLAUDE.md §Core Principles | This research's prediction: handler reorder + microtask defer will fix the bug. Plan should commit the fix and the regression test in one commit, then run UAT immediately. If UAT still fails post-fix, the prediction was wrong and the planner should re-open §Open Questions #1 |

## Sources

### Primary (HIGH confidence)
- `frontend/src/components/TerminalPanel.tsx` (lines 376-412 — hot-swap useEffect, context-loss handler)
- `frontend/src/components/WebGLRecoveryBanner.tsx` (lines 31-62 — 8s auto-dismiss timer)
- `frontend/src/App.tsx` (lines 168-174 — banner state; 308-318 — handler; 976-1020 — banner-stack rendering; 1163-1174 — TerminalPanel mount)
- `frontend/src/style.css` (lines 1737-1815 — .banner-stack + .webgl-recovery-banner styles)
- `frontend/node_modules/@xterm/addon-webgl/src/WebglAddon.ts` (lines 28-97 — emitter chain + activate + dispose hooks)
- `frontend/node_modules/@xterm/addon-webgl/src/WebglRenderer.ts` (lines 63-131 — 3s context-loss timeout; webglcontextrestored handling)
- `frontend/node_modules/@xterm/addon-webgl/typings/addon-webgl.d.ts` (public API contract)
- `frontend/node_modules/@xterm/addon-webgl/package.json` (version 0.19.0)
- `npm view @xterm/addon-webgl version` (verified 0.19.0 current as of 2025-12-22)
- GitHub Issue #55 (full body via `gh issue view 55`)
- `frontend/src/components/__tests__/WebGLRecoveryBanner.test.tsx` (existing test coverage)
- `frontend/src/components/__tests__/TerminalPanel.hot-swap.test.tsx` (existing source-inspection pattern to extend)

### Secondary (MEDIUM confidence)
- React docs — `useState` setter identity stability: https://react.dev/reference/react/useState ([CITED] — known property of React from 16.8+; confirmed in current docs)
- MDN — `queueMicrotask`: https://developer.mozilla.org/en-US/docs/Web/API/queueMicrotask ([CITED] — universal browser support)

### Tertiary (LOW confidence)
- None — every claim grounded in source read.

## Metadata

**Confidence breakdown:**
- Root cause identification: HIGH — source-traced in `WebglAddon.ts` and `WebglRenderer.ts`; matches Issue #55's diagnostic suggestion; refutes a confidently-stated but unverified CONTEXT hypothesis (closure rot).
- Fix pattern: HIGH — follows established defensive-dispose pattern already in `TerminalPanel.tsx:320`.
- Banner contract verification: HIGH — full file read of `WebGLRecoveryBanner.tsx` confirms 8s timer at line 38 already correct.
- DOM-fallback independence (UI-02): HIGH — `WebglAddon.ts:90-97` shows xterm internally resets the render service to a fresh DOM renderer; orthogonal to our React state.
- Test surface recommendation: MEDIUM — vitest + jsdom + react-dom/client pattern is proven by `WebGLRecoveryBanner.test.tsx`; mocking `@xterm/addon-webgl` for behavioral test will need Wave 0 implementation work.

**Research date:** 2026-05-18
**Valid until:** 2026-06-17 (30 days — stable code area; xterm.js addon-webgl 0.19.0 just released 2025-12-22, no imminent breaking change expected)
