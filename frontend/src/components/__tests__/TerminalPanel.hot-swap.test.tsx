/**
 * Phase 93 — Hot-swap source-inspection tests for TerminalPanel.
 *
 * Scrollback-survival rationale (ROADMAP Phase 93 SC#3 / WGL-02):
 * Disposing the WebglAddon does NOT clear the Terminal instance's scrollback
 * buffer. xterm.js architecture: the Terminal owns `term.buffer` (scrollback);
 * addons attach via `term.loadAddon(addon)` and detach via `addon.dispose()`.
 * `dispose()` only tears down the render backend — the parent Terminal and its
 * buffer remain intact. The hot-swap useEffect and onContextLoss handler only
 * call `webglAddon.dispose()` and `clipboardAddon.dispose()`; production code
 * never calls `term.clear()` or `term.reset()`. We assert this correct-by-
 * construction property below: source must not contain `term.clear()` or
 * `term.reset()` anywhere in TerminalPanel.tsx.
 *
 * This rationale is verified at runtime in iPad-UAT UAT-2 step 6 ("Scrollback
 * is intact"). The source-inspection assertions below pin the invariant.
 */
import { describe, it, expect } from 'vitest'
import src from '../TerminalPanel.tsx?raw'

describe('Phase 93 WGL-01 / CLIP-01: hot-swap useEffect structure', () => {
  it('Phase 92 inert-prop invariant lifted (no `void pluginConfig`)', () => {
    expect(src).not.toMatch(/void\s+pluginConfig/)
  })

  it('imports ClipboardAddon from @xterm/addon-clipboard', () => {
    expect(src).toMatch(/import\s+\{\s*ClipboardAddon\s*\}\s+from\s+['"]@xterm\/addon-clipboard['"]/)
  })

  it('declares webglAddonRef and clipboardAddonRef refs', () => {
    expect(src).toContain('webglAddonRef')
    expect(src).toContain('clipboardAddonRef')
  })

  it('calls isSoftwareWebGL() probe for WGL-03 software-rasterizer preemption', () => {
    expect(src).toContain('isSoftwareWebGL()')
  })

  it('exposes onWebGLContextLost callback prop wired through to context-loss handler', () => {
    // Prop in the interface + invocation site (≥ 2 occurrences)
    const matches = src.match(/onWebGLContextLost/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(2)
  })

  it('contains a hot-swap useEffect with [pluginConfig?.webgl, pluginConfig?.clipboard, ...] dep array (Pitfall #1 — specific keys, not the whole object)', () => {
    // Match a useEffect whose dep array contains pluginConfig?.webgl AND pluginConfig?.clipboard
    expect(src).toMatch(/\[\s*pluginConfig\?\.webgl\s*,\s*pluginConfig\?\.clipboard[\s\S]*?\]/)
  })

  it('hot-swap useEffect references WebglAddon AND ClipboardAddon (single useEffect coordinates both)', () => {
    expect(src).toMatch(/WebglAddon[\s\S]{0,1500}?ClipboardAddon|ClipboardAddon[\s\S]{0,1500}?WebglAddon/)
  })

  it('Scrollback preservation: source never calls term.clear() (would discard buffer)', () => {
    expect(src).not.toMatch(/term\.clear\(\)/)
  })

  it('Scrollback preservation: source never calls term.reset() (would discard buffer)', () => {
    expect(src).not.toMatch(/term\.reset\(\)/)
  })

  it('hot-swap useEffect disposes addons on toggle-off (cleanup paths)', () => {
    expect(src).toMatch(/webglAddonRef\.current\.dispose\(\)/)
    expect(src).toMatch(/clipboardAddonRef\.current\.dispose\(\)/)
  })
})

/**
 * Phase 112 UI-01 — onContextLoss handler order invariants.
 *
 * Root cause of GitHub Issue #55 (per 112-RESEARCH.md §1, §Pattern 2): the
 * onContextLoss callback in the WebGL hot-swap branch calls webglAddon.dispose()
 * BEFORE invoking onWebGLContextLost?.('context-loss'). Disposing the WebglAddon
 * tears down the Event.forward emitter chain mid-fire (see
 * node_modules/@xterm/addon-webgl/src/WebglAddon.ts:84-97), aborting the
 * synchronous continuation that would otherwise notify React. The fix
 * (RESEARCH §5) is to notify React first, then defer the dispose to a
 * microtask wrapped in try/catch (matching the established defensive pattern
 * at TerminalPanel.tsx:320 for WebLinksAddon).
 *
 * The three source-inspection assertions below pin this invariant: notify
 * before dispose, queueMicrotask present, and the dispose call wrapped in
 * try/catch. RED on main (pre-fix), GREEN after the Task 2 reorder.
 */
describe('Phase 112 UI-01: onContextLoss handler order — notify before dispose', () => {
  // Capture the body of the WebGL onContextLoss callback registered inside the
  // hot-swap useEffect. We anchor on `webglAddon.onContextLoss(` and then walk
  // the callback body (a balanced-ish capture sufficient for assertion granularity).
  function captureOnContextLossBody(): string {
    const m = src.match(/webglAddon\.onContextLoss\(\s*\(\s*\)\s*=>\s*\{([\s\S]*?)\}\s*\)/)
    expect(m, 'expected to find webglAddon.onContextLoss(() => { ... }) in TerminalPanel source').not.toBeNull()
    return m![1]
  }

  it('onContextLoss callback invokes onWebGLContextLost(...) BEFORE webglAddon.dispose()', () => {
    const body = captureOnContextLossBody()
    // Locate the notify call (tolerant of optional chain + whitespace).
    const notifyIdx = body.search(/onWebGLContextLost\s*\??\.\s*\(\s*['"]context-loss['"]\s*\)/)
    // Locate the dispose call.
    const disposeIdx = body.search(/webglAddon\.dispose\s*\(\s*\)/)
    expect(notifyIdx, 'onWebGLContextLost?.("context-loss") must appear inside the callback body').toBeGreaterThanOrEqual(0)
    expect(disposeIdx, 'webglAddon.dispose() must appear inside the callback body').toBeGreaterThanOrEqual(0)
    expect(notifyIdx, 'notify must precede dispose so the emitter chain is alive when React is told').toBeLessThan(disposeIdx)
  })

  it('onContextLoss callback uses queueMicrotask to defer the dispose work', () => {
    const body = captureOnContextLossBody()
    expect(body).toMatch(/queueMicrotask\s*\(/)
  })

  it('onContextLoss callback wraps webglAddon.dispose() in try/catch (defensive — matches line 320 WebLinksAddon pattern)', () => {
    const body = captureOnContextLossBody()
    // Find the dispose call; ensure a `try {` precedes it within the callback body
    // and a `catch` follows. This matches the established defensive style.
    const disposeIdx = body.search(/webglAddon\.dispose\s*\(\s*\)/)
    expect(disposeIdx, 'dispose call must exist in callback body').toBeGreaterThanOrEqual(0)
    const beforeDispose = body.slice(0, disposeIdx)
    const afterDispose = body.slice(disposeIdx)
    // The most recent `try {` before dispose must not be closed by a `}` before
    // the dispose call — i.e., dispose is inside that try block.
    const lastTryIdx = beforeDispose.lastIndexOf('try')
    expect(lastTryIdx, 'expected a `try` keyword before webglAddon.dispose()').toBeGreaterThanOrEqual(0)
    // catch keyword must follow dispose within the callback body.
    expect(afterDispose).toMatch(/catch\s*(?:\([^)]*\))?\s*\{/)
  })
})
