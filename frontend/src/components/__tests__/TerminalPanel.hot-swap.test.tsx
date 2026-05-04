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
