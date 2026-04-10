import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

// Source-inspection tests for App.tsx modal cleanup (UI-02: Settings as sidebar tab).
// Verifies the old SettingsPanel modal wiring was removed and replaced with SettingsTab.

describe('UI-02 Gap 6: Dead modal code removed from App.tsx', () => {
  it('does NOT contain showSettings state (old modal)', () => {
    expect(raw).not.toContain('showSettings')
  })

  it('does NOT contain handleSettingsClose (old modal handler)', () => {
    expect(raw).not.toContain('handleSettingsClose')
  })

  it('does NOT contain setShowSettings (old modal setter)', () => {
    expect(raw).not.toContain('setShowSettings')
  })

  it('does NOT import SettingsPanel (old modal component)', () => {
    // SettingsPanel is the old modal — should not appear as an import.
    // We check the import section specifically.
    const importEnd = raw.indexOf('\nfunction App')
    expect(importEnd).toBeGreaterThan(-1)
    const importBlock = raw.slice(0, importEnd)
    expect(importBlock).not.toContain('SettingsPanel')
  })

  it('imports SettingsTab component', () => {
    expect(raw).toContain("from './components/SettingsTab'")
  })

  it('defines SETTINGS_TAB constant', () => {
    expect(raw).toContain('SETTINGS_TAB')
  })

  it('defines handleOpenSettings callback', () => {
    expect(raw).toContain('handleOpenSettings')
  })

  it('renders <SettingsTab> inline (not as modal)', () => {
    expect(raw).toContain('<SettingsTab')
  })
})
