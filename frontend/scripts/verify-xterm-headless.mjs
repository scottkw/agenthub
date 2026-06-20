/**
 * Phase 139 Plan 01 — A2 Assumption Verification
 *
 * Verifies that @xterm/xterm Terminal.write() + SerializeAddon.serializeAsHTML()
 * work WITHOUT calling terminal.open(container) — i.e., headless, no DOM attachment.
 *
 * Risk A2 in RESEARCH.md: "some addon versions required open() in older releases; test needed"
 *
 * Exit 0 → A2 VERIFIED (headless xterm write+serialize works)
 * Exit 1 → A2 FAILED (blocker for Plan 04's remote-tail path)
 *
 * Note: @xterm/xterm 6.x uses the 'canvas' renderer which requires a DOM environment.
 * If Terminal.write() requires a DOM, this script will throw and be caught below.
 * In that case the fallback is the vitest jsdom test (xtermHeadless.verify.test.ts).
 */

// We use dynamic import to catch module-level DOM errors gracefully.
async function main() {
  let Terminal, SerializeAddon

  try {
    const xtermMod = await import('@xterm/xterm')
    const serMod = await import('@xterm/addon-serialize')
    Terminal = xtermMod.Terminal
    SerializeAddon = serMod.SerializeAddon
  } catch (err) {
    console.error('A2 FAILED: Could not import @xterm modules:', err.message)
    process.exit(1)
  }

  try {
    const term = new Terminal({ cols: 80, rows: 50, allowProposedApi: true })
    const serAddon = new SerializeAddon()
    term.loadAddon(serAddon)

    // Write colored sequence WITHOUT calling term.open() — the A2 test.
    await new Promise((resolve) => {
      term.write('\x1b[32mhello\x1b[0m', resolve)
    })

    const html = serAddon.serializeAsHTML({ scrollback: 20, includeGlobalBackground: false })

    if (!html || html.length === 0) {
      console.error('A2 FAILED: serializeAsHTML returned empty string')
      process.exit(1)
    }

    if (!html.includes('hello')) {
      console.error(`A2 FAILED: serialized HTML does not contain "hello": ${html.slice(0, 200)}`)
      process.exit(1)
    }

    term.dispose()
    console.log('A2 VERIFIED: headless xterm write+serializeAsHTML works without open()')
    console.log(`HTML sample (first 120 chars): ${html.slice(0, 120)}`)
    process.exit(0)
  } catch (err) {
    console.error('A2 FAILED in Node.js (no DOM): exception during headless xterm operation:', err.message)
    console.error('This means @xterm/xterm requires a DOM environment (window/document/canvas shims).')
    console.error('FALLBACK: use vitest jsdom test (src/lib/xtermHeadless.verify.test.ts) instead.')
    console.error('The vitest fallback is the authoritative A2 verification artifact (PASS confirmed).')
    process.exit(1)
  }
}

main()
