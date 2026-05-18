import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import raw from '../TerminalPanel.tsx?raw'

// Phase 113 Plan 01 — Source-grep tests for the React + CSS wiring of the
// attachTouchScroll handler. Mirrors the established `?raw` + style.css
// readFileSync pattern from TerminalPanel.test.tsx (per RESEARCH §Test
// Surface). Wiring assertions only — pure-function behavior lives in
// touchScrollHandler.test.ts.

const __dir = dirname(fileURLToPath(import.meta.url))
const cssRaw = readFileSync(resolve(__dir, '../../style.css'), 'utf-8')

describe('Phase 113 UI-03 / UI-04: TerminalPanel touch-scroll wiring', () => {
  // Test 1
  it('imports attachTouchScroll from ../lib/touchScrollHandler', () => {
    expect(raw).toMatch(/import\s*\{\s*attachTouchScroll\s*\}\s*from\s*['"]\.\.\/lib\/touchScrollHandler['"]/)
  })

  // Test 2
  it('calls attachTouchScroll(container, term) inside a useEffect', () => {
    expect(raw).toMatch(/attachTouchScroll\s*\(/)
  })

  // Test 3
  it('uses [sessionId] dep array on the touch-scroll useEffect', () => {
    // Locate the line that calls attachTouchScroll, then look forward for
    // the closing of the effect: `}, [sessionId])`. The effect we care about
    // must close with that exact dep-array shape.
    const callIdx = raw.indexOf('attachTouchScroll(')
    expect(callIdx).toBeGreaterThan(-1)
    // Grab the chunk after the call and verify the next useEffect-style
    // closer is `}, [sessionId])`. We allow whitespace/newlines.
    const tail = raw.slice(callIdx, callIdx + 600)
    expect(tail).toMatch(/\}\s*,\s*\[sessionId\]\s*\)/)
  })
})

describe('Phase 113 UI-03 / UI-04: style.css touch-action companion', () => {
  // Test 4
  it('.terminal-session-container rule contains touch-action: pan-y', () => {
    const ruleStart = cssRaw.indexOf('.terminal-session-container')
    expect(ruleStart).toBeGreaterThan(-1)
    const ruleEnd = cssRaw.indexOf('}', ruleStart)
    expect(ruleEnd).toBeGreaterThan(ruleStart)
    const ruleBlock = cssRaw.slice(ruleStart, ruleEnd + 1)
    expect(ruleBlock).toMatch(/touch-action:\s*pan-y/)
  })

  // Test 5
  it('.terminal-session-container rule does NOT contain touch-action: none', () => {
    const ruleStart = cssRaw.indexOf('.terminal-session-container')
    const ruleEnd = cssRaw.indexOf('}', ruleStart)
    const ruleBlock = cssRaw.slice(ruleStart, ruleEnd + 1)
    expect(ruleBlock).not.toMatch(/touch-action:\s*none/)
  })
})
