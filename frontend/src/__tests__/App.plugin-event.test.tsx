import { describe, it, expect } from 'vitest'
import raw from '../App.tsx?raw'
import terminalPanelRaw from '../components/TerminalPanel.tsx?raw'

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
  it('TerminalPanel does NOT consume pluginConfig inside an addon useEffect (Phase 92 contract)', () => {
    // Phase 92 contract: prop is threaded but inert. Phase 93 wires
    // consumption. If a useEffect references pluginConfig, this test
    // fires and the planner is alerted that the contract is being
    // violated.
    const consumesInEffect = /useEffect\([^)]*pluginConfig|useEffect\([^}]*\bpluginConfig\b/.test(terminalPanelRaw)
    expect(consumesInEffect).toBe(false)
  })
})
