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

  it('calls fitTerminal() after fontSize change', () => {
    // fontSize assignment and fitTerminal() must appear in same effect
    const fontSizeEffect = raw.slice(
      raw.indexOf('options.fontSize = fontSize'),
    )
    expect(fontSizeEffect).toContain('fitTerminal(')
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
    // The fit/activation effect depends only on isActive
    expect(raw).toContain('[isActive])')
  })

  it('retains ResizeObserver for subsequent resize handling', () => {
    expect(raw).toContain('new ResizeObserver')
  })

  it('uses fitTerminal() instead of fitAddon.fit() for full-width rendering', () => {
    expect(raw).toContain('fitTerminal(termRef.current!)')
  })

  it('sends terminal dimensions on WebSocket open to sync PTY size', () => {
    // The onOpen callback must send the current terminal size to the PTY.
    // Without this, the resize from fitTerminal() is dropped (WS not yet open)
    // and the CLI process renders to the wrong width.
    expect(raw).toContain('sendResize(term.cols, term.rows)')
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

describe('PAD-01 terminal padding', () => {
  it('.terminal-session-container has padding: 8px for inset from edges', () => {
    expect(cssRaw).toMatch(/\.terminal-session-container\s*\{[^}]*padding:\s*8px/)
  })

  it('.terminal-session-container does NOT have hardcoded background-color (dynamic via inline style)', () => {
    expect(cssRaw).not.toMatch(/\.terminal-session-container\s*\{[^}]*background-color:/)
  })

  it('container div has terminal-session-container class', () => {
    expect(raw).toContain('terminal-session-container')
  })

  it('fitTerminal subtracts parent padding from clientWidth/Height for accurate sizing', () => {
    expect(raw).toContain('clientWidth')
    expect(raw).toContain('clientHeight')
    expect(raw).toContain('paddingLeft')
    expect(raw).toContain('paddingRight')
    expect(raw).toContain('paddingTop')
    expect(raw).toContain('paddingBottom')
  })
})

describe('THM-03: live theme application', () => {
  it('imports ITheme type from @xterm/xterm', () => {
    expect(raw).toContain('ITheme')
  })

  it('TerminalPanelProps includes theme: ITheme', () => {
    const interfaceStart = raw.indexOf('interface TerminalPanelProps')
    const interfaceEnd = raw.indexOf('}', interfaceStart)
    const interfaceBlock = raw.slice(interfaceStart, interfaceEnd + 1)
    expect(interfaceBlock).toContain('theme: ITheme')
  })

  it('passes theme to Terminal constructor (not hardcoded background)', () => {
    // Must NOT have the old hardcoded theme
    expect(raw).not.toContain("theme: { background: '#1a1b26' }")
    // Must pass the theme prop to constructor
    const constructorBlock = raw.slice(raw.indexOf('new Terminal('), raw.indexOf('new Terminal(') + 300)
    expect(constructorBlock).toContain('theme')
  })

  it('has useEffect that assigns options.theme = theme', () => {
    expect(raw).toContain('options.theme = theme')
  })

  it('dedicated theme effect has [theme] dependency array', () => {
    // The standalone theme effect (THM-03) must exist with [theme] dep
    expect(raw).toContain('}, [theme])')
  })

  it('sets backgroundColor from theme.background in inline style', () => {
    expect(raw).toContain("theme.background ?? '#1a1b26'")
  })
})

describe('IMG-01/IMG-02 ImageAddon construction (Plan 96-04)', () => {
  it('TerminalPanel.tsx imports ImageAddon from @xterm/addon-image', () => {
    expect(raw).toContain("import { ImageAddon } from '@xterm/addon-image'")
  })
  it('TerminalPanel.tsx declares imageAddonRef parallel to other addon refs', () => {
    expect(raw).toMatch(/const\s+imageAddonRef\s*=\s*useRef<ImageAddon\s*\|\s*null>\(null\)/)
  })
  it('TerminalPanel.tsx constructs new ImageAddon(...) with enableSizeReports: false (Pitfall #8 regression guard)', () => {
    expect(raw).toContain('new ImageAddon(')
    expect(raw).toContain('enableSizeReports: false')
  })
  it('TerminalPanel.tsx passes pluginConfig?.imageConfig?.storageLimit ?? 16 to ImageAddon constructor', () => {
    expect(raw).toContain('pluginConfig?.imageConfig?.storageLimit ?? 16')
  })
  it('ImageAddon construction lives in MOUNT useEffect, NOT hot-swap useEffect (next-session-only invariant)', () => {
    // Assertion 1: ImageAddon construction text appears in source.
    expect(raw).toContain('new ImageAddon(')

    // Assertion 2: ImageAddon construction appears within ~2000 chars
    // AFTER the Unicode 11 construction marker — proving mount-useEffect
    // placement. Unicode 11 lives in the [sessionId]-keyed mount useEffect.
    const unicode11Idx = raw.indexOf('new Unicode11Addon(')
    expect(unicode11Idx).toBeGreaterThan(-1)
    const imageIdx = raw.indexOf('new ImageAddon(', unicode11Idx)
    expect(imageIdx).toBeGreaterThan(-1)
    expect(imageIdx - unicode11Idx).toBeLessThan(2000)

    // Assertion 3: NO useEffect dep array references pluginConfig?.image
    // or pluginConfig?.imageConfig. The mount-useEffect BODY may (and
    // must) reference these — they live in the construction gate. But
    // a dep array reference would re-run the effect on Settings save,
    // violating the next-session-only invariant.
    //
    // Pragmatic regex: match each dep array `}, [ ... ])` pattern and
    // verify image fields are absent.
    const depArrayMatches = raw.match(/\}\s*,\s*\[([^\]]*)\]\s*\)/g) || []
    expect(depArrayMatches.length).toBeGreaterThan(0)
    for (const depArray of depArrayMatches) {
      expect(depArray).not.toMatch(/pluginConfig\?\.image\b/)
      expect(depArray).not.toMatch(/pluginConfig\?\.imageConfig\b/)
    }
  })
})
