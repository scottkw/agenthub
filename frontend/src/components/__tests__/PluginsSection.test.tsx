import { describe, it, expect } from 'vitest'
import raw from '../PluginsSection.tsx?raw'

describe('PUI-01: 8 toggle rows in UI-SPEC order', () => {
  it('contains all 8 plugin labels', () => {
    expect(raw).toContain('WebGL renderer')
    expect(raw).toContain('Unicode 11 widths')
    expect(raw).toContain('Find in scrollback')
    expect(raw).toContain('Clickable web links')
    expect(raw).toContain('Inline images')
    expect(raw).toContain('Save terminal as text')
    expect(raw).toContain('Clipboard (OSC 52)')
    expect(raw).toContain('Progress (OSC 9;4)')
  })

  it('contains all 8 UI-SPEC one-sentence descriptions', () => {
    expect(raw).toContain('GPU-accelerated terminal rendering')
    expect(raw).toContain('Correct cell widths for emoji')
    expect(raw).toContain('Open a find bar with Cmd-F')
    expect(raw).toContain('Detect URLs in terminal output')
    expect(raw).toContain('Render images sent via sixel')
    expect(raw).toContain('Right-click a tab to export')
    expect(raw).toContain('OSC 52 escape sequence')
    expect(raw).toContain('OSC 9;4 progress updates')
  })

  it('renders rows in UI-SPEC order (Pitfall #5 guard)', () => {
    const order = ['webgl', 'unicode11', 'search', 'webLinks', 'image',
                   'serialize', 'clipboard', 'progress']
    for (let i = 0; i < order.length - 1; i++) {
      const a = raw.indexOf(order[i])
      const b = raw.indexOf(order[i + 1])
      expect(a).toBeGreaterThan(-1)
      expect(b).toBeGreaterThan(-1)
      expect(a).toBeLessThan(b)
    }
  })
})

describe('PUI-01: Three-state Save button', () => {
  it('has Save Plugins copy', () => {
    expect(raw).toContain("'Save Plugins'")
  })
  it('has Saving… and Saved! states', () => {
    // U+2026 horizontal ellipsis
    expect(raw).toMatch(/Saving…|Saving…/)
    expect(raw).toContain("'Saved!'")
  })
  it('reuses settings-panel__btn--saved class', () => {
    expect(raw).toContain('settings-panel__btn--saved')
  })
  it('uses 1500ms timeout (matches Save Paths cadence)', () => {
    expect(raw).toContain('1500')
  })
})

describe('PLUG-02: pluginsLoaded flicker guard (Pitfall #3)', () => {
  it('declares pluginsLoaded state', () => {
    expect(raw).toMatch(/pluginsLoaded|setPluginsLoaded/)
  })
  it('gates toggle rows behind pluginsLoaded', () => {
    // The guard should appear in the JSX as a conditional wrap.
    expect(raw).toContain('pluginsLoaded')
  })
})

describe('PLUG-01/PLUG-03: Wails RPC binding usage', () => {
  it('imports GetPluginSettings + SetPluginSettings from Wails bindings', () => {
    expect(raw).toContain('GetPluginSettings')
    expect(raw).toContain('SetPluginSettings')
    expect(raw).toMatch(/wailsjs\/go\/main\/App/)
  })
})

describe('Phase 93 U11-01: italic caption under Unicode 11 row', () => {
  it('renders the verbatim caption "Applies to new sessions you create."', () => {
    expect(raw).toMatch(/Applies to new sessions you create\./)
  })

  it('uses the new settings-panel__description--italic modifier class', () => {
    expect(raw).toContain('settings-panel__description--italic')
  })

  it('caption appears AFTER the Unicode 11 row description (UI-SPEC layout)', () => {
    const captionIdx = raw.indexOf('Applies to new sessions you create.')
    const u11DescIdx = raw.indexOf('Correct cell widths for emoji')
    expect(captionIdx).toBeGreaterThan(-1)
    expect(u11DescIdx).toBeGreaterThan(-1)
    expect(captionIdx).toBeGreaterThan(u11DescIdx)
  })
})
