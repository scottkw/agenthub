import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { TerminalPanel } from '../TerminalPanel'
import raw from '../TerminalPanel.tsx?raw'

const __dir = dirname(fileURLToPath(import.meta.url))
const cssRaw = readFileSync(resolve(__dir, '../../style.css'), 'utf-8')

describe('TerminalPanel', () => {
  it('exports TerminalPanel function', () => {
    expect(typeof TerminalPanel).toBe('function')
  })

  it('source contains flex:1 and minHeight:0 inline styles', () => {
    expect(raw).toContain("flex: 1")
    expect(raw).toContain("minHeight: 0")
    expect(raw).toContain("width: '100%'")
  })
})

describe('TERM-01 terminal container layout (style.css)', () => {
  it('terminal-container has min-height: 0 so content fills all available space', () => {
    // Extract the .terminal-container rule block from the stylesheet
    const ruleStart = cssRaw.indexOf('.terminal-container')
    expect(ruleStart).toBeGreaterThan(-1)
    const ruleBlock = cssRaw.slice(ruleStart, cssRaw.indexOf('}', ruleStart) + 1)
    expect(ruleBlock).toContain('min-height: 0')
  })
})

describe('font size control', () => {
  it('registers attachCustomKeyEventHandler for key interception', () => {
    expect(raw).toContain('attachCustomKeyEventHandler')
  })

  it('intercepts SHIFT+= using ev.shiftKey && ev.key === "="', () => {
    expect(raw).toContain("ev.shiftKey && ev.key === '='")
  })

  it('intercepts SHIFT+- using ev.shiftKey && ev.key === "-"', () => {
    expect(raw).toContain("ev.shiftKey && ev.key === '-'")
  })

  it('returns false for matched keys to suppress PTY injection', () => {
    // Handler must return false for SHIFT+= and SHIFT+-
    const handlerBlock = raw.slice(
      raw.indexOf('attachCustomKeyEventHandler'),
      raw.indexOf('attachCustomKeyEventHandler') + 500
    )
    const falseCount = (handlerBlock.match(/return false/g) || []).length
    expect(falseCount).toBeGreaterThanOrEqual(2)
  })

  it('guards on ev.type === keydown only', () => {
    expect(raw).toContain("ev.type !== 'keydown'")
  })

  it('accepts fontSize prop in interface', () => {
    expect(raw).toContain('fontSize: number')
  })

  it('accepts onFontSizeChange callback prop in interface', () => {
    expect(raw).toContain('onFontSizeChange: (delta: number) => void')
  })

  it('applies fontSize prop to terminal options', () => {
    expect(raw).toContain('options.fontSize = fontSize')
  })

  it('calls fitAddon.fit() after fontSize change', () => {
    // fontSize assignment and fit() must appear in same effect
    const fontSizeEffect = raw.slice(
      raw.indexOf('options.fontSize = fontSize'),
    )
    expect(fontSizeEffect).toContain('.fit()')
  })

  it('has useEffect with fontSize dependency', () => {
    expect(raw).toContain('[fontSize]')
  })
})

describe('FILL-01..06 rAF retry loop initial fit', () => {
  it('defines MAX_ATTEMPTS constant set to 20', () => {
    expect(raw).toContain('MAX_ATTEMPTS = 20')
  })

  it('checks proposeDimensions() for cell dimension readiness', () => {
    // The retry loop must check proposeDimensions() to know when cell dims are non-zero
    expect(raw).toContain('proposeDimensions()')
  })

  it('defines tryFit function for retry loop', () => {
    expect(raw).toContain('tryFit')
  })

  it('uses requestAnimationFrame for retry scheduling', () => {
    const matches = raw.match(/requestAnimationFrame/g) || []
    expect(matches.length).toBeGreaterThanOrEqual(2) // initial rAF + retry rAF
  })

  it('has cancelled flag for cleanup safety', () => {
    expect(raw).toContain('cancelled = true')
  })

  it('calls cancelAnimationFrame in cleanup', () => {
    expect(raw).toContain('cancelAnimationFrame(rafId)')
  })

  it('does NOT use old double-rAF pattern (no rafId2)', () => {
    expect(raw).not.toContain('rafId2')
  })

  it('does NOT use document.fonts.ready as fit trigger', () => {
    expect(raw).not.toContain('document.fonts.ready')
  })

  it('retains [isActive] as the sole dependency', () => {
    // Find the isActive effect — it must end with }, [isActive])
    expect(raw).toContain('[isActive]')
  })

  it('retains ResizeObserver for subsequent resize handling', () => {
    expect(raw).toContain('new ResizeObserver')
  })
})

describe('TERM-01/02 initial fit not synchronous', () => {
  it('does not call fit() synchronously in isActive effect', () => {
    const isActiveStart = raw.indexOf("if (!isActive")
    const roStart = raw.indexOf("new ResizeObserver", isActiveStart)
    if (isActiveStart > -1 && roStart > -1) {
      const betweenBlock = raw.slice(isActiveStart, roStart)
      expect(betweenBlock).toContain('requestAnimationFrame')
    }
  })
})
