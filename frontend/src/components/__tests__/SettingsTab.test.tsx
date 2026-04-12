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

describe('THM-01: Appearance section with theme selector', () => {
  it('imports xterm-theme library', () => {
    expect(raw).toContain("from 'xterm-theme'")
  })

  it('computes THEME_NAMES at module level', () => {
    expect(raw).toContain('THEME_NAMES = Object.keys(xtermThemes).sort()')
  })

  it('props include selectedTheme', () => {
    expect(raw).toContain('selectedTheme: string')
  })

  it('props include onThemeChange callback', () => {
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
})

describe('SETT-01: Single scrollable page (no sub-tabs)', () => {
  it('does NOT contain settings-panel__tabs div', () => {
    expect(raw).not.toContain('settings-panel__tabs')
  })

  it('does NOT contain settings-panel__tab-btn class', () => {
    expect(raw).not.toContain('settings-panel__tab-btn')
  })

  it('does NOT contain role="tablist"', () => {
    expect(raw).not.toContain('role="tablist"')
  })

  it('does NOT contain activeTab conditional gating', () => {
    expect(raw).not.toContain("activeTab === ")
  })

  it('does NOT have activeTab in props interface', () => {
    const interfaceStart = raw.indexOf('interface SettingsTabProps')
    const interfaceEnd = raw.indexOf('}', interfaceStart)
    expect(interfaceStart).toBeGreaterThan(-1)
    const interfaceBlock = raw.slice(interfaceStart, interfaceEnd + 1)
    expect(interfaceBlock).not.toContain('activeTab')
  })

  it('does NOT have onActiveTabChange in props interface', () => {
    const interfaceStart = raw.indexOf('interface SettingsTabProps')
    const interfaceEnd = raw.indexOf('}', interfaceStart)
    expect(interfaceStart).toBeGreaterThan(-1)
    const interfaceBlock = raw.slice(interfaceStart, interfaceEnd + 1)
    expect(interfaceBlock).not.toContain('onActiveTabChange')
  })
})

describe('SETT-02: Section headers present', () => {
  it('has Appearance section header', () => {
    expect(raw).toContain('<h3>Appearance</h3>')
  })

  it('has Web Server section header', () => {
    expect(raw).toContain('<h3>Web Server</h3>')
  })

  it('has Paths section header', () => {
    expect(raw).toContain('<h3>Paths</h3>')
  })
})

describe('SETT-03: All content groups present simultaneously', () => {
  it('contains theme selector (Appearance group)', () => {
    expect(raw).toContain('settings-panel__theme-select')
  })

  it('contains CT disclosure (Web Server group)', () => {
    expect(raw).toContain('ct-disclosure')
  })

  it('contains CLI paths table (Paths group)', () => {
    expect(raw).toContain('settings-panel__table')
  })

  it('contains Save Paths button (Paths group)', () => {
    expect(raw).toContain('Save Paths')
  })
})

describe('TAILSCALE-PATH-01: Tailscale status in Paths section', () => {
  it('renders Tailscale label in Paths section', () => {
    expect(raw).toContain('Tailscale')
  })

  it('shows connected status with domain or ip when tailscaleHealth.connected is true', () => {
    expect(raw).toContain('tailscaleHealth.connected')
    expect(raw).toContain('tailscaleHealth.domain || tailscaleHealth.ip')
  })

  it('shows installed-but-not-connected message', () => {
    expect(raw).toContain('Installed but not connected')
  })

  it('shows not detected message', () => {
    expect(raw).toContain('Not detected')
  })

  it('shows description about daemon socket connection', () => {
    expect(raw).toContain('local daemon socket')
  })

  it('shows No path configuration needed', () => {
    expect(raw).toContain('No path configuration needed')
  })
})
