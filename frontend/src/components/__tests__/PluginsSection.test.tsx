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
      const a = raw.indexOf(`renderRow('${order[i]}'`)
      const b = raw.indexOf(`renderRow('${order[i + 1]}'`)
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

describe('Phase 97 SER-02: italic secrets-warning caption under Serialize row — Plan 97-05 implementation', () => {
  it('Serialize renderRow carries the verbatim secrets warning as its 4th argument', () => {
    // Verbatim per REQUIREMENTS SER-02:
    const VERBATIM = 'Saved files include any secrets, tokens, or sensitive data printed in the session.'
    expect(raw).toContain(VERBATIM)

    // Positional check — the verbatim string must appear within ~250 chars
    // AFTER the 'Save terminal as text' label (the 2nd argument to the
    // serialize renderRow call). This proves it's the 4th argument to
    // serialize renderRow, not somewhere else (e.g. inside a comment or
    // a different row).
    const labelIdx = raw.indexOf('Save terminal as text')
    expect(labelIdx).toBeGreaterThan(-1)
    const window = raw.slice(labelIdx, labelIdx + 400)
    expect(window).toContain(VERBATIM)
  })
})

describe('IMG-01: italic next-session-only caption under Image row (Plan 96-04)', () => {
  it("Image renderRow carries 'Applies to new sessions you create.' as its 4th argument", () => {
    // The exact string must appear at least TWICE — once for unicode11
    // (existing — Phase 93 U11-01), once for image (this plan adds it).
    // UX consistency is intentional: both addons are next-session-only;
    // they share the same caption verbatim.
    const matches = raw.match(/Applies to new sessions you create\./g) || []
    expect(matches.length).toBeGreaterThanOrEqual(2)

    // Tighter assertion: the caption must appear within ~400 chars AFTER
    // the 'Inline images' label, proving it is the 4th argument of the
    // image renderRow call (not just floating elsewhere).
    const inlineImagesIdx = raw.indexOf("'Inline images'")
    expect(inlineImagesIdx).toBeGreaterThan(-1)
    const captionAfterImage = raw.indexOf('Applies to new sessions you create.', inlineImagesIdx)
    expect(captionAfterImage).toBeGreaterThan(-1)
    expect(captionAfterImage - inlineImagesIdx).toBeLessThan(400)
  })
})

describe('Phase 98 PRG-01: v3.3-flip italic caption under Progress row', () => {
  it('renders the v3.3-flip italic caption under the Progress toggle', () => {
    // Verbatim per REQUIREMENTS PRG-01 (v3.3-flip caption):
    const VERBATIM = 'Default OFF in v3.2 — flips ON in v3.3 after field validation.'
    expect(raw).toContain(VERBATIM)

    // Positional check — the verbatim string must appear within ~300 chars
    // AFTER the 'Progress (OSC 9;4)' label (the 2nd argument to the progress
    // renderRow call). This proves it's the 4th argument to progress renderRow.
    const labelIdx = raw.indexOf("'Progress (OSC 9;4)'")
    expect(labelIdx).toBeGreaterThan(-1)
    const window = raw.slice(labelIdx, labelIdx + 400)
    expect(window).toContain(VERBATIM)
  })

  it('Progress renderRow uses settings-panel__description--italic modifier class', () => {
    // The renderRow helper already routes caption into the italic class.
    // Verify the class exists in the source (set up by prior phases).
    expect(raw).toContain('settings-panel__description--italic')
  })

  it('keeps the Progress toggle default-OFF (progress key is false by default in daemon PluginSettings)', () => {
    // Source-level assertion: the progress row must NOT hardcode checked={true}.
    // Render-time behavior is daemon-side (Phase 92); this asserts the source
    // does not override the daemon default at the component level.
    const progressRowStart = raw.indexOf("'progress'")
    expect(progressRowStart).toBeGreaterThan(-1)
    const progressRowWindow = raw.slice(progressRowStart, progressRowStart + 300)
    expect(progressRowWindow).not.toContain('checked={true}')
  })
})

describe('Phase 99 PUI-02: onPluginToggleSideEffect side-effect callback', () => {
  it('exports PluginToggleKind type and PluginsSectionProps interface', () => {
    expect(raw).toContain('PluginToggleKind')
    expect(raw).toContain('PluginsSectionProps')
    expect(raw).toContain('onPluginToggleSideEffect')
  })

  it('holds a lastSavedRef for diff detection', () => {
    expect(raw).toContain('lastSavedRef')
  })

  it('compares prior.unicode11 vs current after save', () => {
    expect(raw).toContain('prior.unicode11 !== pluginConfig.unicode11')
  })

  it('compares prior.image vs current after save', () => {
    expect(raw).toContain('prior.image !== pluginConfig.image')
  })

  it('updates lastSavedRef.current after successful save', () => {
    // Must appear in handleSavePlugins, after SetPluginSettings call
    const saveHandlerIdx = raw.indexOf('handleSavePlugins')
    expect(saveHandlerIdx).toBeGreaterThan(-1)
    const afterSave = raw.indexOf('lastSavedRef.current = pluginConfig', saveHandlerIdx)
    expect(afterSave).toBeGreaterThan(-1)
  })

  it('calls onPluginToggleSideEffect when kinds array is non-empty', () => {
    expect(raw).toContain('onPluginToggleSideEffect')
    expect(raw).toMatch(/kinds\.length > 0/)
  })
})
