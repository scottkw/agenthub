import { describe, it, expect } from 'vitest'
import raw from '../../components/SettingsTab.tsx?raw'

// Source-inspection tests for SettingsTab.tsx (UI-02: Settings as sidebar tab).
// Verifies the component was refactored from a modal into an inline sidebar tab.

describe('UI-02 Gap 1: SettingsTab exports', () => {
  it('exports SettingsTab function component', () => {
    expect(raw).toContain('export function SettingsTab')
  })
})

describe('UI-02 Gap 2: SettingsTab props interface', () => {
  it('props include clis', () => {
    expect(raw).toContain('clis')
  })

  it('props include tailscaleHealth', () => {
    expect(raw).toContain('tailscaleHealth')
  })

  it('props include onWebServerStateChange', () => {
    expect(raw).toContain('onWebServerStateChange')
  })

  it('props do NOT include isOpen (modal remnant)', () => {
    // isOpen is a modal-specific prop — must not appear in the SettingsTabProps interface.
    // We check the interface block specifically to avoid false positives from comments.
    const interfaceStart = raw.indexOf('interface SettingsTabProps')
    const interfaceEnd = raw.indexOf('}', interfaceStart)
    expect(interfaceStart).toBeGreaterThan(-1)
    const interfaceBlock = raw.slice(interfaceStart, interfaceEnd + 1)
    expect(interfaceBlock).not.toContain('isOpen')
  })

  it('props do NOT include onClose (modal remnant)', () => {
    // onClose is a modal-specific prop — must not appear in the SettingsTabProps interface.
    const interfaceStart = raw.indexOf('interface SettingsTabProps')
    const interfaceEnd = raw.indexOf('}', interfaceStart)
    expect(interfaceStart).toBeGreaterThan(-1)
    const interfaceBlock = raw.slice(interfaceStart, interfaceEnd + 1)
    expect(interfaceBlock).not.toContain('onClose')
  })
})

describe('UI-02 Gap 3: No modal shell classes', () => {
  it('does NOT contain settings-overlay class', () => {
    expect(raw).not.toContain('settings-overlay')
  })

  it('does NOT contain settings-panel__header class', () => {
    expect(raw).not.toContain('settings-panel__header"')
  })

  it('does NOT contain settings-panel__footer class', () => {
    expect(raw).not.toContain('settings-panel__footer"')
  })

  it('does NOT contain settings-panel__close class', () => {
    expect(raw).not.toContain('settings-panel__close"')
  })
})

describe('UI-02 Gap 4: settings-tab outer wrapper', () => {
  it('has className="settings-tab" as outer wrapper', () => {
    expect(raw).toContain('className="settings-tab"')
  })
})

describe('UI-02 Gap 5: Mount-based useEffect', () => {
  it('has useEffect with empty dependency array []', () => {
    // A mount-only effect ends with }, []) pattern.
    expect(raw).toContain('}, [])')
  })

  it('useEffect does NOT guard on isOpen', () => {
    // Old modal pattern: useEffect(() => { if (!isOpen) return; ... }, [isOpen])
    // Should not be present in the sidebar tab version.
    expect(raw).not.toContain('isOpen')
  })
})

describe('THM-01: Appearance tab with theme selector', () => {
  it('imports xterm-theme library', () => {
    expect(raw).toContain("from 'xterm-theme'")
  })

  it('computes THEME_NAMES at module level', () => {
    expect(raw).toContain('THEME_NAMES = Object.keys(xtermThemes).sort()')
  })

  it('activeTab state union includes appearance', () => {
    expect(raw).toContain("'appearance'")
  })

  it('has Appearance tab button with correct CSS class', () => {
    expect(raw).toContain("activeTab === 'appearance'")
    expect(raw).toContain('settings-panel__tab-btn')
  })

  it('props include selectedTheme', () => {
    // selectedTheme: string must appear in the interface block
    // (indexOf('}') would find nested tailscaleHealth's brace, so search the full source)
    expect(raw).toContain('selectedTheme: string')
  })

  it('props include onThemeChange callback', () => {
    // onThemeChange must appear in the interface block
    expect(raw).toContain('onThemeChange: (name: string) => void')
  })

  it('renders theme select with THEME_NAMES options', () => {
    expect(raw).toContain('THEME_NAMES.map')
  })

  it('select value is bound to selectedTheme prop', () => {
    expect(raw).toContain('value={selectedTheme}')
  })

  it('select onChange calls onThemeChange', () => {
    expect(raw).toContain('onThemeChange(e.target.value)')
  })

  it('displays theme names with underscores replaced by spaces', () => {
    expect(raw).toContain("name.replace(/_/g, ' ')")
  })

  it('Appearance tab button has aria-selected attribute', () => {
    // The tab button must include aria-selected for accessibility
    const appearanceSection = raw.slice(raw.indexOf("'appearance'"))
    expect(appearanceSection).toContain('aria-selected')
  })
})
