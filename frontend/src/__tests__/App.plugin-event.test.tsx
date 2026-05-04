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
