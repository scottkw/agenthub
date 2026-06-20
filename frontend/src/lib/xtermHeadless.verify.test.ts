/**
 * Phase 139 Plan 01 — A2 Assumption Verification (vitest jsdom fallback)
 *
 * Verifies that @xterm/xterm Terminal.write() + SerializeAddon.serializeAsHTML()
 * work WITHOUT calling terminal.open(container) in a jsdom environment.
 *
 * Risk A2 in RESEARCH.md: "@xterm/xterm write()+serializeAsHTML() without open()"
 *
 * The Node.js script (scripts/verify-xterm-headless.mjs) detected that
 * @xterm/xterm 6.x requires a DOM-like environment (Terminal is not available
 * in raw Node.js without DOM shims). This vitest test uses the jsdom environment
 * configured in vite.config.ts to provide the necessary globals (document, window).
 *
 * A2 verdict: PASS — headless xterm works in jsdom WITHOUT calling term.open().
 */

import { describe, it, expect } from 'vitest'
import { Terminal } from '@xterm/xterm'
import { SerializeAddon } from '@xterm/addon-serialize'

describe('A2 Verification: headless xterm write + serializeAsHTML without open()', () => {
  it('Terminal.write() + serializeAsHTML() work without calling term.open()', async () => {
    // Construct a Terminal WITHOUT calling open() — the A2 test.
    // allowProposedApi: true is required for SerializeAddon.
    const term = new Terminal({ cols: 80, rows: 50, allowProposedApi: true })
    const serAddon = new SerializeAddon()
    term.loadAddon(serAddon)

    // Write a colored ANSI sequence WITHOUT calling term.open(container).
    // This is the headless use pattern for remote tail rendering in Plan 04.
    await new Promise<void>((resolve) => {
      term.write('\x1b[32mhello\x1b[0m', resolve)
    })

    // serializeAsHTML must return a non-empty string containing "hello".
    const html = serAddon.serializeAsHTML({ scrollback: 20, includeGlobalBackground: false })

    expect(html).toBeTruthy()
    expect(html.length).toBeGreaterThan(0)
    expect(html).toContain('hello')

    // Clean up — dispose is important to avoid resource leaks in test suite.
    term.dispose()
  })

  it('serializeAsHTML output is an HTML string with span elements for styled text', async () => {
    const term = new Terminal({ cols: 80, rows: 50, allowProposedApi: true })
    const serAddon = new SerializeAddon()
    term.loadAddon(serAddon)

    // Write bold + colored text.
    await new Promise<void>((resolve) => {
      term.write('\x1b[1;32mworld\x1b[0m', resolve)
    })

    const html = serAddon.serializeAsHTML({ scrollback: 20, includeGlobalBackground: false })

    // The HTML must contain "world" (the written text).
    expect(html).toContain('world')
    // The serialized HTML is expected to contain span elements with styling.
    // (The exact structure depends on @xterm/addon-serialize internals.)
    expect(html).toContain('<span')

    term.dispose()
  })

  it('Terminal.dispose() works after headless write (no crash)', async () => {
    const term = new Terminal({ cols: 80, rows: 50, allowProposedApi: true })
    const serAddon = new SerializeAddon()
    term.loadAddon(serAddon)

    await new Promise<void>((resolve) => {
      term.write('dispose test\n', resolve)
    })

    // dispose must not throw even without open() having been called.
    expect(() => term.dispose()).not.toThrow()
  })
})
