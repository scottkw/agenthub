import { describe, it, expect } from 'vitest'
import raw from '../PluginsSection.tsx?raw'

describe('PUI-03: <details> disclosures for Search / WebLinks / Image (source-inspection)', () => {
  it('renders three settings-panel__details blocks (one per advanced-config plugin)', () => {
    const matches = raw.match(/settings-panel__details/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(3)
  })

  it('summary copy is verbatim per RESEARCH.md "Claude\'s Discretion"', () => {
    expect(raw).toContain('Search defaults')
    expect(raw).toContain('Link click behavior')
    expect(raw).toContain('Storage limit')
  })

  it('Search disclosure dispatches SetSearchConfig with new daemon.SearchConfig (PUI-04 sub-key contract)', () => {
    expect(raw).toContain('SetSearchConfig')
    expect(raw).toContain('new daemon.SearchConfig')
  })

  it('Web Links disclosure dispatches SetWebLinksConfig with new daemon.WebLinksConfig', () => {
    expect(raw).toContain('SetWebLinksConfig')
    expect(raw).toContain('new daemon.WebLinksConfig')
  })

  it('Inline Images disclosure dispatches SetImageConfig with new daemon.ImageConfig', () => {
    expect(raw).toContain('SetImageConfig')
    expect(raw).toContain('new daemon.ImageConfig')
  })

  it('Web Links modifier <select> exposes platform / cmd / ctrl / none options', () => {
    expect(raw).toContain('value="platform"')
    expect(raw).toContain('value="cmd"')
    expect(raw).toContain('value="ctrl"')
    expect(raw).toContain('value="none"')
  })

  it('Inline Images storageLimit input clamps to [1, 1000] matching daemon api.go:649', () => {
    expect(raw).toMatch(/min=\{?1\}?/)
    expect(raw).toMatch(/max=\{?1000\}?/)
  })
})

describe('PUI-04: anti-race contract — sub-key RPCs do not route through SetPluginSettings', () => {
  it('SetPluginSettings appears exactly twice (1 import + 1 call inside handleSavePlugins)', () => {
    const matches = raw.match(/SetPluginSettings/g) ?? []
    expect(matches.length).toBe(2)
  })

  it('debounces SetImageConfig dispatch (500ms — avoid spamming daemon while user types digits)', () => {
    expect(raw).toMatch(/setTimeout\([\s\S]+?,\s*500\)/)
  })
})
