import { describe, it, expect } from 'vitest'
import raw from '../App.tsx?raw'
import terminalPanelRaw from '../components/TerminalPanel.tsx?raw'
import { daemon } from '../wailsjs/go/models'

describe('PLUG-03: App.tsx Wails event subscription', () => {
  it("registers EventsOn('settings:plugins', ...)", () => {
    expect(raw).toMatch(/EventsOn\(\s*['"]settings:plugins['"]/)
  })
  it('declares pluginConfig state', () => {
    // Either useState<PluginSettings | null> or similar shape
    expect(raw).toMatch(/pluginConfig/)
    expect(raw).toMatch(/setPluginConfig/)
  })
  it('fetches initial state via GetPluginSettings on mount', () => {
    expect(raw).toContain('GetPluginSettings')
  })
  it('cleans up the subscription on unmount', () => {
    // The off* function from EventsOn must be invoked in the cleanup return.
    expect(raw).toMatch(/offPlugins\s*\(\s*\)|offSettingsPlugins\s*\(\s*\)/)
  })
})

describe('PLUG-03: prop drilling into TerminalPanel', () => {
  it('passes pluginConfig prop to TerminalPanel in App.tsx', () => {
    expect(raw).toMatch(/pluginConfig=\{pluginConfig\}/)
  })
  it('TerminalPanel props interface accepts pluginConfig', () => {
    expect(terminalPanelRaw).toContain('pluginConfig')
    // Must be optional for Phase 92 (Pitfall #4 — existing tests
    // construct TerminalPanel without this prop).
    expect(terminalPanelRaw).toMatch(/pluginConfig\?\s*:/)
  })
  it('Phase 93 — TerminalPanel consumes pluginConfig inside addon-load useEffect (inert-prop invariant lifted)', () => {
    // Phase 92 contract: prop was threaded but inert. Phase 93 wires
    // consumption inside the hot-swap useEffect. The dep array contains
    // pluginConfig?.webgl and pluginConfig?.clipboard so the effect re-runs
    // when those toggles change.
    const consumesInEffect = /useEffect\([^)]*pluginConfig|useEffect\([^}]*\bpluginConfig\b/.test(terminalPanelRaw)
    expect(consumesInEffect).toBe(true)
  })

  it('Phase 93 — bare `void pluginConfig` line is removed from TerminalPanel.tsx', () => {
    // The Phase 92 inert-prop sentinel line `void pluginConfig` (used to
    // suppress the unused-variable warning) must no longer appear; the
    // prop is now genuinely consumed inside addon-load useEffects.
    expect(terminalPanelRaw).not.toMatch(/^\s*void\s+pluginConfig\s*$/m)
  })
})

describe('Phase 94 SRC-02: SearchConfig nested type round-trip', () => {
  it('daemon.SearchConfig constructs from JSON-shaped object with three booleans', () => {
    const c = new daemon.SearchConfig({ regex: true, caseSensitive: false, wholeWord: true })
    expect(c.regex).toBe(true)
    expect(c.caseSensitive).toBe(false)
    expect(c.wholeWord).toBe(true)
  })

  it('daemon.PluginSettings preserves nested searchConfig as a SearchConfig instance', () => {
    const ps = new daemon.PluginSettings({
      webgl: true,
      unicode11: true,
      search: true,
      searchConfig: { regex: true, caseSensitive: false, wholeWord: true },
      webLinks: true,
      image: true,
      serialize: true,
      clipboard: true,
      progress: false,
    })
    expect(ps.searchConfig).toBeInstanceOf(daemon.SearchConfig)
    expect(ps.searchConfig.regex).toBe(true)
    expect(ps.searchConfig.caseSensitive).toBe(false)
    expect(ps.searchConfig.wholeWord).toBe(true)
  })

  it('JSON round-trip preserves nested searchConfig shape', () => {
    const json = JSON.stringify({
      webgl: true,
      unicode11: true,
      search: true,
      searchConfig: { regex: false, caseSensitive: true, wholeWord: false },
      webLinks: true,
      image: true,
      serialize: true,
      clipboard: true,
      progress: false,
    })
    const ps = new daemon.PluginSettings(JSON.parse(json))
    expect(ps.searchConfig.regex).toBe(false)
    expect(ps.searchConfig.caseSensitive).toBe(true)
    expect(ps.searchConfig.wholeWord).toBe(false)
  })

  it('App.tsx initial-fetch + settings:plugins event payload shape supports nested searchConfig', () => {
    // Source-inspect: App.tsx must use PluginSettings (or compatible shape)
    // when receiving settings:plugins event payloads, so a payload carrying
    // searchConfig propagates through prop drill to TerminalPanel.
    expect(raw).toContain('GetPluginSettings')
    // TerminalPanel exposes pluginConfig prop for its consumers (FindBar will
    // read pluginConfig.searchConfig in Plan 94-03).
    expect(terminalPanelRaw).toMatch(/pluginConfig\?\s*:/)
  })

  // Phase 95 Plan 95-01 Task 2 — Wave 0 RED scaffold for the webLinksConfig
  // nested struct. Plan 95-04 wires the prop drill so TerminalPanel can
  // read pluginConfig.webLinksConfig.{modifier, confirmOSC8, confirmIDN,
  // confirmTyposquat}. Plan 95-05 wires the SetWebLinksConfig sub-key RPC
  // (this commit).
  it('PluginSettings shape includes webLinksConfig nested object (Plan 95-01 + 95-04)', () => {
    // Mirror the searchConfig assertions immediately above: construct a
    // PluginSettings via the Wails-generated daemon model and assert the
    // nested daemon.WebLinksConfig instance survives the constructor and
    // a JSON round-trip. This is the prop-drill foundation: the same
    // daemon.PluginSettings instance flows from GetPluginSettings →
    // App.tsx pluginConfig state → TerminalPanel via the existing
    // pluginConfig prop (Phase 92 wire) and the addon-load useEffect
    // (Phase 95 Plan 95-04 hot-swap consumer reads
    // pluginConfig.webLinksConfig.{modifier,confirmOSC8,confirmIDN,
    // confirmTyposquat}).
    const ps = new daemon.PluginSettings({
      webgl: true,
      unicode11: true,
      search: true,
      searchConfig: { regex: false, caseSensitive: false, wholeWord: false },
      webLinks: true,
      webLinksConfig: {
        modifier: 'platform',
        confirmOSC8: true,
        confirmIDN: true,
        confirmTyposquat: true,
      },
      image: true,
      serialize: true,
      clipboard: true,
      progress: false,
    })
    expect(ps.webLinksConfig).toBeInstanceOf(daemon.WebLinksConfig)
    expect(ps.webLinksConfig.modifier).toBe('platform')
    expect(ps.webLinksConfig.confirmOSC8).toBe(true)
    expect(ps.webLinksConfig.confirmIDN).toBe(true)
    expect(ps.webLinksConfig.confirmTyposquat).toBe(true)

    // JSON round-trip preserves the webLinksConfig sub-shape (mirrors the
    // searchConfig round-trip test above) — guards against the Phase 95
    // settings:plugins event payload losing the nested object on Wails
    // transport.
    const json = JSON.stringify({
      webgl: true,
      unicode11: true,
      search: true,
      searchConfig: { regex: false, caseSensitive: false, wholeWord: false },
      webLinks: true,
      webLinksConfig: {
        modifier: 'ctrl',
        confirmOSC8: false,
        confirmIDN: true,
        confirmTyposquat: false,
      },
      image: true,
      serialize: true,
      clipboard: true,
      progress: false,
    })
    const ps2 = new daemon.PluginSettings(JSON.parse(json))
    expect(ps2.webLinksConfig.modifier).toBe('ctrl')
    expect(ps2.webLinksConfig.confirmOSC8).toBe(false)
    expect(ps2.webLinksConfig.confirmIDN).toBe(true)
    expect(ps2.webLinksConfig.confirmTyposquat).toBe(false)
  })

  // Phase 96 Plan 96-01 Task 2 — Wave 0 hand-edit landing for the
  // imageConfig nested struct. The daemon.ImageConfig class lands in
  // Plan 96-01 Task 1 (frontend/src/wailsjs/go/models.ts hand-edit), so
  // these tests run GREEN immediately. Plans 96-04 / 96-05 wire the prop
  // drill into TerminalPanel and the SetImageConfig sub-key RPC.
  it('daemon.PluginSettings preserves nested imageConfig as an ImageConfig instance (Plan 96-01 hand-edit)', () => {
    const ps = new daemon.PluginSettings({
      webgl: true, unicode11: true, search: true,
      searchConfig: { regex: false, caseSensitive: false, wholeWord: false },
      webLinks: true,
      webLinksConfig: { modifier: 'platform', confirmOSC8: true, confirmIDN: true, confirmTyposquat: true },
      image: true,
      imageConfig: { storageLimit: 16 },
      serialize: true, clipboard: true, progress: false,
    })
    expect(ps.imageConfig).toBeInstanceOf(daemon.ImageConfig)
    expect(ps.imageConfig.storageLimit).toBe(16)
  })

  it('daemon.PluginSettings imageConfig round-trips through JSON (Plan 96-01 hand-edit)', () => {
    const ps = new daemon.PluginSettings({
      webgl: true, unicode11: true, search: true,
      searchConfig: { regex: false, caseSensitive: false, wholeWord: false },
      webLinks: true,
      webLinksConfig: { modifier: 'platform', confirmOSC8: true, confirmIDN: true, confirmTyposquat: true },
      image: true,
      imageConfig: { storageLimit: 32 },
      serialize: true, clipboard: true, progress: false,
    })
    const roundTrip = JSON.parse(JSON.stringify(ps))
    expect(roundTrip.imageConfig.storageLimit).toBe(32)
  })
})
