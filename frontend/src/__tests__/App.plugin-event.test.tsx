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
})
